package main

import (
	"bytes"
	"fmt"
	"html"
	"log"
	"net/smtp"
	"os"
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
