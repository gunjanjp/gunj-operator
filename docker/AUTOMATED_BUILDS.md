# Docker Hub Automated Build Configuration

**Project**: Gunj Operator  
**Last Updated**: June 12, 2025  

## 🔧 Automated Build Setup

This document describes the automated build configuration for Docker Hub repositories.

## 📋 Build Configurations

### 1. gunj-operator

#### Build Rules:
```yaml
- name: "Latest Release"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "Dockerfile"
  docker_tag: "latest"
  build_context: "/"

- name: "Version Tags"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "Dockerfile"
  docker_tag: "{sourceref}"
  build_context: "/"

- name: "Major Version"
  source_type: "Tag"
  source: "/^v([0-9]+)\.[0-9]+\.[0-9]+$/"
  dockerfile: "Dockerfile"
  docker_tag: "v{/^v([0-9]+)\.[0-9]+\.[0-9]+$/}"
  build_context: "/"

- name: "Minor Version"
  source_type: "Tag"
  source: "/^v([0-9]+\.[0-9]+)\.[0-9]+$/"
  dockerfile: "Dockerfile"
  docker_tag: "v{/^v([0-9]+\.[0-9]+)\.[0-9]+$/}"
  build_context: "/"

- name: "Development"
  source_type: "Branch"
  source: "develop"
  dockerfile: "Dockerfile.dev"
  docker_tag: "develop"
  build_context: "/"

- name: "Nightly"
  source_type: "Branch"
  source: "main"
  dockerfile: "Dockerfile"
  docker_tag: "nightly"
  build_context: "/"
  cron: "0 2 * * *"
```

#### Build Environment Variables:
```yaml
BUILD_DATE: "$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
GIT_COMMIT: "${SOURCE_COMMIT}"
VERSION: "${DOCKER_TAG}"
GOPROXY: "https://proxy.golang.org,direct"
CGO_ENABLED: "0"
```

### 2. gunj-operator-api

#### Build Rules:
```yaml
- name: "Latest Release"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "build/docker/api/Dockerfile"
  docker_tag: "latest"
  build_context: "/"

- name: "Version Tags"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "build/docker/api/Dockerfile"
  docker_tag: "{sourceref}"
  build_context: "/"

- name: "Development"
  source_type: "Branch"
  source: "develop"
  dockerfile: "build/docker/api/Dockerfile.dev"
  docker_tag: "develop"
  build_context: "/"
```

### 3. gunj-operator-ui

#### Build Rules:
```yaml
- name: "Latest Release"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "build/docker/ui/Dockerfile"
  docker_tag: "latest"
  build_context: "/"
  build_args:
    NODE_ENV: "production"
    REACT_APP_VERSION: "${DOCKER_TAG}"

- name: "Version Tags"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "build/docker/ui/Dockerfile"
  docker_tag: "{sourceref}"
  build_context: "/"

- name: "Development"
  source_type: "Branch"
  source: "develop"
  dockerfile: "build/docker/ui/Dockerfile.dev"
  docker_tag: "develop"
  build_context: "/"
  build_args:
    NODE_ENV: "development"
```

### 4. gunj-operator-cli

#### Build Rules:
```yaml
- name: "Latest Release"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "build/docker/cli/Dockerfile"
  docker_tag: "latest"
  build_context: "/"

- name: "Version Tags"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "build/docker/cli/Dockerfile"
  docker_tag: "{sourceref}"
  build_context: "/"
```

### 5. gunj-operator-bundle

#### Build Rules:
```yaml
- name: "Latest Release"
  source_type: "Tag"
  source: "/^v[0-9]+\.[0-9]+\.[0-9]+$/"
  dockerfile: "build/docker/bundle/Dockerfile"
  docker_tag: "latest"
  build_context: "/"

- name: "Demo"
  source_type: "Branch"
  source: "main"
  dockerfile: "build/docker/bundle/Dockerfile.demo"
  docker_tag: "demo"
  build_context: "/"
  build_args:
    INCLUDE_DEMO_DATA: "true"
```

## 🏗️ Build Hooks

### Pre-Build Hook (`hooks/pre_build`):
```bash
#!/bin/bash
# Docker Hub pre-build hook

# Set build arguments
export BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
export GIT_COMMIT=${SOURCE_COMMIT:-$(git rev-parse HEAD)}
export VERSION=${DOCKER_TAG}

# Log build information
echo "Building Gunj Operator"
echo "Version: ${VERSION}"
echo "Commit: ${GIT_COMMIT}"
echo "Date: ${BUILD_DATE}"

# Update build args
echo "BUILD_DATE=${BUILD_DATE}" >> $DOCKERFILE_PATH.args
echo "GIT_COMMIT=${GIT_COMMIT}" >> $DOCKERFILE_PATH.args
echo "VERSION=${VERSION}" >> $DOCKERFILE_PATH.args
```

### Build Hook (`hooks/build`):
```bash
#!/bin/bash
# Docker Hub build hook

# Enable buildkit
export DOCKER_BUILDKIT=1

# Build for multiple architectures
docker buildx create --use --name multiarch-builder || true

# Build and push multi-arch image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg BUILD_DATE=${BUILD_DATE} \
  --build-arg GIT_COMMIT=${GIT_COMMIT} \
  --build-arg VERSION=${VERSION} \
  --tag ${IMAGE_NAME} \
  --push \
  -f ${DOCKERFILE_PATH} \
  .
```

### Post-Build Hook (`hooks/post_push`):
```bash
#!/bin/bash
# Docker Hub post-push hook

# Generate and push SBOM
syft ${IMAGE_NAME} -o spdx-json > sbom.spdx.json
cosign attach sbom --sbom sbom.spdx.json ${IMAGE_NAME}

# Sign the image
cosign sign --key env://COSIGN_PRIVATE_KEY ${IMAGE_NAME}

# Update Docker Hub description
./scripts/update-dockerhub-description.sh ${DOCKER_REPO}
```

## 🔄 GitHub Actions Integration

### Trigger Docker Hub Builds:
```yaml
name: Trigger Docker Hub Build

on:
  push:
    tags:
      - 'v*'
    branches:
      - main
      - develop

jobs:
  trigger:
    runs-on: ubuntu-latest
    steps:
    - name: Trigger Build
      run: |
        curl -X POST \
          -H "Content-Type: application/json" \
          -d '{"source_type": "Tag", "source_name": "${{ github.ref_name }}"}' \
          https://hub.docker.com/api/build/v1/source/${{ secrets.DOCKERHUB_BUILD_TRIGGER }}
```

## 📊 Build Optimization

### Multi-Stage Build Caching:
```dockerfile
# syntax=docker/dockerfile:1.4

# Cache mount for Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Cache mount for Go build cache
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -o operator
```

### Layer Caching Strategy:
1. Base image layers (rarely change)
2. Dependency layers (change occasionally)
3. Source code layers (change frequently)
4. Binary layers (change with source)

## 🔒 Security Configuration

### Build-time Security:
```yaml
# Scan during build
- name: "Security Scan"
  run: |
    trivy image --exit-code 1 --severity HIGH,CRITICAL ${IMAGE_NAME}

# Sign images
- name: "Sign Image"
  run: |
    cosign sign --key $COSIGN_KEY ${IMAGE_NAME}
```

### Runtime Security:
- Non-root user: 65532
- Read-only root filesystem
- No capabilities
- Distroless base image

## 📝 Build Notifications

### Webhook Configuration:
```json
{
  "name": "build-notification",
  "active": true,
  "events": ["build:success", "build:failure"],
  "config": {
    "url": "https://api.gunj-operator.io/webhooks/docker-build",
    "content_type": "json",
    "secret": "${WEBHOOK_SECRET}"
  }
}
```

### Notification Channels:
- Slack: #gunj-operator-builds
- Email: builds@gunj-operator.io
- GitHub Issues (on failure)

## 🚀 Advanced Features

### 1. Matrix Builds:
```yaml
strategy:
  matrix:
    go-version: ['1.21', '1.22']
    platform: ['linux/amd64', 'linux/arm64']
```

### 2. Conditional Builds:
```yaml
# Only build if source changed
- name: "Check Changes"
  run: |
    if git diff --name-only HEAD^ | grep -q "^cmd/operator/"; then
      echo "SHOULD_BUILD=true" >> $GITHUB_ENV
    fi
```

### 3. Build Cache Registry:
```yaml
cache-from: |
  type=registry,ref=gunjanjp/gunj-operator:buildcache
cache-to: |
  type=registry,ref=gunjanjp/gunj-operator:buildcache,mode=max
```

## 📚 Troubleshooting

### Common Issues:

1. **Build Timeout**
   - Increase timeout in Docker Hub settings
   - Optimize Dockerfile for faster builds
   - Use build cache effectively

2. **Multi-arch Failures**
   - Ensure QEMU is properly set up
   - Check architecture-specific dependencies
   - Verify cross-compilation settings

3. **Hook Failures**
   - Check hook script permissions (executable)
   - Verify environment variables
   - Review Docker Hub logs

## 📞 Support

For build issues:
- Docker Hub Forums: https://forums.docker.com
- GitHub Issues: https://github.com/gunjanjp/gunj-operator/issues
- Email: gunjanjp@gmail.com
