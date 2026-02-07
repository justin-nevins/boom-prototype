"""
Deepgram real-time streaming transcription client.
Handles live audio streaming and transcript callbacks.
"""

import asyncio
import logging
import os
from typing import Callable, Optional
from datetime import datetime

from deepgram import (
    DeepgramClient,
    DeepgramClientOptions,
    LiveTranscriptionEvents,
    LiveOptions,
)

logger = logging.getLogger(__name__)


class DeepgramStreamer:
    """
    Real-time audio streaming to Deepgram.
    Creates a WebSocket connection to Deepgram's live transcription API.
    """

    def __init__(
        self,
        room_name: str,
        speaker_name: str,
        on_transcript: Optional[Callable[[str, str, bool], None]] = None,
        api_key: Optional[str] = None
    ):
        """
        Initialize Deepgram streamer.

        Args:
            room_name: Room identifier for logging
            speaker_name: Speaker name for transcript attribution
            on_transcript: Callback(speaker, text, is_final) for transcript events
            api_key: Deepgram API key (defaults to env var)
        """
        self.room_name = room_name
        self.speaker_name = speaker_name
        self.on_transcript = on_transcript
        self.api_key = api_key or os.getenv("DEEPGRAM_API_KEY")

        if not self.api_key:
            raise ValueError("DEEPGRAM_API_KEY not set")

        self._client: Optional[DeepgramClient] = None
        self._connection = None
        self._is_connected = False
        self._reconnect_attempts = 0
        self._max_reconnect_attempts = 5

    async def connect(self) -> bool:
        """Establish connection to Deepgram live transcription."""
        try:
            # Create client with keepalive
            config = DeepgramClientOptions(
                options={"keepalive": "true"}
            )
            self._client = DeepgramClient(self.api_key, config)

            # Configure live transcription options
            options = LiveOptions(
                model="nova-2",
                language="en",
                smart_format=True,
                punctuate=True,
                interim_results=True,
                utterance_end_ms=1000,
                vad_events=True,
                encoding="linear16",
                sample_rate=16000,
                channels=1,
            )

            # Create live connection (async version)
            self._connection = self._client.listen.asynclive.v("1")

            # Register event handlers
            self._connection.on(LiveTranscriptionEvents.Open, self._on_open)
            self._connection.on(LiveTranscriptionEvents.Transcript, self._on_transcript)
            self._connection.on(LiveTranscriptionEvents.Error, self._on_error)
            self._connection.on(LiveTranscriptionEvents.Close, self._on_close)
            self._connection.on(LiveTranscriptionEvents.UtteranceEnd, self._on_utterance_end)

            # Start the connection
            if await self._connection.start(options):
                self._is_connected = True
                self._reconnect_attempts = 0
                logger.info(f"[{self.room_name}] Deepgram connected for {self.speaker_name}")
                return True
            else:
                logger.error(f"[{self.room_name}] Failed to start Deepgram connection")
                return False

        except Exception as e:
            logger.error(f"[{self.room_name}] Deepgram connection error: {e}")
            return False

    async def send_audio(self, audio_data: bytes) -> bool:
        """
        Send audio data to Deepgram.

        Args:
            audio_data: Raw PCM audio bytes (16-bit, 16kHz, mono)

        Returns:
            True if sent successfully
        """
        if not self._is_connected or not self._connection:
            return False

        try:
            await self._connection.send(audio_data)
            return True
        except Exception as e:
            logger.error(f"[{self.room_name}] Error sending audio: {e}")
            return False

    async def close(self):
        """Close the Deepgram connection."""
        if self._connection:
            try:
                await self._connection.finish()
                logger.info(f"[{self.room_name}] Deepgram connection closed for {self.speaker_name}")
            except Exception as e:
                logger.error(f"[{self.room_name}] Error closing Deepgram: {e}")
            finally:
                self._is_connected = False
                self._connection = None

    @property
    def is_connected(self) -> bool:
        return self._is_connected

    # Event handlers

    def _on_open(self, *args, **kwargs):
        """Called when connection opens."""
        logger.debug(f"[{self.room_name}] Deepgram connection opened")

    def _on_transcript(self, *args, **kwargs):
        """Called when transcript is received."""
        try:
            result = kwargs.get("result") or (args[1] if len(args) > 1 else None)
            if not result:
                return

            # Extract transcript from result
            channel = result.channel
            if not channel or not channel.alternatives:
                return

            alternative = channel.alternatives[0]
            transcript = alternative.transcript

            if not transcript or not transcript.strip():
                return

            # Check if this is a final result
            is_final = result.is_final

            # Log for debugging
            log_prefix = "FINAL" if is_final else "interim"
            logger.debug(f"[{self.room_name}] [{log_prefix}] {self.speaker_name}: {transcript}")

            # Invoke callback
            if self.on_transcript:
                self.on_transcript(self.speaker_name, transcript.strip(), is_final)

        except Exception as e:
            logger.error(f"[{self.room_name}] Error processing transcript: {e}")

    def _on_error(self, *args, **kwargs):
        """Called on error."""
        error = kwargs.get("error") or (args[1] if len(args) > 1 else "Unknown error")
        logger.error(f"[{self.room_name}] Deepgram error: {error}")

    def _on_close(self, *args, **kwargs):
        """Called when connection closes."""
        self._is_connected = False
        logger.info(f"[{self.room_name}] Deepgram connection closed for {self.speaker_name}")

    def _on_utterance_end(self, *args, **kwargs):
        """Called when an utterance ends (silence detected)."""
        logger.debug(f"[{self.room_name}] Utterance end detected for {self.speaker_name}")


class DeepgramStreamerManager:
    """
    Manages multiple Deepgram streamers for a room.
    One streamer per speaker for better diarization.
    """

    def __init__(
        self,
        room_name: str,
        on_transcript: Optional[Callable[[str, str, bool], None]] = None
    ):
        self.room_name = room_name
        self.on_transcript = on_transcript
        self._streamers: dict[str, DeepgramStreamer] = {}
        self._lock = asyncio.Lock()

    async def get_or_create_streamer(self, speaker_id: str, speaker_name: str) -> DeepgramStreamer:
        """Get existing streamer or create new one for a speaker."""
        async with self._lock:
            if speaker_id not in self._streamers:
                streamer = DeepgramStreamer(
                    room_name=self.room_name,
                    speaker_name=speaker_name,
                    on_transcript=self.on_transcript
                )
                if await streamer.connect():
                    self._streamers[speaker_id] = streamer
                else:
                    raise Exception(f"Failed to connect Deepgram for {speaker_name}")

            return self._streamers[speaker_id]

    async def send_audio(self, speaker_id: str, audio_data: bytes) -> bool:
        """Send audio data for a specific speaker."""
        async with self._lock:
            streamer = self._streamers.get(speaker_id)
            if streamer and streamer.is_connected:
                return await streamer.send_audio(audio_data)
            return False

    async def close_all(self):
        """Close all streamers."""
        async with self._lock:
            for speaker_id, streamer in list(self._streamers.items()):
                await streamer.close()
            self._streamers.clear()
            logger.info(f"[{self.room_name}] All Deepgram streamers closed")

    async def close_speaker(self, speaker_id: str):
        """Close streamer for a specific speaker."""
        async with self._lock:
            streamer = self._streamers.pop(speaker_id, None)
            if streamer:
                await streamer.close()

    @property
    def active_speakers(self) -> list[str]:
        """List of active speaker IDs."""
        return list(self._streamers.keys())
