# Migration Examples and Code Snippets

## Table of Contents

- [Overview](#overview)
- [Basic Migration Examples](#basic-migration-examples)
- [Component-Specific Examples](#component-specific-examples)
- [Advanced Migration Scenarios](#advanced-migration-scenarios)
- [GitOps Migration Examples](#gitops-migration-examples)
- [Multi-Cluster Migration](#multi-cluster-migration)
- [Custom Configuration Migration](#custom-configuration-migration)
- [Automation Scripts](#automation-scripts)
- [Common Patterns](#common-patterns)

## Overview

This document provides practical examples of migrating ObservabilityPlatform configurations from v1alpha1 to v1beta1. Each example shows the before and after state with explanations.

## Basic Migration Examples

### Example 1: Minimal Platform Configuration

**Scenario**: Basic Prometheus-only deployment

**v1alpha1 Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: monitoring-basic
  namespace: default
spec:
  components:
    prometheus:
      enabled: true
      version: "2.45.0"
```

**v1beta1 Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: monitoring-basic
  namespace: default
  annotations:
    observability.io/migrated-from: "v1alpha1"
    observability.io/migration-date: "2024-06-15"
spec:
  # New global section (required in v1beta1)
  global:
    logLevel: info
    logFormat: json
  
  components:
    prometheus:
      enabled: true
      version: "v2.48.0"  # Note: 'v' prefix required
      
      # New default configurations in v1beta1
      serviceMonitor:
        enabled: true
        interval: 30s
      
      # Default HA configuration
      replicas: 2
      highAvailability:
        enabled: true
```

### Example 2: Resource Specifications

**Scenario**: Platform with specific resource requirements

**v1alpha1 Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: resource-example
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      cpuRequest: "500m"
      cpuLimit: "2000m"
      memoryRequest: "2Gi"
      memoryLimit: "8Gi"
    grafana:
      enabled: true
      cpuRequest: "100m"
      memoryRequest: "128Mi"
```

**v1beta1 Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: resource-example
  namespace: monitoring
spec:
  global:
    logLevel: info
    logFormat: json
    
  components:
    prometheus:
      enabled: true
      version: "v2.48.0"
      
      # Kubernetes standard resource format
      resources:
        requests:
          cpu: "500m"
          memory: "2Gi"
        limits:
          cpu: "2"      # Simplified from "2000m"
          memory: "8Gi"
      
      # QoS class (new in v1beta1)
      qosClass: Burstable
      
    grafana:
      enabled: true
      version: "10.2.0"
      
      resources:
        requests:
          cpu: "100m"
          memory: "128Mi"
        limits:
          cpu: "500m"   # Good practice to set limits
          memory: "512Mi"
```

### Example 3: Storage Configuration

**Scenario**: Persistent storage with retention policies

**v1alpha1 Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: storage-example
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      storageSize: 100Gi
      storageClass: fast-ssd
      retentionDays: 30
    loki:
      enabled: true
      storageSize: 200Gi
      retentionDays: 7
```

**v1beta1 Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: storage-example
  namespace: monitoring
spec:
  global:
    logLevel: info
    logFormat: json
    
  components:
    prometheus:
      enabled: true
      version: "v2.48.0"
      
      # Nested storage configuration
      storage:
        size: 100Gi
        storageClassName: fast-ssd
        # New options in v1beta1
        volumeMode: Filesystem
        accessModes:
          - ReadWriteOnce
        selector:
          matchLabels:
            disk-type: ssd
      
      # Duration format for retention
      retention: 30d
      # New: retention by size
      retentionSize: 90GB
      
    loki:
      enabled: true
      version: "2.9.0"
      
      storage:
        size: 200Gi
        storageClassName: fast-ssd
        
      # Loki-specific retention under limits
      limits:
        retention: 7d
        # New options
        ingestionRate: 10MB
        ingestionBurstSize: 20MB
```

## Component-Specific Examples

### Prometheus Migration

**v1alpha1 Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: prometheus-complex
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      version: "2.45.0"
      customConfig: |
        global:
          scrape_interval: 15s
          evaluation_interval: 15s
          external_labels:
            cluster: 'production'
            region: 'us-east-1'
        
        alerting:
          alertmanagers:
            - static_configs:
              - targets: ['alertmanager:9093']
        
        rule_files:
          - '/etc/prometheus/rules/*.yaml'
        
        scrape_configs:
          - job_name: 'kubernetes-apiservers'
            kubernetes_sd_configs:
              - role: endpoints
            scheme: https
            tls_config:
              ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
            bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
            relabel_configs:
              - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
                action: keep
                regex: default;kubernetes;https
          
          - job_name: 'custom-app'
            static_configs:
              - targets: ['app-1:8080', 'app-2:8080']
            metrics_path: '/metrics'
            scrape_interval: 30s
        
        remote_write:
          - url: "http://cortex:9009/api/v1/push"
            queue_config:
              capacity: 10000
              max_shards: 30
      
      storageSize: 500Gi
      retentionDays: 90
      cpuRequest: "4"
      memoryRequest: "16Gi"
```

**v1beta1 Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: prometheus-complex
  namespace: monitoring
spec:
  global:
    logLevel: info
    logFormat: json
    
  components:
    prometheus:
      enabled: true
      version: "v2.48.0"
      
      # Structured global configuration
      global:
        scrapeInterval: 15s
        evaluationInterval: 15s
        queryTimeout: 2m
        
      # External labels as map
      externalLabels:
        cluster: production
        region: us-east-1
        
      # Alerting configuration
      alerting:
        alertmanagers:
          - namespace: monitoring
            name: alertmanager
            port: 9093
            scheme: http
            pathPrefix: /
            timeout: 10s
            
      # Rule files configuration
      ruleFiles:
        configMapSelector:
          matchLabels:
            prometheus_rule: "true"
        # Or specific ConfigMaps
        configMaps:
          - name: prometheus-rules
            key: rules.yaml
            
      # Service discovery configuration
      serviceDiscoveryConfigs:
        kubernetes:
          - role: endpoints
            namespaces:
              names: []  # All namespaces
          - role: pod
            namespaces:
              names: ["default", "monitoring"]
          - role: service
          - role: node
            
      # Additional scrape configs
      additionalScrapeConfigs:
        - jobName: kubernetes-apiservers
          kubernetesSDConfigs:
            - role: endpoints
          scheme: https
          tlsConfig:
            caFile: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
            insecureSkipVerify: false
          bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
          relabelConfigs:
            - sourceLabels: 
                - __meta_kubernetes_namespace
                - __meta_kubernetes_service_name
                - __meta_kubernetes_endpoint_port_name
              action: keep
              regex: default;kubernetes;https
              
        - jobName: custom-app
          staticConfigs:
            - targets:
                - app-1:8080
                - app-2:8080
              labels:
                app: custom
          metricsPath: /metrics
          scrapeInterval: 30s
          scrapeTimeout: 10s
          
      # Remote write configuration
      remoteWrite:
        - url: http://cortex:9009/api/v1/push
          name: cortex
          remoteTimeout: 30s
          queueConfig:
            capacity: 10000
            maxShards: 30
            minShards: 1
            maxSamplesPerSend: 3000
            batchSendDeadline: 5s
            minBackoff: 30ms
            maxBackoff: 100ms
          metadata:
            send: true
            sendInterval: 1m
          headers:
            X-Scope-OrgID: "production"
            
      # Storage configuration
      storage:
        size: 500Gi
        storageClassName: fast-ssd
        volumeMode: Filesystem
        accessModes:
          - ReadWriteOnce
          
      # Retention configuration
      retention: 90d
      retentionSize: 450GB
      
      # Resource configuration
      resources:
        requests:
          cpu: "4"
          memory: "16Gi"
        limits:
          cpu: "8"
          memory: "32Gi"
          
      # High availability
      replicas: 3
      highAvailability:
        enabled: true
        replicationFactor: 2
        
      # Additional configurations
      nodeSelector:
        node-type: monitoring
      tolerations:
        - key: monitoring
          operator: Equal
          value: "true"
          effect: NoSchedule
      
      # Security context
      securityContext:
        runAsUser: 65534
        runAsNonRoot: true
        fsGroup: 65534
```

### Grafana Migration

**v1alpha1 Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: grafana-example
  namespace: monitoring
spec:
  components:
    grafana:
      enabled: true
      version: "9.5.0"
      adminPassword: "admin123"
      anonymousAccess: true
      ingressEnabled: true
      ingressHost: grafana.example.com
      ingressTLS: true
      plugins:
        - grafana-clock-panel
        - grafana-piechart-panel
      dashboards: |
        - name: kubernetes
          url: https://grafana.com/api/dashboards/12114/revisions/1/download
        - name: prometheus
          url: https://grafana.com/api/dashboards/3662/revisions/2/download
```

**v1beta1 Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: grafana-example
  namespace: monitoring
spec:
  global:
    logLevel: info
    logFormat: json
    
  components:
    grafana:
      enabled: true
      version: "10.2.0"
      
      # Authentication configuration
      auth:
        # Use secret reference instead of plain text
        adminPasswordSecret:
          name: grafana-admin-secret
          key: password
        anonymous:
          enabled: false  # Secure default
          role: Viewer
        basic:
          enabled: true
        oauth:
          enabled: true
          providers:
            - name: google
              clientId: google-client-id
              clientSecret:
                name: grafana-oauth-secret
                key: google-client-secret
              scopes: ["openid", "profile", "email"]
              
      # Ingress configuration
      ingress:
        enabled: true
        host: grafana.example.com
        className: nginx
        tls:
          enabled: true
          secretName: grafana-tls-cert
        annotations:
          cert-manager.io/cluster-issuer: letsencrypt-prod
          nginx.ingress.kubernetes.io/backend-protocol: HTTP
          
      # Plugin configuration
      plugins:
        - name: grafana-clock-panel
          version: latest
        - name: grafana-piechart-panel
          version: latest
        - name: grafana-worldmap-panel
          version: 0.3.3
          
      # Dashboard provisioning
      dashboards:
        automaticProvisioning: true
        providers:
          - name: default
            orgId: 1
            folder: ''
            type: file
            disableDeletion: false
            updateIntervalSeconds: 30
            allowUiUpdates: true
            options:
              path: /var/lib/grafana/dashboards
              
        # ConfigMap-based dashboards
        configMapSelector:
          matchLabels:
            grafana_dashboard: "true"
            
        # URL-based dashboards
        urls:
          - name: kubernetes-cluster-monitoring
            url: https://grafana.com/api/dashboards/12114/revisions/1/download
            folder: Kubernetes
            datasource: Prometheus
          - name: prometheus-stats
            url: https://grafana.com/api/dashboards/3662/revisions/2/download
            folder: Prometheus
            datasource: Prometheus
            
      # Datasource configuration
      datasources:
        automaticProvisioning: true
        deleteDatasources:
          - name: obsolete-datasource
            orgId: 1
        providers:
          - name: prometheus
            type: prometheus
            url: http://prometheus:9090
            access: proxy
            isDefault: true
            jsonData:
              timeInterval: 30s
              queryTimeout: 60s
              httpMethod: POST
          - name: loki
            type: loki
            url: http://loki:3100
            access: proxy
            
      # Notification channels
      notificationChannels:
        - name: slack
          type: slack
          settings:
            url: https://hooks.slack.com/services/XXX/YYY/ZZZ
            recipient: "#alerts"
            
      # Resources
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "1"
          memory: "1Gi"
          
      # Persistence
      persistence:
        enabled: true
        size: 10Gi
        storageClassName: standard
```

### Loki Migration

**v1alpha1 Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: loki-example
  namespace: monitoring
spec:
  components:
    loki:
      enabled: true
      version: "2.8.0"
      storageSize: 100Gi
      retentionDays: 30
      s3Config: |
        endpoint: s3.amazonaws.com
        bucketnames: loki-data
        region: us-east-1
        access_key_id: ${AWS_ACCESS_KEY_ID}
        secret_access_key: ${AWS_SECRET_ACCESS_KEY}
```

**v1beta1 Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: loki-example
  namespace: monitoring
spec:
  global:
    logLevel: info
    logFormat: json
    
  components:
    loki:
      enabled: true
      version: "2.9.0"
      
      # Storage configuration
      storage:
        type: s3
        size: 100Gi  # For local cache
        s3:
          endpoint: s3.amazonaws.com
          region: us-east-1
          bucketNames:
            chunks: loki-chunks
            ruler: loki-ruler
            admin: loki-admin
          # Use secret references
          secretName: loki-s3-secret
          # Or IAM role
          iamRole: arn:aws:iam::123456789012:role/loki-s3-access
          
      # Schema configuration
      schemaConfig:
        configs:
          - from: "2024-01-01"
            store: boltdb-shipper
            objectStore: s3
            schema: v11
            index:
              prefix: index_
              period: 24h
              
      # Limits configuration
      limits:
        retention: 30d
        ingestionRate: 10MB
        ingestionBurstSize: 20MB
        maxStreamsPerUser: 10000
        maxGlobalStreamsPerUser: 10000
        maxLineSize: 256KB
        maxEntriesLimitPerQuery: 5000
        
      # Compactor configuration
      compactor:
        enabled: true
        retentionEnabled: true
        retentionDeleteDelay: 2h
        compactionInterval: 10m
        
      # Table manager (deprecated in newer versions)
      tableManager:
        retentionDeletesEnabled: true
        retentionPeriod: 720h
        
      # Resources
      resources:
        requests:
          cpu: "500m"
          memory: "1Gi"
        limits:
          cpu: "2"
          memory: "4Gi"
          
      # Replication
      replicas: 3
      replicationFactor: 3
      
      # Ring configuration
      ring:
        kvstore:
          store: memberlist
        replicationFactor: 3
```

### Tempo Migration

**v1alpha1 Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: tempo-example
  namespace: monitoring
spec:
  components:
    tempo:
      enabled: true
      version: "2.1.0"
      backend: s3
      s3Bucket: tempo-traces
      retentionHours: 168
```

**v1beta1 Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: tempo-example
  namespace: monitoring
spec:
  global:
    logLevel: info
    logFormat: json
    
  components:
    tempo:
      enabled: true
      version: "2.3.0"
      
      # Storage configuration
      storage:
        backend: s3
        s3:
          bucket: tempo-traces
          endpoint: s3.amazonaws.com
          region: us-east-1
          # Credentials
          secretName: tempo-s3-secret
          # Path prefix
          prefix: traces
          # Additional options
          forcePathStyle: false
          insecure: false
          
      # Trace storage configuration
      trace:
        backend: s3
        wal:
          path: /var/tempo/wal
        s3:
          bucket: tempo-traces
          
      # Retention configuration
      retention: 168h  # 7 days
      
      # Compactor configuration
      compactor:
        enabled: true
        retention: 168h
        compactedBlockRetention: 1h
        blockRetention: 1h
        
      # Query frontend
      queryFrontend:
        enabled: true
        replicas: 2
        search:
          maxDuration: 0s
          
      # Ingester configuration
      ingester:
        maxBlockDuration: 30m
        maxBlockBytes: 1000000000
        
      # Resources
      resources:
        requests:
          cpu: "200m"
          memory: "512Mi"
        limits:
          cpu: "1"
          memory: "2Gi"
          
      # Replicas
      replicas: 2
      
      # Metrics generator (new in v1beta1)
      metricsGenerator:
        enabled: true
        remoteWrite:
          - url: http://prometheus:9090/api/v1/write
        storage:
          path: /var/tempo/wal/generator
        processorsList:
          - service-graphs
          - span-metrics
```

## Advanced Migration Scenarios

### Multi-Component Platform with Integration

**v1alpha1 Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: full-stack
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      version: "2.45.0"
      customConfig: |
        global:
          scrape_interval: 30s
        remote_write:
          - url: http://cortex:9009/api/v1/push
      storageSize: 200Gi
      retentionDays: 30
      
    grafana:
      enabled: true
      version: "9.5.0"
      adminPassword: admin123
      datasources: |
        - name: Prometheus
          type: prometheus
          url: http://prometheus:9090
        - name: Loki
          type: loki
          url: http://loki:3100
        - name: Tempo
          type: tempo
          url: http://tempo:3200
          
    loki:
      enabled: true
      version: "2.8.0"
      storageSize: 300Gi
      retentionDays: 7
      
    tempo:
      enabled: true
      version: "2.1.0"
      backend: local
      retentionHours: 72
      
  alerting:
    alertManager:
      enabled: true
      config: |
        global:
          resolve_timeout: 5m
        route:
          group_by: ['alertname']
          group_wait: 10s
          group_interval: 10s
          repeat_interval: 1h
          receiver: 'slack'
        receivers:
        - name: 'slack'
          slack_configs:
          - api_url: 'YOUR_SLACK_WEBHOOK'
            channel: '#alerts'
```

**v1beta1 Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: full-stack
  namespace: monitoring
  annotations:
    observability.io/description: "Full observability stack with metrics, logs, and traces"
spec:
  global:
    logLevel: info
    logFormat: json
    # Global labels applied to all components
    labels:
      environment: production
      team: platform
    
  components:
    prometheus:
      enabled: true
      version: "v2.48.0"
      
      global:
        scrapeInterval: 30s
        evaluationInterval: 30s
        
      # Remote write to Cortex
      remoteWrite:
        - url: http://cortex:9009/api/v1/push
          name: cortex-prod
          queueConfig:
            capacity: 10000
            maxShards: 30
          headers:
            X-Scope-OrgID: "production"
            
      storage:
        size: 200Gi
        storageClassName: fast-ssd
        
      retention: 30d
      retentionSize: 180GB
      
      resources:
        requests:
          cpu: "2"
          memory: "8Gi"
        limits:
          cpu: "4"
          memory: "16Gi"
          
      # Service monitors for component discovery
      serviceMonitor:
        enabled: true
        selector:
          matchLabels:
            monitoring: "true"
            
      # Pod monitors for pod discovery
      podMonitor:
        enabled: true
        selector:
          matchLabels:
            monitoring: "true"
            
    grafana:
      enabled: true
      version: "10.2.0"
      
      auth:
        adminPasswordSecret:
          name: grafana-admin
          key: password
        providers:
          - name: google
            enabled: true
            clientId: ${GOOGLE_CLIENT_ID}
            clientSecret:
              name: grafana-oauth
              key: google-secret
              
      # Automatic datasource provisioning
      datasources:
        automaticProvisioning: true
        providers:
          - name: Prometheus
            type: prometheus
            url: http://prometheus:9090
            access: proxy
            isDefault: true
            jsonData:
              timeInterval: 30s
              queryTimeout: 60s
              incrementalQuerying: true
              
          - name: Loki
            type: loki
            url: http://loki:3100
            access: proxy
            jsonData:
              derivedFields:
                - datasourceName: Tempo
                  matcherRegex: "traceID=(\\w+)"
                  name: TraceID
                  url: "$${__value.raw}"
                  
          - name: Tempo
            type: tempo
            url: http://tempo:3200
            access: proxy
            jsonData:
              tracesToLogs:
                datasourceName: Loki
                tags: ['pod', 'namespace']
                
      # Dashboard provisioning
      dashboards:
        automaticProvisioning: true
        configMapSelector:
          matchLabels:
            grafana_dashboard: "true"
        providers:
          - name: default
            folder: General
            type: file
            options:
              path: /var/lib/grafana/dashboards/general
              
      # Alert provisioning
      alerting:
        contactPoints:
          - name: slack
            type: slack
            settings:
              url: ${SLACK_WEBHOOK}
              title: "Grafana Alert"
              
      resources:
        requests:
          cpu: "200m"
          memory: "512Mi"
        limits:
          cpu: "1"
          memory: "2Gi"
          
    loki:
      enabled: true
      version: "2.9.0"
      
      storage:
        size: 300Gi
        storageClassName: fast-ssd
        
      limits:
        retention: 7d
        ingestionRate: 10MB
        ingestionBurstSize: 20MB
        
      # Multi-tenant configuration
      auth:
        enabled: false  # Single tenant mode
        
      # Schema config for TSDB
      schemaConfig:
        configs:
          - from: "2024-01-01"
            store: tsdb
            objectStore: filesystem
            schema: v12
            index:
              prefix: index_
              period: 24h
              
      resources:
        requests:
          cpu: "1"
          memory: "2Gi"
        limits:
          cpu: "2"
          memory: "4Gi"
          
    tempo:
      enabled: true
      version: "2.3.0"
      
      storage:
        backend: local
        local:
          path: /var/tempo
          
      retention: 72h
      
      # Metrics generator for RED metrics
      metricsGenerator:
        enabled: true
        remoteWrite:
          - url: http://prometheus:9090/api/v1/write
        storage:
          path: /var/tempo/generator
          
      resources:
        requests:
          cpu: "500m"
          memory: "1Gi"
        limits:
          cpu: "1"
          memory: "2Gi"
          
  # Alerting configuration
  alerting:
    alertmanager:  # Note: lowercase 'm' in v1beta1
      enabled: true
      version: "v0.26.0"
      
      config:
        global:
          resolveTimeout: 5m
          slackApiUrl:
            name: alertmanager-slack
            key: webhook-url
            
        route:
          groupBy: ['alertname', 'cluster', 'service']
          groupWait: 10s
          groupInterval: 10s
          repeatInterval: 1h
          receiver: slack-notifications
          routes:
            - match:
                severity: critical
              receiver: pagerduty
              
        receivers:
          - name: slack-notifications
            slackConfigs:
              - channel: '#alerts'
                title: 'Alert: {{ .GroupLabels.alertname }}'
                text: '{{ range .Alerts }}{{ .Annotations.summary }}\n{{ end }}'
                
          - name: pagerduty
            pagerdutyConfigs:
              - serviceKey:
                  name: alertmanager-pagerduty
                  key: service-key
                  
      resources:
        requests:
          cpu: "100m"
          memory: "128Mi"
        limits:
          cpu: "500m"
          memory: "512Mi"
          
  # Integration configuration
  integration:
    enabled: true
    # Service mesh integration
    serviceMesh:
      enabled: true
      provider: istio
      
    # OpenTelemetry integration
    openTelemetry:
      enabled: true
      collector:
        mode: deployment
        replicas: 2
        
  # Monitoring configuration
  monitoring:
    selfMonitoring:
      enabled: true
      
  # Security configuration
  security:
    rbac:
      create: true
    podSecurityPolicy:
      enabled: true
    networkPolicy:
      enabled: true
      
  # Backup configuration
  backup:
    enabled: true
    schedule: "0 2 * * *"
    retention: 7
    storage:
      type: s3
      s3:
        bucket: observability-backups
        region: us-east-1
```

## GitOps Migration Examples

### ArgoCD Application Migration

**v1alpha1 ArgoCD Application:**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: observability-platform
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/company/gitops
    targetRevision: main
    path: observability/v1alpha1
  destination:
    server: https://kubernetes.default.svc
    namespace: monitoring
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**v1beta1 ArgoCD Application with Migration:**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: observability-platform
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/company/gitops
    targetRevision: feature/migrate-v1beta1  # Migration branch
    path: observability/v1beta1
    
    # Use plugin for conversion
    plugin:
      name: gunj-migration
      env:
        - name: MIGRATION_MODE
          value: "true"
        - name: SOURCE_VERSION
          value: "v1alpha1"
        - name: TARGET_VERSION
          value: "v1beta1"
          
  destination:
    server: https://kubernetes.default.svc
    namespace: monitoring
    
  syncPolicy:
    automated:
      prune: false  # Disable during migration
      selfHeal: false
    syncOptions:
    - CreateNamespace=true
    - PruneLast=true
    - RespectIgnoreDifferences=true
    
  # Ignore differences during migration
  ignoreDifferences:
  - group: observability.io
    kind: ObservabilityPlatform
    jsonPointers:
    - /apiVersion
    - /status
```

### Flux GitOps Migration

**v1alpha1 Flux Configuration:**
```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: GitRepository
metadata:
  name: observability-config
  namespace: flux-system
spec:
  interval: 1m
  ref:
    branch: main
  url: https://github.com/company/observability-config
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: observability-platform
  namespace: flux-system
spec:
  interval: 10m
  path: "./platforms/production"
  prune: true
  sourceRef:
    kind: GitRepository
    name: observability-config
  targetNamespace: monitoring
```

**v1beta1 Flux Configuration with Migration:**
```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: GitRepository
metadata:
  name: observability-config
  namespace: flux-system
spec:
  interval: 1m
  ref:
    branch: migration/v1beta1  # Migration branch
  url: https://github.com/company/observability-config
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: observability-platform
  namespace: flux-system
spec:
  interval: 10m
  path: "./platforms/production"
  prune: false  # Disable during migration
  sourceRef:
    kind: GitRepository
    name: observability-config
  targetNamespace: monitoring
  
  # Post-build substitutions for migration
  postBuild:
    substitute:
      API_VERSION: "observability.io/v1beta1"
      MIGRATION_ENABLED: "true"
      
  # Patches during migration
  patches:
    - target:
        kind: ObservabilityPlatform
        name: production
      patch: |
        - op: replace
          path: /apiVersion
          value: observability.io/v1beta1
        - op: add
          path: /metadata/annotations/observability.io~1migration-timestamp
          value: "${MIGRATION_TIMESTAMP}"
          
  # Health checks
  healthChecks:
    - apiVersion: observability.io/v1beta1
      kind: ObservabilityPlatform
      name: production
      namespace: monitoring
```

## Multi-Cluster Migration

### Hub-Spoke Architecture Migration

**v1alpha1 Hub Configuration:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: central-hub
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      customConfig: |
        global:
          external_labels:
            cluster: 'hub'
            role: 'central'
        
        # Federate from spoke clusters
        scrape_configs:
          - job_name: 'federate-spoke-1'
            scrape_interval: 30s
            honor_labels: true
            metrics_path: '/federate'
            params:
              'match[]':
                - '{job=~".*"}'
            static_configs:
              - targets:
                - 'prometheus-spoke-1.monitoring.svc.cluster.local:9090'
```

**v1beta1 Hub Configuration:**
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: central-hub
  namespace: monitoring
spec:
  global:
    logLevel: info
    logFormat: json
    
  # Multi-cluster federation configuration
  federation:
    enabled: true
    role: hub
    
  components:
    prometheus:
      enabled: true
      version: "v2.48.0"
      
      externalLabels:
        cluster: hub
        role: central
        
      # Federation configuration
      federation:
        enabled: true
        endpoints:
          - name: spoke-1
            url: http://prometheus-spoke-1.monitoring.svc.cluster.local:9090
            interval: 30s
            honorLabels: true
            matchExpressions:
              - '{__name__=~"job:.*"}'
              - '{__name__=~"cluster:.*"}'
          - name: spoke-2
            url: http://prometheus-spoke-2.monitoring.svc.cluster.local:9090
            interval: 30s
            honorLabels: true
            
      # Global query configuration
      globalView:
        enabled: true
        queryTimeout: 5m
        maxSamples: 50000000
        
      # Increased resources for federation
      resources:
        requests:
          cpu: "4"
          memory: "16Gi"
        limits:
          cpu: "8"
          memory: "32Gi"
          
      storage:
        size: 1Ti
        
    # Grafana with multi-cluster datasources
    grafana:
      enabled: true
      version: "10.2.0"
      
      datasources:
        automaticProvisioning: true
        providers:
          - name: prometheus-hub
            type: prometheus
            url: http://prometheus:9090
            isDefault: true
            jsonData:
              customQueryParameters: "cluster=${cluster}"
              
          - name: prometheus-spoke-1
            type: prometheus
            url: http://prometheus-spoke-1:9090
            jsonData:
              customQueryParameters: "cluster=spoke-1"
              
          - name: prometheus-spoke-2
            type: prometheus
            url: http://prometheus-spoke-2:9090
            jsonData:
              customQueryParameters: "cluster=spoke-2"
```

## Custom Configuration Migration

### Complex Prometheus Rules Migration

**v1alpha1 with embedded rules:**
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: rules-example
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      customConfig: |
        rule_files:
          - /etc/prometheus/rules/*.yaml
        
        # Embedded rules (anti-pattern)
        groups:
          - name: example
            interval: 30s
            rules:
              - alert: HighMemoryUsage
                expr: |
                  (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 0.85
                for: 5m
                labels:
                  severity: warning
                annotations:
                  summary: "High memory usage detected"
```

**v1beta1 with proper rule management:**
```yaml
# First, create PrometheusRule CRs
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: memory-alerts
  namespace: monitoring
  labels:
    prometheus: kube-prometheus
    role: alert-rules
spec:
  groups:
    - name: memory
      interval: 30s
      rules:
        - alert: HighMemoryUsage
          expr: |
            (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 0.85
          for: 5m
          labels:
            severity: warning
            component: node
          annotations:
            summary: "High memory usage on {{ $labels.instance }}"
            description: "Memory usage is above 85% (current value: {{ $value | humanizePercentage }})"
---
# Then reference in platform
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: rules-example
  namespace: monitoring
spec:
  global:
    logLevel: info
    logFormat: json
    
  components:
    prometheus:
      enabled: true
      version: "v2.48.0"
      
      # Rule discovery configuration
      ruleSelector:
        matchLabels:
          prometheus: kube-prometheus
          
      # Or use namespace selector
      ruleNamespaceSelector:
        matchLabels:
          monitoring: "true"
          
      # Additional rule files from ConfigMaps
      ruleFiles:
        configMapSelector:
          matchLabels:
            rule-type: prometheus
```

## Automation Scripts

### Complete Migration Script

```bash
#!/bin/bash
# migrate-platforms.sh - Comprehensive migration script

set -euo pipefail

# Configuration
MIGRATION_DIR="./migration-workspace"
BACKUP_DIR="$MIGRATION_DIR/backups"
CONVERTED_DIR="$MIGRATION_DIR/converted"
LOG_FILE="$MIGRATION_DIR/migration-$(date +%Y%m%d-%H%M%S).log"

# Create workspace
mkdir -p "$BACKUP_DIR" "$CONVERTED_DIR"

# Logging function
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Migration function for a single platform
migrate_platform() {
    local namespace=$1
    local name=$2
    local file="$namespace-$name"
    
    log "Migrating platform: $namespace/$name"
    
    # Export current configuration
    kubectl get observabilityplatform "$name" -n "$namespace" -o yaml > "$BACKUP_DIR/$file.yaml"
    
    # Convert to v1beta1
    ./gunj-migrate convert \
        -f "$BACKUP_DIR/$file.yaml" \
        -o "$CONVERTED_DIR/$file.yaml" \
        --add-annotations \
        --validate
    
    # Apply converted configuration
    kubectl apply -f "$CONVERTED_DIR/$file.yaml"
    
    # Wait for ready state
    kubectl wait --for=condition=Ready \
        observabilityplatform/"$name" \
        -n "$namespace" \
        --timeout=600s
    
    log "Platform $namespace/$name migrated successfully"
}

# Main migration process
main() {
    log "Starting platform migration process"
    
    # Get all platforms
    platforms=$(kubectl get observabilityplatforms.observability.io --all-namespaces -o json | \
        jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"')
    
    # Count platforms
    total=$(echo "$platforms" | wc -l)
    count=0
    
    # Migrate each platform
    for platform in $platforms; do
        namespace=$(echo "$platform" | cut -d/ -f1)
        name=$(echo "$platform" | cut -d/ -f2)
        
        count=$((count + 1))
        log "Processing platform $count/$total"
        
        migrate_platform "$namespace" "$name"
        
        # Add delay between migrations
        sleep 30
    done
    
    log "Migration completed successfully!"
    log "Backups stored in: $BACKUP_DIR"
    log "Converted files in: $CONVERTED_DIR"
}

# Run main function
main
```

### Validation Script

```bash
#!/bin/bash
# validate-migration.sh - Post-migration validation

echo "=== Post-Migration Validation ==="

# Function to check platform
check_platform() {
    local namespace=$1
    local name=$2
    
    echo "Checking platform: $namespace/$name"
    
    # Get platform details
    local platform=$(kubectl get observabilityplatform "$name" -n "$namespace" -o json)
    
    # Check API version
    local version=$(echo "$platform" | jq -r '.apiVersion')
    if [[ "$version" == *"v1beta1"* ]]; then
        echo "  ✓ API version: $version"
    else
        echo "  ✗ API version: $version (not v1beta1)"
        return 1
    fi
    
    # Check status
    local phase=$(echo "$platform" | jq -r '.status.phase')
    if [[ "$phase" == "Ready" ]]; then
        echo "  ✓ Status: $phase"
    else
        echo "  ✗ Status: $phase"
        return 1
    fi
    
    # Check for deprecated fields
    local deprecated=$(echo "$platform" | jq -r '[.. | objects | keys[]] | map(select(. == "customConfig" or . == "cpuRequest")) | unique | length')
    if [[ "$deprecated" -eq 0 ]]; then
        echo "  ✓ No deprecated fields found"
    else
        echo "  ✗ Found $deprecated deprecated fields"
        return 1
    fi
    
    return 0
}

# Get all platforms
platforms=$(kubectl get observabilityplatforms.observability.io --all-namespaces -o json | \
    jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"')

# Validate each platform
failed=0
for platform in $platforms; do
    namespace=$(echo "$platform" | cut -d/ -f1)
    name=$(echo "$platform" | cut -d/ -f2)
    
    if ! check_platform "$namespace" "$name"; then
        failed=$((failed + 1))
    fi
    echo ""
done

# Summary
total=$(echo "$platforms" | wc -l)
passed=$((total - failed))

echo "=== Validation Summary ==="
echo "Total platforms: $total"
echo "Passed: $passed"
echo "Failed: $failed"

if [[ "$failed" -eq 0 ]]; then
    echo "✓ All platforms successfully migrated!"
    exit 0
else
    echo "✗ Some platforms failed validation"
    exit 1
fi
```

## Common Patterns

### Pattern 1: Resource Standardization

Always convert to Kubernetes standard resource format:

```yaml
# Before (v1alpha1)
cpuRequest: "1000m"
cpuLimit: "2000m"
memoryRequest: "4Gi"
memoryLimit: "8Gi"

# After (v1beta1)
resources:
  requests:
    cpu: "1"        # Simplified
    memory: "4Gi"
  limits:
    cpu: "2"        # Simplified
    memory: "8Gi"
```

### Pattern 2: Duration Standardization

Convert all time values to duration strings:

```yaml
# Before (v1alpha1)
retentionDays: 30
scrapeInterval: 15  # seconds
timeoutMs: 5000    # milliseconds

# After (v1beta1)
retention: 30d
scrapeInterval: 15s
timeout: 5s
```

### Pattern 3: Structured Configuration

Replace string configurations with structured formats:

```yaml
# Before (v1alpha1)
labels: '{"env": "prod", "team": "platform"}'

# After (v1beta1)
labels:
  env: prod
  team: platform
```

### Pattern 4: Secret References

Replace plain text secrets with references:

```yaml
# Before (v1alpha1)
adminPassword: "admin123"

# After (v1beta1)
adminPasswordSecret:
  name: grafana-admin
  key: password
```

### Pattern 5: Ingress Configuration

Standardize ingress configuration:

```yaml
# Before (v1alpha1)
ingressEnabled: true
ingressHost: app.example.com
ingressTLS: true

# After (v1beta1)
ingress:
  enabled: true
  className: nginx
  host: app.example.com
  tls:
    enabled: true
    secretName: app-tls
```

---

**Document Version**: 1.0  
**Last Updated**: June 15, 2025  
**Next Review**: July 15, 2025

For more examples, visit our [GitHub repository](https://github.com/gunjanjp/gunj-operator/tree/main/examples).
