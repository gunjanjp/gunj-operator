# Field Compatibility Matrix - Gunj Operator API Versions

This document provides a comprehensive field compatibility matrix for conversions between different API versions of the Gunj Operator.

## Overview

The Gunj Operator supports multiple API versions to enable gradual feature adoption and backward compatibility. This matrix details how fields map between versions and what changes occur during conversion.

## Version Support

| Version | Status | Support Level | Notes |
|---------|--------|---------------|--------|
| v1alpha1 | Deprecated | Maintenance only | Original version, will be removed in v2.0.0 |
| v1beta1 | Current | Full support | Current stable version with all features |
| v1 | Planned | Future | GA version planned for Q3 2025 |

## Conversion Compatibility

| From Version | To Version | Compatibility | Data Loss Risk | Notes |
|--------------|------------|---------------|----------------|--------|
| v1alpha1 | v1beta1 | ✅ Full | None | All fields preserved or enhanced |
| v1beta1 | v1alpha1 | ⚠️ Partial | High | Several fields will be lost |
| v1beta1 | v1 | 🔄 Planned | None | Forward compatibility ensured |

## Field Mapping Details

### Core Fields

| v1alpha1 Field | v1beta1 Field | Conversion Type | Notes |
|----------------|---------------|-----------------|--------|
| `spec.paused` | `spec.paused` | Direct | No changes |
| `metadata` | `metadata` | Direct | All Kubernetes metadata preserved |
| `status.phase` | `status.phase` | Direct | Same phase values |
| `status.message` | `status.message` | Direct | No changes |

### Component Configuration

#### Prometheus

| v1alpha1 Field | v1beta1 Field | Conversion Type | Notes |
|----------------|---------------|-----------------|--------|
| `spec.components.prometheus.enabled` | `spec.components.prometheus.enabled` | Direct | No changes |
| `spec.components.prometheus.version` | `spec.components.prometheus.version` | Validated | Version format validation added |
| `spec.components.prometheus.replicas` | `spec.components.prometheus.replicas` | Direct | Default: 1 if not specified |
| `spec.components.prometheus.resources` | `spec.components.prometheus.resources` | Transform | String to `resource.Quantity` |
| `spec.components.prometheus.storage.size` | `spec.components.prometheus.storage.size` | Transform | String to `resource.Quantity` |
| `spec.components.prometheus.customConfig` | `spec.components.prometheus.externalLabels` | Transform | Map extraction for `externalLabels` key |
| N/A | `spec.components.prometheus.additionalScrapeConfigs` | New | Not available in v1alpha1 |

#### Grafana

| v1alpha1 Field | v1beta1 Field | Conversion Type | Notes |
|----------------|---------------|-----------------|--------|
| `spec.components.grafana.enabled` | `spec.components.grafana.enabled` | Direct | No changes |
| `spec.components.grafana.version` | `spec.components.grafana.version` | Validated | Version format validation |
| `spec.components.grafana.adminPassword` | `spec.components.grafana.adminPassword` | Direct | Consider using secrets |
| `spec.components.grafana.ingress` | `spec.components.grafana.ingress` | Enhanced | Additional TLS options in v1beta1 |
| N/A | `spec.components.grafana.plugins` | New | Plugin management added |
| N/A | `spec.components.grafana.smtp` | New | Email configuration added |

#### Loki

| v1alpha1 Field | v1beta1 Field | Conversion Type | Notes |
|----------------|---------------|-----------------|--------|
| `spec.components.loki.enabled` | `spec.components.loki.enabled` | Direct | No changes |
| `spec.components.loki.version` | `spec.components.loki.version` | Validated | Version format validation |
| N/A | `spec.components.loki.replicas` | New | Defaults to 1 |
| `spec.components.loki.storage` | `spec.components.loki.storage` | Transform | String sizes to quantities |
| `spec.components.loki.s3` | `spec.components.loki.s3` | Direct | S3 configuration preserved |
| N/A | `spec.components.loki.compactorEnabled` | New | Compactor feature in v1beta1 |

#### Tempo

| v1alpha1 Field | v1beta1 Field | Conversion Type | Notes |
|----------------|---------------|-----------------|--------|
| `spec.components.tempo.enabled` | `spec.components.tempo.enabled` | Direct | No changes |
| `spec.components.tempo.version` | `spec.components.tempo.version` | Validated | Version format validation |
| N/A | `spec.components.tempo.replicas` | New | Defaults to 1 |
| `spec.components.tempo.storage` | `spec.components.tempo.storage` | Transform | String sizes to quantities |
| N/A | `spec.components.tempo.searchEnabled` | New | Search feature in v1beta1 |

### Global Configuration

| v1alpha1 Field | v1beta1 Field | Conversion Type | Notes |
|----------------|---------------|-----------------|--------|
| `spec.global.externalLabels` | `spec.global.externalLabels` | Direct | Label validation added |
| `spec.global.logLevel` | `spec.global.logLevel` | Direct | No changes |
| `spec.global.nodeSelector` | `spec.global.nodeSelector` | Direct | No changes |
| `spec.global.tolerations` | `spec.global.tolerations` | Transform | String to K8s toleration type |
| N/A | `spec.global.affinity` | New | Pod affinity rules |
| N/A | `spec.global.imagePullSecrets` | New | Private registry support |

### New v1beta1 Features

These fields are only available in v1beta1 and will be lost when converting back to v1alpha1:

| v1beta1 Field | Description | Default Value |
|---------------|-------------|---------------|
| `spec.security` | Security configuration including TLS, PSP, network policies | TLS enabled, PSP/NetPol true |
| `spec.costOptimization` | Cost optimization features and recommendations | Disabled by default |
| `spec.serviceMesh` | Service mesh integration (Istio/Linkerd) | Disabled |
| `spec.multiCluster` | Multi-cluster federation support | Disabled |
| `spec.backup.encryption` | Backup encryption settings | Enabled with AES-256 |
| `spec.highAvailability.antiAffinity` | Pod anti-affinity rules | Preferred anti-affinity |

### Status Field Changes

| v1alpha1 Field | v1beta1 Field | Conversion Type | Notes |
|----------------|---------------|-----------------|--------|
| `status.componentStatus` | `status.componentStatus` | Enhanced | Additional fields in v1beta1 |
| N/A | `status.version` | New | Platform version tracking |
| N/A | `status.health` | New | Overall health status |
| `status.endpoints` | `status.endpoints` | Direct | Service endpoints preserved |

## Deprecation Warnings

### Fields Deprecated in v1beta1

| Field | Deprecation Version | Removal Version | Migration Guide |
|-------|-------------------|-----------------|-----------------|
| `spec.components.prometheus.customConfig` | v1beta1 | v1.0.0 | Use `spec.components.prometheus.externalLabels` for external labels |
| `spec.alerting.alertManager` | v1beta1 | v1.0.0 | Use `spec.alerting.alertmanager` (lowercase 'm') |

## Validation Rules

### Resource Quantities

When converting from v1alpha1 to v1beta1:
- String quantities (e.g., "4Gi") are parsed to Kubernetes `resource.Quantity`
- Invalid quantities will cause validation errors
- Minimum values enforced:
  - Memory: 128Mi
  - CPU: 100m
  - Storage: 1Gi

### Version Validation

- Must follow semantic versioning: `v<major>.<minor>.<patch>`
- Examples: `v2.48.0`, `v10.2.0`
- Major version changes not allowed during conversion

### Label Validation

- Keys and values limited to 63 characters
- Must follow Kubernetes label conventions
- Cannot remove labels during conversion

## Conversion Examples

### Successful v1alpha1 to v1beta1

```yaml
# v1alpha1
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      resources:
        requests:
          memory: "4Gi"
          cpu: "1"
      customConfig:
        externalLabels: '{"cluster": "prod"}'

# Converts to v1beta1:
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      resources:
        requests:
          memory: 4Gi  # Quantity type
          cpu: 1000m   # Quantity type
      externalLabels:
        cluster: prod  # Extracted from customConfig
  security:           # Added with defaults
    tls:
      enabled: true
    podSecurityPolicy: true
    networkPolicy: true
```

### Lossy v1beta1 to v1alpha1

```yaml
# v1beta1
spec:
  components:
    prometheus:
      enabled: true
      additionalScrapeConfigs: |  # LOST
        - job_name: custom
    grafana:
      plugins:                    # LOST
        - grafana-piechart-panel
  security:                       # LOST
    tls:
      enabled: true

# Converts to v1alpha1:
spec:
  components:
    prometheus:
      enabled: true
    grafana:
      enabled: true
# Security, plugins, and additional configs are lost
```

## Best Practices

1. **Always test conversions** in non-production environments first
2. **Create backups** before converting production resources
3. **Use dry-run** to preview conversion results
4. **Monitor metrics** after conversion for any anomalies
5. **Review deprecation warnings** and plan migrations accordingly

## Troubleshooting

### Common Conversion Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Invalid resource quantity | Malformed size strings | Use valid Kubernetes quantities (e.g., "1Gi", "500m") |
| Version validation failure | Non-semver version format | Use format: v1.2.3 |
| Label validation error | Label key/value too long | Limit to 63 characters |
| Selector immutability | Attempting to change selectors | Selectors cannot be changed |

### Debug Commands

```bash
# Dry-run conversion
kubectl convert -f platform.yaml --output-version=observability.io/v1beta1 --dry-run

# Check conversion webhook logs
kubectl logs -n gunj-system deployment/gunj-operator-webhook -f

# View conversion metrics
kubectl port-forward -n gunj-system svc/gunj-operator-metrics 8080:8080
curl http://localhost:8080/metrics | grep conversion
```

## Version Migration Timeline

| Date | Action | Version Changes |
|------|--------|-----------------|
| 2025-01 | v1beta1 Release | v1alpha1 deprecated |
| 2025-07 | v1 Release | v1beta1 becomes legacy |
| 2026-01 | v1alpha1 Removal | Only v1beta1 and v1 supported |
| 2026-07 | v1beta1 Deprecation | v1 is the only supported version |

## Support

For conversion issues or questions:
- GitHub Issues: https://github.com/gunjanjp/gunj-operator/issues
- Email: gunjanjp@gmail.com
- Documentation: https://gunjanjp.github.io/gunj-operator/docs/api/conversion
