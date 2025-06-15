# LokiConfig API Reference

## Overview

The `LokiConfig` custom resource definition (CRD) provides comprehensive configuration options for Grafana Loki, the horizontally-scalable, highly-available, multi-tenant log aggregation system. This CRD allows you to configure all aspects of Loki including storage backends, schema configuration, limits, authentication, and component-specific settings.

## API Version

- **Group**: `observability.io`
- **Version**: `v1beta1`
- **Kind**: `LokiConfig`

## Specification

### Root Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `version` | string | Loki version to deploy (e.g., "2.9.4", "3.0.0") | Yes |
| `storage` | [StorageConfig](#storageconfig) | Storage backend configuration | Yes |
| `schemaConfig` | [SchemaConfig](#schemaconfig) | Chunk and index schema configuration | Yes |
| `limits` | [LimitsConfig](#limitsconfig) | Per-tenant limits and quotas | No |
| `tableManager` | [TableManagerConfig](#tablemanagerconfig) | Table management and retention settings | No |
| `ingester` | [IngesterConfig](#ingesterconfig) | Ingester component configuration | No |
| `querier` | [QuerierConfig](#querierconfig) | Querier component configuration | No |
| `queryFrontend` | [QueryFrontendConfig](#queryfrontendconfig) | Query frontend configuration | No |
| `compactor` | [CompactorConfig](#compactorconfig) | Compactor configuration | No |
| `ruler` | [RulerConfig](#rulerconfig) | Ruler configuration for recording/alerting rules | No |
| `auth` | [AuthConfig](#authconfig) | Authentication configuration | No |
| `multiTenancy` | [MultiTenancyConfig](#multitenancyconfig) | Multi-tenancy configuration | No |
| `server` | [ServerConfig](#serverconfig) | Server configuration | No |
| `customConfig` | map[string]string | Additional custom configuration | No |

### StorageConfig

Defines the storage backend for chunks and indexes.

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `type` | string | Storage backend type: `s3`, `gcs`, `azure`, `filesystem`, `swift` | Yes |
| `s3` | [S3StorageConfig](#s3storageconfig) | S3 storage configuration | When type is `s3` |
| `gcs` | [GCSStorageConfig](#gcsstorageconfig) | Google Cloud Storage configuration | When type is `gcs` |
| `azure` | [AzureStorageConfig](#azurestorageconfig) | Azure Blob Storage configuration | When type is `azure` |
| `filesystem` | [FilesystemStorageConfig](#filesystemstorageconfig) | Filesystem storage configuration | When type is `filesystem` |
| `boltdb` | [BoltDBConfig](#boltdbconfig) | BoltDB configuration for index | No |
| `cache` | [CacheConfig](#cacheconfig) | Cache configuration | No |

#### S3StorageConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `bucketName` | string | S3 bucket name | - |
| `endpoint` | string | S3 endpoint (for S3-compatible storage) | AWS default |
| `region` | string | AWS region | - |
| `accessKeyId` | string | Access key ID | Uses IAM role |
| `secretAccessKey` | SecretKeySelector | Secret access key reference | Uses IAM role |
| `sse` | bool | Enable server-side encryption | false |
| `http2` | bool | Enable HTTP/2 | false |
| `forcePathStyle` | bool | Force path-style addressing | false |

#### GCSStorageConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `bucketName` | string | GCS bucket name | - |
| `serviceAccount` | SecretKeySelector | Service account key reference | Uses instance service account |
| `enableHttp2` | bool | Enable HTTP/2 | false |

#### AzureStorageConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `containerName` | string | Container name | - |
| `accountName` | string | Storage account name | - |
| `accountKey` | SecretKeySelector | Account key reference | - |
| `useManagedIdentity` | bool | Use Azure managed identity | false |
| `userAssignedId` | string | User-assigned identity ID | - |

### SchemaConfig

Defines how chunks and indexes are stored over time. Multiple configurations allow for schema migrations.

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `configs` | []SchemaConfigEntry | List of schema configurations | Yes (min 1) |

#### SchemaConfigEntry

| Field | Type | Description | Options |
|-------|------|-------------|---------|
| `from` | string | Start date (RFC3339 format) | - |
| `store` | string | Index store type | `boltdb`, `boltdb-shipper`, `tsdb` |
| `objectStore` | string | Object store type | `s3`, `gcs`, `azure`, `filesystem`, `swift` |
| `schema` | string | Schema version | `v11`, `v12`, `v13` |
| `index` | [IndexConfig](#indexconfig) | Index configuration | - |
| `chunks` | [ChunksConfig](#chunksconfig) | Chunks configuration | - |
| `rowShards` | int | Number of row shards | 0 |

### LimitsConfig

Configures per-tenant limits and quotas.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `ingestionRateMB` | float64 | Ingestion rate limit in MB/s | 4 |
| `ingestionBurstSizeMB` | float64 | Ingestion burst size in MB | 6 |
| `maxLabelNameLength` | int | Max label name length | 1024 |
| `maxLabelValueLength` | int | Max label value length | 2048 |
| `maxLabelNamesPerSeries` | int | Max label names per series | 30 |
| `rejectOldSamples` | bool | Reject old samples | false |
| `rejectOldSamplesMaxAge` | string | Max age for sample rejection | "168h" |
| `creationGracePeriod` | string | Grace period for out-of-order samples | "10m" |
| `maxStreamsPerUser` | int | Max streams per tenant | 5000 |
| `maxGlobalStreamsPerUser` | int | Max global streams per tenant | 5000 |
| `maxChunksPerQuery` | int | Max chunks per query | 2000000 |
| `maxQuerySeries` | int | Max series per query | - |
| `maxQueryLookback` | string | Max query lookback duration | "0s" |
| `maxQueryLength` | string | Max query time range | "721h" |
| `maxQueryParallelism` | int | Max query parallelism | 32 |
| `maxEntriesLimitPerQuery` | int | Max log entries per query | 5000 |
| `retentionPeriod` | string | Global retention period | - |
| `retentionStream` | []RetentionStreamConfig | Per-stream retention rules | - |

### IngesterConfig

Configures the ingester component responsible for writing logs.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `wal` | [WALConfig](#walconfig) | Write-ahead log configuration | - |
| `lifecycler` | [LifecyclerConfig](#lifecyclerconfig) | Lifecycler configuration | - |
| `chunkIdlePeriod` | string | Time before flushing idle chunks | "30m" |
| `chunkBlockSize` | int | Target block size in bytes | 262144 |
| `chunkTargetSize` | int | Target chunk size in bytes | 1572864 |
| `chunkEncoding` | string | Chunk compression | "gzip" |
| `maxChunkAge` | string | Max chunk age before flush | "2h" |
| `flushOnShutdown` | bool | Flush chunks on shutdown | false |
| `concurrentFlushes` | int | Number of concurrent flushes | - |

#### WALConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enabled` | bool | Enable write-ahead log | false |
| `dir` | string | WAL directory | "/loki/wal" |
| `checkpointDuration` | string | Checkpoint interval | "5m" |
| `flushOnShutdown` | bool | Flush on shutdown | false |
| `replayMemoryCeiling` | string | Memory limit for replay | "4GB" |

### QuerierConfig

Configures the querier component responsible for executing queries.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `maxConcurrent` | int | Max concurrent queries | 10 |
| `tailMaxDuration` | string | Max duration for tail requests | "1h" |
| `queryTimeout` | string | Query timeout | "1m" |
| `queryIngestersWithin` | string | Time range to query ingesters | "3h" |
| `engine` | [EngineConfig](#engineconfig) | Query engine configuration | - |

### QueryFrontendConfig

Configures the query frontend for caching and query optimization.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `compressResponses` | bool | Compress HTTP responses | false |
| `maxOutstandingPerTenant` | int | Max outstanding requests per tenant | 2048 |
| `logQueriesLongerThan` | string | Log slow queries | - |
| `splitQueriesByInterval` | string | Split queries by time interval | "30m" |
| `alignQueriesWithStep` | bool | Align queries with step | false |
| `cacheResults` | bool | Cache query results | false |
| `maxRetries` | int | Max query retries | 5 |
| `paralleliseShardableQueries` | bool | Parallelize shardable queries | false |

### CompactorConfig

Configures the compactor for index compaction and retention.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `workingDirectory` | string | Working directory | "/loki/compactor" |
| `sharedStore` | string | Shared store type | - |
| `compactionInterval` | string | Compaction interval | "10m" |
| `retentionEnabled` | bool | Enable retention | false |
| `retentionDeleteDelay` | string | Delay before deletion | "2h" |
| `retentionDeleteWorkerCount` | int | Number of deletion workers | 150 |
| `maxCompactionParallelism` | int | Max parallel compactions | 1 |

### RulerConfig

Configures the ruler for recording and alerting rules.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enableApi` | bool | Enable ruler API | false |
| `enableSharding` | bool | Enable rule sharding | false |
| `storage` | [RulerStorageConfig](#rulerstorageconfig) | Rule storage configuration | - |
| `evaluationInterval` | string | Rule evaluation interval | "1m" |
| `pollInterval` | string | Rule poll interval | "1m" |
| `alertmanagerUrl` | string | Alertmanager URL | - |
| `externalUrl` | string | External URL for alerts | - |
| `externalLabels` | map[string]string | External labels for alerts | - |
| `enableAlertmanagerV2` | bool | Enable Alertmanager v2 API | false |

### AuthConfig

Configures authentication for Loki.

| Field | Type | Description | Options |
|-------|------|-------------|---------|
| `type` | string | Authentication type | `basic`, `oidc`, `header` |
| `basic` | [BasicAuthConfig](#basicauthconfig) | Basic auth configuration | - |
| `oidc` | [OIDCConfig](#oidcconfig) | OIDC configuration | - |
| `header` | [HeaderAuthConfig](#headerauthconfig) | Header auth configuration | - |

### MultiTenancyConfig

Configures multi-tenancy support.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enabled` | bool | Enable multi-tenancy | false |
| `authEnabled` | bool | Enable authentication | false |
| `tenantIdHeader` | string | Tenant ID header | "X-Scope-OrgID" |
| `tenantIdLabel` | string | Tenant ID label | "tenant_id" |
| `tenants` | []TenantConfig | Per-tenant configurations | - |

### ServerConfig

Configures the Loki server.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `httpListenPort` | int | HTTP listen port | 3100 |
| `grpcListenPort` | int | gRPC listen port | 9095 |
| `registerInstrumentation` | bool | Register instrumentation handlers | false |
| `logLevel` | string | Log level | "info" |
| `logFormat` | string | Log format | "logfmt" |
| `grpcServerMaxRecvMsgSize` | int | Max gRPC receive message size | - |
| `grpcServerMaxSendMsgSize` | int | Max gRPC send message size | - |
| `httpServerReadTimeout` | string | HTTP read timeout | - |
| `httpServerWriteTimeout` | string | HTTP write timeout | - |

## Examples

### Basic Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: LokiConfig
metadata:
  name: loki-basic
  namespace: observability
spec:
  version: "2.9.4"
  storage:
    type: filesystem
    filesystem:
      directory: /loki/chunks
  schemaConfig:
    configs:
      - from: "2024-01-01T00:00:00Z"
        store: boltdb-shipper
        objectStore: filesystem
        schema: v11
```

### Production S3 Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: LokiConfig
metadata:
  name: loki-production
  namespace: observability
spec:
  version: "2.9.4"
  storage:
    type: s3
    s3:
      bucketName: company-loki-logs
      region: us-east-1
      sse: true
    cache:
      enableIndexCache: true
      indexCache:
        type: memcached
        memcached:
          addresses:
            - memcached.observability.svc.cluster.local:11211
  schemaConfig:
    configs:
      - from: "2024-01-01T00:00:00Z"
        store: tsdb
        objectStore: s3
        schema: v12
        rowShards: 32
  limits:
    ingestionRateMB: 100
    ingestionBurstSizeMB: 150
    maxStreamsPerUser: 50000
    retentionPeriod: "2160h"  # 90 days
  multiTenancy:
    enabled: true
    authEnabled: true
```

### Multi-Tenant Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: LokiConfig
metadata:
  name: loki-multitenant
  namespace: observability
spec:
  version: "2.9.4"
  storage:
    type: gcs
    gcs:
      bucketName: company-loki-logs
  schemaConfig:
    configs:
      - from: "2024-01-01T00:00:00Z"
        store: tsdb
        objectStore: gcs
        schema: v12
  auth:
    type: oidc
    oidc:
      issuerUrl: https://auth.company.com
      clientId: loki
      clientSecret:
        name: oidc-secret
        key: client-secret
  multiTenancy:
    enabled: true
    authEnabled: true
    tenants:
      - id: production
        displayName: "Production Team"
        limits:
          ingestionRateMB: 50
          retentionPeriod: "4320h"  # 180 days
      - id: development
        displayName: "Development Team"
        limits:
          ingestionRateMB: 10
          retentionPeriod: "168h"  # 7 days
```

## Best Practices

1. **Storage Backend Selection**
   - Use object storage (S3, GCS, Azure) for production
   - Filesystem storage is only recommended for testing
   - Enable caching for better query performance

2. **Schema Configuration**
   - Plan schema migrations carefully
   - Use TSDB store for better performance with Loki 2.8+
   - Increase row shards for high-volume deployments

3. **Limits and Retention**
   - Set appropriate ingestion limits to prevent abuse
   - Configure per-stream retention for cost optimization
   - Use cardinality limits to control label explosion

4. **Multi-Tenancy**
   - Always enable authentication for multi-tenant setups
   - Configure per-tenant limits based on usage patterns
   - Use external labels for tenant identification

5. **Performance Tuning**
   - Enable WAL for better ingestion reliability
   - Configure appropriate cache sizes
   - Use query frontend for better query performance
   - Enable compactor for index optimization

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   - Reduce `ingestionBurstSizeMB`
   - Lower `maxStreamsPerUser`
   - Enable streaming for queries

2. **Slow Queries**
   - Enable caching (index, chunks, results)
   - Increase `maxQueryParallelism`
   - Use query frontend splitting

3. **Storage Issues**
   - Verify storage permissions
   - Check available disk space
   - Validate cloud credentials

4. **Schema Migration Problems**
   - Never modify existing schema entries
   - Only add new entries with future dates
   - Test migrations in staging first

## Integration with ObservabilityPlatform

The LokiConfig CRD is designed to be referenced by the ObservabilityPlatform CRD:

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
spec:
  components:
    loki:
      enabled: true
      configRef:
        name: loki-production
        namespace: observability
```

This separation allows for:
- Reusable configurations across multiple platforms
- Independent lifecycle management
- Easier testing and validation
- GitOps-friendly configuration management
