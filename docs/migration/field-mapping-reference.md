# Field Mapping Reference

This document provides a complete mapping of fields between different API versions of the ObservabilityPlatform CRD. Use this reference to understand how fields translate between versions during migration.

## Table of Contents

- [Overview](#overview)
- [Field Mapping Format](#field-mapping-format)
- [Core Fields](#core-fields)
- [Component Fields](#component-fields)
- [Security Fields](#security-fields)
- [Monitoring Fields](#monitoring-fields)
- [Advanced Configuration](#advanced-configuration)
- [Annotations and Labels](#annotations-and-labels)
- [Status Fields](#status-fields)
- [Examples](#examples)

## Overview

This reference maps every field from v1alpha1 to v1beta1, showing:
- ✅ **Direct mappings**: Fields that map 1:1
- 🔄 **Transformed mappings**: Fields that require transformation
- ❌ **Removed fields**: Fields with no equivalent
- ➕ **New fields**: Fields only in v1beta1
- ⚠️ **Deprecated fields**: Fields that will be removed

## Field Mapping Format

Each mapping is presented in the following format:

```
v1alpha1 Path → v1beta1 Path
Type: [type change if any]
Transform: [transformation required if any]
Default: [default value changes if any]
Notes: [additional information]
```

## Core Fields

### Metadata Fields

```yaml
# ✅ Direct Mappings
metadata.name → metadata.name
metadata.namespace → metadata.namespace
metadata.labels → metadata.labels
metadata.annotations → metadata.annotations

# 🔄 Transformed Mappings
metadata.clusterName → metadata.labels["observability.io/cluster"]
  Transform: Move to label
  
# ➕ New in v1beta1
→ metadata.finalizers
  Default: ["observability.io/platform-protection"]
→ metadata.ownerReferences
  Notes: Automatically set for created resources
```

### Spec Root Fields

```yaml
# 🔄 Transformed Mappings
spec.globalConfig → spec.global
  Type: object → object
  Notes: Field renamed for consistency

spec.paused → spec.global.paused
  Type: bool → bool
  Transform: Moved under global

# ❌ Removed Fields
spec.experimental → [REMOVED]
  Migration: Features moved to stable sections
spec.legacyMode → [REMOVED]
  Migration: No longer supported

# ➕ New in v1beta1
→ spec.global.externalLabels
  Type: map[string]string
  Default: {}
→ spec.global.logLevel
  Type: string
  Default: "info"
  Values: "debug", "info", "warn", "error"
→ spec.global.monitoring
  Type: object
  Notes: Self-monitoring configuration
```

## Component Fields

### Prometheus Configuration

```yaml
# ✅ Direct Mappings
spec.components.prometheus.enabled → spec.components.prometheus.enabled
spec.components.prometheus.version → spec.components.prometheus.version
spec.components.prometheus.replicas → spec.components.prometheus.replicas
spec.components.prometheus.retention → spec.components.prometheus.retention

# 🔄 Transformed Mappings
spec.components.prometheus.storageSize → spec.components.prometheus.storage.size
  Transform: Nested under storage object
  
spec.components.prometheus.storageClass → spec.components.prometheus.storage.storageClassName
  Transform: Renamed and nested

spec.components.prometheus.cpuRequest → spec.components.prometheus.resources.requests.cpu
spec.components.prometheus.memoryRequest → spec.components.prometheus.resources.requests.memory
spec.components.prometheus.cpuLimit → spec.components.prometheus.resources.limits.cpu
spec.components.prometheus.memoryLimit → spec.components.prometheus.resources.limits.memory
  Transform: Kubernetes-standard resource format

spec.components.prometheus.customConfig → [MULTIPLE FIELDS]
  Transform: Split into specific fields:
    customConfig.externalLabels → spec.components.prometheus.externalLabels
      Type: string (JSON) → map[string]string
    customConfig.remoteWrite → spec.components.prometheus.remoteWrite
      Type: string (YAML) → []RemoteWriteSpec
    customConfig.scrapeConfigs → spec.components.prometheus.additionalScrapeConfigs
      Type: string (YAML) → []ScrapeConfig

# ❌ Removed Fields  
spec.components.prometheus.legacyConfig → [REMOVED]
spec.components.prometheus.customRules → [REMOVED]
  Migration: Use separate PrometheusRule CRD

# ➕ New in v1beta1
→ spec.components.prometheus.serviceMonitorSelector
  Type: *metav1.LabelSelector
  Default: nil
  Notes: Selects ServiceMonitors to include

→ spec.components.prometheus.podMonitorSelector
  Type: *metav1.LabelSelector
  Default: nil

→ spec.components.prometheus.probeSelector
  Type: *metav1.LabelSelector
  Default: nil

→ spec.components.prometheus.ruleSelector
  Type: *metav1.LabelSelector
  Default: nil

→ spec.components.prometheus.tsdb
  Type: TSDBSpec
  Notes: TSDB-specific configuration

→ spec.components.prometheus.walCompression
  Type: bool
  Default: true

→ spec.components.prometheus.exemplars
  Type: *ExemplarSpec
  Notes: Exemplar storage configuration
```

### Grafana Configuration

```yaml
# ✅ Direct Mappings
spec.components.grafana.enabled → spec.components.grafana.enabled
spec.components.grafana.version → spec.components.grafana.version
spec.components.grafana.replicas → spec.components.grafana.replicas

# 🔄 Transformed Mappings
spec.components.grafana.adminPassword → spec.components.grafana.security.adminPassword
  Transform: Moved under security

spec.components.grafana.dashboards → spec.components.grafana.dashboards.configMapSelector
  Type: string → metav1.LabelSelector
  Transform: Path-based → Selector-based

spec.components.grafana.datasources → spec.components.grafana.datasources
  Type: string (YAML) → []DataSourceSpec
  Transform: String to structured array

spec.components.grafana.plugins → spec.components.grafana.plugins
  Type: string (comma-separated) → []string
  Transform: Split string to array

spec.components.grafana.ingressEnabled → spec.components.grafana.ingress.enabled
spec.components.grafana.ingressHost → spec.components.grafana.ingress.host
spec.components.grafana.ingressTLS → spec.components.grafana.ingress.tls.enabled
  Transform: Flattened → Nested structure

# ➕ New in v1beta1
→ spec.components.grafana.security.secretName
  Type: string
  Notes: External secret for admin credentials

→ spec.components.grafana.persistence
  Type: PersistenceSpec
  Default: {enabled: true, size: "10Gi"}

→ spec.components.grafana.sidecar
  Type: SidecarSpec
  Notes: Dashboard/datasource sidecar config

→ spec.components.grafana.smtp
  Type: SMTPSpec
  Notes: Email notification configuration

→ spec.components.grafana.ldap
  Type: LDAPSpec
  Notes: LDAP authentication config
```

### Loki Configuration

```yaml
# ✅ Direct Mappings
spec.components.loki.enabled → spec.components.loki.enabled
spec.components.loki.version → spec.components.loki.version

# 🔄 Transformed Mappings
spec.components.loki.storage → spec.components.loki.storage
  Type: string → StorageSpec
  Transform: 
    "s3" → {type: "s3", s3: {...}}
    "gcs" → {type: "gcs", gcs: {...}}
    "filesystem" → {type: "filesystem", filesystem: {...}}

spec.components.loki.retentionDays → spec.components.loki.limits.retentionPeriod
  Type: int → string
  Transform: days → duration (e.g., 7 → "168h")

# ➕ New in v1beta1
→ spec.components.loki.compactor
  Type: CompactorSpec
  Notes: Compaction configuration

→ spec.components.loki.querier
  Type: QuerierSpec
  Notes: Query performance tuning

→ spec.components.loki.ingester
  Type: IngesterSpec
  Notes: Ingestion configuration

→ spec.components.loki.limits
  Type: LimitsSpec
  Notes: Per-tenant limits

→ spec.components.loki.ruler
  Type: RulerSpec
  Notes: Alerting rule evaluation
```

### Tempo Configuration

```yaml
# ✅ Direct Mappings
spec.components.tempo.enabled → spec.components.tempo.enabled
spec.components.tempo.version → spec.components.tempo.version

# 🔄 Transformed Mappings
spec.components.tempo.storage → spec.components.tempo.storage
  Notes: Similar to Loki storage transformation

# ➕ New in v1beta1
→ spec.components.tempo.search
  Type: SearchSpec
  Default: {enabled: true}
  Notes: Search capabilities

→ spec.components.tempo.metrics
  Type: MetricsGeneratorSpec
  Notes: Metrics generation from traces

→ spec.components.tempo.multiTenancy
  Type: MultiTenancySpec
  Default: {enabled: false}
```

## Security Fields

```yaml
# ➕ All Security Fields are New in v1beta1
→ spec.security.tls.enabled
  Type: bool
  Default: true

→ spec.security.tls.certManager
  Type: CertManagerSpec
  Notes: Auto-generate certificates

→ spec.security.tls.certificates
  Type: map[string]CertificateSpec
  Notes: Manual certificate configuration

→ spec.security.podSecurityPolicy
  Type: bool
  Default: true

→ spec.security.networkPolicy
  Type: bool
  Default: true

→ spec.security.rbac.create
  Type: bool
  Default: true

→ spec.security.rbac.rules
  Type: []rbacv1.PolicyRule
  Notes: Additional RBAC rules

→ spec.security.serviceAccounts
  Type: map[string]ServiceAccountSpec
  Notes: Per-component service accounts

→ spec.security.securityContext
  Type: *corev1.SecurityContext
  Default: {runAsNonRoot: true, fsGroup: 65534}
```

## Monitoring Fields

```yaml
# ➕ All Monitoring Fields are New in v1beta1
→ spec.monitoring.selfMonitoring
  Type: bool
  Default: true

→ spec.monitoring.serviceMonitor.enabled
  Type: bool
  Default: true

→ spec.monitoring.prometheusRule.enabled
  Type: bool
  Default: true

→ spec.monitoring.grafanaDashboard.enabled
  Type: bool
  Default: true

→ spec.monitoring.metrics.platform
  Type: bool
  Default: true

→ spec.monitoring.metrics.components
  Type: bool
  Default: true

→ spec.monitoring.alerts
  Type: []AlertSpec
  Notes: Platform-level alerts
```

## Advanced Configuration

### Cost Optimization

```yaml
# ➕ All Cost Optimization Fields are New in v1beta1
→ spec.costOptimization.enabled
  Type: bool
  Default: false

→ spec.costOptimization.recommendations
  Type: bool
  Default: true

→ spec.costOptimization.autoScale.enabled
  Type: bool
  Default: false

→ spec.costOptimization.autoScale.minReplicas
  Type: int32
  Default: 1

→ spec.costOptimization.autoScale.maxReplicas
  Type: int32
  Default: 10

→ spec.costOptimization.resourceQuotas
  Type: ResourceQuotaSpec
  Notes: Enforce resource limits
```

### Service Mesh Integration

```yaml
# ➕ All Service Mesh Fields are New in v1beta1
→ spec.serviceMesh.enabled
  Type: bool
  Default: false

→ spec.serviceMesh.provider
  Type: string
  Values: "istio", "linkerd", "consul"

→ spec.serviceMesh.mtls.mode
  Type: string
  Default: "STRICT"
  Values: "DISABLE", "PERMISSIVE", "STRICT"

→ spec.serviceMesh.tracing.enabled
  Type: bool
  Default: true

→ spec.serviceMesh.injection.enabled
  Type: bool
  Default: true
```

### Multi-Cluster Configuration

```yaml
# ➕ All Multi-Cluster Fields are New in v1beta1
→ spec.multiCluster.enabled
  Type: bool
  Default: false

→ spec.multiCluster.mode
  Type: string
  Values: "primary", "remote"

→ spec.multiCluster.primary.endpoint
  Type: string
  Notes: Primary cluster API endpoint

→ spec.multiCluster.federation.enabled
  Type: bool
  Default: false

→ spec.multiCluster.federation.clusters
  Type: []ClusterSpec
  Notes: List of federated clusters
```

## Annotations and Labels

### Standard Annotations

```yaml
# 🔄 Transformed Annotations
observability.io/reconcile-period → observability.io/reconcile-frequency
  Transform: Rename for clarity

observability.io/skip-reconcile → observability.io/paused
  Type: "true"/"false" → "true"/"false"
  Transform: Rename for consistency

# ➕ New Annotations
→ observability.io/force-upgrade
  Values: "true"/"false"
  Notes: Force component upgrade

→ observability.io/backup-before-upgrade
  Values: "true"/"false"
  Default: "true"

→ observability.io/migration-version
  Notes: Track migration source version

→ observability.io/cost-center
  Notes: Cost allocation tracking

→ observability.io/data-classification
  Values: "public", "internal", "confidential", "restricted"
```

### Standard Labels

```yaml
# ✅ Direct Mappings
app.kubernetes.io/name → app.kubernetes.io/name
app.kubernetes.io/instance → app.kubernetes.io/instance
app.kubernetes.io/version → app.kubernetes.io/version
app.kubernetes.io/component → app.kubernetes.io/component
app.kubernetes.io/part-of → app.kubernetes.io/part-of
app.kubernetes.io/managed-by → app.kubernetes.io/managed-by

# ➕ New Labels
→ observability.io/platform
  Notes: Platform name reference
→ observability.io/component
  Notes: Component type
→ observability.io/tier
  Values: "platform", "monitoring", "logging", "tracing"
```

## Status Fields

### Status Root Fields

```yaml
# ✅ Direct Mappings
status.phase → status.phase
status.message → status.message
status.conditions → status.conditions

# 🔄 Transformed Mappings
status.lastUpdated → status.lastTransitionTime
  Type: string → metav1.Time
  Transform: Parse to Kubernetes time

status.ready → [REMOVED]
  Migration: Check phase == "Ready"

# ➕ New Status Fields
→ status.observedGeneration
  Type: int64
  Notes: Last processed generation

→ status.componentsReady
  Type: int32
  Notes: Number of ready components

→ status.componentsTotal
  Type: int32
  Notes: Total number of components

→ status.healthStatus
  Type: string
  Values: "Healthy", "Degraded", "Unhealthy"

→ status.endpoints
  Type: map[string]string
  Notes: Component endpoint URLs

→ status.version
  Type: string
  Notes: Currently deployed version

→ status.operatorVersion
  Type: string
  Notes: Operator version managing platform
```

### Component Status Fields

```yaml
# ➕ New Component Status Structure
→ status.componentStatus.<component>.phase
  Type: string
  Values: "Pending", "Installing", "Ready", "Failed", "Upgrading"

→ status.componentStatus.<component>.ready
  Type: bool
  Notes: Simple ready check

→ status.componentStatus.<component>.version
  Type: string
  Notes: Deployed version

→ status.componentStatus.<component>.lastTransitionTime
  Type: metav1.Time
  Notes: Last status change

→ status.componentStatus.<component>.message
  Type: string
  Notes: Human-readable status

→ status.componentStatus.<component>.replicas
  Type: int32
  Notes: Current replica count

→ status.componentStatus.<component>.readyReplicas
  Type: int32
  Notes: Number of ready replicas

→ status.componentStatus.<component>.endpoints
  Type: []string
  Notes: Component endpoints
```

## Examples

### Complete v1alpha1 to v1beta1 Transformation

#### v1alpha1 Example

```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: production
  namespace: monitoring
  clusterName: prod-cluster
spec:
  paused: false
  globalConfig:
    region: us-east-1
  components:
    prometheus:
      enabled: true
      version: v2.45.0
      replicas: 2
      storageSize: 100Gi
      storageClass: fast-ssd
      retention: 30
      cpuRequest: "2"
      memoryRequest: "8Gi"
      cpuLimit: "4"
      memoryLimit: "16Gi"
      customConfig:
        externalLabels: '{"cluster": "prod", "region": "us-east-1"}'
        remoteWrite: |
          - url: http://cortex:9009/api/v1/push
        scrapeConfigs: |
          - job_name: custom-app
            static_configs:
              - targets: ['app:8080']
    grafana:
      enabled: true
      version: 9.5.0
      adminPassword: secretPassword
      dashboards: "/etc/dashboards"
      datasources: |
        - name: Prometheus
          type: prometheus
          url: http://prometheus:9090
      plugins: "grafana-piechart-panel,grafana-clock-panel"
      ingressEnabled: true
      ingressHost: grafana.example.com
      ingressTLS: true
```

#### v1beta1 Transformation

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
  namespace: monitoring
  labels:
    observability.io/cluster: prod-cluster  # Moved from clusterName
  finalizers:
    - observability.io/platform-protection  # New: Added automatically
spec:
  global:  # Renamed from globalConfig
    paused: false  # Moved here
    externalLabels:  # New location for global labels
      region: us-east-1
    logLevel: info  # New field
  components:
    prometheus:
      enabled: true
      version: v2.45.0
      replicas: 2
      storage:  # Restructured
        size: 100Gi
        storageClassName: fast-ssd  # Renamed
      retention: 30d  # Format change: add 'd' suffix
      resources:  # Kubernetes-standard format
        requests:
          cpu: "2"
          memory: "8Gi"
        limits:
          cpu: "4"
          memory: "16Gi"
      externalLabels:  # Moved from customConfig, type changed
        cluster: prod
        region: us-east-1
      remoteWrite:  # Moved from customConfig, structured
        - url: http://cortex:9009/api/v1/push
      additionalScrapeConfigs:  # Renamed from scrapeConfigs
        - job_name: custom-app
          static_configs:
            - targets: ['app:8080']
      # New fields with defaults
      serviceMonitorSelector: {}
      walCompression: true
    grafana:
      enabled: true
      version: 9.5.0
      security:  # New structure
        adminPassword: secretPassword
      dashboards:  # Changed to selector-based
        configMapSelector:
          matchLabels:
            grafana_dashboard: "1"
      datasources:  # Structured array instead of YAML string
        - name: Prometheus
          type: prometheus
          url: http://prometheus:9090
      plugins:  # Array instead of comma-separated
        - grafana-piechart-panel
        - grafana-clock-panel
      ingress:  # Restructured
        enabled: true
        host: grafana.example.com
        tls:
          enabled: true
          secretName: grafana-tls  # New: Explicit secret
      # New fields
      persistence:
        enabled: true
        size: 10Gi
      sidecar:
        dashboards:
          enabled: true
  # New top-level sections
  security:
    tls:
      enabled: true
      certManager:
        enabled: true
    podSecurityPolicy: true
    networkPolicy: true
    rbac:
      create: true
  monitoring:
    selfMonitoring: true
    serviceMonitor:
      enabled: true
  costOptimization:
    enabled: true
    recommendations: true
status:
  # Enhanced status structure
  phase: Ready
  message: "All components are running"
  observedGeneration: 5  # New
  lastTransitionTime: "2024-01-15T10:00:00Z"  # Renamed
  componentsReady: 2  # New
  componentsTotal: 2  # New
  healthStatus: Healthy  # New
  componentStatus:  # New detailed status
    prometheus:
      phase: Ready
      ready: true
      version: v2.45.0
      replicas: 2
      readyReplicas: 2
    grafana:
      phase: Ready
      ready: true
      version: 9.5.0
      replicas: 1
      readyReplicas: 1
```

### Field Transformation Script

```bash
#!/bin/bash
# transform-fields.sh - Helper script for field transformation

# Function to transform customConfig
transform_custom_config() {
    local input=$1
    
    # Extract externalLabels
    external_labels=$(echo "$input" | yq e '.externalLabels' - | yq e '. | from_json' -)
    
    # Extract remoteWrite
    remote_write=$(echo "$input" | yq e '.remoteWrite' - | yq e '.' -)
    
    # Extract scrapeConfigs
    scrape_configs=$(echo "$input" | yq e '.scrapeConfigs' - | yq e '.' -)
    
    # Output transformed structure
    cat <<EOF
externalLabels: $external_labels
remoteWrite: $remote_write
additionalScrapeConfigs: $scrape_configs
EOF
}

# Transform storage configuration
transform_storage() {
    local size=$1
    local class=$2
    
    cat <<EOF
storage:
  size: $size
  storageClassName: $class
EOF
}

# Transform resources
transform_resources() {
    local cpu_req=$1
    local mem_req=$2
    local cpu_limit=$3
    local mem_limit=$4
    
    cat <<EOF
resources:
  requests:
    cpu: "$cpu_req"
    memory: "$mem_req"
  limits:
    cpu: "$cpu_limit"
    memory: "$mem_limit"
EOF
}

# Main transformation
yq eval '
  .apiVersion = "observability.io/v1beta1" |
  .metadata.labels["observability.io/cluster"] = .metadata.clusterName |
  del(.metadata.clusterName) |
  .spec.global = .spec.globalConfig |
  del(.spec.globalConfig) |
  .spec.global.paused = .spec.paused |
  del(.spec.paused)
' "$1"
```

## Best Practices

1. **Use the Mapping Reference**
   - Keep this document handy during migrations
   - Check for all affected fields before starting
   - Plan transformations for complex fields

2. **Test Transformations**
   - Always test field mappings in development first
   - Validate the transformed YAML
   - Check that no data is lost

3. **Automate Where Possible**
   - Use transformation scripts for large deployments
   - Validate results programmatically
   - Keep transformation logic versioned

4. **Document Custom Fields**
   - If you use custom annotations/labels, document them
   - Plan how custom fields will migrate
   - Consider if custom fields should become standard

## Related Documentation

- [Breaking Changes Guide](breaking-changes.md)
- [Migration Tools](migration-tools.md)
- [API Reference](../api-reference.md)
- [Version Compatibility Matrix](version-compatibility-matrix.md)
