#!/usr/bin/env bash
# Development script for Gunj Operator monorepo

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Setup development environment
setup() {
    info "Setting up development environment..."
    
    # Check prerequisites
    info "Checking prerequisites..."
    
    if ! command_exists go; then
        error "Go is not installed. Please install Go 1.21+"
    fi
    
    if ! command_exists node; then
        error "Node.js is not installed. Please install Node.js 20+"
    fi
    
    if ! command_exists docker; then
        warn "Docker is not installed. You won't be able to build images."
    fi
    
    if ! command_exists kubectl; then
        warn "kubectl is not installed. You won't be able to deploy to Kubernetes."
    fi
    
    # Install Go tools
    info "Installing Go tools..."
    make install-tools
    
    # Install Node dependencies
    info "Installing Node dependencies..."
    npm install
    
    # Setup Git hooks
    info "Setting up Git hooks..."
    if command_exists npx; then
        npx husky install
    fi
    
    # Initialize Go modules
    info "Initializing Go modules..."
    make mod-tidy
    
    info "Development environment setup complete!"
}

# Build all components
build_all() {
    info "Building all components..."
    make build
    make build-ui
    info "Build complete!"
}

# Run tests
test_all() {
    info "Running all tests..."
    make test
    make test-ui
    info "All tests passed!"
}

# Start development servers
dev() {
    info "Starting development servers..."
    npm run dev
}

# Clean build artifacts
clean() {
    info "Cleaning build artifacts..."
    make clean-all
    info "Clean complete!"
}

# Generate code
generate() {
    info "Generating code..."
    make generate
    info "Code generation complete!"
}

# Lint code
lint() {
    info "Linting code..."
    make lint
    npm run lint
    info "Linting complete!"
}

# Show help
show_help() {
    cat << EOF
Gunj Operator Development Script

Usage: $0 <command>

Commands:
    setup       Setup development environment
    build       Build all components
    test        Run all tests
    dev         Start development servers
    clean       Clean build artifacts
    generate    Generate code (CRDs, DeepCopy, etc.)
    lint        Lint all code
    help        Show this help message

Examples:
    $0 setup    # First time setup
    $0 dev      # Start development
    $0 test     # Run tests before committing

EOF
}

# Main script logic
main() {
    cd "$PROJECT_ROOT"
    
    case "${1:-}" in
        setup)
            setup
            ;;
        build)
            build_all
            ;;
        test)
            test_all
            ;;
        dev)
            dev
            ;;
        clean)
            clean
            ;;
        generate)
            generate
            ;;
        lint)
            lint
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            error "Unknown command: ${1:-}. Use '$0 help' for usage."
            ;;
    esac
}

main "$@"
