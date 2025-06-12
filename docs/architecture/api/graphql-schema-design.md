# Gunj Operator GraphQL Schema Design

**Version**: 1.0.0  
**Date**: June 12, 2025  
**Status**: Draft  
**Author**: Gunjan JP  

## Table of Contents
1. [Overview](#overview)
2. [Schema Design Principles](#schema-design-principles)
3. [Scalar Types](#scalar-types)
4. [Enum Types](#enum-types)
5. [Input Types](#input-types)
6. [Object Types](#object-types)
7. [Query Operations](#query-operations)
8. [Mutation Operations](#mutation-operations)
9. [Subscription Operations](#subscription-operations)
10. [Error Handling](#error-handling)
11. [Pagination](#pagination)
12. [Authentication & Authorization](#authentication--authorization)

## Overview

The Gunj Operator GraphQL API provides a flexible, type-safe query interface for managing observability platforms. It complements the REST API by allowing clients to request exactly the data they need in a single request.

## Schema Design Principles

1. **Type Safety**: Strong typing for all fields and operations
2. **Consistency**: Naming conventions aligned with Kubernetes and REST API
3. **Flexibility**: Support for partial field selection
4. **Performance**: DataLoader pattern for N+1 query prevention
5. **Real-time**: Subscriptions for live updates
6. **Compatibility**: Data structures match REST API responses

## Scalar Types

```graphql
# Custom scalar types
scalar DateTime    # ISO 8601 date-time string
scalar JSON       # Arbitrary JSON data
scalar Duration   # Duration in Go format (e.g., "5m", "1h")
scalar Quantity   # Kubernetes resource quantity (e.g., "100Mi", "2Gi")
scalar Percentage # Percentage value (0-100)
scalar URL        # Valid URL string
scalar SemVer     # Semantic version string
```

## Enum Types

```graphql
# Platform status phases
enum PlatformPhase {
  PENDING
  INSTALLING
  READY
  FAILED
  UPGRADING
  TERMINATING
}

# Component types
enum ComponentType {
  PROMETHEUS
  GRAFANA
  LOKI
  TEMPO
  ALERTMANAGER
}

# Component status
enum ComponentStatus {
  HEALTHY
  UNHEALTHY
  DEGRADED
  UNKNOWN
}

# Operation types
enum OperationType {
  BACKUP
  RESTORE
  UPGRADE
  SCALE
  RESTART
}

# Operation status
enum OperationStatus {
  PENDING
  IN_PROGRESS
  COMPLETED
  FAILED
  CANCELLED
}

# Sort order
enum SortOrder {
  ASC
  DESC
}

# Platform sort fields
enum PlatformSortField {
  NAME
  NAMESPACE
  CREATED_AT
  UPDATED_AT
  PHASE
}

# Event types
enum EventType {
  NORMAL
  WARNING
  ERROR
}

# Backup compression types
enum CompressionType {
  NONE
  GZIP
  ZSTD
}

# Cost period
enum CostPeriod {
  HOUR
  DAY
  WEEK
  MONTH
  YEAR
}

# Recommendation type
enum RecommendationType {
  RESOURCE_OPTIMIZATION
  STORAGE_CLEANUP
  VERSION_UPGRADE
  SECURITY_UPDATE
  COST_REDUCTION
}
```

## Input Types

```graphql
# Pagination input
input PaginationInput {
  page: Int = 1
  limit: Int = 20
  sort: PlatformSortField
  order: SortOrder = DESC
}

# Label selector input
input LabelSelectorInput {
  matchLabels: JSON
  matchExpressions: [LabelSelectorRequirementInput!]
}

input LabelSelectorRequirementInput {
  key: String!
  operator: String!
  values: [String!]
}

# Time range input
input TimeRangeInput {
  start: DateTime!
  end: DateTime!
  step: Duration
}

# Resource requirements input
input ResourceRequirementsInput {
  requests: ResourceListInput
  limits: ResourceListInput
}

input ResourceListInput {
  cpu: Quantity
  memory: Quantity
  storage: Quantity
}

# Platform creation input
input CreatePlatformInput {
  name: String!
  namespace: String = "default"
  labels: JSON
  annotations: JSON
  spec: PlatformSpecInput!
}

# Platform spec input
input PlatformSpecInput {
  components: ComponentsInput!
  global: GlobalConfigInput
  alerting: AlertingConfigInput
}

# Components configuration input
input ComponentsInput {
  prometheus: PrometheusSpecInput
  grafana: GrafanaSpecInput
  loki: LokiSpecInput
  tempo: TempoSpecInput
}

# Prometheus configuration input
input PrometheusSpecInput {
  enabled: Boolean = true
  version: SemVer!
  replicas: Int = 1
  resources: ResourceRequirementsInput
  storage: StorageConfigInput
  retention: Duration = "30d"
  externalLabels: JSON
  remoteWrite: [RemoteWriteConfigInput!]
}

# Storage configuration input
input StorageConfigInput {
  size: Quantity!
  storageClassName: String
  volumeMode: String = "Filesystem"
}

# Remote write configuration input
input RemoteWriteConfigInput {
  url: URL!
  basicAuth: BasicAuthInput
  tlsConfig: TLSConfigInput
  writeRelabelConfigs: JSON
}

input BasicAuthInput {
  username: String!
  passwordSecretRef: SecretKeySelectorInput!
}

input SecretKeySelectorInput {
  name: String!
  key: String!
}

input TLSConfigInput {
  insecureSkipVerify: Boolean = false
  ca: String
  cert: String
  key: String
}

# Grafana configuration input
input GrafanaSpecInput {
  enabled: Boolean = true
  version: SemVer!
  replicas: Int = 1
  resources: ResourceRequirementsInput
  adminPassword: String
  ingress: IngressConfigInput
  datasources: [DatasourceConfigInput!]
}

input IngressConfigInput {
  enabled: Boolean = false
  className: String
  host: String!
  path: String = "/"
  tls: IngressTLSInput
}

input IngressTLSInput {
  enabled: Boolean = false
  secretName: String
}

input DatasourceConfigInput {
  name: String!
  type: String!
  url: URL!
  access: String = "proxy"
  isDefault: Boolean = false
}

# Loki configuration input
input LokiSpecInput {
  enabled: Boolean = true
  version: SemVer!
  replicas: Int = 1
  resources: ResourceRequirementsInput
  storage: StorageConfigInput
  s3: S3ConfigInput
}

input S3ConfigInput {
  enabled: Boolean = false
  bucketName: String!
  region: String!
  endpoint: URL
  accessKeyId: String
  secretAccessKey: String
}

# Tempo configuration input
input TempoSpecInput {
  enabled: Boolean = true
  version: SemVer!
  replicas: Int = 1
  resources: ResourceRequirementsInput
  storage: StorageConfigInput
}

# Global configuration input
input GlobalConfigInput {
  externalLabels: JSON
  logLevel: String = "info"
}

# Alerting configuration input
input AlertingConfigInput {
  alertmanager: AlertmanagerConfigInput
}

input AlertmanagerConfigInput {
  enabled: Boolean = true
  config: JSON!
}

# Update platform input
input UpdatePlatformInput {
  labels: JSON
  annotations: JSON
  spec: PlatformSpecInput
}

# Patch platform input
input PatchPlatformInput {
  patches: [JSONPatchInput!]!
}

input JSONPatchInput {
  op: String!
  path: String!
  value: JSON
  from: String
}

# Backup operation input
input BackupInput {
  destination: String!
  includeData: Boolean = true
  includeConfigs: Boolean = true
  compression: CompressionType = GZIP
}

# Restore operation input
input RestoreInput {
  source: String!
  overwrite: Boolean = false
  components: [ComponentType!]
}

# Upgrade operation input
input UpgradeInput {
  targetVersion: SemVer!
  strategy: String = "rolling"
  backupFirst: Boolean = true
}

# Scale operation input
input ScaleInput {
  replicas: Int!
}

# Webhook input
input WebhookInput {
  name: String!
  url: URL!
  events: [String!]!
  headers: JSON
  active: Boolean = true
}

# Authentication input
input LoginInput {
  username: String!
  password: String!
}

input RefreshTokenInput {
  refreshToken: String!
}
```

## Object Types

```graphql
# Kubernetes metadata
type ObjectMeta {
  name: String!
  namespace: String!
  uid: String!
  resourceVersion: String!
  generation: Int!
  creationTimestamp: DateTime!
  deletionTimestamp: DateTime
  labels: JSON
  annotations: JSON
  finalizers: [String!]
}

# Platform type
type Platform {
  metadata: ObjectMeta!
  spec: PlatformSpec!
  status: PlatformStatus!
  
  # Computed fields
  components: [Component!]!
  metrics(range: TimeRangeInput!): PlatformMetrics!
  health: PlatformHealth!
  events(since: DateTime, limit: Int = 100): [Event!]!
  operations(status: OperationStatus): [Operation!]!
  cost(period: CostPeriod = MONTH): CostAnalysis!
  recommendations: [Recommendation!]!
}

# Platform specification
type PlatformSpec {
  components: Components!
  global: GlobalConfig
  alerting: AlertingConfig
}

# Platform status
type PlatformStatus {
  phase: PlatformPhase!
  message: String
  reason: String
  conditions: [Condition!]!
  componentStatuses: JSON
  observedGeneration: Int
  lastUpdated: DateTime!
}

# Condition type
type Condition {
  type: String!
  status: String!
  reason: String
  message: String
  lastTransitionTime: DateTime!
}

# Components configuration
type Components {
  prometheus: PrometheusSpec
  grafana: GrafanaSpec
  loki: LokiSpec
  tempo: TempoSpec
}

# Component interface
interface Component {
  type: ComponentType!
  enabled: Boolean!
  version: SemVer!
  status: ComponentStatus!
  health: ComponentHealth!
  resources: ResourceRequirements
}

# Prometheus component
type PrometheusComponent implements Component {
  type: ComponentType!
  enabled: Boolean!
  version: SemVer!
  status: ComponentStatus!
  health: ComponentHealth!
  resources: ResourceRequirements
  spec: PrometheusSpec!
  metrics: PrometheusMetrics!
  targets: [ScrapeTarget!]!
}

# Grafana component
type GrafanaComponent implements Component {
  type: ComponentType!
  enabled: Boolean!
  version: SemVer!
  status: ComponentStatus!
  health: ComponentHealth!
  resources: ResourceRequirements
  spec: GrafanaSpec!
  dashboards: [Dashboard!]!
  datasources: [Datasource!]!
}

# Loki component
type LokiComponent implements Component {
  type: ComponentType!
  enabled: Boolean!
  version: SemVer!
  status: ComponentStatus!
  health: ComponentHealth!
  resources: ResourceRequirements
  spec: LokiSpec!
  ingestionRate: Float!
  storageUsage: Quantity!
}

# Tempo component
type TempoComponent implements Component {
  type: ComponentType!
  enabled: Boolean!
  version: SemVer!
  status: ComponentStatus!
  health: ComponentHealth!
  resources: ResourceRequirements
  spec: TempoSpec!
  traceCount: Int!
  spanRate: Float!
}

# Component specifications (matching input types)
type PrometheusSpec {
  enabled: Boolean!
  version: SemVer!
  replicas: Int!
  resources: ResourceRequirements
  storage: StorageConfig
  retention: Duration!
  externalLabels: JSON
  remoteWrite: [RemoteWriteConfig!]
}

type GrafanaSpec {
  enabled: Boolean!
  version: SemVer!
  replicas: Int!
  resources: ResourceRequirements
  ingress: IngressConfig
  datasources: [DatasourceConfig!]
}

type LokiSpec {
  enabled: Boolean!
  version: SemVer!
  replicas: Int!
  resources: ResourceRequirements
  storage: StorageConfig
  s3: S3Config
}

type TempoSpec {
  enabled: Boolean!
  version: SemVer!
  replicas: Int!
  resources: ResourceRequirements
  storage: StorageConfig
}

# Supporting types
type ResourceRequirements {
  requests: ResourceList
  limits: ResourceList
}

type ResourceList {
  cpu: Quantity
  memory: Quantity
  storage: Quantity
}

type StorageConfig {
  size: Quantity!
  storageClassName: String
  volumeMode: String!
}

type IngressConfig {
  enabled: Boolean!
  className: String
  host: String!
  path: String!
  tls: IngressTLS
}

type IngressTLS {
  enabled: Boolean!
  secretName: String
}

# Health and metrics types
type PlatformHealth {
  overall: ComponentStatus!
  components: JSON!
  lastCheck: DateTime!
}

type ComponentHealth {
  status: ComponentStatus!
  ready: Boolean!
  replicas: ReplicaStatus!
  lastProbe: DateTime!
}

type ReplicaStatus {
  desired: Int!
  ready: Int!
  available: Int!
}

type PlatformMetrics {
  cpu: TimeSeriesData!
  memory: TimeSeriesData!
  storage: TimeSeriesData!
  requestRate: TimeSeriesData!
  errorRate: TimeSeriesData!
  latency: LatencyMetrics!
}

type TimeSeriesData {
  timestamps: [DateTime!]!
  values: [Float!]!
  unit: String!
}

type LatencyMetrics {
  p50: Float!
  p90: Float!
  p95: Float!
  p99: Float!
}

# Event type
type Event {
  type: EventType!
  reason: String!
  message: String!
  source: EventSource!
  firstTimestamp: DateTime!
  lastTimestamp: DateTime!
  count: Int!
}

type EventSource {
  component: String
  host: String
}

# Operation type
type Operation {
  id: String!
  type: OperationType!
  status: OperationStatus!
  platform: Platform!
  startedAt: DateTime!
  completedAt: DateTime
  error: String
  details: JSON
}

# Cost analysis
type CostAnalysis {
  period: CostPeriod!
  total: Float!
  breakdown: CostBreakdown!
  trend: CostTrend!
  recommendations: [CostRecommendation!]!
}

type CostBreakdown {
  byComponent: JSON!
  byResource: JSON!
  byNamespace: JSON!
}

type CostTrend {
  current: Float!
  previous: Float!
  change: Float!
  changePercent: Percentage!
}

type CostRecommendation {
  type: RecommendationType!
  description: String!
  potentialSavings: Float!
  effort: String!
  risk: String!
}

# Recommendation type
type Recommendation {
  id: String!
  type: RecommendationType!
  priority: String!
  title: String!
  description: String!
  impact: String!
  effort: String!
  automated: Boolean!
}

# Dashboard type
type Dashboard {
  uid: String!
  title: String!
  tags: [String!]!
  panels: [Panel!]!
  variables: [Variable!]!
}

type Panel {
  id: Int!
  title: String!
  type: String!
  gridPos: GridPos!
  targets: [Target!]!
}

type GridPos {
  x: Int!
  y: Int!
  w: Int!
  h: Int!
}

type Target {
  refId: String!
  expr: String!
  datasource: String!
}

type Variable {
  name: String!
  type: String!
  query: String!
  current: JSON
}

# Datasource type
type Datasource {
  uid: String!
  name: String!
  type: String!
  url: URL!
  access: String!
  isDefault: Boolean!
  status: DatasourceStatus!
}

type DatasourceStatus {
  connected: Boolean!
  message: String
  lastCheck: DateTime!
}

# Scrape target type
type ScrapeTarget {
  endpoint: String!
  state: String!
  labels: JSON!
  lastScrape: DateTime!
  lastError: String
}

# User type
type User {
  id: String!
  username: String!
  email: String!
  roles: [String!]!
  permissions: [String!]!
  createdAt: DateTime!
  lastLogin: DateTime
}

# Auth response
type AuthResponse {
  token: String!
  expiresIn: Int!
  refreshToken: String!
  user: User!
}

# Webhook type
type Webhook {
  id: String!
  name: String!
  url: URL!
  events: [String!]!
  headers: JSON
  active: Boolean!
  createdAt: DateTime!
  lastTriggered: DateTime
  failureCount: Int!
}

# Version info
type VersionInfo {
  version: SemVer!
  gitCommit: String!
  buildDate: DateTime!
  goVersion: String!
  platform: String!
  features: [String!]!
}

# Pagination info
type PageInfo {
  page: Int!
  limit: Int!
  total: Int!
  pages: Int!
  hasNext: Boolean!
  hasPrev: Boolean!
}

# Platform connection (for pagination)
type PlatformConnection {
  items: [Platform!]!
  pageInfo: PageInfo!
}
```

## Query Operations

```graphql
type Query {
  # Platform queries
  platforms(
    namespace: String
    labels: LabelSelectorInput
    pagination: PaginationInput
  ): PlatformConnection!
  
  platform(name: String!, namespace: String = "default"): Platform
  
  # Component queries
  components(
    platformName: String!
    namespace: String = "default"
  ): [Component!]!
  
  component(
    platformName: String!
    componentType: ComponentType!
    namespace: String = "default"
  ): Component
  
  # Metrics queries
  platformMetrics(
    name: String!
    namespace: String = "default"
    range: TimeRangeInput!
  ): PlatformMetrics!
  
  # Health queries
  platformHealth(
    name: String!
    namespace: String = "default"
  ): PlatformHealth!
  
  # Events queries
  events(
    platformName: String
    namespace: String
    type: EventType
    since: DateTime
    limit: Int = 100
  ): [Event!]!
  
  # Operations queries
  operations(
    platformName: String
    namespace: String
    type: OperationType
    status: OperationStatus
    limit: Int = 50
  ): [Operation!]!
  
  operation(id: String!): Operation
  
  # Cost queries
  costAnalysis(
    platformName: String!
    namespace: String = "default"
    period: CostPeriod = MONTH
  ): CostAnalysis!
  
  # Recommendations
  recommendations(
    platformName: String!
    namespace: String = "default"
    type: RecommendationType
  ): [Recommendation!]!
  
  # User queries
  currentUser: User
  users(role: String): [User!]!
  
  # Webhook queries
  webhooks(active: Boolean): [Webhook!]!
  webhook(id: String!): Webhook
  
  # System queries
  version: VersionInfo!
  health: SystemHealth!
  
  # Configuration validation
  validatePlatformConfig(input: CreatePlatformInput!): ValidationResult!
}

# System health
type SystemHealth {
  status: String!
  checks: JSON!
  timestamp: DateTime!
}

# Validation result
type ValidationResult {
  valid: Boolean!
  errors: [ValidationError!]!
  warnings: [ValidationWarning!]!
}

type ValidationError {
  field: String!
  message: String!
  code: String!
}

type ValidationWarning {
  field: String!
  message: String!
  code: String!
}
```

## Mutation Operations

```graphql
type Mutation {
  # Platform mutations
  createPlatform(input: CreatePlatformInput!): Platform!
  
  updatePlatform(
    name: String!
    namespace: String = "default"
    input: UpdatePlatformInput!
  ): Platform!
  
  patchPlatform(
    name: String!
    namespace: String = "default"
    input: PatchPlatformInput!
  ): Platform!
  
  deletePlatform(
    name: String!
    namespace: String = "default"
    cascade: Boolean = true
    gracePeriod: Int = 30
  ): Boolean!
  
  # Component mutations
  enableComponent(
    platformName: String!
    componentType: ComponentType!
    namespace: String = "default"
  ): Component!
  
  disableComponent(
    platformName: String!
    componentType: ComponentType!
    namespace: String = "default"
  ): Component!
  
  updateComponent(
    platformName: String!
    componentType: ComponentType!
    namespace: String = "default"
    spec: JSON!
  ): Component!
  
  # Operation mutations
  backupPlatform(
    name: String!
    namespace: String = "default"
    input: BackupInput!
  ): Operation!
  
  restorePlatform(
    name: String!
    namespace: String = "default"
    input: RestoreInput!
  ): Operation!
  
  upgradePlatform(
    name: String!
    namespace: String = "default"
    input: UpgradeInput!
  ): Operation!
  
  scaleComponent(
    platformName: String!
    componentType: ComponentType!
    namespace: String = "default"
    input: ScaleInput!
  ): Component!
  
  cancelOperation(id: String!): Operation!
  
  # Recommendation mutations
  applyRecommendation(
    recommendationId: String!
    dryRun: Boolean = false
  ): ApplyRecommendationResult!
  
  dismissRecommendation(
    recommendationId: String!
    reason: String
  ): Recommendation!
  
  # Authentication mutations
  login(input: LoginInput!): AuthResponse!
  refreshToken(input: RefreshTokenInput!): AuthResponse!
  logout: Boolean!
  
  # Webhook mutations
  createWebhook(input: WebhookInput!): Webhook!
  updateWebhook(id: String!, input: WebhookInput!): Webhook!
  deleteWebhook(id: String!): Boolean!
  testWebhook(id: String!): WebhookTestResult!
}

# Apply recommendation result
type ApplyRecommendationResult {
  success: Boolean!
  changes: [Change!]!
  error: String
}

type Change {
  resource: String!
  field: String!
  oldValue: JSON
  newValue: JSON
}

# Webhook test result
type WebhookTestResult {
  success: Boolean!
  statusCode: Int
  response: String
  error: String
  duration: Int!
}
```

## Subscription Operations

```graphql
type Subscription {
  # Platform subscriptions
  platformStatus(
    name: String!
    namespace: String = "default"
  ): PlatformStatus!
  
  platformEvents(
    name: String!
    namespace: String = "default"
    types: [EventType!]
  ): Event!
  
  # Component subscriptions
  componentStatus(
    platformName: String!
    componentType: ComponentType!
    namespace: String = "default"
  ): ComponentStatus!
  
  componentMetrics(
    platformName: String!
    componentType: ComponentType!
    namespace: String = "default"
  ): ComponentMetrics!
  
  # Operation subscriptions
  operationStatus(id: String!): Operation!
  
  # Real-time metrics
  metrics(
    platformName: String!
    namespace: String = "default"
    metricTypes: [MetricType!]
  ): MetricUpdate!
  
  # Log streaming
  logs(
    platformName: String!
    componentType: ComponentType!
    namespace: String = "default"
    filter: LogFilter
  ): LogEntry!
  
  # Alert streaming
  alerts(
    platformName: String
    namespace: String
    severity: [String!]
  ): Alert!
}

# Metric types for subscription
enum MetricType {
  CPU
  MEMORY
  DISK
  NETWORK
  REQUEST_RATE
  ERROR_RATE
  LATENCY
}

# Metric update
type MetricUpdate {
  type: MetricType!
  timestamp: DateTime!
  value: Float!
  labels: JSON!
}

# Component metrics
type ComponentMetrics {
  component: ComponentType!
  cpu: Float!
  memory: Float!
  disk: Float!
  timestamp: DateTime!
}

# Log filter
input LogFilter {
  level: String
  component: String
  search: String
  since: DateTime
}

# Log entry
type LogEntry {
  timestamp: DateTime!
  level: String!
  component: String!
  message: String!
  labels: JSON!
}

# Alert type
type Alert {
  fingerprint: String!
  status: String!
  labels: JSON!
  annotations: JSON!
  startsAt: DateTime!
  endsAt: DateTime
  generatorURL: URL!
}
```

## Error Handling

```graphql
# GraphQL errors follow the standard format:
{
  "errors": [
    {
      "message": "Platform not found",
      "extensions": {
        "code": "PLATFORM_NOT_FOUND",
        "platform": "production",
        "namespace": "monitoring"
      },
      "path": ["platform"],
      "locations": [{"line": 2, "column": 3}]
    }
  ]
}

# Error codes enum (for documentation)
enum ErrorCode {
  # Authentication errors
  UNAUTHENTICATED
  UNAUTHORIZED
  
  # Resource errors
  NOT_FOUND
  ALREADY_EXISTS
  CONFLICT
  
  # Validation errors
  INVALID_INPUT
  VALIDATION_ERROR
  
  # Operation errors
  OPERATION_FAILED
  OPERATION_TIMEOUT
  OPERATION_CANCELLED
  
  # System errors
  INTERNAL_ERROR
  SERVICE_UNAVAILABLE
  RATE_LIMITED
}
```

## Pagination

The GraphQL API uses cursor-based pagination for better performance with large datasets:

```graphql
# Extended connection pattern
type PlatformConnection {
  edges: [PlatformEdge!]!
  nodes: [Platform!]!  # Convenience field
  pageInfo: PageInfo!
  totalCount: Int!
}

type PlatformEdge {
  cursor: String!
  node: Platform!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

# Query with cursor pagination
type Query {
  platformsCursor(
    first: Int
    after: String
    last: Int
    before: String
    namespace: String
    labels: LabelSelectorInput
  ): PlatformConnection!
}
```

## Authentication & Authorization

```graphql
# Directives for field-level authorization
directive @auth(requires: [String!]!) on FIELD_DEFINITION
directive @hasRole(roles: [String!]!) on FIELD_DEFINITION
directive @hasPermission(permissions: [String!]!) on FIELD_DEFINITION

# Example usage:
type Platform {
  metadata: ObjectMeta!
  spec: PlatformSpec!
  status: PlatformStatus!
  
  # Sensitive operations require specific permissions
  backupCredentials: JSON @hasPermission(permissions: ["platform:backup:read"])
  costAnalysis: CostAnalysis! @hasRole(roles: ["admin", "finance"])
}

type Mutation {
  deletePlatform(name: String!, namespace: String!): Boolean! 
    @hasPermission(permissions: ["platform:delete"])
}
```

## Schema Extensions

```graphql
# Federation support for multi-service architecture
extend type Platform @key(fields: "metadata { name namespace }") {
  metadata: ObjectMeta! @external
  # Additional fields from other services
}

# Custom directives
directive @deprecated(reason: String!) on FIELD_DEFINITION | ENUM_VALUE
directive @complexity(value: Int!) on FIELD_DEFINITION
directive @rateLimit(max: Int!, window: String!) on FIELD_DEFINITION

# Complexity example
type Query {
  platforms: [Platform!]! @complexity(value: 10)
  platformMetrics(range: TimeRangeInput!): PlatformMetrics! @complexity(value: 50)
}
```

## Client Generation

The schema supports automatic client generation for multiple languages:

```yaml
# codegen.yml
schema: ./schema.graphql
generates:
  # TypeScript client
  ./src/generated/graphql.ts:
    plugins:
      - typescript
      - typescript-operations
      - typescript-react-apollo
  
  # Go client
  ./pkg/client/graphql/generated.go:
    plugins:
      - go
  
  # Python client
  ./python/gunj_operator/graphql/schema.py:
    plugins:
      - python
```

## Performance Considerations

1. **DataLoader Pattern**: Use for batching and caching
2. **Query Complexity**: Limit query depth and complexity
3. **Field Resolvers**: Lazy loading for expensive fields
4. **Subscriptions**: Use efficient pub/sub system
5. **Caching**: HTTP caching headers and Apollo Cache
6. **Rate Limiting**: Per-user and per-query limits

## Schema Evolution

1. **Backward Compatibility**: Never remove or rename fields
2. **Deprecation**: Mark fields with @deprecated directive
3. **Versioning**: Use field arguments for versioned behavior
4. **Migration**: Provide migration guides for breaking changes