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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var lokiconfiglog = logf.Log.WithName("lokiconfig-resource")

func (r *LokiConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-observability-io-v1beta1-lokiconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=lokiconfigs,verbs=create;update,versions=v1beta1,name=mlokiconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &LokiConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *LokiConfig) Default() {
	lokiconfiglog.Info("default", "name", r.Name)

	// Set default values
	if r.Spec.Server == nil {
		r.Spec.Server = &ServerConfig{}
	}
	if r.Spec.Server.HTTPListenPort == 0 {
		r.Spec.Server.HTTPListenPort = 3100
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

	// Set default multi-tenancy headers
	if r.Spec.MultiTenancy != nil && r.Spec.MultiTenancy.Enabled {
		if r.Spec.MultiTenancy.TenantIDHeader == "" {
			r.Spec.MultiTenancy.TenantIDHeader = "X-Scope-OrgID"
		}
		if r.Spec.MultiTenancy.TenantIDLabel == "" {
			r.Spec.MultiTenancy.TenantIDLabel = "tenant_id"
		}
	}

	// Set default schema configurations
	for i := range r.Spec.SchemaConfig.Configs {
		config := &r.Spec.SchemaConfig.Configs[i]
		
		// Default index configuration
		if config.Index == nil {
			config.Index = &IndexConfig{}
		}
		if config.Index.Period == "" {
			config.Index.Period = "24h"
		}
		if config.Index.Prefix == "" {
			config.Index.Prefix = "index_"
		}

		// Default chunks configuration
		if config.Chunks == nil {
			config.Chunks = &ChunksConfig{}
		}
		if config.Chunks.Period == "" {
			config.Chunks.Period = "24h"
		}
		if config.Chunks.Prefix == "" {
			config.Chunks.Prefix = "chunks_"
		}
	}

	// Set default ingester configuration
	if r.Spec.Ingester != nil {
		if r.Spec.Ingester.ChunkIdlePeriod == "" {
			r.Spec.Ingester.ChunkIdlePeriod = "30m"
		}
		if r.Spec.Ingester.ChunkBlockSize == 0 {
			r.Spec.Ingester.ChunkBlockSize = 262144
		}
		if r.Spec.Ingester.ChunkTargetSize == 0 {
			r.Spec.Ingester.ChunkTargetSize = 1572864
		}
		if r.Spec.Ingester.ChunkEncoding == "" {
			r.Spec.Ingester.ChunkEncoding = "gzip"
		}
		if r.Spec.Ingester.MaxChunkAge == "" {
			r.Spec.Ingester.MaxChunkAge = "2h"
		}

		// Default WAL configuration
		if r.Spec.Ingester.WAL != nil && r.Spec.Ingester.WAL.Enabled {
			if r.Spec.Ingester.WAL.Dir == "" {
				r.Spec.Ingester.WAL.Dir = "/loki/wal"
			}
			if r.Spec.Ingester.WAL.CheckpointDuration == "" {
				r.Spec.Ingester.WAL.CheckpointDuration = "5m"
			}
			if r.Spec.Ingester.WAL.ReplayMemoryCeiling == "" {
				r.Spec.Ingester.WAL.ReplayMemoryCeiling = "4GB"
			}
		}

		// Default lifecycler configuration
		if r.Spec.Ingester.Lifecycler != nil {
			if r.Spec.Ingester.Lifecycler.NumTokens == 0 {
				r.Spec.Ingester.Lifecycler.NumTokens = 128
			}
			if r.Spec.Ingester.Lifecycler.HeartbeatPeriod == "" {
				r.Spec.Ingester.Lifecycler.HeartbeatPeriod = "5s"
			}
			if r.Spec.Ingester.Lifecycler.JoinAfter == "" {
				r.Spec.Ingester.Lifecycler.JoinAfter = "0s"
			}
			if r.Spec.Ingester.Lifecycler.MinReadyDuration == "" {
				r.Spec.Ingester.Lifecycler.MinReadyDuration = "60s"
			}
			if r.Spec.Ingester.Lifecycler.FinalSleep == "" {
				r.Spec.Ingester.Lifecycler.FinalSleep = "0s"
			}

			// Default ring configuration
			if r.Spec.Ingester.Lifecycler.Ring != nil {
				if r.Spec.Ingester.Lifecycler.Ring.ReplicationFactor == 0 {
					r.Spec.Ingester.Lifecycler.Ring.ReplicationFactor = 3
				}
				if r.Spec.Ingester.Lifecycler.Ring.KVStore != nil && r.Spec.Ingester.Lifecycler.Ring.KVStore.Store == "" {
					r.Spec.Ingester.Lifecycler.Ring.KVStore.Store = "inmemory"
				}
			}
		}
	}

	// Set default limits
	if r.Spec.Limits != nil {
		if r.Spec.Limits.IngestionRateMB == 0 {
			r.Spec.Limits.IngestionRateMB = 4
		}
		if r.Spec.Limits.IngestionBurstSizeMB == 0 {
			r.Spec.Limits.IngestionBurstSizeMB = 6
		}
		if r.Spec.Limits.MaxLabelNameLength == 0 {
			r.Spec.Limits.MaxLabelNameLength = 1024
		}
		if r.Spec.Limits.MaxLabelValueLength == 0 {
			r.Spec.Limits.MaxLabelValueLength = 2048
		}
		if r.Spec.Limits.MaxLabelNamesPerSeries == 0 {
			r.Spec.Limits.MaxLabelNamesPerSeries = 30
		}
		if r.Spec.Limits.RejectOldSamplesMaxAge == "" {
			r.Spec.Limits.RejectOldSamplesMaxAge = "168h"
		}
		if r.Spec.Limits.CreationGracePeriod == "" {
			r.Spec.Limits.CreationGracePeriod = "10m"
		}
		if r.Spec.Limits.MaxStreamsPerUser == 0 {
			r.Spec.Limits.MaxStreamsPerUser = 5000
		}
		if r.Spec.Limits.MaxGlobalStreamsPerUser == 0 {
			r.Spec.Limits.MaxGlobalStreamsPerUser = 5000
		}
		if r.Spec.Limits.MaxChunksPerQuery == 0 {
			r.Spec.Limits.MaxChunksPerQuery = 2000000
		}
		if r.Spec.Limits.MaxQueryLookback == "" {
			r.Spec.Limits.MaxQueryLookback = "0s"
		}
		if r.Spec.Limits.MaxQueryLength == "" {
			r.Spec.Limits.MaxQueryLength = "721h"
		}
		if r.Spec.Limits.MaxQueryParallelism == 0 {
			r.Spec.Limits.MaxQueryParallelism = 32
		}
		if r.Spec.Limits.MaxEntriesLimitPerQuery == 0 {
			r.Spec.Limits.MaxEntriesLimitPerQuery = 5000
		}
		if r.Spec.Limits.MaxCacheFreshnessPerQuery == "" {
			r.Spec.Limits.MaxCacheFreshnessPerQuery = "1m"
		}
		if r.Spec.Limits.MaxStreamsMatchersPerQuery == 0 {
			r.Spec.Limits.MaxStreamsMatchersPerQuery = 1000
		}
		if r.Spec.Limits.MaxConcurrentTailRequests == 0 {
			r.Spec.Limits.MaxConcurrentTailRequests = 10
		}
	}

	// Set default querier configuration
	if r.Spec.Querier != nil {
		if r.Spec.Querier.MaxConcurrent == 0 {
			r.Spec.Querier.MaxConcurrent = 10
		}
		if r.Spec.Querier.TailMaxDuration == "" {
			r.Spec.Querier.TailMaxDuration = "1h"
		}
		if r.Spec.Querier.QueryTimeout == "" {
			r.Spec.Querier.QueryTimeout = "1m"
		}
		if r.Spec.Querier.QueryIngestersWithin == "" {
			r.Spec.Querier.QueryIngestersWithin = "3h"
		}

		if r.Spec.Querier.Engine != nil {
			if r.Spec.Querier.Engine.Timeout == "" {
				r.Spec.Querier.Engine.Timeout = "5m"
			}
			if r.Spec.Querier.Engine.MaxLookBackPeriod == "" {
				r.Spec.Querier.Engine.MaxLookBackPeriod = "30d"
			}
		}
	}

	// Set default query frontend configuration
	if r.Spec.QueryFrontend != nil {
		if r.Spec.QueryFrontend.MaxOutstandingPerTenant == 0 {
			r.Spec.QueryFrontend.MaxOutstandingPerTenant = 2048
		}
		if r.Spec.QueryFrontend.MaxRetries == 0 {
			r.Spec.QueryFrontend.MaxRetries = 5
		}
		if r.Spec.QueryFrontend.SplitQueriesByInterval == "" {
			r.Spec.QueryFrontend.SplitQueriesByInterval = "30m"
		}
		if r.Spec.QueryFrontend.SchedulerWorkerConcurrency == 0 {
			r.Spec.QueryFrontend.SchedulerWorkerConcurrency = 5
		}
	}

	// Set default compactor configuration
	if r.Spec.Compactor != nil {
		if r.Spec.Compactor.WorkingDirectory == "" {
			r.Spec.Compactor.WorkingDirectory = "/loki/compactor"
		}
		if r.Spec.Compactor.CompactionInterval == "" {
			r.Spec.Compactor.CompactionInterval = "10m"
		}
		if r.Spec.Compactor.RetentionDeleteDelay == "" {
			r.Spec.Compactor.RetentionDeleteDelay = "2h"
		}
		if r.Spec.Compactor.RetentionDeleteWorkerCount == 0 {
			r.Spec.Compactor.RetentionDeleteWorkerCount = 150
		}
		if r.Spec.Compactor.DeleteRequestCancelPeriod == "" {
			r.Spec.Compactor.DeleteRequestCancelPeriod = "24h"
		}
		if r.Spec.Compactor.MaxCompactionParallelism == 0 {
			r.Spec.Compactor.MaxCompactionParallelism = 1
		}
	}

	// Set default ruler configuration
	if r.Spec.Ruler != nil {
		if r.Spec.Ruler.EvaluationInterval == "" {
			r.Spec.Ruler.EvaluationInterval = "1m"
		}
		if r.Spec.Ruler.PollInterval == "" {
			r.Spec.Ruler.PollInterval = "1m"
		}
		if r.Spec.Ruler.AlertmanagerRefreshInterval == "" {
			r.Spec.Ruler.AlertmanagerRefreshInterval = "1m"
		}
		if r.Spec.Ruler.NotificationQueueCapacity == 0 {
			r.Spec.Ruler.NotificationQueueCapacity = 10000
		}
		if r.Spec.Ruler.NotificationTimeout == "" {
			r.Spec.Ruler.NotificationTimeout = "10s"
		}
		if r.Spec.Ruler.SearchPendingFor == "" {
			r.Spec.Ruler.SearchPendingFor = "5m"
		}
		if r.Spec.Ruler.FlushPeriod == "" {
			r.Spec.Ruler.FlushPeriod = "1m"
		}

		// Default ruler storage
		if r.Spec.Ruler.Storage != nil && r.Spec.Ruler.Storage.Local != nil {
			if r.Spec.Ruler.Storage.Local.Directory == "" {
				r.Spec.Ruler.Storage.Local.Directory = "/loki/rules"
			}
		}
	}

	// Set default table manager configuration
	if r.Spec.TableManager != nil {
		if r.Spec.TableManager.PollInterval == "" {
			r.Spec.TableManager.PollInterval = "10m"
		}
		if r.Spec.TableManager.CreationGracePeriod == "" {
			r.Spec.TableManager.CreationGracePeriod = "10m"
		}
	}

	// Set default cache configuration
	if r.Spec.Storage.Cache != nil {
		// Index cache defaults
		if r.Spec.Storage.Cache.EnableIndexCache && r.Spec.Storage.Cache.IndexCache != nil {
			if r.Spec.Storage.Cache.IndexCache.Type == "inmemory" && r.Spec.Storage.Cache.IndexCache.InMemorySize == "" {
				r.Spec.Storage.Cache.IndexCache.InMemorySize = "500MB"
			}
			if r.Spec.Storage.Cache.IndexCache.Memcached != nil {
				if r.Spec.Storage.Cache.IndexCache.Memcached.Timeout == "" {
					r.Spec.Storage.Cache.IndexCache.Memcached.Timeout = "100ms"
				}
				if r.Spec.Storage.Cache.IndexCache.Memcached.MaxIdleConns == 0 {
					r.Spec.Storage.Cache.IndexCache.Memcached.MaxIdleConns = 16
				}
			}
		}

		// Results cache defaults
		if r.Spec.Storage.Cache.EnableResultsCache && r.Spec.Storage.Cache.ResultsCache != nil {
			if r.Spec.Storage.Cache.ResultsCache.MaxFreshness == "" {
				r.Spec.Storage.Cache.ResultsCache.MaxFreshness = "10m"
			}
		}
	}

	// Set default BoltDB configuration
	if r.Spec.Storage.BoltDB != nil && r.Spec.Storage.BoltDB.Directory == "" {
		r.Spec.Storage.BoltDB.Directory = "/loki/index"
	}

	// Set default filesystem storage
	if r.Spec.Storage.Type == "filesystem" && r.Spec.Storage.Filesystem != nil {
		if r.Spec.Storage.Filesystem.Directory == "" {
			r.Spec.Storage.Filesystem.Directory = "/loki/chunks"
		}
	}
}

// +kubebuilder:webhook:path=/validate-observability-io-v1beta1-lokiconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=lokiconfigs,verbs=create;update,versions=v1beta1,name=vlokiconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &LokiConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *LokiConfig) ValidateCreate() (admission.Warnings, error) {
	lokiconfiglog.Info("validate create", "name", r.Name)

	var allErrs field.ErrorList

	// Validate storage configuration
	if err := r.validateStorageConfig(field.NewPath("spec").Child("storage")); err != nil {
		allErrs = append(allErrs, err...)
	}

	// Validate schema configuration
	if err := r.validateSchemaConfig(field.NewPath("spec").Child("schemaConfig")); err != nil {
		allErrs = append(allErrs, err...)
	}

	// Validate limits configuration
	if r.Spec.Limits != nil {
		if err := r.validateLimitsConfig(field.NewPath("spec").Child("limits")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate authentication configuration
	if r.Spec.Auth != nil {
		if err := r.validateAuthConfig(field.NewPath("spec").Child("auth")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate multi-tenancy configuration
	if r.Spec.MultiTenancy != nil && r.Spec.MultiTenancy.Enabled {
		if err := r.validateMultiTenancyConfig(field.NewPath("spec").Child("multiTenancy")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	// Validate component configurations
	if r.Spec.Ingester != nil {
		if err := r.validateIngesterConfig(field.NewPath("spec").Child("ingester")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	if r.Spec.Ruler != nil {
		if err := r.validateRulerConfig(field.NewPath("spec").Child("ruler")); err != nil {
			allErrs = append(allErrs, err...)
		}
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, allErrs.ToAggregate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LokiConfig) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	lokiconfiglog.Info("validate update", "name", r.Name)

	oldConfig, ok := old.(*LokiConfig)
	if !ok {
		return nil, fmt.Errorf("expected LokiConfig but got %T", old)
	}

	var allErrs field.ErrorList

	// Validate immutable fields
	if oldConfig.Spec.Storage.Type != r.Spec.Storage.Type {
		allErrs = append(allErrs, field.Forbidden(
			field.NewPath("spec").Child("storage").Child("type"),
			"storage type cannot be changed after creation"))
	}

	// Validate schema changes
	if err := r.validateSchemaUpdate(oldConfig.Spec.SchemaConfig, r.Spec.SchemaConfig); err != nil {
		allErrs = append(allErrs, err...)
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
func (r *LokiConfig) ValidateDelete() (admission.Warnings, error) {
	lokiconfiglog.Info("validate delete", "name", r.Name)
	// No special validation for delete
	return nil, nil
}

// validateStorageConfig validates the storage configuration
func (r *LokiConfig) validateStorageConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	switch r.Spec.Storage.Type {
	case "s3":
		if r.Spec.Storage.S3 == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("s3"), "S3 configuration is required when storage type is s3"))
		} else {
			if r.Spec.Storage.S3.BucketName == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("s3").Child("bucketName"), "bucket name is required"))
			}
			if r.Spec.Storage.S3.Region == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("s3").Child("region"), "region is required"))
			}
		}
	case "gcs":
		if r.Spec.Storage.GCS == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("gcs"), "GCS configuration is required when storage type is gcs"))
		} else {
			if r.Spec.Storage.GCS.BucketName == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("gcs").Child("bucketName"), "bucket name is required"))
			}
		}
	case "azure":
		if r.Spec.Storage.Azure == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("azure"), "Azure configuration is required when storage type is azure"))
		} else {
			if r.Spec.Storage.Azure.ContainerName == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("azure").Child("containerName"), "container name is required"))
			}
			if r.Spec.Storage.Azure.AccountName == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("azure").Child("accountName"), "account name is required"))
			}
			if !r.Spec.Storage.Azure.UseManagedIdentity && r.Spec.Storage.Azure.AccountKey == nil {
				allErrs = append(allErrs, field.Required(fldPath.Child("azure").Child("accountKey"), "account key is required when not using managed identity"))
			}
		}
	case "filesystem":
		if r.Spec.Storage.Filesystem == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("filesystem"), "Filesystem configuration is required when storage type is filesystem"))
		}
	}

	// Validate cache configuration
	if r.Spec.Storage.Cache != nil {
		if r.Spec.Storage.Cache.EnableIndexCache && r.Spec.Storage.Cache.IndexCache == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("cache").Child("indexCache"), "index cache configuration is required when index cache is enabled"))
		}
		if r.Spec.Storage.Cache.EnableChunkCache && r.Spec.Storage.Cache.ChunkCache == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("cache").Child("chunkCache"), "chunk cache configuration is required when chunk cache is enabled"))
		}
		if r.Spec.Storage.Cache.EnableResultsCache && r.Spec.Storage.Cache.ResultsCache == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("cache").Child("resultsCache"), "results cache configuration is required when results cache is enabled"))
		}
	}

	return allErrs
}

// validateSchemaConfig validates the schema configuration
func (r *LokiConfig) validateSchemaConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if len(r.Spec.SchemaConfig.Configs) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("configs"), "at least one schema config is required"))
		return allErrs
	}

	// Parse and validate schema dates
	var prevDate time.Time
	for i, config := range r.Spec.SchemaConfig.Configs {
		configPath := fldPath.Child("configs").Index(i)

		// Parse date
		date, err := time.Parse(time.RFC3339, config.From)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(configPath.Child("from"), config.From, "invalid RFC3339 date format"))
			continue
		}

		// Check date ordering
		if i > 0 && !date.After(prevDate) {
			allErrs = append(allErrs, field.Invalid(configPath.Child("from"), config.From, "schema dates must be in ascending order"))
		}
		prevDate = date

		// Validate store and object store compatibility
		if config.Store == "boltdb" && config.ObjectStore != "filesystem" {
			allErrs = append(allErrs, field.Invalid(configPath.Child("objectStore"), config.ObjectStore, "boltdb store requires filesystem object store"))
		}

		// Validate schema version
		switch config.Schema {
		case "v11":
			if config.Store == "tsdb" {
				allErrs = append(allErrs, field.Invalid(configPath.Child("store"), config.Store, "tsdb store requires schema v12 or higher"))
			}
		case "v12", "v13":
			// Valid for all store types
		default:
			allErrs = append(allErrs, field.Invalid(configPath.Child("schema"), config.Schema, "unsupported schema version"))
		}
	}

	return allErrs
}

// validateSchemaUpdate validates schema configuration updates
func (r *LokiConfig) validateSchemaUpdate(oldSchema, newSchema SchemaConfig) field.ErrorList {
	var allErrs field.ErrorList
	fldPath := field.NewPath("spec").Child("schemaConfig")

	// Cannot remove schema entries
	if len(newSchema.Configs) < len(oldSchema.Configs) {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("configs"), "cannot remove schema config entries"))
		return allErrs
	}

	// Existing entries must not change (except adding new fields)
	for i := 0; i < len(oldSchema.Configs); i++ {
		if i >= len(newSchema.Configs) {
			break
		}

		oldConfig := oldSchema.Configs[i]
		newConfig := newSchema.Configs[i]
		configPath := fldPath.Child("configs").Index(i)

		if oldConfig.From != newConfig.From {
			allErrs = append(allErrs, field.Forbidden(configPath.Child("from"), "cannot change schema start date"))
		}
		if oldConfig.Store != newConfig.Store {
			allErrs = append(allErrs, field.Forbidden(configPath.Child("store"), "cannot change store type"))
		}
		if oldConfig.ObjectStore != newConfig.ObjectStore {
			allErrs = append(allErrs, field.Forbidden(configPath.Child("objectStore"), "cannot change object store type"))
		}
		if oldConfig.Schema != newConfig.Schema {
			allErrs = append(allErrs, field.Forbidden(configPath.Child("schema"), "cannot change schema version"))
		}
	}

	return allErrs
}

// validateLimitsConfig validates the limits configuration
func (r *LokiConfig) validateLimitsConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	limits := r.Spec.Limits

	// Validate durations
	durations := map[string]string{
		"rejectOldSamplesMaxAge":    limits.RejectOldSamplesMaxAge,
		"creationGracePeriod":       limits.CreationGracePeriod,
		"maxQueryLookback":          limits.MaxQueryLookback,
		"maxQueryLength":            limits.MaxQueryLength,
		"maxCacheFreshnessPerQuery": limits.MaxCacheFreshnessPerQuery,
		"splitQueriesByInterval":    limits.SplitQueriesByInterval,
		"retentionPeriod":          limits.RetentionPeriod,
	}

	for field, duration := range durations {
		if duration != "" && !isValidDuration(duration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child(field), duration, "invalid duration format"))
		}
	}

	// Validate retention streams
	for i, stream := range limits.RetentionStream {
		streamPath := fldPath.Child("retentionStream").Index(i)
		
		if stream.Selector == "" {
			allErrs = append(allErrs, field.Required(streamPath.Child("selector"), "selector is required"))
		}
		
		if !isValidDuration(stream.Period) {
			allErrs = append(allErrs, field.Invalid(streamPath.Child("period"), stream.Period, "invalid duration format"))
		}
	}

	// Validate logical constraints
	if limits.IngestionBurstSizeMB < limits.IngestionRateMB {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ingestionBurstSizeMB"), limits.IngestionBurstSizeMB, "burst size must be greater than or equal to ingestion rate"))
	}

	if limits.MaxGlobalStreamsPerUser < limits.MaxStreamsPerUser {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("maxGlobalStreamsPerUser"), limits.MaxGlobalStreamsPerUser, "global streams limit must be greater than or equal to per-user streams limit"))
	}

	return allErrs
}

// validateAuthConfig validates the authentication configuration
func (r *LokiConfig) validateAuthConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	switch r.Spec.Auth.Type {
	case "basic":
		if r.Spec.Auth.Basic == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("basic"), "basic auth configuration is required when auth type is basic"))
		} else {
			if r.Spec.Auth.Basic.Username == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("basic").Child("username"), "username is required"))
			}
			if r.Spec.Auth.Basic.Password == nil {
				allErrs = append(allErrs, field.Required(fldPath.Child("basic").Child("password"), "password is required"))
			}
		}
	case "oidc":
		if r.Spec.Auth.OIDC == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("oidc"), "OIDC configuration is required when auth type is oidc"))
		} else {
			if r.Spec.Auth.OIDC.IssuerURL == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("oidc").Child("issuerUrl"), "issuer URL is required"))
			}
			if r.Spec.Auth.OIDC.ClientID == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("oidc").Child("clientId"), "client ID is required"))
			}
		}
	case "header":
		if r.Spec.Auth.Header == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("header"), "header auth configuration is required when auth type is header"))
		} else {
			if r.Spec.Auth.Header.HeaderName == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("header").Child("headerName"), "header name is required"))
			}
		}
	}

	return allErrs
}

// validateMultiTenancyConfig validates the multi-tenancy configuration
func (r *LokiConfig) validateMultiTenancyConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	// If multi-tenancy is enabled, auth should also be enabled
	if r.Spec.MultiTenancy.Enabled && !r.Spec.MultiTenancy.AuthEnabled {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("authEnabled"), false, "authentication must be enabled when multi-tenancy is enabled"))
	}

	// Validate tenant configurations
	tenantIDs := make(map[string]bool)
	for i, tenant := range r.Spec.MultiTenancy.Tenants {
		tenantPath := fldPath.Child("tenants").Index(i)

		if tenant.ID == "" {
			allErrs = append(allErrs, field.Required(tenantPath.Child("id"), "tenant ID is required"))
		} else if tenantIDs[tenant.ID] {
			allErrs = append(allErrs, field.Duplicate(tenantPath.Child("id"), tenant.ID))
		} else {
			tenantIDs[tenant.ID] = true
		}

		// Validate tenant-specific limits
		if tenant.Limits != nil {
			if err := r.validateLimitsConfig(tenantPath.Child("limits")); err != nil {
				allErrs = append(allErrs, err...)
			}
		}
	}

	return allErrs
}

// validateIngesterConfig validates the ingester configuration
func (r *LokiConfig) validateIngesterConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	ingester := r.Spec.Ingester

	// Validate durations
	durations := map[string]string{
		"chunkIdlePeriod":  ingester.ChunkIdlePeriod,
		"maxChunkAge":      ingester.MaxChunkAge,
		"flushCheckPeriod": ingester.FlushCheckPeriod,
	}

	for field, duration := range durations {
		if duration != "" && !isValidDuration(duration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child(field), duration, "invalid duration format"))
		}
	}

	// Validate chunk size constraints
	if ingester.ChunkTargetSize > 0 && ingester.ChunkBlockSize > 0 {
		if ingester.ChunkTargetSize < ingester.ChunkBlockSize {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("chunkTargetSize"), ingester.ChunkTargetSize, "chunk target size must be greater than chunk block size"))
		}
	}

	// Validate WAL configuration
	if ingester.WAL != nil && ingester.WAL.Enabled {
		if ingester.WAL.Dir == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("wal").Child("dir"), "WAL directory is required when WAL is enabled"))
		}
		if ingester.WAL.CheckpointDuration != "" && !isValidDuration(ingester.WAL.CheckpointDuration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("wal").Child("checkpointDuration"), ingester.WAL.CheckpointDuration, "invalid duration format"))
		}
	}

	// Validate lifecycler configuration
	if ingester.Lifecycler != nil {
		lifecyclerPath := fldPath.Child("lifecycler")
		
		// Validate durations
		lifecyclerDurations := map[string]string{
			"heartbeatPeriod": ingester.Lifecycler.HeartbeatPeriod,
			"joinAfter":       ingester.Lifecycler.JoinAfter,
			"minReadyDuration": ingester.Lifecycler.MinReadyDuration,
			"finalSleep":      ingester.Lifecycler.FinalSleep,
		}

		for field, duration := range lifecyclerDurations {
			if duration != "" && !isValidDuration(duration) {
				allErrs = append(allErrs, field.Invalid(lifecyclerPath.Child(field), duration, "invalid duration format"))
			}
		}

		// Validate ring configuration
		if ingester.Lifecycler.Ring != nil && ingester.Lifecycler.Ring.KVStore != nil {
			kvPath := lifecyclerPath.Child("ring").Child("kvStore")
			
			switch ingester.Lifecycler.Ring.KVStore.Store {
			case "consul":
				if ingester.Lifecycler.Ring.KVStore.Consul == nil {
					allErrs = append(allErrs, field.Required(kvPath.Child("consul"), "consul configuration is required when store is consul"))
				}
			case "etcd":
				if ingester.Lifecycler.Ring.KVStore.Etcd == nil {
					allErrs = append(allErrs, field.Required(kvPath.Child("etcd"), "etcd configuration is required when store is etcd"))
				}
			}
		}
	}

	return allErrs
}

// validateRulerConfig validates the ruler configuration
func (r *LokiConfig) validateRulerConfig(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	ruler := r.Spec.Ruler

	// Validate durations
	durations := map[string]string{
		"evaluationInterval":          ruler.EvaluationInterval,
		"pollInterval":               ruler.PollInterval,
		"alertmanagerRefreshInterval": ruler.AlertmanagerRefreshInterval,
		"notificationTimeout":        ruler.NotificationTimeout,
		"searchPendingFor":           ruler.SearchPendingFor,
		"flushPeriod":               ruler.FlushPeriod,
	}

	for field, duration := range durations {
		if duration != "" && !isValidDuration(duration) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child(field), duration, "invalid duration format"))
		}
	}

	// Validate storage configuration
	if ruler.Storage != nil {
		storagePath := fldPath.Child("storage")
		
		switch ruler.Storage.Type {
		case "local":
			if ruler.Storage.Local == nil {
				allErrs = append(allErrs, field.Required(storagePath.Child("local"), "local storage configuration is required when storage type is local"))
			}
		case "s3":
			if ruler.Storage.S3 == nil {
				allErrs = append(allErrs, field.Required(storagePath.Child("s3"), "S3 configuration is required when storage type is s3"))
			}
		case "gcs":
			if ruler.Storage.GCS == nil {
				allErrs = append(allErrs, field.Required(storagePath.Child("gcs"), "GCS configuration is required when storage type is gcs"))
			}
		case "azure":
			if ruler.Storage.Azure == nil {
				allErrs = append(allErrs, field.Required(storagePath.Child("azure"), "Azure configuration is required when storage type is azure"))
			}
		}
	}

	// Validate alertmanager configuration
	if ruler.EnableAlertmanagerV2 && ruler.AlertmanagerURL == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("alertmanagerUrl"), "alertmanager URL is required when alertmanager v2 is enabled"))
	}

	return allErrs
}

// isValidDuration checks if a duration string is valid
func isValidDuration(duration string) bool {
	if duration == "" {
		return true
	}
	_, err := time.ParseDuration(duration)
	return err == nil
}
