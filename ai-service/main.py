"""
Boom AI Transcription Service

Joins a LiveKit room as a bot, captures audio, and sends real-time
transcriptions to the backend WebSocket.
"""

import asyncio
import os
import json
import logging
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


class TranscriptionBot:
    def __init__(self, room_name: str):
        self.room_name = room_name
        self.room = rtc.Room()
        self.deepgram = DeepgramClient(DEEPGRAM_API_KEY)
        self.dg_connection = None
        self.backend_ws = None
        self.audio_buffer = bytearray()

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

        # Setup Deepgram
        await self.setup_deepgram()

        # Setup LiveKit event handlers
        self.room.on("track_subscribed", self.on_track_subscribed)
        self.room.on("disconnected", self.on_disconnected)

        # Connect to LiveKit
        logger.info(f"Connecting to room: {self.room_name}")
        await self.room.connect(LIVEKIT_URL, jwt)
        logger.info("Connected to LiveKit room")

    async def connect_backend_ws(self):
        """Connect to backend WebSocket for broadcasting transcriptions."""
        ws_url = f"{BACKEND_WS_URL}/ws/transcription/{self.room_name}"
        logger.info(f"Connecting to backend WebSocket: {ws_url}")

        session = aiohttp.ClientSession()
        self.backend_ws = await session.ws_connect(ws_url)
        logger.info("Connected to backend WebSocket")

    async def setup_deepgram(self):
        """Setup Deepgram live transcription."""
        self.dg_connection = self.deepgram.listen.asyncwebsocket.v("1")

        bot_self = self  # Capture self for inner functions

        # Handle transcription results
        async def on_message(client, result, **kwargs):
            try:
                transcript = result.channel.alternatives[0].transcript
                if transcript.strip():
                    await bot_self.send_transcription(transcript)
            except Exception as e:
                logger.error(f"Error processing transcript: {e}")

        async def on_error(client, error, **kwargs):
            logger.error(f"Deepgram error: {error}")

        self.dg_connection.on(LiveTranscriptionEvents.Transcript, on_message)
        self.dg_connection.on(LiveTranscriptionEvents.Error, on_error)

        # Start connection with options
        options = LiveOptions(
            model="nova-2",
            language="en",
            smart_format=True,
            interim_results=True,
            utterance_end_ms="1000",
            vad_events=True,
            endpointing=300,
        )

        await self.dg_connection.start(options)
        logger.info("Deepgram connection started")

    async def on_track_subscribed(
        self,
        track: rtc.Track,
        publication: rtc.RemoteTrackPublication,
        participant: rtc.RemoteParticipant,
    ):
        """Handle new audio track subscription."""
        if track.kind != rtc.TrackKind.KIND_AUDIO:
            return

        logger.info(f"Subscribed to audio from: {participant.identity}")

        audio_stream = rtc.AudioStream(track)

        async for frame_event in audio_stream:
            # Send audio to Deepgram
            if self.dg_connection:
                await self.dg_connection.send(frame_event.frame.data.tobytes())

    async def send_transcription(self, text: str, speaker: str = "Unknown"):
        """Send transcription to backend WebSocket."""
        if self.backend_ws:
            message = json.dumps({
                "text": text,
                "speaker": speaker,
                "room": self.room_name,
            })
            await self.backend_ws.send_str(message)
            logger.debug(f"Sent transcription: {text[:50]}...")

    async def on_disconnected(self):
        """Handle room disconnection."""
        logger.info("Disconnected from room")
        await self.cleanup()

    async def cleanup(self):
        """Cleanup connections."""
        if self.dg_connection:
            await self.dg_connection.finish()
        if self.backend_ws:
            await self.backend_ws.close()
        await self.room.disconnect()


def run_service():
    """Run as HTTP service that spawns bots on demand."""
    from aiohttp import web

    bots = {}

    async def health(request):
        """Health check endpoint for Fly.io."""
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

    app = web.Application()
    app.router.add_get("/health", health)
    app.router.add_post("/join", join_room)
    app.router.add_post("/leave", leave_room)

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
