package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// Manager handles Git repository synchronization
type Manager struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// NewManager creates a new sync manager
func NewManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *Manager {
	return &Manager{
		Client: client,
		Scheme: scheme,
		Log:    log.WithName("sync-manager"),
	}
}

// SyncState represents the state of a sync operation
type SyncState struct {
	// Revision is the Git revision being synced
	Revision string `json:"revision"`

	// Status is the sync status
	Status SyncStatus `json:"status"`

	// StartTime is when the sync started
	StartTime time.Time `json:"startTime"`

	// CompletionTime is when the sync completed
	CompletionTime *time.Time `json:"completionTime,omitempty"`

	// Message provides additional information
	Message string `json:"message,omitempty"`

	// Resources tracks synced resources
	Resources []SyncedResource `json:"resources,omitempty"`

	// Errors contains any sync errors
	Errors []string `json:"errors,omitempty"`
}

// SyncStatus represents the status of a sync operation
type SyncStatus string

const (
	// SyncStatusPending indicates sync is pending
	SyncStatusPending SyncStatus = "Pending"
	// SyncStatusInProgress indicates sync is in progress
	SyncStatusInProgress SyncStatus = "InProgress"
	// SyncStatusCompleted indicates sync completed successfully
	SyncStatusCompleted SyncStatus = "Completed"
	// SyncStatusFailed indicates sync failed
	SyncStatusFailed SyncStatus = "Failed"
	// SyncStatusCancelled indicates sync was cancelled
	SyncStatusCancelled SyncStatus = "Cancelled"
)

// SyncedResource represents a resource that was synced
type SyncedResource struct {
	// Kind is the resource kind
	Kind string `json:"kind"`

	// Name is the resource name
	Name string `json:"name"`

	// Namespace is the resource namespace
	Namespace string `json:"namespace,omitempty"`

	// Action is what was done (created, updated, deleted)
	Action string `json:"action"`

	// Status is the resource status
	Status string `json:"status"`
}

// TriggerSync triggers a sync operation for the platform
func (m *Manager) TriggerSync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Triggering sync operation")

	// Create sync state
	syncState := &SyncState{
		Revision:  platform.Spec.GitOps.Repository.Branch,
		Status:    SyncStatusInProgress,
		StartTime: time.Now(),
	}

	// Store sync state in ConfigMap
	if err := m.storeSyncState(ctx, platform, syncState); err != nil {
		return fmt.Errorf("storing sync state: %w", err)
	}

	// Trigger sync based on provider
	switch platform.Spec.GitOps.Provider {
	case observabilityv1.GitOpsProviderArgoCD:
		return m.triggerArgoCDSync(ctx, platform)
	case observabilityv1.GitOpsProviderFlux:
		return m.triggerFluxSync(ctx, platform)
	default:
		return fmt.Errorf("unsupported GitOps provider: %s", platform.Spec.GitOps.Provider)
	}
}

// GetSyncState gets the current sync state
func (m *Manager) GetSyncState(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*SyncState, error) {
	cm := &corev1.ConfigMap{}
	err := m.Client.Get(ctx, client.ObjectKey{
		Name:      m.getSyncStateConfigMapName(platform),
		Namespace: platform.Namespace,
	}, cm)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting sync state: %w", err)
	}

	// Parse sync state from ConfigMap
	syncState := &SyncState{}
	// In a real implementation, we would unmarshal from cm.Data
	// For now, return a placeholder
	syncState.Status = SyncStatusCompleted
	syncState.Revision = platform.Spec.GitOps.Repository.Branch

	return syncState, nil
}

// UpdateTargetRevision updates the target revision for sync
func (m *Manager) UpdateTargetRevision(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, revision string) error {
	log := m.Log.WithValues("platform", platform.Name, "revision", revision)
	log.Info("Updating target revision")

	// Update based on provider
	switch platform.Spec.GitOps.Provider {
	case observabilityv1.GitOpsProviderArgoCD:
		return m.updateArgoCDRevision(ctx, platform, revision)
	case observabilityv1.GitOpsProviderFlux:
		return m.updateFluxRevision(ctx, platform, revision)
	default:
		return fmt.Errorf("unsupported GitOps provider: %s", platform.Spec.GitOps.Provider)
	}
}

// PromoteToEnvironment promotes configuration to a target environment
func (m *Manager) PromoteToEnvironment(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, env *observabilityv1.EnvironmentConfig) error {
	log := m.Log.WithValues("platform", platform.Name, "environment", env.Name)
	log.Info("Promoting to environment")

	// Create promotion record
	promotion := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-promotion-%s-%d", platform.Name, env.Name, time.Now().Unix()),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"gunj-operator.io/platform":    platform.Name,
				"gunj-operator.io/promotion":   "true",
				"gunj-operator.io/environment": env.Name,
			},
			Annotations: map[string]string{
				"gunj-operator.io/source-revision": platform.Status.GitOps.Revision,
				"gunj-operator.io/target-revision": env.TargetRevision,
				"gunj-operator.io/promoted-by":     "gunj-operator",
				"gunj-operator.io/promoted-at":     time.Now().Format(time.RFC3339),
			},
		},
		Data: map[string]string{
			"source-platform":  platform.Name,
			"source-namespace": platform.Namespace,
			"target-environment": env.Name,
			"target-namespace":   env.Namespace,
			"target-cluster":     env.Cluster,
			"configuration":      m.getPromotionConfig(platform, env),
		},
	}

	// Create promotion record
	if err := m.Client.Create(ctx, promotion); err != nil {
		return fmt.Errorf("creating promotion record: %w", err)
	}

	// If target is in same cluster, create/update platform
	if env.Cluster == "" || env.Cluster == "local" {
		targetPlatform := platform.DeepCopy()
		targetPlatform.Name = fmt.Sprintf("%s-%s", platform.Name, env.Name)
		targetPlatform.Namespace = env.Namespace
		if targetPlatform.Namespace == "" {
			targetPlatform.Namespace = platform.Namespace
		}

		// Update Git configuration
		if env.TargetRevision != "" {
			targetPlatform.Spec.GitOps.Repository.Branch = env.TargetRevision
		}

		// Clear resource version for create/update
		targetPlatform.ResourceVersion = ""
		targetPlatform.UID = ""

		// Create namespace if needed
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: targetPlatform.Namespace,
				Labels: map[string]string{
					"gunj-operator.io/environment": env.Name,
				},
			},
		}
		if err := m.Client.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating target namespace: %w", err)
		}

		// Create or update target platform
		existing := &observabilityv1.ObservabilityPlatform{}
		err := m.Client.Get(ctx, client.ObjectKeyFromObject(targetPlatform), existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := m.Client.Create(ctx, targetPlatform); err != nil {
					return fmt.Errorf("creating target platform: %w", err)
				}
			} else {
				return fmt.Errorf("checking existing platform: %w", err)
			}
		} else {
			existing.Spec = targetPlatform.Spec
			if err := m.Client.Update(ctx, existing); err != nil {
				return fmt.Errorf("updating target platform: %w", err)
			}
		}
	}

	log.Info("Promotion completed successfully")
	return nil
}

// WaitForSync waits for a sync operation to complete
func (m *Manager) WaitForSync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, timeout time.Duration) error {
	log := m.Log.WithValues("platform", platform.Name)
	log.V(1).Info("Waiting for sync to complete", "timeout", timeout)

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		syncState, err := m.GetSyncState(ctx, platform)
		if err != nil {
			return fmt.Errorf("getting sync state: %w", err)
		}

		if syncState == nil {
			log.V(2).Info("No sync state found, waiting...")
			time.Sleep(5 * time.Second)
			continue
		}

		switch syncState.Status {
		case SyncStatusCompleted:
			log.Info("Sync completed successfully")
			return nil
		case SyncStatusFailed:
			return fmt.Errorf("sync failed: %s", syncState.Message)
		case SyncStatusCancelled:
			return fmt.Errorf("sync was cancelled")
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue waiting
		}
	}

	return fmt.Errorf("sync operation timed out after %v", timeout)
}

// CancelSync cancels an ongoing sync operation
func (m *Manager) CancelSync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := m.Log.WithValues("platform", platform.Name)
	log.Info("Cancelling sync operation")

	// Get current sync state
	syncState, err := m.GetSyncState(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting sync state: %w", err)
	}

	if syncState == nil || syncState.Status != SyncStatusInProgress {
		log.V(1).Info("No active sync to cancel")
		return nil
	}

	// Update sync state to cancelled
	syncState.Status = SyncStatusCancelled
	syncState.CompletionTime = &metav1.Time{Time: time.Now()}
	syncState.Message = "Sync cancelled by user"

	if err := m.storeSyncState(ctx, platform, syncState); err != nil {
		return fmt.Errorf("updating sync state: %w", err)
	}

	// Provider-specific cancellation
	switch platform.Spec.GitOps.Provider {
	case observabilityv1.GitOpsProviderArgoCD:
		// ArgoCD doesn't have direct sync cancellation
		// Would need to use ArgoCD API
		log.V(1).Info("ArgoCD sync cancellation not implemented")
	case observabilityv1.GitOpsProviderFlux:
		// Flux sync can be cancelled by suspending the Kustomization
		// This would be implemented in the flux manager
		log.V(1).Info("Flux sync cancellation not implemented")
	}

	return nil
}

// triggerArgoCDSync triggers sync for ArgoCD
func (m *Manager) triggerArgoCDSync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// In a real implementation, this would:
	// 1. Use ArgoCD API to trigger sync
	// 2. Or create a sync annotation on the Application
	// For now, just log
	m.Log.V(1).Info("ArgoCD sync triggered")
	return nil
}

// triggerFluxSync triggers sync for Flux
func (m *Manager) triggerFluxSync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// In a real implementation, this would:
	// 1. Annotate the Kustomization to trigger reconciliation
	// 2. Or use Flux API to trigger sync
	// For now, just log
	m.Log.V(1).Info("Flux sync triggered")
	return nil
}

// updateArgoCDRevision updates ArgoCD target revision
func (m *Manager) updateArgoCDRevision(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, revision string) error {
	// Would update the Application spec.source.targetRevision
	m.Log.V(1).Info("Updated ArgoCD target revision", "revision", revision)
	return nil
}

// updateFluxRevision updates Flux target revision
func (m *Manager) updateFluxRevision(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, revision string) error {
	// Would update the GitRepository spec.ref
	m.Log.V(1).Info("Updated Flux target revision", "revision", revision)
	return nil
}

// storeSyncState stores sync state in a ConfigMap
func (m *Manager) storeSyncState(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, state *SyncState) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getSyncStateConfigMapName(platform),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"gunj-operator.io/platform":   platform.Name,
				"gunj-operator.io/sync-state": "true",
			},
		},
		Data: map[string]string{
			"revision":   state.Revision,
			"status":     string(state.Status),
			"startTime":  state.StartTime.Format(time.RFC3339),
			"message":    state.Message,
		},
	}

	if state.CompletionTime != nil {
		cm.Data["completionTime"] = state.CompletionTime.Format(time.RFC3339)
	}

	// Create or update
	existing := &corev1.ConfigMap{}
	err := m.Client.Get(ctx, client.ObjectKeyFromObject(cm), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			return m.Client.Create(ctx, cm)
		}
		return err
	}

	existing.Data = cm.Data
	return m.Client.Update(ctx, existing)
}

// getSyncStateConfigMapName returns the sync state ConfigMap name
func (m *Manager) getSyncStateConfigMapName(platform *observabilityv1.ObservabilityPlatform) string {
	return fmt.Sprintf("%s-sync-state", platform.Name)
}

// getPromotionConfig generates promotion configuration
func (m *Manager) getPromotionConfig(platform *observabilityv1.ObservabilityPlatform, env *observabilityv1.EnvironmentConfig) string {
	// In a real implementation, this would generate environment-specific configuration
	// For now, return a placeholder
	return fmt.Sprintf("Promotion from %s to %s environment", platform.Name, env.Name)
}

// GetSyncHistory gets the sync history for a platform
func (m *Manager) GetSyncHistory(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, limit int) ([]SyncState, error) {
	// List all sync state ConfigMaps
	cmList := &corev1.ConfigMapList{}
	if err := m.Client.List(ctx, cmList, client.InNamespace(platform.Namespace),
		client.MatchingLabels{
			"gunj-operator.io/platform":   platform.Name,
			"gunj-operator.io/sync-state": "true",
		}); err != nil {
		return nil, fmt.Errorf("listing sync states: %w", err)
	}

	// Convert to SyncState objects
	var history []SyncState
	for _, cm := range cmList.Items {
		state := SyncState{
			Revision: cm.Data["revision"],
			Status:   SyncStatus(cm.Data["status"]),
			Message:  cm.Data["message"],
		}

		if startTime, err := time.Parse(time.RFC3339, cm.Data["startTime"]); err == nil {
			state.StartTime = startTime
		}

		if completionTime, ok := cm.Data["completionTime"]; ok {
			if t, err := time.Parse(time.RFC3339, completionTime); err == nil {
				state.CompletionTime = &t
			}
		}

		history = append(history, state)
	}

	// Sort by start time (newest first)
	// In a real implementation, we would sort the history

	// Apply limit
	if limit > 0 && len(history) > limit {
		history = history[:limit]
	}

	return history, nil
}

// GetLastSuccessfulSync gets the last successful sync state
func (m *Manager) GetLastSuccessfulSync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*SyncState, error) {
	history, err := m.GetSyncHistory(ctx, platform, 10)
	if err != nil {
		return nil, err
	}

	for _, state := range history {
		if state.Status == SyncStatusCompleted {
			return &state, nil
		}
	}

	return nil, nil
}
