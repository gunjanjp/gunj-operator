# Gunj Operator Webhooks

This package contains the admission webhooks for the Gunj Operator, implementing both mutating and validating webhooks for the ObservabilityPlatform CRD.

## Overview

Webhooks provide a way to extend the Kubernetes API server with custom validation and mutation logic. The Gunj Operator uses webhooks to:

1. **Mutating Webhooks**: Set default values and normalize configurations
2. **Validating Webhooks**: Ensure configurations are valid and safe

## Structure

```
internal/webhook/
├── setup.go                    # Main webhook setup
└── v1beta1/
    ├── observabilityplatform_webhook.go       # ObservabilityPlatform webhook implementation
    └── observabilityplatform_webhook_test.go  # Comprehensive tests
```

## Webhook Implementations

### ObservabilityPlatform Webhook

The ObservabilityPlatform webhook provides:

#### Defaulting (Mutating Webhook)
- Sets default replicas for all components
- Configures default resource requests and limits
- Sets retention policies for metrics and logs
- Generates secure passwords for Grafana admin
- Adjusts configurations for High Availability mode
- Sets default backup schedules

#### Validation (Validating Webhook)
- **Create Validation**:
  - Ensures at least one component is enabled
  - Validates version formats (semantic versioning)
  - Checks resource requirements meet minimums
  - Validates storage size requirements
  - Verifies network configurations
  - Validates security settings
  - Checks backup configurations
  - Ensures HA requirements are met

- **Update Validation**:
  - Prevents changes to immutable fields (e.g., storage class)
  - Blocks version downgrades
  - Warns about major version upgrades
  - Validates scaling changes
  - All create validations apply

- **Delete Validation**:
  - Warns about data loss for platforms with backups
  - Warns about persistent storage deletion
  - Extra warnings for production platforms

## Usage

### Enabling Webhooks

1. **Install cert-manager** (required for webhook certificates):
```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

2. **Deploy the operator with webhooks enabled**:
```bash
make deploy ENABLE_WEBHOOKS=true
```

### Testing Webhooks

Run the comprehensive test suite:
```bash
# Unit tests
make test-webhooks

# Integration tests with a real cluster
make test-webhooks-integration
```

### Example Validations

#### Valid Configuration
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: valid-platform
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      resources:
        requests:
          memory: "2Gi"
          cpu: "500m"
      storage:
        size: 10Gi
```

#### Invalid Configuration (will be rejected)
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: invalid-platform
spec:
  components:
    prometheus:
      enabled: true
      version: "invalid-version"  # ❌ Invalid version format
      resources:
        requests:
          memory: "64Mi"  # ❌ Too low (minimum 128Mi)
          cpu: "5m"       # ❌ Too low (minimum 10m)
      storage:
        size: 500Mi       # ❌ Too small (minimum 1Gi)
```

## Extending Webhooks

### Adding a New Webhook

1. **Create the webhook implementation**:
```go
// internal/webhook/v1beta1/newresource_webhook.go
package v1beta1

type NewResourceWebhook struct{}

func (r *NewResourceWebhook) Default(ctx context.Context, obj runtime.Object) error {
    // Implement defaulting logic
}

func (r *NewResourceWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
    // Implement creation validation
}

func (r *NewResourceWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
    // Implement update validation
}

func (r *NewResourceWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
    // Implement deletion validation
}
```

2. **Register the webhook** in `setup.go`:
```go
newResourceWebhook := &v1beta1webhook.NewResourceWebhook{}
if err := newResourceWebhook.SetupWebhookWithManager(mgr); err != nil {
    return err
}
```

3. **Add webhook configuration** in `config/webhook/manifests.yaml`

4. **Write comprehensive tests**

### Best Practices

1. **Validation Messages**: Provide clear, actionable error messages
2. **Warnings**: Use warnings for non-blocking issues that users should be aware of
3. **Performance**: Keep webhook logic fast to avoid API server timeouts
4. **Idempotency**: Defaulting webhooks must be idempotent
5. **Testing**: Write comprehensive tests for all validation scenarios

## Troubleshooting

### Common Issues

1. **Webhook timeout**: Ensure webhook logic completes within 10 seconds
2. **Certificate issues**: Check cert-manager is running and certificates are valid
3. **Webhook not called**: Verify webhook configuration and service are correct

### Debug Commands

```bash
# Check webhook configurations
kubectl get mutatingwebhookconfiguration
kubectl get validatingwebhookconfiguration

# View webhook logs
kubectl logs -n gunj-system deployment/gunj-operator -f | grep webhook

# Test webhook directly
kubectl create --dry-run=server -f platform.yaml
```

## Security Considerations

1. **Fail Closed**: Webhooks use `failurePolicy: Fail` to ensure security
2. **TLS**: All webhook communication is over TLS
3. **RBAC**: Webhook service account has minimal permissions
4. **Validation**: Comprehensive input validation prevents injection attacks

## Performance Considerations

1. **Caching**: Webhook reuses clients and caches where possible
2. **Minimal Logic**: Complex operations are deferred to controllers
3. **Fast Validation**: All validations complete in milliseconds

## Monitoring

Webhooks expose metrics:
- `webhook_request_total`: Total webhook requests
- `webhook_request_duration_seconds`: Request processing time
- `webhook_admission_duration_seconds`: Admission decision time
- `webhook_rejection_total`: Total rejected requests

## Contributing

When contributing webhook changes:
1. Follow the existing patterns
2. Add comprehensive validation tests
3. Document all validation rules
4. Update this README
5. Consider backward compatibility

## References

- [Kubernetes Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- [Kubebuilder Webhook Documentation](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html)
- [Webhook Best Practices](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#best-practices-and-warnings)
