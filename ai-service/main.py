"""
Boom AI Service - Real-Time Transcription

Joins LiveKit rooms as a participant, streams audio to Deepgram for
real-time transcription, and generates notes with Claude on meeting end.
"""

import asyncio
import os
import logging
from dotenv import load_dotenv
import aiohttp
from aiohttp import web

from livekit_handler import TranscriptionAgentManager
from notes_generator import generate_notes_from_text

load_dotenv()

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("ai-service")

# Configuration
BACKEND_API_URL = os.getenv("BACKEND_API_URL", "http://localhost:8080")


def validate_env():
    """Validate required environment variables at startup."""
    required = [
        "DEEPGRAM_API_KEY",
        "ANTHROPIC_API_KEY",
        "LIVEKIT_URL",
        "LIVEKIT_API_KEY",
        "LIVEKIT_API_SECRET",
    ]
    missing = [k for k in required if not os.getenv(k)]
    if missing:
        raise RuntimeError(f"Missing environment variables: {missing}")


async def broadcast_transcript(room_name: str, transcript_data: dict):
    """Send transcript update to backend for WebSocket broadcast."""
    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"{BACKEND_API_URL}/api/internal/transcript",
                json={
                    "room_name": room_name,
                    "speaker": transcript_data["speaker"],
                    "text": transcript_data["text"],
                    "is_final": transcript_data["is_final"],
                    "timestamp": transcript_data["timestamp"],
                }
            ) as resp:
                if resp.status != 200:
                    logger.warning(f"Failed to broadcast transcript: {resp.status}")
    except Exception as e:
        logger.error(f"Error broadcasting transcript: {e}")


async def save_notes_to_backend(room_name: str, markdown: str, usage: dict):
    """Save generated notes to backend."""
    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"{BACKEND_API_URL}/api/meetings/{room_name}/notes",
                json={
                    "markdown": markdown,
                    "model": "claude-sonnet-4-20250514",
                    "inputTokens": usage.get("input_tokens", 0),
                    "outputTokens": usage.get("output_tokens", 0),
                }
            ) as resp:
                if resp.status == 200:
                    logger.info(f"Notes saved to backend for room {room_name}")
                else:
                    logger.error(f"Failed to save notes: {await resp.text()}")
    except Exception as e:
        logger.error(f"Error saving notes to backend: {e}")


def run_service():
    """Run as HTTP service."""
    # Initialize manager with broadcast callback
    agent_manager = TranscriptionAgentManager(
        on_transcript_broadcast=lambda room, data: asyncio.create_task(
            broadcast_transcript(room, data)
        )
    )

    async def health(request):
        """Health check endpoint."""
        active_rooms = await agent_manager.get_active_rooms()
        return web.json_response({
            "status": "ok",
            "service": "ai-realtime",
            "active_rooms": active_rooms,
            "room_count": len(active_rooms),
        })

    async def join_room(request):
        """
        Join a LiveKit room and start real-time transcription.

        Expected payload:
        {
            "room_name": "room-xxx"
        }
        """
        try:
            data = await request.json()
            room_name = data.get("room_name")

            if not room_name:
                return web.json_response(
                    {"error": "room_name required"},
                    status=400
                )

            # Check if already in room
            if await agent_manager.is_room_active(room_name):
                return web.json_response({
                    "status": "already_joined",
                    "room_name": room_name,
                })

            # Join the room
            logger.info(f"Joining room: {room_name}")
            success = await agent_manager.join_room(room_name)

            if success:
                return web.json_response({
                    "status": "joined",
                    "room_name": room_name,
                })
            else:
                return web.json_response(
                    {"error": "Failed to join room", "room_name": room_name},
                    status=500
                )

        except Exception as e:
            logger.error(f"Error joining room: {e}")
            return web.json_response(
                {"error": str(e)},
                status=500
            )

    async def leave_room(request):
        """
        Leave a LiveKit room and generate notes from transcript.

        Expected payload:
        {
            "room_name": "room-xxx"
        }

        Returns:
        {
            "status": "completed",
            "markdown": "# Meeting Notes...",
            "usage": {...}
        }
        """
        try:
            data = await request.json()
            room_name = data.get("room_name")

            if not room_name:
                return web.json_response(
                    {"error": "room_name required"},
                    status=400
                )

            # Check if room is active
            if not await agent_manager.is_room_active(room_name):
                return web.json_response(
                    {"error": "Room not active", "room_name": room_name},
                    status=404
                )

            logger.info(f"Leaving room: {room_name}")

            # Leave room and get transcript
            transcript = await agent_manager.leave_room(room_name)

            if not transcript:
                logger.warning(f"No transcript available for room {room_name}")
                return web.json_response({
                    "status": "completed",
                    "room_name": room_name,
                    "markdown": "# Meeting Notes\n\nNo transcript was captured for this meeting.",
                    "usage": {"input_tokens": 0, "output_tokens": 0}
                })

            logger.info(f"Generating notes for room {room_name} ({len(transcript)} chars)")

            # Generate notes with Claude
            result = await generate_notes_from_text(transcript)

            # Save to backend
            await save_notes_to_backend(room_name, result["markdown"], result["usage"])

            return web.json_response({
                "status": "completed",
                "room_name": room_name,
                "markdown": result["markdown"],
                "usage": result["usage"],
            })

        except Exception as e:
            logger.error(f"Error leaving room: {e}")
            return web.json_response(
                {"error": str(e)},
                status=500
            )

    async def get_rooms(request):
        """List all active transcription rooms."""
        active_rooms = await agent_manager.get_active_rooms()
        return web.json_response({
            "rooms": active_rooms,
            "count": len(active_rooms),
        })

    async def on_shutdown(app):
        """Cleanup on shutdown."""
        logger.info("Shutting down AI service...")
        await agent_manager.shutdown()

    # Create app
    app = web.Application()
    app.on_shutdown.append(on_shutdown)

    # Routes
    app.router.add_get("/health", health)
    app.router.add_post("/join", join_room)
    app.router.add_post("/leave", leave_room)
    app.router.add_get("/rooms", get_rooms)

    logger.info("Starting real-time transcription service on port 8081")
    web.run_app(app, port=8081)


if __name__ == "__main__":
    validate_env()
    run_service()
