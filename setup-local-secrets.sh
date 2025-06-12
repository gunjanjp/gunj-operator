#!/bin/bash
# setup-local-secrets.sh - Set up secrets for local development
# This script helps developers set up their local environment with necessary secrets

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Header
echo "================================================"
echo "Gunj Operator - Local Secret Setup"
echo "================================================"
echo ""

# Check if .env exists
if [ -f .env ]; then
    print_warn ".env file already exists. Backing up to .env.backup"
    cp .env .env.backup
fi

# Create .env.example if it doesn't exist
if [ ! -f .env.example ]; then
    cat > .env.example << 'EOF'
# Docker Hub Credentials
DOCKER_USERNAME=your-docker-username
DOCKER_PASSWORD=your-docker-access-token

# GitHub Token (optional for local development)
GITHUB_TOKEN=your-github-token

# Slack Webhook (optional)
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# GPG Signing (optional)
GPG_PRIVATE_KEY_PATH=/path/to/your/gpg/key
GPG_PASSPHRASE=your-gpg-passphrase

# Security Scanning (optional)
SNYK_TOKEN=your-snyk-token
SONAR_TOKEN=your-sonar-token

# Local Development
LOCAL_REGISTRY=localhost:5000
SKIP_PUSH=true
EOF
    print_info "Created .env.example file"
fi

# Interactive setup
echo ""
echo "Let's set up your local secrets. Press Enter to skip optional values."
echo ""

# Function to read secret input
read_secret() {
    local prompt="$1"
    local var_name="$2"
    local required="$3"
    local secret_value=""
    
    if [ "$required" = "true" ]; then
        while [ -z "$secret_value" ]; do
            read -s -p "$prompt (required): " secret_value
            echo ""
            if [ -z "$secret_value" ]; then
                print_error "This value is required!"
            fi
        done
    else
        read -s -p "$prompt (optional): " secret_value
        echo ""
    fi
    
    if [ -n "$secret_value" ]; then
        echo "$var_name=$secret_value" >> .env
    fi
}

# Start creating .env file
echo "# Gunj Operator Local Development Secrets" > .env
echo "# Generated on $(date)" >> .env
echo "" >> .env

# Docker Hub setup
print_info "Setting up Docker Hub credentials..."
echo "Visit https://hub.docker.com/settings/security to create an access token"
echo ""

read -p "Docker Hub username: " docker_username
if [ -n "$docker_username" ]; then
    echo "DOCKER_USERNAME=$docker_username" >> .env
    read_secret "Docker Hub access token" "DOCKER_PASSWORD" "true"
else
    print_warn "Skipping Docker Hub setup - you won't be able to push images"
    echo "SKIP_PUSH=true" >> .env
fi

echo "" >> .env

# GitHub Token (optional)
print_info "Setting up GitHub token (optional)..."
echo "This is only needed for GitHub API operations in local development"
read_secret "GitHub personal access token" "GITHUB_TOKEN" "false"

echo "" >> .env

# Slack Webhook (optional)
print_info "Setting up Slack notifications (optional)..."
read_secret "Slack webhook URL" "SLACK_WEBHOOK_URL" "false"

echo "" >> .env

# Local development settings
print_info "Adding local development settings..."
cat >> .env << 'EOF'

# Local Development Settings
LOCAL_REGISTRY=localhost:5000
K8S_CONTEXT=kind-gunj-operator
LOG_LEVEL=debug
ENABLE_WEBHOOKS=false
SKIP_TLS_VERIFY=true
EOF

# Create .gitignore if it doesn't exist
if [ ! -f .gitignore ] || ! grep -q "^\.env$" .gitignore; then
    echo "" >> .gitignore
    echo "# Secret files" >> .gitignore
    echo ".env" >> .gitignore
    echo ".env.local" >> .gitignore
    echo ".env.*.local" >> .gitignore
    echo "*.key" >> .gitignore
    echo "*.pem" >> .gitignore
    echo "*.p12" >> .gitignore
    print_info "Updated .gitignore to exclude secret files"
fi

# Verify setup
echo ""
print_info "Verifying setup..."

if [ -f .env ]; then
    # Source the .env file
    set -a
    source .env
    set +a
    
    # Check required variables
    if [ -n "${DOCKER_USERNAME:-}" ] && [ -n "${DOCKER_PASSWORD:-}" ]; then
        print_info "✅ Docker Hub credentials configured"
    else
        if [ "${SKIP_PUSH:-false}" != "true" ]; then
            print_warn "⚠️  Docker Hub credentials not configured"
        fi
    fi
    
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        print_info "✅ GitHub token configured"
    else
        print_info "ℹ️  GitHub token not configured (optional)"
    fi
    
    if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
        print_info "✅ Slack webhook configured"
    else
        print_info "ℹ️  Slack webhook not configured (optional)"
    fi
fi

# Create helper scripts
print_info "Creating helper scripts..."

# Create load-secrets.sh
cat > load-secrets.sh << 'EOF'
#!/bin/bash
# Source this file to load secrets into your shell session
# Usage: source ./load-secrets.sh

if [ -f .env ]; then
    set -a
    source .env
    set +a
    echo "✅ Secrets loaded from .env"
    echo "Available secrets:"
    env | grep -E '^(DOCKER_|GITHUB_|SLACK_|SNYK_|SONAR_|GPG_|LOCAL_)' | cut -d= -f1 | sort
else
    echo "❌ No .env file found. Run ./setup-local-secrets.sh first"
fi
EOF
chmod +x load-secrets.sh

# Create test-docker-login.sh
cat > test-docker-login.sh << 'EOF'
#!/bin/bash
# Test Docker Hub login with configured credentials

source ./load-secrets.sh

if [ -z "$DOCKER_USERNAME" ] || [ -z "$DOCKER_PASSWORD" ]; then
    echo "❌ Docker credentials not configured"
    exit 1
fi

echo "Testing Docker Hub login..."
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

if [ $? -eq 0 ]; then
    echo "✅ Docker Hub login successful"
    docker logout
else
    echo "❌ Docker Hub login failed"
    exit 1
fi
EOF
chmod +x test-docker-login.sh

# Summary
echo ""
echo "================================================"
echo "Setup Complete!"
echo "================================================"
echo ""
echo "Files created:"
echo "  - .env (your secret configuration)"
echo "  - .env.example (template for others)"
echo "  - load-secrets.sh (load secrets into shell)"
echo "  - test-docker-login.sh (test Docker login)"
echo ""
echo "Next steps:"
echo "  1. Review .env file and add any missing values"
echo "  2. Run: source ./load-secrets.sh"
echo "  3. Test Docker login: ./test-docker-login.sh"
echo "  4. Never commit .env to version control!"
echo ""
print_info "Happy developing! 🚀"