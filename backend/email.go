package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/yuin/goldmark"
)

// TriggerEmailWorkflow sends meeting summary emails via SMTP
func TriggerEmailWorkflow(roomName string, notes string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	smtpFrom := os.Getenv("SMTP_FROM")

	if smtpHost == "" || smtpUser == "" || smtpPass == "" {
		log.Println("SMTP not configured, skipping email")
		return nil
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if smtpFrom == "" {
		smtpFrom = smtpUser
	}

	subs, err := GetEmailSubscriptionsByRoom(roomName)
	if err != nil || len(subs) == 0 {
		log.Printf("No email subscriptions for room %s", roomName)
		return nil
	}

	timestamp := time.Now().Format("Jan 2, 2006 at 3:04 PM")
	htmlBody := buildEmailHTML(roomName, notes, timestamp)
	subject := fmt.Sprintf("Meeting Notes: %s", roomName)

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	addr := smtpHost + ":" + smtpPort

	for _, sub := range subs {
		msg := buildMIMEMessage(smtpFrom, sub.Email, sub.ParticipantName, subject, htmlBody)
		err := smtp.SendMail(addr, auth, smtpFrom, []string{sub.Email}, msg)
		if err != nil {
			log.Printf("Failed to send email to %s: %v", sub.Email, err)
			continue
		}
		log.Printf("Email sent to %s for room %s", sub.Email, roomName)
	}

	return nil
}

func buildMIMEMessage(from, to, toName, subject, htmlBody string) []byte {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: Boom Meeting Notes <%s>\r\n", from))
	if toName != "" {
		buf.WriteString(fmt.Sprintf("To: %s <%s>\r\n", toName, to))
	} else {
		buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(htmlBody)
	return buf.Bytes()
}

func buildEmailHTML(roomName string, markdownNotes string, timestamp string) string {
	// Convert markdown to HTML using goldmark
	var htmlBuf bytes.Buffer
	notesHTML := ""
	if err := goldmark.Convert([]byte(markdownNotes), &htmlBuf); err != nil {
		// Fallback: escaped markdown in <pre>
		notesHTML = fmt.Sprintf("<pre style=\"white-space:pre-wrap;font-family:monospace;font-size:14px;color:#333;\">%s</pre>", html.EscapeString(markdownNotes))
	} else {
		notesHTML = htmlBuf.String()
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background-color:#f4f4f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f7;padding:24px 0;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%;background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">

<!-- Header -->
<tr>
<td style="background:linear-gradient(135deg,#2563eb,#1e40af);padding:32px 40px;">
<h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:600;">Meeting Notes</h1>
<p style="margin:8px 0 0;color:#bfdbfe;font-size:14px;">%s</p>
</td>
</tr>

<!-- Meta -->
<tr>
<td style="padding:24px 40px 0;">
<table width="100%%" cellpadding="0" cellspacing="0">
<tr>
<td style="padding:12px 16px;background-color:#f0f4ff;border-radius:6px;">
<p style="margin:0;font-size:13px;color:#6b7280;">Room</p>
<p style="margin:4px 0 0;font-size:15px;color:#1e293b;font-weight:600;">%s</p>
</td>
</tr>
</table>
</td>
</tr>

<!-- Notes -->
<tr>
<td style="padding:24px 40px;">
<div style="font-size:15px;line-height:1.7;color:#374151;">
%s
</div>
</td>
</tr>

<!-- Footer -->
<tr>
<td style="padding:24px 40px;border-top:1px solid #e5e7eb;">
<p style="margin:0;font-size:12px;color:#9ca3af;text-align:center;">
Sent by Boom &mdash; Video meetings with AI-powered notes
</p>
</td>
</tr>

</table>
</td></tr>
</table>
</body>
</html>`, html.EscapeString(timestamp), html.EscapeString(roomName), notesHTML)
}

// ---------------------------------------------------------------------------
// SMTP Config Helper
// ---------------------------------------------------------------------------

type smtpConfig struct {
	host, port, user, pass, from string
}

func getSMTPConfig() *smtpConfig {
	cfg := &smtpConfig{
		host: os.Getenv("SMTP_HOST"),
		port: os.Getenv("SMTP_PORT"),
		user: os.Getenv("SMTP_USER"),
		pass: os.Getenv("SMTP_PASSWORD"),
		from: os.Getenv("SMTP_FROM"),
	}
	if cfg.port == "" {
		cfg.port = "587"
	}
	if cfg.from == "" {
		cfg.from = cfg.user
	}
	return cfg
}

func (c *smtpConfig) isConfigured() bool {
	return c.host != "" && c.user != "" && c.pass != ""
}

func (c *smtpConfig) send(to, toName, subject, htmlBody string, attachments []emailAttachment) error {
	auth := smtp.PlainAuth("", c.user, c.pass, c.host)
	msg := buildMIMEMessageWithAttachment(c.from, to, toName, subject, htmlBody, attachments)
	return smtp.SendMail(c.host+":"+c.port, auth, c.from, []string{to}, msg)
}

// ---------------------------------------------------------------------------
// ICS Generator
// ---------------------------------------------------------------------------

func generateICS(roomName, summary, description, location string, start, end time.Time, organizerName, organizerEmail string) []byte {
	const icsTime = "20060102T150405Z"
	uid := roomName + "@boom.video"

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Boom Video//Meeting Scheduler//EN\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:REQUEST\r\n")
	b.WriteString("BEGIN:VEVENT\r\n")
	b.WriteString("UID:" + uid + "\r\n")
	b.WriteString("DTSTAMP:" + time.Now().UTC().Format(icsTime) + "\r\n")
	b.WriteString("DTSTART:" + start.UTC().Format(icsTime) + "\r\n")
	b.WriteString("DTEND:" + end.UTC().Format(icsTime) + "\r\n")
	b.WriteString("SUMMARY:" + icsEscape(summary) + "\r\n")
	b.WriteString("DESCRIPTION:" + icsEscape(description) + "\r\n")
	b.WriteString("LOCATION:" + icsEscape(location) + "\r\n")
	b.WriteString(fmt.Sprintf("ORGANIZER;CN=%s:mailto:%s\r\n", icsEscape(organizerName), organizerEmail))
	b.WriteString("STATUS:CONFIRMED\r\n")
	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")
	return []byte(b.String())
}

func icsEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// ---------------------------------------------------------------------------
// MIME with Attachments
// ---------------------------------------------------------------------------

type emailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func buildMIMEMessageWithAttachment(from, to, toName, subject, htmlBody string, attachments []emailAttachment) []byte {
	var buf bytes.Buffer
	boundary := fmt.Sprintf("BoomBoundary%d", time.Now().UnixNano())

	buf.WriteString(fmt.Sprintf("From: Boom Video <%s>\r\n", from))
	if toName != "" {
		buf.WriteString(fmt.Sprintf("To: %s <%s>\r\n", toName, to))
	} else {
		buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) == 0 {
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
		buf.WriteString(htmlBody)
		return buf.Bytes()
	}

	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))

	// HTML part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
	buf.WriteString(htmlBody)
	buf.WriteString("\r\n")

	// Attachment parts
	for _, att := range attachments {
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", att.ContentType))
		buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename))
		buf.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		encoded := base64.StdEncoding.EncodeToString(att.Data)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			buf.WriteString(encoded[i:end] + "\r\n")
		}
	}
	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// Send Functions
// ---------------------------------------------------------------------------

// SendInviteEmail sends a meeting invite to the client with ICS attachment
func SendInviteEmail(meeting *ScheduledMeeting, attendee *MeetingAttendee, inviteLink string) error {
	cfg := getSMTPConfig()
	if !cfg.isConfigured() {
		log.Println("SMTP not configured, skipping invite email")
		return nil
	}

	icsData := generateICS(
		meeting.RoomName,
		fmt.Sprintf("Video Meeting with %s", meeting.HostName),
		fmt.Sprintf("Join at: %s", inviteLink),
		inviteLink,
		meeting.ScheduledAt,
		meeting.ScheduledAt.Add(60*time.Minute),
		meeting.HostName,
		meeting.HostEmail,
	)

	subject := fmt.Sprintf("You're invited to a video meeting with %s", meeting.HostName)
	htmlBody := buildInviteEmailHTML(meeting, inviteLink, false)

	attachment := emailAttachment{
		Filename:    "meeting.ics",
		ContentType: "text/calendar; charset=\"UTF-8\"; method=REQUEST",
		Data:        icsData,
	}

	if err := cfg.send(attendee.Email, attendee.Name, subject, htmlBody, []emailAttachment{attachment}); err != nil {
		log.Printf("Failed to send invite email to %s: %v", attendee.Email, err)
		return err
	}
	log.Printf("Invite email sent to %s", attendee.Email)
	return nil
}

// SendConfirmationEmail sends a confirmation to the host after scheduling
func SendConfirmationEmail(meeting *ScheduledMeeting, inviteLink string) error {
	cfg := getSMTPConfig()
	if !cfg.isConfigured() {
		return nil
	}

	subject := fmt.Sprintf("Meeting scheduled with %s", meeting.ClientName)
	htmlBody := buildInviteEmailHTML(meeting, inviteLink, true)

	if err := cfg.send(meeting.HostEmail, meeting.HostName, subject, htmlBody, nil); err != nil {
		log.Printf("Failed to send confirmation to host %s: %v", meeting.HostEmail, err)
		return err
	}
	log.Printf("Confirmation email sent to host %s", meeting.HostEmail)
	return nil
}

// SendCancellationEmail notifies the client that a meeting was cancelled
func SendCancellationEmail(meeting *ScheduledMeeting, attendee *MeetingAttendee) error {
	cfg := getSMTPConfig()
	if !cfg.isConfigured() || attendee.Email == "" {
		return nil
	}

	subject := fmt.Sprintf("Meeting cancelled — %s", meeting.ScheduledAt.Format("Jan 2 at 3:04 PM"))
	htmlBody := buildCancellationEmailHTML(meeting)

	if err := cfg.send(attendee.Email, attendee.Name, subject, htmlBody, nil); err != nil {
		log.Printf("Failed to send cancellation email to %s: %v", attendee.Email, err)
		return err
	}
	log.Printf("Cancellation email sent to %s", attendee.Email)
	return nil
}

// SendReminderEmail sends a reminder to both host and client ~15 min before meeting
func SendReminderEmail(meeting *ScheduledMeeting, inviteLink string) error {
	cfg := getSMTPConfig()
	if !cfg.isConfigured() {
		return nil
	}

	subject := "Reminder: Your meeting starts in ~15 minutes"

	// Send to all attendees
	for _, a := range meeting.Attendees {
		if a.Email != "" {
			clientHTML := buildReminderEmailHTML(meeting, inviteLink, false)
			if err := cfg.send(a.Email, a.Name, subject, clientHTML, nil); err != nil {
				log.Printf("Failed to send reminder to attendee %s: %v", a.Email, err)
			}
		}
	}

	// Send to host
	hostHTML := buildReminderEmailHTML(meeting, inviteLink, true)
	if err := cfg.send(meeting.HostEmail, meeting.HostName, subject, hostHTML, nil); err != nil {
		log.Printf("Failed to send reminder to host %s: %v", meeting.HostEmail, err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// HTML Template Builders
// ---------------------------------------------------------------------------

func buildInviteEmailHTML(meeting *ScheduledMeeting, inviteLink string, isHost bool) string {
	var heading, intro string
	if isHost {
		heading = "Meeting Confirmed"
		intro = fmt.Sprintf("Your meeting with <strong>%s</strong> (%s) has been scheduled.",
			html.EscapeString(meeting.ClientName), html.EscapeString(meeting.ClientEmail))
	} else {
		heading = "You're Invited"
		intro = fmt.Sprintf("<strong>%s</strong> has invited you to a video meeting.",
			html.EscapeString(meeting.HostName))
	}

	scheduledTime := meeting.ScheduledAt.Format("Monday, January 2, 2006 at 3:04 PM UTC")

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background-color:#f4f4f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f7;padding:24px 0;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%;background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">

<!-- Header -->
<tr>
<td style="background:linear-gradient(135deg,#2563eb,#1e40af);padding:32px 40px;">
<h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:600;">%s</h1>
<p style="margin:8px 0 0;color:#bfdbfe;font-size:14px;">Boom Video Meeting</p>
</td>
</tr>

<!-- Body -->
<tr>
<td style="padding:24px 40px;">
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#374151;">%s</p>

<table width="100%%" cellpadding="0" cellspacing="0" style="margin:16px 0;">
<tr>
<td style="padding:12px 16px;background-color:#f0f4ff;border-radius:6px;">
<p style="margin:0;font-size:13px;color:#6b7280;">When</p>
<p style="margin:4px 0 0;font-size:15px;color:#1e293b;font-weight:600;">%s</p>
</td>
</tr>
<tr><td style="height:8px;"></td></tr>
<tr>
<td style="padding:12px 16px;background-color:#f0f4ff;border-radius:6px;">
<p style="margin:0;font-size:13px;color:#6b7280;">Meeting Room</p>
<p style="margin:4px 0 0;font-size:15px;color:#1e293b;font-weight:600;">%s</p>
</td>
</tr>
</table>

<table width="100%%" cellpadding="0" cellspacing="0" style="margin:24px 0;">
<tr>
<td align="center">
<a href="%s" style="display:inline-block;padding:12px 32px;background-color:#0396A6;color:#ffffff;text-decoration:none;border-radius:6px;font-size:15px;font-weight:600;">Join Meeting</a>
</td>
</tr>
</table>

<p style="margin:16px 0 0;font-size:13px;color:#9ca3af;text-align:center;">Or copy this link: %s</p>
</td>
</tr>

<!-- Footer -->
<tr>
<td style="padding:24px 40px;border-top:1px solid #e5e7eb;">
<p style="margin:0;font-size:12px;color:#9ca3af;text-align:center;">
Sent by Boom &mdash; Video meetings with AI-powered notes
</p>
</td>
</tr>

</table>
</td></tr>
</table>
</body>
</html>`,
		html.EscapeString(heading),
		intro,
		html.EscapeString(scheduledTime),
		html.EscapeString(meeting.RoomName),
		html.EscapeString(inviteLink),
		html.EscapeString(inviteLink),
	)
}

func buildCancellationEmailHTML(meeting *ScheduledMeeting) string {
	scheduledTime := meeting.ScheduledAt.Format("Monday, January 2, 2006 at 3:04 PM UTC")

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background-color:#f4f4f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f7;padding:24px 0;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%;background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">

<!-- Header -->
<tr>
<td style="background:linear-gradient(135deg,#dc2626,#991b1b);padding:32px 40px;">
<h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:600;">Meeting Cancelled</h1>
<p style="margin:8px 0 0;color:#fecaca;font-size:14px;">Boom Video Meeting</p>
</td>
</tr>

<!-- Body -->
<tr>
<td style="padding:24px 40px;">
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#374151;">
Your meeting with <strong>%s</strong> has been cancelled.
</p>

<table width="100%%" cellpadding="0" cellspacing="0" style="margin:16px 0;">
<tr>
<td style="padding:12px 16px;background-color:#fef2f2;border-radius:6px;">
<p style="margin:0;font-size:13px;color:#6b7280;">Originally scheduled</p>
<p style="margin:4px 0 0;font-size:15px;color:#1e293b;font-weight:600;">%s</p>
</td>
</tr>
</table>

<p style="margin:16px 0 0;font-size:14px;color:#6b7280;">
If you believe this was a mistake, please reach out to the meeting host directly.
</p>
</td>
</tr>

<!-- Footer -->
<tr>
<td style="padding:24px 40px;border-top:1px solid #e5e7eb;">
<p style="margin:0;font-size:12px;color:#9ca3af;text-align:center;">
Sent by Boom &mdash; Video meetings with AI-powered notes
</p>
</td>
</tr>

</table>
</td></tr>
</table>
</body>
</html>`,
		html.EscapeString(meeting.HostName),
		html.EscapeString(scheduledTime),
	)
}

func buildReminderEmailHTML(meeting *ScheduledMeeting, inviteLink string, isHost bool) string {
	var withPerson string
	if isHost {
		withPerson = meeting.ClientName
	} else {
		withPerson = meeting.HostName
	}

	scheduledTime := meeting.ScheduledAt.Format("3:04 PM UTC")

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background-color:#f4f4f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f7;padding:24px 0;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%;background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">

<!-- Header -->
<tr>
<td style="background:linear-gradient(135deg,#f59e0b,#d97706);padding:32px 40px;">
<h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:600;">Meeting Reminder</h1>
<p style="margin:8px 0 0;color:#fef3c7;font-size:14px;">Starting in ~15 minutes</p>
</td>
</tr>

<!-- Body -->
<tr>
<td style="padding:24px 40px;">
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#374151;">
Your meeting with <strong>%s</strong> starts at <strong>%s</strong>.
</p>

<table width="100%%" cellpadding="0" cellspacing="0" style="margin:24px 0;">
<tr>
<td align="center">
<a href="%s" style="display:inline-block;padding:12px 32px;background-color:#0396A6;color:#ffffff;text-decoration:none;border-radius:6px;font-size:15px;font-weight:600;">Join Meeting</a>
</td>
</tr>
</table>
</td>
</tr>

<!-- Footer -->
<tr>
<td style="padding:24px 40px;border-top:1px solid #e5e7eb;">
<p style="margin:0;font-size:12px;color:#9ca3af;text-align:center;">
Sent by Boom &mdash; Video meetings with AI-powered notes
</p>
</td>
</tr>

</table>
</td></tr>
</table>
</body>
</html>`,
		html.EscapeString(withPerson),
		html.EscapeString(scheduledTime),
		html.EscapeString(inviteLink),
	)
}
