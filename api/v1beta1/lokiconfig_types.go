/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LokiConfigSpec defines the desired state of LokiConfig
type LokiConfigSpec struct {
	// Version of Loki to use
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^(2\.9\.\d+|3\.\d+\.\d+)$`
	Version string `json:"version"`

	// Storage configuration for Loki
	// +kubebuilder:validation:Required
	Storage LokiStorageConfig `json:"storage"`

	// SchemaConfig configures the chunk index schemas
	// +kubebuilder:validation:Required
	SchemaConfig SchemaConfig `json:"schemaConfig"`

	// Limits configuration for tenants
	// +optional
	Limits *LimitsConfig `json:"limits,omitempty"`

	// TableManager configuration for retention
	// +optional
	TableManager *TableManagerConfig `json:"tableManager,omitempty"`

	// Ingester configuration
	// +optional
	Ingester *IngesterConfig `json:"ingester,omitempty"`

	// Querier configuration
	// +optional
	Querier *QuerierConfig `json:"querier,omitempty"`

	// QueryFrontend configuration
	// +optional
	QueryFrontend *QueryFrontendConfig `json:"queryFrontend,omitempty"`

	// Compactor configuration
	// +optional
	Compactor *CompactorConfig `json:"compactor,omitempty"`

	// Ruler configuration for recording/alerting rules
	// +optional
	Ruler *RulerConfig `json:"ruler,omitempty"`

	// Authentication configuration
	// +optional
	Auth *LokiAuthConfig `json:"auth,omitempty"`

	// MultiTenancy configuration
	// +optional
	MultiTenancy *MultiTenancyConfig `json:"multiTenancy,omitempty"`

	// Server configuration
	// +optional
	Server *ServerConfig `json:"server,omitempty"`

	// Additional custom configuration
	// +optional
	CustomConfig map[string]string `json:"customConfig,omitempty"`
}

// LokiStorageConfig defines storage backend configuration
type LokiStorageConfig struct {
	// Type of storage backend
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=s3;gcs;azure;filesystem;swift
	Type string `json:"type"`

	// S3 storage configuration
	// +optional
	S3 *S3StorageConfig `json:"s3,omitempty"`

	// GCS storage configuration
	// +optional
	GCS *GCSStorageConfig `json:"gcs,omitempty"`

	// Azure storage configuration
	// +optional
	Azure *AzureStorageConfig `json:"azure,omitempty"`

	// Filesystem storage configuration
	// +optional
	Filesystem *FilesystemStorageConfig `json:"filesystem,omitempty"`

	// BoltDB configuration for index
	// +optional
	BoltDB *BoltDBConfig `json:"boltdb,omitempty"`

	// Cache configuration
	// +optional
	Cache *CacheConfig `json:"cache,omitempty"`
}

// S3StorageConfig defines S3 storage configuration
type S3StorageConfig struct {
	// S3 bucket name
	// +kubebuilder:validation:Required
	BucketName string `json:"bucketName"`

	// S3 endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// AWS region
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// Access key ID
	// +optional
	AccessKeyID string `json:"accessKeyId,omitempty"`

	// Secret access key reference
	// +optional
	SecretAccessKey *corev1.SecretKeySelector `json:"secretAccessKey,omitempty"`

	// Enable S3 SSE encryption
	// +optional
	SSE bool `json:"sse,omitempty"`

	// Enable HTTP2
	// +optional
	HTTP2 bool `json:"http2,omitempty"`

	// Force path-style addressing
	// +optional
	ForcePathStyle bool `json:"forcePathStyle,omitempty"`
}

// GCSStorageConfig defines Google Cloud Storage configuration
type GCSStorageConfig struct {
	// GCS bucket name
	// +kubebuilder:validation:Required
	BucketName string `json:"bucketName"`

	// Service account key reference
	// +optional
	ServiceAccount *corev1.SecretKeySelector `json:"serviceAccount,omitempty"`

	// Enable HTTP2
	// +optional
	EnableHTTP2 bool `json:"enableHttp2,omitempty"`
}

// AzureStorageConfig defines Azure Blob Storage configuration
type AzureStorageConfig struct {
	// Container name
	// +kubebuilder:validation:Required
	ContainerName string `json:"containerName"`

	// Account name
	// +kubebuilder:validation:Required
	AccountName string `json:"accountName"`

	// Account key reference
	// +optional
	AccountKey *corev1.SecretKeySelector `json:"accountKey,omitempty"`

	// Use managed identity
	// +optional
	UseManagedIdentity bool `json:"useManagedIdentity,omitempty"`

	// User assigned identity ID
	// +optional
	UserAssignedID string `json:"userAssignedId,omitempty"`
}

// FilesystemStorageConfig defines filesystem storage configuration
type FilesystemStorageConfig struct {
	// Directory path
	// +kubebuilder:validation:Required
	Directory string `json:"directory"`

	// Persistent volume claim
	// +optional
	PersistentVolumeClaim *corev1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
}

// BoltDBConfig defines BoltDB configuration for index
type BoltDBConfig struct {
	// Directory path for BoltDB files
	// +kubebuilder:validation:Required
	Directory string `json:"directory"`
}

// CacheConfig defines cache configuration
type CacheConfig struct {
	// Enable index query cache
	// +optional
	EnableIndexCache bool `json:"enableIndexCache,omitempty"`

	// Index cache configuration
	// +optional
	IndexCache *IndexCacheConfig `json:"indexCache,omitempty"`

	// Enable chunk cache
	// +optional
	EnableChunkCache bool `json:"enableChunkCache,omitempty"`

	// Chunk cache configuration
	// +optional
	ChunkCache *ChunkCacheConfig `json:"chunkCache,omitempty"`

	// Enable results cache
	// +optional
	EnableResultsCache bool `json:"enableResultsCache,omitempty"`

	// Results cache configuration
	// +optional
	ResultsCache *ResultsCacheConfig `json:"resultsCache,omitempty"`
}

// IndexCacheConfig defines index cache configuration
type IndexCacheConfig struct {
	// Cache type
	// +kubebuilder:validation:Enum=inmemory;memcached;redis
	Type string `json:"type"`

	// Memcached configuration
	// +optional
	Memcached *MemcachedConfig `json:"memcached,omitempty"`

	// Redis configuration
	// +optional
	Redis *RedisConfig `json:"redis,omitempty"`

	// In-memory cache size
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+(\.\d+)?([KMGT]i?)?$`
	InMemorySize string `json:"inMemorySize,omitempty"`
}

// ChunkCacheConfig defines chunk cache configuration
type ChunkCacheConfig struct {
	// Cache type
	// +kubebuilder:validation:Enum=inmemory;memcached;redis
	Type string `json:"type"`

	// Memcached configuration
	// +optional
	Memcached *MemcachedConfig `json:"memcached,omitempty"`

	// Redis configuration
	// +optional
	Redis *RedisConfig `json:"redis,omitempty"`
}

// ResultsCacheConfig defines results cache configuration
type ResultsCacheConfig struct {
	// Cache type
	// +kubebuilder:validation:Enum=inmemory;memcached;redis
	Type string `json:"type"`

	// Memcached configuration
	// +optional
	Memcached *MemcachedConfig `json:"memcached,omitempty"`

	// Redis configuration
	// +optional
	Redis *RedisConfig `json:"redis,omitempty"`

	// Max freshness duration
	// +optional
	MaxFreshness string `json:"maxFreshness,omitempty"`
}

// MemcachedConfig defines Memcached configuration
type MemcachedConfig struct {
	// Memcached addresses
	// +kubebuilder:validation:Required
	Addresses []string `json:"addresses"`

	// Timeout
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// Max idle connections
	// +optional
	MaxIdleConns int `json:"maxIdleConns,omitempty"`
}

// RedisConfig defines Redis configuration
type RedisConfig struct {
	// Redis endpoint
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`

	// Password reference
	// +optional
	Password *corev1.SecretKeySelector `json:"password,omitempty"`

	// DB number
	// +optional
	DB int `json:"db,omitempty"`

	// Enable TLS
	// +optional
	TLS bool `json:"tls,omitempty"`
}

// SchemaConfig defines the chunk index schema configuration
type SchemaConfig struct {
	// Config entries
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Configs []SchemaConfigEntry `json:"configs"`
}

// SchemaConfigEntry defines a schema configuration entry
type SchemaConfigEntry struct {
	// Start date of the schema (RFC3339)
	// +kubebuilder:validation:Required
	From string `json:"from"`

	// Store type for index
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=boltdb;boltdb-shipper;tsdb
	Store string `json:"store"`

	// Object store type
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=s3;gcs;azure;filesystem;swift
	ObjectStore string `json:"objectStore"`

	// Schema version
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=v11;v12;v13
	Schema string `json:"schema"`

	// Index configuration
	// +optional
	Index *IndexConfig `json:"index,omitempty"`

	// Chunks configuration
	// +optional
	Chunks *ChunksConfig `json:"chunks,omitempty"`

	// Row shards
	// +optional
	// +kubebuilder:validation:Minimum=0
	RowShards int `json:"rowShards,omitempty"`
}

// IndexConfig defines index configuration
type IndexConfig struct {
	// Prefix for index
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// Period for index rotation
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[hd]$`
	Period string `json:"period,omitempty"`

	// Tags for index
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// ChunksConfig defines chunks configuration
type ChunksConfig struct {
	// Prefix for chunks
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// Period for chunk rotation
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[hd]$`
	Period string `json:"period,omitempty"`

	// Tags for chunks
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// LimitsConfig defines per-tenant limits
type LimitsConfig struct {
	// Ingestion rate limit in MB/s
	// +optional
	// +kubebuilder:validation:Minimum=0
	IngestionRateMB float64 `json:"ingestionRateMB,omitempty"`

	// Ingestion burst size in MB
	// +optional
	// +kubebuilder:validation:Minimum=0
	IngestionBurstSizeMB float64 `json:"ingestionBurstSizeMB,omitempty"`

	// Max label name length
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxLabelNameLength int `json:"maxLabelNameLength,omitempty"`

	// Max label value length
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxLabelValueLength int `json:"maxLabelValueLength,omitempty"`

	// Max label names per series
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxLabelNamesPerSeries int `json:"maxLabelNamesPerSeries,omitempty"`

	// Reject old samples
	// +optional
	RejectOldSamples bool `json:"rejectOldSamples,omitempty"`

	// Reject old samples max age
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	RejectOldSamplesMaxAge string `json:"rejectOldSamplesMaxAge,omitempty"`

	// Creation grace period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	CreationGracePeriod string `json:"creationGracePeriod,omitempty"`

	// Max streams per user
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxStreamsPerUser int `json:"maxStreamsPerUser,omitempty"`

	// Max global streams per user
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxGlobalStreamsPerUser int `json:"maxGlobalStreamsPerUser,omitempty"`

	// Max chunks per query
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxChunksPerQuery int `json:"maxChunksPerQuery,omitempty"`

	// Max query series
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxQuerySeries int `json:"maxQuerySeries,omitempty"`

	// Max query lookback
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	MaxQueryLookback string `json:"maxQueryLookback,omitempty"`

	// Max query length
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	MaxQueryLength string `json:"maxQueryLength,omitempty"`

	// Max query parallelism
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxQueryParallelism int `json:"maxQueryParallelism,omitempty"`

	// Max entries limit per query
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxEntriesLimitPerQuery int `json:"maxEntriesLimitPerQuery,omitempty"`

	// Max cache freshness per query
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	MaxCacheFreshnessPerQuery string `json:"maxCacheFreshnessPerQuery,omitempty"`

	// Split queries by interval
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	SplitQueriesByInterval string `json:"splitQueriesByInterval,omitempty"`

	// Per-stream rate limit
	// +optional
	// +kubebuilder:validation:Minimum=0
	PerStreamRateLimit int `json:"perStreamRateLimit,omitempty"`

	// Per-stream rate limit burst
	// +optional
	// +kubebuilder:validation:Minimum=0
	PerStreamRateLimitBurst int `json:"perStreamRateLimitBurst,omitempty"`

	// Cardinality limit
	// +optional
	// +kubebuilder:validation:Minimum=0
	CardinalityLimit int `json:"cardinalityLimit,omitempty"`

	// Max streams matchers per query
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxStreamsMatchersPerQuery int `json:"maxStreamsMatchersPerQuery,omitempty"`

	// Max concurrent tail requests
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxConcurrentTailRequests int `json:"maxConcurrentTailRequests,omitempty"`

	// Retention period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[hd]$`
	RetentionPeriod string `json:"retentionPeriod,omitempty"`

	// Retention stream
	// +optional
	RetentionStream []RetentionStreamConfig `json:"retentionStream,omitempty"`
}

// RetentionStreamConfig defines retention configuration per stream
type RetentionStreamConfig struct {
	// Period for retention
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^\d+[hd]$`
	Period string `json:"period"`

	// Priority of the rule
	// +optional
	// +kubebuilder:validation:Minimum=0
	Priority int `json:"priority,omitempty"`

	// Selector for matching streams
	// +kubebuilder:validation:Required
	Selector string `json:"selector"`
}

// TableManagerConfig defines table management configuration
type TableManagerConfig struct {
	// Retention deletes enabled
	// +optional
	RetentionDeletesEnabled bool `json:"retentionDeletesEnabled,omitempty"`

	// Retention period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[hd]$`
	RetentionPeriod string `json:"retentionPeriod,omitempty"`

	// Poll interval
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	PollInterval string `json:"pollInterval,omitempty"`

	// Creation grace period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	CreationGracePeriod string `json:"creationGracePeriod,omitempty"`
}

// IngesterConfig defines ingester configuration
type IngesterConfig struct {
	// WAL configuration
	// +optional
	WAL *WALConfig `json:"wal,omitempty"`

	// Lifecycler configuration
	// +optional
	Lifecycler *LifecyclerConfig `json:"lifecycler,omitempty"`

	// Chunk idle period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	ChunkIdlePeriod string `json:"chunkIdlePeriod,omitempty"`

	// Chunk block size
	// +optional
	// +kubebuilder:validation:Minimum=0
	ChunkBlockSize int `json:"chunkBlockSize,omitempty"`

	// Chunk target size
	// +optional
	// +kubebuilder:validation:Minimum=0
	ChunkTargetSize int `json:"chunkTargetSize,omitempty"`

	// Chunk encoding
	// +optional
	// +kubebuilder:validation:Enum=none;gzip;lz4-64k;snappy;lz4-256k;lz4-1M;lz4;flate;zstd
	ChunkEncoding string `json:"chunkEncoding,omitempty"`

	// Max chunk age
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	MaxChunkAge string `json:"maxChunkAge,omitempty"`

	// Flush on shutdown
	// +optional
	FlushOnShutdown bool `json:"flushOnShutdown,omitempty"`

	// Flush check period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	FlushCheckPeriod string `json:"flushCheckPeriod,omitempty"`

	// Concurrent flushes
	// +optional
	// +kubebuilder:validation:Minimum=0
	ConcurrentFlushes int `json:"concurrentFlushes,omitempty"`

	// Max transfer retries
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxTransferRetries int `json:"maxTransferRetries,omitempty"`
}

// WALConfig defines Write-Ahead Log configuration
type WALConfig struct {
	// Enable WAL
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// WAL directory
	// +optional
	Dir string `json:"dir,omitempty"`

	// Checkpoint duration
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	CheckpointDuration string `json:"checkpointDuration,omitempty"`

	// Flush on shutdown
	// +optional
	FlushOnShutdown bool `json:"flushOnShutdown,omitempty"`

	// Replay memory ceiling
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+(\.\d+)?([KMGT]i?)?$`
	ReplayMemoryCeiling string `json:"replayMemoryCeiling,omitempty"`
}

// LifecyclerConfig defines lifecycler configuration
type LifecyclerConfig struct {
	// Ring configuration
	// +optional
	Ring *RingConfig `json:"ring,omitempty"`

	// Number of tokens
	// +optional
	// +kubebuilder:validation:Minimum=1
	NumTokens int `json:"numTokens,omitempty"`

	// Heartbeat period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	HeartbeatPeriod string `json:"heartbeatPeriod,omitempty"`

	// Join after
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	JoinAfter string `json:"joinAfter,omitempty"`

	// Min ready duration
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	MinReadyDuration string `json:"minReadyDuration,omitempty"`

	// Final sleep
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	FinalSleep string `json:"finalSleep,omitempty"`
}

// RingConfig defines hash ring configuration
type RingConfig struct {
	// KV store
	// +optional
	KVStore *KVStoreConfig `json:"kvStore,omitempty"`

	// Replication factor
	// +optional
	// +kubebuilder:validation:Minimum=1
	ReplicationFactor int `json:"replicationFactor,omitempty"`

	// Zone awareness enabled
	// +optional
	ZoneAwarenessEnabled bool `json:"zoneAwarenessEnabled,omitempty"`
}

// KVStoreConfig defines key-value store configuration
type KVStoreConfig struct {
	// Store type
	// +kubebuilder:validation:Enum=consul;etcd;inmemory;memberlist
	Store string `json:"store"`

	// Consul configuration
	// +optional
	Consul *ConsulConfig `json:"consul,omitempty"`

	// Etcd configuration
	// +optional
	Etcd *EtcdConfig `json:"etcd,omitempty"`

	// Prefix
	// +optional
	Prefix string `json:"prefix,omitempty"`
}

// ConsulConfig defines Consul configuration
type ConsulConfig struct {
	// Host address
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// ACL token reference
	// +optional
	ACLToken *corev1.SecretKeySelector `json:"aclToken,omitempty"`

	// HTTP client timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	HTTPClientTimeout string `json:"httpClientTimeout,omitempty"`

	// Consistent reads
	// +optional
	ConsistentReads bool `json:"consistentReads,omitempty"`
}

// EtcdConfig defines Etcd configuration
type EtcdConfig struct {
	// Endpoints
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Endpoints []string `json:"endpoints"`

	// Dial timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	DialTimeout string `json:"dialTimeout,omitempty"`

	// Max retries
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxRetries int `json:"maxRetries,omitempty"`
}

// QuerierConfig defines querier configuration
type QuerierConfig struct {
	// Max concurrent
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxConcurrent int `json:"maxConcurrent,omitempty"`

	// Tail max duration
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	TailMaxDuration string `json:"tailMaxDuration,omitempty"`

	// Query timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	QueryTimeout string `json:"queryTimeout,omitempty"`

	// Query ingester within
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	QueryIngestersWithin string `json:"queryIngestersWithin,omitempty"`

	// Engine configuration
	// +optional
	Engine *EngineConfig `json:"engine,omitempty"`
}

// EngineConfig defines query engine configuration
type EngineConfig struct {
	// Timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	Timeout string `json:"timeout,omitempty"`

	// Max look back period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	MaxLookBackPeriod string `json:"maxLookBackPeriod,omitempty"`
}

// QueryFrontendConfig defines query frontend configuration
type QueryFrontendConfig struct {
	// Compress responses
	// +optional
	CompressResponses bool `json:"compressResponses,omitempty"`

	// Max outstanding requests per tenant
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxOutstandingPerTenant int `json:"maxOutstandingPerTenant,omitempty"`

	// Log queries longer than
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	LogQueriesLongerThan string `json:"logQueriesLongerThan,omitempty"`

	// Downstream URL
	// +optional
	DownstreamURL string `json:"downstreamURL,omitempty"`

	// Split queries by interval
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	SplitQueriesByInterval string `json:"splitQueriesByInterval,omitempty"`

	// Align queries with step
	// +optional
	AlignQueriesWithStep bool `json:"alignQueriesWithStep,omitempty"`

	// Cache results
	// +optional
	CacheResults bool `json:"cacheResults,omitempty"`

	// Max retries
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxRetries int `json:"maxRetries,omitempty"`

	// Parallelise shardable queries
	// +optional
	ParalleliseShardableQueries bool `json:"paralleliseShardableQueries,omitempty"`

	// Scheduler address
	// +optional
	SchedulerAddress string `json:"schedulerAddress,omitempty"`

	// Scheduler DNS lookup period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	SchedulerDNSLookupPeriod string `json:"schedulerDnsLookupPeriod,omitempty"`

	// Scheduler worker concurrency
	// +optional
	// +kubebuilder:validation:Minimum=0
	SchedulerWorkerConcurrency int `json:"schedulerWorkerConcurrency,omitempty"`
}

// CompactorConfig defines compactor configuration
type CompactorConfig struct {
	// Working directory
	// +optional
	WorkingDirectory string `json:"workingDirectory,omitempty"`

	// Shared store
	// +optional
	SharedStore string `json:"sharedStore,omitempty"`

	// Compaction interval
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	CompactionInterval string `json:"compactionInterval,omitempty"`

	// Retention enabled
	// +optional
	RetentionEnabled bool `json:"retentionEnabled,omitempty"`

	// Retention delete delay
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	RetentionDeleteDelay string `json:"retentionDeleteDelay,omitempty"`

	// Retention delete worker count
	// +optional
	// +kubebuilder:validation:Minimum=0
	RetentionDeleteWorkerCount int `json:"retentionDeleteWorkerCount,omitempty"`

	// Delete request cancel period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smhd]$`
	DeleteRequestCancelPeriod string `json:"deleteRequestCancelPeriod,omitempty"`

	// Max compaction parallelism
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxCompactionParallelism int `json:"maxCompactionParallelism,omitempty"`
}

// RulerConfig defines ruler configuration for recording/alerting rules
type RulerConfig struct {
	// Enable API
	// +optional
	EnableAPI bool `json:"enableApi,omitempty"`

	// Enable sharding
	// +optional
	EnableSharding bool `json:"enableSharding,omitempty"`

	// Storage configuration
	// +optional
	Storage *RulerStorageConfig `json:"storage,omitempty"`

	// Rule path
	// +optional
	RulePath string `json:"rulePath,omitempty"`

	// Evaluation interval
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	EvaluationInterval string `json:"evaluationInterval,omitempty"`

	// Poll interval
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	PollInterval string `json:"pollInterval,omitempty"`

	// Alertmanager URL
	// +optional
	AlertmanagerURL string `json:"alertmanagerUrl,omitempty"`

	// External URL
	// +optional
	ExternalURL string `json:"externalUrl,omitempty"`

	// External labels
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// Enable alertmanager v2
	// +optional
	EnableAlertmanagerV2 bool `json:"enableAlertmanagerV2,omitempty"`

	// Alertmanager refresh interval
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	AlertmanagerRefreshInterval string `json:"alertmanagerRefreshInterval,omitempty"`

	// Notification queue capacity
	// +optional
	// +kubebuilder:validation:Minimum=0
	NotificationQueueCapacity int `json:"notificationQueueCapacity,omitempty"`

	// Notification timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	NotificationTimeout string `json:"notificationTimeout,omitempty"`

	// Search pending for
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	SearchPendingFor string `json:"searchPendingFor,omitempty"`

	// Ring configuration
	// +optional
	Ring *RingConfig `json:"ring,omitempty"`

	// Flush period
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	FlushPeriod string `json:"flushPeriod,omitempty"`

	// Enable query stats
	// +optional
	EnableQueryStats bool `json:"enableQueryStats,omitempty"`
}

// RulerStorageConfig defines ruler storage configuration
type RulerStorageConfig struct {
	// Type of storage
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=local;s3;gcs;azure;swift
	Type string `json:"type"`

	// Local storage configuration
	// +optional
	Local *LocalRuleStorageConfig `json:"local,omitempty"`

	// S3 configuration
	// +optional
	S3 *S3StorageConfig `json:"s3,omitempty"`

	// GCS configuration
	// +optional
	GCS *GCSStorageConfig `json:"gcs,omitempty"`

	// Azure configuration
	// +optional
	Azure *AzureStorageConfig `json:"azure,omitempty"`
}

// LocalRuleStorageConfig defines local rule storage configuration
type LocalRuleStorageConfig struct {
	// Directory path
	// +kubebuilder:validation:Required
	Directory string `json:"directory"`
}

// LokiAuthConfig defines authentication configuration
type LokiAuthConfig struct {
	// Type of authentication
	// +kubebuilder:validation:Enum=basic;oidc;header
	Type string `json:"type"`

	// Basic auth configuration
	// +optional
	Basic *BasicAuthConfig `json:"basic,omitempty"`

	// OIDC configuration
	// +optional
	OIDC *OIDCConfig `json:"oidc,omitempty"`

	// Header auth configuration
	// +optional
	Header *HeaderAuthConfig `json:"header,omitempty"`
}

// BasicAuthConfig defines basic authentication configuration
type BasicAuthConfig struct {
	// Username
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// Password reference
	// +kubebuilder:validation:Required
	Password *corev1.SecretKeySelector `json:"password"`
}

// OIDCConfig defines OIDC authentication configuration
type OIDCConfig struct {
	// Issuer URL
	// +kubebuilder:validation:Required
	IssuerURL string `json:"issuerUrl"`

	// Client ID
	// +kubebuilder:validation:Required
	ClientID string `json:"clientId"`

	// Client secret reference
	// +optional
	ClientSecret *corev1.SecretKeySelector `json:"clientSecret,omitempty"`

	// Redirect URL
	// +optional
	RedirectURL string `json:"redirectUrl,omitempty"`

	// Group claim
	// +optional
	GroupClaim string `json:"groupClaim,omitempty"`

	// Username claim
	// +optional
	UsernameClaim string `json:"usernameClaim,omitempty"`
}

// HeaderAuthConfig defines header-based authentication configuration
type HeaderAuthConfig struct {
	// Header name
	// +kubebuilder:validation:Required
	HeaderName string `json:"headerName"`

	// Header value prefix
	// +optional
	HeaderValuePrefix string `json:"headerValuePrefix,omitempty"`
}

// MultiTenancyConfig defines multi-tenancy configuration
type MultiTenancyConfig struct {
	// Enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Authentication enabled
	// +optional
	AuthEnabled bool `json:"authEnabled,omitempty"`

	// Tenant ID header
	// +optional
	// +kubebuilder:default="X-Scope-OrgID"
	TenantIDHeader string `json:"tenantIdHeader,omitempty"`

	// Tenant ID label
	// +optional
	// +kubebuilder:default="tenant_id"
	TenantIDLabel string `json:"tenantIdLabel,omitempty"`

	// Tenant configuration
	// +optional
	Tenants []TenantConfig `json:"tenants,omitempty"`
}

// TenantConfig defines per-tenant configuration
type TenantConfig struct {
	// Tenant ID
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// Display name
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Limits override
	// +optional
	Limits *LimitsConfig `json:"limits,omitempty"`
}

// ServerConfig defines server configuration
type ServerConfig struct {
	// HTTP listen port
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=3100
	HTTPListenPort int `json:"httpListenPort,omitempty"`

	// gRPC listen port
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=9095
	GRPCListenPort int `json:"grpcListenPort,omitempty"`

	// Register instrumentation
	// +optional
	RegisterInstrumentation bool `json:"registerInstrumentation,omitempty"`

	// Log level
	// +optional
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// +kubebuilder:default="info"
	LogLevel string `json:"logLevel,omitempty"`

	// Log format
	// +optional
	// +kubebuilder:validation:Enum=json;logfmt
	// +kubebuilder:default="logfmt"
	LogFormat string `json:"logFormat,omitempty"`

	// gRPC server max receive message size
	// +optional
	// +kubebuilder:validation:Minimum=0
	GRPCServerMaxRecvMsgSize int `json:"grpcServerMaxRecvMsgSize,omitempty"`

	// gRPC server max send message size
	// +optional
	// +kubebuilder:validation:Minimum=0
	GRPCServerMaxSendMsgSize int `json:"grpcServerMaxSendMsgSize,omitempty"`

	// gRPC server max concurrent streams
	// +optional
	// +kubebuilder:validation:Minimum=0
	GRPCServerMaxConcurrentStreams int `json:"grpcServerMaxConcurrentStreams,omitempty"`

	// gRPC server max connection idle
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	GRPCServerMaxConnectionIdle string `json:"grpcServerMaxConnectionIdle,omitempty"`

	// gRPC server max connection age
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	GRPCServerMaxConnectionAge string `json:"grpcServerMaxConnectionAge,omitempty"`

	// gRPC server max connection age grace
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	GRPCServerMaxConnectionAgeGrace string `json:"grpcServerMaxConnectionAgeGrace,omitempty"`

	// gRPC server keepalive time
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	GRPCServerKeepaliveTime string `json:"grpcServerKeepaliveTime,omitempty"`

	// gRPC server keepalive timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	GRPCServerKeepaliveTimeout string `json:"grpcServerKeepaliveTimeout,omitempty"`

	// HTTP server read timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	HTTPServerReadTimeout string `json:"httpServerReadTimeout,omitempty"`

	// HTTP server write timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	HTTPServerWriteTimeout string `json:"httpServerWriteTimeout,omitempty"`

	// HTTP server idle timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[smh]$`
	HTTPServerIdleTimeout string `json:"httpServerIdleTimeout,omitempty"`
}

// LokiConfigStatus defines the observed state of LokiConfig
type LokiConfigStatus struct {
	// Phase indicates the current state of the Loki configuration
	// +optional
	// +kubebuilder:validation:Enum=Pending;Validating;Ready;Failed
	Phase string `json:"phase,omitempty"`

	// Message provides additional information about the current phase
	// +optional
	Message string `json:"message,omitempty"`

	// LastUpdated indicates when the configuration was last processed
	// +optional
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// ConfigHash represents a hash of the current configuration
	// +optional
	ConfigHash string `json:"configHash,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed spec
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ValidationErrors contains any validation errors found
	// +optional
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// Applied indicates whether the configuration has been applied
	// +optional
	Applied bool `json:"applied,omitempty"`

	// AppliedAt indicates when the configuration was last applied
	// +optional
	AppliedAt *metav1.Time `json:"appliedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=lc;lokiconf
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.storage.type`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LokiConfig is the Schema for the lokiconfigs API
type LokiConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LokiConfigSpec   `json:"spec,omitempty"`
	Status LokiConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LokiConfigList contains a list of LokiConfig
type LokiConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LokiConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LokiConfig{}, &LokiConfigList{})
}
