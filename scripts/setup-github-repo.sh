#!/bin/bash
# GitHub Repository Setup Script for Multi-arch Build Support
# This script helps configure the GitHub repository for multi-architecture builds
# Version: 2.0

set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# Repository details
REPO_OWNER="gunjanjp"
REPO_NAME="gunj-operator"
REPO_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}.git"

echo -e "${BLUE}=== GitHub Repository Setup for Multi-arch Builds ===${NC}"
echo -e "Repository: ${REPO_URL}"
echo ""

# Check if git is initialized
if [ ! -d ".git" ]; then
    echo -e "${YELLOW}Initializing git repository...${NC}"
    git init
    git remote add origin ${REPO_URL}
fi

# Create initial commit structure
echo -e "${BLUE}Creating initial repository structure...${NC}"

# Create README
cat > README.md << 'EOF'
# Gunj Operator

[![Build Status](https://github.com/gunjanjp/gunj-operator/workflows/Build%20Matrix/badge.svg)](https://github.com/gunjanjp/gunj-operator/actions)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/gunjanjp/gunj-operator)](https://goreportcard.com/report/github.com/gunjanjp/gunj-operator)
[![Docker Hub](https://img.shields.io/docker/pulls/gunjanjp/gunj-operator)](https://hub.docker.com/r/gunjanjp/gunj-operator)

Enterprise Observability Platform Kubernetes Operator - A cloud-native operator for managing Prometheus, Grafana, Loki, and Tempo.

## 🚀 Features

- **Multi-Architecture Support**: Native support for amd64, arm64, and arm/v7
- **Kubernetes Native**: Built using the operator pattern
- **Complete Observability Stack**: Prometheus, Grafana, Loki, and Tempo
- **GitOps Ready**: Seamless integration with ArgoCD and Flux
- **Production Ready**: HA configurations, backup/restore, and monitoring

## 📋 Supported Architectures

| Component | linux/amd64 | linux/arm64 | linux/arm/v7 | darwin/amd64 | darwin/arm64 | windows/amd64 |
|-----------|-------------|-------------|--------------|--------------|--------------|---------------|
| Operator  | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| API Server| ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| CLI       | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| UI        | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

## 🔧 Quick Start

```bash
# Install using Helm
helm repo add gunj-operator https://gunjanjp.github.io/gunj-operator/charts
helm install gunj-operator gunj-operator/gunj-operator --namespace gunj-system --create-namespace

# Create your first observability platform
kubectl apply -f examples/basic-platform.yaml
```

## 📚 Documentation

- [Installation Guide](docs/installation.md)
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api-reference.md)
- [Contributing Guide](CONTRIBUTING.md)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Contact

- Email: gunjanjp@gmail.com
- Slack: [#gunj-operator](https://kubernetes.slack.com/messages/gunj-operator)
- Issues: [GitHub Issues](https://github.com/gunjanjp/gunj-operator/issues)
EOF

# Create LICENSE file
cat > LICENSE << 'EOF'
MIT License

Copyright (c) 2025 Gunjan

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
EOF

# Create .gitignore
cat > .gitignore << 'EOF'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
dist/
bin/

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out
coverage/
test-results/

# Go workspace file
go.work

# Dependency directories
vendor/

# IDE specific files
.idea/
.vscode/
*.swp
*.swo
*~

# OS specific files
.DS_Store
Thumbs.db

# Build artifacts
/gunj-operator
/gunj-api-server
/gunj-cli

# Docker
.dockerignore
*.tar

# Kubernetes
kubeconfig
*.kubeconfig

# UI
ui/node_modules/
ui/dist/
ui/build/
ui/.env.local
ui/.env.*.local
ui/npm-debug.log*
ui/yarn-debug.log*
ui/yarn-error.log*

# Temporary files
*.tmp
*.bak
.cache/

# Secrets
*.key
*.pem
*.crt
secrets/
EOF

# Create GitHub Actions secrets documentation
cat > .github/SECRETS.md << 'EOF'
# GitHub Actions Secrets Configuration

The following secrets need to be configured in your GitHub repository for the multi-architecture build matrix to work properly:

## Required Secrets

### Container Registry

- **DOCKER_PASSWORD**: Docker Hub password or access token
  - Required for pushing images to Docker Hub
  - Create at: https://hub.docker.com/settings/security

### Code Signing (Optional but Recommended)

- **COSIGN_PRIVATE_KEY**: Cosign private key for signing container images
  - Generate with: `cosign generate-key-pair`
  - Store the private key as this secret

### Notifications (Optional)

- **SLACK_WEBHOOK_URL**: Slack webhook for build notifications
  - Create at: https://api.slack.com/messaging/webhooks

## Setting Secrets

1. Go to Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Add each secret with the appropriate name and value

## Build Matrix Configuration

The build matrix is configured to support multiple architectures:

- **linux/amd64**: Standard x86_64 Linux
- **linux/arm64**: 64-bit ARM (AWS Graviton, Apple Silicon under Docker)
- **linux/arm/v7**: 32-bit ARM (Raspberry Pi)

Each component is built for all supported architectures automatically.
EOF

# Create PR template for multi-arch considerations
mkdir -p .github/PULL_REQUEST_TEMPLATE
cat > .github/pull_request_template.md << 'EOF'
## Description

Brief description of the changes in this PR.

## Type of Change

- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Refactoring

## Multi-Architecture Considerations

- [ ] Changes tested on linux/amd64
- [ ] Changes tested on linux/arm64 (or confirmed architecture-agnostic)
- [ ] Changes tested on linux/arm/v7 (or confirmed architecture-agnostic)
- [ ] No architecture-specific code added
- [ ] Architecture-specific code properly conditionally compiled

## Testing

- [ ] Unit tests pass on all architectures
- [ ] Integration tests pass
- [ ] E2E tests pass (if applicable)
- [ ] Manual testing completed

## Checklist

- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] No new warnings generated
- [ ] Dependent changes merged

## Build Matrix Status

The automated build matrix will test this PR on:
- linux/amd64
- linux/arm64  
- linux/arm/v7

Please wait for all architecture builds to complete before merging.
EOF

# Create build badge configuration
cat > .github/build-badges.md << 'EOF'
# Build Status Badges

Add these badges to your README to show multi-architecture build status:

## Overall Build Status
```markdown
[![Build Matrix](https://github.com/gunjanjp/gunj-operator/workflows/Build%20Matrix/badge.svg)](https://github.com/gunjanjp/gunj-operator/actions/workflows/build-matrix.yml)
```

## Architecture-Specific Status
```markdown
[![AMD64 Build](https://img.shields.io/badge/amd64-passing-green)](https://github.com/gunjanjp/gunj-operator/actions?query=workflow%3A%22Build+Matrix%22+GOARCH%3Damd64)
[![ARM64 Build](https://img.shields.io/badge/arm64-passing-green)](https://github.com/gunjanjp/gunj-operator/actions?query=workflow%3A%22Build+Matrix%22+GOARCH%3Darm64)
[![ARMv7 Build](https://img.shields.io/badge/armv7-passing-green)](https://github.com/gunjanjp/gunj-operator/actions?query=workflow%3A%22Build+Matrix%22+GOARCH%3Darm)
```

## Container Image Badges
```markdown
[![Docker Hub](https://img.shields.io/docker/pulls/gunjanjp/gunj-operator)](https://hub.docker.com/r/gunjanjp/gunj-operator)
[![Image Size](https://img.shields.io/docker/image-size/gunjanjp/gunj-operator/latest)](https://hub.docker.com/r/gunjanjp/gunj-operator)
```
EOF

# Create initial commit
echo -e "${BLUE}Creating initial commit...${NC}"
git add .
git commit -m "feat: initial multi-architecture build matrix setup

- Add comprehensive GitHub Actions workflow for multi-arch builds
- Support for linux/amd64, linux/arm64, and linux/arm/v7
- Automated testing across all architectures
- Container image building and manifest creation
- Build configuration for all components
- Local development scripts for multi-arch builds

This commit establishes the foundation for building the Gunj Operator
across multiple architectures, ensuring compatibility with various
deployment environments from cloud to edge computing."

echo -e "${GREEN}✅ Repository setup complete!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Create the repository on GitHub: https://github.com/new"
echo "   - Name: gunj-operator"
echo "   - Description: Enterprise Observability Platform Kubernetes Operator"
echo "   - Public repository"
echo "   - Do NOT initialize with README (we have one)"
echo ""
echo "2. Push the initial commit:"
echo "   git push -u origin main"
echo ""
echo "3. Configure secrets in GitHub:"
echo "   - Go to Settings → Secrets and variables → Actions"
echo "   - Add DOCKER_PASSWORD secret"
echo "   - See .github/SECRETS.md for details"
echo ""
echo "4. Enable GitHub Actions:"
echo "   - Actions should be enabled by default"
echo "   - First workflow run will start automatically on push"
echo ""
echo "5. Set up Docker Hub repositories:"
echo "   - Create repositories on Docker Hub:"
echo "     - gunjanjp/gunj-operator"
echo "     - gunjanjp/gunj-api"
echo "     - gunjanjp/gunj-cli"
echo "     - gunjanjp/gunj-ui"
