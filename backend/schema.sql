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

-- recordings table (for batch transcription pivot)
CREATE TABLE IF NOT EXISTS recordings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id INTEGER NOT NULL,
    egress_id TEXT UNIQUE NOT NULL,
    status TEXT DEFAULT 'recording', -- recording, processing, completed, failed
    audio_url TEXT,
    duration_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (meeting_id) REFERENCES meetings(id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_meetings_room_name ON meetings(room_name);
CREATE INDEX IF NOT EXISTS idx_notes_meeting ON meeting_notes(meeting_id);
CREATE INDEX IF NOT EXISTS idx_recordings_meeting ON recordings(meeting_id);
CREATE INDEX IF NOT EXISTS idx_recordings_egress ON recordings(egress_id);

-- email_subscriptions table (for meeting summary emails)
CREATE TABLE IF NOT EXISTS email_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id INTEGER NOT NULL,
    participant_name TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (meeting_id) REFERENCES meetings(id),
    UNIQUE(meeting_id, email)
);

CREATE INDEX IF NOT EXISTS idx_email_subs_meeting ON email_subscriptions(meeting_id);
