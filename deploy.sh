#!/bin/bash
# Boom Prototype Deployment Script
# Deploys to DigitalOcean droplet (do-stoic)
#
# Usage: ./deploy.sh [backend|ai|frontend|all]
#
# Environment variables are read from the RUNNING containers on do-stoic,
# not from a local .env file. This prevents config drift.

set -e

HOST="do-stoic"
NETWORK="n8n-docker-caddy_default"
REPO="https://github.com/justin-nevins/boom-prototype.git"

COMPONENT="${1:-all}"

echo "=== Boom Prototype Deployment ==="
echo "Component: $COMPONENT"
echo ""

# Pull env vars from running containers on do-stoic
load_backend_env() {
    eval "$(ssh "$HOST" "docker inspect boom-backend --format '{{json .Config.Env}}'" | python3 -c "
import json, sys
for e in json.load(sys.stdin):
    k, v = e.split('=', 1)
    if any(k.startswith(p) for p in ['LIVEKIT_','SMTP_','JWT_','BOOM_','DEEPGRAM_','ANTHROPIC_']):
        print(f'export {k}=\"{v}\"')
")"
}

load_ai_env() {
    eval "$(ssh "$HOST" "docker inspect boom-ai --format '{{json .Config.Env}}'" | python3 -c "
import json, sys
for e in json.load(sys.stdin):
    k, v = e.split('=', 1)
    if any(k.startswith(p) for p in ['LIVEKIT_','DEEPGRAM_','ANTHROPIC_']):
        print(f'export {k}=\"{v}\"')
")"
}

deploy_backend() {
    echo "=== Deploying Backend ==="
    load_backend_env
    echo "Using LIVEKIT_URL=$LIVEKIT_URL"
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
            -e SMTP_HOST="$SMTP_HOST" \
            -e SMTP_PORT="$SMTP_PORT" \
            -e SMTP_USER="$SMTP_USER" \
            -e SMTP_PASSWORD="$SMTP_PASSWORD" \
            -e SMTP_FROM="$SMTP_FROM" \
            -e JWT_SECRET="$JWT_SECRET" \
            -e BOOM_ADMIN_PASSWORD="$BOOM_ADMIN_PASSWORD" \
            -e BOOM_API_KEY="$BOOM_API_KEY" \
            boom-backend:latest
        rm -rf /tmp/boom-prototype
        echo "Backend deployed successfully"
REMOTE
}

deploy_ai() {
    echo "=== Deploying AI Service ==="
    load_ai_env
    echo "Using LIVEKIT_URL=$LIVEKIT_URL"
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
    load_backend_env
    echo "Using LIVEKIT_URL=$LIVEKIT_URL"
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
