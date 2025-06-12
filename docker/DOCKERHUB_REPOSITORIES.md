# Docker Hub Repository Structure

**Project**: Gunj Operator  
**Docker Hub Organization**: gunjanjp  
**Last Updated**: June 12, 2025  

## 📦 Repository Overview

The Gunj Operator project uses multiple Docker Hub repositories to distribute container images for different components. Each repository follows semantic versioning and provides multi-architecture support.

## 🏗️ Repository Structure

### 1. Main Operator Repository
**Repository**: `docker.io/gunjanjp/gunj-operator`  
**Description**: Core Kubernetes operator for managing observability platforms  
**Architectures**: linux/amd64, linux/arm64  

#### Tags:
- `latest` - Latest stable release
- `v2.0.0` - Specific version
- `v2.0` - Minor version (receives patch updates)
- `v2` - Major version (receives minor and patch updates)
- `develop` - Development branch builds
- `nightly` - Automated nightly builds

### 2. API Server Repository
**Repository**: `docker.io/gunjanjp/gunj-operator-api`  
**Description**: RESTful and GraphQL API server for the Gunj Operator  
**Architectures**: linux/amd64, linux/arm64  

#### Tags:
- `latest` - Latest stable release
- `v2.0.0` - Specific version
- `v2.0` - Minor version
- `v2` - Major version
- `develop` - Development builds

### 3. UI Repository
**Repository**: `docker.io/gunjanjp/gunj-operator-ui`  
**Description**: React-based web UI for managing observability platforms  
**Architectures**: linux/amd64, linux/arm64  

#### Tags:
- `latest` - Latest stable release
- `v2.0.0` - Specific version
- `v2.0` - Minor version
- `v2` - Major version
- `develop` - Development builds

### 4. CLI Repository
**Repository**: `docker.io/gunjanjp/gunj-operator-cli`  
**Description**: Command-line interface for Gunj Operator management  
**Architectures**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64  

#### Tags:
- `latest` - Latest stable release
- `v2.0.0` - Specific version
- `v2.0` - Minor version
- `v2` - Major version

### 5. All-in-One Repository
**Repository**: `docker.io/gunjanjp/gunj-operator-bundle`  
**Description**: Combined image with operator, API server, and UI for easy deployment  
**Architectures**: linux/amd64, linux/arm64  

#### Tags:
- `latest` - Latest stable release
- `v2.0.0` - Specific version
- `demo` - Demo configuration with sample data

## 📋 Repository Descriptions

### gunj-operator
```
Enterprise-grade Kubernetes operator for deploying and managing observability platforms. 
Automates deployment of Prometheus, Grafana, Loki, and Tempo with built-in best practices.

Features:
• Declarative configuration via CRDs
• Automated upgrades and rollbacks
• Multi-tenancy support
• GitOps integration
• Cost optimization
• Self-healing capabilities

Documentation: https://github.com/gunjanjp/gunj-operator
License: MIT
```

### gunj-operator-api
```
RESTful and GraphQL API server for the Gunj Operator.
Provides programmatic access to manage observability platforms.

Features:
• RESTful API with OpenAPI 3.0 spec
• GraphQL API with subscriptions
• JWT authentication
• RBAC authorization
• Rate limiting
• Webhook support

Documentation: https://github.com/gunjanjp/gunj-operator/tree/main/api
License: MIT
```

### gunj-operator-ui
```
Modern React-based web interface for the Gunj Operator.
Intuitive UI for managing enterprise observability platforms.

Features:
• Real-time platform status
• Drag-and-drop dashboard builder
• Resource usage visualization
• Alert management
• Dark mode support
• Mobile responsive

Documentation: https://github.com/gunjanjp/gunj-operator/tree/main/ui
License: MIT
```

### gunj-operator-cli
```
Command-line interface for the Gunj Operator.
Manage observability platforms from your terminal.

Features:
• Platform management commands
• Configuration validation
• Backup and restore
• Resource optimization
• Kubectl plugin support
• Shell completion

Documentation: https://github.com/gunjanjp/gunj-operator/tree/main/cli
License: MIT
```

### gunj-operator-bundle
```
All-in-one container image for quick Gunj Operator deployments.
Perfect for demos, development, and small-scale deployments.

Includes:
• Gunj Operator
• API Server
• Web UI
• Pre-configured settings
• Demo data (optional)

Documentation: https://github.com/gunjanjp/gunj-operator/tree/main/bundle
License: MIT
```

## 🔒 Security Scanning

All images undergo security scanning before publication:
- Trivy vulnerability scanning
- Snyk container scanning
- Docker Scout analysis
- SBOM generation with Syft
- Image signing with Cosign

## 📊 Image Sizes

Target sizes for production images:
- `gunj-operator`: < 100MB
- `gunj-operator-api`: < 150MB
- `gunj-operator-ui`: < 50MB (nginx)
- `gunj-operator-cli`: < 30MB
- `gunj-operator-bundle`: < 300MB

## 🏷️ Labeling Standards

All images include standard OCI labels:
```dockerfile
LABEL org.opencontainers.image.title="Gunj Operator"
LABEL org.opencontainers.image.description="Enterprise Observability Platform Operator"
LABEL org.opencontainers.image.url="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.source="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.vendor="gunjanjp@gmail.com"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
```

## 🔄 Automated Builds

Automated builds are triggered by:
- Push to main branch → latest tag
- Push to develop branch → develop tag
- Git tag push → version tags
- Scheduled nightly → nightly tag
- Pull requests → temporary tags for testing

Build matrix includes:
- Multiple architectures (amd64, arm64)
- Multiple Go versions (for compatibility testing)
- Multiple base images (distroless, alpine, scratch)

## 📚 Usage Examples

### Pull the latest operator image:
```bash
docker pull gunjanjp/gunj-operator:latest
```

### Run the operator locally:
```bash
docker run -d \
  --name gunj-operator \
  -v ~/.kube/config:/home/nonroot/.kube/config:ro \
  gunjanjp/gunj-operator:latest
```

### Run the all-in-one bundle:
```bash
docker run -d \
  --name gunj-bundle \
  -p 8080:8080 \
  -p 8090:8090 \
  -p 3000:3000 \
  gunjanjp/gunj-operator-bundle:demo
```

### Use specific version:
```bash
docker pull gunjanjp/gunj-operator:v2.0.0
```

## 🚀 CI/CD Integration

GitHub Actions workflows automatically:
1. Build images on PR/push
2. Run security scans
3. Push to Docker Hub
4. Update Docker Hub descriptions
5. Generate and sign SBOMs
6. Create vulnerability reports

## 📞 Support

For issues or questions:
- GitHub Issues: https://github.com/gunjanjp/gunj-operator/issues
- Email: gunjanjp@gmail.com
- Docker Hub: https://hub.docker.com/u/gunjanjp
