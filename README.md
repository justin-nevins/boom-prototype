# Boom - Video Conferencing Prototype

A working video conferencing prototype built in 10 hours using LiveKit + AI transcription.

## Features

- ✅ Video calls (1:1 and group up to 50)
- ✅ Screen sharing
- ✅ Text chat
- ✅ **Real-time transcription**
- ✅ Mobile-responsive web
- ✅ Shareable meeting links

## Quick Start (5 minutes)

### 1. Get API Keys

1. **LiveKit Cloud** (free): https://cloud.livekit.io
   - Create project → Copy API Key & Secret

2. **Deepgram** (free tier): https://deepgram.com
   - Create account → Copy API Key

### 2. Setup

```bash
# Clone
git clone https://github.com/your-org/boom-prototype
cd boom-prototype

# Configure
cp .env.example .env
# Edit .env with your API keys

# Run
docker-compose up
```

### 3. Open

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │────▶│   Backend   │────▶│  LiveKit    │
│   (React)   │     │    (Go)     │     │   Cloud     │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐     ┌─────────────┐
                    │ AI Service  │────▶│  Deepgram   │
                    │  (Python)   │     │     API     │
                    └─────────────┘     └─────────────┘
```

## Development

### Frontend (React + LiveKit)

```bash
cd frontend
npm install
npm run dev
```

### Backend (Go)

```bash
cd backend
go mod download
go run main.go
```

### AI Service (Python)

```bash
cd ai-service
pip install -r requirements.txt
python main.py
```

## Deployment

### Frontend → Vercel

```bash
cd frontend
vercel --prod
```

### Backend → Fly.io

```bash
cd backend
fly launch
fly deploy
```

### AI Service → Fly.io

```bash
cd ai-service
fly launch
fly deploy
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/rooms` | Create a new room |
| POST | `/api/token` | Get LiveKit access token |
| GET | `/api/rooms/:id` | Get room details |
| WS | `/ws/transcription/:room` | Transcription stream |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `LIVEKIT_API_KEY` | LiveKit API key |
| `LIVEKIT_API_SECRET` | LiveKit API secret |
| `LIVEKIT_URL` | LiveKit WebSocket URL |
| `DEEPGRAM_API_KEY` | Deepgram API key |

## Project Structure

```
boom-prototype/
├── frontend/          # React + LiveKit UI
├── backend/           # Go API server
├── ai-service/        # Python transcription bot
├── infrastructure/    # Docker, deployment configs
├── docs/              # Documentation
└── docker-compose.yml
```

## Next Steps (Post-Prototype)

- [ ] User authentication (Clerk/Auth0)
- [ ] Meeting scheduling
- [ ] Recording & playback
- [ ] E2EE encryption
- [ ] Native mobile apps
- [ ] Virtual backgrounds

## License

MIT
