# Artifact Storage Documentation

## Overview

The Gunj Operator uses a multi-tiered artifact storage system to manage build outputs, test results, and release packages across different stages of the development lifecycle.

## Storage Backends

### 1. GitHub Artifacts
- **Purpose**: Short-term storage for CI/CD workflows
- **Retention**: 7-90 days based on artifact type
- **Use Cases**:
  - Build outputs from PR checks
  - Test results and coverage reports
  - Temporary storage between workflow jobs

### 2. Amazon S3
- **Purpose**: Long-term storage and distribution
- **Retention**: Configurable with lifecycle policies
- **Use Cases**:
  - Release artifacts
  - Historical build archives
  - Large binary distributions

### 3. Container Registries
- **Purpose**: Container image storage and distribution
- **Supported Registries**:
  - Docker Hub (primary)
  - GitHub Container Registry (ghcr.io)
  - AWS ECR Public (optional)
- **Retention**: Keep last 10 versions per tag

### 4. Package Managers
- **Homebrew**: macOS CLI distribution
- **NPM**: UI components package
- **APT/YUM**: Linux distribution packages
- **Chocolatey**: Windows package manager

## Artifact Types

### Binary Artifacts
```
Structure:
artifacts/
├── binaries/
│   ├── operator/
│   │   ├── linux-amd64/
│   │   ├── linux-arm64/
│   │   └── linux-arm-v7/
│   ├── api/
│   ├── cli/
│   └── ui/
```

### Container Images
- Multi-architecture support (amd64, arm64, arm/v7)
- Automatic manifest creation
- Signed with cosign

### Test Results
- JUnit XML format
- JSON reports for custom tools
- HTML coverage reports

## Configuration

### GitHub Actions Secrets

Required secrets for artifact storage:

```yaml
# Docker Hub
DOCKER_USERNAME: Your Docker Hub username
DOCKER_PASSWORD: Docker Hub access token

# AWS S3 (optional)
AWS_ACCESS_KEY_ID: AWS access key
AWS_SECRET_ACCESS_KEY: AWS secret key

# Package Publishing
NPM_TOKEN: NPM publish token
HOMEBREW_TAP_TOKEN: GitHub PAT for Homebrew tap
CHOCO_API_KEY: Chocolatey API key

# Signing
COSIGN_PRIVATE_KEY: Container signing key
GPG_PRIVATE_KEY: Package signing key
GPG_PASSPHRASE: GPG key passphrase
```

### Retention Policies

Default retention periods:

| Artifact Type | GitHub | S3 | Registry |
|--------------|--------|-----|----------|
| Development | 7 days | 7 days | 1 day |
| Branch builds | 30 days | 30 days | 7 days |
| Tagged releases | 365 days | Forever | Forever |
| Test results | 30 days | 30 days | N/A |
| Coverage reports | 90 days | 90 days | N/A |

### Storage Costs Optimization

1. **Lifecycle Policies**:
   - Transition to Glacier after 90 days
   - Delete development builds after 7 days
   - Compress all artifacts with gzip -9

2. **Intelligent Tiering**:
   - Enable S3 Intelligent-Tiering
   - Use CloudFront for distribution

3. **Cleanup Automation**:
   - Daily cleanup of expired artifacts
   - Weekly deep cleanup of S3
   - Monthly cost reports

## Usage

### Local Development

```bash
# Store artifacts locally
./scripts/artifact-manager.sh store -t binary -c operator

# Upload to S3
./scripts/artifact-manager.sh upload -s s3 -v v2.0.0

# Download artifacts
./scripts/artifact-manager.sh download -t binary -c cli -v v2.0.0

# List available artifacts
./scripts/artifact-manager.sh list -s s3

# Clean old artifacts
./scripts/artifact-manager.sh clean --older-than 7d
```

### CI/CD Workflows

```yaml
# Use artifact management workflow
- uses: ./.github/workflows/artifact-management.yml
  with:
    artifact_type: binary
    retention_days: 30
```

### Publishing Releases

```bash
# Trigger publishing workflow
gh workflow run publish-artifacts.yml \
  -f version=v2.0.0 \
  -f channels=dockerhub,homebrew,npm
```

## Artifact Naming Conventions

### Binaries
```
Pattern: {component}-{version}-{os}-{arch}
Example: gunj-operator-v2.0.0-linux-amd64
```

### Containers
```
Pattern: {registry}/{namespace}/{component}:{version}-{arch}
Example: docker.io/gunjanjp/gunj-operator:v2.0.0-arm64
```

### Archives
```
Pattern: gunj-operator-{version}-{platform}.tar.gz
Example: gunj-operator-v2.0.0-darwin-arm64.tar.gz
```

## Security

### Signing
- All container images are signed with cosign
- Binary artifacts include SHA256 checksums
- Release packages are GPG signed

### Access Control
- S3 bucket policies restrict access
- Container registries use scoped tokens
- GitHub artifacts respect repository permissions

### Scanning
- All artifacts scanned with Trivy
- Container images scanned before push
- SBOM generated for all releases

## Monitoring

### Metrics
- Storage usage by type and backend
- Upload/download duration
- Artifact access frequency
- Cost tracking

### Alerts
- Storage quota warnings
- Upload failure notifications
- Expired artifact cleanup reports

## Troubleshooting

### Common Issues

1. **Upload Failures**
   ```bash
   # Check AWS credentials
   aws sts get-caller-identity
   
   # Verify S3 permissions
   aws s3 ls s3://gunj-operator-artifacts/
   ```

2. **GitHub Artifact Limits**
   - Max file size: 2GB
   - Max total size: 10GB per workflow
   - Solution: Use S3 for large artifacts

3. **Registry Push Failures**
   ```bash
   # Re-authenticate
   docker logout
   docker login
   
   # Check rate limits
   docker pull gunjanjp/gunj-operator:latest
   ```

## Best Practices

1. **Version Everything**: Always tag artifacts with versions
2. **Clean Regularly**: Set up automated cleanup
3. **Monitor Costs**: Track storage and bandwidth usage
4. **Backup Critical**: Keep multiple copies of releases
5. **Document Changes**: Update this doc with new artifacts

## Future Enhancements

- [ ] CDN distribution for binaries
- [ ] P2P distribution for large artifacts
- [ ] Blockchain-based verification
- [ ] AI-powered cleanup policies
- [ ] Cost prediction models
