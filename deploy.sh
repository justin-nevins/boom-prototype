#!/bin/bash
# Boom Prototype Deployment Script
# Run this after: flyctl auth login

set -e

echo "=== Boom Prototype Deployment ==="
echo ""

# Check if fly is authenticated
if ! flyctl auth whoami >/dev/null 2>&1; then
    echo "Error: Not logged in to Fly.io"
    echo "Run: flyctl auth login"
    exit 1
fi

# Read environment variables from .env
if [ -f .env ]; then
    source .env
else
    echo "Error: .env file not found"
    exit 1
fi

echo "Step 1: Deploying Backend to Fly.io..."
cd backend

# Create app if it doesn't exist
if ! flyctl apps list | grep -q "boom-backend"; then
    flyctl apps create boom-backend --machines
fi

# Set secrets
flyctl secrets set \
    LIVEKIT_URL="$LIVEKIT_URL" \
    LIVEKIT_API_KEY="$LIVEKIT_API_KEY" \
    LIVEKIT_API_SECRET="$LIVEKIT_API_SECRET" \
    FRONTEND_URL="https://boom-prototype.vercel.app" \
    AI_SERVICE_URL="https://boom-ai.fly.dev" \
    --app boom-backend

# Deploy
flyctl deploy --app boom-backend

echo ""
echo "Step 2: Deploying AI Service to Fly.io..."
cd ../ai-service

# Create app if it doesn't exist
if ! flyctl apps list | grep -q "boom-ai"; then
    flyctl apps create boom-ai --machines
fi

# Set secrets
flyctl secrets set \
    LIVEKIT_URL="$LIVEKIT_URL" \
    LIVEKIT_API_KEY="$LIVEKIT_API_KEY" \
    LIVEKIT_API_SECRET="$LIVEKIT_API_SECRET" \
    DEEPGRAM_API_KEY="$DEEPGRAM_API_KEY" \
    BACKEND_WS_URL="wss://boom-backend.fly.dev" \
    --app boom-ai

# Deploy
flyctl deploy --app boom-ai

echo ""
echo "Step 3: Frontend deployment instructions..."
echo "The frontend needs to be deployed to Vercel:"
echo ""
echo "  cd frontend"
echo "  npx vercel --prod"
echo ""
echo "Set these environment variables in Vercel dashboard:"
echo "  VITE_BACKEND_URL=https://boom-backend.fly.dev"
echo "  VITE_LIVEKIT_URL=$LIVEKIT_URL"
echo ""
echo "=== Deployment Complete ==="
echo ""
echo "URLs:"
echo "  Backend:  https://boom-backend.fly.dev"
echo "  AI:       https://boom-ai.fly.dev"
echo "  Frontend: https://boom-prototype.vercel.app (after Vercel deploy)"
