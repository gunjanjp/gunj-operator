// Package gitops provides GitOps integration for the Gunj Operator
package gitops

import (
	"context"
	"time"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GitOpsProvider represents supported GitOps providers
type GitOpsProvider string

const (
	// ProviderArgoCD represents ArgoCD GitOps provider
	ProviderArgoCD GitOpsProvider = "argocd"
	// ProviderFlux represents Flux GitOps provider
	ProviderFlux GitOpsProvider = "flux"
)

// GitOpsConfig represents GitOps configuration for a platform
type GitOpsConfig struct {
	// Provider specifies the GitOps provider (argocd or flux)
	Provider GitOpsProvider `json:"provider"`
	
	// Repository contains Git repository configuration
	Repository GitRepository `json:"repository"`
	
	// SyncPolicy defines how to sync resources
	SyncPolicy SyncPolicy `json:"syncPolicy"`
	
	// Promotion defines multi-environment promotion settings
	Promotion *PromotionConfig `json:"promotion,omitempty"`
	
	// Rollback defines rollback configuration
	Rollback *RollbackConfig `json:"rollback,omitempty"`
	
	// DriftDetection enables drift detection and remediation
	DriftDetection *DriftDetectionConfig `json:"driftDetection,omitempty"`
}

// GitRepository represents Git repository configuration
type GitRepository struct {
	// URL is the Git repository URL
	URL string `json:"url"`
	
	// Branch is the Git branch to track
	Branch string `json:"branch"`
	
	// Path is the path within the repository
	Path string `json:"path"`
	
	// SecretRef references a secret containing Git credentials
	SecretRef *SecretReference `json:"secretRef,omitempty"`
	
	// Interval is the sync interval
	Interval metav1.Duration `json:"interval,omitempty"`
}

// SecretReference references a Kubernetes secret
type SecretReference struct {
	// Name is the secret name
	Name string `json:"name"`
	
	// Namespace is the secret namespace
	Namespace string `json:"namespace,omitempty"`
}

// SyncPolicy defines synchronization policy
type SyncPolicy struct {
	// Automated enables automated sync
	Automated bool `json:"automated"`
	
	// Prune enables resource pruning
	Prune bool `json:"prune"`
	
	// SelfHeal enables automatic remediation
	SelfHeal bool `json:"selfHeal"`
	
	// Retry defines retry configuration
	Retry *RetryPolicy `json:"retry,omitempty"`
	
	// SyncOptions provides additional sync options
	SyncOptions []string `json:"syncOptions,omitempty"`
}

// RetryPolicy defines retry configuration
type RetryPolicy struct {
	// Limit is the maximum number of retries
	Limit int `json:"limit"`
	
	// Backoff defines backoff strategy
	Backoff *BackoffPolicy `json:"backoff,omitempty"`
}

// BackoffPolicy defines backoff configuration
type BackoffPolicy struct {
	// Duration is the base duration
	Duration metav1.Duration `json:"duration"`
	
	// Factor is the multiplication factor
	Factor int `json:"factor"`
	
	// MaxDuration is the maximum duration
	MaxDuration metav1.Duration `json:"maxDuration"`
}

// PromotionConfig defines multi-environment promotion settings
type PromotionConfig struct {
	// Environments defines the promotion pipeline
	Environments []Environment `json:"environments"`
	
	// Strategy defines promotion strategy
	Strategy PromotionStrategy `json:"strategy"`
	
	// ApprovalRequired indicates if manual approval is needed
	ApprovalRequired bool `json:"approvalRequired"`
}

// Environment represents an environment in the promotion pipeline
type Environment struct {
	// Name is the environment name
	Name string `json:"name"`
	
	// Namespace is the target namespace
	Namespace string `json:"namespace"`
	
	// Branch is the Git branch for this environment
	Branch string `json:"branch"`
	
	// AutoPromote enables automatic promotion to next environment
	AutoPromote bool `json:"autoPromote"`
	
	// PromotionPolicy defines promotion requirements
	PromotionPolicy *PromotionPolicy `json:"promotionPolicy,omitempty"`
}

// PromotionStrategy defines how promotions are performed
type PromotionStrategy string

const (
	// PromotionStrategyManual requires manual promotion
	PromotionStrategyManual PromotionStrategy = "manual"
	// PromotionStrategyAutomatic enables automatic promotion
	PromotionStrategyAutomatic PromotionStrategy = "automatic"
	// PromotionStrategyProgressive enables progressive rollout
	PromotionStrategyProgressive PromotionStrategy = "progressive"
)

// PromotionPolicy defines promotion requirements
type PromotionPolicy struct {
	// MinReplicaAvailability is the minimum replica availability percentage
	MinReplicaAvailability int `json:"minReplicaAvailability"`
	
	// HealthCheckDuration is how long to monitor health before promotion
	HealthCheckDuration metav1.Duration `json:"healthCheckDuration"`
	
	// MetricThresholds defines metric-based promotion gates
	MetricThresholds []MetricThreshold `json:"metricThresholds,omitempty"`
}

// MetricThreshold defines a metric-based threshold
type MetricThreshold struct {
	// Name is the metric name
	Name string `json:"name"`
	
	// Query is the PromQL query
	Query string `json:"query"`
	
	// Threshold is the threshold value
	Threshold float64 `json:"threshold"`
	
	// Operator is the comparison operator (>, <, >=, <=, ==)
	Operator string `json:"operator"`
}

// RollbackConfig defines rollback configuration
type RollbackConfig struct {
	// Enabled enables automatic rollback
	Enabled bool `json:"enabled"`
	
	// MaxHistory is the maximum number of rollback points to keep
	MaxHistory int `json:"maxHistory"`
	
	// Triggers defines rollback triggers
	Triggers []RollbackTrigger `json:"triggers"`
}

// RollbackTrigger defines when to trigger a rollback
type RollbackTrigger struct {
	// Type is the trigger type
	Type RollbackTriggerType `json:"type"`
	
	// Threshold is the threshold for the trigger
	Threshold string `json:"threshold"`
	
	// Duration is how long the condition must persist
	Duration metav1.Duration `json:"duration"`
}

// RollbackTriggerType defines types of rollback triggers
type RollbackTriggerType string

const (
	// RollbackTriggerTypeHealthCheck triggers on health check failure
	RollbackTriggerTypeHealthCheck RollbackTriggerType = "healthCheck"
	// RollbackTriggerTypeMetric triggers on metric threshold
	RollbackTriggerTypeMetric RollbackTriggerType = "metric"
	// RollbackTriggerTypeError triggers on error rate
	RollbackTriggerTypeError RollbackTriggerType = "error"
)

// DriftDetectionConfig defines drift detection settings
type DriftDetectionConfig struct {
	// Enabled enables drift detection
	Enabled bool `json:"enabled"`
	
	// Interval is the drift check interval
	Interval metav1.Duration `json:"interval"`
	
	// Action defines what to do when drift is detected
	Action DriftAction `json:"action"`
	
	// IgnoreFields lists fields to ignore during drift detection
	IgnoreFields []string `json:"ignoreFields,omitempty"`
}

// DriftAction defines actions to take on drift
type DriftAction string

const (
	// DriftActionNotify only notifies about drift
	DriftActionNotify DriftAction = "notify"
	// DriftActionRemediate automatically remediates drift
	DriftActionRemediate DriftAction = "remediate"
)

// GitOpsStatus represents the status of GitOps integration
type GitOpsStatus struct {
	// Provider is the active GitOps provider
	Provider GitOpsProvider `json:"provider"`
	
	// SyncStatus represents current sync status
	SyncStatus SyncStatus `json:"syncStatus"`
	
	// LastSyncTime is when the last sync occurred
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
	
	// LastSyncRevision is the last synced Git revision
	LastSyncRevision string `json:"lastSyncRevision,omitempty"`
	
	// DriftStatus represents drift detection status
	DriftStatus *DriftStatus `json:"driftStatus,omitempty"`
	
	// PromotionStatus represents promotion pipeline status
	PromotionStatus *PromotionStatus `json:"promotionStatus,omitempty"`
	
	// Conditions represents GitOps conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SyncStatus represents synchronization status
type SyncStatus string

const (
	// SyncStatusSynced indicates resources are synced
	SyncStatusSynced SyncStatus = "Synced"
	// SyncStatusOutOfSync indicates resources are out of sync
	SyncStatusOutOfSync SyncStatus = "OutOfSync"
	// SyncStatusSyncing indicates sync is in progress
	SyncStatusSyncing SyncStatus = "Syncing"
	// SyncStatusUnknown indicates unknown sync status
	SyncStatusUnknown SyncStatus = "Unknown"
)

// DriftStatus represents drift detection status
type DriftStatus struct {
	// Detected indicates if drift was detected
	Detected bool `json:"detected"`
	
	// LastCheckTime is when drift was last checked
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`
	
	// DriftedResources lists resources with drift
	DriftedResources []DriftedResource `json:"driftedResources,omitempty"`
}

// DriftedResource represents a resource with drift
type DriftedResource struct {
	// Name is the resource name
	Name string `json:"name"`
	
	// Kind is the resource kind
	Kind string `json:"kind"`
	
	// Namespace is the resource namespace
	Namespace string `json:"namespace,omitempty"`
	
	// Fields lists drifted fields
	Fields []string `json:"fields"`
}

// PromotionStatus represents promotion pipeline status
type PromotionStatus struct {
	// CurrentEnvironment is the current environment
	CurrentEnvironment string `json:"currentEnvironment"`
	
	// PromotionHistory lists recent promotions
	PromotionHistory []PromotionEvent `json:"promotionHistory,omitempty"`
}

// PromotionEvent represents a promotion event
type PromotionEvent struct {
	// From is the source environment
	From string `json:"from"`
	
	// To is the target environment
	To string `json:"to"`
	
	// Time is when the promotion occurred
	Time metav1.Time `json:"time"`
	
	// Revision is the Git revision promoted
	Revision string `json:"revision"`
	
	// Status is the promotion status
	Status string `json:"status"`
}

// GitOpsController defines the interface for GitOps controllers
type GitOpsController interface {
	// Reconcile reconciles GitOps state
	Reconcile(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error
	
	// Sync synchronizes resources with Git
	Sync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error
	
	// GetStatus returns current GitOps status
	GetStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*GitOpsStatus, error)
	
	// Rollback performs a rollback
	Rollback(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, revision string) error
	
	// Promote promotes to the next environment
	Promote(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv string) error
}

// GitSynchronizer defines the interface for Git synchronization
type GitSynchronizer interface {
	// Clone clones a Git repository
	Clone(ctx context.Context, repo GitRepository) (string, error)
	
	// Pull pulls latest changes
	Pull(ctx context.Context, repoPath string) error
	
	// GetRevision gets current revision
	GetRevision(ctx context.Context, repoPath string) (string, error)
	
	// GetFiles gets files from repository
	GetFiles(ctx context.Context, repoPath string, pattern string) ([]string, error)
	
	// Cleanup cleans up cloned repository
	Cleanup(ctx context.Context, repoPath string) error
}

// DriftDetector defines the interface for drift detection
type DriftDetector interface {
	// DetectDrift detects configuration drift
	DetectDrift(ctx context.Context, expected, actual runtime.Object) (*DriftResult, error)
	
	// Remediate remediates detected drift
	Remediate(ctx context.Context, drift *DriftResult) error
}

// DriftResult represents drift detection result
type DriftResult struct {
	// HasDrift indicates if drift was detected
	HasDrift bool
	
	// DriftedFields lists fields with drift
	DriftedFields map[string]DriftDetail
}

// DriftDetail provides details about a drifted field
type DriftDetail struct {
	// Path is the field path
	Path string
	
	// Expected is the expected value
	Expected interface{}
	
	// Actual is the actual value
	Actual interface{}
}

// PromotionManager defines the interface for managing promotions
type PromotionManager interface {
	// CanPromote checks if promotion is allowed
	CanPromote(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv string) (bool, error)
	
	// Promote performs the promotion
	Promote(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv string) error
	
	// GetPromotionHistory gets promotion history
	GetPromotionHistory(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) ([]PromotionEvent, error)
}

// RollbackManager defines the interface for managing rollbacks
type RollbackManager interface {
	// CreateSnapshot creates a rollback snapshot
	CreateSnapshot(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error
	
	// ListSnapshots lists available snapshots
	ListSnapshots(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) ([]RollbackSnapshot, error)
	
	// Rollback performs a rollback to a snapshot
	Rollback(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, snapshotID string) error
	
	// ShouldRollback checks if rollback should be triggered
	ShouldRollback(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (bool, string, error)
}

// RollbackSnapshot represents a rollback point
type RollbackSnapshot struct {
	// ID is the snapshot ID
	ID string `json:"id"`
	
	// Revision is the Git revision
	Revision string `json:"revision"`
	
	// Timestamp is when the snapshot was created
	Timestamp metav1.Time `json:"timestamp"`
	
	// Platform is the platform configuration
	Platform *observabilityv1.ObservabilityPlatform `json:"platform"`
	
	// Status is the platform status at snapshot time
	Status observabilityv1.ObservabilityPlatformStatus `json:"status"`
}

// WebhookHandler defines the interface for handling Git webhooks
type WebhookHandler interface {
	// HandlePush handles push events
	HandlePush(ctx context.Context, event PushEvent) error
	
	// HandlePullRequest handles pull request events
	HandlePullRequest(ctx context.Context, event PullRequestEvent) error
	
	// HandleTag handles tag events
	HandleTag(ctx context.Context, event TagEvent) error
}

// PushEvent represents a Git push event
type PushEvent struct {
	// Repository is the repository URL
	Repository string `json:"repository"`
	
	// Branch is the branch name
	Branch string `json:"branch"`
	
	// Revision is the new revision
	Revision string `json:"revision"`
	
	// Author is the commit author
	Author string `json:"author"`
	
	// Message is the commit message
	Message string `json:"message"`
	
	// Timestamp is the push timestamp
	Timestamp time.Time `json:"timestamp"`
}

// PullRequestEvent represents a pull request event
type PullRequestEvent struct {
	// Repository is the repository URL
	Repository string `json:"repository"`
	
	// Number is the PR number
	Number int `json:"number"`
	
	// Action is the PR action (opened, closed, merged)
	Action string `json:"action"`
	
	// SourceBranch is the source branch
	SourceBranch string `json:"sourceBranch"`
	
	// TargetBranch is the target branch
	TargetBranch string `json:"targetBranch"`
}

// TagEvent represents a Git tag event
type TagEvent struct {
	// Repository is the repository URL
	Repository string `json:"repository"`
	
	// Tag is the tag name
	Tag string `json:"tag"`
	
	// Revision is the tagged revision
	Revision string `json:"revision"`
	
	// Message is the tag message
	Message string `json:"message"`
}
