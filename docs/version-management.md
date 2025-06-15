# Chart Version Management

The Gunj Operator provides comprehensive chart version management capabilities to ensure safe, controlled, and automated version updates for all observability components.

## Features

### 1. Version Registry
- Tracks available versions for all components
- Automatic version discovery from Helm repositories
- Caches version information for performance
- Supports multiple chart repositories

### 2. Version Constraints
- Semantic versioning support
- Flexible constraint syntax (e.g., `>=2.45.0 <3.0.0`)
- Version exclusion lists
- Policy-based validation

### 3. Compatibility Matrix
- Ensures component version compatibility
- Prevents incompatible version combinations
- Customizable compatibility rules
- Tested combination tracking

### 4. Update Notifications
- Real-time update availability notifications
- Categorized by update type (patch, minor, major, security)
- Multiple notification channels (webhook, email, K8s events)
- Configurable notification preferences

### 5. Version Pinning
- Pin components to specific versions
- Time-based expiration
- Audit trail for pins
- Override capabilities for emergencies

### 6. Version History
- Complete version change history
- Rollback capabilities
- Performance metrics for upgrades
- Audit logging

### 7. Automated Testing
- Pre-update validation tests
- Post-update verification
- Integration testing
- Performance baseline comparison

### 8. Smart Updates
- Automatic update scheduling
- Update windows configuration
- Gradual rollout support
- Automatic rollback on failure

## Configuration

### Basic Version Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
spec:
  components:
    prometheus:
      enabled: true
      version: "2.48.0"  # Specific version
    grafana:
      enabled: true
      version: "stable"  # Latest stable version
    loki:
      enabled: true
      version: ">=2.8.0 <3.0.0"  # Version constraint
```

### Advanced Version Management

```yaml
spec:
  versionManagement:
    enabled: true
    notifications:
      enabled: true
      channels:
        - type: webhook
          url: https://hooks.slack.com/...
    updateWindows:
      - start: "02:00"
        end: "06:00"
        days: ["Sunday"]
    testing:
      enabled: true
      preUpdateTests:
        - name: "health-check"
```

## Version Pinning

Pin a component to a specific version:

```bash
kubectl apply -f - <<EOF
apiVersion: observability.io/v1beta1
kind: VersionPin
metadata:
  name: prometheus-pin
spec:
  component: prometheus
  version: "2.48.0"
  reason: "Stability requirement for Q2"
  expiresAt: "2025-07-01T00:00:00Z"
EOF
```

## Version Overrides

Override version constraints for specific scenarios:

```bash
kubectl apply -f - <<EOF
apiVersion: observability.io/v1beta1
kind: VersionOverride
metadata:
  name: grafana-upgrade-override
spec:
  component: grafana
  toVersion: "10.2.0"
  reason: "Early adoption of new features"
  force: false
  approvedBy: "platform-admin"
EOF
```

## Update Policies

### Automatic Updates

Configure automatic updates for non-breaking changes:

```yaml
versionPolicy:
  autoUpdate: true
  updateType: "minor"  # Auto-update minor and patch versions
  constraints:
    - ">=2.45.0"
    - "<3.0.0"
```

### Manual Approval

Require approval for critical components:

```yaml
versionPolicy:
  autoUpdate: false
  requireApproval: true
  approvers:
    - "platform-team"
```

## Compatibility Rules

Define custom compatibility rules:

```yaml
compatibility:
  customRules:
    - component: "prometheus"
      version: ">=2.45.0"
      requires:
        grafana: ">=9.0.0"
        loki: ">=2.8.0"
```

## CLI Usage

### Check for Updates

```bash
# Check all components for updates
kubectl gunj version check

# Check specific component
kubectl gunj version check prometheus

# Show available versions
kubectl gunj version list prometheus
```

### Manage Versions

```bash
# Pin a version
kubectl gunj version pin prometheus 2.48.0 --reason "Stability"

# Unpin a version
kubectl gunj version unpin prometheus

# Show version history
kubectl gunj version history prometheus

# Rollback to previous version
kubectl gunj version rollback prometheus
```

### Test Versions

```bash
# Run compatibility test
kubectl gunj version test --components prometheus=2.49.0,grafana=10.3.0

# Run integration test
kubectl gunj version test-integration
```

## Best Practices

1. **Use Version Constraints**: Instead of specific versions, use constraints to allow automatic patch updates
   ```yaml
   version: "~2.48.0"  # Allows 2.48.x updates
   ```

2. **Test Before Production**: Always test version combinations in staging
   ```yaml
   testing:
     enabled: true
     environments: ["staging", "qa"]
   ```

3. **Configure Update Windows**: Schedule updates during low-traffic periods
   ```yaml
   updateWindows:
     - start: "02:00"
       end: "06:00"
       timezone: "America/New_York"
   ```

4. **Monitor Version Changes**: Enable comprehensive logging and notifications
   ```yaml
   history:
     enabled: true
     retention: "90d"
   ```

5. **Plan Major Upgrades**: Use version overrides for controlled major version upgrades
   ```yaml
   # Test in staging first
   # Create override with approval
   # Monitor closely after upgrade
   ```

## Troubleshooting

### Version Conflicts

If you encounter version compatibility issues:

1. Check the compatibility matrix:
   ```bash
   kubectl gunj version compatibility --show
   ```

2. Find compatible versions:
   ```bash
   kubectl gunj version suggest --component prometheus --with grafana=10.2.0
   ```

3. Use override if necessary:
   ```bash
   kubectl gunj version override prometheus 2.49.0 --force --reason "Critical fix"
   ```

### Update Failures

If an update fails:

1. Check the version history:
   ```bash
   kubectl gunj version history --failed
   ```

2. Review logs:
   ```bash
   kubectl logs -n gunj-system deployment/gunj-operator | grep version
   ```

3. Rollback if needed:
   ```bash
   kubectl gunj version rollback prometheus --auto
   ```

## API Reference

### Version Management API

```bash
# REST API endpoints
GET    /api/v1/versions/{component}
GET    /api/v1/versions/{component}/available
POST   /api/v1/versions/{component}/pin
DELETE /api/v1/versions/{component}/pin
GET    /api/v1/versions/compatibility
POST   /api/v1/versions/test
GET    /api/v1/versions/history
```

### GraphQL Queries

```graphql
query GetVersionInfo {
  component(name: "prometheus") {
    currentVersion
    availableVersions
    latestVersion
    updateAvailable
    compatibility {
      compatible
      issues
    }
  }
}

mutation PinVersion {
  pinVersion(input: {
    component: "prometheus"
    version: "2.48.0"
    reason: "Stability requirement"
  }) {
    success
    message
  }
}
```

## Security Considerations

1. **Version Verification**: All chart versions are verified against signatures
2. **CVE Scanning**: Automatic scanning for known vulnerabilities
3. **Update Approval**: Critical updates require explicit approval
4. **Audit Trail**: All version changes are logged with user attribution
5. **Rollback Protection**: Automatic rollback on security issues

## Performance Impact

Version management has minimal performance impact:

- Registry updates: Background process every 6 hours
- Version checks: Cached for 1 hour
- Compatibility checks: < 100ms
- History queries: Indexed for fast retrieval

## Extending Version Management

### Custom Validators

Add custom version validation logic:

```go
validator.AddPolicy(version.ValidationPolicy{
    Name: "production-stability",
    Validate: func(v string) error {
        // Custom validation logic
    },
})
```

### Custom Notification Handlers

Implement custom notification channels:

```go
type CustomHandler struct{}

func (h *CustomHandler) Handle(ctx context.Context, notification *version.UpgradeNotification) error {
    // Send to custom channel
}
```

## Migration from Manual Management

If migrating from manual version management:

1. **Inventory Current Versions**: Document all currently deployed versions
2. **Create Compatibility Matrix**: Define known working combinations
3. **Set Initial Pins**: Pin current versions initially
4. **Enable Gradually**: Start with notifications, then enable auto-updates
5. **Monitor Closely**: Watch version history and rollback if needed

## Future Enhancements

Planned features for version management:

- **AI-Powered Recommendations**: ML-based version compatibility predictions
- **Canary Updates**: Gradual rollout with automatic promotion
- **Cross-Cluster Coordination**: Synchronized updates across clusters
- **Dependency Resolution**: Automatic resolution of complex dependencies
- **Performance Prediction**: Estimate performance impact of updates
