# Helm Integration for Component Managers

## Overview

The Gunj Operator supports both native Kubernetes resource management and Helm-based deployment for observability components. This document describes the Helm integration feature added in micro-task MT 2.2.2.5.

## Architecture

### Component Overview

1. **HelmClient**: Provides core Helm operations (install, upgrade, uninstall, rollback)
2. **ValueBuilder**: Builds and validates Helm values for each component
3. **Repository**: Manages Helm chart repositories
4. **VersionManager**: Handles version compatibility and upgrade paths
5. **BaseHelmManager**: Common functionality for all Helm-based component managers

### Manager Mode

The operator can run in two modes:
- **Native Mode**: Direct Kubernetes resource management (default for upgrades)
- **Helm Mode**: Uses Helm charts for component deployment (default for new installations)

The mode is controlled by the `GUNJ_MANAGER_MODE` environment variable:
```bash
# Use Helm mode
export GUNJ_MANAGER_MODE=helm

# Use native mode
export GUNJ_MANAGER_MODE=native
```

## Component Managers

Each component (Prometheus, Grafana, Loki, Tempo) has two implementations:
1. Native manager (e.g., `prometheus_manager.go`)
2. Helm manager (e.g., `prometheus_manager_helm.go`)

### Prometheus Helm Manager

Uses the `prometheus-community/prometheus` chart with customized values:
- Configurable replicas and resources
- Persistent storage configuration
- Remote write support
- External labels
- Custom scrape configurations

### Grafana Helm Manager

Uses the `grafana/grafana` chart with:
- Automatic datasource configuration
- Dashboard sidecar for dynamic loading
- Ingress support
- Custom plugins

### Loki Helm Manager

Uses the `grafana/loki` chart with:
- Single binary or distributed mode
- S3 backend support
- Retention policies
- Compactor configuration

### Tempo Helm Manager

Uses the `grafana/tempo` chart with:
- Multiple trace receivers (Jaeger, Zipkin, OTLP)
- Metrics generator
- S3 backend support
- Search capabilities

## Configuration

### Platform Spec Example

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
      version: v2.48.0  # Helm chart version
      replicas: 3
      resources:
        requests:
          memory: "4Gi"
          cpu: "1"
      storage:
        size: 100Gi
        storageClassName: fast-ssd
      config:  # Custom Helm values override
        server:
          extraFlags:
            - web.enable-lifecycle
```

### Value Building

The `ValueBuilder` translates the ObservabilityPlatform spec into Helm values:

1. **Base Values**: Default values for each component
2. **Spec Translation**: Maps CRD fields to Helm values
3. **Custom Config**: Allows direct Helm value overrides
4. **Global Settings**: Applies platform-wide configurations

## Features

### Version Management

- Automatic latest version detection
- Version compatibility checking
- Multi-step upgrade paths for major versions
- Rollback support

### Repository Management

- Automatic repository configuration
- Support for custom repositories
- Repository index caching
- Chart search capabilities

### Lifecycle Management

- Install with custom values
- Upgrade with version compatibility checks
- Rollback to previous versions
- Uninstall with cleanup
- Helm test execution

### Multi-tenancy

- Namespace isolation
- Resource naming conventions
- RBAC configuration per platform

## Usage

### Enabling Helm Mode

1. Set environment variable:
   ```bash
   kubectl set env deployment/gunj-operator -n gunj-system GUNJ_MANAGER_MODE=helm
   ```

2. Or update the deployment:
   ```yaml
   env:
   - name: GUNJ_MANAGER_MODE
     value: "helm"
   ```

### Custom Values

Provide custom Helm values through the `config` field:

```yaml
spec:
  components:
    prometheus:
      config:
        server:
          persistentVolume:
            size: 200Gi
        nodeExporter:
          enabled: true
```

### Helm Hooks

The operator respects Helm hooks for:
- Pre-install/post-install
- Pre-upgrade/post-upgrade
- Pre-delete/post-delete
- Test hooks

### Rollback

Rollback is supported through the API or by updating the version in the spec:

```yaml
# Rollback to specific version
spec:
  components:
    prometheus:
      version: v2.47.0  # Previous version
```

## Benefits

1. **Ecosystem Compatibility**: Use community Helm charts
2. **Version Management**: Built-in versioning and upgrades
3. **Configuration Flexibility**: Full Helm values support
4. **Rollback Support**: Native Helm rollback capabilities
5. **Test Integration**: Helm test execution
6. **Community Updates**: Automatic chart updates

## Migration

### From Native to Helm

1. Backup existing data
2. Update operator to Helm mode
3. Apply platform with migration annotations:
   ```yaml
   annotations:
     observability.io/migration-mode: "true"
   ```

### From Helm to Native

1. Export current Helm values
2. Update operator to Native mode
3. Apply platform configuration
4. Verify resource creation

## Troubleshooting

### Common Issues

1. **Chart Not Found**
   - Ensure repositories are configured
   - Check chart name and version

2. **Version Compatibility**
   - Check upgrade path for major versions
   - Review breaking changes

3. **Resource Conflicts**
   - Check for existing resources
   - Verify namespace permissions

### Debug Commands

```bash
# Check Helm releases
kubectl exec -n gunj-system deployment/gunj-operator -- helm list -A

# Get release values
kubectl exec -n gunj-system deployment/gunj-operator -- helm get values RELEASE_NAME -n NAMESPACE

# Check release history
kubectl exec -n gunj-system deployment/gunj-operator -- helm history RELEASE_NAME -n NAMESPACE
```

## Future Enhancements

1. **OCI Registry Support**: Use OCI-based chart repositories
2. **Chart Caching**: Local chart caching for offline support
3. **Custom Charts**: Support for custom/private charts
4. **Helm Plugins**: Integration with Helm plugins
5. **GitOps Integration**: Better ArgoCD/Flux support
