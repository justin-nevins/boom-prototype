"""
Boom AI Service - Batch Transcription

Downloads meeting recordings, transcribes with Deepgram batch API,
and generates notes with Claude.
"""

import asyncio
import os
import logging
import tempfile
from datetime import datetime
from dotenv import load_dotenv
from deepgram import DeepgramClient, PrerecordedOptions
import aiohttp
from aiohttp import web

load_dotenv()

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("ai-service")

# Configuration
DEEPGRAM_API_KEY = os.getenv("DEEPGRAM_API_KEY")
BACKEND_API_URL = os.getenv("BACKEND_API_URL", "http://localhost:8080")


def validate_env():
    """Validate required environment variables at startup."""
    required = ["DEEPGRAM_API_KEY", "ANTHROPIC_API_KEY"]
    missing = [k for k in required if not os.getenv(k)]
    if missing:
        raise RuntimeError(f"Missing environment variables: {missing}")


async def download_audio(url: str) -> bytes:
    """Download audio file from URL."""
    logger.info(f"Downloading audio from: {url}")
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as resp:
            if resp.status != 200:
                raise Exception(f"Failed to download audio: {resp.status}")
            return await resp.read()


async def transcribe_audio(audio_data: bytes) -> list[dict]:
    """
    Transcribe audio using Deepgram batch API.

    Returns list of transcript entries with speaker diarization.
    """
    logger.info(f"Transcribing {len(audio_data)} bytes of audio")

    deepgram = DeepgramClient(DEEPGRAM_API_KEY)

    options = PrerecordedOptions(
        model="nova-2",
        language="en",
        smart_format=True,
        diarize=True,  # Enable speaker diarization
        punctuate=True,
        paragraphs=True,
    )

    source = {"buffer": audio_data, "mimetype": "audio/ogg"}

    response = await asyncio.get_event_loop().run_in_executor(
        None,
        lambda: deepgram.listen.rest.v("1").transcribe_file(source, options)
    )

    # Extract transcript with speaker info
    transcript = []

    results = response.results
    if results and results.channels:
        channel = results.channels[0]
        if channel.alternatives:
            alt = channel.alternatives[0]

            # If we have word-level data with speakers
            if alt.words:
                current_speaker = None
                current_text = []
                current_start = 0

                for word in alt.words:
                    speaker = f"Speaker {word.speaker}" if hasattr(word, 'speaker') else "Speaker"

                    if speaker != current_speaker:
                        # Save previous segment
                        if current_text:
                            transcript.append({
                                "speaker": current_speaker or "Speaker",
                                "text": " ".join(current_text),
                                "timestamp": format_timestamp(current_start)
                            })
                        current_speaker = speaker
                        current_text = [word.punctuated_word or word.word]
                        current_start = word.start
                    else:
                        current_text.append(word.punctuated_word or word.word)

                # Save last segment
                if current_text:
                    transcript.append({
                        "speaker": current_speaker or "Speaker",
                        "text": " ".join(current_text),
                        "timestamp": format_timestamp(current_start)
                    })

            # Fallback: use paragraph-level if no word data
            elif alt.paragraphs and alt.paragraphs.paragraphs:
                for para in alt.paragraphs.paragraphs:
                    speaker = f"Speaker {para.speaker}" if hasattr(para, 'speaker') else "Speaker"
                    for sentence in para.sentences:
                        transcript.append({
                            "speaker": speaker,
                            "text": sentence.text,
                            "timestamp": format_timestamp(sentence.start)
                        })

            # Last fallback: just use the full transcript
            else:
                transcript.append({
                    "speaker": "Speaker",
                    "text": alt.transcript,
                    "timestamp": format_timestamp(0)
                })

    logger.info(f"Transcribed into {len(transcript)} segments")
    return transcript


def format_timestamp(seconds: float) -> str:
    """Format seconds into ISO timestamp."""
    return datetime.utcfromtimestamp(seconds).strftime("%H:%M:%S")


async def update_backend_status(room_name: str, egress_id: str, status: str, notes_markdown: str = None):
    """Update backend with transcription status."""
    async with aiohttp.ClientSession() as session:
        if status == "completed" and notes_markdown:
            # Save notes to backend
            async with session.post(
                f"{BACKEND_API_URL}/api/meetings/{room_name}/notes",
                json={
                    "markdown": notes_markdown,
                    "model": "claude-sonnet-4-20250514",
                    "inputTokens": 0,
                    "outputTokens": 0
                }
            ) as resp:
                if resp.status == 200:
                    logger.info(f"Notes saved to backend for room {room_name}")
                else:
                    logger.error(f"Failed to save notes: {await resp.text()}")


def run_service():
    """Run as HTTP service."""
    from notes_generator import generate_notes

    processing_tasks = {}  # Track in-progress transcriptions

    async def health(request):
        """Health check endpoint."""
        return web.json_response({
            "status": "ok",
            "service": "ai-batch",
            "processing": len(processing_tasks)
        })

    async def transcribe_recording(request):
        """
        Transcribe a recording and generate notes.

        Expected payload:
        {
            "room_name": "room-xxx",
            "audio_url": "https://...",
            "egress_id": "EG_xxx"
        }
        """
        data = await request.json()
        room_name = data.get("room_name")
        audio_url = data.get("audio_url")
        egress_id = data.get("egress_id")

        if not all([room_name, audio_url, egress_id]):
            return web.json_response({"error": "Missing required fields"}, status=400)

        # Check if already processing
        if egress_id in processing_tasks:
            return web.json_response({"status": "already_processing"})

        # Start async processing
        async def process():
            try:
                processing_tasks[egress_id] = "downloading"

                # Download audio
                audio_data = await download_audio(audio_url)

                processing_tasks[egress_id] = "transcribing"

                # Transcribe with Deepgram
                transcript = await transcribe_audio(audio_data)

                processing_tasks[egress_id] = "generating_notes"

                # Generate notes with Claude
                logger.info(f"Generating notes for {room_name} with {len(transcript)} entries")
                result = await generate_notes(transcript)

                processing_tasks[egress_id] = "saving"

                # Save to backend
                await update_backend_status(room_name, egress_id, "completed", result["markdown"])

                logger.info(f"Completed processing for {room_name}")

            except Exception as e:
                logger.error(f"Error processing {room_name}: {e}")
                await update_backend_status(room_name, egress_id, "failed")
            finally:
                if egress_id in processing_tasks:
                    del processing_tasks[egress_id]

        asyncio.create_task(process())

        return web.json_response({
            "status": "processing",
            "room_name": room_name,
            "egress_id": egress_id
        })

    async def get_status(request):
        """Get processing status for an egress."""
        egress_id = request.query.get("egress_id")
        if egress_id and egress_id in processing_tasks:
            return web.json_response({
                "status": "processing",
                "stage": processing_tasks[egress_id]
            })
        return web.json_response({"status": "not_found"})

    # Legacy endpoint for backwards compatibility (returns error)
    async def join_room(request):
        return web.json_response({
            "error": "Live transcription disabled. Use batch transcription via stop-recording endpoint.",
            "status": "deprecated"
        }, status=410)

    async def leave_room(request):
        return web.json_response({"status": "deprecated"}, status=410)

    async def generate_notes_handler(request):
        return web.json_response({
            "error": "Use transcribe-recording endpoint instead",
            "status": "deprecated"
        }, status=410)

    app = web.Application()
    app.router.add_get("/health", health)
    app.router.add_post("/transcribe-recording", transcribe_recording)
    app.router.add_get("/status", get_status)

    # Legacy endpoints (deprecated)
    app.router.add_post("/join", join_room)
    app.router.add_post("/leave", leave_room)
    app.router.add_post("/generate-notes", generate_notes_handler)

    logger.info("Starting batch transcription service on port 8081")
    web.run_app(app, port=8081)


if __name__ == "__main__":
    validate_env()
    run_service()
