#!/bin/bash
# Development environment validation script for Gunj Operator

set -e

echo "🔍 Gunj Operator Development Environment Validator"
echo "================================================"
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check function
check() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
    else
        echo -e "${RED}✗${NC} $1"
        return 1
    fi
}

# Warning function
warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

echo "1. Checking Docker..."
docker --version > /dev/null 2>&1
check "Docker is installed"

docker ps > /dev/null 2>&1
check "Docker daemon is running"

echo ""
echo "2. Checking Docker Compose..."
docker-compose --version > /dev/null 2>&1
check "Docker Compose is installed"

echo ""
echo "3. Checking required directories..."
for dir in hack hack/db hack/prometheus hack/grafana/provisioning hack/loki hack/tempo hack/otel; do
    if [ -d "$dir" ]; then
        check "Directory $dir exists"
    else
        warn "Directory $dir missing - run 'make setup-dev'"
    fi
done

echo ""
echo "4. Checking configuration files..."
for file in docker-compose.yml Dockerfile.dev Makefile .devcontainer/devcontainer.json; do
    if [ -f "$file" ]; then
        check "File $file exists"
    else
        warn "File $file missing"
    fi
done

echo ""
echo "5. Checking port availability..."
for port in 3000 8080 8081 9090 3001 3100 3200 5432 6379 8000; do
    if ! lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        check "Port $port is available"
    else
        warn "Port $port is in use"
    fi
done

echo ""
echo "6. Checking development container..."
if docker-compose ps dev 2>/dev/null | grep -q "Up"; then
    check "Development container is running"
    
    echo ""
    echo "7. Checking container tools..."
    docker-compose exec -T dev go version > /dev/null 2>&1
    check "Go is available in container"
    
    docker-compose exec -T dev kubectl version --client > /dev/null 2>&1
    check "kubectl is available in container"
    
    docker-compose exec -T dev helm version > /dev/null 2>&1
    check "Helm is available in container"
else
    warn "Development container is not running - run 'make dev-up'"
fi

echo ""
echo "8. Checking database connection..."
if docker-compose ps postgres 2>/dev/null | grep -q "Up"; then
    docker-compose exec -T postgres pg_isready -U gunj > /dev/null 2>&1
    check "PostgreSQL is ready"
else
    warn "PostgreSQL is not running"
fi

echo ""
echo "9. Checking Redis connection..."
if docker-compose ps redis 2>/dev/null | grep -q "Up"; then
    docker-compose exec -T redis redis-cli ping > /dev/null 2>&1
    check "Redis is ready"
else
    warn "Redis is not running"
fi

echo ""
echo "10. Checking VS Code extensions (if applicable)..."
if command -v code > /dev/null 2>&1; then
    if code --list-extensions | grep -q "ms-vscode-remote.remote-containers"; then
        check "VS Code Remote Containers extension is installed"
    else
        warn "VS Code Remote Containers extension not installed"
    fi
else
    warn "VS Code not found in PATH"
fi

echo ""
echo "================================================"
echo "Validation complete!"
echo ""
echo "Next steps:"
echo "  1. Run 'make quickstart' to start the development environment"
echo "  2. Run 'make dev-shell' to enter the development container"
echo "  3. Run 'make dev-info' to see environment information"
echo ""
echo "For more information, see README-DEV.md"
