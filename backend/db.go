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

// Recording represents a meeting recording for batch transcription
type Recording struct {
	ID          int64      `json:"id"`
	MeetingID   int64      `json:"meetingId"`
	EgressID    string     `json:"egressId"`
	Status      string     `json:"status"` // recording, processing, completed, failed
	AudioURL    string     `json:"audioUrl,omitempty"`
	DurationMS  int64      `json:"durationMs,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// CreateRecording inserts a new recording record
func CreateRecording(meetingID int64, egressID string) (*Recording, error) {
	result, err := db.Exec(
		"INSERT INTO recordings (meeting_id, egress_id, status) VALUES (?, ?, 'recording')",
		meetingID, egressID,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Recording{
		ID:        id,
		MeetingID: meetingID,
		EgressID:  egressID,
		Status:    "recording",
		CreatedAt: time.Now(),
	}, nil
}

// GetRecordingByEgressID retrieves a recording by egress ID
func GetRecordingByEgressID(egressID string) (*Recording, error) {
	var r Recording
	var audioURL sql.NullString
	var durationMS sql.NullInt64
	var completedAt sql.NullTime

	err := db.QueryRow(
		"SELECT id, meeting_id, egress_id, status, audio_url, duration_ms, created_at, completed_at FROM recordings WHERE egress_id = ?",
		egressID,
	).Scan(&r.ID, &r.MeetingID, &r.EgressID, &r.Status, &audioURL, &durationMS, &r.CreatedAt, &completedAt)
	if err != nil {
		return nil, err
	}

	if audioURL.Valid {
		r.AudioURL = audioURL.String
	}
	if durationMS.Valid {
		r.DurationMS = durationMS.Int64
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	return &r, nil
}

// GetActiveRecordingByMeeting retrieves the active recording for a meeting
func GetActiveRecordingByMeeting(meetingID int64) (*Recording, error) {
	var r Recording
	var audioURL sql.NullString
	var durationMS sql.NullInt64
	var completedAt sql.NullTime

	err := db.QueryRow(
		"SELECT id, meeting_id, egress_id, status, audio_url, duration_ms, created_at, completed_at FROM recordings WHERE meeting_id = ? AND status = 'recording' ORDER BY created_at DESC LIMIT 1",
		meetingID,
	).Scan(&r.ID, &r.MeetingID, &r.EgressID, &r.Status, &audioURL, &durationMS, &r.CreatedAt, &completedAt)
	if err != nil {
		return nil, err
	}

	if audioURL.Valid {
		r.AudioURL = audioURL.String
	}
	if durationMS.Valid {
		r.DurationMS = durationMS.Int64
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	return &r, nil
}

// UpdateRecordingStatus updates a recording's status
func UpdateRecordingStatus(egressID, status string, audioURL string, durationMS int64) error {
	if status == "completed" || status == "failed" {
		_, err := db.Exec(
			"UPDATE recordings SET status = ?, audio_url = ?, duration_ms = ?, completed_at = CURRENT_TIMESTAMP WHERE egress_id = ?",
			status, audioURL, durationMS, egressID,
		)
		return err
	}
	_, err := db.Exec("UPDATE recordings SET status = ? WHERE egress_id = ?", status, egressID)
	return err
}

// EmailSubscription represents a participant's email subscription for meeting summaries
type EmailSubscription struct {
	ID              int64     `json:"id"`
	MeetingID       int64     `json:"meetingId"`
	ParticipantName string    `json:"participantName"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"createdAt"`
}

// CreateEmailSubscription adds an email subscription for a meeting
func CreateEmailSubscription(roomName, participantName, email string) (*EmailSubscription, error) {
	// Get or create meeting
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		meeting, err = CreateMeeting(roomName, "")
		if err != nil {
			return nil, err
		}
	}

	result, err := db.Exec(
		"INSERT INTO email_subscriptions (meeting_id, participant_name, email) VALUES (?, ?, ?) ON CONFLICT(meeting_id, email) DO UPDATE SET participant_name = ?",
		meeting.ID, participantName, email, participantName,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &EmailSubscription{
		ID:              id,
		MeetingID:       meeting.ID,
		ParticipantName: participantName,
		Email:           email,
		CreatedAt:       time.Now(),
	}, nil
}

// GetEmailSubscriptionsByRoom retrieves all email subscriptions for a room
func GetEmailSubscriptionsByRoom(roomName string) ([]EmailSubscription, error) {
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		"SELECT id, meeting_id, participant_name, email, created_at FROM email_subscriptions WHERE meeting_id = ?",
		meeting.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []EmailSubscription
	for rows.Next() {
		var s EmailSubscription
		if err := rows.Scan(&s.ID, &s.MeetingID, &s.ParticipantName, &s.Email, &s.CreatedAt); err != nil {
			continue
		}
		subs = append(subs, s)
	}
	return subs, nil
}

// DeleteEmailSubscription removes an email subscription
func DeleteEmailSubscription(roomName, email string) error {
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM email_subscriptions WHERE meeting_id = ? AND email = ?", meeting.ID, email)
	return err
}
