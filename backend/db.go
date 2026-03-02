package main

import (
	"database/sql"
	_ "embed"
	"fmt"
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

	// Idempotent migration: add reminder_sent column for scheduling reminders
	db.Exec("ALTER TABLE scheduled_meetings ADD COLUMN reminder_sent INTEGER DEFAULT 0")

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

// ScheduledMeeting represents a future meeting created by a host
type ScheduledMeeting struct {
	ID          int64     `json:"id"`
	RoomName    string    `json:"roomName"`
	HostUserID  int64     `json:"hostUserId"`
	HostName    string    `json:"hostName,omitempty"`
	HostEmail   string    `json:"-"`
	ClientName  string    `json:"clientName"`
	ClientEmail string    `json:"clientEmail"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	Attendees   []MeetingAttendee `json:"attendees,omitempty"`
}

// MeetingAttendee represents an attendee for a scheduled meeting
type MeetingAttendee struct {
	ID        int64     `json:"id"`
	MeetingID int64     `json:"meetingId"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateScheduledMeeting inserts a new scheduled meeting
func CreateScheduledMeeting(roomName string, hostUserID int64, clientName, clientEmail string, scheduledAt time.Time) (*ScheduledMeeting, error) {
	result, err := db.Exec(
		"INSERT INTO scheduled_meetings (room_name, host_user_id, client_name, client_email, scheduled_at) VALUES (?, ?, ?, ?, ?)",
		roomName, hostUserID, clientName, clientEmail, scheduledAt,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &ScheduledMeeting{
		ID:          id,
		RoomName:    roomName,
		HostUserID:  hostUserID,
		ClientName:  clientName,
		ClientEmail: clientEmail,
		ScheduledAt: scheduledAt,
		Status:      "scheduled",
		CreatedAt:   time.Now(),
	}, nil
}

// GetScheduledMeetingByRoom retrieves a scheduled meeting by room name
func GetScheduledMeetingByRoom(roomName string) (*ScheduledMeeting, error) {
	var m ScheduledMeeting
	var hostName, hostEmail string
	err := db.QueryRow(
		`SELECT sm.id, sm.room_name, sm.host_user_id, u.name, u.email, sm.client_name, sm.client_email, sm.scheduled_at, sm.status, sm.created_at
		 FROM scheduled_meetings sm
		 JOIN users u ON sm.host_user_id = u.id
		 WHERE sm.room_name = ?`,
		roomName,
	).Scan(&m.ID, &m.RoomName, &m.HostUserID, &hostName, &hostEmail, &m.ClientName, &m.ClientEmail, &m.ScheduledAt, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	m.HostName = hostName
	m.HostEmail = hostEmail
	m.Attendees, _ = GetAttendeesByMeeting(m.ID)
	return &m, nil
}

// ListScheduledMeetingsByHost returns scheduled meetings for a host
func ListScheduledMeetingsByHost(hostUserID int64) ([]ScheduledMeeting, error) {
	rows, err := db.Query(
		`SELECT sm.id, sm.room_name, sm.host_user_id, u.name, u.email, sm.client_name, sm.client_email, sm.scheduled_at, sm.status, sm.created_at
		 FROM scheduled_meetings sm
		 JOIN users u ON sm.host_user_id = u.id
		 WHERE sm.host_user_id = ? AND sm.status IN ('scheduled', 'active')
		 ORDER BY sm.scheduled_at ASC`,
		hostUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []ScheduledMeeting
	for rows.Next() {
		var m ScheduledMeeting
		var hostName, hostEmail string
		if err := rows.Scan(&m.ID, &m.RoomName, &m.HostUserID, &hostName, &hostEmail, &m.ClientName, &m.ClientEmail, &m.ScheduledAt, &m.Status, &m.CreatedAt); err != nil {
			continue
		}
		m.HostName = hostName
		m.HostEmail = hostEmail
		m.Attendees, _ = GetAttendeesByMeeting(m.ID)
		meetings = append(meetings, m)
	}
	return meetings, nil
}

// UpdateScheduledMeetingStatus updates the status of a scheduled meeting
func UpdateScheduledMeetingStatus(id int64, status string) error {
	_, err := db.Exec("UPDATE scheduled_meetings SET status = ? WHERE id = ?", status, id)
	return err
}

// CancelScheduledMeeting cancels a scheduled meeting owned by the given user
func CancelScheduledMeeting(id, hostUserID int64) (*ScheduledMeeting, error) {
	// First fetch the meeting details for email notification
	var m ScheduledMeeting
	var hostName, hostEmail string
	err := db.QueryRow(
		`SELECT sm.id, sm.room_name, sm.host_user_id, u.name, u.email, sm.client_name, sm.client_email, sm.scheduled_at, sm.status, sm.created_at
		 FROM scheduled_meetings sm
		 JOIN users u ON sm.host_user_id = u.id
		 WHERE sm.id = ? AND sm.host_user_id = ?`,
		id, hostUserID,
	).Scan(&m.ID, &m.RoomName, &m.HostUserID, &hostName, &hostEmail, &m.ClientName, &m.ClientEmail, &m.ScheduledAt, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("meeting not found or not owned by user")
	}
	m.HostName = hostName
	m.HostEmail = hostEmail
	m.Attendees, _ = GetAttendeesByMeeting(m.ID)

	result, err := db.Exec("UPDATE scheduled_meetings SET status = 'cancelled' WHERE id = ? AND host_user_id = ?", id, hostUserID)
	if err != nil {
		return nil, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("meeting not found or not owned by user")
	}
	m.Status = "cancelled"
	return &m, nil
}

// CreateMeetingAttendees inserts attendees for a scheduled meeting
func CreateMeetingAttendees(meetingID int64, attendees []MeetingAttendee) error {
	for _, a := range attendees {
		_, err := db.Exec(
			"INSERT OR IGNORE INTO meeting_attendees (meeting_id, name, email) VALUES (?, ?, ?)",
			meetingID, a.Name, a.Email,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAttendeesByMeeting returns all attendees for a scheduled meeting
func GetAttendeesByMeeting(meetingID int64) ([]MeetingAttendee, error) {
	rows, err := db.Query(
		"SELECT id, meeting_id, name, email, created_at FROM meeting_attendees WHERE meeting_id = ? ORDER BY id ASC",
		meetingID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attendees []MeetingAttendee
	for rows.Next() {
		var a MeetingAttendee
		var email sql.NullString
		if err := rows.Scan(&a.ID, &a.MeetingID, &a.Name, &email, &a.CreatedAt); err != nil {
			continue
		}
		if email.Valid {
			a.Email = email.String
		}
		attendees = append(attendees, a)
	}
	return attendees, nil
}

// DeleteAttendeesByMeeting removes all attendees for a meeting
func DeleteAttendeesByMeeting(meetingID int64) error {
	_, err := db.Exec("DELETE FROM meeting_attendees WHERE meeting_id = ?", meetingID)
	return err
}

// TranscriptChunk represents a persisted transcript chunk
type TranscriptChunk struct {
	ID             int64     `json:"id"`
	MeetingID      int64     `json:"meetingId"`
	ChunkIndex     int       `json:"chunkIndex"`
	TranscriptText string    `json:"transcriptText"`
	StartTime      time.Time `json:"startTime"`
	EndTime        time.Time `json:"endTime"`
	EntryCount     int       `json:"entryCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

// SaveTranscriptChunk saves a transcript chunk to the database
func SaveTranscriptChunk(roomName string, chunkIndex int, text string, startTime, endTime time.Time, entryCount int) (*TranscriptChunk, error) {
	// Get or create meeting
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		meeting, err = CreateMeeting(roomName, "")
		if err != nil {
			return nil, err
		}
	}

	result, err := db.Exec(
		`INSERT INTO transcript_chunks (meeting_id, chunk_index, transcript_text, start_time, end_time, entry_count)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(meeting_id, chunk_index) DO UPDATE SET
		 transcript_text = ?, start_time = ?, end_time = ?, entry_count = ?`,
		meeting.ID, chunkIndex, text, startTime, endTime, entryCount,
		text, startTime, endTime, entryCount,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &TranscriptChunk{
		ID:             id,
		MeetingID:      meeting.ID,
		ChunkIndex:     chunkIndex,
		TranscriptText: text,
		StartTime:      startTime,
		EndTime:        endTime,
		EntryCount:     entryCount,
		CreatedAt:      time.Now(),
	}, nil
}

// GetTranscriptChunks retrieves all chunks for a meeting ordered by chunk_index
func GetTranscriptChunks(roomName string) ([]TranscriptChunk, error) {
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT id, meeting_id, chunk_index, transcript_text, start_time, end_time, entry_count, created_at
		 FROM transcript_chunks
		 WHERE meeting_id = ?
		 ORDER BY chunk_index ASC`,
		meeting.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []TranscriptChunk
	for rows.Next() {
		var c TranscriptChunk
		if err := rows.Scan(&c.ID, &c.MeetingID, &c.ChunkIndex, &c.TranscriptText, &c.StartTime, &c.EndTime, &c.EntryCount, &c.CreatedAt); err != nil {
			continue
		}
		chunks = append(chunks, c)
	}
	return chunks, nil
}

// DeleteTranscriptChunks removes all chunks for a meeting
func DeleteTranscriptChunks(roomName string) error {
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM transcript_chunks WHERE meeting_id = ?", meeting.ID)
	return err
}

// GetUserByEmail retrieves a user by email address
func GetUserByEmail(email string) (*User, error) {
	var u User
	err := db.QueryRow(
		"SELECT id, email, name, created_at FROM users WHERE email = ?", email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUpcomingMeetingsForReminder returns scheduled meetings in the given time window that haven't had reminders sent
func GetUpcomingMeetingsForReminder(windowStart, windowEnd time.Time) ([]ScheduledMeeting, error) {
	rows, err := db.Query(
		`SELECT sm.id, sm.room_name, sm.host_user_id, u.name, u.email, sm.client_name, sm.client_email, sm.scheduled_at, sm.status, sm.created_at
		 FROM scheduled_meetings sm
		 JOIN users u ON sm.host_user_id = u.id
		 WHERE sm.status = 'scheduled'
		   AND sm.reminder_sent = 0
		   AND sm.scheduled_at >= ?
		   AND sm.scheduled_at <= ?`,
		windowStart, windowEnd,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []ScheduledMeeting
	for rows.Next() {
		var m ScheduledMeeting
		var hostName, hostEmail string
		if err := rows.Scan(&m.ID, &m.RoomName, &m.HostUserID, &hostName, &hostEmail, &m.ClientName, &m.ClientEmail, &m.ScheduledAt, &m.Status, &m.CreatedAt); err != nil {
			continue
		}
		m.HostName = hostName
		m.HostEmail = hostEmail
		m.Attendees, _ = GetAttendeesByMeeting(m.ID)
		meetings = append(meetings, m)
	}
	return meetings, nil
}

// MarkReminderSent marks a scheduled meeting's reminder as sent
func MarkReminderSent(id int64) error {
	_, err := db.Exec("UPDATE scheduled_meetings SET reminder_sent = 1 WHERE id = ?", id)
	return err
}
