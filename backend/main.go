package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

var (
	livekitHost    string
	apiKey         string
	apiSecret      string
	aiServiceURL   string
	roomClient     *lksdk.RoomServiceClient
	egressClient   *lksdk.EgressClient
	transcriptWS   = make(map[string]map[*websocket.Conn]bool) // room -> connections
	transcriptLock sync.RWMutex
)

func validateEnv() {
	required := []string{"LIVEKIT_URL", "LIVEKIT_API_KEY", "LIVEKIT_API_SECRET", "FRONTEND_URL"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			log.Fatalf("Missing required environment variable: %s", key)
		}
	}
}

func main() {
	godotenv.Load()
	validateEnv()

	livekitHost = os.Getenv("LIVEKIT_URL")
	apiKey = os.Getenv("LIVEKIT_API_KEY")
	apiSecret = os.Getenv("LIVEKIT_API_SECRET")
	aiServiceURL = os.Getenv("AI_SERVICE_URL")
	if aiServiceURL == "" {
		aiServiceURL = "http://localhost:8081"
	}

	// Initialize database
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize auth (seed users, set JWT secret)
	initAuth()

	roomClient = lksdk.NewRoomServiceClient(livekitHost, apiKey, apiSecret)
	egressClient = lksdk.NewEgressClient(livekitHost, apiKey, apiSecret)

	app := fiber.New()

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     os.Getenv("FRONTEND_URL"),
		AllowMethods:     "GET, POST, DELETE, OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "backend",
		})
	})

	// Auth routes
	app.Post("/api/auth/login", loginHandler)
	app.Get("/api/auth/me", authRequired(), meHandler)

	// Routes (room creation requires auth)
	app.Post("/api/rooms", authRequired(), createRoom)
	app.Post("/api/token", getToken)
	app.Get("/api/rooms/:id", getRoom)

	// Scheduling routes
	app.Post("/api/scheduled-meetings", authRequired(), createScheduledMeetingHandler)
	app.Get("/api/scheduled-meetings", authRequired(), listScheduledMeetingsHandler)
	app.Delete("/api/scheduled-meetings/:id", authRequired(), cancelScheduledMeetingHandler)
	app.Post("/api/scheduled-meetings/:id/start", authRequired(), startScheduledMeetingHandler)
	app.Get("/api/join/:room", getJoinInfoHandler)

	// Notes API
	app.Post("/api/meetings/:room/notes", saveNotesHandler)
	app.Get("/api/meetings/:room/notes", getNotesHandler)
	app.Get("/api/meetings", listMeetingsHandler)

	// Email subscription API
	app.Post("/api/meetings/:room/subscribe-email", subscribeEmailHandler)
	app.Get("/api/meetings/:room/email-subscriptions", getEmailSubscriptionsHandler)
	app.Delete("/api/meetings/:room/unsubscribe-email", unsubscribeEmailHandler)

	// Real-time transcription API
	app.Post("/api/meetings/:room/start-transcription", startTranscriptionHandler)
	app.Post("/api/meetings/:room/end-transcription", endTranscriptionHandler)
	app.Post("/api/internal/transcript", receiveTranscriptHandler)

	// Egress (recording) API - deprecated, kept for backwards compatibility
	app.Post("/api/meetings/:room/start-recording", startRecordingHandler)
	app.Post("/api/meetings/:room/stop-recording", stopRecordingHandler)
	app.Get("/api/meetings/:room/recording-status", getRecordingStatusHandler)

	// WebSocket for transcription broadcast
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/transcription/:room", websocket.New(handleTranscriptionWS))

	// Graceful shutdown
	go func() {
		log.Println("Backend starting on :8080")
		if err := app.Listen(":8080"); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	app.Shutdown()
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

type CreateRoomResponse struct {
	RoomName string `json:"roomName"`
	RoomID   string `json:"roomId"`
}

func createRoom(c *fiber.Ctx) error {
	var req CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	roomName := req.Name
	if roomName == "" {
		roomName = generateRoomName()
	}

	room, err := roomClient.CreateRoom(context.Background(), &livekit.CreateRoomRequest{
		Name:            roomName,
		EmptyTimeout:    10 * 60, // 10 minutes
		MaxParticipants: 50,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(CreateRoomResponse{
		RoomName: room.Name,
		RoomID:   room.Sid,
	})
}

type TokenRequest struct {
	RoomName        string `json:"roomName"`
	ParticipantName string `json:"participantName"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

func getToken(c *fiber.Ctx) error {
	var req TokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Use unique identity per connection so multiple devices can join as the same name
	identity := fmt.Sprintf("%s-%d", req.ParticipantName, rand.Intn(100000))

	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     req.RoomName,
	}
	at.AddGrant(grant).
		SetIdentity(identity).
		SetName(req.ParticipantName).
		SetValidFor(24 * time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(TokenResponse{Token: token})
}

// Egress (Recording) Handlers

func startRecordingHandler(c *fiber.Ctx) error {
	roomName := c.Params("room")

	// Get or create meeting
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		meeting, err = CreateMeeting(roomName, "")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create meeting"})
		}
	}

	// Check if already recording
	existingRec, _ := GetActiveRecordingByMeeting(meeting.ID)
	if existingRec != nil {
		return c.JSON(fiber.Map{
			"status":   "already_recording",
			"egressId": existingRec.EgressID,
		})
	}

	// Start room composite egress (audio only for transcription)
	egressReq := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		AudioOnly: true,
		Output: &livekit.RoomCompositeEgressRequest_File{
			File: &livekit.EncodedFileOutput{
				FileType: livekit.EncodedFileType_OGG,
				Filepath: roomName + "-{time}.ogg",
			},
		},
	}

	info, err := egressClient.StartRoomCompositeEgress(context.Background(), egressReq)
	if err != nil {
		log.Printf("Failed to start egress: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Save recording to database
	rec, err := CreateRecording(meeting.ID, info.EgressId)
	if err != nil {
		log.Printf("Failed to save recording: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save recording"})
	}

	log.Printf("Started recording for room %s, egress ID: %s", roomName, info.EgressId)

	return c.JSON(fiber.Map{
		"status":      "recording",
		"egressId":    info.EgressId,
		"recordingId": rec.ID,
	})
}

func stopRecordingHandler(c *fiber.Ctx) error {
	roomName := c.Params("room")

	// Get meeting
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Meeting not found"})
	}

	// Get active recording
	rec, err := GetActiveRecordingByMeeting(meeting.ID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "No active recording"})
	}

	// Stop egress
	info, err := egressClient.StopEgress(context.Background(), &livekit.StopEgressRequest{
		EgressId: rec.EgressID,
	})
	if err != nil {
		log.Printf("Failed to stop egress: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Extract file URL from egress result
	var audioURL string
	var durationMS int64
	if info.GetFile() != nil {
		audioURL = info.GetFile().Location
		durationMS = info.GetFile().Duration / 1000000 // nanoseconds to ms
	}

	// Update recording status
	UpdateRecordingStatus(rec.EgressID, "processing", audioURL, durationMS)

	log.Printf("Stopped recording for room %s, audio URL: %s", roomName, audioURL)

	// Trigger batch transcription in AI service
	go func() {
		if aiServiceURL == "" {
			return
		}
		payload := []byte(`{"room_name": "` + roomName + `", "audio_url": "` + audioURL + `", "egress_id": "` + rec.EgressID + `"}`)
		resp, err := http.Post(aiServiceURL+"/transcribe-recording", "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Printf("Failed to trigger batch transcription: %v", err)
			UpdateRecordingStatus(rec.EgressID, "failed", audioURL, durationMS)
			return
		}
		defer resp.Body.Close()
		log.Printf("Batch transcription triggered for room: %s", roomName)
	}()

	return c.JSON(fiber.Map{
		"status":     "processing",
		"egressId":   rec.EgressID,
		"audioUrl":   audioURL,
		"durationMs": durationMS,
	})
}

func getRecordingStatusHandler(c *fiber.Ctx) error {
	roomName := c.Params("room")

	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Meeting not found"})
	}

	rec, err := GetActiveRecordingByMeeting(meeting.ID)
	if err != nil {
		// Check for completed recordings
		return c.JSON(fiber.Map{"status": "no_recording"})
	}

	return c.JSON(fiber.Map{
		"status":     rec.Status,
		"egressId":   rec.EgressID,
		"audioUrl":   rec.AudioURL,
		"durationMs": rec.DurationMS,
	})
}

// Real-time transcription handlers

func startTranscriptionHandler(c *fiber.Ctx) error {
	roomName := c.Params("room")

	// Get or create meeting
	meeting, err := GetMeetingByRoom(roomName)
	if err != nil {
		meeting, err = CreateMeeting(roomName, "")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create meeting"})
		}
	}

	// Call AI service to join the room
	payload := []byte(`{"room_name": "` + roomName + `"}`)
	resp, err := http.Post(aiServiceURL+"/join", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Failed to start transcription: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to connect to AI service"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Status(500).JSON(fiber.Map{"error": "AI service failed to join room"})
	}

	log.Printf("Started transcription for room %s, meeting ID: %d", roomName, meeting.ID)

	return c.JSON(fiber.Map{
		"status":    "transcribing",
		"roomName":  roomName,
		"meetingId": meeting.ID,
	})
}

func endTranscriptionHandler(c *fiber.Ctx) error {
	roomName := c.Params("room")

	// Call AI service to leave the room and generate notes
	payload := []byte(`{"room_name": "` + roomName + `"}`)
	resp, err := http.Post(aiServiceURL+"/leave", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Failed to end transcription: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to connect to AI service"})
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return c.Status(404).JSON(fiber.Map{"error": "Room not active"})
	}

	if resp.StatusCode != 200 {
		return c.Status(500).JSON(fiber.Map{"error": "AI service failed to process notes"})
	}

	log.Printf("Ended transcription for room %s, notes should be saved automatically", roomName)

	return c.JSON(fiber.Map{
		"status":   "processing",
		"roomName": roomName,
	})
}

// TranscriptMessage represents an incoming transcript from AI service
type TranscriptMessage struct {
	RoomName  string `json:"room_name"`
	Speaker   string `json:"speaker"`
	Text      string `json:"text"`
	IsFinal   bool   `json:"is_final"`
	Timestamp string `json:"timestamp"`
}

func receiveTranscriptHandler(c *fiber.Ctx) error {
	var msg TranscriptMessage
	if err := c.BodyParser(&msg); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Broadcast to all WebSocket clients for this room
	broadcastJSON := []byte(`{"speaker":"` + msg.Speaker + `","text":"` + msg.Text + `","is_final":` + boolToString(msg.IsFinal) + `,"timestamp":"` + msg.Timestamp + `"}`)
	broadcastToRoom(msg.RoomName, broadcastJSON)

	return c.JSON(fiber.Map{"status": "broadcast"})
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func getRoom(c *fiber.Ctx) error {
	roomID := c.Params("id")

	rooms, err := roomClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{
		Names: []string{roomID},
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if len(rooms.Rooms) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Room not found"})
	}

	room := rooms.Rooms[0]
	return c.JSON(fiber.Map{
		"name":         room.Name,
		"sid":          room.Sid,
		"participants": room.NumParticipants,
	})
}

func handleTranscriptionWS(c *websocket.Conn) {
	room := c.Params("room")

	// Register connection with mutex
	transcriptLock.Lock()
	if transcriptWS[room] == nil {
		transcriptWS[room] = make(map[*websocket.Conn]bool)
	}
	transcriptWS[room][c] = true
	transcriptLock.Unlock()

	defer func() {
		transcriptLock.Lock()
		delete(transcriptWS[room], c)
		transcriptLock.Unlock()
		c.Close()
	}()

	// Keep connection alive, receive messages from AI service
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		// Broadcast to all clients in room
		broadcastToRoom(room, msg)
	}
}

func broadcastToRoom(room string, msg []byte) {
	transcriptLock.RLock()
	defer transcriptLock.RUnlock()
	for conn := range transcriptWS[room] {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}

// Scheduling handlers

type CreateScheduledMeetingRequest struct {
	ClientName  string `json:"clientName"`
	ClientEmail string `json:"clientEmail"`
	ScheduledAt string `json:"scheduledAt"` // ISO 8601
}

func createScheduledMeetingHandler(c *fiber.Ctx) error {
	var req CreateScheduledMeetingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid date format, use ISO 8601"})
	}

	hostUserID := c.Locals("userID").(int64)
	roomName := generateRoomName()

	meeting, err := CreateScheduledMeeting(roomName, hostUserID, req.ClientName, req.ClientEmail, scheduledAt)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create scheduled meeting"})
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	inviteLink := fmt.Sprintf("%s/join/%s", frontendURL, roomName)

	return c.JSON(fiber.Map{
		"id":          meeting.ID,
		"roomName":    meeting.RoomName,
		"scheduledAt": meeting.ScheduledAt,
		"inviteLink":  inviteLink,
		"clientName":  meeting.ClientName,
		"clientEmail": meeting.ClientEmail,
	})
}

func listScheduledMeetingsHandler(c *fiber.Ctx) error {
	hostUserID := c.Locals("userID").(int64)

	meetings, err := ListScheduledMeetingsByHost(hostUserID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if meetings == nil {
		meetings = []ScheduledMeeting{}
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	var results []fiber.Map
	for _, m := range meetings {
		results = append(results, fiber.Map{
			"id":          m.ID,
			"roomName":    m.RoomName,
			"clientName":  m.ClientName,
			"clientEmail": m.ClientEmail,
			"scheduledAt": m.ScheduledAt,
			"status":      m.Status,
			"inviteLink":  fmt.Sprintf("%s/join/%s", frontendURL, m.RoomName),
		})
	}
	if results == nil {
		results = []fiber.Map{}
	}

	return c.JSON(results)
}

func cancelScheduledMeetingHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	var id int64
	fmt.Sscanf(idStr, "%d", &id)

	hostUserID := c.Locals("userID").(int64)

	if err := CancelScheduledMeeting(id, hostUserID); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "cancelled"})
}

func startScheduledMeetingHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	var id int64
	fmt.Sscanf(idStr, "%d", &id)

	hostUserID := c.Locals("userID").(int64)

	// Get the scheduled meeting
	var roomName string
	var meetingHostID int64
	err := db.QueryRow("SELECT room_name, host_user_id FROM scheduled_meetings WHERE id = ? AND status = 'scheduled'", id).Scan(&roomName, &meetingHostID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Scheduled meeting not found"})
	}
	if meetingHostID != hostUserID {
		return c.Status(403).JSON(fiber.Map{"error": "Not your meeting"})
	}

	// Create the LiveKit room
	room, err := roomClient.CreateRoom(context.Background(), &livekit.CreateRoomRequest{
		Name:            roomName,
		EmptyTimeout:    10 * 60,
		MaxParticipants: 50,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Update status to active
	UpdateScheduledMeetingStatus(id, "active")

	return c.JSON(fiber.Map{
		"status":   "active",
		"roomName": room.Name,
		"roomId":   room.Sid,
	})
}

func getJoinInfoHandler(c *fiber.Ctx) error {
	roomName := c.Params("room")

	meeting, err := GetScheduledMeetingByRoom(roomName)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Meeting not found"})
	}

	return c.JSON(fiber.Map{
		"roomName":    meeting.RoomName,
		"hostName":    meeting.HostName,
		"clientName":  meeting.ClientName,
		"scheduledAt": meeting.ScheduledAt,
		"status":      meeting.Status,
	})
}

var verbs = []string{
	"flying", "jumping", "running", "dancing", "singing",
	"cooking", "painting", "reading", "writing", "building",
	"sailing", "climbing", "glowing", "spinning", "drifting",
	"roaming", "floating", "shining", "rolling", "charging",
	"blazing", "cruising", "soaring", "surfing", "hiking",
	"fishing", "mixing", "coding", "gaming", "racing",
}

var nouns = []string{
	"falcon", "tiger", "dolphin", "phoenix", "panther",
	"rocket", "comet", "summit", "canyon", "river",
	"garden", "castle", "forest", "island", "ocean",
	"crystal", "thunder", "breeze", "sunset", "meadow",
	"glacier", "volcano", "nebula", "aurora", "horizon",
	"compass", "lantern", "anchor", "bridge", "beacon",
}

func generateRoomName() string {
	verb := verbs[rand.Intn(len(verbs))]
	noun := nouns[rand.Intn(len(nouns))]
	return verb + "-" + noun
}

// Notes API handlers

type SaveNotesRequest struct {
	Markdown     string `json:"markdown"`
	Model        string `json:"model"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

func saveNotesHandler(c *fiber.Ctx) error {
	room := c.Params("room")
	var req SaveNotesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	notes, err := SaveNotes(room, req.Markdown, req.Model, req.InputTokens, req.OutputTokens)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Trigger email workflow in background (non-blocking)
	go TriggerEmailWorkflow(room, req.Markdown)

	return c.JSON(fiber.Map{
		"status": "saved",
		"id":     notes.ID,
	})
}

func getNotesHandler(c *fiber.Ctx) error {
	room := c.Params("room")

	notes, err := GetNotesByRoom(room)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Notes not found"})
	}

	return c.JSON(notes)
}

func listMeetingsHandler(c *fiber.Ctx) error {
	meetings, err := ListMeetingsWithNotes(20)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(meetings)
}

// Email subscription handlers

type SubscribeEmailRequest struct {
	Email           string `json:"email"`
	ParticipantName string `json:"participantName"`
}

func subscribeEmailHandler(c *fiber.Ctx) error {
	room := c.Params("room")
	var req SubscribeEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Email == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Email is required"})
	}

	sub, err := CreateEmailSubscription(room, req.ParticipantName, req.Email)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status": "subscribed",
		"id":     sub.ID,
		"email":  sub.Email,
	})
}

func getEmailSubscriptionsHandler(c *fiber.Ctx) error {
	room := c.Params("room")

	subs, err := GetEmailSubscriptionsByRoom(room)
	if err != nil {
		return c.JSON(fiber.Map{
			"subscriptions": []EmailSubscription{},
			"count":         0,
		})
	}

	return c.JSON(fiber.Map{
		"subscriptions": subs,
		"count":         len(subs),
	})
}

type UnsubscribeEmailRequest struct {
	Email string `json:"email"`
}

func unsubscribeEmailHandler(c *fiber.Ctx) error {
	room := c.Params("room")
	var req UnsubscribeEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if err := DeleteEmailSubscription(room, req.Email); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "unsubscribed"})
}
