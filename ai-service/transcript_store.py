"""
In-memory transcript storage for real-time transcription.
Thread-safe storage of transcript entries keyed by room name.
"""

import asyncio
from dataclasses import dataclass, field
from datetime import datetime
from typing import Dict, List, Optional
import uuid


@dataclass
class TranscriptEntry:
    """A single transcript entry from a speaker."""
    id: str
    speaker: str
    text: str
    timestamp: datetime
    is_final: bool = True

    def to_dict(self) -> dict:
        return {
            "id": self.id,
            "speaker": self.speaker,
            "text": self.text,
            "timestamp": self.timestamp.isoformat(),
            "is_final": self.is_final
        }

    def to_broadcast_dict(self) -> dict:
        """Format for WebSocket broadcast to frontend."""
        return {
            "speaker": self.speaker,
            "text": self.text,
            "timestamp": self.timestamp.strftime("%H:%M:%S"),
            "is_final": self.is_final
        }


@dataclass
class RoomTranscript:
    """Transcript data for a single room."""
    room_name: str
    entries: List[TranscriptEntry] = field(default_factory=list)
    started_at: datetime = field(default_factory=datetime.utcnow)
    max_entries: int = 10000

    def add_entry(self, speaker: str, text: str, is_final: bool = True) -> TranscriptEntry:
        """Add a new transcript entry."""
        entry = TranscriptEntry(
            id=str(uuid.uuid4()),
            speaker=speaker,
            text=text,
            timestamp=datetime.utcnow(),
            is_final=is_final
        )

        # Enforce max entries limit
        if len(self.entries) >= self.max_entries:
            # Remove oldest 10% when limit reached
            remove_count = self.max_entries // 10
            self.entries = self.entries[remove_count:]

        self.entries.append(entry)
        return entry

    def get_all_entries(self) -> List[TranscriptEntry]:
        """Get all transcript entries."""
        return self.entries.copy()

    def get_final_entries(self) -> List[TranscriptEntry]:
        """Get only final (non-interim) entries for notes generation."""
        return [e for e in self.entries if e.is_final]

    def format_for_notes(self) -> str:
        """Format transcript for Claude notes generation."""
        lines = []
        for entry in self.get_final_entries():
            time_str = entry.timestamp.strftime("%H:%M:%S")
            lines.append(f"[{time_str}] {entry.speaker}: {entry.text}")
        return "\n".join(lines)

    def entry_count(self) -> int:
        return len(self.entries)


class TranscriptStore:
    """
    Global store for all room transcripts.
    Thread-safe with asyncio locks.
    """

    def __init__(self):
        self._rooms: Dict[str, RoomTranscript] = {}
        self._lock = asyncio.Lock()

    async def get_or_create_room(self, room_name: str) -> RoomTranscript:
        """Get existing room transcript or create new one."""
        async with self._lock:
            if room_name not in self._rooms:
                self._rooms[room_name] = RoomTranscript(room_name=room_name)
            return self._rooms[room_name]

    async def add_entry(
        self,
        room_name: str,
        speaker: str,
        text: str,
        is_final: bool = True
    ) -> TranscriptEntry:
        """Add a transcript entry to a room."""
        room = await self.get_or_create_room(room_name)
        async with self._lock:
            return room.add_entry(speaker, text, is_final)

    async def get_transcript(self, room_name: str) -> Optional[RoomTranscript]:
        """Get transcript for a room."""
        async with self._lock:
            return self._rooms.get(room_name)

    async def get_formatted_transcript(self, room_name: str) -> Optional[str]:
        """Get formatted transcript for notes generation."""
        room = await self.get_transcript(room_name)
        if room:
            return room.format_for_notes()
        return None

    async def clear_room(self, room_name: str) -> Optional[RoomTranscript]:
        """Remove and return a room's transcript."""
        async with self._lock:
            return self._rooms.pop(room_name, None)

    async def room_exists(self, room_name: str) -> bool:
        """Check if a room has an active transcript."""
        async with self._lock:
            return room_name in self._rooms

    async def get_room_stats(self, room_name: str) -> Optional[dict]:
        """Get statistics for a room's transcript."""
        room = await self.get_transcript(room_name)
        if room:
            return {
                "room_name": room_name,
                "entry_count": room.entry_count(),
                "started_at": room.started_at.isoformat(),
                "final_entries": len(room.get_final_entries())
            }
        return None

    async def list_active_rooms(self) -> List[str]:
        """List all rooms with active transcripts."""
        async with self._lock:
            return list(self._rooms.keys())


# Global singleton instance
transcript_store = TranscriptStore()
