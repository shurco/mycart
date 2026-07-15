#!/bin/bash

# myCart Development Startup Script
# This script builds and runs the PortOne-integrated myCart in Docker

set -e

echo "=========================================="
echo "  myCart Development Environment"
echo "  with PortOne Payment Gateway"
echo "=========================================="
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running"
    echo "Please start Docker and try again"
    exit 1
fi

echo "✅ Docker is running"
echo ""

# Check if docker-compose is available
if ! command -v docker compose &> /dev/null && ! command -v docker-compose &> /dev/null; then
    echo "❌ Error: docker-compose is not installed"
    echo "Please install docker-compose and try again"
    exit 1
fi

# Use docker compose or docker-compose based on availability
if command -v docker compose &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

echo "Using: $DOCKER_COMPOSE"
echo ""

# Parse command line arguments
BUILD_FLAG="--build"
DETACH_FLAG=""
ACTION="up"

for arg in "$@"; do
    case $arg in
        --no-build)
            BUILD_FLAG=""
            shift
            ;;
        -d|--detach)
            DETACH_FLAG="-d"
            shift
            ;;
        stop)
            ACTION="stop"
            shift
            ;;
        down)
            ACTION="down"
            shift
            ;;
        logs)
            ACTION="logs"
            shift
            ;;
        restart)
            ACTION="restart"
            shift
            ;;
        *)
            # Unknown option
            ;;
    esac
done

# Execute action
case $ACTION in
    up)
        echo "🔨 Building and starting myCart development environment..."
        echo ""
        $DOCKER_COMPOSE -f docker-compose.dev.yml up $BUILD_FLAG $DETACH_FLAG

        if [ -n "$DETACH_FLAG" ]; then
            echo ""
            echo "=========================================="
            echo "✅ myCart is running in the background!"
            echo "=========================================="
            echo ""
            echo "Access points:"
            echo "  • Storefront:    http://localhost:8080"
            echo "  • Admin Panel:   http://localhost:8080/admin"
            echo "  • API:           http://localhost:8080/api"
            echo "  • Health Check:  http://localhost:8080/ping"
            echo ""
            echo "Useful commands:"
            echo "  • View logs:     ./dev-start.sh logs"
            echo "  • Stop:          ./dev-start.sh stop"
            echo "  • Restart:       ./dev-start.sh restart"
            echo "  • Shutdown:      ./dev-start.sh down"
            echo ""
            echo "First time setup:"
            echo "  1. Go to http://localhost:8080"
            echo "  2. Complete installation wizard"
            echo "  3. Login to admin panel"
            echo "  4. Configure PortOne at Settings → Payment → PortOne"
            echo ""
        fi
        ;;
    stop)
        echo "⏸️  Stopping myCart..."
        $DOCKER_COMPOSE -f docker-compose.dev.yml stop
        echo "✅ Stopped"
        ;;
    down)
        echo "🛑 Shutting down myCart..."
        $DOCKER_COMPOSE -f docker-compose.dev.yml down
        echo "✅ Shut down complete"
        ;;
    logs)
        echo "📋 Showing logs (press Ctrl+C to exit)..."
        echo ""
        $DOCKER_COMPOSE -f docker-compose.dev.yml logs -f mycart-dev
        ;;
    restart)
        echo "🔄 Restarting myCart..."
        $DOCKER_COMPOSE -f docker-compose.dev.yml restart
        echo "✅ Restarted"
        ;;
esac
