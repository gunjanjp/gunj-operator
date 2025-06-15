package gitops

import (
	"context"

	"github.com/fluxcd/flux2/v2/api/v1beta2"
	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GitOpsManager manages GitOps integrations for ObservabilityPlatform
type GitOpsManager interface {
	// SetupGitOps configures GitOps for the platform
	SetupGitOps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	
	// SyncWithGit synchronizes the platform configuration with Git repository
	SyncWithGit(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	
	// PromoteEnvironment promotes configuration between environments
	PromoteEnvironment(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, from, to string) error
	
	// Rollback rolls back to a previous configuration
	Rollback(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision string) error
	
	// GetSyncStatus returns the current GitOps sync status
	GetSyncStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*GitOpsSyncStatus, error)
	
	// ValidateGitOpsConfig validates GitOps configuration
	ValidateGitOpsConfig(platform *observabilityv1beta1.ObservabilityPlatform) error
}

// GitOpsProvider represents supported GitOps providers
type GitOpsProvider string

const (
	// ArgoCD provider
	ArgoCD GitOpsProvider = "argocd"
	// Flux provider
	Flux GitOpsProvider = "flux"
)

// GitOpsSyncStatus represents the sync status with Git
type GitOpsSyncStatus struct {
	// Provider being used
	Provider GitOpsProvider `json:"provider"`
	
	// LastSyncTime is the last successful sync time
	LastSyncTime string `json:"lastSyncTime,omitempty"`
	
	// Revision is the current Git revision
	Revision string `json:"revision,omitempty"`
	
	// SyncState represents the sync state
	SyncState SyncState `json:"syncState"`
	
	// Message provides additional information
	Message string `json:"message,omitempty"`
	
	// Environments shows status per environment
	Environments map[string]EnvironmentStatus `json:"environments,omitempty"`
}

// SyncState represents the state of GitOps sync
type SyncState string

const (
	// SyncStateSynced indicates resources are in sync
	SyncStateSynced SyncState = "Synced"
	// SyncStateOutOfSync indicates resources are out of sync
	SyncStateOutOfSync SyncState = "OutOfSync"
	// SyncStateSyncing indicates sync is in progress
	SyncStateSyncing SyncState = "Syncing"
	// SyncStateUnknown indicates unknown state
	SyncStateUnknown SyncState = "Unknown"
	// SyncStateError indicates sync error
	SyncStateError SyncState = "Error"
)

// EnvironmentStatus represents the status of a specific environment
type EnvironmentStatus struct {
	// Name of the environment
	Name string `json:"name"`
	
	// Namespace where deployed
	Namespace string `json:"namespace"`
	
	// Revision deployed in this environment
	Revision string `json:"revision"`
	
	// PromotedFrom indicates the source environment
	PromotedFrom string `json:"promotedFrom,omitempty"`
	
	// PromotedAt indicates when the promotion happened
	PromotedAt string `json:"promotedAt,omitempty"`
	
	// Status of the environment
	Status string `json:"status"`
	
	// Health of components in this environment
	Health string `json:"health"`
}

// GitOpsConfig holds GitOps configuration
type GitOpsConfig struct {
	// Provider to use (argocd or flux)
	Provider GitOpsProvider `json:"provider"`
	
	// Repository configuration
	Repository GitRepository `json:"repository"`
	
	// Environments configuration
	Environments []Environment `json:"environments"`
	
	// AutoSync enables automatic synchronization
	AutoSync bool `json:"autoSync"`
	
	// AutoPromotion configuration
	AutoPromotion *AutoPromotionConfig `json:"autoPromotion,omitempty"`
	
	// Rollback configuration
	RollbackConfig *RollbackConfig `json:"rollbackConfig,omitempty"`
}

// GitRepository represents Git repository configuration
type GitRepository struct {
	// URL of the Git repository
	URL string `json:"url"`
	
	// Branch to track
	Branch string `json:"branch"`
	
	// Path within the repository
	Path string `json:"path"`
	
	// SecretRef for authentication
	SecretRef string `json:"secretRef,omitempty"`
	
	// Interval for polling
	Interval string `json:"interval,omitempty"`
}

// Environment represents an environment configuration
type Environment struct {
	// Name of the environment
	Name string `json:"name"`
	
	// Namespace to deploy to
	Namespace string `json:"namespace"`
	
	// Branch to track for this environment
	Branch string `json:"branch,omitempty"`
	
	// Path override for this environment
	Path string `json:"path,omitempty"`
	
	// PromotionPolicy defines promotion rules
	PromotionPolicy *PromotionPolicy `json:"promotionPolicy,omitempty"`
}

// PromotionPolicy defines rules for environment promotion
type PromotionPolicy struct {
	// AutoPromotion enables automatic promotion
	AutoPromotion bool `json:"autoPromotion"`
	
	// RequiredTests that must pass
	RequiredTests []string `json:"requiredTests,omitempty"`
	
	// ApprovalRequired indicates if manual approval is needed
	ApprovalRequired bool `json:"approvalRequired"`
	
	// PromoteAfter duration to wait before promotion
	PromoteAfter string `json:"promoteAfter,omitempty"`
}

// AutoPromotionConfig configures automatic promotion
type AutoPromotionConfig struct {
	// Enabled turns on auto-promotion
	Enabled bool `json:"enabled"`
	
	// Strategy for promotion (sequential, parallel)
	Strategy string `json:"strategy"`
	
	// MaxParallel environments to promote simultaneously
	MaxParallel int `json:"maxParallel,omitempty"`
}

// RollbackConfig configures rollback behavior
type RollbackConfig struct {
	// AutoRollback enables automatic rollback on failure
	AutoRollback bool `json:"autoRollback"`
	
	// FailureThreshold before triggering rollback
	FailureThreshold int `json:"failureThreshold"`
	
	// Window for detecting failures
	Window string `json:"window"`
	
	// MaxHistory to keep for rollback
	MaxHistory int `json:"maxHistory"`
}

// NewGitOpsManager creates a new GitOps manager
func NewGitOpsManager(client client.Client, scheme *runtime.Scheme, provider GitOpsProvider) (GitOpsManager, error) {
	switch provider {
	case ArgoCD:
		return NewArgoCDManager(client, scheme)
	case Flux:
		return NewFluxManager(client, scheme)
	default:
		return NewMultiProviderManager(client, scheme)
	}
}
