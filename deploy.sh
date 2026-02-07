#!/bin/bash
# Boom Prototype Deployment Script
# Deploys backend and ai-service to DigitalOcean droplet (do-stoic)
# Frontend is served via Caddy on the same droplet

set -e

HOST="do-stoic"
NETWORK="n8n-docker-caddy_default"
REPO="https://github.com/Cato-Pine/boom-prototype.git"

echo "=== Boom Prototype Deployment ==="
echo ""

# Read environment variables from .env
if [ -f .env ]; then
    source .env
else
    echo "Error: .env file not found"
    exit 1
fi

COMPONENT="${1:-all}"

deploy_backend() {
    echo "=== Deploying Backend ==="
    ssh "$HOST" bash -s <<REMOTE
        set -e
        cd /tmp
        rm -rf boom-prototype
        git clone "$REPO"
        cd boom-prototype/backend
        docker build -t boom-backend:latest .
        docker stop boom-backend 2>/dev/null || true
        docker rm boom-backend 2>/dev/null || true
        docker run -d --name boom-backend \
            --network "$NETWORK" \
            --restart unless-stopped \
            -e LIVEKIT_API_KEY="$LIVEKIT_API_KEY" \
            -e LIVEKIT_API_SECRET="$LIVEKIT_API_SECRET" \
            -e LIVEKIT_URL="$LIVEKIT_URL" \
            -e FRONTEND_URL=https://meet.nevins.cloud \
            -e AI_SERVICE_URL=http://boom-ai:8081 \
            boom-backend:latest
        rm -rf /tmp/boom-prototype
        echo "Backend deployed successfully"
REMOTE
}

deploy_ai() {
    echo "=== Deploying AI Service ==="
    ssh "$HOST" bash -s <<REMOTE
        set -e
        cd /tmp
        rm -rf boom-prototype
        git clone "$REPO"
        cd boom-prototype/ai-service
        docker build -t boom-ai:latest .
        docker stop boom-ai 2>/dev/null || true
        docker rm boom-ai 2>/dev/null || true
        docker run -d --name boom-ai \
            --network "$NETWORK" \
            --restart unless-stopped \
            -e LIVEKIT_API_KEY="$LIVEKIT_API_KEY" \
            -e LIVEKIT_API_SECRET="$LIVEKIT_API_SECRET" \
            -e LIVEKIT_URL="$LIVEKIT_URL" \
            -e DEEPGRAM_API_KEY="$DEEPGRAM_API_KEY" \
            -e ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
            -e BACKEND_WS_URL=ws://boom-backend:8080 \
            -e BACKEND_API_URL=http://boom-backend:8080 \
            boom-ai:latest
        rm -rf /tmp/boom-prototype
        echo "AI Service deployed successfully"
REMOTE
}

deploy_frontend() {
    echo "=== Deploying Frontend ==="
    ssh "$HOST" bash -s <<REMOTE
        set -e
        cd /tmp
        rm -rf boom-prototype
        git clone "$REPO"
        cd boom-prototype/frontend
        docker build \
            --build-arg VITE_BACKEND_URL=https://meet-api.nevins.cloud \
            --build-arg VITE_LIVEKIT_URL="$LIVEKIT_URL" \
            -t boom-frontend:latest .
        docker stop boom-frontend 2>/dev/null || true
        docker rm boom-frontend 2>/dev/null || true
        docker run -d --name boom-frontend \
            --network "$NETWORK" \
            --restart unless-stopped \
            boom-frontend:latest
        rm -rf /tmp/boom-prototype
        echo "Frontend deployed successfully"
REMOTE
}

case "$COMPONENT" in
    backend)  deploy_backend ;;
    ai)       deploy_ai ;;
    frontend) deploy_frontend ;;
    all)
        deploy_backend
        echo ""
        deploy_ai
        echo ""
        deploy_frontend
        ;;
    *)
        echo "Usage: ./deploy.sh [backend|ai|frontend|all]"
        exit 1
        ;;
esac

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "URLs:"
echo "  Frontend:  https://meet.nevins.cloud"
echo "  Backend:   https://meet-api.nevins.cloud"
echo "  AI:        https://ai.nevins.cloud"
