# GraphQL Query Examples for Gunj Operator

This file contains practical examples of GraphQL queries, mutations, and subscriptions for the Gunj Operator API.

## Query Examples

### 1. List Platforms with Pagination

```graphql
query ListPlatforms($namespace: String, $page: Int, $limit: Int) {
  platforms(
    namespace: $namespace
    pagination: { page: $page, limit: $limit, sort: CREATED_AT, order: DESC }
  ) {
    items {
      metadata {
        name
        namespace
        labels
        creationTimestamp
      }
      status {
        phase
        message
        lastUpdated
      }
      components {
        type
        enabled
        version
        status
      }
    }
    pageInfo {
      page
      limit
      total
      hasNext
      hasPrev
    }
  }
}
```

### 2. Get Platform Details with Metrics

```graphql
query GetPlatformDetails($name: String!, $namespace: String!) {
  platform(name: $name, namespace: $namespace) {
    metadata {
      name
      namespace
      uid
      labels
      annotations
    }
    spec {
      components {
        prometheus {
          enabled
          version
          replicas
          retention
        }
        grafana {
          enabled
          version
          ingress {
            enabled
            host
          }
        }
      }
    }
    status {
      phase
      conditions {
        type
        status
        reason
        message
        lastTransitionTime
      }
    }
    health {
      overall
      components
      lastCheck
    }
    metrics(range: { start: "2025-06-12T00:00:00Z", end: "2025-06-12T12:00:00Z" }) {
      cpu {
        timestamps
        values
        unit
      }
      memory {
        timestamps
        values
        unit
      }
      requestRate {
        timestamps
        values
        unit
      }
    }
  }
}
```

### 3. Get Component Status

```graphql
query GetComponentStatus($platform: String!, $component: ComponentType!) {
  component(platformName: $platform, componentType: $component) {
    type
    enabled
    version
    status
    health {
      status
      ready
      replicas {
        desired
        ready
        available
      }
      lastProbe
    }
    ... on PrometheusComponent {
      targets {
        endpoint
        state
        labels
        lastScrape
        lastError
      }
      metrics {
        samplesIngested
        targetsActive
        seriesCount
        memoryUsage
      }
    }
    ... on GrafanaComponent {
      dashboards {
        uid
        title
        tags
      }
      datasources {
        name
        type
        url
        status {
          connected
          message
          lastCheck
        }
      }
    }
  }
}
```

### 4. Cost Analysis Query

```graphql
query GetCostAnalysis($platform: String!, $period: CostPeriod!) {
  costAnalysis(platformName: $platform, period: $period) {
    period
    total
    breakdown {
      byComponent
      byResource
    }
    trend {
      current
      previous
      change
      changePercent
    }
    recommendations {
      type
      description
      potentialSavings
      effort
      risk
    }
  }
}
```

### 5. Platform Events with Filtering

```graphql
query GetPlatformEvents($platform: String!, $since: DateTime, $type: EventType) {
  events(platformName: $platform, since: $since, type: $type, limit: 50) {
    type
    reason
    message
    source {
      component
      host
    }
    firstTimestamp
    lastTimestamp
    count
  }
}
```

## Mutation Examples

### 1. Create Platform

```graphql
mutation CreatePlatform($input: CreatePlatformInput!) {
  createPlatform(input: $input) {
    metadata {
      name
      namespace
      uid
    }
    status {
      phase
      message
    }
  }
}

# Variables
{
  "input": {
    "name": "production",
    "namespace": "monitoring",
    "labels": {
      "env": "prod",
      "team": "platform"
    },
    "spec": {
      "components": {
        "prometheus": {
          "enabled": true,
          "version": "v2.48.0",
          "replicas": 3,
          "resources": {
            "requests": {
              "cpu": "1",
              "memory": "4Gi"
            },
            "limits": {
              "cpu": "2",
              "memory": "8Gi"
            }
          },
          "storage": {
            "size": "100Gi",
            "storageClassName": "fast-ssd"
          },
          "retention": "30d"
        },
        "grafana": {
          "enabled": true,
          "version": "10.2.0",
          "ingress": {
            "enabled": true,
            "host": "grafana.example.com",
            "tls": {
              "enabled": true,
              "secretName": "grafana-tls"
            }
          }
        }
      },
      "global": {
        "externalLabels": {
          "cluster": "production",
          "region": "us-east-1"
        }
      }
    }
  }
}
```

### 2. Update Platform

```graphql
mutation UpdatePlatform($name: String!, $namespace: String!, $input: UpdatePlatformInput!) {
  updatePlatform(name: $name, namespace: $namespace, input: $input) {
    metadata {
      name
      resourceVersion
    }
    spec {
      components {
        prometheus {
          version
          replicas
        }
      }
    }
    status {
      phase
    }
  }
}

# Variables
{
  "name": "production",
  "namespace": "monitoring",
  "input": {
    "spec": {
      "components": {
        "prometheus": {
          "version": "v2.49.0",
          "replicas": 5
        }
      }
    }
  }
}
```

### 3. Scale Component

```graphql
mutation ScaleComponent($platform: String!, $component: ComponentType!, $replicas: Int!) {
  scaleComponent(
    platformName: $platform
    componentType: $component
    input: { replicas: $replicas }
  ) {
    type
    health {
      replicas {
        desired
        ready
        available
      }
    }
  }
}
```

### 4. Backup Platform

```graphql
mutation BackupPlatform($name: String!, $destination: String!) {
  backupPlatform(
    name: $name
    input: {
      destination: $destination
      includeData: true
      includeConfigs: true
      compression: GZIP
    }
  ) {
    id
    type
    status
    startedAt
    details
  }
}
```

### 5. Apply Recommendation

```graphql
mutation ApplyRecommendation($id: String!, $dryRun: Boolean) {
  applyRecommendation(recommendationId: $id, dryRun: $dryRun) {
    success
    changes {
      resource
      field
      oldValue
      newValue
    }
    error
  }
}
```

## Subscription Examples

### 1. Platform Status Updates

```graphql
subscription WatchPlatformStatus($name: String!, $namespace: String!) {
  platformStatus(name: $name, namespace: $namespace) {
    phase
    message
    conditions {
      type
      status
      reason
      lastTransitionTime
    }
    lastUpdated
  }
}
```

### 2. Real-time Metrics

```graphql
subscription StreamMetrics($platform: String!, $metrics: [MetricType!]) {
  metrics(platformName: $platform, metricTypes: $metrics) {
    type
    timestamp
    value
    labels
  }
}
```

### 3. Log Streaming

```graphql
subscription StreamLogs($platform: String!, $component: ComponentType!, $filter: LogFilter) {
  logs(
    platformName: $platform
    componentType: $component
    filter: $filter
  ) {
    timestamp
    level
    component
    message
    labels
  }
}

# Variables
{
  "platform": "production",
  "component": "PROMETHEUS",
  "filter": {
    "level": "error",
    "since": "2025-06-12T10:00:00Z"
  }
}
```

### 4. Alert Streaming

```graphql
subscription StreamAlerts($platform: String, $severity: [String!]) {
  alerts(platformName: $platform, severity: $severity) {
    fingerprint
    status
    labels
    annotations
    startsAt
    endsAt
  }
}
```

### 5. Operation Progress

```graphql
subscription WatchOperation($id: String!) {
  operationStatus(id: $id) {
    id
    type
    status
    completedAt
    error
    details
  }
}
```

## Complex Query Examples

### 1. Multi-Platform Dashboard

```graphql
query DashboardOverview {
  platforms {
    items {
      metadata {
        name
        namespace
      }
      status {
        phase
      }
      health {
        overall
      }
      cost(period: MONTH) {
        total
        trend {
          changePercent
        }
      }
    }
  }
  events(type: ERROR, limit: 10) {
    message
    source {
      component
    }
    lastTimestamp
  }
}
```

### 2. Platform Comparison

```graphql
query ComparePlatforms($platforms: [String!]!) {
  platform1: platform(name: $platforms[0]) {
    ...PlatformComparison
  }
  platform2: platform(name: $platforms[1]) {
    ...PlatformComparison
  }
}

fragment PlatformComparison on Platform {
  metadata {
    name
  }
  components {
    type
    version
    status
  }
  metrics(range: { start: "2025-06-12T00:00:00Z", end: "2025-06-12T12:00:00Z" }) {
    cpu {
      values
    }
    memory {
      values
    }
  }
  cost(period: MONTH) {
    total
  }
}
```

### 3. Batch Operations

```graphql
mutation BatchScaleComponents($operations: [ScaleOperation!]!) {
  results: {
    prometheus: scaleComponent(
      platformName: $operations[0].platform
      componentType: PROMETHEUS
      input: { replicas: $operations[0].replicas }
    ) {
      type
      health {
        replicas {
          desired
        }
      }
    }
    grafana: scaleComponent(
      platformName: $operations[1].platform
      componentType: GRAFANA
      input: { replicas: $operations[1].replicas }
    ) {
      type
      health {
        replicas {
          desired
        }
      }
    }
  }
}
```

## Authentication Examples

### 1. Login

```graphql
mutation Login($username: String!, $password: String!) {
  login(input: { username: $username, password: $password }) {
    token
    expiresIn
    refreshToken
    user {
      id
      username
      email
      roles
      permissions
    }
  }
}
```

### 2. Refresh Token

```graphql
mutation RefreshToken($refreshToken: String!) {
  refreshToken(input: { refreshToken: $refreshToken }) {
    token
    expiresIn
    refreshToken
  }
}
```

## Error Handling Example

```graphql
query GetPlatformWithErrorHandling($name: String!) {
  platform(name: $name) {
    metadata {
      name
    }
    status {
      phase
    }
  }
}

# Response with error
{
  "data": {
    "platform": null
  },
  "errors": [
    {
      "message": "Platform not found",
      "extensions": {
        "code": "PLATFORM_NOT_FOUND",
        "platform": "non-existent",
        "namespace": "default"
      },
      "path": ["platform"]
    }
  ]
}
```

## Using Directives

### 1. With Authentication

```graphql
query SecureQuery {
  currentUser @auth(requires: ["read:user"]) {
    id
    username
    roles
  }
  costAnalysis(platformName: "production") @hasRole(roles: ["admin", "finance"]) {
    total
    breakdown {
      byComponent
    }
  }
}
```

### 2. With Complexity Limits

```graphql
# This query has high complexity and might be rate-limited
query ComplexQuery {
  platforms {
    items {
      metadata {
        name
      }
      metrics(range: { start: "2025-06-01T00:00:00Z", end: "2025-06-12T00:00:00Z" }) {
        cpu {
          values
        }
        memory {
          values
        }
      }
      cost(period: YEAR) {
        breakdown {
          byComponent
          byResource
        }
      }
    }
  }
}
```