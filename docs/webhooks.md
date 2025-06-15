# Gunj Operator Webhook Documentation

## Overview

The Gunj Operator uses Kubernetes admission webhooks to validate ObservabilityPlatform resources before they are stored in the cluster. This ensures that only valid configurations are accepted, preventing runtime errors and misconfigurations.

## Validating Webhooks

### Purpose

Validating webhooks intercept requests to create or update ObservabilityPlatform resources and validate them against a set of rules. If validation fails, the request is rejected with a clear error message.

### Validation Rules

#### Component Validation

1. **At least one component must be enabled**
   - The platform must have at least one of Prometheus, Grafana, Loki, or Tempo enabled
   - Empty or all-disabled configurations are rejected

2. **Version format validation**
   - Component versions must follow semantic versioning: `vX.Y.Z` or `X.Y.Z`
   - Pre-release versions are allowed: `vX.Y.Z-rc.1`

3. **Replica validation**
   - Replica count must be at least 1
   - For high availability mode, replicas must be at least 2

4. **Resource validation**
   - CPU and memory quantities must be valid Kubernetes resource quantities
   - Resource requests cannot exceed limits
   - Examples: `100m`, `500Mi`, `2Gi`, `1`

#### Prometheus-specific Validation

- **Storage size**: Must be a valid quantity (e.g., `10Gi`, `100Gi`)
- **Retention**: Must follow the pattern `\d+[hdwmy]` (e.g., `30d`, `12h`, `1w`)
- **External labels**: Must be valid Kubernetes label names and values
- **Remote write**: URL must be provided if remote write is configured

#### Grafana-specific Validation

- **Ingress**: Host must be specified when ingress is enabled
- **Persistence**: Size must be a valid quantity

#### Loki-specific Validation

- **S3 storage**: When enabled, bucket name and region are required
- **Storage size**: Must be a valid quantity

#### Tempo-specific Validation

- **Storage size**: Must be a valid quantity

#### Global Settings Validation

- **Log level**: Must be one of: `debug`, `info`, `warn`, `error`
- **External labels**: Must follow Kubernetes label conventions

#### High Availability Validation

When HA is enabled:
- Prometheus must have at least 2 replicas
- Grafana must have at least 2 replicas
- Anti-affinity must be `hard` or `soft`

#### Backup Settings Validation

When backups are enabled:
- **Schedule**: Must be a valid cron expression
- **Retention**: Must be at least 1 day
- **Storage type**: Must be one of: `s3`, `azure`, `gcs`, `local`

### Update Validation

In addition to create validation, updates have additional checks:

1. **Version downgrade warnings**: Warns when downgrading component versions
2. **Immutable field validation**: Certain fields cannot be changed after creation

### Examples

#### Valid Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      replicas: 2
      resources:
        requests:
          cpu: "500m"
          memory: "2Gi"
        limits:
          cpu: "2"
          memory: "4Gi"
```

#### Invalid Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: invalid
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      version: "invalid"  # ERROR: Invalid version format
      replicas: 0         # ERROR: Must be at least 1
```

### Error Messages

When validation fails, the webhook returns detailed error messages:

```
error validating data: ValidationError(ObservabilityPlatform.spec.components.prometheus.version): 
invalid value: invalid, must match pattern ^v?\d+\.\d+\.\d+$
```

### Troubleshooting

1. **Webhook not working**
   - Ensure cert-manager is installed
   - Check webhook service is running
   - Verify webhook configuration is applied

2. **Validation too strict**
   - Check the validation rules in the webhook code
   - Submit an issue if you believe validation is incorrect

3. **Debugging validation**
   - Use `kubectl --validate=false` to bypass client-side validation
   - Check operator logs for webhook errors

## Webhook Configuration

### Prerequisites

- cert-manager installed in the cluster for certificate management
- Webhook service running on port 9443

### Installation

The webhook is automatically configured when installing the operator:

```bash
kubectl apply -f config/webhook/
```

### Manual Testing

Test the webhook with example configurations:

```bash
# Should succeed
kubectl apply -f examples/valid-platform.yaml

# Should fail with validation errors
kubectl apply -f examples/invalid-platform.yaml
```

## Development

### Adding New Validations

1. Edit `api/v1beta1/observabilityplatform_webhook.go`
2. Add validation logic in the appropriate validation method
3. Add tests in `api/v1beta1/test/observabilityplatform_webhook_test.go`
4. Update this documentation

### Testing Webhooks

Run webhook tests:

```bash
go test ./api/v1beta1/... -v
```

### Debugging

Enable debug logging in the operator:

```yaml
env:
- name: LOG_LEVEL
  value: debug
```

Check webhook logs:

```bash
kubectl logs -n gunj-system deployment/gunj-operator -f | grep webhook
```
