# 10-Hour Video Conferencing Prototype

**Strategy: Build on LiveKit (open source) + Custom UI + AI Transcription**

Don't reinvent WebRTC. Use LiveKit's battle-tested SFU and focus human time on differentiation.

---

## Architecture: Maximum Leverage

```
┌─────────────────────────────────────────────────────────────┐
│                    WHAT WE BUILD (10 hrs)                   │
├─────────────────────────────────────────────────────────────┤
│  Custom React UI  │  AI Transcription  │  Simple Backend   │
│  (3 hours)        │  (2 hours)         │  (1 hour)         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                 WHAT WE USE (Open Source)                   │
├─────────────────────────────────────────────────────────────┤
│  LiveKit Server   │  LiveKit SDKs  │  Whisper API/Deepgram │
│  (SFU, Signaling) │  (WebRTC)      │  (Speech-to-text)     │
└─────────────────────────────────────────────────────────────┘
```

**Why LiveKit:**
- Open source SFU (Apache 2.0)
- Free cloud tier for prototyping
- SDKs for Web, React, iOS, Android, Flutter
- Built-in: screen share, recording, simulcast
- 5 minutes to first video call

---

## Agent Deployment (6 Parallel Agents)

```
┌─────────────────────────────────────────────────────────────┐
│                      HUMAN (You)                            │
│         Setup accounts, test, deploy, 10 hrs total          │
└─────────────────────────┬───────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
   ┌─────────┐      ┌─────────┐      ┌─────────┐
   │ Agent 1 │      │ Agent 2 │      │ Agent 3 │
   │ Frontend│      │ Backend │      │   AI    │
   │ (React) │      │  (Go)   │      │(Python) │
   └─────────┘      └─────────┘      └─────────┘
        │                 │                 │
        ▼                 ▼                 ▼
   ┌─────────┐      ┌─────────┐      ┌─────────┐
   │ Agent 4 │      │ Agent 5 │      │ Agent 6 │
   │ Mobile  │      │  Infra  │      │  Docs   │
   │(React N)│      │(Docker) │      │  /Demo  │
   └─────────┘      └─────────┘      └─────────┘
```

---

## Hour-by-Hour Execution Plan

### Hour 0: Setup (Human - 30 min)

```bash
# 1. Create LiveKit Cloud account (free tier)
# https://cloud.livekit.io → Get API Key + Secret

# 2. Create project structure
mkdir boom-prototype && cd boom-prototype
mkdir -p frontend backend ai-service infrastructure

# 3. Set environment variables
export LIVEKIT_API_KEY="your-api-key"
export LIVEKIT_API_SECRET="your-api-secret"
export LIVEKIT_URL="wss://your-project.livekit.cloud"

# 4. Start all agents with their assigned directories
```

---

### Hours 1-3: Frontend Agent (React + LiveKit)

**Agent 1 Prompt:**
```
Build a video conferencing web app using React + LiveKit.

Tech: React 18, TypeScript, Tailwind CSS, @livekit/components-react

Requirements:
1. Landing page with "Create Meeting" and "Join Meeting" buttons
2. Meeting room with:
   - Video grid (auto-layout for 1-49 participants)
   - Mute/unmute audio button
   - Camera on/off button
   - Screen share button
   - Leave meeting button
   - Chat sidebar
   - Live transcription overlay (receives from WebSocket)
3. Pre-join screen (camera preview, name input)
4. Mobile responsive

Use LiveKit's React components as base, customize styling.
Output to: /frontend

Key files to create:
- src/App.tsx (routing)
- src/pages/Home.tsx
- src/pages/Room.tsx
- src/components/VideoGrid.tsx
- src/components/Controls.tsx
- src/components/Chat.tsx
- src/components/Transcription.tsx
- src/hooks/useTranscription.ts (WebSocket to AI service)
```

**Agent 1 delivers:** Complete React app, ~2000 lines

---

### Hours 1-2: Backend Agent (Go + LiveKit Server SDK)

**Agent 2 Prompt:**
```
Build a minimal backend for video conferencing using Go + LiveKit Server SDK.

Tech: Go 1.21+, Fiber/Chi, LiveKit Server SDK, SQLite

Endpoints needed:
1. POST /api/rooms - Create a room, return room name
2. POST /api/token - Generate LiveKit access token
   - Input: room_name, participant_name
   - Output: JWT token for LiveKit connection
3. GET /api/rooms/:id - Get room info
4. WebSocket /ws/transcription/:room - Broadcast transcriptions to room

Requirements:
- Stateless (can use in-memory or SQLite for room metadata)
- CORS enabled for frontend
- Environment variables for LiveKit credentials

Output to: /backend

Key files:
- main.go
- handlers/room.go
- handlers/token.go
- handlers/websocket.go
- livekit/client.go
```

**Agent 2 delivers:** ~500 lines Go backend

---

### Hours 1-2: AI Agent (Real-time Transcription)

**Agent 3 Prompt:**
```
Build a real-time transcription service that listens to LiveKit audio tracks.

Tech: Python 3.11+, faster-whisper (local) OR Deepgram SDK (cloud)

Architecture:
1. Connect to LiveKit room as a hidden participant (bot)
2. Subscribe to all audio tracks
3. Buffer audio in 3-second chunks
4. Send to Whisper/Deepgram for transcription
5. Broadcast results via WebSocket to backend

Option A - Local Whisper (free, needs GPU):
- Use faster-whisper with whisper-small model
- ~1GB VRAM required

Option B - Deepgram (cloud, has free tier):
- Real-time streaming API
- Better for prototype demo

Output to: /ai-service

Key files:
- main.py
- transcriber.py
- livekit_bot.py
- requirements.txt

Include Dockerfile for easy deployment.
```

**Agent 3 delivers:** ~400 lines Python service

---

### Hour 2-3: Mobile Agent (React Native or Skip)

**Agent 4 Prompt (Optional - only if time permits):**
```
Create a React Native app using LiveKit React Native SDK.

If time is limited, just create:
1. Expo project scaffold
2. Single screen that joins a meeting
3. Basic video grid

This can be a "phase 2" item. Web prototype is priority.

Output to: /mobile
```

**Agent 4 delivers:** Basic scaffold OR skipped for web focus

---

### Hour 2-3: Infrastructure Agent

**Agent 5 Prompt:**
```
Create deployment configuration for the prototype.

Deliverables:
1. docker-compose.yml - Run all services locally
2. Dockerfile for each service (frontend, backend, ai-service)
3. fly.toml - Fly.io deployment config
4. Vercel config for frontend (vercel.json)
5. Simple deploy script

Requirements:
- Frontend on Vercel (free)
- Backend on Fly.io (free tier)
- AI service on Fly.io or Railway
- Use LiveKit Cloud (no self-hosting for prototype)

Output to: /infrastructure

Also create:
- README.md with setup instructions
- .env.example
```

**Agent 5 delivers:** Complete deployment configs

---

### Hour 3: Integration & Testing (Human + All Agents)

```
Human tasks (30 min):
├── Run docker-compose up locally
├── Test video call with 2 browser tabs
├── Verify transcription appears
├── Fix any CORS or WebSocket issues
└── Agents fix bugs in real-time
```

---

### Hour 4-5: Polish Agent (Docs + Demo)

**Agent 6 Prompt:**
```
Create documentation and demo materials.

Deliverables:
1. README.md - Complete setup guide
2. DEMO_SCRIPT.md - 5-minute demo walkthrough
3. Landing page copy (hero, features, CTA)
4. Simple logo (SVG, text-based is fine)
5. Screenshot placeholders / descriptions

Also create:
- API documentation (OpenAPI spec)
- Architecture diagram (Mermaid)

Output to: /docs and update /frontend landing page
```

---

## Final Directory Structure

```
boom-prototype/
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Home.tsx
│   │   │   └── Room.tsx
│   │   ├── components/
│   │   │   ├── VideoGrid.tsx
│   │   │   ├── Controls.tsx
│   │   │   ├── Chat.tsx
│   │   │   └── Transcription.tsx
│   │   └── App.tsx
│   ├── package.json
│   ├── Dockerfile
│   └── vercel.json
│
├── backend/
│   ├── main.go
│   ├── handlers/
│   ├── go.mod
│   └── Dockerfile
│
├── ai-service/
│   ├── main.py
│   ├── transcriber.py
│   ├── requirements.txt
│   └── Dockerfile
│
├── infrastructure/
│   ├── docker-compose.yml
│   ├── fly.toml
│   └── deploy.sh
│
├── docs/
│   ├── README.md
│   ├── DEMO_SCRIPT.md
│   ├── API.md
│   └── ARCHITECTURE.md
│
└── .env.example
```

---

## Hour-by-Hour Summary

| Hour | Human Activity | Agent Activity |
|------|---------------|----------------|
| 0 | Setup accounts, env vars | - |
| 1 | Monitor agents | Frontend + Backend + AI start |
| 2 | Light review | All agents building |
| 3 | Integration testing | Agents fix bugs |
| 4 | Manual QA | Polish + docs |
| 5 | Deploy to Vercel/Fly | Infra agent assists |
| 6 | Test production | Bug fixes |
| 7 | Demo prep | Demo script |
| 8 | Buffer / features | Optional mobile |
| 9 | Final testing | Final fixes |
| 10 | Launch! | - |

---

## Prototype Features (What You Get)

### Included (10 hours):
- ✅ Video calls (1:1 and group up to 50)
- ✅ Screen sharing
- ✅ Text chat
- ✅ **Real-time transcription** (differentiator!)
- ✅ Mobile-responsive web
- ✅ Shareable meeting links
- ✅ Mute/camera controls
- ✅ Deployed and accessible

### Not Included (add later):
- ❌ User accounts / authentication
- ❌ Meeting scheduling
- ❌ Recording
- ❌ E2EE (LiveKit has it, just not configured)
- ❌ Native mobile apps
- ❌ Virtual backgrounds
- ❌ Breakout rooms

---

## Cost: $0-50

| Service | Cost |
|---------|------|
| LiveKit Cloud | Free tier (100 participants/month) |
| Vercel | Free tier |
| Fly.io | Free tier ($5 credit) |
| Deepgram | Free tier (12,000 mins) |
| Domain (optional) | $10-15 |
| **Total** | **$0-15** |

---

## Quick Start Commands

```bash
# Clone and setup
git clone https://github.com/your-org/boom-prototype
cd boom-prototype
cp .env.example .env
# Edit .env with your LiveKit credentials

# Run locally
docker-compose up

# Deploy
cd frontend && vercel --prod
cd ../backend && fly deploy
cd ../ai-service && fly deploy

# Test
open https://boom-prototype.vercel.app
```

---

## Agent Prompts (Copy-Paste Ready)

### Start All Agents Simultaneously

**Terminal 1 - Frontend Agent:**
```
claude --directory ./frontend "Build a video conferencing React app using LiveKit. Include: landing page, meeting room with video grid, controls (mute, camera, screenshare, leave), chat sidebar, and a transcription overlay that receives text via WebSocket from ws://localhost:8080/ws/transcription/{roomId}. Use @livekit/components-react, Tailwind CSS, TypeScript. Make it look modern and clean, not like a generic template."
```

**Terminal 2 - Backend Agent:**
```
claude --directory ./backend "Build a Go backend with: POST /api/rooms (create room), POST /api/token (generate LiveKit JWT), WebSocket /ws/transcription/:room (broadcast transcriptions). Use Fiber, LiveKit Server SDK. Read LIVEKIT_API_KEY and LIVEKIT_API_SECRET from env. Include CORS middleware. Keep it minimal."
```

**Terminal 3 - AI Agent:**
```
claude --directory ./ai-service "Build a Python transcription bot that: 1) Joins a LiveKit room as a bot participant, 2) Subscribes to audio tracks, 3) Buffers 3-second chunks, 4) Sends to Deepgram streaming API, 5) Broadcasts results to backend WebSocket. Use livekit Python SDK and deepgram-sdk. Include Dockerfile."
```

**Terminal 4 - Infra Agent:**
```
claude --directory ./infrastructure "Create: docker-compose.yml for local dev (frontend:3000, backend:8080, ai:8081), Dockerfiles for each service, fly.toml for backend and ai-service, vercel.json for frontend. Include .env.example and a deploy.sh script."
```

---

## Success Criteria

After 10 hours, you should be able to:

1. **Share a link** → Friend joins your video call
2. **See each other** → Video works on both sides
3. **Talk** → Audio is clear
4. **Read transcription** → What you say appears as text
5. **Share screen** → Show a presentation
6. **Chat** → Send text messages
7. **Works on phone** → Mobile browser support

**That's a working prototype.** Everything else is polish.
