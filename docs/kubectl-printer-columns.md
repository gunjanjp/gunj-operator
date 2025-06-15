# Kubectl Printer Columns for ObservabilityPlatform

This document explains the printer columns implementation for the ObservabilityPlatform CRD, which customizes the output of `kubectl get` commands.

## Overview

Printer columns allow users to see important information about ObservabilityPlatform resources at a glance without needing to use `-o wide` or inspect the full YAML/JSON output.

## Column Definitions

### v1beta1 Columns

The v1beta1 API version includes comprehensive printer columns:

| Column | Type | JSONPath | Priority | Description |
|--------|------|----------|----------|-------------|
| Phase | string | `.status.phase` | 0 | Current phase of the platform (Pending, Installing, Ready, Failed, Upgrading, Degraded) |
| Health | string | `.status.health.status` | 0 | Overall health status (Healthy, Degraded, Unhealthy, Unknown) |
| Components | string | `.status.health.healthyCount` | 1 | Number of healthy components |
| Version | string | `.status.version` | 1 | Platform version |
| Prometheus | string | `.status.componentStatus.prometheus.phase` | 1 | Prometheus component status |
| Grafana | string | `.status.componentStatus.grafana.phase` | 1 | Grafana component status |
| Loki | string | `.status.componentStatus.loki.phase` | 1 | Loki component status |
| Tempo | string | `.status.componentStatus.tempo.phase` | 1 | Tempo component status |
| Message | string | `.status.message` | 1 | Descriptive status message |
| Age | date | `.metadata.creationTimestamp` | 0 | Time since creation |

### v1alpha1 Columns

The v1alpha1 API version includes essential printer columns:

| Column | Type | JSONPath | Priority | Description |
|--------|------|----------|----------|-------------|
| Phase | string | `.status.phase` | 0 | Current phase of the platform |
| Ready | string | `.status.conditions[?(@.type=='Ready')].status` | 1 | Ready condition status |
| Prometheus | string | `.status.componentStatus.prometheus.phase` | 1 | Prometheus component status |
| Grafana | string | `.status.componentStatus.grafana.phase` | 1 | Grafana component status |
| Message | string | `.status.message` | 1 | Status message |
| Age | date | `.metadata.creationTimestamp` | 0 | Time since creation |

## Priority Levels

- **Priority 0** (default): Always shown in standard `kubectl get` output
- **Priority 1**: Only shown when using `kubectl get -o wide`

## Usage Examples

### Basic Output

```bash
$ kubectl get observabilityplatforms
NAME                 PHASE        HEALTH     AGE
production-full      Ready        Healthy    5d
staging-installing   Installing   Unknown    2m
dev-degraded        Degraded     Degraded   1d
qa-upgrading        Upgrading    Degraded   10m
```

### Wide Output

```bash
$ kubectl get observabilityplatforms -o wide
NAME                 PHASE        HEALTH     COMPONENTS   VERSION   PROMETHEUS   GRAFANA   LOKI      TEMPO     MESSAGE                                        AGE
production-full      Ready        Healthy    4            v2.0.0    Ready        Ready     Ready     Ready     All components are running successfully        5d
staging-installing   Installing   Unknown    0                      Deploying    Pending                       Installing Prometheus and Grafana components   2m
dev-degraded        Degraded     Degraded   2            v2.0.0    Ready        Ready     Failed              Loki component failed to start: insufficient memory   1d
qa-upgrading        Upgrading    Degraded   1            v2.0.0    Upgrading    Ready                         Upgrading components to new versions           10m
```

### Namespace-specific Output

```bash
$ kubectl get op -n monitoring
NAME              PHASE   HEALTH    AGE
production-full   Ready   Healthy   5d
```

### Watching Changes

```bash
$ kubectl get op -w
NAME                 PHASE        HEALTH     AGE
staging-installing   Installing   Unknown    2m
staging-installing   Installing   Unknown    3m
staging-installing   Ready        Healthy    4m
```

## Resource Categories

The CRD is also tagged with categories, allowing grouped queries:

```bash
# Get all observability resources
$ kubectl get observability

# Get all platform resources
$ kubectl get platform
```

## Implementation Notes

1. **JSONPath Expressions**: The printer columns use JSONPath to extract values from the resource status.

2. **Component Status**: Individual component statuses (Prometheus, Grafana, etc.) are shown only in wide output to avoid cluttering the default view.

3. **Health Count**: The Components column shows the count of healthy components as a quick indicator of platform health.

4. **Graceful Degradation**: If a field doesn't exist (e.g., a component is not enabled), the column will be empty rather than causing an error.

5. **Backward Compatibility**: v1alpha1 maintains a simpler set of columns appropriate for its reduced feature set.

## Best Practices

1. **Regular Monitoring**: Use `kubectl get op -w` to monitor platform status during deployments or upgrades.

2. **Troubleshooting**: The Message column provides quick insights into issues without needing to describe the resource.

3. **Scripting**: The consistent column output makes it easy to script monitoring and automation:
   ```bash
   kubectl get op -o custom-columns=NAME:.metadata.name,PHASE:.status.phase,HEALTH:.status.health.status
   ```

4. **Filtering**: Use field selectors to filter by status:
   ```bash
   kubectl get op --field-selector status.phase=Failed
   ```

## Customization

While the default printer columns are defined in the CRD, users can always customize the output using:

- `kubectl get op -o custom-columns=...`
- `kubectl get op -o jsonpath=...`
- `kubectl get op -o go-template=...`

This provides flexibility while maintaining sensible defaults for most use cases.
