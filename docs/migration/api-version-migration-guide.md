# API Version Migration Guide

This guide helps you migrate your ObservabilityPlatform resources between different API versions of the Gunj Operator.

## Table of Contents

- [Overview](#overview)
- [Migration Paths](#migration-paths)
- [Pre-Migration Checklist](#pre-migration-checklist)
- [Migration Methods](#migration-methods)
- [Step-by-Step Migration](#step-by-step-migration)
- [Troubleshooting](#troubleshooting)
- [Rollback Procedures](#rollback-procedures)

## Overview

The Gunj Operator supports multiple API versions to enable gradual adoption of new features while maintaining backward compatibility. This guide covers migrating resources between these versions.

### Supported Versions

- **v1alpha1**: Original version (deprecated, will be removed in v2.0.0)
- **v1beta1**: Current stable version with enhanced features
- **v1**: Future GA version (planned)

## Migration Paths

### v1alpha1 → v1beta1 (Recommended)

**Risk Level**: Low  
**Data Loss**: None  
**New Features Available**:
- Security configuration (TLS, PSP, Network Policies)
- Cost optimization features
- Service mesh integration
- Multi-cluster support
- Enhanced monitoring capabilities

### v1beta1 → v1alpha1 (Not Recommended)

**Risk Level**: High  
**Data Loss**: Significant  
**Lost Features**:
- Security configuration
- Cost optimization settings
- Service mesh integration
- Advanced Grafana plugins
- Additional Prometheus scrape configs
- Loki compactor settings
- Tempo search capabilities

## Pre-Migration Checklist

Before migrating your resources:

- [ ] **Backup existing resources**
  ```bash
  kubectl get observabilityplatforms -A -o yaml > platforms-backup.yaml
  ```

- [ ] **Check current versions**
  ```bash
  kubectl api-resources | grep observability
  ```

- [ ] **Review breaking changes**
  - Check the [Field Compatibility Matrix](field-compatibility-matrix.md)
  - Identify any custom configurations that might be affected

- [ ] **Test in non-production**
  - Always test migrations in development/staging first
  - Verify all components work as expected

- [ ] **Plan maintenance window**
  - Migrations may cause brief interruptions
  - Schedule during low-traffic periods

## Migration Methods

### Method 1: Automatic Conversion (Recommended)

The operator handles conversion automatically when you apply resources with different API versions.

**Advantages**:
- Seamless conversion
- Validation and warnings
- Automatic rollback on failure

**Process**:
1. Update the apiVersion in your YAML
2. Apply the resource
3. The conversion webhook handles the rest

### Method 2: Manual Migration

For more control over the migration process.

**Advantages**:
- Review changes before applying
- Modify configurations during migration
- Batch processing capabilities

**Process**:
1. Export existing resources
2. Convert using kubectl or custom scripts
3. Review and modify as needed
4. Apply converted resources

### Method 3: Dry-Run Migration

Test migrations without making changes.

**Advantages**:
- Preview conversion results
- Identify potential issues
- No risk to running systems

**Process**:
1. Use dry-run flag with kubectl
2. Review conversion warnings
3. Make necessary adjustments
4. Perform actual migration

## Step-by-Step Migration

### Migrating from v1alpha1 to v1beta1

#### 1. Perform Dry-Run

```bash
# Single resource dry-run
kubectl apply -f platform.yaml --dry-run=server -o yaml > converted.yaml

# Review the converted output
diff platform.yaml converted.yaml
```

#### 2. Update API Version

```yaml
# Before (v1alpha1)
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: production
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      customConfig:
        externalLabels: '{"cluster": "prod"}'

# After (v1beta1)
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      externalLabels:  # Migrated from customConfig
        cluster: prod
  security:            # New feature - configure as needed
    tls:
      enabled: true
```

#### 3. Handle Deprecated Fields

Fields that are deprecated in v1beta1:

| Deprecated Field | Migration Path |
|-----------------|----------------|
| `spec.components.prometheus.customConfig` | Use specific fields like `externalLabels` |
| `spec.alerting.alertManager` | Use `spec.alerting.alertmanager` (lowercase) |

#### 4. Configure New Features

Take advantage of new v1beta1 features:

```yaml
spec:
  # Security configuration
  security:
    tls:
      enabled: true
      certSecret: platform-tls
    podSecurityPolicy: true
    networkPolicy: true
    
  # Cost optimization
  costOptimization:
    enabled: true
    recommendations: true
    autoScale: true
    
  # Enhanced monitoring
  components:
    prometheus:
      additionalScrapeConfigs:
        - job_name: custom-metrics
          static_configs:
            - targets: ['custom-exporter:9090']
```

#### 5. Apply the Migration

```bash
# Apply with validation
kubectl apply -f platform-v1beta1.yaml

# Monitor the migration
kubectl get observabilityplatform production -w

# Check events
kubectl describe observabilityplatform production
```

#### 6. Verify Migration

```bash
# Check platform status
kubectl get observabilityplatform production -o jsonpath='{.status.phase}'

# Verify all components are running
kubectl get pods -l app.kubernetes.io/managed-by=gunj-operator

# Check conversion metrics
kubectl port-forward -n gunj-system svc/gunj-operator-metrics 8080:8080
curl http://localhost:8080/metrics | grep conversion
```

### Batch Migration Script

For migrating multiple resources:

```bash
#!/bin/bash

# migrate-platforms.sh
NAMESPACE=${1:-default}
BACKUP_DIR="backups/$(date +%Y%m%d-%H%M%S)"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup existing platforms
echo "Creating backups..."
kubectl get observabilityplatforms -n "$NAMESPACE" -o yaml > "$BACKUP_DIR/platforms-backup.yaml"

# Get list of platforms
platforms=$(kubectl get observabilityplatforms -n "$NAMESPACE" -o name)

for platform in $platforms; do
    name=$(echo "$platform" | cut -d'/' -f2)
    echo "Migrating $name..."
    
    # Export current state
    kubectl get "$platform" -n "$NAMESPACE" -o yaml > "$BACKUP_DIR/${name}-original.yaml"
    
    # Update API version
    kubectl get "$platform" -n "$NAMESPACE" -o yaml | \
        sed 's/apiVersion: observability.io\/v1alpha1/apiVersion: observability.io\/v1beta1/' | \
        kubectl apply --dry-run=client -f - -o yaml > "$BACKUP_DIR/${name}-converted.yaml"
    
    # Apply if dry-run succeeds
    if [ $? -eq 0 ]; then
        kubectl apply -f "$BACKUP_DIR/${name}-converted.yaml"
        echo "✓ $name migrated successfully"
    else
        echo "✗ $name migration failed - check $BACKUP_DIR/${name}-converted.yaml"
    fi
done

echo "Migration complete. Backups saved to $BACKUP_DIR"
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Validation Errors

**Error**: `error validating data: ValidationError(ObservabilityPlatform.spec.components.prometheus.resources)`

**Solution**: Ensure resource quantities use valid Kubernetes formats:
```yaml
# Correct
resources:
  requests:
    memory: 4Gi
    cpu: 1000m

# Incorrect
resources:
  requests:
    memory: 4GB  # Should be 4Gi
    cpu: 1 core  # Should be 1000m or 1
```

#### 2. Deprecated Field Warnings

**Warning**: `Field spec.components.prometheus.customConfig is deprecated`

**Solution**: Migrate to the new field structure:
```yaml
# Old (v1alpha1)
customConfig:
  externalLabels: '{"cluster": "prod"}'

# New (v1beta1)
externalLabels:
  cluster: prod
```

#### 3. Data Loss Warnings

**Warning**: `Field spec.security will be lost when converting to v1alpha1`

**Solution**: Only downgrade if absolutely necessary. Export important configurations first:
```bash
# Export security config before downgrade
kubectl get observabilityplatform production -o jsonpath='{.spec.security}' > security-config.json
```

#### 4. Conversion Webhook Failures

**Error**: `conversion webhook for observability.io/v1beta1 failed`

**Solution**:
1. Check webhook pod logs:
   ```bash
   kubectl logs -n gunj-system deployment/gunj-operator-webhook
   ```
2. Verify webhook certificate is valid:
   ```bash
   kubectl get validatingwebhookconfigurations gunj-operator-webhook -o yaml
   ```
3. Restart webhook if needed:
   ```bash
   kubectl rollout restart deployment/gunj-operator-webhook -n gunj-system
   ```

### Debug Commands

```bash
# Check conversion metrics
curl -s http://localhost:8080/metrics | grep gunj_operator_conversion

# View conversion events
kubectl get events --field-selector reason=ConversionFailed

# Check webhook logs with debug
kubectl logs -n gunj-system deployment/gunj-operator-webhook -f --v=5

# Test conversion manually
kubectl convert -f platform-v1alpha1.yaml --output-version observability.io/v1beta1
```

## Rollback Procedures

If migration causes issues, you can rollback to the previous version.

### Automatic Rollback

The operator automatically creates snapshots before conversion:

```bash
# List available rollback points
kubectl get observabilityplatform production -o jsonpath='{.metadata.annotations.conversion-snapshots}'

# Trigger rollback
kubectl annotate observabilityplatform production conversion.observability.io/rollback=true
```

### Manual Rollback

1. **Restore from backup**:
   ```bash
   kubectl apply -f platforms-backup.yaml
   ```

2. **Force specific version**:
   ```bash
   kubectl patch observabilityplatform production --type='merge' -p '{"spec":{"forceVersion":"v1alpha1"}}'
   ```

3. **Delete and recreate**:
   ```bash
   kubectl delete observabilityplatform production
   kubectl apply -f production-v1alpha1.yaml
   ```

### Post-Rollback Verification

```bash
# Verify platform is running
kubectl get observabilityplatform production

# Check component health
kubectl get pods -l platform=production

# Review logs for errors
kubectl logs -l app.kubernetes.io/name=gunj-operator --tail=100
```

## Best Practices

1. **Always use dry-run first**
   - Preview changes before applying
   - Identify potential issues early

2. **Migrate incrementally**
   - Start with non-production environments
   - Migrate one platform at a time
   - Monitor each migration

3. **Document customizations**
   - Keep track of custom configurations
   - Document any workarounds needed

4. **Monitor after migration**
   - Check metrics and logs
   - Verify all features work
   - Monitor for performance changes

5. **Plan for the future**
   - Stay informed about deprecations
   - Plan migrations well in advance
   - Test new features in lower environments

## Getting Help

- **Documentation**: https://gunjanjp.github.io/gunj-operator/docs
- **GitHub Issues**: https://github.com/gunjanjp/gunj-operator/issues
- **Community Slack**: #gunj-operator
- **Email Support**: gunjanjp@gmail.com

### Useful Links

- [API Reference](../api-reference.md)
- [Field Compatibility Matrix](field-compatibility-matrix.md)
- [Troubleshooting Guide](../troubleshooting.md)
- [Release Notes](../releases/)
