# Artifact Storage Documentation

## Overview

The Gunj Operator project implements a comprehensive artifact storage strategy across multiple backends to ensure reliable, cost-effective, and secure storage of build artifacts, container images, and packages.

## Storage Backends

### 1. GitHub Actions Artifacts
- **Purpose**: Temporary storage for build artifacts, test results, and coverage reports
- **Retention**: 7-365 days based on artifact type
- **Limits**: 10GB total storage per repository
- **Access**: Token-based authentication

### 2. AWS S3
- **Purpose**: Long-term storage and backup of artifacts
- **Bucket**: `gunj-operator-artifacts`
- **Storage Classes**:
  - Standard: Hot data (0-30 days)
  - Standard-IA: Warm data (30-90 days)
  - Glacier: Cold data (90+ days)
- **Encryption**: AES-256 at rest, TLS in transit

### 3. Container Registries
- **Docker Hub**: Public images at `gunjanjp/*`
- **GitHub Packages**: `ghcr.io/gunjanjp/*`
- **AWS ECR Public**: `public.ecr.aws/gunj-operator/*`

### 4. Package Registries
- **NPM**: JavaScript SDK and UI packages
- **Homebrew**: macOS CLI distribution
- **APT/YUM**: Linux package distribution
- **Chocolatey**: Windows package distribution

## Workflows

### Artifact Management (`artifact-management.yml`)
Handles the storage and retrieval of build artifacts across all backends.

**Features**:
- Multi-backend storage with fallback
- Automatic compression and deduplication
- Metadata tracking and manifest generation
- S3 lifecycle policy management

**Usage**:
```yaml
- uses: ./.github/workflows/artifact-management.yml
  with:
    artifact_type: binary
    retention_days: 30
```

### Artifact Caching (`artifact-caching.yml`)
Optimizes build performance through intelligent caching strategies.

**Features**:
- Multi-level cache hierarchy
- Cache warming for common dependencies
- Hit rate monitoring and optimization
- Automatic cleanup of stale caches

**Usage**:
```yaml
- uses: ./.github/workflows/artifact-caching.yml
  with:
    cache_key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    cache_paths: ~/go/pkg/mod
    cache_type: dependency
```

### Artifact Retention (`artifact-retention.yml`)
Manages lifecycle and cleanup of artifacts based on retention policies.

**Features**:
- Configurable retention policies by artifact type
- Automatic cleanup scheduling
- Dry-run mode for policy testing
- Compliance reporting

**Schedule**: Daily at 2 AM UTC

### Artifact Monitoring (`artifact-monitoring.yml`)
Monitors storage usage, costs, performance, and security.

**Features**:
- Real-time usage metrics
- Cost analysis and optimization
- Performance monitoring
- Security auditing

**Schedule**: Every 6 hours

### Publishing (`publish-artifacts.yml`)
Publishes artifacts to various package managers and registries.

**Triggers**:
- Release publication
- Manual workflow dispatch

## Configuration

### Central Configuration (`storage-config.yml`)
All storage backends and policies are configured in `.github/artifact-storage/storage-config.yml`.

Key sections:
- `storage_backends`: Backend-specific configurations
- `artifact_categories`: Artifact type definitions
- `caching`: Cache strategies and policies
- `security`: Signing and scanning configuration
- `monitoring`: Metrics and alerting
- `cost_optimization`: Cost reduction strategies

### Environment Variables

Required secrets:
- `AWS_ACCESS_KEY_ID`: AWS access key for S3
- `AWS_SECRET_ACCESS_KEY`: AWS secret key
- `DOCKER_USERNAME`: Docker Hub username
- `DOCKER_PASSWORD`: Docker Hub password
- `NPM_TOKEN`: NPM publishing token
- `HOMEBREW_TAP_TOKEN`: GitHub token for Homebrew tap
- `GPG_PRIVATE_KEY`: GPG key for signing
- `COSIGN_KEY`: Cosign key for container signing

## Best Practices

### 1. Artifact Naming
```
<component>-<version>-<platform>-<arch>.<ext>
Example: gunj-operator-v2.0.0-linux-amd64.tar.gz
```

### 2. Retention Guidelines
- Development builds: 7 days
- Feature branches: 14 days
- Main branch: 30 days
- Releases: 365 days
- Test results: 7 days
- Security scans: 90 days

### 3. Cost Optimization
- Enable lifecycle transitions for S3
- Use compression for all artifacts
- Implement deduplication
- Regular cleanup of old artifacts
- Monitor usage trends

### 4. Security
- Enable encryption for all storage
- Use signed artifacts for releases
- Implement access controls
- Enable audit logging
- Regular security scans

## Monitoring

### Metrics
Metrics are pushed to Prometheus and visualized in Grafana:
- Storage usage by backend
- Cache hit rates
- Upload/download performance
- Cost trends
- Security events

### Alerts
Automatic alerts for:
- Storage usage > 80% (warning) or > 90% (critical)
- Monthly cost > $500 (warning) or > $1000 (critical)
- Failed artifact uploads
- Security violations

### Dashboards
- **Storage Overview**: Usage across all backends
- **Performance**: Cache and transfer metrics
- **Cost Analysis**: Spending breakdown and trends
- **Security**: Access patterns and compliance

## Troubleshooting

### Common Issues

1. **Artifact upload fails**
   - Check storage quota
   - Verify credentials
   - Check network connectivity
   - Review size limits

2. **Cache miss rate high**
   - Review cache keys
   - Check restore keys
   - Verify cache paths
   - Consider cache warming

3. **High storage costs**
   - Review retention policies
   - Enable lifecycle transitions
   - Implement compression
   - Clean up old artifacts

4. **Slow artifact operations**
   - Check network bandwidth
   - Optimize artifact size
   - Use parallel uploads
   - Consider regional storage

### Debug Commands

```bash
# Check GitHub Actions storage usage
gh api /repos/gunjanjp/gunj-operator/actions/cache/usage

# List S3 bucket size
aws s3 ls s3://gunj-operator-artifacts --recursive --summarize

# Check Docker Hub rate limits
docker run --rm curlimages/curl -s \
  "https://auth.docker.io/token?service=registry.docker.io&scope=repository:ratelimitpreview/test:pull" | \
  jq -r .token | jwt decode -

# Analyze cache effectiveness
gh run list --workflow=ci.yml --json conclusion,updatedAt | \
  jq '[.[] | select(.conclusion=="success")] | length'
```

## Migration Guide

### From Manual Storage
1. Inventory existing artifacts
2. Create migration workflow
3. Upload to new storage backends
4. Update references
5. Verify accessibility
6. Clean up old storage

### Updating Retention Policies
1. Edit `storage-config.yml`
2. Test with dry-run mode
3. Review impact analysis
4. Apply changes
5. Monitor results

## Support

For issues or questions:
- Create an issue with `artifact-storage` label
- Check workflow logs for errors
- Review monitoring dashboards
- Contact: gunjanjp@gmail.com

## Roadmap

### Q3 2025
- [ ] Add support for Azure Blob Storage
- [ ] Implement artifact versioning
- [ ] Add CDN integration
- [ ] Enhanced security scanning

### Q4 2025
- [ ] Machine learning for cost optimization
- [ ] Predictive storage scaling
- [ ] Advanced deduplication
- [ ] Multi-region replication

---

Last Updated: June 12, 2025
