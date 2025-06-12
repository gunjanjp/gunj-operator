#!/bin/bash
# Setup Artifact Storage for Local Development
# This script configures local artifact storage for the Gunj Operator project
# Version: 2.0

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# Project paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ARTIFACT_DIR="${PROJECT_ROOT}/artifacts"
CONFIG_DIR="${PROJECT_ROOT}/.github"

echo -e "${BLUE}=== Gunj Operator Artifact Storage Setup ===${NC}"
echo ""

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"
    
    local missing=()
    
    # Check required tools
    command -v git >/dev/null 2>&1 || missing+=("git")
    command -v docker >/dev/null 2>&1 || missing+=("docker")
    command -v aws >/dev/null 2>&1 || missing+=("aws-cli")
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo -e "${YELLOW}Optional tools not found: ${missing[*]}${NC}"
        echo "Some features may not be available without these tools."
    fi
    
    echo -e "${GREEN}Prerequisites check complete${NC}"
}

# Setup local artifact directories
setup_local_storage() {
    echo -e "${BLUE}Setting up local artifact storage...${NC}"
    
    # Create directory structure
    mkdir -p "$ARTIFACT_DIR"/{binaries,containers,test-results,coverage,releases}
    mkdir -p "$ARTIFACT_DIR"/.metadata
    mkdir -p "$HOME/.cache/gunj-operator/artifacts"
    
    # Create .gitignore for artifacts
    cat > "$ARTIFACT_DIR/.gitignore" << 'EOF'
# Ignore all artifacts by default
*
!.gitignore
!.metadata/
!README.md

# But track metadata
!.metadata/*
EOF

    # Create README
    cat > "$ARTIFACT_DIR/README.md" << 'EOF'
# Local Artifact Storage

This directory contains build artifacts for local development.

## Structure

- `binaries/` - Compiled binaries for all platforms
- `containers/` - Container images and manifests  
- `test-results/` - Test execution results
- `coverage/` - Code coverage reports
- `releases/` - Release packages
- `.metadata/` - Artifact metadata and manifests

## Usage

Use the artifact manager script to interact with artifacts:

```bash
../scripts/artifact-manager.sh --help
```

## Important

This directory is ignored by git. Do not commit artifacts to the repository.
EOF

    echo -e "${GREEN}Local storage structure created${NC}"
}

# Configure AWS S3 (optional)
configure_s3() {
    echo -e "${BLUE}Configure S3 artifact storage? (y/N)${NC}"
    read -r response
    
    if [[ "$response" =~ ^[Yy]$ ]]; then
        echo "Enter S3 bucket name (default: gunj-operator-artifacts):"
        read -r bucket_name
        bucket_name=${bucket_name:-gunj-operator-artifacts}
        
        echo "Enter AWS region (default: us-east-1):"
        read -r aws_region
        aws_region=${aws_region:-us-east-1}
        
        # Create local S3 config
        mkdir -p "$HOME/.config/gunj-operator"
        cat > "$HOME/.config/gunj-operator/s3-config.json" << EOF
{
    "bucket": "$bucket_name",
    "region": "$aws_region",
    "prefix": "artifacts",
    "lifecycle_enabled": true
}
EOF
        
        echo -e "${GREEN}S3 configuration saved${NC}"
        
        # Test S3 access
        if command -v aws >/dev/null 2>&1; then
            echo -e "${BLUE}Testing S3 access...${NC}"
            if aws s3 ls "s3://$bucket_name" >/dev/null 2>&1; then
                echo -e "${GREEN}S3 access confirmed${NC}"
            else
                echo -e "${YELLOW}Could not access S3 bucket. Check your AWS credentials.${NC}"
            fi
        fi
    fi
}

# Setup GitHub Actions secrets template
setup_github_secrets() {
    echo -e "${BLUE}Creating GitHub secrets template...${NC}"
    
    cat > "$PROJECT_ROOT/.github/secrets-template.env" << 'EOF'
# GitHub Actions Secrets Template
# Copy this file to .env and fill in your values
# DO NOT commit .env file!

# Container Registry
DOCKER_USERNAME=your-docker-username
DOCKER_PASSWORD=your-docker-access-token

# AWS S3 (optional)
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key

# Package Publishing
NPM_TOKEN=your-npm-token
HOMEBREW_TAP_TOKEN=github-pat-for-homebrew
CHOCO_API_KEY=chocolatey-api-key

# Signing Keys
COSIGN_PRIVATE_KEY=your-cosign-private-key
GPG_PRIVATE_KEY=your-gpg-private-key
GPG_PASSPHRASE=your-gpg-passphrase

# Notifications
SLACK_WEBHOOK_URL=your-slack-webhook
EOF

    # Add .env to .gitignore
    if ! grep -q "^\.env$" "$PROJECT_ROOT/.gitignore" 2>/dev/null; then
        echo ".env" >> "$PROJECT_ROOT/.gitignore"
    fi
    
    echo -e "${GREEN}Secrets template created at .github/secrets-template.env${NC}"
}

# Setup artifact signing
setup_signing() {
    echo -e "${BLUE}Setup artifact signing? (y/N)${NC}"
    read -r response
    
    if [[ "$response" =~ ^[Yy]$ ]]; then
        # Check for cosign
        if command -v cosign >/dev/null 2>&1; then
            echo -e "${BLUE}Generating cosign key pair...${NC}"
            cosign generate-key-pair
            echo -e "${GREEN}Cosign keys generated. Store cosign.key securely!${NC}"
        else
            echo -e "${YELLOW}Cosign not installed. Install from: https://github.com/sigstore/cosign${NC}"
        fi
        
        # Check for GPG
        if command -v gpg >/dev/null 2>&1; then
            echo -e "${BLUE}Configure GPG signing? (y/N)${NC}"
            read -r gpg_response
            
            if [[ "$gpg_response" =~ ^[Yy]$ ]]; then
                echo "Enter your GPG key ID (or press Enter to generate new):"
                read -r gpg_key_id
                
                if [[ -z "$gpg_key_id" ]]; then
                    echo -e "${BLUE}Generating new GPG key...${NC}"
                    gpg --full-generate-key
                fi
                
                echo -e "${GREEN}GPG configured for signing${NC}"
            fi
        fi
    fi
}

# Create artifact lifecycle policy
create_lifecycle_policy() {
    echo -e "${BLUE}Creating artifact lifecycle policy...${NC}"
    
    cat > "$CONFIG_DIR/artifact-lifecycle.json" << 'EOF'
{
    "rules": [
        {
            "name": "delete-old-dev-artifacts",
            "description": "Remove development artifacts after 7 days",
            "filters": {
                "branch": ["develop", "feature/*"],
                "age_days": 7
            },
            "actions": ["delete"]
        },
        {
            "name": "archive-old-releases",
            "description": "Archive releases older than 90 days",
            "filters": {
                "tag": "v*",
                "age_days": 90
            },
            "actions": ["compress", "move-to-archive"]
        },
        {
            "name": "cleanup-test-results",
            "description": "Remove test results after 30 days",
            "filters": {
                "type": "test-results",
                "age_days": 30
            },
            "actions": ["delete"]
        }
    ],
    "schedules": {
        "cleanup": "0 2 * * *",
        "archive": "0 3 * * 0"
    }
}
EOF
    
    echo -e "${GREEN}Lifecycle policy created${NC}"
}

# Setup development shortcuts
setup_shortcuts() {
    echo -e "${BLUE}Creating development shortcuts...${NC}"
    
    # Create artifacts helper script
    cat > "$PROJECT_ROOT/artifacts.sh" << 'EOF'
#!/bin/bash
# Quick artifact management shortcuts

case "$1" in
    list)
        ./scripts/artifact-manager.sh list -s local
        ;;
    store)
        ./scripts/artifact-manager.sh store -t binary -c all
        ;;
    clean)
        ./scripts/artifact-manager.sh clean --older-than 7d
        ;;
    upload)
        ./scripts/artifact-manager.sh upload -s s3 -v dev
        ;;
    *)
        echo "Usage: $0 {list|store|clean|upload}"
        exit 1
        ;;
esac
EOF
    
    chmod +x "$PROJECT_ROOT/artifacts.sh"
    
    echo -e "${GREEN}Created artifacts.sh helper script${NC}"
}

# Main setup flow
main() {
    echo ""
    check_prerequisites
    echo ""
    
    setup_local_storage
    echo ""
    
    configure_s3
    echo ""
    
    setup_github_secrets
    echo ""
    
    setup_signing
    echo ""
    
    create_lifecycle_policy
    echo ""
    
    setup_shortcuts
    echo ""
    
    echo -e "${GREEN}=== Artifact Storage Setup Complete ===${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Review and configure .github/secrets-template.env"
    echo "2. Test artifact storage: ./artifacts.sh list"
    echo "3. Configure GitHub Actions secrets in your repository"
    echo "4. Read the documentation: docs/artifact-storage.md"
    echo ""
    echo -e "${BLUE}Happy building!${NC}"
}

# Run main
main
