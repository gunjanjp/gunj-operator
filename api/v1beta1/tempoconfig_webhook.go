/*
Copyright 2025 Gunjan Jalori.

Licensed under the MIT License.
*/

package v1beta1

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var tempoconfiglog = logf.Log.WithName("tempoconfig-resource")

func (r *TempoConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-observability-io-v1beta1-tempoconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=tempoconfigs,verbs=create;update,versions=v1beta1,name=mtempoconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &TempoConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *TempoConfig) Default() {
	tempoconfiglog.Info("default", "name", r.Name)

	// Set default storage configuration
	if r.Spec.Storage == nil {
		r.Spec.Storage = &TempoStorageConfig{}
	}
	if r.Spec.Storage.Trace == nil {
		r.Spec.Storage.Trace = &TempoTraceStorageConfig{}
	}
	if r.Spec.Storage.Trace.Backend == "" {
		r.Spec.Storage.Trace.Backend = "s3" // Default to S3
	}
	if r.Spec.Storage.Trace.BackendBlocklistPoll == "" {
		r.Spec.Storage.Trace.BackendBlocklistPoll = "5m"
	}

	// Set default WAL configuration
	if r.Spec.Storage.Trace.WAL == nil {
		r.Spec.Storage.Trace.WAL = &TempoWALConfig{}
	}
	if r.Spec.Storage.Trace.WAL.Path == "" {
		r.Spec.Storage.Trace.WAL.Path = "/var/tempo/wal"
	}
	if r.Spec.Storage.Trace.WAL.Encoding == "" {
		r.Spec.Storage.Trace.WAL.Encoding = "zstd"
	}
	if r.Spec.Storage.Trace.WAL.SearchEncoding == "" {
		r.Spec.Storage.Trace.WAL.SearchEncoding = "10m"
	}
	if r.Spec.Storage.Trace.WAL.BlocksToKeep == 0 {
		r.Spec.Storage.Trace.WAL.BlocksToKeep = 10
	}

	// Set default block configuration
	if r.Spec.Storage.Trace.Block == nil {
		r.Spec.Storage.Trace.Block = &TempoBlockConfig{}
	}
	if r.Spec.Storage.Trace.Block.Version == "" {
		r.Spec.Storage.Trace.Block.Version = "v2"
	}
	if r.Spec.Storage.Trace.Block.Encoding == "" {
		r.Spec.Storage.Trace.Block.Encoding = "zstd"
	}
	if r.Spec.Storage.Trace.Block.RowGroupSizeBytes == 0 {
		r.Spec.Storage.Trace.Block.RowGroupSizeBytes = 100_000
	}

	// Set default pool configuration
	if r.Spec.Storage.Trace.Pool == nil {
		r.Spec.Storage.Trace.Pool = &TempoPoolConfig{}
	}
	if r.Spec.Storage.Trace.Pool.MaxWorkers == 0 {
		r.Spec.Storage.Trace.Pool.MaxWorkers = 400
	}
	if r.Spec.Storage.Trace.Pool.QueueDepth == 0 {
		r.Spec.Storage.Trace.Pool.QueueDepth = 20000
	}

	// Set default distributor configuration
	if r.Spec.Distributor == nil {
		r.Spec.Distributor = &TempoDistributorConfig{}
	}

	// Set default receivers
	if r.Spec.Distributor.Receivers == nil {
		r.Spec.Distributor.Receivers = &TempoReceiversConfig{}
	}

	// Set default OTLP receiver
	if r.Spec.Distributor.Receivers.OTLP == nil {
		r.Spec.Distributor.Receivers.OTLP = &OTLPReceiverConfig{}
	}
	if r.Spec.Distributor.Receivers.OTLP.GRPC == nil {
		r.Spec.Distributor.Receivers.OTLP.GRPC = &OTLPGRPCReceiverConfig{}
	}
	if r.Spec.Distributor.Receivers.OTLP.GRPC.Endpoint == "" {
		r.Spec.Distributor.Receivers.OTLP.GRPC.Endpoint = "0.0.0.0:4317"
	}
	if r.Spec.Distributor.Receivers.OTLP.GRPC.MaxRecvMsgSizeBytes == "" {
		r.Spec.Distributor.Receivers.OTLP.GRPC.MaxRecvMsgSizeBytes = "4194304" // 4MB
	}
	if r.Spec.Distributor.Receivers.OTLP.GRPC.MaxConcurrentStreams == 0 {
		r.Spec.Distributor.Receivers.OTLP.GRPC.MaxConcurrentStreams = 1000
	}

	// Set default HTTP receiver
	if r.Spec.Distributor.Receivers.OTLP.HTTP == nil {
		r.Spec.Distributor.Receivers.OTLP.HTTP = &OTLPHTTPReceiverConfig{}
	}
	if r.Spec.Distributor.Receivers.OTLP.HTTP.Endpoint == "" {
		r.Spec.Distributor.Receivers.OTLP.HTTP.Endpoint = "0.0.0.0:4318"
	}
	if r.Spec.Distributor.Receivers.OTLP.HTTP.MaxRequestBodySize == "" {
		r.Spec.Distributor.Receivers.OTLP.HTTP.MaxRequestBodySize = "20971520" // 20MB
	}

	// Set default Jaeger receiver if enabled
	if r.Spec.Distributor.Receivers.Jaeger != nil {
		if r.Spec.Distributor.Receivers.Jaeger.Protocols == nil {
			r.Spec.Distributor.Receivers.Jaeger.Protocols = &JaegerProtocolsConfig{}
		}
		if r.Spec.Distributor.Receivers.Jaeger.Protocols.GRPC == nil {
			r.Spec.Distributor.Receivers.Jaeger.Protocols.GRPC = &JaegerGRPCConfig{}
		}
		if r.Spec.Distributor.Receivers.Jaeger.Protocols.GRPC.Endpoint == "" {
			r.Spec.Distributor.Receivers.Jaeger.Protocols.GRPC.Endpoint = "0.0.0.0:14250"
		}
		if r.Spec.Distributor.Receivers.Jaeger.Protocols.ThriftHTTP == nil {
			r.Spec.Distributor.Receivers.Jaeger.Protocols.ThriftHTTP = &JaegerThriftHTTPConfig{}
		}
		if r.Spec.Distributor.Receivers.Jaeger.Protocols.ThriftHTTP.Endpoint == "" {
			r.Spec.Distributor.Receivers.Jaeger.Protocols.ThriftHTTP.Endpoint = "0.0.0.0:14268"
		}
	}

	// Set default Zipkin receiver if enabled
	if r.Spec.Distributor.Receivers.Zipkin != nil {
		if r.Spec.Distributor.Receivers.Zipkin.Endpoint == "" {
			r.Spec.Distributor.Receivers.Zipkin.Endpoint = "0.0.0.0:9411"
		}
	}

	// Set default rate limiting
	if r.Spec.Distributor.RateLimit == nil {
		r.Spec.Distributor.RateLimit = &TempoRateLimitConfig{}
	}
	if r.Spec.Distributor.RateLimit.RateLimiterBurstSize == 0 {
		r.Spec.Distributor.RateLimit.RateLimiterBurstSize = 100
	}
	if r.Spec.Distributor.RateLimit.Strategy == "" {
		r.Spec.Distributor.RateLimit.Strategy = "local"
	}

	// Set default ingester configuration
	if r.Spec.Ingester == nil {
		r.Spec.Ingester = &TempoIngesterConfig{}
	}
	if r.Spec.Ingester.MaxBlockDuration == "" {
		r.Spec.Ingester.MaxBlockDuration = "1h"
	}
	if r.Spec.Ingester.MaxBlockBytes == "" {
		r.Spec.Ingester.MaxBlockBytes = "1073741824" // 1GB
	}
	if r.Spec.Ingester.CompleteBlockTimeout == "" {
		r.Spec.Ingester.CompleteBlockTimeout = "15m"
	}
	if r.Spec.Ingester.MaxTracesPerBlock == 0 {
		r.Spec.Ingester.MaxTracesPerBlock = 1000000
	}
	if r.Spec.Ingester.ConcurrentFlushes == 0 {
		r.Spec.Ingester.ConcurrentFlushes = 4
	}
	if r.Spec.Ingester.FlushCheckPeriod == "" {
		r.Spec.Ingester.FlushCheckPeriod = "10s"
	}
	if r.Spec.Ingester.FlushOpTimeout == "" {
		r.Spec.Ingester.FlushOpTimeout = "10m"
	}

	// Set default lifecycler configuration
	if r.Spec.Ingester.LifecyclerConfig == nil {
		r.Spec.Ingester.LifecyclerConfig = &IngesterLifecyclerConfig{}
	}
	if r.Spec.Ingester.LifecyclerConfig.NumTokens == 0 {
		r.Spec.Ingester.LifecyclerConfig.NumTokens = 512
	}
	if r.Spec.Ingester.LifecyclerConfig.HeartbeatPeriod == "" {
		r.Spec.Ingester.LifecyclerConfig.HeartbeatPeriod = "5s"
	}
	if r.Spec.Ingester.LifecyclerConfig.JoinAfter == "" {
		r.Spec.Ingester.LifecyclerConfig.JoinAfter = "0s"
	}
	if r.Spec.Ingester.LifecyclerConfig.MinReadyDuration == "" {
		r.Spec.Ingester.LifecyclerConfig.MinReadyDuration = "15s"
	}

	// Set default compactor configuration
	if r.Spec.Compactor == nil {
		r.Spec.Compactor = &TempoCompactorConfig{}
	}
	if r.Spec.Compactor.CompactionWindow == "" {
		r.Spec.Compactor.CompactionWindow = "1h"
	}
	if r.Spec.Compactor.MaxCompactionObjects == 0 {
		r.Spec.Compactor.MaxCompactionObjects = 1000000
	}
	if r.Spec.Compactor.MaxBlockBytes == "" {
		r.Spec.Compactor.MaxBlockBytes = "107374182400" // 100GB
	}
	if r.Spec.Compactor.BlockRetention == "" {
		r.Spec.Compactor.BlockRetention = "336h" // 14 days
	}
	if r.Spec.Compactor.CompactedBlockRetention == "" {
		r.Spec.Compactor.CompactedBlockRetention = "1h"
	}
	if r.Spec.Compactor.MaxTracesPerBlock == 0 {
		r.Spec.Compactor.MaxTracesPerBlock = 5000000
	}
	if r.Spec.Compactor.RetentionIterations == 0 {
		r.Spec.Compactor.RetentionIterations = 10
	}
	if r.Spec.Compactor.TenantShardSize == 0 {
		r.Spec.Compactor.TenantShardSize = 1
	}
	if r.Spec.Compactor.CompactionCycle == "" {
		r.Spec.Compactor.CompactionCycle = "30s"
	}

	// Set default querier configuration
	if r.Spec.Querier == nil {
		r.Spec.Querier = &TempoQuerierConfig{}
	}
	if r.Spec.Querier.MaxOutstandingPerTenant == 0 {
		r.Spec.Querier.MaxOutstandingPerTenant = 2000
	}
	if r.Spec.Querier.ConcurrentRequests == 0 {
		r.Spec.Querier.ConcurrentRequests = 20
	}
	if r.Spec.Querier.WorkerParallelism == 0 {
		r.Spec.Querier.WorkerParallelism = 2
	}

	// Set default search configuration
	if r.Spec.Querier.Search == nil {
		r.Spec.Querier.Search = &TempoSearchConfig{}
	}
	if r.Spec.Querier.Search.MaxDuration == "" {
		r.Spec.Querier.Search.MaxDuration = "0s" // No limit
	}
	if r.Spec.Querier.Search.DefaultResultLimit == 0 {
		r.Spec.Querier.Search.DefaultResultLimit = 20
	}
	if r.Spec.Querier.Search.MaxResultLimit == 0 {
		r.Spec.Querier.Search.MaxResultLimit = 0 // No limit
	}
	if r.Spec.Querier.Search.ConcurrentJobs == 0 {
		r.Spec.Querier.Search.ConcurrentJobs = 1000
	}
	if r.Spec.Querier.Search.TargetBytesPerJob == "" {
		r.Spec.Querier.Search.TargetBytesPerJob = "104857600" // 100MB
	}
	if r.Spec.Querier.Search.ChunkSizeBytes == "" {
		r.Spec.Querier.Search.ChunkSizeBytes = "1048576" // 1MB
	}
	if r.Spec.Querier.Search.PrefetchJobs == 0 {
		r.Spec.Querier.Search.PrefetchJobs = 100
	}

	// Set default query frontend configuration
	if r.Spec.QueryFrontend == nil {
		r.Spec.QueryFrontend = &TempoQueryFrontendConfig{}
	}
	if r.Spec.QueryFrontend.MaxRetries == 0 {
		r.Spec.QueryFrontend.MaxRetries = 2
	}

	// Set default search configuration in query frontend
	if r.Spec.QueryFrontend.Search == nil {
		r.Spec.QueryFrontend.Search = &TempoQueryFrontendSearchConfig{}
	}
	if r.Spec.QueryFrontend.Search.ConcurrentJobs == 0 {
		r.Spec.QueryFrontend.Search.ConcurrentJobs = 1000
	}
	if r.Spec.QueryFrontend.Search.TargetBytesPerJob == "" {
		r.Spec.QueryFrontend.Search.TargetBytesPerJob = "104857600" // 100MB
	}
	if r.Spec.QueryFrontend.Search.MaxDuration == "" {
		r.Spec.QueryFrontend.Search.MaxDuration = "168h" // 7 days
	}
	if r.Spec.QueryFrontend.Search.QueryIngestersUntil == "" {
		r.Spec.QueryFrontend.Search.QueryIngestersUntil = "15m"
	}
	if r.Spec.QueryFrontend.Search.QueryTimeout == "" {
		r.Spec.QueryFrontend.Search.QueryTimeout = "5m"
	}

	// Set default server configuration
	if r.Spec.Server == nil {
		r.Spec.Server = &TempoServerConfig{}
	}
	if r.Spec.Server.HTTPListenPort == 0 {
		r.Spec.Server.HTTPListenPort = 3200
	}
	if r.Spec.Server.GRPCListenPort == 0 {
		r.Spec.Server.GRPCListenPort = 9095
	}
	if r.Spec.Server.LogLevel == "" {
		r.Spec.Server.LogLevel = "info"
	}
	if r.Spec.Server.LogFormat == "" {
		r.Spec.Server.LogFormat = "logfmt"
	}
	if r.Spec.Server.HTTPListenAddress == "" {
		r.Spec.Server.HTTPListenAddress = "0.0.0.0"
	}
	if r.Spec.Server.GRPCListenAddress == "" {
		r.Spec.Server.GRPCListenAddress = "0.0.0.0"
	}
	if r.Spec.Server.HTTPIdleTimeout == "" {
		r.Spec.Server.HTTPIdleTimeout = "120s"
	}
	if r.Spec.Server.HTTPWriteTimeout == "" {
		r.Spec.Server.HTTPWriteTimeout = "30s"
	}
	if r.Spec.Server.GracefulShutdownTimeout == "" {
		r.Spec.Server.GracefulShutdownTimeout = "30s"
	}
	if r.Spec.Server.GRPCMaxRecvMsgSize == "" {
		r.Spec.Server.GRPCMaxRecvMsgSize = "4194304" // 4MB
	}
	if r.Spec.Server.GRPCMaxSendMsgSize == "" {
		r.Spec.Server.GRPCMaxSendMsgSize = "4194304" // 4MB
	}
	if r.Spec.Server.GRPCMaxConcurrentStreams == 0 {
		r.Spec.Server.GRPCMaxConcurrentStreams = 100
	}

	// Set default overrides configuration
	if r.Spec.Overrides == nil {
		r.Spec.Overrides = &TempoOverridesConfig{}
	}
	if r.Spec.Overrides.OverridesReloadPeriod == "" {
		r.Spec.Overrides.OverridesReloadPeriod = "10s"
	}

	// Set default tenant overrides
	if r.Spec.Overrides.Defaults == nil {
		r.Spec.Overrides.Defaults = &TenantOverrides{}
	}
	if r.Spec.Overrides.Defaults.IngestionRateLimitBytes == 0 {
		r.Spec.Overrides.Defaults.IngestionRateLimitBytes = 15000000 // 15MB
	}
	if r.Spec.Overrides.Defaults.IngestionBurstSizeBytes == 0 {
		r.Spec.Overrides.Defaults.IngestionBurstSizeBytes = 20000000 // 20MB
	}
	if r.Spec.Overrides.Defaults.MaxTracesPerUser == 0 {
		r.Spec.Overrides.Defaults.MaxTracesPerUser = 10000
	}
	if r.Spec.Overrides.Defaults.MaxBytesPerTrace == "" {
		r.Spec.Overrides.Defaults.MaxBytesPerTrace = "5242880" // 5MB
	}
	if r.Spec.Overrides.Defaults.MaxSearchDuration == "" {
		r.Spec.Overrides.Defaults.MaxSearchDuration = "0s" // No limit
	}
	if r.Spec.Overrides.Defaults.MaxGlobalTracesPerUser == 0 {
		r.Spec.Overrides.Defaults.MaxGlobalTracesPerUser = 0 // No limit
	}
	if r.Spec.Overrides.Defaults.BlockRetention == "" {
		r.Spec.Overrides.Defaults.BlockRetention = "0s" // Use compactor retention
	}
	if r.Spec.Overrides.Defaults.MaxSearchBatchSize == 0 {
		r.Spec.Overrides.Defaults.MaxSearchBatchSize = 5000
	}
	if r.Spec.Overrides.Defaults.MaxSpansPerTrace == 0 {
		r.Spec.Overrides.Defaults.MaxSpansPerTrace = 50000
	}

	// Set default memberlist configuration
	if r.Spec.MemberlistKV == nil {
		r.Spec.MemberlistKV = &TempoMemberlistConfig{}
	}
	if r.Spec.MemberlistKV.BindAddr == "" {
		r.Spec.MemberlistKV.BindAddr = "0.0.0.0:7946"
	}
	if r.Spec.MemberlistKV.BindPort == 0 {
		r.Spec.MemberlistKV.BindPort = 7946
	}
	if r.Spec.MemberlistKV.JoinInterval == "" {
		r.Spec.MemberlistKV.JoinInterval = "1s"
	}
	if r.Spec.MemberlistKV.MaxJoinBackoff == "" {
		r.Spec.MemberlistKV.MaxJoinBackoff = "1m"
	}
	if r.Spec.MemberlistKV.MaxJoinRetries == 0 {
		r.Spec.MemberlistKV.MaxJoinRetries = 10
	}
	if r.Spec.MemberlistKV.MinJoinBackoff == "" {
		r.Spec.MemberlistKV.MinJoinBackoff = "1s"
	}
	if r.Spec.MemberlistKV.PushPullInterval == "" {
		r.Spec.MemberlistKV.PushPullInterval = "30s"
	}
	if r.Spec.MemberlistKV.RetransmitMult == 0 {
		r.Spec.MemberlistKV.RetransmitMult = 4
	}
	if r.Spec.MemberlistKV.GossipInterval == "" {
		r.Spec.MemberlistKV.GossipInterval = "200ms"
	}
	if r.Spec.MemberlistKV.GossipNodes == 0 {
		r.Spec.MemberlistKV.GossipNodes = 3
	}
	if r.Spec.MemberlistKV.GossipToDeadNodesTime == "" {
		r.Spec.MemberlistKV.GossipToDeadNodesTime = "30s"
	}
	if r.Spec.MemberlistKV.DeadNodeReclaimTime == "" {
		r.Spec.MemberlistKV.DeadNodeReclaimTime = "0s"
	}

	// Set defaults for storage backends
	r.defaultStorageBackend()

	// Set defaults for metrics generator if enabled
	if r.Spec.Metrics != nil && r.Spec.Metrics.Enabled {
		r.defaultMetricsGenerator()
	}
}

// defaultStorageBackend sets defaults for the configured storage backend
func (r *TempoConfig) defaultStorageBackend() {
	trace := r.Spec.Storage.Trace

	switch trace.Backend {
	case "s3":
		if trace.S3 == nil {
			trace.S3 = &TempoS3StorageConfig{}
		}
		if trace.S3.SignatureVersion == "" {
			trace.S3.SignatureVersion = "v4"
		}
		if trace.S3.HedgeRequestsUpTo == 0 {
			trace.S3.HedgeRequestsUpTo = 2
		}
		if trace.S3.PartSize == "" {
			trace.S3.PartSize = "5242880" // 5MB
		}

	case "gcs":
		if trace.GCS == nil {
			trace.GCS = &TempoGCSStorageConfig{}
		}
		if trace.GCS.ChunkBufferSize == 0 {
			trace.GCS.ChunkBufferSize = 10 * 1024 * 1024 // 10MB
		}
		if trace.GCS.RequestTimeout == "" {
			trace.GCS.RequestTimeout = "0s" // No timeout
		}
		if trace.GCS.HedgeRequestsUpTo == 0 {
			trace.GCS.HedgeRequestsUpTo = 2
		}

	case "azure":
		if trace.Azure == nil {
			trace.Azure = &TempoAzureStorageConfig{}
		}
		if trace.Azure.MaxBuffers == "" {
			trace.Azure.MaxBuffers = "4"
		}
		if trace.Azure.BufferSize == "" {
			trace.Azure.BufferSize = "3145728" // 3MB
		}
		if trace.Azure.HedgeRequestsUpTo == 0 {
			trace.Azure.HedgeRequestsUpTo = 2
		}

	case "filesystem":
		if trace.Filesystem == nil {
			trace.Filesystem = &TempoFilesystemStorageConfig{}
		}
		if trace.Filesystem.Path == "" {
			trace.Filesystem.Path = "/var/tempo/traces"
		}

	case "redis":
		if trace.Redis == nil {
			trace.Redis = &TempoRedisStorageConfig{}
		}
		if trace.Redis.DB == 0 {
			trace.Redis.DB = 0
		}
		if trace.Redis.Timeout == "" {
			trace.Redis.Timeout = "5s"
		}
		if trace.Redis.TTL == "" {
			trace.Redis.TTL = "336h" // 14 days
		}
	}
}

// defaultMetricsGenerator sets defaults for metrics generator
func (r *TempoConfig) defaultMetricsGenerator() {
	metrics := r.Spec.Metrics

	if metrics.MetricsFlushInterval == "" {
		metrics.MetricsFlushInterval = "30s"
	}
	if metrics.RemoteWriteFlushDeadline == "" {
		metrics.RemoteWriteFlushDeadline = "1m"
	}

	// Ring configuration
	if metrics.RingConfig == nil {
		metrics.RingConfig = &MetricsGeneratorRingConfig{}
	}
	if metrics.RingConfig.KVStore == "" {
		metrics.RingConfig.KVStore = "memberlist"
	}
	if metrics.RingConfig.HeartbeatPeriod == "" {
		metrics.RingConfig.HeartbeatPeriod = "5s"
	}
	if metrics.RingConfig.HeartbeatTimeout == "" {
		metrics.RingConfig.HeartbeatTimeout = "1m"
	}

	// Processor configuration
	if metrics.Processor == nil {
		metrics.Processor = &MetricsGeneratorProcessorConfig{}
	}

	// Service graphs
	if metrics.Processor.ServiceGraphs != nil && metrics.Processor.ServiceGraphs.Enabled {
		if metrics.Processor.ServiceGraphs.MaxItems == 0 {
			metrics.Processor.ServiceGraphs.MaxItems = 10000
		}
		if metrics.Processor.ServiceGraphs.WaitTime == "" {
			metrics.Processor.ServiceGraphs.WaitTime = "10s"
		}
		if metrics.Processor.ServiceGraphs.Workers == 0 {
			metrics.Processor.ServiceGraphs.Workers = 10
		}
		if len(metrics.Processor.ServiceGraphs.HistogramBuckets) == 0 {
			metrics.Processor.ServiceGraphs.HistogramBuckets = []float64{
				0.002, 0.004, 0.008, 0.016, 0.032, 0.064, 0.128, 0.256, 0.512, 1.02, 2.05, 4.10,
			}
		}
	}

	// Span metrics
	if metrics.Processor.SpanMetrics != nil && metrics.Processor.SpanMetrics.Enabled {
		if metrics.Processor.SpanMetrics.AggregationInterval == "" {
			metrics.Processor.SpanMetrics.AggregationInterval = "60s"
		}
		if len(metrics.Processor.SpanMetrics.HistogramBuckets) == 0 {
			metrics.Processor.SpanMetrics.HistogramBuckets = []float64{
				0.002, 0.004, 0.008, 0.016, 0.032, 0.064, 0.128, 0.256, 0.512, 1.02, 2.05, 4.10,
			}
		}
	}

	// Registry configuration
	if metrics.Registry == nil {
		metrics.Registry = &MetricsGeneratorRegistryConfig{}
	}
	if metrics.Registry.MaxSeries == 0 {
		metrics.Registry.MaxSeries = 0 // No limit
	}
	if metrics.Registry.MaxSeriesPerLabelSet == 0 {
		metrics.Registry.MaxSeriesPerLabelSet = 0 // No limit
	}
	if metrics.Registry.StaleDuration == "" {
		metrics.Registry.StaleDuration = "5m"
	}

	// Storage configuration
	if metrics.Storage == nil {
		metrics.Storage = &MetricsGeneratorStorageConfig{}
	}
	if metrics.Storage.Path == "" {
		metrics.Storage.Path = "/var/tempo/metrics-generator/wal"
	}

	// WAL configuration
	if metrics.Storage.WAL == nil {
		metrics.Storage.WAL = &MetricsGeneratorWALConfig{}
	}
	if metrics.Storage.WAL.Path == "" {
		metrics.Storage.WAL.Path = "/var/tempo/metrics-generator/wal"
	}
	if metrics.Storage.WAL.TruncateFrequency == "" {
		metrics.Storage.WAL.TruncateFrequency = "1h"
	}
	if metrics.Storage.WAL.MinWALTime == "" {
		metrics.Storage.WAL.MinWALTime = "0s"
	}
	if metrics.Storage.WAL.MaxWALTime == "" {
		metrics.Storage.WAL.MaxWALTime = "4h"
	}
}

// +kubebuilder:webhook:path=/validate-observability-io-v1beta1-tempoconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=tempoconfigs,verbs=create;update,versions=v1beta1,name=vtempoconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &TempoConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *TempoConfig) ValidateCreate() (admission.Warnings, error) {
	tempoconfiglog.Info("validate create", "name", r.Name)

	var allErrs field.ErrorList

	// Validate storage configuration
	if err := r.validateStorageConfig(field.NewPath("spec").Child("storage")); err != nil {
		allErrs = append(allErrs, err...)
	}

	// Validate distributor configuration
	if r.Spec.Distributor != nil {
		if err := r.validateDistributorConfig(field.NewPath("spec").Child("distributor")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate ingester configuration
	if r.Spec.Ingester != nil {
		if err := r.validateIngesterConfig(field.NewPath("spec").Child("ingester")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate compactor configuration
	if r.Spec.Compactor != nil {
		if err := r.validateCompactorConfig(field.NewPath("spec").Child("compactor")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate querier configuration
	if r.Spec.Querier != nil {
		if err := r.validateQuerierConfig(field.NewPath("spec").Child("querier")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate query frontend configuration
	if r.Spec.QueryFrontend != nil {
		if err := r.validateQueryFrontendConfig(field.NewPath("spec").Child("queryFrontend")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate server configuration
	if r.Spec.Server != nil {
		if err := r.validateServerConfig(field.NewPath("spec").Child("server")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate overrides configuration
	if r.Spec.Overrides != nil {
		if err := r.validateOverridesConfig(field.NewPath("spec").Child("overrides")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate metrics generator configuration
	if r.Spec.Metrics != nil && r.Spec.Metrics.Enabled {
		if err := r.validateMetricsGeneratorConfig(field.NewPath("spec").Child("metrics")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, allErrs.ToAggregate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *TempoConfig) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	tempoconfiglog.Info("validate update", "name", r.Name)

	oldConfig, ok := old.(*TempoConfig)
	if !ok {
		return nil, fmt.Errorf("expected TempoConfig but got %T", old)
	}

	var allErrs field.ErrorList

	// Validate immutable fields
	if oldConfig.Spec.Storage.Trace.Backend != r.Spec.Storage.Trace.Backend {
		allErrs = append(allErrs, field.Forbidden(
			field.NewPath("spec").Child("storage").Child("trace").Child("backend"),
			"storage backend type cannot be changed after creation"))
	}

	// Validate block version changes
	if oldConfig.Spec.Storage.Trace.Block != nil && r.Spec.Storage.Trace.Block != nil {
		if oldConfig.Spec.Storage.Trace.Block.Version != r.Spec.Storage.Trace.Block.Version {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("storage").Child("trace").Child("block").Child("version"),
				r.Spec.Storage.Trace.Block.Version,
				"block version cannot be changed without data migration"))
		}
	}

	// Run standard validation
	warnings, err := r.ValidateCreate()
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), r.Spec, err.Error()))
	}

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, allErrs.ToAggregate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *TempoConfig) ValidateDelete() (admission.Warnings, error) {
	tempoconfiglog.Info("validate delete", "name", r.Name)
	// No special validation for delete
	return nil, nil
}

// validateStorageConfig validates the storage configuration
func (r *TempoConfig) validateStorageConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if r.Spec.Storage == nil || r.Spec.Storage.Trace == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("trace"), "trace storage configuration is required"))
		return allErrs
	}

	trace := r.Spec.Storage.Trace
	tracePath := fldPath.Child("trace")

	// Validate backend configuration
	switch trace.Backend {
	case "s3":
		if trace.S3 == nil {
			allErrs = append(allErrs, field.Required(tracePath.Child("s3"), "S3 configuration is required when backend is s3"))
		} else {
			allErrs = append(allErrs, r.validateS3Config(tracePath.Child("s3"), trace.S3)...)
		}
	case "gcs":
		if trace.GCS == nil {
			allErrs = append(allErrs, field.Required(tracePath.Child("gcs"), "GCS configuration is required when backend is gcs"))
		} else {
			allErrs = append(allErrs, r.validateGCSConfig(tracePath.Child("gcs"), trace.GCS)...)
		}
	case "azure":
		if trace.Azure == nil {
			allErrs = append(allErrs, field.Required(tracePath.Child("azure"), "Azure configuration is required when backend is azure"))
		} else {
			allErrs = append(allErrs, r.validateAzureConfig(tracePath.Child("azure"), trace.Azure)...)
		}
	case "filesystem":
		if trace.Filesystem == nil {
			allErrs = append(allErrs, field.Required(tracePath.Child("filesystem"), "Filesystem configuration is required when backend is filesystem"))
		} else {
			allErrs = append(allErrs, r.validateFilesystemConfig(tracePath.Child("filesystem"), trace.Filesystem)...)
		}
	case "redis":
		if trace.Redis == nil {
			allErrs = append(allErrs, field.Required(tracePath.Child("redis"), "Redis configuration is required when backend is redis"))
		} else {
			allErrs = append(allErrs, r.validateRedisConfig(tracePath.Child("redis"), trace.Redis)...)
		}
	default:
		allErrs = append(allErrs, field.Invalid(tracePath.Child("backend"), trace.Backend, "unsupported storage backend"))
	}

	// Validate WAL configuration
	if trace.WAL != nil {
		allErrs = append(allErrs, r.validateWALConfig(tracePath.Child("wal"), trace.WAL)...)
	}

	// Validate block configuration
	if trace.Block != nil {
		allErrs = append(allErrs, r.validateBlockConfig(tracePath.Child("block"), trace.Block)...)
	}

	// Validate cache configuration
	if trace.Cache != nil {
		allErrs = append(allErrs, r.validateCacheConfig(tracePath.Child("cache"), trace.Cache)...)
	}

	return allErrs
}

// validateS3Config validates S3 storage configuration
func (r *TempoConfig) validateS3Config(fldPath *field.Path, s3 *TempoS3StorageConfig) field.ErrorList {
	var allErrs field.ErrorList

	if s3.Bucket == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("bucket"), "bucket name is required"))
	}

	// Validate endpoint URL if provided
	if s3.Endpoint != "" {
		if _, err := url.Parse(s3.Endpoint); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("endpoint"), s3.Endpoint, "invalid endpoint URL"))
		}
	}

	// Validate signature version
	if s3.SignatureVersion != "" && s3.SignatureVersion != "s3v4" && s3.SignatureVersion != "s3v2" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("signatureVersion"), s3.SignatureVersion, "signature version must be s3v4 or s3v2"))
	}

	// Validate storage class
	validStorageClasses := map[string]bool{
		"STANDARD":            true,
		"REDUCED_REDUNDANCY":  true,
		"STANDARD_IA":         true,
		"ONEZONE_IA":          true,
		"INTELLIGENT_TIERING": true,
		"GLACIER":             true,
		"DEEP_ARCHIVE":        true,
		"GLACIER_IR":          true,
	}
	if s3.StorageClass != "" && !validStorageClasses[s3.StorageClass] {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("storageClass"), s3.StorageClass, "invalid storage class"))
	}

	// Validate SSE configuration
	if s3.SSE != nil {
		if s3.SSE.Type != "SSE-S3" && s3.SSE.Type != "SSE-KMS" {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("sse").Child("type"), s3.SSE.Type, "SSE type must be SSE-S3 or SSE-KMS"))
		}
		if s3.SSE.Type == "SSE-KMS" && s3.SSE.KMSKeyID == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("sse").Child("kmsKeyId"), "KMS key ID is required for SSE-KMS"))
		}
	}

	// Validate part size
	if s3.PartSize != "" {
		if _, err := parseByteSize(s3.PartSize); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("partSize"), s3.PartSize, "invalid byte size format"))
		}
	}

	return allErrs
}

// validateGCSConfig validates GCS storage configuration
func (r *TempoConfig) validateGCSConfig(fldPath *field.Path, gcs *TempoGCSStorageConfig) field.ErrorList {
	var allErrs field.ErrorList

	if gcs.BucketName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("bucketName"), "bucket name is required"))
	}

	// Validate request timeout
	if gcs.RequestTimeout != "" && !isValidDuration(gcs.RequestTimeout) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("requestTimeout"), gcs.RequestTimeout, "invalid duration format"))
	}

	// Validate service account configuration
	if gcs.ServiceAccount != nil {
		if gcs.ServiceAccount.KeyFile == "" && gcs.ServiceAccount.KeySecret.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("serviceAccount"), "either keyFile or keySecret must be specified"))
		}
	}

	return allErrs
}

// validateAzureConfig validates Azure storage configuration
func (r *TempoConfig) validateAzureConfig(fldPath *field.Path, azure *TempoAzureStorageConfig) field.ErrorList {
	var allErrs field.ErrorList

	if azure.ContainerName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("containerName"), "container name is required"))
	}

	if azure.AccountName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("accountName"), "account name is required"))
	}

	// Validate authentication method
	hasAuth := false
	if azure.AccountKey.Name != "" {
		hasAuth = true
	}
	if azure.ConnectionString.Name != "" {
		hasAuth = true
	}
	if azure.UseManagedIdentity {
		hasAuth = true
	}

	if !hasAuth {
		allErrs = append(allErrs, field.Required(fldPath, "one of accountKey, connectionString, or useManagedIdentity must be specified"))
	}

	// Validate endpoint URL if provided
	if azure.Endpoint != "" {
		if _, err := url.Parse(azure.Endpoint); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("endpoint"), azure.Endpoint, "invalid endpoint URL"))
		}
	}

	// Validate buffer sizes
	if azure.MaxBuffers != "" {
		if _, err := parseByteSize(azure.MaxBuffers); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("maxBuffers"), azure.MaxBuffers, "invalid byte size format"))
		}
	}
	if azure.BufferSize != "" {
		if _, err := parseByteSize(azure.BufferSize); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("bufferSize"), azure.BufferSize, "invalid byte size format"))
		}
	}

	return allErrs
}

// validateFilesystemConfig validates filesystem storage configuration
func (r *TempoConfig) validateFilesystemConfig(fldPath *field.Path, fs *TempoFilesystemStorageConfig) field.ErrorList {
	var allErrs field.ErrorList

	if fs.Path == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("path"), "storage path is required"))
	}

	return allErrs
}

// validateRedisConfig validates Redis storage configuration
func (r *TempoConfig) validateRedisConfig(fldPath *field.Path, redis *TempoRedisStorageConfig) field.ErrorList {
	var allErrs field.ErrorList

	if redis.Endpoint == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("endpoint"), "endpoint is required"))
	}

	// Validate timeout
	if redis.Timeout != "" && !isValidDuration(redis.Timeout) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("timeout"), redis.Timeout, "invalid duration format"))
	}

	// Validate TTL
	if redis.TTL != "" && !isValidDuration(redis.TTL) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ttl"), redis.TTL, "invalid duration format"))
	}

	// Validate TLS configuration
	if redis.TLS != nil && redis.TLS.Enabled {
		if redis.TLS.CertFile != "" && redis.TLS.KeyFile == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("tls").Child("keyFile"), "key file is required when cert file is specified"))
		}
		if redis.TLS.KeyFile != "" && redis.TLS.CertFile == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("tls").Child("certFile"), "cert file is required when key file is specified"))
		}
	}

	return allErrs
}

// validateWALConfig validates WAL configuration
func (r *TempoConfig) validateWALConfig(fldPath *field.Path, wal *TempoWALConfig) field.ErrorList {
	var allErrs field.ErrorList

	if wal.Path == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("path"), "WAL path is required"))
	}

	// Validate encoding
	validEncodings := map[string]bool{
		"none":   true,
		"snappy": true,
		"lz4":    true,
		"gzip":   true,
		"zstd":   true,
	}
	if wal.Encoding != "" && !validEncodings[wal.Encoding] {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("encoding"), wal.Encoding, "invalid encoding type"))
	}

	// Validate search encoding duration
	if wal.SearchEncoding != "" && !isValidDuration(wal.SearchEncoding) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("searchEncoding"), wal.SearchEncoding, "invalid duration format"))
	}

	return allErrs
}

// validateBlockConfig validates block configuration
func (r *TempoConfig) validateBlockConfig(fldPath *field.Path, block *TempoBlockConfig) field.ErrorList {
	var allErrs field.ErrorList

	// Validate block version
	if block.Version != "v2" && block.Version != "vParquet" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), block.Version, "block version must be v2 or vParquet"))
	}

	// Validate encoding
	validEncodings := map[string]bool{
		"none":     true,
		"gzip":     true,
		"lz4-64k":  true,
		"lz4-256k": true,
		"lz4-1M":   true,
		"lz4":      true,
		"snappy":   true,
		"zstd":     true,
		"s2":       true,
	}
	if block.Encoding != "" && !validEncodings[block.Encoding] {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("encoding"), block.Encoding, "invalid encoding type"))
	}

	// Validate bloom filter configuration
	if block.BloomFilter != nil {
		if block.BloomFilter.FalsePositive < 0 || block.BloomFilter.FalsePositive > 1 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("bloomFilter").Child("falsePositive"), block.BloomFilter.FalsePositive, "false positive rate must be between 0 and 1"))
		}
	}

	return allErrs
}

// validateCacheConfig validates cache configuration
func (r *TempoConfig) validateCacheConfig(fldPath *field.Path, cache *TempoCacheConfig) field.ErrorList {
	var allErrs field.ErrorList

	switch cache.Backend {
	case "redis":
		if cache.Redis == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("redis"), "Redis configuration is required when backend is redis"))
		} else {
			if cache.Redis.Endpoint == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("redis").Child("endpoint"), "endpoint is required"))
			}
			if cache.Redis.Timeout != "" && !isValidDuration(cache.Redis.Timeout) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("redis").Child("timeout"), cache.Redis.Timeout, "invalid duration format"))
			}
			if cache.Redis.Expiration != "" && !isValidDuration(cache.Redis.Expiration) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("redis").Child("expiration"), cache.Redis.Expiration, "invalid duration format"))
			}
		}
	case "memcached":
		if cache.Memcached == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("memcached"), "Memcached configuration is required when backend is memcached"))
		} else {
			if cache.Memcached.Host == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("memcached").Child("host"), "host is required"))
			}
			if cache.Memcached.Timeout != "" && !isValidDuration(cache.Memcached.Timeout) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("memcached").Child("timeout"), cache.Memcached.Timeout, "invalid duration format"))
			}
			if cache.Memcached.UpdateInterval != "" && !isValidDuration(cache.Memcached.UpdateInterval) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("memcached").Child("updateInterval"), cache.Memcached.UpdateInterval, "invalid duration format"))
			}
		}
	default:
		allErrs = append(allErrs, field.Invalid(fldPath.Child("backend"), cache.Backend, "cache backend must be redis or memcached"))
	}

	return allErrs
}

// validateDistributorConfig validates distributor configuration
func (r *TempoConfig) validateDistributorConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	distributor := r.Spec.Distributor

	// Validate receivers
	if distributor.Receivers != nil {
		allErrs = append(allErrs, r.validateReceiversConfig(fldPath.Child("receivers"), distributor.Receivers)...)
	}

	// Validate log level
	if distributor.LogReceivedTraces != "" {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[distributor.LogReceivedTraces] {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("logReceivedTraces"), distributor.LogReceivedTraces, "invalid log level"))
		}
	}

	// Validate rate limit configuration
	if distributor.RateLimit != nil {
		if distributor.RateLimit.Strategy != "local" && distributor.RateLimit.Strategy != "global" {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("rateLimit").Child("strategy"), distributor.RateLimit.Strategy, "strategy must be local or global"))
		}
	}

	return allErrs
}

// validateReceiversConfig validates receivers configuration
func (r *TempoConfig) validateReceiversConfig(fldPath *field.Path, receivers *TempoReceiversConfig) field.ErrorList {
	var allErrs field.ErrorList

	// Validate OTLP receiver
	if receivers.OTLP != nil {
		if receivers.OTLP.GRPC != nil {
			allErrs = append(allErrs, r.validateEndpoint(fldPath.Child("otlp").Child("grpc").Child("endpoint"), receivers.OTLP.GRPC.Endpoint)...)
			if receivers.OTLP.GRPC.MaxRecvMsgSizeBytes != "" {
				if _, err := parseByteSize(receivers.OTLP.GRPC.MaxRecvMsgSizeBytes); err != nil {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("otlp").Child("grpc").Child("maxRecvMsgSizeBytes"), receivers.OTLP.GRPC.MaxRecvMsgSizeBytes, "invalid byte size format"))
				}
			}
		}
		if receivers.OTLP.HTTP != nil {
			allErrs = append(allErrs, r.validateEndpoint(fldPath.Child("otlp").Child("http").Child("endpoint"), receivers.OTLP.HTTP.Endpoint)...)
			if receivers.OTLP.HTTP.MaxRequestBodySize != "" {
				if _, err := parseByteSize(receivers.OTLP.HTTP.MaxRequestBodySize); err != nil {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("otlp").Child("http").Child("maxRequestBodySize"), receivers.OTLP.HTTP.MaxRequestBodySize, "invalid byte size format"))
				}
			}
		}
	}

	// Validate Jaeger receiver
	if receivers.Jaeger != nil && receivers.Jaeger.Protocols != nil {
		if receivers.Jaeger.Protocols.GRPC != nil {
			allErrs = append(allErrs, r.validateEndpoint(fldPath.Child("jaeger").Child("protocols").Child("grpc").Child("endpoint"), receivers.Jaeger.Protocols.GRPC.Endpoint)...)
		}
		if receivers.Jaeger.Protocols.ThriftHTTP != nil {
			allErrs = append(allErrs, r.validateEndpoint(fldPath.Child("jaeger").Child("protocols").Child("thriftHttp").Child("endpoint"), receivers.Jaeger.Protocols.ThriftHTTP.Endpoint)...)
		}
		if receivers.Jaeger.Protocols.ThriftBinary != nil {
			allErrs = append(allErrs, r.validateEndpoint(fldPath.Child("jaeger").Child("protocols").Child("thriftBinary").Child("endpoint"), receivers.Jaeger.Protocols.ThriftBinary.Endpoint)...)
		}
		if receivers.Jaeger.Protocols.ThriftCompact != nil {
			allErrs = append(allErrs, r.validateEndpoint(fldPath.Child("jaeger").Child("protocols").Child("thriftCompact").Child("endpoint"), receivers.Jaeger.Protocols.ThriftCompact.Endpoint)...)
		}
	}

	// Validate Zipkin receiver
	if receivers.Zipkin != nil {
		allErrs = append(allErrs, r.validateEndpoint(fldPath.Child("zipkin").Child("endpoint"), receivers.Zipkin.Endpoint)...)
	}

	// Validate Kafka receiver
	if receivers.Kafka != nil {
		if len(receivers.Kafka.Brokers) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("kafka").Child("brokers"), "at least one broker is required"))
		}
		if receivers.Kafka.Topic == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("kafka").Child("topic"), "topic is required"))
		}
		
		// Validate protocol version
		validProtocols := map[string]bool{"plaintext": true, "ssl": true, "sasl_plaintext": true, "sasl_ssl": true}
		if receivers.Kafka.ProtocolVersion != "" && !validProtocols[receivers.Kafka.ProtocolVersion] {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("kafka").Child("protocolVersion"), receivers.Kafka.ProtocolVersion, "invalid protocol version"))
		}
		
		// Validate encoding
		validEncodings := map[string]bool{
			"otlp_proto": true, "jaeger_proto": true, "jaeger_json": true,
			"zipkin_proto": true, "zipkin_json": true, "zipkin_thrift": true,
		}
		if receivers.Kafka.Encoding != "" && !validEncodings[receivers.Kafka.Encoding] {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("kafka").Child("encoding"), receivers.Kafka.Encoding, "invalid encoding"))
		}
	}

	return allErrs
}

// validateIngesterConfig validates ingester configuration
func (r *TempoConfig) validateIngesterConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	ingester := r.Spec.Ingester

	// Validate durations
	durations := map[string]string{
		"maxBlockDuration":     ingester.MaxBlockDuration,
		"completeBlockTimeout": ingester.CompleteBlockTimeout,
		"flushCheckPeriod":     ingester.FlushCheckPeriod,
		"flushOpTimeout":       ingester.FlushOpTimeout,
	}

	for field, duration := range durations {
		if duration != "" && !isValidDuration(duration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child(field), duration, "invalid duration format"))
		}
	}

	// Validate max block bytes
	if ingester.MaxBlockBytes != "" {
		if _, err := parseByteSize(ingester.MaxBlockBytes); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("maxBlockBytes"), ingester.MaxBlockBytes, "invalid byte size format"))
		}
	}

	// Validate lifecycler configuration
	if ingester.LifecyclerConfig != nil {
		lifecyclerPath := fldPath.Child("lifecyclerConfig")
		
		// Validate durations
		lifecyclerDurations := map[string]string{
			"ringCheckPeriod": ingester.LifecyclerConfig.RingCheckPeriod,
			"joinAfter":       ingester.LifecyclerConfig.JoinAfter,
			"minReadyDuration": ingester.LifecyclerConfig.MinReadyDuration,
			"heartbeatPeriod": ingester.LifecyclerConfig.HeartbeatPeriod,
		}

		for field, duration := range lifecyclerDurations {
			if duration != "" && !isValidDuration(duration) {
				allErrs = append(allErrs, field.Invalid(lifecyclerPath.Child(field), duration, "invalid duration format"))
			}
		}

		// Validate port range
		if ingester.LifecyclerConfig.Port < 0 || ingester.LifecyclerConfig.Port > 65535 {
			allErrs = append(allErrs, field.Invalid(lifecyclerPath.Child("port"), ingester.LifecyclerConfig.Port, "port must be between 0 and 65535"))
		}
	}

	return allErrs
}

// validateCompactorConfig validates compactor configuration
func (r *TempoConfig) validateCompactorConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	compactor := r.Spec.Compactor

	// Validate durations
	durations := map[string]string{
		"blockRetention":          compactor.BlockRetention,
		"compactedBlockRetention": compactor.CompactedBlockRetention,
		"compactionWindow":        compactor.CompactionWindow,
		"compactionCycle":         compactor.CompactionCycle,
	}

	for field, duration := range durations {
		if duration != "" && !isValidDuration(duration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child(field), duration, "invalid duration format"))
		}
	}

	// Validate byte sizes
	byteSizes := map[string]string{
		"maxBlockBytes":      compactor.MaxBlockBytes,
		"flushSizeBytes":     compactor.FlushSizeBytes,
		"iteratorBufferSize": compactor.IteratorBufferSize,
	}

	for field, size := range byteSizes {
		if size != "" {
			if _, err := parseByteSize(size); err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child(field), size, "invalid byte size format"))
			}
		}
	}

	// Validate retention is not less than compacted block retention
	if compactor.BlockRetention != "" && compactor.CompactedBlockRetention != "" {
		blockRetention, _ := time.ParseDuration(compactor.BlockRetention)
		compactedRetention, _ := time.ParseDuration(compactor.CompactedBlockRetention)
		if blockRetention < compactedRetention {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("blockRetention"), compactor.BlockRetention, "block retention must be greater than or equal to compacted block retention"))
		}
	}

	return allErrs
}

// validateQuerierConfig validates querier configuration
func (r *TempoConfig) validateQuerierConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	querier := r.Spec.Querier

	// Validate search configuration
	if querier.Search != nil {
		searchPath := fldPath.Child("search")
		if querier.Search.MaxDuration != "" && !isValidDuration(querier.Search.MaxDuration) {
			allErrs = append(allErrs, field.Invalid(searchPath.Child("maxDuration"), querier.Search.MaxDuration, "invalid duration format"))
		}
		if querier.Search.TargetBytesPerJob != "" {
			if _, err := parseByteSize(querier.Search.TargetBytesPerJob); err != nil {
				allErrs = append(allErrs, field.Invalid(searchPath.Child("targetBytesPerJob"), querier.Search.TargetBytesPerJob, "invalid byte size format"))
			}
		}
		if querier.Search.ChunkSizeBytes != "" {
			if _, err := parseByteSize(querier.Search.ChunkSizeBytes); err != nil {
				allErrs = append(allErrs, field.Invalid(searchPath.Child("chunkSizeBytes"), querier.Search.ChunkSizeBytes, "invalid byte size format"))
			}
		}
		if querier.Search.CacheConnectionTimeout != "" && !isValidDuration(querier.Search.CacheConnectionTimeout) {
			allErrs = append(allErrs, field.Invalid(searchPath.Child("cacheConnectionTimeout"), querier.Search.CacheConnectionTimeout, "invalid duration format"))
		}
	}

	// Validate max bytes per tag values
	if querier.MaxBytesPerTagValues != "" {
		if _, err := parseByteSize(querier.MaxBytesPerTagValues); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("maxBytesPerTagValues"), querier.MaxBytesPerTagValues, "invalid byte size format"))
		}
	}

	// Validate frontend configuration
	if querier.Frontend != nil && querier.Frontend.WorkerGRPCClientConfig != nil {
		clientPath := fldPath.Child("frontend").Child("workerGrpcClientConfig")
		if querier.Frontend.WorkerGRPCClientConfig.MaxRecvMsgSize != "" {
			if _, err := parseByteSize(querier.Frontend.WorkerGRPCClientConfig.MaxRecvMsgSize); err != nil {
				allErrs = append(allErrs, field.Invalid(clientPath.Child("maxRecvMsgSize"), querier.Frontend.WorkerGRPCClientConfig.MaxRecvMsgSize, "invalid byte size format"))
			}
		}
		if querier.Frontend.WorkerGRPCClientConfig.MaxSendMsgSize != "" {
			if _, err := parseByteSize(querier.Frontend.WorkerGRPCClientConfig.MaxSendMsgSize); err != nil {
				allErrs = append(allErrs, field.Invalid(clientPath.Child("maxSendMsgSize"), querier.Frontend.WorkerGRPCClientConfig.MaxSendMsgSize, "invalid byte size format"))
			}
		}
	}

	return allErrs
}

// validateQueryFrontendConfig validates query frontend configuration
func (r *TempoConfig) validateQueryFrontendConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	frontend := r.Spec.QueryFrontend

	// Validate search configuration
	if frontend.Search != nil {
		searchPath := fldPath.Child("search")
		if frontend.Search.TargetBytesPerJob != "" {
			if _, err := parseByteSize(frontend.Search.TargetBytesPerJob); err != nil {
				allErrs = append(allErrs, field.Invalid(searchPath.Child("targetBytesPerJob"), frontend.Search.TargetBytesPerJob, "invalid byte size format"))
			}
		}
		durations := map[string]string{
			"maxDuration":         frontend.Search.MaxDuration,
			"queryIngestersUntil": frontend.Search.QueryIngestersUntil,
			"queryTimeout":        frontend.Search.QueryTimeout,
		}
		for field, duration := range durations {
			if duration != "" && !isValidDuration(duration) {
				allErrs = append(allErrs, field.Invalid(searchPath.Child(field), duration, "invalid duration format"))
			}
		}
	}

	// Validate trace by ID configuration
	if frontend.TraceByID != nil {
		traceByIDPath := fldPath.Child("traceById")
		if frontend.TraceByID.QueryTimeout != "" && !isValidDuration(frontend.TraceByID.QueryTimeout) {
			allErrs = append(allErrs, field.Invalid(traceByIDPath.Child("queryTimeout"), frontend.TraceByID.QueryTimeout, "invalid duration format"))
		}
		if frontend.TraceByID.HedgeRequestsAt != "" && !isValidDuration(frontend.TraceByID.HedgeRequestsAt) {
			allErrs = append(allErrs, field.Invalid(traceByIDPath.Child("hedgeRequestsAt"), frontend.TraceByID.HedgeRequestsAt, "invalid duration format"))
		}
	}

	// Validate metrics configuration
	if frontend.Metrics != nil {
		metricsPath := fldPath.Child("metrics")
		durations := map[string]string{
			"queryTimeout":      frontend.Metrics.QueryTimeout,
			"maxDuration":       frontend.Metrics.MaxDuration,
			"queryBackendAfter": frontend.Metrics.QueryBackendAfter,
		}
		for field, duration := range durations {
			if duration != "" && !isValidDuration(duration) {
				allErrs = append(allErrs, field.Invalid(metricsPath.Child(field), duration, "invalid duration format"))
			}
		}
	}

	return allErrs
}

// validateServerConfig validates server configuration
func (r *TempoConfig) validateServerConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	server := r.Spec.Server

	// Validate log level
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if server.LogLevel != "" && !validLogLevels[server.LogLevel] {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("logLevel"), server.LogLevel, "invalid log level"))
	}

	// Validate log format
	if server.LogFormat != "" && server.LogFormat != "logfmt" && server.LogFormat != "json" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("logFormat"), server.LogFormat, "log format must be logfmt or json"))
	}

	// Validate port ranges
	if server.HTTPListenPort < 0 || server.HTTPListenPort > 65535 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("httpListenPort"), server.HTTPListenPort, "port must be between 0 and 65535"))
	}
	if server.GRPCListenPort < 0 || server.GRPCListenPort > 65535 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("grpcListenPort"), server.GRPCListenPort, "port must be between 0 and 65535"))
	}

	// Validate durations
	durations := map[string]string{
		"httpIdleTimeout":         server.HTTPIdleTimeout,
		"httpWriteTimeout":        server.HTTPWriteTimeout,
		"gracefulShutdownTimeout": server.GracefulShutdownTimeout,
	}
	for field, duration := range durations {
		if duration != "" && !isValidDuration(duration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child(field), duration, "invalid duration format"))
		}
	}

	// Validate message sizes
	if server.GRPCMaxRecvMsgSize != "" {
		if _, err := parseByteSize(server.GRPCMaxRecvMsgSize); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("grpcMaxRecvMsgSize"), server.GRPCMaxRecvMsgSize, "invalid byte size format"))
		}
	}
	if server.GRPCMaxSendMsgSize != "" {
		if _, err := parseByteSize(server.GRPCMaxSendMsgSize); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("grpcMaxSendMsgSize"), server.GRPCMaxSendMsgSize, "invalid byte size format"))
		}
	}

	return allErrs
}

// validateOverridesConfig validates overrides configuration
func (r *TempoConfig) validateOverridesConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	overrides := r.Spec.Overrides

	// Validate reload period
	if overrides.OverridesReloadPeriod != "" && !isValidDuration(overrides.OverridesReloadPeriod) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("overridesReloadPeriod"), overrides.OverridesReloadPeriod, "invalid duration format"))
	}

	// Validate default tenant overrides
	if overrides.Defaults != nil {
		allErrs = append(allErrs, r.validateTenantOverrides(fldPath.Child("defaults"), overrides.Defaults)...)
	}

	// Validate per-tenant overrides
	for tenant, config := range overrides.PerTenantOverrides {
		allErrs = append(allErrs, r.validateTenantOverrides(fldPath.Child("perTenantOverrides").Key(tenant), config)...)
	}

	return allErrs
}

// validateTenantOverrides validates tenant-specific overrides
func (r *TempoConfig) validateTenantOverrides(fldPath *field.Path, tenant *TenantOverrides) field.ErrorList {
	var allErrs field.ErrorList

	// Validate byte sizes
	if tenant.MaxBytesPerTrace != "" {
		if _, err := parseByteSize(tenant.MaxBytesPerTrace); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("maxBytesPerTrace"), tenant.MaxBytesPerTrace, "invalid byte size format"))
		}
	}
	if tenant.MaxBytesPerTagValuesQuery != "" {
		if _, err := parseByteSize(tenant.MaxBytesPerTagValuesQuery); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("maxBytesPerTagValuesQuery"), tenant.MaxBytesPerTagValuesQuery, "invalid byte size format"))
		}
	}

	// Validate durations
	durations := map[string]string{
		"maxSearchDuration": tenant.MaxSearchDuration,
		"blockRetention":    tenant.BlockRetention,
	}
	for field, duration := range durations {
		if duration != "" && !isValidDuration(duration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child(field), duration, "invalid duration format"))
		}
	}

	// Validate ingestion burst is greater than or equal to rate
	if tenant.IngestionBurstSizeBytes < tenant.IngestionRateLimitBytes {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ingestionBurstSizeBytes"), tenant.IngestionBurstSizeBytes, "burst size must be greater than or equal to rate limit"))
	}

	return allErrs
}

// validateMetricsGeneratorConfig validates metrics generator configuration
func (r *TempoConfig) validateMetricsGeneratorConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	metrics := r.Spec.Metrics

	// Validate durations
	durations := map[string]string{
		"metricsFlushInterval":     metrics.MetricsFlushInterval,
		"remoteWriteFlushDeadline": metrics.RemoteWriteFlushDeadline,
	}
	for field, duration := range durations {
		if duration != "" && !isValidDuration(duration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child(field), duration, "invalid duration format"))
		}
	}

	// Validate ring configuration
	if metrics.RingConfig != nil {
		ringPath := fldPath.Child("ringConfig")
		if metrics.RingConfig.HeartbeatPeriod != "" && !isValidDuration(metrics.RingConfig.HeartbeatPeriod) {
			allErrs = append(allErrs, field.Invalid(ringPath.Child("heartbeatPeriod"), metrics.RingConfig.HeartbeatPeriod, "invalid duration format"))
		}
		if metrics.RingConfig.HeartbeatTimeout != "" && !isValidDuration(metrics.RingConfig.HeartbeatTimeout) {
			allErrs = append(allErrs, field.Invalid(ringPath.Child("heartbeatTimeout"), metrics.RingConfig.HeartbeatTimeout, "invalid duration format"))
		}
	}

	// Validate processor configuration
	if metrics.Processor != nil {
		// Validate service graphs
		if metrics.Processor.ServiceGraphs != nil && metrics.Processor.ServiceGraphs.Enabled {
			sgPath := fldPath.Child("processor").Child("serviceGraphs")
			if metrics.Processor.ServiceGraphs.WaitTime != "" && !isValidDuration(metrics.Processor.ServiceGraphs.WaitTime) {
				allErrs = append(allErrs, field.Invalid(sgPath.Child("waitTime"), metrics.Processor.ServiceGraphs.WaitTime, "invalid duration format"))
			}
		}

		// Validate span metrics
		if metrics.Processor.SpanMetrics != nil && metrics.Processor.SpanMetrics.Enabled {
			smPath := fldPath.Child("processor").Child("spanMetrics")
			if metrics.Processor.SpanMetrics.AggregationInterval != "" && !isValidDuration(metrics.Processor.SpanMetrics.AggregationInterval) {
				allErrs = append(allErrs, field.Invalid(smPath.Child("aggregationInterval"), metrics.Processor.SpanMetrics.AggregationInterval, "invalid duration format"))
			}
		}
	}

	// Validate storage configuration
	if metrics.Storage != nil && len(metrics.Storage.RemoteWrite) > 0 {
		for i, rw := range metrics.Storage.RemoteWrite {
			rwPath := fldPath.Child("storage").Child("remoteWrite").Index(i)
			if rw.URL == "" {
				allErrs = append(allErrs, field.Required(rwPath.Child("url"), "remote write URL is required"))
			} else if _, err := url.Parse(rw.URL); err != nil {
				allErrs = append(allErrs, field.Invalid(rwPath.Child("url"), rw.URL, "invalid URL"))
			}
			if rw.RemoteTimeout != "" && !isValidDuration(rw.RemoteTimeout) {
				allErrs = append(allErrs, field.Invalid(rwPath.Child("remoteTimeout"), rw.RemoteTimeout, "invalid duration format"))
			}
		}
	}

	// Validate registry configuration
	if metrics.Registry != nil {
		if metrics.Registry.StaleDuration != "" && !isValidDuration(metrics.Registry.StaleDuration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("registry").Child("staleDuration"), metrics.Registry.StaleDuration, "invalid duration format"))
		}
	}

	return allErrs
}

// validateEndpoint validates an endpoint string
func (r *TempoConfig) validateEndpoint(fldPath *field.Path, endpoint string) field.ErrorList {
	var allErrs field.ErrorList

	if endpoint == "" {
		return allErrs
	}

	// Check if it's a valid host:port format
	parts := strings.Split(endpoint, ":")
	if len(parts) != 2 {
		allErrs = append(allErrs, field.Invalid(fldPath, endpoint, "endpoint must be in host:port format"))
		return allErrs
	}

	// Validate port
	_, err := parsePort(parts[1])
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, endpoint, "invalid port number"))
	}

	return allErrs
}

// Helper functions

// isValidDuration checks if a duration string is valid
func isValidDuration(duration string) bool {
	if duration == "" || duration == "0" || duration == "0s" {
		return true
	}
	_, err := time.ParseDuration(duration)
	return err == nil
}

// parseByteSize parses a byte size string (e.g., "10MB", "1GB")
func parseByteSize(size string) (int64, error) {
	// Simple implementation - you might want to use a proper library
	size = strings.TrimSpace(size)
	if size == "" || size == "0" {
		return 0, nil
	}

	// Handle pure numeric values as bytes
	if v, err := fmt.Sscanf(size, "%d", new(int64)); err == nil && v == 1 {
		var bytes int64
		fmt.Sscanf(size, "%d", &bytes)
		return bytes, nil
	}

	// Handle units
	units := map[string]int64{
		"B":   1,
		"KB":  1024,
		"MB":  1024 * 1024,
		"GB":  1024 * 1024 * 1024,
		"TB":  1024 * 1024 * 1024 * 1024,
		"KiB": 1024,
		"MiB": 1024 * 1024,
		"GiB": 1024 * 1024 * 1024,
		"TiB": 1024 * 1024 * 1024 * 1024,
	}

	for unit, multiplier := range units {
		if strings.HasSuffix(strings.ToUpper(size), unit) {
			numStr := strings.TrimSuffix(strings.ToUpper(size), unit)
			num, err := fmt.Sscanf(numStr, "%d", new(int64))
			if err != nil || num != 1 {
				return 0, fmt.Errorf("invalid byte size format")
			}
			var value int64
			fmt.Sscanf(numStr, "%d", &value)
			return value * multiplier, nil
		}
	}

	return 0, fmt.Errorf("invalid byte size format")
}

// parsePort parses a port string and validates it
func parsePort(port string) (int, error) {
	var p int
	if _, err := fmt.Sscanf(port, "%d", &p); err != nil {
		return 0, err
	}
	if p < 0 || p > 65535 {
		return 0, fmt.Errorf("port out of range")
	}
	return p, nil
}
