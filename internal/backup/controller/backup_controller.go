package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/backup"
	"github.com/gunjanjp/gunj-operator/internal/backup/providers"
	"github.com/gunjanjp/gunj-operator/internal/backup/storage"
	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// BackupControllerName is the name of the backup controller
	BackupControllerName = "backup-controller"
	
	// BackupFinalizer is the finalizer for backups
	BackupFinalizer = "backup.observability.io/finalizer"
	
	// BackupAnnotationPrefix is the prefix for backup annotations
	BackupAnnotationPrefix = "backup.observability.io/"
)

// BackupController handles backup operations
type BackupController struct {
	client         client.Client
	scheme         *runtime.Scheme
	log            logr.Logger
	recorder       record.EventRecorder
	storageManager *storage.Manager
	providers      map[backup.StorageProvider]backup.BackupProvider
	metrics        *backupMetrics
	
	// In-memory backup tracking
	backupsMu sync.RWMutex
	backups   map[string]*backupState
	
	// Configuration
	config BackupControllerConfig
}

// BackupControllerConfig contains controller configuration
type BackupControllerConfig struct {
	// MaxConcurrentBackups is the maximum concurrent backups
	MaxConcurrentBackups int
	
	// DefaultTTL is the default backup TTL
	DefaultTTL time.Duration
	
	// RetryInterval is the retry interval for failed backups
	RetryInterval time.Duration
	
	// MaxRetries is the maximum retries for failed backups
	MaxRetries int
	
	// MetricsEnabled enables metrics collection
	MetricsEnabled bool
	
	// EncryptionEnabled enables backup encryption
	EncryptionEnabled bool
	
	// CompressionEnabled enables backup compression
	CompressionEnabled bool
}

// backupState tracks the state of a backup
type backupState struct {
	spec      *backup.BackupSpec
	status    *backup.BackupStatus
	retries   int
	startTime time.Time
	items     []backup.BackupItem
}

// backupMetrics contains Prometheus metrics
type backupMetrics struct {
	backupsTotal     *prometheus.CounterVec
	backupDuration   *prometheus.HistogramVec
	backupSize       *prometheus.GaugeVec
	backupErrors     *prometheus.CounterVec
	storageUsage     prometheus.Gauge
	activeBackups    prometheus.Gauge
}

// NewBackupController creates a new backup controller
func NewBackupController(
	client client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	recorder record.EventRecorder,
	config BackupControllerConfig,
) (*BackupController, error) {
	bc := &BackupController{
		client:   client,
		scheme:   scheme,
		log:      log.WithName(BackupControllerName),
		recorder: recorder,
		config:   config,
		backups:  make(map[string]*backupState),
		providers: make(map[backup.StorageProvider]backup.BackupProvider),
	}
	
	// Initialize metrics
	if config.MetricsEnabled {
		bc.metrics = bc.initMetrics()
	}
	
	// Initialize storage manager
	storageManager, err := storage.NewManager(log)
	if err != nil {
		return nil, fmt.Errorf("creating storage manager: %w", err)
	}
	bc.storageManager = storageManager
	
	// Initialize providers
	if err := bc.initProviders(); err != nil {
		return nil, fmt.Errorf("initializing providers: %w", err)
	}
	
	return bc, nil
}

// initProviders initializes backup storage providers
func (bc *BackupController) initProviders() error {
	// S3 provider
	s3Provider, err := providers.NewS3Provider(bc.log)
	if err != nil {
		bc.log.Error(err, "Failed to initialize S3 provider")
	} else {
		bc.providers[backup.S3Provider] = s3Provider
	}
	
	// GCS provider
	gcsProvider, err := providers.NewGCSProvider(bc.log)
	if err != nil {
		bc.log.Error(err, "Failed to initialize GCS provider")
	} else {
		bc.providers[backup.GCSProvider] = gcsProvider
	}
	
	// Azure Blob provider
	azureProvider, err := providers.NewAzureBlobProvider(bc.log)
	if err != nil {
		bc.log.Error(err, "Failed to initialize Azure Blob provider")
	} else {
		bc.providers[backup.AzureBlobProvider] = azureProvider
	}
	
	// Local provider (always available)
	localProvider := providers.NewLocalProvider(bc.log)
	bc.providers[backup.LocalProvider] = localProvider
	
	if len(bc.providers) == 0 {
		return fmt.Errorf("no backup providers available")
	}
	
	return nil
}

// initMetrics initializes Prometheus metrics
func (bc *BackupController) initMetrics() *backupMetrics {
	metrics := &backupMetrics{
		backupsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_backup_total",
				Help: "Total number of backups",
			},
			[]string{"type", "status"},
		),
		backupDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "gunj_backup_duration_seconds",
				Help: "Backup duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
			},
			[]string{"type"},
		),
		backupSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gunj_backup_size_bytes",
				Help: "Backup size in bytes",
			},
			[]string{"name"},
		),
		backupErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_backup_errors_total",
				Help: "Total number of backup errors",
			},
			[]string{"type", "reason"},
		),
		storageUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gunj_backup_storage_usage_bytes",
				Help: "Total backup storage usage in bytes",
			},
		),
		activeBackups: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gunj_backup_active",
				Help: "Number of active backups",
			},
		),
	}
	
	// Register metrics
	prometheus.MustRegister(
		metrics.backupsTotal,
		metrics.backupDuration,
		metrics.backupSize,
		metrics.backupErrors,
		metrics.storageUsage,
		metrics.activeBackups,
	)
	
	return metrics
}

// CreateBackup creates a new backup
func (bc *BackupController) CreateBackup(ctx context.Context, spec *backup.BackupSpec) (*backup.BackupStatus, error) {
	backupName := bc.generateBackupName(spec)
	
	bc.log.V(1).Info("Creating backup", "name", backupName, "type", spec.Type)
	
	// Check if backup already exists
	bc.backupsMu.RLock()
	if _, exists := bc.backups[backupName]; exists {
		bc.backupsMu.RUnlock()
		return nil, fmt.Errorf("backup %s already exists", backupName)
	}
	bc.backupsMu.RUnlock()
	
	// Initialize backup state
	state := &backupState{
		spec: spec,
		status: &backup.BackupStatus{
			Phase:          backup.BackupPhasePending,
			StartTimestamp: &metav1.Time{Time: time.Now()},
		},
		startTime: time.Now(),
	}
	
	// Store backup state
	bc.backupsMu.Lock()
	bc.backups[backupName] = state
	bc.backupsMu.Unlock()
	
	// Update metrics
	if bc.metrics != nil {
		bc.metrics.activeBackups.Inc()
	}
	
	// Start backup asynchronously
	go bc.runBackup(ctx, backupName, state)
	
	return state.status, nil
}

// runBackup runs the backup process
func (bc *BackupController) runBackup(ctx context.Context, backupName string, state *backupState) {
	defer func() {
		// Update metrics
		if bc.metrics != nil {
			bc.metrics.activeBackups.Dec()
			duration := time.Since(state.startTime).Seconds()
			bc.metrics.backupDuration.WithLabelValues(string(state.spec.Type)).Observe(duration)
		}
	}()
	
	// Update status to in progress
	state.status.Phase = backup.BackupPhaseInProgress
	
	// Run pre-backup hooks
	if state.spec.Hooks != nil && len(state.spec.Hooks.PreBackup) > 0 {
		bc.log.V(1).Info("Running pre-backup hooks", "backup", backupName)
		if err := bc.runHooks(ctx, state.spec.Hooks.PreBackup); err != nil {
			bc.handleBackupError(state, fmt.Errorf("pre-backup hooks failed: %w", err))
			return
		}
	}
	
	// Collect resources to backup
	bc.log.V(1).Info("Collecting resources", "backup", backupName)
	items, err := bc.collectResources(ctx, state.spec)
	if err != nil {
		bc.handleBackupError(state, fmt.Errorf("collecting resources: %w", err))
		return
	}
	state.items = items
	state.status.BackupItemsCount = len(items)
	
	// Create backup data
	bc.log.V(1).Info("Creating backup data", "backup", backupName, "items", len(items))
	backupData, err := bc.createBackupData(ctx, state)
	if err != nil {
		bc.handleBackupError(state, fmt.Errorf("creating backup data: %w", err))
		return
	}
	
	// Compress if enabled
	if bc.config.CompressionEnabled && state.spec.CompressionConfig != nil {
		bc.log.V(1).Info("Compressing backup", "backup", backupName)
		compressedData, err := bc.compressData(backupData, state.spec.CompressionConfig)
		if err != nil {
			bc.handleBackupError(state, fmt.Errorf("compressing backup: %w", err))
			return
		}
		backupData = compressedData
	}
	
	// Encrypt if enabled
	if bc.config.EncryptionEnabled && state.spec.EncryptionConfig != nil {
		bc.log.V(1).Info("Encrypting backup", "backup", backupName)
		encryptedData, err := bc.encryptData(ctx, backupData, state.spec.EncryptionConfig)
		if err != nil {
			bc.handleBackupError(state, fmt.Errorf("encrypting backup: %w", err))
			return
		}
		backupData = encryptedData
	}
	
	state.status.BackupSizeBytes = int64(len(backupData))
	
	// Upload to storage
	bc.log.V(1).Info("Uploading backup", "backup", backupName, "size", state.status.BackupSizeBytes)
	provider, ok := bc.providers[state.spec.StorageLocation.Provider]
	if !ok {
		bc.handleBackupError(state, fmt.Errorf("provider %s not available", state.spec.StorageLocation.Provider))
		return
	}
	
	if err := provider.Upload(ctx, state.spec, backupData); err != nil {
		bc.handleBackupError(state, fmt.Errorf("uploading backup: %w", err))
		return
	}
	
	// Take volume snapshots if requested
	if state.spec.SnapshotVolumes {
		bc.log.V(1).Info("Taking volume snapshots", "backup", backupName)
		if err := bc.takeVolumeSnapshots(ctx, state); err != nil {
			state.status.Warnings = append(state.status.Warnings, fmt.Sprintf("Volume snapshots failed: %v", err))
		}
	}
	
	// Run post-backup hooks
	if state.spec.Hooks != nil && len(state.spec.Hooks.PostBackup) > 0 {
		bc.log.V(1).Info("Running post-backup hooks", "backup", backupName)
		if err := bc.runHooks(ctx, state.spec.Hooks.PostBackup); err != nil {
			state.status.Warnings = append(state.status.Warnings, fmt.Sprintf("Post-backup hooks failed: %v", err))
		}
	}
	
	// Mark backup as completed
	state.status.Phase = backup.BackupPhaseCompleted
	state.status.CompletionTimestamp = &metav1.Time{Time: time.Now()}
	state.status.BackupLocation = bc.getBackupLocation(backupName, state.spec)
	
	// Update metrics
	if bc.metrics != nil {
		bc.metrics.backupsTotal.WithLabelValues(string(state.spec.Type), "success").Inc()
		bc.metrics.backupSize.WithLabelValues(backupName).Set(float64(state.status.BackupSizeBytes))
	}
	
	bc.log.Info("Backup completed successfully", "backup", backupName, "duration", time.Since(state.startTime))
	
	// Schedule cleanup if TTL is set
	if state.spec.TTL != nil {
		bc.scheduleBackupCleanup(backupName, state.spec.TTL.Duration)
	}
}

// collectResources collects resources to backup
func (bc *BackupController) collectResources(ctx context.Context, spec *backup.BackupSpec) ([]backup.BackupItem, error) {
	var items []backup.BackupItem
	
	// Get namespaces to backup
	namespaces, err := bc.getNamespacesToBackup(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("getting namespaces: %w", err)
	}
	
	// Collect ObservabilityPlatform resources
	platforms := &observabilityv1.ObservabilityPlatformList{}
	if err := bc.client.List(ctx, platforms); err != nil {
		return nil, fmt.Errorf("listing platforms: %w", err)
	}
	
	for _, platform := range platforms.Items {
		// Check if platform should be included
		if !bc.shouldIncludeResource(platform.Namespace, "ObservabilityPlatform", namespaces, spec) {
			continue
		}
		
		// Check label selector
		if spec.LabelSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(spec.LabelSelector)
			if err != nil {
				return nil, fmt.Errorf("parsing label selector: %w", err)
			}
			if !selector.Matches(labels.Set(platform.Labels)) {
				continue
			}
		}
		
		// Add platform to backup
		item := backup.BackupItem{
			Name:      platform.Name,
			Namespace: platform.Namespace,
			GroupVersionKind: metav1.GroupVersionKind{
				Group:   observabilityv1.GroupVersion.Group,
				Version: observabilityv1.GroupVersion.Version,
				Kind:    "ObservabilityPlatform",
			},
		}
		
		// Convert to raw data
		data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&platform)
		if err != nil {
			return nil, fmt.Errorf("converting platform %s: %w", platform.Name, err)
		}
		item.Data.Raw, err = runtime.Encode(runtime.DefaultUnstructuredConverter, data)
		if err != nil {
			return nil, fmt.Errorf("encoding platform %s: %w", platform.Name, err)
		}
		
		items = append(items, item)
		
		// Collect related resources
		relatedItems, err := bc.collectRelatedResources(ctx, &platform, namespaces, spec)
		if err != nil {
			bc.log.Error(err, "Failed to collect related resources", "platform", platform.Name)
		} else {
			items = append(items, relatedItems...)
		}
	}
	
	// Collect cluster resources if requested
	if spec.IncludeClusterResources {
		clusterItems, err := bc.collectClusterResources(ctx, spec)
		if err != nil {
			bc.log.Error(err, "Failed to collect cluster resources")
		} else {
			items = append(items, clusterItems...)
		}
	}
	
	return items, nil
}

// collectRelatedResources collects resources related to a platform
func (bc *BackupController) collectRelatedResources(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	namespaces []string,
	spec *backup.BackupSpec,
) ([]backup.BackupItem, error) {
	var items []backup.BackupItem
	
	// Collect ConfigMaps
	configMaps := &corev1.ConfigMapList{}
	if err := bc.client.List(ctx, configMaps, client.InNamespace(platform.Namespace)); err != nil {
		return nil, fmt.Errorf("listing configmaps: %w", err)
	}
	
	for _, cm := range configMaps.Items {
		// Check if owned by platform
		if !bc.isOwnedByPlatform(&cm, platform) {
			continue
		}
		
		if !bc.shouldIncludeResource(cm.Namespace, "ConfigMap", namespaces, spec) {
			continue
		}
		
		item := backup.BackupItem{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			GroupVersionKind: metav1.GroupVersionKind{
				Version: "v1",
				Kind:    "ConfigMap",
			},
		}
		
		data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cm)
		if err != nil {
			return nil, fmt.Errorf("converting configmap %s: %w", cm.Name, err)
		}
		item.Data.Raw, err = runtime.Encode(runtime.DefaultUnstructuredConverter, data)
		if err != nil {
			return nil, fmt.Errorf("encoding configmap %s: %w", cm.Name, err)
		}
		
		items = append(items, item)
	}
	
	// Collect Secrets
	secrets := &corev1.SecretList{}
	if err := bc.client.List(ctx, secrets, client.InNamespace(platform.Namespace)); err != nil {
		return nil, fmt.Errorf("listing secrets: %w", err)
	}
	
	for _, secret := range secrets.Items {
		// Check if owned by platform
		if !bc.isOwnedByPlatform(&secret, platform) {
			continue
		}
		
		if !bc.shouldIncludeResource(secret.Namespace, "Secret", namespaces, spec) {
			continue
		}
		
		item := backup.BackupItem{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			GroupVersionKind: metav1.GroupVersionKind{
				Version: "v1",
				Kind:    "Secret",
			},
			Metadata: map[string]string{
				"encrypted": "true",
			},
		}
		
		data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&secret)
		if err != nil {
			return nil, fmt.Errorf("converting secret %s: %w", secret.Name, err)
		}
		item.Data.Raw, err = runtime.Encode(runtime.DefaultUnstructuredConverter, data)
		if err != nil {
			return nil, fmt.Errorf("encoding secret %s: %w", secret.Name, err)
		}
		
		items = append(items, item)
	}
	
	return items, nil
}

// getNamespacesToBackup gets the list of namespaces to backup
func (bc *BackupController) getNamespacesToBackup(ctx context.Context, spec *backup.BackupSpec) ([]string, error) {
	// If specific namespaces are included, use those
	if len(spec.IncludedNamespaces) > 0 {
		return spec.IncludedNamespaces, nil
	}
	
	// Otherwise, get all namespaces and apply exclusions
	namespaceList := &corev1.NamespaceList{}
	if err := bc.client.List(ctx, namespaceList); err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}
	
	var namespaces []string
	excludeMap := make(map[string]bool)
	for _, ns := range spec.ExcludedNamespaces {
		excludeMap[ns] = true
	}
	
	for _, ns := range namespaceList.Items {
		if !excludeMap[ns.Name] {
			namespaces = append(namespaces, ns.Name)
		}
	}
	
	return namespaces, nil
}

// shouldIncludeResource checks if a resource should be included in the backup
func (bc *BackupController) shouldIncludeResource(namespace, resourceType string, namespaces []string, spec *backup.BackupSpec) bool {
	// Check namespace
	namespaceIncluded := false
	for _, ns := range namespaces {
		if ns == namespace {
			namespaceIncluded = true
			break
		}
	}
	if !namespaceIncluded {
		return false
	}
	
	// Check if resource type is excluded
	for _, excluded := range spec.ExcludedResources {
		if excluded == resourceType {
			return false
		}
	}
	
	// Check if resource type is explicitly included
	if len(spec.IncludedResources) > 0 {
		included := false
		for _, inc := range spec.IncludedResources {
			if inc == resourceType {
				included = true
				break
			}
		}
		return included
	}
	
	return true
}

// isOwnedByPlatform checks if a resource is owned by a platform
func (bc *BackupController) isOwnedByPlatform(obj client.Object, platform *observabilityv1.ObservabilityPlatform) bool {
	for _, owner := range obj.GetOwnerReferences() {
		if owner.Kind == "ObservabilityPlatform" && owner.Name == platform.Name {
			return true
		}
	}
	return false
}

// handleBackupError handles backup errors
func (bc *BackupController) handleBackupError(state *backupState, err error) {
	bc.log.Error(err, "Backup failed")
	
	state.status.Phase = backup.BackupPhaseFailed
	state.status.CompletionTimestamp = &metav1.Time{Time: time.Now()}
	state.status.Errors = append(state.status.Errors, err.Error())
	
	// Update metrics
	if bc.metrics != nil {
		bc.metrics.backupsTotal.WithLabelValues(string(state.spec.Type), "failed").Inc()
		bc.metrics.backupErrors.WithLabelValues(string(state.spec.Type), "backup_failed").Inc()
	}
	
	// Check if we should retry
	state.retries++
	if state.retries < bc.config.MaxRetries {
		bc.log.V(1).Info("Scheduling backup retry", "retries", state.retries, "maxRetries", bc.config.MaxRetries)
		time.AfterFunc(bc.config.RetryInterval, func() {
			state.status.Phase = backup.BackupPhasePending
			state.status.Errors = nil
			go bc.runBackup(context.Background(), bc.generateBackupName(state.spec), state)
		})
	}
}

// generateBackupName generates a unique backup name
func (bc *BackupController) generateBackupName(spec *backup.BackupSpec) string {
	return fmt.Sprintf("backup-%s-%d", spec.Type, time.Now().Unix())
}

// getBackupLocation returns the backup storage location
func (bc *BackupController) getBackupLocation(backupName string, spec *backup.BackupSpec) string {
	return fmt.Sprintf("%s://%s/%s/%s", spec.StorageLocation.Provider, spec.StorageLocation.Bucket, spec.StorageLocation.Prefix, backupName)
}

// Other helper methods would be implemented here...
// runHooks, createBackupData, compressData, encryptData, takeVolumeSnapshots, etc.

// GetBackup gets backup status
func (bc *BackupController) GetBackup(ctx context.Context, name string) (*backup.BackupStatus, error) {
	bc.backupsMu.RLock()
	defer bc.backupsMu.RUnlock()
	
	state, exists := bc.backups[name]
	if !exists {
		return nil, fmt.Errorf("backup %s not found", name)
	}
	
	return state.status, nil
}

// DeleteBackup deletes a backup
func (bc *BackupController) DeleteBackup(ctx context.Context, name string) error {
	bc.backupsMu.Lock()
	state, exists := bc.backups[name]
	if !exists {
		bc.backupsMu.Unlock()
		return fmt.Errorf("backup %s not found", name)
	}
	
	// Mark for deletion
	state.status.Phase = backup.BackupPhaseDeleting
	bc.backupsMu.Unlock()
	
	// Delete from storage
	provider, ok := bc.providers[state.spec.StorageLocation.Provider]
	if !ok {
		return fmt.Errorf("provider %s not available", state.spec.StorageLocation.Provider)
	}
	
	if err := provider.Delete(ctx, state.spec); err != nil {
		return fmt.Errorf("deleting backup from storage: %w", err)
	}
	
	// Remove from tracking
	bc.backupsMu.Lock()
	delete(bc.backups, name)
	bc.backupsMu.Unlock()
	
	// Update metrics
	if bc.metrics != nil {
		bc.metrics.backupSize.DeleteLabelValues(name)
	}
	
	return nil
}

// ListBackups lists all backups
func (bc *BackupController) ListBackups(ctx context.Context) ([]backup.BackupStatus, error) {
	bc.backupsMu.RLock()
	defer bc.backupsMu.RUnlock()
	
	var statuses []backup.BackupStatus
	for _, state := range bc.backups {
		statuses = append(statuses, *state.status)
	}
	
	return statuses, nil
}

// scheduleBackupCleanup schedules backup cleanup after TTL
func (bc *BackupController) scheduleBackupCleanup(backupName string, ttl time.Duration) {
	time.AfterFunc(ttl, func() {
		ctx := context.Background()
		if err := bc.DeleteBackup(ctx, backupName); err != nil {
			bc.log.Error(err, "Failed to cleanup backup", "backup", backupName)
		} else {
			bc.log.V(1).Info("Backup cleaned up after TTL", "backup", backupName, "ttl", ttl)
		}
	})
}
