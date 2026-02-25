"""
Chunk Persister - Saves transcript chunks to backend every 5 minutes.

This ensures transcript data is never lost, even if the AI service crashes.
Chunks are persisted to the backend database and can be used to:
1. Recover transcripts after a crash
2. Generate notes from persisted chunks if real-time generation fails
"""

import asyncio
import logging
import os
from datetime import datetime
from typing import Dict, Optional

import aiohttp

from transcript_store import transcript_store

logger = logging.getLogger("chunk-persister")

BACKEND_API_URL = os.getenv("BACKEND_API_URL", "http://localhost:8080")
CHUNK_INTERVAL_SECONDS = 300  # 5 minutes


class ChunkPersister:
    """
    Periodically saves transcript chunks to the backend database.
    One persister per room.
    """

    def __init__(self, room_name: str):
        self.room_name = room_name
        self.chunk_index = 0
        self._task: Optional[asyncio.Task] = None
        self._running = False

    async def start(self):
        """Start the periodic chunk saving task."""
        if self._running:
            return

        self._running = True
        self._task = asyncio.create_task(self._run_periodic())
        logger.info(f"[{self.room_name}] Chunk persister started (interval: {CHUNK_INTERVAL_SECONDS}s)")

    async def stop(self):
        """Stop the persister and save final chunk."""
        self._running = False

        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass

        # Save final chunk
        await self.save_chunk()
        logger.info(f"[{self.room_name}] Chunk persister stopped, final chunk saved")

    async def _run_periodic(self):
        """Run the periodic chunk saving loop."""
        try:
            while self._running:
                await asyncio.sleep(CHUNK_INTERVAL_SECONDS)
                if self._running:
                    await self.save_chunk()
        except asyncio.CancelledError:
            pass

    async def save_chunk(self) -> bool:
        """
        Save current transcript entries as a chunk to the backend.

        Returns True if chunk was saved successfully.
        """
        try:
            room = await transcript_store.get_transcript(self.room_name)
            if not room:
                logger.debug(f"[{self.room_name}] No transcript found, skipping chunk save")
                return False

            # Get entries since last checkpoint
            text = room.format_entries_since_checkpoint()
            if not text.strip():
                logger.debug(f"[{self.room_name}] No new entries since last checkpoint")
                return False

            # Get timestamps
            start_time, end_time = room.get_checkpoint_timestamps()
            entries_since_checkpoint = room.get_entries_since_checkpoint()
            entry_count = len(entries_since_checkpoint)

            # Save to backend
            async with aiohttp.ClientSession() as session:
                payload = {
                    "room_name": self.room_name,
                    "chunk_index": self.chunk_index,
                    "transcript_text": text,
                    "start_time": start_time.isoformat() + "Z",
                    "end_time": end_time.isoformat() + "Z",
                    "entry_count": entry_count,
                }

                async with session.post(
                    f"{BACKEND_API_URL}/api/internal/transcript-chunk",
                    json=payload
                ) as resp:
                    if resp.status == 200:
                        # Mark checkpoint in transcript store
                        room.mark_checkpoint()
                        logger.info(
                            f"[{self.room_name}] Saved chunk {self.chunk_index} "
                            f"({entry_count} entries, {len(text)} chars)"
                        )
                        self.chunk_index += 1
                        return True
                    else:
                        error = await resp.text()
                        logger.error(f"[{self.room_name}] Failed to save chunk: {error}")
                        return False

        except Exception as e:
            logger.error(f"[{self.room_name}] Error saving chunk: {e}")
            return False


class ChunkPersisterManager:
    """
    Manages chunk persisters for multiple rooms.
    """

    def __init__(self):
        self._persisters: Dict[str, ChunkPersister] = {}
        self._lock = asyncio.Lock()

    async def start_persister(self, room_name: str) -> ChunkPersister:
        """Start a chunk persister for a room."""
        async with self._lock:
            if room_name in self._persisters:
                return self._persisters[room_name]

            persister = ChunkPersister(room_name)
            await persister.start()
            self._persisters[room_name] = persister
            return persister

    async def stop_persister(self, room_name: str):
        """Stop and remove the chunk persister for a room."""
        async with self._lock:
            persister = self._persisters.pop(room_name, None)
            if persister:
                await persister.stop()

    async def shutdown(self):
        """Stop all persisters."""
        async with self._lock:
            for persister in self._persisters.values():
                await persister.stop()
            self._persisters.clear()


# Global singleton
chunk_persister_manager = ChunkPersisterManager()
