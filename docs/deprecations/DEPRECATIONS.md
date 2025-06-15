# Deprecation Guide for Gunj Operator

Last updated: 2025-06-14

## Table of Contents

- [Overview](#overview)
- [Deprecation Timeline](#deprecation-timeline)
- [API Version Deprecations](#api-version-deprecations)
- [Field Deprecations](#field-deprecations)
- [Value Deprecations](#value-deprecations)
- [Feature Deprecations](#feature-deprecations)
- [Migration Examples](#migration-examples)
- [Deprecation Policy](#deprecation-policy)
- [Checking for Deprecations](#checking-for-deprecations)
- [FAQ](#faq)

## Overview

This guide lists all deprecated features, fields, and values in the Gunj Operator. Deprecations are categorized by severity:

- 🚨 **Critical**: Will be removed in the next major version
- ⚠️  **Warning**: Deprecated but supported for at least two more versions
- ℹ️  **Info**: Deprecated for better alternatives but no removal planned

## Deprecation Timeline

| Date | Version | Event | Severity |
|------|---------|-------|----------|
| 2025-04-01 | v1beta1 | Deprecated: spec.tls.manual | ⚠️ |
| 2025-05-01 | v1beta1 | Deprecated: spec.components.prometheus.version < v2.40.0 | 🚨 |
| 2025-05-15 | v1beta1 | Deprecated: spec.storage.class | ℹ️ |
| 2025-06-01 | v1beta1 | Deprecated: spec.monitoring | ⚠️ |
| 2025-06-01 | v1beta1 | Deprecated: observability.io/v1alpha1 | ⚠️ |
| 2026-01-01 | v2.0.0 | Removal planned: observability.io/v1alpha1 | 🚨 |

## API Version Deprecations

### ⚠️ observability.io/v1alpha1

**Status**: API version v1alpha1 is deprecated and will be removed in v2.0.0

**Timeline**:
- Deprecated in: v1beta1 (since 2025-06-01)
- Will be removed in: v2.0.0
- Planned removal date: 2026-01-01

**Migration Guide**:

```yaml
# Update your resources to use v1beta1:
# 1. Change apiVersion from observability.io/v1alpha1 to observability.io/v1beta1
# 2. Update any deprecated fields as indicated in other warnings
# 3. Test your configurations thoroughly before the v1alpha1 removal
```

---

## Field Deprecations

### v1alpha1

#### ⚠️ spec.monitoring

**Status**: The 'spec.monitoring' field is deprecated and will be removed in v1

**Timeline**:
- Deprecated in: v1beta1 (since 2025-06-01)
- Will be removed in: v1

**Alternative**: Use `spec.components` instead

**Migration Guide**:

```yaml
# Migrate from 'spec.monitoring' to 'spec.components':
# Before:
#   spec:
#     monitoring:
#       prometheus:
#         enabled: true
# 
# After:
#   spec:
#     components:
#       prometheus:
#         enabled: true
```

---

#### ℹ️ spec.storage.class

**Status**: The 'spec.storage.class' field is deprecated. Use 'spec.storage.storageClassName' instead

**Timeline**:
- Deprecated in: v1beta1 (since 2025-05-15)
- Will be removed in: v1

**Alternative**: Use `spec.storage.storageClassName` instead

**Migration Guide**:

```yaml
# Update your storage configuration:
# Before:
#   spec:
#     storage:
#       class: fast-ssd
# 
# After:
#   spec:
#     storage:
#       storageClassName: fast-ssd
```

---

## Value Deprecations

### 🚨 spec.components.prometheus.version

**Status**: Prometheus versions below v2.40.0 are deprecated due to security vulnerabilities

**Deprecated Value**: `v2.30.0`

**Timeline**:
- Deprecated in: v1beta1 (since 2025-05-01)
- Will be removed in: v1

**Migration Guide**:

```yaml
# Update to a supported Prometheus version:
# Minimum supported version: v2.40.0
# Recommended version: v2.48.0 or later
# 
# Update your configuration:
#   spec:
#     components:
#       prometheus:
#         version: v2.48.0
```

---

## Feature Deprecations

### ⚠️ spec.tls.manual

**Status**: Manual TLS configuration is deprecated. Use cert-manager integration instead

**Timeline**:
- Deprecated in: v1beta1 (since 2025-04-01)
- Will be removed in: v1

**Alternative**: Use `spec.tls.certManager` instead

**Migration Guide**:

```yaml
# Migrate to cert-manager based TLS:
# 1. Install cert-manager in your cluster
# 2. Remove spec.tls.manual configuration
# 3. Use spec.tls.certManager instead:
# 
#   spec:
#     tls:
#       certManager:
#         enabled: true
#         issuerRef:
#           name: letsencrypt-prod
#           kind: ClusterIssuer
```

---

## Migration Examples

### Complete v1alpha1 to v1beta1 Migration

Here's a complete example of migrating an ObservabilityPlatform from v1alpha1 to v1beta1:

**Before (v1alpha1)**:
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: production
spec:
  monitoring:
    prometheus:
      enabled: true
      version: v2.30.0
  storage:
    class: fast-ssd
  tls:
    manual:
      cert: |
        -----BEGIN CERTIFICATE-----
        ...
```

**After (v1beta1)**:
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
spec:
  components:  # Changed from 'monitoring'
    prometheus:
      enabled: true
      version: v2.48.0  # Updated to supported version
  storage:
    storageClassName: fast-ssd  # Changed from 'class'
  tls:
    certManager:  # Changed from 'manual'
      enabled: true
      issuerRef:
        name: letsencrypt-prod
        kind: ClusterIssuer
```

## Checking for Deprecations

### Using kubectl

You can check your resources for deprecations using kubectl:

```bash
# Apply with dry-run to see warnings
kubectl apply -f my-platform.yaml --dry-run=server

# The output will include deprecation warnings:
# Warning: spec.monitoring is deprecated, use spec.components instead
# Warning: Prometheus version v2.30.0 is deprecated due to security vulnerabilities
```

### Using the Operator Logs

Check the operator logs for deprecation warnings:

```bash
# Check operator logs for deprecation warnings
kubectl logs -n gunj-system deployment/gunj-operator | grep -i deprecat
```

### Using the Deprecation Documentation Tool

Generate up-to-date deprecation documentation:

```bash
# Generate markdown documentation
go run cmd/deprecation-doc/main.go

# Generate YAML format
go run cmd/deprecation-doc/main.go -format yaml

# Generate all formats
go run cmd/deprecation-doc/main.go -all
```

## Deprecation Policy

The Gunj Operator follows these deprecation guidelines:

### Version Support

- **Alpha versions** (v1alpha1, v1alpha2, etc.): No compatibility guarantees
- **Beta versions** (v1beta1, v1beta2, etc.): Compatible for at least 2 releases
- **Stable versions** (v1, v2, etc.): Compatible for at least 3 releases

### Deprecation Process

1. **Announcement**: Deprecations are announced in release notes
2. **Warning Period**: Deprecated features show warnings but continue to work
3. **Migration Period**: At least 2 releases for beta, 3 for stable APIs
4. **Removal**: Features are removed only in major version updates

### Monitoring Deprecations

You can monitor deprecation usage in your cluster:

```bash
# Check operator logs for deprecation warnings
kubectl logs -n gunj-system deployment/gunj-operator | grep -i deprecat

# Use kubectl deprecations (if available)
kubectl deprecations
```

### Getting Help

If you need help with migrations:

- Check the [migration examples](#migration-examples) above
- Join our [community Slack](https://gunj-operator.slack.com)
- Open an issue on [GitHub](https://github.com/gunjanjp/gunj-operator/issues)
- Email: gunjanjp@gmail.com

## FAQ

### Q: What happens if I don't migrate before removal?

A: Your resources will fail to apply after the deprecated features are removed. The operator will reject resources using removed fields or API versions. Always test migrations in a non-production environment first.

### Q: Can I disable deprecation warnings?

A: No, deprecation warnings cannot be disabled. They are important notifications to help you prepare for future changes.

### Q: How do I know which version of the operator I'm running?

A: Check the operator version:
```bash
kubectl get deployment -n gunj-system gunj-operator -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### Q: Will my existing resources be automatically migrated?

A: No, automatic migration is limited to simple cases. You need to manually update your resource definitions to use the new fields and API versions.

### Q: Can I skip versions when upgrading?

A: It's recommended to upgrade sequentially through versions to ensure all migrations are properly handled. Skipping versions may lead to compatibility issues.

### Q: How do I test migrations safely?

A: 
1. Create a test namespace
2. Apply your resources with `--dry-run=server` to check for warnings
3. Deploy in the test namespace and verify functionality
4. Use version control to track your migration changes
5. Have a rollback plan ready

### Q: What if a deprecated feature is critical for my use case?

A: If you depend on a deprecated feature:
1. Open an issue explaining your use case
2. Engage with the community to find alternatives
3. Consider contributing a replacement feature
4. Plan for migration well before the removal date
