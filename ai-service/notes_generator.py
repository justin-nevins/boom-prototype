"""
Meeting Notes Generator using Claude API

Generates structured markdown notes from meeting transcripts.
"""

import os
import logging
from anthropic import AsyncAnthropic

logger = logging.getLogger("notes-generator")

ANTHROPIC_API_KEY = os.getenv("ANTHROPIC_API_KEY")

SYSTEM_PROMPT = """You are a meeting notes assistant. Given a meeting transcript, generate clear, well-organized notes in Markdown format.

Structure your notes with:
1. **Meeting Summary** - 2-3 sentence overview of what was discussed
2. **Key Discussion Points** - Main topics covered with brief details
3. **Decisions Made** - Any decisions or conclusions reached
4. **Action Items** - Tasks assigned with owners if mentioned (use checkboxes: - [ ])
5. **Follow-ups** - Items that need future attention or discussion

Guidelines:
- Be concise but comprehensive
- Use bullet points for readability
- Include speaker names when attributing specific statements or decisions
- Highlight important items with **bold**
- If no action items or decisions were made, you can omit those sections
- Format timestamps as readable times if they add context
- Group related topics together logically"""


async def generate_notes(transcript: list[dict]) -> dict:
    """
    Generate meeting notes from transcript using Claude.

    Args:
        transcript: List of transcript entries with speaker, text, timestamp

    Returns:
        dict with markdown notes and token usage
    """
    if not ANTHROPIC_API_KEY:
        raise ValueError("ANTHROPIC_API_KEY not set")

    client = AsyncAnthropic(api_key=ANTHROPIC_API_KEY)

    # Format transcript for Claude
    formatted = format_transcript(transcript)

    logger.info(f"Generating notes from {len(transcript)} transcript entries")

    message = await client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=2000,
        system=SYSTEM_PROMPT,
        messages=[
            {
                "role": "user",
                "content": f"Generate meeting notes from this transcript:\n\n{formatted}"
            }
        ]
    )

    return {
        "markdown": message.content[0].text,
        "model": "claude-sonnet-4-20250514",
        "usage": {
            "input_tokens": message.usage.input_tokens,
            "output_tokens": message.usage.output_tokens
        }
    }


def format_transcript(entries: list[dict]) -> str:
    """Format transcript entries into readable text for Claude."""
    lines = []
    for entry in entries:
        timestamp = entry.get("timestamp", "")
        # Simplify timestamp to just time
        if "T" in timestamp:
            timestamp = timestamp.split("T")[1][:8]
        speaker = entry.get("speaker", "Unknown")
        text = entry.get("text", "")
        lines.append(f"[{timestamp}] {speaker}: {text}")
    return "\n".join(lines)


async def generate_notes_from_text(formatted_transcript: str) -> dict:
    """
    Generate meeting notes from pre-formatted transcript string.

    Args:
        formatted_transcript: Pre-formatted transcript string
                             (e.g., "[HH:MM:SS] Speaker: text\\n...")

    Returns:
        dict with markdown notes and token usage
    """
    if not ANTHROPIC_API_KEY:
        raise ValueError("ANTHROPIC_API_KEY not set")

    if not formatted_transcript or not formatted_transcript.strip():
        return {
            "markdown": "# Meeting Notes\n\nNo transcript available for this meeting.",
            "model": "claude-sonnet-4-20250514",
            "usage": {"input_tokens": 0, "output_tokens": 0}
        }

    client = AsyncAnthropic(api_key=ANTHROPIC_API_KEY)

    logger.info(f"Generating notes from {len(formatted_transcript)} chars of transcript")

    message = await client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=2000,
        system=SYSTEM_PROMPT,
        messages=[
            {
                "role": "user",
                "content": f"Generate meeting notes from this transcript:\n\n{formatted_transcript}"
            }
        ]
    )

    return {
        "markdown": message.content[0].text,
        "model": "claude-sonnet-4-20250514",
        "usage": {
            "input_tokens": message.usage.input_tokens,
            "output_tokens": message.usage.output_tokens
        }
    }
