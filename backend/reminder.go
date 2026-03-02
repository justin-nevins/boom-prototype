package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

// StartReminderJob launches a background goroutine that checks for upcoming
// meetings every minute and sends reminder emails ~15 min before start.
func StartReminderJob() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		log.Println("Reminder job started, checking every 1 minute")
		for range ticker.C {
			checkAndSendReminders()
		}
	}()
}

func checkAndSendReminders() {
	now := time.Now()
	// Window: meetings between 14 and 16 minutes from now
	windowStart := now.Add(14 * time.Minute)
	windowEnd := now.Add(16 * time.Minute)

	meetings, err := GetUpcomingMeetingsForReminder(windowStart, windowEnd)
	if err != nil {
		log.Printf("Reminder job: failed to query meetings: %v", err)
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	for _, m := range meetings {
		inviteLink := fmt.Sprintf("%s/join/%s", frontendURL, m.RoomName)
		meeting := m // capture for safety
		if err := SendReminderEmail(&meeting, inviteLink); err != nil {
			log.Printf("Reminder job: failed to send reminder for meeting %d: %v", meeting.ID, err)
			continue
		}
		if err := MarkReminderSent(meeting.ID); err != nil {
			log.Printf("Reminder job: failed to mark reminder sent for %d: %v", meeting.ID, err)
		} else {
			log.Printf("Reminder sent for meeting %d (%s) scheduled at %s",
				meeting.ID, meeting.RoomName, meeting.ScheduledAt.Format(time.RFC3339))
		}
	}
}
