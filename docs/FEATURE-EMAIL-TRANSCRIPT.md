# Feature: Automatic Email Transcript/Summary Distribution

## Overview
Send meeting transcript summaries to opted-in participants via email when the meeting ends.

## Architecture (n8n Integration)

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Frontend     │     │     Backend     │     │      n8n        │
│  (React Room)   │────▶│   (Go API)      │────▶│   (Webhook)     │
│                 │     │                 │     │                 │
│ Email opt-in UI │     │ Store emails    │     │ Format & Send   │
│ in meeting      │     │ Trigger webhook │     │ via Gmail/SMTP  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Implementation Plan

### Phase 1: Database Schema

**File: `backend/schema.sql`**

```sql
-- Participant email subscriptions for meeting summaries
CREATE TABLE IF NOT EXISTS meeting_email_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id INTEGER NOT NULL,
    participant_name TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (meeting_id) REFERENCES meetings(id),
    UNIQUE(meeting_id, email)
);
```

### Phase 2: Backend API

**File: `backend/main.go`**

Add endpoints:

```go
// Subscribe to email summary
POST /api/meetings/:room/subscribe-email
Body: { "email": "user@example.com", "participantName": "John" }
Response: { "status": "subscribed" }

// Get subscriptions (for debugging)
GET /api/meetings/:room/email-subscriptions
Response: { "subscriptions": [...], "count": 2 }
```

**File: `backend/email.go`** (new)

```go
// N8N_WEBHOOK_URL from environment
func TriggerEmailWorkflow(roomName string, notes string, subscriptions []EmailSubscription) error {
    payload := map[string]interface{}{
        "roomName":      roomName,
        "notes":         notes,
        "timestamp":     time.Now().Format(time.RFC3339),
        "recipients":    subscriptions,
    }

    resp, err := http.Post(os.Getenv("N8N_EMAIL_WEBHOOK_URL"),
        "application/json",
        bytes.NewBuffer(jsonPayload))
    // handle response...
}
```

**Trigger point:** After notes are saved in `handleStopRecording` or when AI service completes.

### Phase 3: Frontend UI

**File: `frontend/src/components/EmailSubscription.tsx`** (new)

```typescript
interface EmailSubscriptionProps {
  roomName: string;
  participantName: string;
}

export default function EmailSubscription({ roomName, participantName }: EmailSubscriptionProps) {
  const [email, setEmail] = useState('');
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [isOpen, setIsOpen] = useState(false);

  // Small envelope icon button in header
  // Click opens popover with:
  // - Email input
  // - "Send me the summary" checkbox
  // - Subscribe button
}
```

**File: `frontend/src/pages/Room.tsx`**

Add to header (next to "Copy invite link"):
```tsx
<EmailSubscription
  roomName={roomName}
  participantName={participantName}
/>
```

### Phase 4: n8n Workflow

**Webhook Trigger** receives:
```json
{
  "roomName": "room-20260105-123456",
  "notes": "## Meeting Summary\n\n### Key Points\n- ...",
  "timestamp": "2026-01-05T19:30:00Z",
  "recipients": [
    { "email": "alice@example.com", "participantName": "Alice" },
    { "email": "bob@example.com", "participantName": "Bob" }
  ]
}
```

**Workflow Steps:**
1. **Webhook** - Receive data
2. **Split In Batches** - Process each recipient
3. **HTML Template** - Format email with notes (Markdown to HTML)
4. **Gmail/SMTP Node** - Send email
5. **Optional: Log to Google Sheet** - Track sent emails

**Email Template:**
```
Subject: Meeting Summary - {{roomName}}

Hi {{participantName}},

Here's the summary from your recent meeting:

{{notes as HTML}}

---
Sent via Boom Video Conferencing
```

### Phase 5: Environment Variables

**Add to `.env`:**
```
N8N_EMAIL_WEBHOOK_URL=https://your-n8n.instance/webhook/boom-email-summary
```

## File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `backend/schema.sql` | Modify | Add email_subscriptions table |
| `backend/db.go` | Modify | Add subscription CRUD functions |
| `backend/main.go` | Modify | Add API endpoints |
| `backend/email.go` | Create | n8n webhook trigger |
| `frontend/src/components/EmailSubscription.tsx` | Create | Email opt-in UI |
| `frontend/src/pages/Room.tsx` | Modify | Add EmailSubscription to header |
| `.env.example` | Modify | Add N8N_EMAIL_WEBHOOK_URL |

## UI Mockup

```
┌─────────────────────────────────────────────────────────┐
│ Boom | room-123    [Background] [📧] [Copy link] [End]  │
│                         │                               │
│                         ▼                               │
│                    ┌──────────────────┐                 │
│                    │ Get Summary Email│                 │
│                    │                  │                 │
│                    │ Email:           │                 │
│                    │ [____________]   │                 │
│                    │                  │                 │
│                    │ [Subscribe]      │                 │
│                    └──────────────────┘                 │
│                                                         │
│    ┌─────────────────────────────────────────────┐     │
│    │                                             │     │
│    │              Video Feed                     │     │
│    │                                             │     │
│    └─────────────────────────────────────────────┘     │
│                                                         │
│    [Mic] [Camera] [Share] [Chat] [Leave]               │
└─────────────────────────────────────────────────────────┘
```

## Implementation Order

1. **Backend first** - Schema + API endpoints + n8n webhook trigger
2. **n8n workflow** - Create and test with curl
3. **Frontend last** - UI component + integration

## Testing Checklist

- [ ] Subscribe during meeting
- [ ] Multiple participants subscribe
- [ ] Webhook fires when meeting ends with notes
- [ ] n8n receives correct payload
- [ ] Email formatted correctly
- [ ] Email delivered successfully
- [ ] Handle invalid emails gracefully
- [ ] Handle n8n downtime gracefully
