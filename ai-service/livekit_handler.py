"""
LiveKit room handler for real-time transcription.
Joins rooms as a participant, subscribes to audio tracks,
and streams audio to Deepgram for transcription.
"""

import asyncio
import logging
import os
from typing import Callable, Optional
from datetime import datetime

from livekit import rtc, api
import numpy as np

from deepgram_streamer import DeepgramStreamerManager
from transcript_store import transcript_store, TranscriptEntry

logger = logging.getLogger(__name__)

# Audio constants
DEEPGRAM_SAMPLE_RATE = 16000
LIVEKIT_SAMPLE_RATE = 48000
RESAMPLE_RATIO = LIVEKIT_SAMPLE_RATE // DEEPGRAM_SAMPLE_RATE  # 3


def resample_audio(audio_data: np.ndarray, from_rate: int, to_rate: int) -> np.ndarray:
    """
    Simple downsampling by taking every Nth sample.
    For 48kHz -> 16kHz, take every 3rd sample.
    """
    if from_rate == to_rate:
        return audio_data

    ratio = from_rate // to_rate
    return audio_data[::ratio]


def int16_to_bytes(audio_data: np.ndarray) -> bytes:
    """Convert numpy int16 array to bytes for Deepgram."""
    return audio_data.astype(np.int16).tobytes()


class TranscriptionAgent:
    """
    LiveKit room participant that transcribes audio in real-time.
    """

    def __init__(
        self,
        room_name: str,
        on_transcript_broadcast: Optional[Callable[[str, dict], None]] = None
    ):
        """
        Initialize transcription agent.

        Args:
            room_name: LiveKit room to join
            on_transcript_broadcast: Callback(room_name, transcript_dict) for broadcasting
        """
        self.room_name = room_name
        self.on_transcript_broadcast = on_transcript_broadcast

        self._room: Optional[rtc.Room] = None
        self._deepgram_manager: Optional[DeepgramStreamerManager] = None
        self._is_connected = False
        self._audio_streams: dict[str, asyncio.Task] = {}

        # LiveKit credentials
        self._livekit_url = os.getenv("LIVEKIT_URL")
        self._api_key = os.getenv("LIVEKIT_API_KEY")
        self._api_secret = os.getenv("LIVEKIT_API_SECRET")

        if not all([self._livekit_url, self._api_key, self._api_secret]):
            raise ValueError("LIVEKIT_URL, LIVEKIT_API_KEY, LIVEKIT_API_SECRET required")

    async def join(self) -> bool:
        """Join the LiveKit room and start transcribing."""
        try:
            # Generate access token
            token = self._generate_token()

            # Create room instance
            self._room = rtc.Room()

            # Set up event handlers
            self._room.on("participant_connected", self._on_participant_connected)
            self._room.on("participant_disconnected", self._on_participant_disconnected)
            self._room.on("track_subscribed", self._on_track_subscribed)
            self._room.on("track_unsubscribed", self._on_track_unsubscribed)
            self._room.on("disconnected", self._on_disconnected)

            # Initialize Deepgram manager
            self._deepgram_manager = DeepgramStreamerManager(
                room_name=self.room_name,
                on_transcript=self._handle_transcript
            )

            # Connect to room
            await self._room.connect(self._livekit_url, token)
            self._is_connected = True

            logger.info(f"[{self.room_name}] Transcription agent joined room")

            # Subscribe to existing participants' audio
            for participant in self._room.remote_participants.values():
                await self._subscribe_to_participant(participant)

            return True

        except Exception as e:
            logger.error(f"[{self.room_name}] Failed to join room: {e}")
            return False

    async def leave(self) -> Optional[str]:
        """
        Leave the room and return the accumulated transcript.

        Returns:
            Formatted transcript for notes generation
        """
        try:
            # Close all Deepgram connections
            if self._deepgram_manager:
                await self._deepgram_manager.close_all()

            # Cancel all audio stream tasks
            for task in self._audio_streams.values():
                task.cancel()
            self._audio_streams.clear()

            # Disconnect from room
            if self._room:
                await self._room.disconnect()

            self._is_connected = False
            logger.info(f"[{self.room_name}] Transcription agent left room")

            # Get formatted transcript
            transcript = await transcript_store.get_formatted_transcript(self.room_name)

            # Clear the room's transcript from store
            await transcript_store.clear_room(self.room_name)

            return transcript

        except Exception as e:
            logger.error(f"[{self.room_name}] Error leaving room: {e}")
            return None

    def _generate_token(self) -> str:
        """Generate LiveKit access token for the transcription agent."""
        token = api.AccessToken(self._api_key, self._api_secret)
        token.with_identity(f"transcriber-{self.room_name}")
        token.with_name("Transcription Service")

        # Grant permissions - only subscribe, don't publish
        grant = api.VideoGrants(
            room_join=True,
            room=self.room_name,
            can_subscribe=True,
            can_publish=False,
            can_publish_data=False,
        )
        token.with_grants(grant)

        return token.to_jwt()

    async def _subscribe_to_participant(self, participant: rtc.RemoteParticipant):
        """Subscribe to a participant's audio tracks."""
        speaker_name = participant.name or participant.identity or "Unknown"
        logger.info(f"[{self.room_name}] Subscribing to {speaker_name}")

        for publication in participant.track_publications.values():
            if publication.track and publication.kind == rtc.TrackKind.KIND_AUDIO:
                await self._start_audio_stream(
                    participant.identity,
                    speaker_name,
                    publication.track
                )

    async def _start_audio_stream(
        self,
        participant_id: str,
        speaker_name: str,
        track: rtc.Track
    ):
        """Start streaming audio from a track to Deepgram."""
        if participant_id in self._audio_streams:
            return

        async def stream_audio():
            try:
                # Get or create Deepgram streamer for this speaker
                streamer = await self._deepgram_manager.get_or_create_streamer(
                    participant_id,
                    speaker_name
                )

                # Create audio stream from track
                audio_stream = rtc.AudioStream(track)

                async for frame_event in audio_stream:
                    if not self._is_connected:
                        break

                    frame = frame_event.frame

                    # Convert to numpy array
                    audio_data = np.frombuffer(frame.data, dtype=np.int16)

                    # Resample from 48kHz to 16kHz
                    resampled = resample_audio(
                        audio_data,
                        frame.sample_rate,
                        DEEPGRAM_SAMPLE_RATE
                    )

                    # Send to Deepgram
                    audio_bytes = int16_to_bytes(resampled)
                    await streamer.send_audio(audio_bytes)

            except asyncio.CancelledError:
                logger.debug(f"[{self.room_name}] Audio stream cancelled for {speaker_name}")
            except Exception as e:
                logger.error(f"[{self.room_name}] Audio stream error for {speaker_name}: {e}")

        # Start the audio streaming task
        task = asyncio.create_task(stream_audio())
        self._audio_streams[participant_id] = task
        logger.info(f"[{self.room_name}] Started audio stream for {speaker_name}")

    def _handle_transcript(self, speaker: str, text: str, is_final: bool):
        """Handle incoming transcript from Deepgram."""
        # Store in transcript store (fire and forget)
        asyncio.create_task(
            transcript_store.add_entry(self.room_name, speaker, text, is_final)
        )

        # Broadcast to backend/frontend
        if self.on_transcript_broadcast:
            transcript_data = {
                "speaker": speaker,
                "text": text,
                "is_final": is_final,
                "timestamp": datetime.utcnow().strftime("%H:%M:%S")
            }
            self.on_transcript_broadcast(self.room_name, transcript_data)

    # LiveKit event handlers

    def _on_participant_connected(self, participant: rtc.RemoteParticipant):
        """Called when a new participant joins."""
        logger.info(f"[{self.room_name}] Participant connected: {participant.identity}")
        asyncio.create_task(self._subscribe_to_participant(participant))

    def _on_participant_disconnected(self, participant: rtc.RemoteParticipant):
        """Called when a participant leaves."""
        logger.info(f"[{self.room_name}] Participant disconnected: {participant.identity}")
        # Close their Deepgram connection
        if self._deepgram_manager:
            asyncio.create_task(
                self._deepgram_manager.close_speaker(participant.identity)
            )
        # Cancel their audio stream
        task = self._audio_streams.pop(participant.identity, None)
        if task:
            task.cancel()

    def _on_track_subscribed(
        self,
        track: rtc.Track,
        publication: rtc.RemoteTrackPublication,
        participant: rtc.RemoteParticipant
    ):
        """Called when subscribed to a track."""
        if track.kind == rtc.TrackKind.KIND_AUDIO:
            speaker_name = participant.name or participant.identity or "Unknown"
            logger.info(f"[{self.room_name}] Audio track subscribed from {speaker_name}")
            asyncio.create_task(
                self._start_audio_stream(participant.identity, speaker_name, track)
            )

    def _on_track_unsubscribed(
        self,
        track: rtc.Track,
        publication: rtc.RemoteTrackPublication,
        participant: rtc.RemoteParticipant
    ):
        """Called when unsubscribed from a track."""
        logger.debug(f"[{self.room_name}] Track unsubscribed from {participant.identity}")

    def _on_disconnected(self):
        """Called when disconnected from room."""
        self._is_connected = False
        logger.info(f"[{self.room_name}] Disconnected from room")

    @property
    def is_connected(self) -> bool:
        return self._is_connected


class TranscriptionAgentManager:
    """
    Manages transcription agents for multiple rooms.
    """

    def __init__(
        self,
        on_transcript_broadcast: Optional[Callable[[str, dict], None]] = None
    ):
        self.on_transcript_broadcast = on_transcript_broadcast
        self._agents: dict[str, TranscriptionAgent] = {}
        self._lock = asyncio.Lock()

    async def join_room(self, room_name: str) -> bool:
        """Create and join a room with a transcription agent."""
        async with self._lock:
            if room_name in self._agents:
                logger.warning(f"[{room_name}] Agent already exists")
                return True

            agent = TranscriptionAgent(
                room_name=room_name,
                on_transcript_broadcast=self.on_transcript_broadcast
            )

            if await agent.join():
                self._agents[room_name] = agent
                return True
            return False

    async def leave_room(self, room_name: str) -> Optional[str]:
        """Leave a room and return the transcript."""
        async with self._lock:
            agent = self._agents.pop(room_name, None)
            if agent:
                return await agent.leave()
            return None

    async def is_room_active(self, room_name: str) -> bool:
        """Check if a room has an active agent."""
        async with self._lock:
            return room_name in self._agents

    async def get_active_rooms(self) -> list[str]:
        """List all active rooms."""
        async with self._lock:
            return list(self._agents.keys())

    async def shutdown(self):
        """Shutdown all agents."""
        async with self._lock:
            for room_name, agent in list(self._agents.items()):
                await agent.leave()
            self._agents.clear()
            logger.info("All transcription agents shut down")
