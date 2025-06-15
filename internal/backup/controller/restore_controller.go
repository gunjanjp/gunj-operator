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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	// RestoreControllerName is the name of the restore controller
	RestoreControllerName = "restore-controller"
	
	// RestoreFinalizer is the finalizer for restores
	RestoreFinalizer = "restore.observability.io/finalizer"
	
	// RestoreAnnotationPrefix is the prefix for restore annotations
	RestoreAnnotationPrefix = "restore.observability.io/"
)

// RestoreController handles restore operations
type RestoreController struct {
	client         client.Client
	scheme         *runtime.Scheme
	log            logr.Logger
	recorder       record.EventRecorder
	storageManager *storage.Manager
	providers      map[backup.StorageProvider]backup.BackupProvider
	backupCtrl     *BackupController
	metrics        *restoreMetrics
	
	// In-memory restore tracking
	restoresMu sync.RWMutex
	restores   map[string]*restoreState
	
	// Configuration
	config RestoreControllerConfig
}

// RestoreControllerConfig contains controller configuration
type RestoreControllerConfig struct {
	// MaxConcurrentRestores is the maximum concurrent restores
	MaxConcurrentRestores int
	
	// RetryInterval is the retry interval for failed restores
	RetryInterval time.Duration
	
	// MaxRetries is the maximum retries for failed restores
	MaxRetries int
	
	// ValidationEnabled enables restore validation
	ValidationEnabled bool
	
	// DryRun enables dry-run mode
	DryRun bool
	
	// MetricsEnabled enables metrics collection
	MetricsEnabled bool
}

// restoreState tracks the state of a restore
type restoreState struct {
	spec      *backup.RestoreSpec
	status    *backup.RestoreStatus
	retries   int
	startTime time.Time
	backup    *backup.BackupStatus
}

// restoreMetrics contains Prometheus metrics
type restoreMetrics struct {
	restoresTotal    *prometheus.CounterVec
	restoreDuration  *prometheus.HistogramVec
	restoreItems     *prometheus.GaugeVec
	restoreErrors    *prometheus.CounterVec
	activeRestores   prometheus.Gauge
}

// NewRestoreController creates a new restore controller
func NewRestoreController(
	client client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	recorder record.EventRecorder,
	backupCtrl *BackupController,
	config RestoreControllerConfig,
) (*RestoreController, error) {
	rc := &RestoreController{
		client:     client,
		scheme:     scheme,
		log:        log.WithName(RestoreControllerName),
		recorder:   recorder,
		backupCtrl: backupCtrl,
		config:     config,
		restores:   make(map[string]*restoreState),
		providers:  backupCtrl.providers, // Share providers with backup controller
	}
	
	// Initialize metrics
	if config.MetricsEnabled {
		rc.metrics = rc.initMetrics()
	}
	
	// Initialize storage manager
	storageManager, err := storage.NewManager(log)
	if err != nil {
		return nil, fmt.Errorf("creating storage manager: %w", err)
	}
	rc.storageManager = storageManager
	
	return rc, nil
}

// initMetrics initializes Prometheus metrics
func (rc *RestoreController) initMetrics() *restoreMetrics {
	metrics := &restoreMetrics{
		restoresTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_restore_total",
				Help: "Total number of restores",
			},
			[]string{"status"},
		),
		restoreDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "gunj_restore_duration_seconds",
				Help: "Restore duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
			},
			[]string{"backup"},
		),
		restoreItems: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gunj_restore_items",
				Help: "Number of items restored",
			},
			[]string{"name"},
		),
		restoreErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_restore_errors_total",
				Help: "Total number of restore errors",
			},
			[]string{"reason"},
		),
		activeRestores: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gunj_restore_active",
				Help: "Number of active restores",
			},
		),
	}
	
	// Register metrics
	prometheus.MustRegister(
		metrics.restoresTotal,
		metrics.restoreDuration,
		metrics.restoreItems,
		metrics.restoreErrors,
		metrics.activeRestores,
	)
	
	return metrics
}

// CreateRestore creates a new restore
func (rc *RestoreController) CreateRestore(ctx context.Context, spec *backup.RestoreSpec) (*backup.RestoreStatus, error) {
	restoreName := rc.generateRestoreName(spec)
	
	rc.log.V(1).Info("Creating restore", "name", restoreName, "backup", spec.BackupName)
	
	// Check if restore already exists
	rc.restoresMu.RLock()
	if _, exists := rc.restores[restoreName]; exists {
		rc.restoresMu.RUnlock()
		return nil, fmt.Errorf("restore %s already exists", restoreName)
	}
	rc.restoresMu.RUnlock()
	
	// Get backup to restore from
	backupStatus, err := rc.backupCtrl.GetBackup(ctx, spec.BackupName)
	if err != nil {
		return nil, fmt.Errorf("getting backup %s: %w", spec.BackupName, err)
	}
	
	// Validate backup is restorable
	if backupStatus.Phase != backup.BackupPhaseCompleted {
		return nil, fmt.Errorf("backup %s is not completed (phase: %s)", spec.BackupName, backupStatus.Phase)
	}
	
	// Initialize restore state
	state := &restoreState{
		spec:   spec,
		backup: backupStatus,
		status: &backup.RestoreStatus{
			Phase:          backup.RestorePhasePending,
			StartTimestamp: &metav1.Time{Time: time.Now()},
		},
		startTime: time.Now(),
	}
	
	// Store restore state
	rc.restoresMu.Lock()
	rc.restores[restoreName] = state
	rc.restoresMu.Unlock()
	
	// Update metrics
	if rc.metrics != nil {
		rc.metrics.activeRestores.Inc()
	}
	
	// Start restore asynchronously
	go rc.runRestore(ctx, restoreName, state)
	
	return state.status, nil
}

// runRestore runs the restore process
func (rc *RestoreController) runRestore(ctx context.Context, restoreName string, state *restoreState) {
	defer func() {
		// Update metrics
		if rc.metrics != nil {
			rc.metrics.activeRestores.Dec()
			duration := time.Since(state.startTime).Seconds()
			rc.metrics.restoreDuration.WithLabelValues(state.spec.BackupName).Observe(duration)
		}
	}()
	
	// Update status to in progress
	state.status.Phase = backup.RestorePhaseInProgress
	
	// Run pre-restore hooks
	if state.spec.Hooks != nil && len(state.spec.Hooks.PreRestore) > 0 {
		rc.log.V(1).Info("Running pre-restore hooks", "restore", restoreName)
		if err := rc.runHooks(ctx, state.spec.Hooks.PreRestore); err != nil {
			rc.handleRestoreError(state, fmt.Errorf("pre-restore hooks failed: %w", err))
			return
		}
	}
	
	// Download backup data
	rc.log.V(1).Info("Downloading backup data", "restore", restoreName, "backup", state.spec.BackupName)
	backupData, err := rc.downloadBackup(ctx, state)
	if err != nil {
		rc.handleRestoreError(state, fmt.Errorf("downloading backup: %w", err))
		return
	}
	
	// Decrypt if encrypted
	if rc.isEncrypted(backupData) {
		rc.log.V(1).Info("Decrypting backup", "restore", restoreName)
		decryptedData, err := rc.decryptData(ctx, backupData)
		if err != nil {
			rc.handleRestoreError(state, fmt.Errorf("decrypting backup: %w", err))
			return
		}
		backupData = decryptedData
	}
	
	// Decompress if compressed
	if rc.isCompressed(backupData) {
		rc.log.V(1).Info("Decompressing backup", "restore", restoreName)
		decompressedData, err := rc.decompressData(backupData)
		if err != nil {
			rc.handleRestoreError(state, fmt.Errorf("decompressing backup: %w", err))
			return
		}
		backupData = decompressedData
	}
	
	// Parse backup items
	rc.log.V(1).Info("Parsing backup items", "restore", restoreName)
	items, err := rc.parseBackupData(backupData)
	if err != nil {
		rc.handleRestoreError(state, fmt.Errorf("parsing backup data: %w", err))
		return
	}
	
	// Validate if required
	if rc.config.ValidationEnabled && state.spec.ValidationPolicy != nil {
		rc.log.V(1).Info("Validating backup data", "restore", restoreName)
		validationStatus := rc.validateBackupData(items, state.spec.ValidationPolicy)
		state.status.ValidationStatus = validationStatus
		
		if !validationStatus.Valid {
			rc.handleRestoreError(state, fmt.Errorf("validation failed: %v", validationStatus.Errors))
			return
		}
	}
	
	// Apply namespace mapping if provided
	if len(state.spec.NamespaceMapping) > 0 {
		rc.log.V(1).Info("Applying namespace mapping", "restore", restoreName)
		items = rc.applyNamespaceMapping(items, state.spec.NamespaceMapping)
	}
	
	// Filter items based on restore spec
	rc.log.V(1).Info("Filtering items", "restore", restoreName)
	filteredItems := rc.filterItems(items, state.spec)
	
	// Restore items
	rc.log.V(1).Info("Restoring items", "restore", restoreName, "count", len(filteredItems))
	if rc.config.DryRun {
		rc.log.Info("Dry run mode - skipping actual restore", "restore", restoreName)
		state.status.RestoredItemsCount = len(filteredItems)
	} else {
		restoredCount, errors := rc.restoreItems(ctx, filteredItems, state.spec)
		state.status.RestoredItemsCount = restoredCount
		if len(errors) > 0 {
			state.status.Errors = errors
			state.status.Phase = backup.RestorePhaseFailed
			rc.handleRestoreError(state, fmt.Errorf("some items failed to restore"))
			return
		}
	}
	
	// Restore volume snapshots if requested
	if state.spec.RestorePVs {
		rc.log.V(1).Info("Restoring volume snapshots", "restore", restoreName)
		if err := rc.restoreVolumeSnapshots(ctx, state); err != nil {
			state.status.Warnings = append(state.status.Warnings, fmt.Sprintf("Volume restore failed: %v", err))
		}
	}
	
	// Run post-restore hooks
	if state.spec.Hooks != nil && len(state.spec.Hooks.PostRestore) > 0 {
		rc.log.V(1).Info("Running post-restore hooks", "restore", restoreName)
		if err := rc.runHooks(ctx, state.spec.Hooks.PostRestore); err != nil {
			state.status.Warnings = append(state.status.Warnings, fmt.Sprintf("Post-restore hooks failed: %v", err))
		}
	}
	
	// Mark restore as completed
	state.status.Phase = backup.RestorePhaseCompleted
	state.status.CompletionTimestamp = &metav1.Time{Time: time.Now()}
	
	// Update metrics
	if rc.metrics != nil {
		rc.metrics.restoresTotal.WithLabelValues("success").Inc()
		rc.metrics.restoreItems.WithLabelValues(restoreName).Set(float64(state.status.RestoredItemsCount))
	}
	
	rc.log.Info("Restore completed successfully", "restore", restoreName, "duration", time.Since(state.startTime))
}

// restoreItems restores the backup items
func (rc *RestoreController) restoreItems(ctx context.Context, items []backup.BackupItem, spec *backup.RestoreSpec) (int, []string) {
	var restoredCount int
	var errors []string
	
	// Group items by type for efficient restoration
	itemsByType := make(map[metav1.GroupVersionKind][]backup.BackupItem)
	for _, item := range items {
		itemsByType[item.GroupVersionKind] = append(itemsByType[item.GroupVersionKind], item)
	}
	
	// Restore in order: namespaces first, then other resources
	// This ensures namespaces exist before namespace-scoped resources
	
	// Restore namespaces first
	for gvk, typeItems := range itemsByType {
		if gvk.Kind != "Namespace" {
			continue
		}
		
		for _, item := range typeItems {
			if err := rc.restoreItem(ctx, item, spec); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to restore %s/%s: %v", item.Namespace, item.Name, err))
				rc.log.Error(err, "Failed to restore item", "name", item.Name, "kind", gvk.Kind)
			} else {
				restoredCount++
				state.status.Progress = (restoredCount * 100) / len(items)
			}
		}
		delete(itemsByType, gvk)
	}
	
	// Restore ObservabilityPlatforms next
	for gvk, typeItems := range itemsByType {
		if gvk.Kind != "ObservabilityPlatform" {
			continue
		}
		
		for _, item := range typeItems {
			if err := rc.restoreItem(ctx, item, spec); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to restore %s/%s: %v", item.Namespace, item.Name, err))
				rc.log.Error(err, "Failed to restore item", "name", item.Name, "namespace", item.Namespace, "kind", gvk.Kind)
			} else {
				restoredCount++
				state.status.Progress = (restoredCount * 100) / len(items)
			}
		}
		delete(itemsByType, gvk)
	}
	
	// Restore remaining resources
	for gvk, typeItems := range itemsByType {
		for _, item := range typeItems {
			if err := rc.restoreItem(ctx, item, spec); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to restore %s/%s: %v", item.Namespace, item.Name, err))
				rc.log.Error(err, "Failed to restore item", "name", item.Name, "namespace", item.Namespace, "kind", gvk.Kind)
			} else {
				restoredCount++
				state.status.Progress = (restoredCount * 100) / len(items)
			}
		}
	}
	
	return restoredCount, errors
}

// restoreItem restores a single backup item
func (rc *RestoreController) restoreItem(ctx context.Context, item backup.BackupItem, spec *backup.RestoreSpec) error {
	// Parse the item data
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(item.Data.Raw, obj); err != nil {
		return fmt.Errorf("unmarshaling item data: %w", err)
	}
	
	// Apply restore modifications
	rc.applyRestoreModifications(obj, spec)
	
	// Check if resource already exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())
	err := rc.client.Get(ctx, types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}, existing)
	
	if err == nil {
		// Resource exists, update it
		rc.log.V(2).Info("Updating existing resource", "name", obj.GetName(), "namespace", obj.GetNamespace(), "kind", obj.GetKind())
		
		// Preserve certain fields from existing resource
		rc.preserveExistingFields(existing, obj, spec)
		
		if err := rc.client.Update(ctx, obj); err != nil {
			return fmt.Errorf("updating resource: %w", err)
		}
	} else if errors.IsNotFound(err) {
		// Resource doesn't exist, create it
		rc.log.V(2).Info("Creating resource", "name", obj.GetName(), "namespace", obj.GetNamespace(), "kind", obj.GetKind())
		
		if err := rc.client.Create(ctx, obj); err != nil {
			return fmt.Errorf("creating resource: %w", err)
		}
	} else {
		return fmt.Errorf("checking resource existence: %w", err)
	}
	
	return nil
}

// applyRestoreModifications applies modifications to a resource during restore
func (rc *RestoreController) applyRestoreModifications(obj *unstructured.Unstructured, spec *backup.RestoreSpec) {
	// Remove server-managed fields
	unstructured.RemoveNestedField(obj.Object, "metadata", "uid")
	unstructured.RemoveNestedField(obj.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(obj.Object, "metadata", "generation")
	unstructured.RemoveNestedField(obj.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(obj.Object, "metadata", "deletionTimestamp")
	unstructured.RemoveNestedField(obj.Object, "metadata", "selfLink")
	unstructured.RemoveNestedField(obj.Object, "status")
	
	// Add restore annotation
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[RestoreAnnotationPrefix+"name"] = spec.BackupName
	annotations[RestoreAnnotationPrefix+"timestamp"] = time.Now().Format(time.RFC3339)
	obj.SetAnnotations(annotations)
	
	// Apply node port preservation if needed
	if spec.PreserveNodePorts && obj.GetKind() == "Service" {
		// Preserve the node ports from the backup
		// This would require parsing the service spec
	}
}

// preserveExistingFields preserves certain fields from existing resource
func (rc *RestoreController) preserveExistingFields(existing, restored *unstructured.Unstructured, spec *backup.RestoreSpec) {
	// Preserve resource version for update
	restored.SetResourceVersion(existing.GetResourceVersion())
	
	// Preserve certain annotations/labels if needed
	// This can be customized based on requirements
}

// filterItems filters backup items based on restore spec
func (rc *RestoreController) filterItems(items []backup.BackupItem, spec *backup.RestoreSpec) []backup.BackupItem {
	var filtered []backup.BackupItem
	
	// Create maps for quick lookup
	includeNamespaces := make(map[string]bool)
	for _, ns := range spec.IncludedNamespaces {
		includeNamespaces[ns] = true
	}
	
	excludeNamespaces := make(map[string]bool)
	for _, ns := range spec.ExcludedNamespaces {
		excludeNamespaces[ns] = true
	}
	
	includeResources := make(map[string]bool)
	for _, res := range spec.IncludedResources {
		includeResources[res] = true
	}
	
	excludeResources := make(map[string]bool)
	for _, res := range spec.ExcludedResources {
		excludeResources[res] = true
	}
	
	for _, item := range items {
		// Check namespace filters
		if len(includeNamespaces) > 0 && !includeNamespaces[item.Namespace] {
			continue
		}
		if excludeNamespaces[item.Namespace] {
			continue
		}
		
		// Check resource filters
		if len(includeResources) > 0 && !includeResources[item.GroupVersionKind.Kind] {
			continue
		}
		if excludeResources[item.GroupVersionKind.Kind] {
			continue
		}
		
		// Check label selector if provided
		if spec.LabelSelector != nil {
			// This would require parsing the item data to check labels
			// For now, we'll include it
		}
		
		filtered = append(filtered, item)
	}
	
	return filtered
}

// handleRestoreError handles restore errors
func (rc *RestoreController) handleRestoreError(state *restoreState, err error) {
	rc.log.Error(err, "Restore failed")
	
	state.status.Phase = backup.RestorePhaseFailed
	state.status.CompletionTimestamp = &metav1.Time{Time: time.Now()}
	state.status.Errors = append(state.status.Errors, err.Error())
	
	// Update metrics
	if rc.metrics != nil {
		rc.metrics.restoresTotal.WithLabelValues("failed").Inc()
		rc.metrics.restoreErrors.WithLabelValues("restore_failed").Inc()
	}
	
	// Check if we should retry
	state.retries++
	if state.retries < rc.config.MaxRetries {
		rc.log.V(1).Info("Scheduling restore retry", "retries", state.retries, "maxRetries", rc.config.MaxRetries)
		time.AfterFunc(rc.config.RetryInterval, func() {
			state.status.Phase = backup.RestorePhasePending
			state.status.Errors = nil
			go rc.runRestore(context.Background(), rc.generateRestoreName(state.spec), state)
		})
	}
}

// Other helper methods would be implemented here...
// runHooks, downloadBackup, parseBackupData, validateBackupData, etc.

// generateRestoreName generates a unique restore name
func (rc *RestoreController) generateRestoreName(spec *backup.RestoreSpec) string {
	return fmt.Sprintf("restore-%s-%d", spec.BackupName, time.Now().Unix())
}

// GetRestore gets restore status
func (rc *RestoreController) GetRestore(ctx context.Context, name string) (*backup.RestoreStatus, error) {
	rc.restoresMu.RLock()
	defer rc.restoresMu.RUnlock()
	
	state, exists := rc.restores[name]
	if !exists {
		return nil, fmt.Errorf("restore %s not found", name)
	}
	
	return state.status, nil
}

// ListRestores lists all restores
func (rc *RestoreController) ListRestores(ctx context.Context) ([]backup.RestoreStatus, error) {
	rc.restoresMu.RLock()
	defer rc.restoresMu.RUnlock()
	
	var statuses []backup.RestoreStatus
	for _, state := range rc.restores {
		statuses = append(statuses, *state.status)
	}
	
	return statuses, nil
}

// applyNamespaceMapping applies namespace mapping to items
func (rc *RestoreController) applyNamespaceMapping(items []backup.BackupItem, mapping map[string]string) []backup.BackupItem {
	var mapped []backup.BackupItem
	
	for _, item := range items {
		mappedItem := item
		
		// Apply namespace mapping if exists
		if newNamespace, ok := mapping[item.Namespace]; ok {
			mappedItem.Namespace = newNamespace
		}
		
		mapped = append(mapped, mappedItem)
	}
	
	return mapped
}
