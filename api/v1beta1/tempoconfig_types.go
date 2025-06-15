/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TempoConfigSpec defines the desired state of TempoConfig
type TempoConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// TargetPlatform references the ObservabilityPlatform this config applies to
	TargetPlatform corev1.LocalObjectReference `json:"targetPlatform"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Paused indicates whether this configuration should be applied
	Paused bool `json:"paused,omitempty"`

	// +kubebuilder:validation:Optional
	// Storage defines storage backend configuration
	Storage *TempoStorageConfig `json:"storage,omitempty"`

	// +kubebuilder:validation:Optional
	// Distributor defines distributor configuration
	Distributor *DistributorConfig `json:"distributor,omitempty"`

	// +kubebuilder:validation:Optional
	// Ingester defines ingester configuration
	Ingester *IngesterConfig `json:"ingester,omitempty"`

	// +kubebuilder:validation:Optional
	// Compactor defines compactor configuration
	Compactor *CompactorConfig `json:"compactor,omitempty"`

	// +kubebuilder:validation:Optional
	// Querier defines querier configuration
	Querier *QuerierConfig `json:"querier,omitempty"`

	// +kubebuilder:validation:Optional
	// QueryFrontend defines query frontend configuration
	QueryFrontend *QueryFrontendConfig `json:"queryFrontend,omitempty"`

	// +kubebuilder:validation:Optional
	// Receivers defines trace receivers configuration
	Receivers *ReceiversConfig `json:"receivers,omitempty"`

	// +kubebuilder:validation:Optional
	// Overrides defines per-tenant overrides
	Overrides *OverridesConfig `json:"overrides,omitempty"`

	// +kubebuilder:validation:Optional
	// GlobalConfig defines global Tempo configuration
	GlobalConfig *GlobalTempoConfig `json:"globalConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// SearchConfig defines search configuration
	SearchConfig *SearchConfig `json:"searchConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// MetricsGenerator defines metrics generation configuration
	MetricsGenerator *MetricsGeneratorConfig `json:"metricsGenerator,omitempty"`

	// +kubebuilder:validation:Optional
	// MultiTenancy defines multi-tenancy configuration
	MultiTenancy *TempoMultiTenancyConfig `json:"multiTenancy,omitempty"`

	// +kubebuilder:validation:Optional
	// AuthConfig defines authentication configuration
	AuthConfig *TempoAuthConfig `json:"authConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// RateLimiting defines rate limiting configuration
	RateLimiting *TempoRateLimitingConfig `json:"rateLimiting,omitempty"`
}

// TempoStorageConfig defines storage backend configuration
type TempoStorageConfig struct {
	// +kubebuilder:validation:Optional
	// Trace defines trace storage configuration
	Trace *TraceStorageConfig `json:"trace,omitempty"`

	// +kubebuilder:validation:Optional
	// WAL defines Write-Ahead Log configuration
	WAL *TempoWALConfig `json:"wal,omitempty"`

	// +kubebuilder:validation:Optional
	// Cache defines caching configuration
	Cache *TempoCacheConfig `json:"cache,omitempty"`
}

// TraceStorageConfig defines trace storage configuration
type TraceStorageConfig struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=s3;gcs;azure;filesystem;swift
	// Backend type for trace storage
	Backend string `json:"backend"`

	// +kubebuilder:validation:Optional
	// S3 configuration (if backend is s3)
	S3 *TempoS3Config `json:"s3,omitempty"`

	// +kubebuilder:validation:Optional
	// GCS configuration (if backend is gcs)
	GCS *TempoGCSConfig `json:"gcs,omitempty"`

	// +kubebuilder:validation:Optional
	// Azure configuration (if backend is azure)
	Azure *TempoAzureConfig `json:"azure,omitempty"`

	// +kubebuilder:validation:Optional
	// Filesystem configuration (if backend is filesystem)
	Filesystem *TempoFilesystemConfig `json:"filesystem,omitempty"`

	// +kubebuilder:validation:Optional
	// Swift configuration (if backend is swift)
	Swift *TempoSwiftConfig `json:"swift,omitempty"`

	// +kubebuilder:validation:Optional
	// Pool defines connection pool settings
	Pool *TempoPoolConfig `json:"pool,omitempty"`

	// +kubebuilder:validation:Optional
	// BackendConfig contains backend-specific configuration
	BackendConfig map[string]string `json:"backendConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// BlocklistPoll defines blocklist polling interval
	BlocklistPoll string `json:"blocklistPoll,omitempty"`

	// +kubebuilder:validation:Optional
	// BlocklistPollConcurrency defines polling concurrency
	BlocklistPollConcurrency int32 `json:"blocklistPollConcurrency,omitempty"`
}

// TempoS3Config defines S3 storage configuration
type TempoS3Config struct {
	// +kubebuilder:validation:Required
	// Bucket name
	Bucket string `json:"bucket"`

	// +kubebuilder:validation:Optional
	// Endpoint for S3 API
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// Region for S3
	Region string `json:"region,omitempty"`

	// +kubebuilder:validation:Optional
	// AccessKey for authentication
	AccessKey string `json:"accessKey,omitempty"`

	// +kubebuilder:validation:Optional
	// SecretKey for authentication
	SecretKey corev1.SecretKeySelector `json:"secretKey,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Insecure disables TLS
	Insecure bool `json:"insecure,omitempty"`

	// +kubebuilder:validation:Optional
	// SignatureVersion for S3
	SignatureVersion string `json:"signatureVersion,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// ForcePathStyle for S3 URLs
	ForcePathStyle bool `json:"forcePathStyle,omitempty"`

	// +kubebuilder:validation:Optional
	// StorageClass for objects
	StorageClass string `json:"storageClass,omitempty"`

	// +kubebuilder:validation:Optional
	// SSE defines server-side encryption
	SSE *S3SSEConfig `json:"sse,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags to add to objects
	Tags map[string]string `json:"tags,omitempty"`
}

// S3SSEConfig defines S3 server-side encryption
type S3SSEConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=SSE-S3;SSE-KMS
	// Type of encryption
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// KMSKeyID for SSE-KMS
	KMSKeyID string `json:"kmsKeyId,omitempty"`

	// +kubebuilder:validation:Optional
	// KMSEncryptionContext for SSE-KMS
	KMSEncryptionContext string `json:"kmsEncryptionContext,omitempty"`
}

// TempoGCSConfig defines Google Cloud Storage configuration
type TempoGCSConfig struct {
	// +kubebuilder:validation:Required
	// BucketName for GCS
	BucketName string `json:"bucketName"`

	// +kubebuilder:validation:Optional
	// ChunkBufferSize for uploads
	ChunkBufferSize int32 `json:"chunkBufferSize,omitempty"`

	// +kubebuilder:validation:Optional
	// RequestTimeout for GCS operations
	RequestTimeout string `json:"requestTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// ServiceAccount for authentication
	ServiceAccount corev1.SecretKeySelector `json:"serviceAccount,omitempty"`

	// +kubebuilder:validation:Optional
	// ObjectCacheControl for cache headers
	ObjectCacheControl string `json:"objectCacheControl,omitempty"`
}

// TempoAzureConfig defines Azure Blob Storage configuration
type TempoAzureConfig struct {
	// +kubebuilder:validation:Required
	// ContainerName for Azure
	ContainerName string `json:"containerName"`

	// +kubebuilder:validation:Required
	// AccountName for Azure
	AccountName string `json:"accountName"`

	// +kubebuilder:validation:Optional
	// AccountKey for authentication
	AccountKey corev1.SecretKeySelector `json:"accountKey,omitempty"`

	// +kubebuilder:validation:Optional
	// ConnectionString for Azure
	ConnectionString corev1.SecretKeySelector `json:"connectionString,omitempty"`

	// +kubebuilder:validation:Optional
	// EndpointSuffix for Azure
	EndpointSuffix string `json:"endpointSuffix,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// UseAzureAD for authentication
	UseAzureAD bool `json:"useAzureAD,omitempty"`

	// +kubebuilder:validation:Optional
	// ClientID for Azure AD
	ClientID string `json:"clientId,omitempty"`

	// +kubebuilder:validation:Optional
	// ClientSecret for Azure AD
	ClientSecret corev1.SecretKeySelector `json:"clientSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// TenantID for Azure AD
	TenantID string `json:"tenantId,omitempty"`
}

// TempoFilesystemConfig defines filesystem storage configuration
type TempoFilesystemConfig struct {
	// +kubebuilder:validation:Required
	// Path for storage
	Path string `json:"path"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// CreateDirectories automatically
	CreateDirectories bool `json:"createDirectories,omitempty"`

	// +kubebuilder:validation:Optional
	// Permissions for created directories
	Permissions string `json:"permissions,omitempty"`
}

// TempoSwiftConfig defines OpenStack Swift configuration
type TempoSwiftConfig struct {
	// +kubebuilder:validation:Required
	// AuthURL for Swift
	AuthURL string `json:"authUrl"`

	// +kubebuilder:validation:Required
	// Username for authentication
	Username string `json:"username"`

	// +kubebuilder:validation:Required
	// Password for authentication
	Password corev1.SecretKeySelector `json:"password"`

	// +kubebuilder:validation:Required
	// ContainerName for Swift
	ContainerName string `json:"containerName"`

	// +kubebuilder:validation:Optional
	// ProjectName for Swift
	ProjectName string `json:"projectName,omitempty"`

	// +kubebuilder:validation:Optional
	// UserDomainName for Swift
	UserDomainName string `json:"userDomainName,omitempty"`

	// +kubebuilder:validation:Optional
	// ProjectDomainName for Swift
	ProjectDomainName string `json:"projectDomainName,omitempty"`

	// +kubebuilder:validation:Optional
	// RegionName for Swift
	RegionName string `json:"regionName,omitempty"`
}

// TempoPoolConfig defines connection pool configuration
type TempoPoolConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100
	// MaxWorkers for parallel operations
	MaxWorkers int32 `json:"maxWorkers,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// QueueDepth for operations
	QueueDepth int32 `json:"queueDepth,omitempty"`
}

// TempoWALConfig defines Write-Ahead Log configuration
type TempoWALConfig struct {
	// +kubebuilder:validation:Optional
	// Path for WAL storage
	Path string `json:"path,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// CompletedFilesCleanupAge for cleanup
	CompletedFilesCleanupAge string `json:"completedFilesCleanupAge,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// IngesterFlushOpTimeout enables timeout
	IngesterFlushOpTimeout bool `json:"ingesterFlushOpTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// IngesterFlushOpTimeoutDuration duration
	IngesterFlushOpTimeoutDuration string `json:"ingesterFlushOpTimeoutDuration,omitempty"`
}

// TempoCacheConfig defines caching configuration
type TempoCacheConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=memcached;redis;inmemory
	// Type of cache backend
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// Memcached configuration
	Memcached *MemcachedConfig `json:"memcached,omitempty"`

	// +kubebuilder:validation:Optional
	// Redis configuration
	Redis *RedisConfig `json:"redis,omitempty"`

	// +kubebuilder:validation:Optional
	// InMemory configuration
	InMemory *InMemoryCacheConfig `json:"inmemory,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1h"
	// TTL for cache entries
	TTL string `json:"ttl,omitempty"`
}

// MemcachedConfig defines Memcached configuration
type MemcachedConfig struct {
	// +kubebuilder:validation:Required
	// Host addresses
	Host string `json:"host"`

	// +kubebuilder:validation:Optional
	// Service name for DNS discovery
	Service string `json:"service,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="2s"
	// Timeout for operations
	Timeout string `json:"timeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100
	// MaxIdleConns in pool
	MaxIdleConns int32 `json:"maxIdleConns,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=16
	// MaxItemSize in MB
	MaxItemSize int32 `json:"maxItemSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// UpdateInterval for DNS
	UpdateInterval string `json:"updateInterval,omitempty"`
}

// RedisConfig defines Redis configuration
type RedisConfig struct {
	// +kubebuilder:validation:Required
	// Endpoint for Redis
	Endpoint string `json:"endpoint"`

	// +kubebuilder:validation:Optional
	// Username for authentication
	Username string `json:"username,omitempty"`

	// +kubebuilder:validation:Optional
	// Password for authentication
	Password corev1.SecretKeySelector `json:"password,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0
	// DB number
	DB int32 `json:"db,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5s"
	// Timeout for operations
	Timeout string `json:"timeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// MaxConnectionAge in minutes
	MaxConnectionAge string `json:"maxConnectionAge,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSConfigSpec `json:"tls,omitempty"`
}

// InMemoryCacheConfig defines in-memory cache configuration
type InMemoryCacheConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=50000
	// MaxSize entries
	MaxSize int32 `json:"maxSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1h"
	// TTL for entries
	TTL string `json:"ttl,omitempty"`
}

// DistributorConfig defines distributor configuration
type DistributorConfig struct {
	// +kubebuilder:validation:Optional
	// Receivers configuration
	Receivers *ReceiversConfig `json:"receivers,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=4096
	// LogReceivedTraces enables trace logging
	LogReceivedTraces bool `json:"logReceivedTraces,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// QueueSize for processing
	QueueSize int32 `json:"queueSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100
	// Workers for processing
	Workers int32 `json:"workers,omitempty"`

	// +kubebuilder:validation:Optional
	// ForwarderConfig for forwarding traces
	ForwarderConfig *ForwarderConfig `json:"forwarderConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ExtendedSearch configuration
	ExtendedSearch *ExtendedSearchConfig `json:"extendedSearch,omitempty"`
}

// ReceiversConfig defines trace receivers configuration
type ReceiversConfig struct {
	// +kubebuilder:validation:Optional
	// OTLP receiver configuration
	OTLP *OTLPReceiverConfig `json:"otlp,omitempty"`

	// +kubebuilder:validation:Optional
	// Jaeger receiver configuration
	Jaeger *JaegerReceiverConfig `json:"jaeger,omitempty"`

	// +kubebuilder:validation:Optional
	// Zipkin receiver configuration
	Zipkin *ZipkinReceiverConfig `json:"zipkin,omitempty"`

	// +kubebuilder:validation:Optional
	// OpenCensus receiver configuration
	OpenCensus *OpenCensusReceiverConfig `json:"opencensus,omitempty"`

	// +kubebuilder:validation:Optional
	// Kafka receiver configuration
	Kafka *KafkaReceiverConfig `json:"kafka,omitempty"`
}

// OTLPReceiverConfig defines OTLP receiver configuration
type OTLPReceiverConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled flag for OTLP receiver
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPC configuration
	GRPC *OTLPGRPCConfig `json:"grpc,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTP configuration
	HTTP *OTLPHTTPConfig `json:"http,omitempty"`
}

// OTLPGRPCConfig defines OTLP gRPC configuration
type OTLPGRPCConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0:4317"
	// Endpoint to listen on
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSServerConfig `json:"tls,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=65536
	// MaxRecvMsgSize in KB
	MaxRecvMsgSize int32 `json:"maxRecvMsgSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=4
	// MaxConcurrentStreams allowed
	MaxConcurrentStreams int32 `json:"maxConcurrentStreams,omitempty"`

	// +kubebuilder:validation:Optional
	// Headers to add
	Headers map[string]string `json:"headers,omitempty"`
}

// OTLPHTTPConfig defines OTLP HTTP configuration
type OTLPHTTPConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0:4318"
	// Endpoint to listen on
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSServerConfig `json:"tls,omitempty"`

	// +kubebuilder:validation:Optional
	// CORS configuration
	CORS *CORSConfig `json:"cors,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100
	// MaxRequestBodySize in MB
	MaxRequestBodySize int32 `json:"maxRequestBodySize,omitempty"`

	// +kubebuilder:validation:Optional
	// Headers to add
	Headers map[string]string `json:"headers,omitempty"`
}

// TLSServerConfig defines TLS server configuration
type TLSServerConfig struct {
	// +kubebuilder:validation:Optional
	// CertFile path
	CertFile string `json:"certFile,omitempty"`

	// +kubebuilder:validation:Optional
	// KeyFile path
	KeyFile string `json:"keyFile,omitempty"`

	// +kubebuilder:validation:Optional
	// ClientCAFile for mTLS
	ClientCAFile string `json:"clientCAFile,omitempty"`

	// +kubebuilder:validation:Optional
	// MinVersion of TLS
	MinVersion string `json:"minVersion,omitempty"`

	// +kubebuilder:validation:Optional
	// CipherSuites allowed
	CipherSuites []string `json:"cipherSuites,omitempty"`
}

// CORSConfig defines CORS configuration
type CORSConfig struct {
	// +kubebuilder:validation:Optional
	// AllowedOrigins for CORS
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`

	// +kubebuilder:validation:Optional
	// AllowedHeaders for CORS
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=86400
	// MaxAge in seconds
	MaxAge int32 `json:"maxAge,omitempty"`
}

// JaegerReceiverConfig defines Jaeger receiver configuration
type JaegerReceiverConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled flag for Jaeger receiver
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// RemoteSampling configuration
	RemoteSampling *RemoteSamplingConfig `json:"remoteSampling,omitempty"`

	// +kubebuilder:validation:Optional
	// Protocols configuration
	Protocols *JaegerProtocolsConfig `json:"protocols,omitempty"`
}

// RemoteSamplingConfig defines remote sampling configuration
type RemoteSamplingConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="http://localhost:5778/sampling"
	// HostEndpoint for sampling
	HostEndpoint string `json:"hostEndpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="jaeger-collector"
	// ServiceName for sampling
	ServiceName string `json:"serviceName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="60s"
	// RefreshInterval for strategies
	RefreshInterval string `json:"refreshInterval,omitempty"`
}

// JaegerProtocolsConfig defines Jaeger protocols configuration
type JaegerProtocolsConfig struct {
	// +kubebuilder:validation:Optional
	// GRPC configuration
	GRPC *JaegerGRPCProtocol `json:"grpc,omitempty"`

	// +kubebuilder:validation:Optional
	// ThriftHTTP configuration
	ThriftHTTP *JaegerThriftHTTPProtocol `json:"thriftHttp,omitempty"`

	// +kubebuilder:validation:Optional
	// ThriftBinary configuration
	ThriftBinary *JaegerThriftBinaryProtocol `json:"thriftBinary,omitempty"`

	// +kubebuilder:validation:Optional
	// ThriftCompact configuration
	ThriftCompact *JaegerThriftCompactProtocol `json:"thriftCompact,omitempty"`
}

// JaegerGRPCProtocol defines Jaeger gRPC protocol configuration
type JaegerGRPCProtocol struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0:14250"
	// Endpoint to listen on
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSServerConfig `json:"tls,omitempty"`
}

// JaegerThriftHTTPProtocol defines Jaeger Thrift HTTP protocol configuration
type JaegerThriftHTTPProtocol struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0:14268"
	// Endpoint to listen on
	Endpoint string `json:"endpoint,omitempty"`
}

// JaegerThriftBinaryProtocol defines Jaeger Thrift Binary protocol configuration
type JaegerThriftBinaryProtocol struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0:6832"
	// Endpoint to listen on
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// QueueSize for processing
	QueueSize int32 `json:"queueSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// Workers for processing
	Workers int32 `json:"workers,omitempty"`
}

// JaegerThriftCompactProtocol defines Jaeger Thrift Compact protocol configuration
type JaegerThriftCompactProtocol struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0:6831"
	// Endpoint to listen on
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// QueueSize for processing
	QueueSize int32 `json:"queueSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// Workers for processing
	Workers int32 `json:"workers,omitempty"`
}

// ZipkinReceiverConfig defines Zipkin receiver configuration
type ZipkinReceiverConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled flag for Zipkin receiver
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0:9411"
	// Endpoint to listen on
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// ParseStringTags enables string tag parsing
	ParseStringTags bool `json:"parseStringTags,omitempty"`
}

// OpenCensusReceiverConfig defines OpenCensus receiver configuration
type OpenCensusReceiverConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled flag for OpenCensus receiver
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0:55678"
	// Endpoint to listen on
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSServerConfig `json:"tls,omitempty"`
}

// KafkaReceiverConfig defines Kafka receiver configuration
type KafkaReceiverConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled flag for Kafka receiver
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Required
	// Brokers list
	Brokers []string `json:"brokers"`

	// +kubebuilder:validation:Required
	// Topic to consume from
	Topic string `json:"topic"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="tempo"
	// GroupID for consumer group
	GroupID string `json:"groupId,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="jaeger"
	// Encoding of messages
	Encoding string `json:"encoding,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSConfigSpec `json:"tls,omitempty"`

	// +kubebuilder:validation:Optional
	// SASL configuration
	SASL *SASLConfig `json:"sasl,omitempty"`
}

// SASLConfig defines SASL configuration
type SASLConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled flag for SASL
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=plain;scram-sha-256;scram-sha-512
	// Mechanism for SASL
	Mechanism string `json:"mechanism,omitempty"`

	// +kubebuilder:validation:Optional
	// Username for SASL
	Username string `json:"username,omitempty"`

	// +kubebuilder:validation:Optional
	// Password for SASL
	Password corev1.SecretKeySelector `json:"password,omitempty"`
}

// ForwarderConfig defines trace forwarding configuration
type ForwarderConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// Workers for forwarding
	Workers int32 `json:"workers,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100
	// QueueSize for forwarding
	QueueSize int32 `json:"queueSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	// RetryBackoff duration
	RetryBackoff string `json:"retryBackoff,omitempty"`
}

// ExtendedSearchConfig defines extended search configuration
type ExtendedSearchConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// MinDuration for search in microseconds
	MinDuration int64 `json:"minDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0
	// MaxDuration for search in microseconds (0 = no limit)
	MaxDuration int64 `json:"maxDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags to index
	Tags []string `json:"tags,omitempty"`
}

// IngesterConfig defines ingester configuration
type IngesterConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// CompleteBlockTimeout for block completion
	CompleteBlockTimeout string `json:"completeBlockTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100000
	// MaxBlockDuration in traces
	MaxBlockDuration int64 `json:"maxBlockDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=500000000
	// MaxBlockBytes size
	MaxBlockBytes int64 `json:"maxBlockBytes,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// FlushCheckPeriod interval
	FlushCheckPeriod string `json:"flushCheckPeriod,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	// FlushOpTimeout duration
	FlushOpTimeout string `json:"flushOpTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// MaxTracesPerBlock limit
	MaxTracesPerBlock int32 `json:"maxTracesPerBlock,omitempty"`

	// +kubebuilder:validation:Optional
	// LifecyclerConfig for ingester
	LifecyclerConfig *LifecyclerConfig `json:"lifecyclerConfig,omitempty"`
}

// LifecyclerConfig defines ingester lifecycler configuration
type LifecyclerConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// HeartbeatPeriod interval
	HeartbeatPeriod string `json:"heartbeatPeriod,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=3
	// NumTokens for the ring
	NumTokens int32 `json:"numTokens,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0s"
	// JoinAfter delay
	JoinAfter string `json:"joinAfter,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// MinReadyDuration before serving
	MinReadyDuration string `json:"minReadyDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// Interface names to use
	InterfaceNames []string `json:"interfaceNames,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0s"
	// FinalSleep before shutdown
	FinalSleep string `json:"finalSleep,omitempty"`

	// +kubebuilder:validation:Optional
	// Address to advertise
	Address string `json:"address,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=7946
	// Port to advertise
	Port int32 `json:"port,omitempty"`
}

// CompactorConfig defines compactor configuration
type CompactorConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1h"
	// BlockRetention duration
	BlockRetention string `json:"blockRetention,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="14d"
	// CompactedBlockRetention duration
	CompactedBlockRetention string `json:"compactedBlockRetention,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// CompactionWindow duration
	CompactionWindow string `json:"compactionWindow,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=4
	// MaxCompactionObjects per operation
	MaxCompactionObjects int32 `json:"maxCompactionObjects,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100000000
	// MaxBlockBytes for compacted blocks
	MaxBlockBytes int64 `json:"maxBlockBytes,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// BlockSyncDelay duration
	BlockSyncDelay string `json:"blockSyncDelay,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="48h"
	// RetentionConcurrency duration
	RetentionConcurrency int32 `json:"retentionConcurrency,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// MaxTimePerTenant processing limit
	MaxTimePerTenant string `json:"maxTimePerTenant,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=3
	// CompactionCycle interval
	CompactionCycle string `json:"compactionCycle,omitempty"`
}

// QuerierConfig defines querier configuration
type QuerierConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=2
	// ConcurrentRequests allowed
	ConcurrentRequests int32 `json:"concurrentRequests,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// SearchDefaultResultLimit for queries
	SearchDefaultResultLimit int32 `json:"searchDefaultResultLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="48h"
	// SearchMaxDuration for queries
	SearchMaxDuration string `json:"searchMaxDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// SearchMaxResultLimit for queries
	SearchMaxResultLimit int32 `json:"searchMaxResultLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=3
	// SearchExternalHedgeRequestsAt attempts
	SearchExternalHedgeRequestsAt int32 `json:"searchExternalHedgeRequestsAt,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="8s"
	// SearchExternalHedgeRequestsUpTo duration
	SearchExternalHedgeRequestsUpTo string `json:"searchExternalHedgeRequestsUpTo,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// MaxConcurrentQueries allowed
	MaxConcurrentQueries int32 `json:"maxConcurrentQueries,omitempty"`

	// +kubebuilder:validation:Optional
	// Frontend configuration
	Frontend *FrontendClientConfig `json:"frontend,omitempty"`
}

// FrontendClientConfig defines frontend client configuration
type FrontendClientConfig struct {
	// +kubebuilder:validation:Optional
	// Address of the frontend
	Address string `json:"address,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPC client configuration
	GRPC *GRPCClientConfig `json:"grpc,omitempty"`
}

// GRPCClientConfig defines gRPC client configuration
type GRPCClientConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Insecure connection
	Insecure bool `json:"insecure,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSConfigSpec `json:"tls,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="2m"
	// Timeout for requests
	Timeout string `json:"timeout,omitempty"`
}

// QueryFrontendConfig defines query frontend configuration
type QueryFrontendConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="48h"
	// MaxRetries for queries
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// +kubebuilder:validation:Optional
	// Search configuration
	Search *SearchSpec `json:"search,omitempty"`

	// +kubebuilder:validation:Optional
	// TraceByID configuration
	TraceByID *TraceByIDConfig `json:"traceById,omitempty"`

	// +kubebuilder:validation:Optional
	// Metrics configuration
	Metrics *MetricsConfig `json:"metrics,omitempty"`
}

// SearchSpec defines search configuration for query frontend
type SearchSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=300
	// ConcurrentJobs for search
	ConcurrentJobs int32 `json:"concurrentJobs,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// TargetBytesPerJob for sharding
	TargetBytesPerJob int64 `json:"targetBytesPerJob,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// DefaultResultLimit for queries
	DefaultResultLimit int32 `json:"defaultResultLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// MaxResultLimit for queries
	MaxResultLimit int32 `json:"maxResultLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="48h"
	// MaxDuration for queries
	MaxDuration string `json:"maxDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// QueryShards for parallelization
	QueryShards int32 `json:"queryShards,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=2
	// HedgeRequestsAt attempts
	HedgeRequestsAt int32 `json:"hedgeRequestsAt,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="8s"
	// HedgeRequestsUpTo duration
	HedgeRequestsUpTo string `json:"hedgeRequestsUpTo,omitempty"`
}

// TraceByIDConfig defines trace by ID query configuration
type TraceByIDConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// ConcurrentRequests allowed
	ConcurrentRequests int32 `json:"concurrentRequests,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=2
	// HedgeRequestsAt attempts
	HedgeRequestsAt int32 `json:"hedgeRequestsAt,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="8s"
	// HedgeRequestsUpTo duration
	HedgeRequestsUpTo string `json:"hedgeRequestsUpTo,omitempty"`

	// +kubebuilder:validation:Optional
	// QueryShards for parallelization
	QueryShards int32 `json:"queryShards,omitempty"`
}

// MetricsConfig defines metrics query configuration
type MetricsConfig struct {
	// +kubebuilder:validation:Optional
	// PrometheusEndpoint for querying
	PrometheusEndpoint string `json:"prometheusEndpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// ConcurrentRequests allowed
	ConcurrentRequests int32 `json:"concurrentRequests,omitempty"`

	// +kubebuilder:validation:Optional
	// TargetBytesPerRequest for sharding
	TargetBytesPerRequest int64 `json:"targetBytesPerRequest,omitempty"`
}

// GlobalTempoConfig defines global Tempo configuration
type GlobalTempoConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="info"
	// LogLevel for Tempo
	LogLevel string `json:"logLevel,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="logfmt"
	// LogFormat for Tempo
	LogFormat string `json:"logFormat,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTPListenPort for internal server
	HTTPListenPort int32 `json:"httpListenPort,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPCListenPort for internal server
	GRPCListenPort int32 `json:"grpcListenPort,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=3100
	// HTTPListenAddress for internal server
	HTTPListenAddress string `json:"httpListenAddress,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=9095
	// GRPCListenAddress for internal server
	GRPCListenAddress string `json:"grpcListenAddress,omitempty"`

	// +kubebuilder:validation:Optional
	// RegisterInstrumentation enables metrics
	RegisterInstrumentation bool `json:"registerInstrumentation,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// GracefulShutdownTimeout duration
	GracefulShutdownTimeout string `json:"gracefulShutdownTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// LogSourceIPsEnabled enables source IP logging
	LogSourceIPsEnabled bool `json:"logSourceIpsEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	// LogSourceIPsHeader for source IP
	LogSourceIPsHeader string `json:"logSourceIpsHeader,omitempty"`

	// +kubebuilder:validation:Optional
	// LogSourceIPsRegex for filtering
	LogSourceIPsRegex string `json:"logSourceIpsRegex,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=4194304
	// GRPCServerMaxRecvMsgSize in bytes
	GRPCServerMaxRecvMsgSize int64 `json:"grpcServerMaxRecvMsgSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=4194304
	// GRPCServerMaxSendMsgSize in bytes
	GRPCServerMaxSendMsgSize int64 `json:"grpcServerMaxSendMsgSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=150
	// GRPCServerMaxConcurrentStreams allowed
	GRPCServerMaxConcurrentStreams int32 `json:"grpcServerMaxConcurrentStreams,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPCServerMaxConnectionIdle duration
	GRPCServerMaxConnectionIdle string `json:"grpcServerMaxConnectionIdle,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPCServerMaxConnectionAge duration
	GRPCServerMaxConnectionAge string `json:"grpcServerMaxConnectionAge,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPCServerMaxConnectionAgeGrace duration
	GRPCServerMaxConnectionAgeGrace string `json:"grpcServerMaxConnectionAgeGrace,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPCServerKeepaliveTime duration
	GRPCServerKeepaliveTime string `json:"grpcServerKeepaliveTime,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPCServerKeepaliveTimeout duration
	GRPCServerKeepaliveTimeout string `json:"grpcServerKeepaliveTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	// HTTPServerReadTimeout duration
	HTTPServerReadTimeout string `json:"httpServerReadTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	// HTTPServerWriteTimeout duration
	HTTPServerWriteTimeout string `json:"httpServerWriteTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// HTTPServerIdleTimeout duration
	HTTPServerIdleTimeout string `json:"httpServerIdleTimeout,omitempty"`
}

// SearchConfig defines search configuration
type SearchConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled flag for search
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1800
	// MaxConcurrentIndexJobs allowed
	MaxConcurrentIndexJobs int32 `json:"maxConcurrentIndexJobs,omitempty"`

	// +kubebuilder:validation:Optional
	// CompleteBlockTimeout for indexing
	CompleteBlockTimeout string `json:"completeBlockTimeout,omitempty"`
}

// MetricsGeneratorConfig defines metrics generation configuration
type MetricsGeneratorConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled flag for metrics generation
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// RingConfig for metrics generator
	RingConfig *RingConfig `json:"ringConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Processor configuration
	Processor *ProcessorConfig `json:"processor,omitempty"`

	// +kubebuilder:validation:Optional
	// Storage configuration
	Storage *MetricsStorageConfig `json:"storage,omitempty"`

	// +kubebuilder:validation:Optional
	// Registry configuration
	Registry *RegistryConfig `json:"registry,omitempty"`

	// +kubebuilder:validation:Optional
	// MetricsIngestionSlack duration
	MetricsIngestionSlack string `json:"metricsIngestionSlack,omitempty"`

	// +kubebuilder:validation:Optional
	// RemoteWriteHeaders for metrics
	RemoteWriteHeaders map[string]string `json:"remoteWriteHeaders,omitempty"`
}

// RingConfig defines hash ring configuration
type RingConfig struct {
	// +kubebuilder:validation:Optional
	// KVStore configuration
	KVStore *KVStoreConfig `json:"kvStore,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// HeartbeatTimeout duration
	HeartbeatTimeout string `json:"heartbeatTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=3
	// ReplicationFactor for data
	ReplicationFactor int32 `json:"replicationFactor,omitempty"`
}

// KVStoreConfig defines key-value store configuration
type KVStoreConfig struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=consul;etcd;inmemory;memberlist
	// Store type
	Store string `json:"store"`

	// +kubebuilder:validation:Optional
	// Prefix for keys
	Prefix string `json:"prefix,omitempty"`

	// +kubebuilder:validation:Optional
	// ConsulConfig if using consul
	ConsulConfig *ConsulConfig `json:"consulConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ETCDConfig if using etcd
	ETCDConfig *ETCDConfig `json:"etcdConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// MemberlistConfig if using memberlist
	MemberlistConfig *MemberlistConfig `json:"memberlistConfig,omitempty"`
}

// ConsulConfig defines Consul configuration
type ConsulConfig struct {
	// +kubebuilder:validation:Required
	// Host address
	Host string `json:"host"`

	// +kubebuilder:validation:Optional
	// ACL token
	ACLToken string `json:"aclToken,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="20s"
	// HTTPClientTimeout duration
	HTTPClientTimeout string `json:"httpClientTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=8
	// ConsistentReads enabled
	ConsistentReads bool `json:"consistentReads,omitempty"`
}

// ETCDConfig defines etcd configuration
type ETCDConfig struct {
	// +kubebuilder:validation:Required
	// Endpoints list
	Endpoints []string `json:"endpoints"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	// DialTimeout duration
	DialTimeout string `json:"dialTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// MaxRetries for operations
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSConfigSpec `json:"tls,omitempty"`
}

// MemberlistConfig defines memberlist configuration
type MemberlistConfig struct {
	// +kubebuilder:validation:Optional
	// JoinMembers list
	JoinMembers []string `json:"joinMembers,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// MinJoinBackoff duration
	MinJoinBackoff string `json:"minJoinBackoff,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// MaxJoinBackoff duration
	MaxJoinBackoff string `json:"maxJoinBackoff,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=3
	// MaxJoinRetries allowed
	MaxJoinRetries int32 `json:"maxJoinRetries,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// AbortIfClusterJoinFails flag
	AbortIfClusterJoinFails bool `json:"abortIfClusterJoinFails,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0
	// RejoinInterval duration (0 = disabled)
	RejoinInterval string `json:"rejoinInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=5s
	// LeftIngestersTimeout duration
	LeftIngestersTimeout string `json:"leftIngestersTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1m
	// LeaveTimeout duration
	LeaveTimeout string `json:"leaveTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// BindAddr for memberlist
	BindAddr string `json:"bindAddr,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=7946
	// BindPort for memberlist
	BindPort int32 `json:"bindPort,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=256
	// PacketDialTimeout in milliseconds
	PacketDialTimeout int32 `json:"packetDialTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1400
	// PacketWriteTimeout in milliseconds
	PacketWriteTimeout int32 `json:"packetWriteTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="lan"
	// TransportType for communication
	TransportType string `json:"transportType,omitempty"`
}

// ProcessorConfig defines metrics processor configuration
type ProcessorConfig struct {
	// +kubebuilder:validation:Optional
	// ServiceGraphs configuration
	ServiceGraphs *ServiceGraphsConfig `json:"serviceGraphs,omitempty"`

	// +kubebuilder:validation:Optional
	// SpanMetrics configuration
	SpanMetrics *SpanMetricsConfig `json:"spanMetrics,omitempty"`
}

// ServiceGraphsConfig defines service graphs configuration
type ServiceGraphsConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled flag
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// Dimensions to include
	Dimensions []string `json:"dimensions,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// Wait duration
	Wait string `json:"wait,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10000
	// MaxItems to track
	MaxItems int32 `json:"maxItems,omitempty"`

	// +kubebuilder:validation:Optional
	// Workers for processing
	Workers int32 `json:"workers,omitempty"`
}

// SpanMetricsConfig defines span metrics configuration
type SpanMetricsConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled flag
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// Dimensions to include
	Dimensions []string `json:"dimensions,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// EnableTargetInfo flag
	EnableTargetInfo bool `json:"enableTargetInfo,omitempty"`

	// +kubebuilder:validation:Optional
	// HistogramBuckets for latency
	HistogramBuckets []float64 `json:"histogramBuckets,omitempty"`

	// +kubebuilder:validation:Optional
	// IntrinsicDimensions configuration
	IntrinsicDimensions map[string]bool `json:"intrinsicDimensions,omitempty"`

	// +kubebuilder:validation:Optional
	// FilterPolicies for spans
	FilterPolicies []FilterPolicy `json:"filterPolicies,omitempty"`
}

// FilterPolicy defines span filter policy
type FilterPolicy struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=include;exclude
	// Type of filter
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// Match configuration
	Match *MatchConfig `json:"match,omitempty"`
}

// MatchConfig defines match configuration for filters
type MatchConfig struct {
	// +kubebuilder:validation:Optional
	// MatchType for the filter
	MatchType string `json:"matchType,omitempty"`

	// +kubebuilder:validation:Optional
	// Attributes to match
	Attributes []AttributeMatch `json:"attributes,omitempty"`
}

// AttributeMatch defines attribute matching
type AttributeMatch struct {
	// +kubebuilder:validation:Required
	// Key to match
	Key string `json:"key"`

	// +kubebuilder:validation:Optional
	// Value to match
	Value interface{} `json:"value,omitempty"`
}

// MetricsStorageConfig defines metrics storage configuration
type MetricsStorageConfig struct {
	// +kubebuilder:validation:Optional
	// Path for storage
	Path string `json:"path,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=prometheus
	// RemoteWriteBackend type
	RemoteWriteBackend string `json:"remoteWriteBackend,omitempty"`

	// +kubebuilder:validation:Optional
	// RemoteWrite endpoints
	RemoteWrite []RemoteWriteConfig `json:"remoteWrite,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="15s"
	// RemoteWriteFlushDeadline duration
	RemoteWriteFlushDeadline string `json:"remoteWriteFlushDeadline,omitempty"`

	// +kubebuilder:validation:Optional
	// WAL configuration
	WAL *MetricsWALConfig `json:"wal,omitempty"`
}

// RemoteWriteConfig defines remote write configuration for metrics
type RemoteWriteConfig struct {
	// +kubebuilder:validation:Required
	// URL endpoint
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	// Headers to send
	Headers map[string]string `json:"headers,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// SendExemplars flag
	SendExemplars bool `json:"sendExemplars,omitempty"`

	// +kubebuilder:validation:Optional
	// TLS configuration
	TLS *TLSConfigSpec `json:"tls,omitempty"`
}

// MetricsWALConfig defines metrics WAL configuration
type MetricsWALConfig struct {
	// +kubebuilder:validation:Optional
	// Path for WAL
	Path string `json:"path,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// WALSegmentDuration duration
	WALSegmentDuration string `json:"walSegmentDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="2m"
	// WALTruncateFrequency duration
	WALTruncateFrequency string `json:"walTruncateFrequency,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10m"
	// MaxWALTime duration
	MaxWALTime string `json:"maxWalTime,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// MinWALTime duration
	MinWALTime string `json:"minWalTime,omitempty"`
}

// RegistryConfig defines metrics registry configuration
type RegistryConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=20000
	// MaxLabelNameLength allowed
	MaxLabelNameLength int32 `json:"maxLabelNameLength,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=20000
	// MaxLabelValueLength allowed
	MaxLabelValueLength int32 `json:"maxLabelValueLength,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=30
	// MaxLabels per metric
	MaxLabels int32 `json:"maxLabels,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000000
	// MaxMetrics allowed
	MaxMetrics int32 `json:"maxMetrics,omitempty"`
}

// OverridesConfig defines per-tenant overrides
type OverridesConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1MB"
	// MaxBytesPerTrace limit
	MaxBytesPerTrace string `json:"maxBytesPerTrace,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10000
	// MaxTracesPerUser limit
	MaxTracesPerUser int32 `json:"maxTracesPerUser,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=20000
	// MaxGlobalTracesPerUser limit
	MaxGlobalTracesPerUser int32 `json:"maxGlobalTracesPerUser,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100000
	// MaxBytesPerTagValuesQuery limit
	MaxBytesPerTagValuesQuery int32 `json:"maxBytesPerTagValuesQuery,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=5000000
	// MaxBlocksPerTagValuesQuery limit
	MaxBlocksPerTagValuesQuery int32 `json:"maxBlocksPerTagValuesQuery,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=15000
	// IngestionRateLimitBytes per second
	IngestionRateLimitBytes int64 `json:"ingestionRateLimitBytes,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=20000
	// IngestionBurstSizeBytes allowed
	IngestionBurstSizeBytes int64 `json:"ingestionBurstSizeBytes,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0s"
	// BlockRetention duration (0 = global default)
	BlockRetention string `json:"blockRetention,omitempty"`

	// +kubebuilder:validation:Optional
	// PerTenantOverrideConfig path
	PerTenantOverrideConfig string `json:"perTenantOverrideConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	// PerTenantOverridePeriod reload interval
	PerTenantOverridePeriod string `json:"perTenantOverridePeriod,omitempty"`

	// +kubebuilder:validation:Optional
	// Tenants specific overrides
	Tenants map[string]TenantOverride `json:"tenants,omitempty"`
}

// TenantOverride defines per-tenant override values
type TenantOverride struct {
	// +kubebuilder:validation:Optional
	// MaxBytesPerTrace limit
	MaxBytesPerTrace string `json:"maxBytesPerTrace,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxTracesPerUser limit
	MaxTracesPerUser int32 `json:"maxTracesPerUser,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxGlobalTracesPerUser limit
	MaxGlobalTracesPerUser int32 `json:"maxGlobalTracesPerUser,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxBytesPerTagValuesQuery limit
	MaxBytesPerTagValuesQuery int32 `json:"maxBytesPerTagValuesQuery,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxBlocksPerTagValuesQuery limit
	MaxBlocksPerTagValuesQuery int32 `json:"maxBlocksPerTagValuesQuery,omitempty"`

	// +kubebuilder:validation:Optional
	// IngestionRateLimitBytes per second
	IngestionRateLimitBytes int64 `json:"ingestionRateLimitBytes,omitempty"`

	// +kubebuilder:validation:Optional
	// IngestionBurstSizeBytes allowed
	IngestionBurstSizeBytes int64 `json:"ingestionBurstSizeBytes,omitempty"`

	// +kubebuilder:validation:Optional
	// BlockRetention duration
	BlockRetention string `json:"blockRetention,omitempty"`
}

// TempoMultiTenancyConfig defines multi-tenancy configuration
type TempoMultiTenancyConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled flag for multi-tenancy
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="X-Scope-OrgID"
	// Header for tenant ID
	Header string `json:"header,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="anonymous"
	// DefaultTenant when header missing
	DefaultTenant string `json:"defaultTenant,omitempty"`
}

// TempoAuthConfig defines authentication configuration
type TempoAuthConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled flag for authentication
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=basic;oauth2;oidc
	// Type of authentication
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// BasicAuth configuration
	BasicAuth *BasicAuthConfig `json:"basicAuth,omitempty"`

	// +kubebuilder:validation:Optional
	// OAuth2 configuration
	OAuth2 *OAuth2Config `json:"oauth2,omitempty"`

	// +kubebuilder:validation:Optional
	// OIDC configuration
	OIDC *OIDCConfig `json:"oidc,omitempty"`
}

// OAuth2Config defines OAuth2 configuration
type OAuth2Config struct {
	// +kubebuilder:validation:Required
	// ClientID for OAuth2
	ClientID string `json:"clientId"`

	// +kubebuilder:validation:Required
	// ClientSecret for OAuth2
	ClientSecret corev1.SecretKeySelector `json:"clientSecret"`

	// +kubebuilder:validation:Required
	// TokenURL for OAuth2
	TokenURL string `json:"tokenUrl"`

	// +kubebuilder:validation:Optional
	// Scopes for OAuth2
	Scopes []string `json:"scopes,omitempty"`

	// +kubebuilder:validation:Optional
	// EndpointParams for OAuth2
	EndpointParams map[string]string `json:"endpointParams,omitempty"`
}

// OIDCConfig defines OIDC configuration
type OIDCConfig struct {
	// +kubebuilder:validation:Required
	// IssuerURL for OIDC
	IssuerURL string `json:"issuerUrl"`

	// +kubebuilder:validation:Required
	// ClientID for OIDC
	ClientID string `json:"clientId"`

	// +kubebuilder:validation:Optional
	// ClientSecret for OIDC
	ClientSecret corev1.SecretKeySelector `json:"clientSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="email"
	// UsernameClaim in JWT
	UsernameClaim string `json:"usernameClaim,omitempty"`

	// +kubebuilder:validation:Optional
	// GroupsClaim in JWT
	GroupsClaim string `json:"groupsClaim,omitempty"`

	// +kubebuilder:validation:Optional
	// Audience for validation
	Audience string `json:"audience,omitempty"`
}

// TempoRateLimitingConfig defines rate limiting configuration
type TempoRateLimitingConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled flag for rate limiting
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100
	// RPS (requests per second) limit
	RPS int32 `json:"rps,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1000
	// Burst size allowed
	Burst int32 `json:"burst,omitempty"`

	// +kubebuilder:validation:Optional
	// ByTenant rate limits
	ByTenant map[string]TenantRateLimit `json:"byTenant,omitempty"`
}

// TenantRateLimit defines per-tenant rate limits
type TenantRateLimit struct {
	// +kubebuilder:validation:Optional
	// RPS limit
	RPS int32 `json:"rps,omitempty"`

	// +kubebuilder:validation:Optional
	// Burst size
	Burst int32 `json:"burst,omitempty"`
}

// TempoConfigStatus defines the observed state of TempoConfig
type TempoConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Enum=Pending;Applied;Failed;Invalid
	// Phase represents the current phase of the configuration
	Phase string `json:"phase,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed spec
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastAppliedTime is when the configuration was last applied
	LastAppliedTime *metav1.Time `json:"lastAppliedTime,omitempty"`

	// LastAppliedHash is the hash of the last applied configuration
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`

	// Message provides additional information about the current state
	Message string `json:"message,omitempty"`

	// ValidationErrors contains any validation errors
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// SyncStatus shows the synchronization status with target platform
	SyncStatus *SyncStatus `json:"syncStatus,omitempty"`

	// ComponentStatus shows individual component status
	ComponentStatus map[string]ComponentHealthStatus `json:"componentStatus,omitempty"`
}

// ComponentHealthStatus defines component health status
type ComponentHealthStatus struct {
	// +kubebuilder:validation:Enum=Healthy;Unhealthy;Unknown
	// Health status
	Health string `json:"health"`

	// Message about the health
	Message string `json:"message,omitempty"`

	// LastChecked time
	LastChecked *metav1.Time `json:"lastChecked,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=tempoconfig;tempoconfigs,categories={observability,tempo}
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetPlatform.name`,description="Target ObservabilityPlatform"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="Current phase"
// +kubebuilder:printcolumn:name="Sync",type=string,JSONPath=`.status.syncStatus.state`,description="Sync status"
// +kubebuilder:printcolumn:name="Backend",type=string,JSONPath=`.spec.storage.trace.backend`,description="Storage backend"
// +kubebuilder:printcolumn:name="Last Applied",type=date,JSONPath=`.status.lastAppliedTime`,description="Last applied time"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Time since creation"

// TempoConfig is the Schema for the tempoconfigs API
type TempoConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TempoConfigSpec   `json:"spec,omitempty"`
	Status TempoConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TempoConfigList contains a list of TempoConfig
type TempoConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TempoConfig `json:"items"`
}

// Hub marks this type as a conversion hub.
func (*TempoConfig) Hub() {}

func init() {
	SchemeBuilder.Register(&TempoConfig{}, &TempoConfigList{})
}
