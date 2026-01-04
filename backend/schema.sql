-- Boom Prototype Database Schema

-- meetings table
CREATE TABLE IF NOT EXISTS meetings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_name TEXT UNIQUE NOT NULL,
    room_sid TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    ended_at DATETIME
);

-- meeting_notes table
CREATE TABLE IF NOT EXISTS meeting_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id INTEGER NOT NULL,
    notes_markdown TEXT NOT NULL,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    model_used TEXT DEFAULT 'claude-sonnet-4-20250514',
    input_tokens INTEGER,
    output_tokens INTEGER,
    FOREIGN KEY (meeting_id) REFERENCES meetings(id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_meetings_room_name ON meetings(room_name);
CREATE INDEX IF NOT EXISTS idx_notes_meeting ON meeting_notes(meeting_id);
