package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// N8NEmailPayload is the payload sent to n8n webhook for email delivery
type N8NEmailPayload struct {
	RoomName   string               `json:"roomName"`
	Notes      string               `json:"notes"`
	Timestamp  string               `json:"timestamp"`
	Recipients []EmailSubscription  `json:"recipients"`
}

// TriggerEmailWorkflow sends meeting summary to n8n for email delivery
func TriggerEmailWorkflow(roomName string, notes string) error {
	webhookURL := os.Getenv("N8N_EMAIL_WEBHOOK_URL")
	if webhookURL == "" {
		log.Println("N8N_EMAIL_WEBHOOK_URL not set, skipping email trigger")
		return nil
	}

	// Get all email subscriptions for this room
	subs, err := GetEmailSubscriptionsByRoom(roomName)
	if err != nil || len(subs) == 0 {
		log.Printf("No email subscriptions for room %s", roomName)
		return nil
	}

	payload := N8NEmailPayload{
		RoomName:   roomName,
		Notes:      notes,
		Timestamp:  time.Now().Format(time.RFC3339),
		Recipients: subs,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Failed to trigger n8n email workflow: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Email workflow triggered for room %s, %d recipients", roomName, len(subs))
	} else {
		log.Printf("n8n webhook returned status %d", resp.StatusCode)
	}

	return nil
}
