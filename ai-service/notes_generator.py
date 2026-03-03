"""
Meeting Notes Generator using Claude API

Generates structured markdown notes from meeting transcripts.
Supports both single-pass and chunked generation for long meetings.
Supports multiple note structure types (basic, working_group).
"""

import os
import asyncio
import logging
from typing import List
from anthropic import AsyncAnthropic

logger = logging.getLogger("notes-generator")

ANTHROPIC_API_KEY = os.getenv("ANTHROPIC_API_KEY")

# --- Basic (default) prompts ---

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

CHUNK_SUMMARY_PROMPT = """You are a meeting notes assistant. Given a portion of a meeting transcript, extract the key points discussed.

Output a brief summary with:
- Main topics discussed (2-5 bullet points)
- Any decisions or action items mentioned
- Key statements or quotes worth noting

Keep it concise - this will be combined with other chunk summaries later."""

CONSOLIDATION_PROMPT = """You are a meeting notes assistant. You have been given summaries of different portions of a long meeting. Consolidate these into comprehensive meeting notes.

Structure your notes with:
1. **Meeting Summary** - 2-3 sentence overview of what was discussed
2. **Key Discussion Points** - Main topics covered with brief details
3. **Decisions Made** - Any decisions or conclusions reached
4. **Action Items** - Tasks assigned with owners if mentioned (use checkboxes: - [ ])
5. **Follow-ups** - Items that need future attention or discussion

Guidelines:
- Combine related topics from different chunks
- Remove redundancy while preserving important details
- Use bullet points for readability
- Highlight important items with **bold**
- If no action items or decisions were made, you can omit those sections"""

# --- Working Group prompts ---

WORKING_GROUP_SYSTEM_PROMPT = """You are a meeting minutes recorder for a startup accelerator Working Group session. These are weekly ~2-hour meetings where founders share updates, give feedback, and review sprint deliverables.

Structure the meeting minutes as follows:

# Working Group Meeting Minutes

## Attendance
List all participants identified in the transcript.

## Previous Challenges Review
Summarize any discussion of challenges or issues raised in prior meetings. If none discussed, write "No previous challenges reviewed."

## Founder Updates
For EACH founder who speaks, create a subsection:
### [Founder Name]
- **Success**: Their one key win or thing going well
- **Challenge**: Their one key challenge or area needing improvement
- **Group Discussion**: Key feedback, suggestions, or perspectives shared by the group

## Sprint Deliverables Review
Summarize discussion of current sprint tasks, deliverables, or weekly assignments. Include status updates and any feedback given.

## Open Discussion
Any additional topics, networking conversation, or items discussed outside the structured agenda.

## Items for Next Meeting
- Action items with owners if mentioned (use checkboxes: - [ ])
- Topics to follow up on
- Anything flagged for review at the next Working Group session

Guidelines:
- Attribute statements to specific founders by name
- Focus on capturing the substance of feedback and discussion, not just surface-level updates
- Highlight important insights with **bold**
- Be thorough on challenges and group discussion - this is the most valuable part
- If a section has no content from the transcript, include it with "Not discussed" rather than omitting it"""

WORKING_GROUP_CHUNK_SUMMARY_PROMPT = """You are a meeting minutes recorder for a startup accelerator Working Group session. Given a portion of the meeting transcript, extract key information.

Focus on capturing:
- Which founders spoke and their success/challenge updates
- Group feedback and discussion points
- Any sprint deliverable discussion
- Action items or follow-ups mentioned

Keep it concise - this will be combined with other chunk summaries later."""

WORKING_GROUP_CONSOLIDATION_PROMPT = """You are a meeting minutes recorder for a startup accelerator Working Group session. You have been given summaries of different portions of a long meeting. Consolidate these into comprehensive meeting minutes.

Structure the minutes as follows:

# Working Group Meeting Minutes

## Attendance
List all participants identified across all chunks.

## Previous Challenges Review
Summarize any discussion of challenges or issues raised in prior meetings. If none discussed, write "No previous challenges reviewed."

## Founder Updates
For EACH founder who spoke, create a subsection:
### [Founder Name]
- **Success**: Their one key win or thing going well
- **Challenge**: Their one key challenge or area needing improvement
- **Group Discussion**: Key feedback, suggestions, or perspectives shared by the group

## Sprint Deliverables Review
Summarize discussion of current sprint tasks, deliverables, or weekly assignments.

## Open Discussion
Any additional topics discussed outside the structured agenda.

## Items for Next Meeting
- Action items with owners if mentioned (use checkboxes: - [ ])
- Topics to follow up on
- Anything flagged for review at the next Working Group session

Guidelines:
- Combine related information from different chunks
- Remove redundancy while preserving important details
- Attribute statements to specific founders by name
- If a section has no content, include it with "Not discussed" rather than omitting it"""


def get_prompts(note_type: str = "basic") -> dict:
    """Return the prompt set for the given note type."""
    if note_type == "working_group":
        return {
            "system": WORKING_GROUP_SYSTEM_PROMPT,
            "chunk": WORKING_GROUP_CHUNK_SUMMARY_PROMPT,
            "consolidation": WORKING_GROUP_CONSOLIDATION_PROMPT,
        }
    return {
        "system": SYSTEM_PROMPT,
        "chunk": CHUNK_SUMMARY_PROMPT,
        "consolidation": CONSOLIDATION_PROMPT,
    }


async def generate_notes(transcript: list[dict], note_type: str = "basic") -> dict:
    """
    Generate meeting notes from transcript using Claude.

    Args:
        transcript: List of transcript entries with speaker, text, timestamp
        note_type: Type of notes to generate ("basic" or "working_group")

    Returns:
        dict with markdown notes and token usage
    """
    if not ANTHROPIC_API_KEY:
        raise ValueError("ANTHROPIC_API_KEY not set")

    client = AsyncAnthropic(api_key=ANTHROPIC_API_KEY)
    prompts = get_prompts(note_type)

    # Format transcript for Claude
    formatted = format_transcript(transcript)

    logger.info(f"Generating {note_type} notes from {len(transcript)} transcript entries")

    message = await client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=8192,
        system=prompts["system"],
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


async def generate_notes_from_text(formatted_transcript: str, note_type: str = "basic") -> dict:
    """
    Generate meeting notes from pre-formatted transcript string.

    Args:
        formatted_transcript: Pre-formatted transcript string
                             (e.g., "[HH:MM:SS] Speaker: text\\n...")
        note_type: Type of notes to generate ("basic" or "working_group")

    Returns:
        dict with markdown notes and token usage
    """
    if not ANTHROPIC_API_KEY:
        raise ValueError("ANTHROPIC_API_KEY not set")

    if not formatted_transcript or not formatted_transcript.strip():
        return {
            "markdown": "# Meeting Notes\n\nNo transcript available for this meeting.",
            "model": "claude-opus-4-6",
            "usage": {"input_tokens": 0, "output_tokens": 0}
        }

    client = AsyncAnthropic(api_key=ANTHROPIC_API_KEY)
    prompts = get_prompts(note_type)

    logger.info(f"Generating {note_type} notes from {len(formatted_transcript)} chars of transcript")

    message = await client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=8192,
        system=prompts["system"],
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


# Chunked generation for long meetings

async def generate_chunk_summary(chunk_text: str, chunk_index: int, note_type: str = "basic") -> dict:
    """
    Generate a brief summary of a single transcript chunk.

    This is faster than full notes generation and uses less tokens.
    Used for long meetings that need to be processed in parts.
    """
    if not ANTHROPIC_API_KEY:
        raise ValueError("ANTHROPIC_API_KEY not set")

    if not chunk_text or not chunk_text.strip():
        return {
            "summary": f"[Chunk {chunk_index}: No transcript content]",
            "chunk_index": chunk_index,
            "usage": {"input_tokens": 0, "output_tokens": 0}
        }

    client = AsyncAnthropic(api_key=ANTHROPIC_API_KEY)
    prompts = get_prompts(note_type)

    logger.info(f"Generating summary for chunk {chunk_index} ({len(chunk_text)} chars)")

    message = await client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=1024,  # Shorter output for chunk summaries
        system=prompts["chunk"],
        messages=[
            {
                "role": "user",
                "content": f"Summarize this portion of the meeting (chunk {chunk_index + 1}):\n\n{chunk_text}"
            }
        ]
    )

    return {
        "summary": message.content[0].text,
        "chunk_index": chunk_index,
        "usage": {
            "input_tokens": message.usage.input_tokens,
            "output_tokens": message.usage.output_tokens
        }
    }


async def generate_notes_from_chunks(chunks: List[dict], note_type: str = "basic") -> dict:
    """
    Generate meeting notes from multiple transcript chunks.

    Two-pass approach:
    1. Generate summary for each chunk (in parallel)
    2. Consolidate chunk summaries into final notes

    Args:
        chunks: List of dicts with 'transcriptText' and 'chunkIndex' keys
        note_type: Type of notes to generate ("basic" or "working_group")

    Returns:
        dict with markdown notes and total token usage
    """
    if not ANTHROPIC_API_KEY:
        raise ValueError("ANTHROPIC_API_KEY not set")

    if not chunks:
        return {
            "markdown": "# Meeting Notes\n\nNo transcript chunks available.",
            "model": "claude-sonnet-4-20250514",
            "usage": {"input_tokens": 0, "output_tokens": 0}
        }

    logger.info(f"Generating {note_type} notes from {len(chunks)} chunks")

    # Pass 1: Generate summaries for all chunks in parallel
    summary_tasks = [
        generate_chunk_summary(
            chunk.get("transcriptText", chunk.get("transcript_text", "")),
            chunk.get("chunkIndex", chunk.get("chunk_index", i)),
            note_type,
        )
        for i, chunk in enumerate(chunks)
    ]

    chunk_results = await asyncio.gather(*summary_tasks, return_exceptions=True)

    # Collect successful summaries
    summaries = []
    total_input_tokens = 0
    total_output_tokens = 0

    for result in chunk_results:
        if isinstance(result, Exception):
            logger.error(f"Chunk summary failed: {result}")
            continue
        summaries.append(f"### Chunk {result['chunk_index'] + 1}\n{result['summary']}")
        total_input_tokens += result["usage"]["input_tokens"]
        total_output_tokens += result["usage"]["output_tokens"]

    if not summaries:
        return {
            "markdown": "# Meeting Notes\n\nFailed to process transcript chunks.",
            "model": "claude-sonnet-4-20250514",
            "usage": {"input_tokens": total_input_tokens, "output_tokens": total_output_tokens}
        }

    # Pass 2: Consolidate summaries into final notes
    combined_summaries = "\n\n".join(summaries)

    client = AsyncAnthropic(api_key=ANTHROPIC_API_KEY)
    prompts = get_prompts(note_type)

    logger.info(f"Consolidating {len(summaries)} chunk summaries into final notes")

    message = await client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=8192,
        system=prompts["consolidation"],
        messages=[
            {
                "role": "user",
                "content": f"Consolidate these meeting chunk summaries into comprehensive notes:\n\n{combined_summaries}"
            }
        ]
    )

    total_input_tokens += message.usage.input_tokens
    total_output_tokens += message.usage.output_tokens

    return {
        "markdown": message.content[0].text,
        "model": "claude-sonnet-4-20250514",
        "chunks_processed": len(summaries),
        "usage": {
            "input_tokens": total_input_tokens,
            "output_tokens": total_output_tokens
        }
    }
