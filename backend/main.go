package main

import (
	"bytes"
	"context"
	"log"
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

	roomClient = lksdk.NewRoomServiceClient(livekitHost, apiKey, apiSecret)

	app := fiber.New()

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     os.Getenv("FRONTEND_URL"),
		AllowMethods:     "GET, POST, OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept",
		AllowCredentials: true,
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "backend",
		})
	})

	// Routes
	app.Post("/api/rooms", createRoom)
	app.Post("/api/token", getToken)
	app.Get("/api/rooms/:id", getRoom)

	// Notes API
	app.Post("/api/meetings/:room/notes", saveNotesHandler)
	app.Get("/api/meetings/:room/notes", getNotesHandler)
	app.Get("/api/meetings", listMeetingsHandler)

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

	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     req.RoomName,
	}
	at.AddGrant(grant).
		SetIdentity(req.ParticipantName).
		SetValidFor(24 * time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Trigger transcription bot to join room
	triggerTranscriptionBot(req.RoomName)

	return c.JSON(TokenResponse{Token: token})
}

func triggerTranscriptionBot(roomName string) {
	if aiServiceURL == "" {
		return
	}
	go func() {
		payload := []byte(`{"room_name": "` + roomName + `"}`)
		resp, err := http.Post(aiServiceURL+"/join", "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Printf("Failed to trigger transcription bot: %v", err)
			return
		}
		defer resp.Body.Close()
		log.Printf("Transcription bot triggered for room: %s", roomName)
	}()
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

func generateRoomName() string {
	return "room-" + time.Now().Format("20060102-150405")
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
