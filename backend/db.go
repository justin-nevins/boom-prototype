package main

import (
	"database/sql"
	_ "embed"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

var db *sql.DB

func initDB() error {
	var err error
	db, err = sql.Open("sqlite", "./boom.db")
	if err != nil {
		return err
	}

	// Enable WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return err
	}

	// Run schema migrations
	_, err = db.Exec(schemaSQL)
	if err != nil {
		return err
	}

	log.Println("Database initialized")
	return nil
}

// Meeting represents a meeting record
type Meeting struct {
	ID        int64      `json:"id"`
	RoomName  string     `json:"roomName"`
	RoomSID   string     `json:"roomSid"`
	CreatedAt time.Time  `json:"createdAt"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

// MeetingNotes represents generated notes for a meeting
type MeetingNotes struct {
	ID           int64     `json:"id"`
	MeetingID    int64     `json:"meetingId"`
	Markdown     string    `json:"markdown"`
	GeneratedAt  time.Time `json:"generatedAt"`
	ModelUsed    string    `json:"modelUsed"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
}

// CreateMeeting inserts a new meeting record
func CreateMeeting(roomName, roomSID string) (*Meeting, error) {
	result, err := db.Exec(
		"INSERT INTO meetings (room_name, room_sid) VALUES (?, ?) ON CONFLICT(room_name) DO UPDATE SET room_sid = ?",
		roomName, roomSID, roomSID,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Meeting{
		ID:        id,
		RoomName:  roomName,
		RoomSID:   roomSID,
		CreatedAt: time.Now(),
	}, nil
}

// GetMeetingByRoom retrieves a meeting by room name
func GetMeetingByRoom(roomName string) (*Meeting, error) {
	var m Meeting
	var endedAt sql.NullTime
	err := db.QueryRow(
		"SELECT id, room_name, room_sid, created_at, ended_at FROM meetings WHERE room_name = ?",
		roomName,
	).Scan(&m.ID, &m.RoomName, &m.RoomSID, &m.CreatedAt, &endedAt)
	if err != nil {
		return nil, err
	}
	if endedAt.Valid {
		m.EndedAt = &endedAt.Time
	}
	return &m, nil
}

// SaveNotes stores generated notes for a meeting
func SaveNotes(roomName string, markdown string, model string, inputTokens, outputTokens int) (*MeetingNotes, error) {
	// Get or create meeting
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		// Create meeting if not exists
		meeting, err = CreateMeeting(roomName, "")
		if err != nil {
			return nil, err
		}
	}

	result, err := db.Exec(
		"INSERT INTO meeting_notes (meeting_id, notes_markdown, model_used, input_tokens, output_tokens) VALUES (?, ?, ?, ?, ?)",
		meeting.ID, markdown, model, inputTokens, outputTokens,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &MeetingNotes{
		ID:           id,
		MeetingID:    meeting.ID,
		Markdown:     markdown,
		GeneratedAt:  time.Now(),
		ModelUsed:    model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}

// GetNotesByRoom retrieves the latest notes for a room
func GetNotesByRoom(roomName string) (*MeetingNotes, error) {
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		return nil, err
	}

	var n MeetingNotes
	err = db.QueryRow(
		"SELECT id, meeting_id, notes_markdown, generated_at, model_used, input_tokens, output_tokens FROM meeting_notes WHERE meeting_id = ? ORDER BY generated_at DESC LIMIT 1",
		meeting.ID,
	).Scan(&n.ID, &n.MeetingID, &n.Markdown, &n.GeneratedAt, &n.ModelUsed, &n.InputTokens, &n.OutputTokens)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// ListMeetingsWithNotes returns recent meetings that have notes
func ListMeetingsWithNotes(limit int) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT m.id, m.room_name, m.created_at, n.generated_at, n.model_used
		FROM meetings m
		INNER JOIN meeting_notes n ON m.id = n.meeting_id
		ORDER BY n.generated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var roomName string
		var createdAt, generatedAt time.Time
		var model string
		if err := rows.Scan(&id, &roomName, &createdAt, &generatedAt, &model); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":          id,
			"roomName":    roomName,
			"createdAt":   createdAt,
			"generatedAt": generatedAt,
			"model":       model,
		})
	}
	return results, nil
}
