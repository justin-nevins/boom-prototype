"""
Boom AI Transcription Service

Joins a LiveKit room as a bot, captures audio, and sends real-time
transcriptions to the backend WebSocket.
"""

import asyncio
import os
import json
import logging
from datetime import datetime
from dotenv import load_dotenv
from livekit import api, rtc
from deepgram import DeepgramClient, LiveOptions, LiveTranscriptionEvents
import aiohttp

load_dotenv()

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("transcription-bot")

# Configuration
LIVEKIT_URL = os.getenv("LIVEKIT_URL")
LIVEKIT_API_KEY = os.getenv("LIVEKIT_API_KEY")
LIVEKIT_API_SECRET = os.getenv("LIVEKIT_API_SECRET")
DEEPGRAM_API_KEY = os.getenv("DEEPGRAM_API_KEY")
BACKEND_WS_URL = os.getenv("BACKEND_WS_URL", "ws://localhost:8080")


def validate_env():
    """Validate required environment variables at startup."""
    required = ["LIVEKIT_URL", "LIVEKIT_API_KEY", "LIVEKIT_API_SECRET", "DEEPGRAM_API_KEY"]
    missing = [k for k in required if not os.getenv(k)]
    if missing:
        raise RuntimeError(f"Missing environment variables: {missing}")


class ParticipantTranscriber:
    """Handles transcription for a single participant."""

    def __init__(self, participant_id: str, deepgram_client: DeepgramClient, send_callback):
        self.participant_id = participant_id
        self.deepgram = deepgram_client
        self.send_callback = send_callback
        self.dg_connection = None
        self.frame_count = 0

    async def start(self):
        """Start Deepgram connection for this participant."""
        self.dg_connection = self.deepgram.listen.asyncwebsocket.v("1")

        participant_id = self.participant_id
        send_callback = self.send_callback

        # Use sync wrappers to avoid coroutine issues with Deepgram SDK
        def on_message(client, result, **kwargs):
            try:
                if not result.is_final:
                    return
                transcript = result.channel.alternatives[0].transcript
                if transcript.strip():
                    logger.info(f"[{participant_id}] {transcript[:60]}")
                    asyncio.create_task(send_callback(transcript, participant_id))
            except Exception as e:
                logger.error(f"Error processing transcript for {participant_id}: {e}")

        def on_error(client, error, **kwargs):
            logger.error(f"Deepgram error for {participant_id}: {error}")

        self.dg_connection.on(LiveTranscriptionEvents.Transcript, on_message)
        self.dg_connection.on(LiveTranscriptionEvents.Error, on_error)

        # LiveKit sends 48kHz 16-bit PCM audio
        options = LiveOptions(
            model="nova-2",
            language="en",
            smart_format=True,
            interim_results=False,
            utterance_end_ms=1500,
            vad_events=True,
            endpointing=500,
            encoding="linear16",
            sample_rate=48000,
            channels=1,
        )

        result = await self.dg_connection.start(options)
        if not result:
            logger.error(f"Failed to start Deepgram for {participant_id}")
            return False
        logger.info(f"Started Deepgram for participant: {participant_id}")
        return True

    async def send_audio(self, audio_data: bytes):
        """Send audio frame to Deepgram."""
        if self.dg_connection:
            await self.dg_connection.send(audio_data)
            self.frame_count += 1

    async def stop(self):
        """Stop Deepgram connection."""
        if self.dg_connection:
            await self.dg_connection.finish()
            logger.info(f"Stopped Deepgram for participant: {self.participant_id} (processed {self.frame_count} frames)")


class TranscriptionBot:
    def __init__(self, room_name: str):
        self.room_name = room_name
        self.room = rtc.Room()
        self.deepgram = DeepgramClient(DEEPGRAM_API_KEY)
        self.backend_ws = None
        self.participant_transcribers: dict[str, ParticipantTranscriber] = {}
        self._session = None
        self.full_transcript: list[dict] = []  # Store all transcripts for note generation

    async def connect(self):
        """Connect to LiveKit room and start transcription."""
        # Generate bot token
        token = api.AccessToken(LIVEKIT_API_KEY, LIVEKIT_API_SECRET)
        token.with_identity("transcription-bot")
        token.with_name("Transcription Bot")
        token.with_grants(api.VideoGrants(
            room_join=True,
            room=self.room_name,
        ))
        jwt = token.to_jwt()

        # Connect to backend WebSocket
        await self.connect_backend_ws()

        # Setup LiveKit event handlers
        def on_track_subscribed_sync(track, publication, participant):
            asyncio.create_task(self.on_track_subscribed(track, publication, participant))

        def on_track_unsubscribed_sync(track, publication, participant):
            asyncio.create_task(self.on_track_unsubscribed(track, publication, participant))

        def on_disconnected_sync():
            asyncio.create_task(self.on_disconnected())

        self.room.on("track_subscribed", on_track_subscribed_sync)
        self.room.on("track_unsubscribed", on_track_unsubscribed_sync)
        self.room.on("disconnected", on_disconnected_sync)

        # Connect to LiveKit
        logger.info(f"Connecting to room: {self.room_name}")
        await self.room.connect(LIVEKIT_URL, jwt)
        logger.info("Connected to LiveKit room")

    async def connect_backend_ws(self):
        """Connect to backend WebSocket for broadcasting transcriptions."""
        ws_url = f"{BACKEND_WS_URL}/ws/transcription/{self.room_name}"
        logger.info(f"Connecting to backend WebSocket: {ws_url}")

        self._session = aiohttp.ClientSession()
        self.backend_ws = await self._session.ws_connect(ws_url)
        logger.info("Connected to backend WebSocket")

    async def on_track_subscribed(
        self,
        track: rtc.Track,
        publication: rtc.RemoteTrackPublication,
        participant: rtc.RemoteParticipant,
    ):
        """Handle new audio track subscription."""
        if track.kind != rtc.TrackKind.KIND_AUDIO:
            return

        participant_id = participant.identity
        logger.info(f"Subscribed to audio from: {participant_id}")

        # Create transcriber for this participant if not exists
        if participant_id not in self.participant_transcribers:
            transcriber = ParticipantTranscriber(
                participant_id,
                self.deepgram,
                self.send_transcription
            )
            await transcriber.start()
            self.participant_transcribers[participant_id] = transcriber

        transcriber = self.participant_transcribers[participant_id]
        audio_stream = rtc.AudioStream(track)

        async for frame_event in audio_stream:
            await transcriber.send_audio(frame_event.frame.data.tobytes())

    async def on_track_unsubscribed(
        self,
        track: rtc.Track,
        publication: rtc.RemoteTrackPublication,
        participant: rtc.RemoteParticipant,
    ):
        """Handle audio track unsubscription."""
        if track.kind != rtc.TrackKind.KIND_AUDIO:
            return

        participant_id = participant.identity
        if participant_id in self.participant_transcribers:
            await self.participant_transcribers[participant_id].stop()
            del self.participant_transcribers[participant_id]
            logger.info(f"Removed transcriber for: {participant_id}")

    async def send_transcription(self, text: str, speaker: str):
        """Send transcription to backend WebSocket and store for notes."""
        # Store in full transcript for note generation
        self.full_transcript.append({
            "speaker": speaker,
            "text": text,
            "timestamp": datetime.utcnow().isoformat()
        })

        # Broadcast to frontend via backend WebSocket
        if self.backend_ws:
            message = json.dumps({
                "text": text,
                "speaker": speaker,
                "room": self.room_name,
            })
            await self.backend_ws.send_str(message)

    async def on_disconnected(self):
        """Handle room disconnection."""
        logger.info("Disconnected from room")
        await self.cleanup()

    async def cleanup(self):
        """Cleanup connections."""
        # Stop all participant transcribers
        for transcriber in self.participant_transcribers.values():
            await transcriber.stop()
        self.participant_transcribers.clear()

        if self.backend_ws:
            await self.backend_ws.close()
        if self._session:
            await self._session.close()
        await self.room.disconnect()


def run_service():
    """Run as HTTP service that spawns bots on demand."""
    from aiohttp import web
    from notes_generator import generate_notes

    bots = {}
    BACKEND_API_URL = os.getenv("BACKEND_API_URL", "http://localhost:8080")

    async def health(request):
        """Health check endpoint."""
        return web.json_response({
            "status": "ok",
            "service": "ai",
            "bots_active": len(bots)
        })

    async def join_room(request):
        data = await request.json()
        room_name = data.get("room_name")

        if room_name in bots:
            return web.json_response({"status": "already_joined"})

        bot = TranscriptionBot(room_name)
        bots[room_name] = bot
        asyncio.create_task(bot.connect())

        return web.json_response({"status": "joined", "room": room_name})

    async def leave_room(request):
        data = await request.json()
        room_name = data.get("room_name")

        if room_name in bots:
            await bots[room_name].cleanup()
            del bots[room_name]
            return web.json_response({"status": "left"})

        return web.json_response({"status": "not_found"}, status=404)

    async def generate_notes_handler(request):
        """Generate meeting notes from transcript using Claude."""
        data = await request.json()
        room_name = data.get("room_name")

        if room_name not in bots:
            return web.json_response({"error": "Bot not active for this room"}, status=404)

        bot = bots[room_name]
        transcript = bot.full_transcript

        if not transcript:
            return web.json_response({"error": "No transcript available"}, status=400)

        try:
            # Generate notes with Claude
            logger.info(f"Generating notes for room {room_name} with {len(transcript)} entries")
            result = await generate_notes(transcript)

            # Save notes to backend
            async with aiohttp.ClientSession() as session:
                async with session.post(
                    f"{BACKEND_API_URL}/api/meetings/{room_name}/notes",
                    json={
                        "markdown": result["markdown"],
                        "model": result["model"],
                        "inputTokens": result["usage"]["input_tokens"],
                        "outputTokens": result["usage"]["output_tokens"]
                    }
                ) as resp:
                    if resp.status != 200:
                        logger.warning(f"Failed to save notes to backend: {await resp.text()}")

            return web.json_response({
                "status": "success",
                "markdown": result["markdown"],
                "usage": result["usage"]
            })

        except Exception as e:
            logger.error(f"Error generating notes: {e}")
            return web.json_response({"error": str(e)}, status=500)

    app = web.Application()
    app.router.add_get("/health", health)
    app.router.add_post("/join", join_room)
    app.router.add_post("/leave", leave_room)
    app.router.add_post("/generate-notes", generate_notes_handler)

    logger.info("Starting transcription service on port 8081")
    web.run_app(app, port=8081)


async def run_single_room(room_name: str):
    """Run for a specific room."""
    bot = TranscriptionBot(room_name)
    await bot.connect()

    # Keep running
    try:
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        await bot.cleanup()


if __name__ == "__main__":
    import sys

    # Validate environment on startup
    validate_env()

    if len(sys.argv) < 2:
        run_service()
    else:
        asyncio.run(run_single_room(sys.argv[1]))
