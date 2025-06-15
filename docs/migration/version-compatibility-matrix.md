# Version Compatibility Matrix

## Overview

This document provides a comprehensive compatibility matrix for all versions of the Gunj Operator, including API versions, Kubernetes versions, component versions, and feature availability.

## Quick Reference

### Current Recommendations

| Component | Recommended Version | Minimum Version | Maximum Version |
|-----------|-------------------|-----------------|-----------------|
| Operator | v1.5.0+ | v1.0.0 | latest |
| Kubernetes | v1.28 | v1.26 | v1.29 |
| API Version | v1beta1 | v1alpha1* | v1beta1 |
| Prometheus | v2.48.0 | v2.40.0 | v2.48.0 |
| Grafana | 10.2.0 | 9.0.0 | 10.2.0 |
| Loki | 2.9.0 | 2.7.0 | 2.9.0 |
| Tempo | 2.3.0 | 2.0.0 | 2.3.0 |

*v1alpha1 is deprecated and will be removed in v2.0.0

## Detailed Compatibility Matrix

### Operator Version Compatibility

| Operator Version | API Versions Supported | Kubernetes Versions | Status | End of Support |
|-----------------|----------------------|-------------------|---------|----------------|
| 0.1.0 - 0.4.9 | v1alpha1 | 1.24 - 1.25 | EOL | 2023-07-15 |
| 0.5.0 - 0.9.9 | v1alpha1 | 1.24 - 1.26 | Deprecated | 2024-01-15 |
| 1.0.0 - 1.4.9 | v1alpha1, v1beta1* | 1.25 - 1.27 | Maintenance | 2024-07-15 |
| 1.5.0 - 1.9.9 | v1alpha1*, v1beta1 | 1.26 - 1.28 | **Current** | 2025-01-15 |
| 2.0.0+ | v1beta1, v1 | 1.27 - 1.29 | Future | TBD |

**Legend:**
- EOL: End of Life - no support
- Deprecated: Security fixes only
- Maintenance: Bug fixes only
- Current: Full support
- `*` Limited support (read-only or conversion only)

### API Version Features

| Feature | v1alpha1 | v1beta1 | v1 |
|---------|----------|---------|-----|
| Basic Deployment | ✅ | ✅ | ✅ |
| Custom Config | ✅ (string) | ⚠️ (structured) | ❌ |
| Resource Management | ⚠️ (basic) | ✅ (standard) | ✅ |
| High Availability | ❌ | ✅ | ✅ |
| Multi-cluster | ❌ | ✅ | ✅ |
| Cost Management | ❌ | ✅ | ✅ |
| Auto-scaling | ❌ | ✅ | ✅ |
| Backup/Restore | ⚠️ | ✅ | ✅ |
| GitOps Integration | ⚠️ | ✅ | ✅ |
| Service Mesh | ❌ | ✅ | ✅ |
| FIPS Compliance | ❌ | ⚠️ | ✅ |

### Kubernetes API Compatibility

| Kubernetes Version | Supported APIs | Operator Versions | Notes |
|-------------------|----------------|------------------|--------|
| 1.24 | v1alpha1 | 0.1.0 - 0.9.9 | EOL |
| 1.25 | v1alpha1, v1beta1 | 0.5.0 - 1.4.9 | |
| 1.26 | v1alpha1, v1beta1 | 0.9.0 - 1.9.9 | |
| 1.27 | v1beta1 | 1.0.0+ | Recommended |
| 1.28 | v1beta1 | 1.5.0+ | **Current** |
| 1.29 | v1beta1, v1 | 1.5.0+ | Latest |
| 1.30 | v1beta1, v1 | 2.0.0+ | Future |

### Component Version Compatibility

#### Prometheus Compatibility

| Operator Version | Min Prometheus | Max Prometheus | Recommended | Features |
|-----------------|----------------|----------------|-------------|----------|
| 0.x | 2.30.0 | 2.40.0 | 2.35.0 | Basic |
| 1.0 - 1.4 | 2.35.0 | 2.45.0 | 2.40.0 | Remote Write v2 |
| 1.5 - 1.9 | 2.40.0 | 2.48.0 | 2.48.0 | Agent Mode |
| 2.0+ | 2.45.0 | latest | 2.48.0 | Native Histograms |

#### Grafana Compatibility

| Operator Version | Min Grafana | Max Grafana | Recommended | Features |
|-----------------|-------------|-------------|-------------|----------|
| 0.x | 8.0.0 | 9.0.0 | 8.5.0 | Basic |
| 1.0 - 1.4 | 8.5.0 | 9.5.0 | 9.0.0 | Unified Alerting |
| 1.5 - 1.9 | 9.0.0 | 10.2.0 | 10.2.0 | Public Dashboards |
| 2.0+ | 10.0.0 | latest | 10.2.0 | Correlations |

#### Loki Compatibility

| Operator Version | Min Loki | Max Loki | Recommended | Features |
|-----------------|----------|----------|-------------|----------|
| 0.x | 2.4.0 | 2.6.0 | 2.5.0 | Basic |
| 1.0 - 1.4 | 2.6.0 | 2.8.0 | 2.7.0 | TSDB Index |
| 1.5 - 1.9 | 2.7.0 | 2.9.0 | 2.9.0 | Pattern Parser |
| 2.0+ | 2.8.0 | latest | 2.9.0 | Bloom Filters |

#### Tempo Compatibility

| Operator Version | Min Tempo | Max Tempo | Recommended | Features |
|-----------------|-----------|-----------|-------------|----------|
| 0.x | 1.5.0 | 2.0.0 | 1.5.0 | Basic |
| 1.0 - 1.4 | 2.0.0 | 2.2.0 | 2.1.0 | Parquet |
| 1.5 - 1.9 | 2.0.0 | 2.3.0 | 2.3.0 | TraceQL |
| 2.0+ | 2.2.0 | latest | 2.3.0 | Metrics Gen |

### Storage Backend Compatibility

| Storage Type | v1alpha1 | v1beta1 | v1 | Notes |
|-------------|----------|---------|-----|--------|
| Local Disk | ✅ | ✅ | ✅ | Default |
| AWS EBS | ✅ | ✅ | ✅ | gp3 recommended |
| GCP PD | ✅ | ✅ | ✅ | pd-ssd recommended |
| Azure Disk | ✅ | ✅ | ✅ | Premium SSD |
| Ceph/Rook | ⚠️ | ✅ | ✅ | v1.11+ required |
| Longhorn | ⚠️ | ✅ | ✅ | v1.4+ required |
| OpenEBS | ❌ | ✅ | ✅ | v3.0+ required |
| Portworx | ❌ | ✅ | ✅ | Enterprise only |

### Cloud Provider Integration

| Provider | v1alpha1 | v1beta1 | v1 | Features |
|----------|----------|---------|-----|----------|
| AWS | ⚠️ | ✅ | ✅ | S3, EBS, IAM |
| GCP | ⚠️ | ✅ | ✅ | GCS, PD, Workload Identity |
| Azure | ⚠️ | ✅ | ✅ | Blob, Disk, AD |
| On-Premise | ✅ | ✅ | ✅ | MinIO, NFS |
| OpenStack | ❌ | ⚠️ | ✅ | Swift, Cinder |
| vSphere | ❌ | ⚠️ | ✅ | vSAN |

### Authentication Provider Compatibility

| Provider | v1alpha1 | v1beta1 | v1 | Notes |
|----------|----------|---------|-----|--------|
| Basic Auth | ✅ | ✅ | ⚠️ | Deprecated in v1 |
| OIDC | ❌ | ✅ | ✅ | Recommended |
| LDAP | ❌ | ✅ | ✅ | |
| SAML | ❌ | ✅ | ✅ | |
| OAuth2 | ❌ | ✅ | ✅ | |
| mTLS | ❌ | ⚠️ | ✅ | |

### Ingress Controller Compatibility

| Controller | v1alpha1 | v1beta1 | v1 | Notes |
|------------|----------|---------|-----|--------|
| NGINX | ✅ | ✅ | ✅ | Recommended |
| Traefik | ⚠️ | ✅ | ✅ | v2.0+ |
| HAProxy | ⚠️ | ✅ | ✅ | |
| Istio | ❌ | ✅ | ✅ | v1.15+ |
| Contour | ❌ | ✅ | ✅ | |
| Ambassador | ❌ | ⚠️ | ✅ | |

### Monitoring Integration Compatibility

| Integration | v1alpha1 | v1beta1 | v1 | Notes |
|-------------|----------|---------|-----|--------|
| Prometheus Federation | ✅ | ✅ | ✅ | |
| Thanos | ❌ | ✅ | ✅ | Sidecar mode |
| Cortex | ⚠️ | ✅ | ✅ | Remote write |
| Mimir | ❌ | ✅ | ✅ | Remote write |
| VictoriaMetrics | ❌ | ⚠️ | ✅ | |
| DataDog | ❌ | ✅ | ✅ | Agent required |
| New Relic | ❌ | ✅ | ✅ | Agent required |

### GitOps Tool Compatibility

| Tool | v1alpha1 | v1beta1 | v1 | Notes |
|------|----------|---------|-----|--------|
| ArgoCD | ⚠️ | ✅ | ✅ | v2.0+ recommended |
| Flux | ⚠️ | ✅ | ✅ | v2 only |
| Jenkins X | ❌ | ⚠️ | ✅ | |
| Spinnaker | ❌ | ⚠️ | ✅ | |
| GitLab | ⚠️ | ✅ | ✅ | Auto DevOps |

### CLI Tool Compatibility

| CLI Version | Operator Versions | Features |
|-------------|------------------|----------|
| 0.1.0 - 0.9.9 | 0.x | Basic CRUD |
| 1.0.0 - 1.4.9 | 1.0 - 1.4 | Migration support |
| 1.5.0 - 1.9.9 | 1.5+ | Full management |
| 2.0.0+ | 2.0+ | Advanced features |

### SDK Compatibility

| SDK | Language | Min Version | Operator Versions |
|-----|----------|-------------|-------------------|
| Go | Go | 1.19+ | All |
| Python | Python | 3.8+ | 1.0+ |
| Java | Java | 11+ | 1.5+ |
| JavaScript | Node.js | 16+ | 1.5+ |
| .NET | C# | 6.0+ | 2.0+ |

## Migration Path Recommendations

### From v0.x to v1.x

1. Upgrade to v0.9.9 first
2. Migrate configurations to v1beta1
3. Upgrade to v1.5.0
4. Test thoroughly
5. Proceed to latest v1.x

### From v1alpha1 to v1beta1

1. Use operator v1.5.0+
2. Run migration tool
3. Test in staging
4. Apply to production
5. Remove v1alpha1 usage

### To Latest Version

Always follow this path:
- Current version → Latest patch → Next minor → Target version

Example: 1.5.0 → 1.5.x → 1.6.0 → 1.9.x

## Feature Deprecation Timeline

| Feature | Deprecated | Removed | Alternative |
|---------|------------|---------|-------------|
| customConfig | v1.0.0 | v2.0.0 | Structured config |
| Basic Auth | v1.5.0 | v2.0.0 | OIDC/SAML |
| v1alpha1 API | v1.0.0 | v2.0.0 | v1beta1 |
| Manual HA | v1.5.0 | v2.0.0 | Automatic HA |
| JSON configs | v1.5.0 | v2.0.0 | YAML only |

## Support Lifecycle

| Version Series | Release Date | End of Support | Support Type |
|----------------|--------------|----------------|--------------|
| 0.x | 2023-01-15 | 2024-01-15 | EOL |
| 1.0 - 1.4 | 2023-07-15 | 2024-07-15 | Security only |
| 1.5 - 1.9 | 2024-01-15 | 2025-01-15 | Full support |
| 2.0+ | 2024-07-15 | 2026-01-15 | Future |

### Support Definitions

- **EOL**: No support, upgrade immediately
- **Security only**: Critical security fixes only
- **Bug fixes**: Security and bug fixes
- **Full support**: All fixes and improvements

## Compatibility Testing

### Test Matrix

We test the following combinations:

| | K8s 1.26 | K8s 1.27 | K8s 1.28 | K8s 1.29 |
|---|----------|----------|----------|----------|
| Operator 1.5 | ✅ | ✅ | ✅ | ✅ |
| Operator 1.9 | ✅ | ✅ | ✅ | ✅ |
| Operator 2.0 | ❌ | ✅ | ✅ | ✅ |

### Component Testing

Each release is tested with:
- Latest stable component versions
- Previous stable versions
- Minimum supported versions

## Known Incompatibilities

### Version-Specific Issues

#### v1.5.0
- Incompatible with Istio < 1.15
- Issues with Longhorn < 1.4
- Requires Kubernetes 1.26+

#### v1.9.0
- Breaking change in Grafana datasource format
- Requires Prometheus 2.40+
- Network policies mandatory

### Workarounds

1. **Istio Compatibility**
   ```yaml
   metadata:
     annotations:
       sidecar.istio.io/inject: "false"
   ```

2. **Longhorn Performance**
   ```yaml
   storage:
     storageClassName: longhorn-ssd
     accessModes:
       - ReadWriteOnce
   ```

## Planning Your Deployment

### New Installations

For new installations, use:
- Latest operator version
- Latest Kubernetes version
- v1beta1 API
- Recommended component versions

### Upgrades

Before upgrading:
1. Check compatibility matrix
2. Review breaking changes
3. Test in non-production
4. Plan rollback strategy
5. Schedule maintenance window

### Version Selection Criteria

Choose versions based on:
- Kubernetes version constraints
- Feature requirements
- Security requirements
- Support lifecycle
- Integration needs

## Contact for Compatibility Issues

If you encounter compatibility issues not listed here:

1. Check [GitHub Issues](https://github.com/gunjanjp/gunj-operator/issues)
2. Join [Slack #gunj-operator](https://kubernetes.slack.com)
3. Email: gunjanjp@gmail.com

---

**Document Version**: 1.0  
**Last Updated**: June 15, 2025  
**Next Review**: July 15, 2025
