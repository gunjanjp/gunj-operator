# Configuration Validation

## Overview

The Gunj Operator includes comprehensive configuration validation to ensure that ObservabilityPlatform resources are correctly configured before deployment. This validation happens through admission webhooks and covers various aspects of the configuration.

## Validation Types

### 1. Component Version Compatibility

The operator validates that component versions are compatible with each other. Not all versions of Prometheus, Grafana, Loki, and Tempo work well together.

**Example Compatible Versions:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
    grafana:
      enabled: true
      version: 10.2.0  # Compatible with Prometheus v2.48.0
    loki:
      enabled: true
      version: 2.9.0   # Compatible with Prometheus v2.48.0
    tempo:
      enabled: true
      version: 2.3.0   # Compatible with Prometheus v2.48.0
```

### 2. Configuration Conflict Detection

The validator checks for conflicting configurations that could cause deployment issues:

- **HA Mode Conflicts**: Ensures sufficient replicas when HA is enabled
- **Resource Conflicts**: Validates that resource requests don't exceed limits
- **Storage Conflicts**: Checks for storage class availability and conflicts
- **Global vs Component Settings**: Detects conflicting settings between global and component-specific configurations

**Example HA Configuration:**
```yaml
spec:
  highAvailability:
    enabled: true
    antiAffinity: soft
  components:
    prometheus:
      enabled: true
      replicas: 3  # Must be at least 2 for HA mode
      storage:
        size: 100Gi  # Storage required for HA mode
```

### 3. Configuration Completeness

Ensures all required configurations are present:

- At least one component must be enabled
- Required fields for enabled features (e.g., backup schedule)
- Dependencies between components (e.g., Alertmanager requires Prometheus)

**Example Backup Configuration:**
```yaml
spec:
  backup:
    enabled: true
    schedule: "0 2 * * *"  # Required when backup is enabled
    storageType: s3        # Required when backup is enabled
    storageConfig:
      bucket: my-backups   # Required for S3 storage
      region: us-east-1    # Required for S3 storage
```

### 4. Resource Validation

Validates all resource specifications:

- CPU and memory quantities are valid
- Storage sizes use proper units
- Resource requests don't exceed limits

**Example Resource Configuration:**
```yaml
spec:
  components:
    prometheus:
      resources:
        requests:
          cpu: "500m"      # Valid CPU quantity
          memory: "2Gi"    # Valid memory quantity
        limits:
          cpu: "2"         # Must be >= requests
          memory: "8Gi"    # Must be >= requests
```

### 5. Configuration Dependencies

Validates that dependent configurations are properly set:

- OpenTelemetry Collector requires at least one backend
- Grafana data sources require corresponding components to be enabled
- Remote write configurations require proper authentication

**Example Dependency Configuration:**
```yaml
spec:
  components:
    opentelemetryCollector:
      enabled: true
      mode: deployment
    # At least one backend must be enabled
    prometheus:
      enabled: true
    grafana:
      enabled: true
      dataSources:
        - name: prometheus
          type: prometheus
          url: http://prometheus:9090
          # Prometheus must be enabled for this data source
```

### 6. Deprecated Configuration Detection

The validator warns about deprecated configuration options:

```yaml
# This will generate a warning
spec:
  components:
    prometheus:
      adminUser: admin  # Deprecated: use auth.adminUser instead
```

## Validation Error Messages

The validator provides detailed error messages to help fix configuration issues:

```
spec.components.grafana.version: Invalid value: "9.0.0": Grafana version 9.0.0 is not compatible with Prometheus version v2.48.0
spec.components.prometheus.storage: Required value: storage is required for Prometheus in HA mode
spec.backup.storageConfig.bucket: Required value: bucket is required for S3 backup
```

## Custom Validation Rules

### Version Format Validation

Versions must follow semantic versioning:
- Valid: `v2.48.0`, `2.48.0`, `10.2.0`
- Invalid: `2.48`, `latest`, `invalid-version`

### Retention Format Validation

Retention periods must use valid time units:
- Valid: `30d`, `24h`, `52w`, `12m`, `5y`
- Invalid: `30days`, `24`, `invalid`

### Label Validation

Labels must follow Kubernetes naming conventions:
- Keys: max 63 characters, alphanumeric with dots
- Values: max 63 characters, alphanumeric with hyphens

## Configuration Examples

### Minimal Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: minimal
spec:
  components:
    prometheus:
      enabled: true
```

### Production Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      replicas: 3
      resources:
        requests:
          cpu: "1"
          memory: "4Gi"
        limits:
          cpu: "2"
          memory: "8Gi"
      storage:
        size: 100Gi
        retention: 30d
      externalLabels:
        cluster: production
        region: us-east-1
    grafana:
      enabled: true
      version: 10.2.0
      replicas: 2
      ingress:
        enabled: true
        host: grafana.example.com
        tls:
          enabled: true
          secretName: grafana-tls
    loki:
      enabled: true
      version: 2.9.0
      storage:
        size: 200Gi
        s3:
          enabled: true
          bucketName: my-loki-logs
          region: us-east-1
  highAvailability:
    enabled: true
    antiAffinity: hard
  backup:
    enabled: true
    schedule: "0 2 * * *"
    retention: 7
    storageType: s3
    storageConfig:
      bucket: observability-backups
      region: us-east-1
```

## Troubleshooting

### Common Validation Errors

1. **Version Compatibility Error**
   ```
   Error: Grafana version 9.0.0 is not compatible with Prometheus version v2.48.0
   ```
   **Solution**: Check the version compatibility matrix and use compatible versions.

2. **Missing Required Fields**
   ```
   Error: backup schedule is required when backup is enabled
   ```
   **Solution**: Add the missing required field to your configuration.

3. **Resource Validation Error**
   ```
   Error: CPU request cannot exceed limit
   ```
   **Solution**: Ensure resource requests are less than or equal to limits.

4. **Component Dependency Error**
   ```
   Error: Prometheus data source requires Prometheus to be enabled
   ```
   **Solution**: Enable the required component or remove the dependent configuration.

### Validation Bypass

In rare cases where you need to bypass validation (not recommended for production):

```yaml
metadata:
  annotations:
    observability.io/skip-validation: "true"  # Use with caution
```

## Best Practices

1. **Start with Minimal Configuration**: Begin with the smallest working configuration and add features incrementally.

2. **Use Version Compatibility Matrix**: Always check component version compatibility before upgrading.

3. **Test in Non-Production**: Validate configurations in development/staging before production.

4. **Monitor Deprecation Warnings**: Address deprecated configurations before they're removed.

5. **Resource Planning**: Plan resource requirements based on workload and scale appropriately.

## Version Compatibility Matrix

| Prometheus | Compatible Grafana | Compatible Loki | Compatible Tempo | Compatible OTel |
|------------|-------------------|-----------------|------------------|-----------------|
| v2.48.x    | 10.0, 10.1, 10.2 | 2.9             | 2.3              | 0.91, 0.92      |
| v2.47.x    | 9.5, 10.0, 10.1  | 2.8, 2.9        | 2.2, 2.3         | 0.88-0.91       |
| v2.46.x    | 9.4, 9.5, 10.0   | 2.8             | 2.2              | 0.86-0.88       |

## Related Documentation

- [ObservabilityPlatform CRD Reference](./crd-reference.md)
- [Component Configuration Guide](./component-configuration.md)
- [Troubleshooting Guide](./troubleshooting.md)
