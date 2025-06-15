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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitOpsIntegrationType defines the type of GitOps tool
type GitOpsIntegrationType string

const (
	// GitOpsArgoCD represents ArgoCD integration
	GitOpsArgoCD GitOpsIntegrationType = "ArgoCD"
	// GitOpsFlux represents Flux integration
	GitOpsFlux GitOpsIntegrationType = "Flux"
)

// GitOpsIntegrationSpec defines the desired state of GitOps integration
type GitOpsIntegrationSpec struct {
	// Type specifies the GitOps tool type (ArgoCD or Flux)
	// +kubebuilder:validation:Enum=ArgoCD;Flux
	Type GitOpsIntegrationType `json:"type"`

	// Repository contains Git repository configuration
	Repository GitRepositorySpec `json:"repository"`

	// SyncPolicy defines synchronization behavior
	SyncPolicy SyncPolicySpec `json:"syncPolicy,omitempty"`

	// Environments defines multi-environment configuration
	Environments []EnvironmentSpec `json:"environments,omitempty"`

	// Promotion defines environment promotion configuration
	Promotion *PromotionSpec `json:"promotion,omitempty"`

	// DriftDetection enables drift detection and remediation
	DriftDetection *DriftDetectionSpec `json:"driftDetection,omitempty"`

	// Rollback defines rollback configuration
	Rollback *RollbackSpec `json:"rollback,omitempty"`
}

// GitRepositorySpec defines Git repository configuration
type GitRepositorySpec struct {
	// URL is the Git repository URL
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Branch is the Git branch to track
	// +kubebuilder:default:="main"
	Branch string `json:"branch,omitempty"`

	// Path is the path within the repository
	// +kubebuilder:default:="/"
	Path string `json:"path,omitempty"`

	// CredentialsSecret references a secret containing Git credentials
	CredentialsSecret *SecretReference `json:"credentialsSecret,omitempty"`

	// WebhookSecret is used for validating webhook payloads
	WebhookSecret *SecretReference `json:"webhookSecret,omitempty"`
}

// SecretReference references a Kubernetes secret
type SecretReference struct {
	// Name is the secret name
	Name string `json:"name"`
	
	// Namespace is the secret namespace
	// +kubebuilder:default:="default"
	Namespace string `json:"namespace,omitempty"`
	
	// Key is the key within the secret
	// +kubebuilder:default:="value"
	Key string `json:"key,omitempty"`
}

// SyncPolicySpec defines synchronization policy
type SyncPolicySpec struct {
	// Automated enables automated sync
	Automated *AutomatedSyncPolicy `json:"automated,omitempty"`

	// SyncOptions provides sync options
	SyncOptions []string `json:"syncOptions,omitempty"`

	// Retry defines retry policy
	Retry *RetryPolicy `json:"retry,omitempty"`
}

// AutomatedSyncPolicy defines automated sync configuration
type AutomatedSyncPolicy struct {
	// Prune enables automated pruning
	Prune bool `json:"prune,omitempty"`

	// SelfHeal enables automated self-healing
	SelfHeal bool `json:"selfHeal,omitempty"`

	// AllowEmpty allows syncing empty directories
	AllowEmpty bool `json:"allowEmpty,omitempty"`
}

// RetryPolicy defines retry configuration
type RetryPolicy struct {
	// Limit is the maximum number of retry attempts
	Limit int `json:"limit,omitempty"`

	// Backoff defines backoff configuration
	Backoff *BackoffPolicy `json:"backoff,omitempty"`
}

// BackoffPolicy defines backoff configuration
type BackoffPolicy struct {
	// Duration is the initial backoff duration
	Duration string `json:"duration,omitempty"`

	// Factor is the backoff multiplication factor
	Factor int32 `json:"factor,omitempty"`

	// MaxDuration is the maximum backoff duration
	MaxDuration string `json:"maxDuration,omitempty"`
}

// EnvironmentSpec defines an environment configuration
type EnvironmentSpec struct {
	// Name is the environment name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Branch is the Git branch for this environment
	Branch string `json:"branch,omitempty"`

	// Path is the path within the repository for this environment
	Path string `json:"path,omitempty"`

	// TargetRevision is the target revision for this environment
	TargetRevision string `json:"targetRevision,omitempty"`

	// AutoSync enables automatic synchronization for this environment
	AutoSync bool `json:"autoSync,omitempty"`

	// PromotionGates defines gates before promotion to this environment
	PromotionGates []PromotionGate `json:"promotionGates,omitempty"`
}

// PromotionGate defines a promotion gate
type PromotionGate struct {
	// Type is the gate type
	// +kubebuilder:validation:Enum=Manual;Test;Metric;Time
	Type string `json:"type"`

	// Config contains gate-specific configuration
	Config map[string]string `json:"config,omitempty"`
}

// PromotionSpec defines promotion configuration
type PromotionSpec struct {
	// Strategy defines the promotion strategy
	// +kubebuilder:validation:Enum=Manual;Progressive;BlueGreen;Canary
	Strategy string `json:"strategy"`

	// AutoPromotion enables automatic promotion
	AutoPromotion bool `json:"autoPromotion,omitempty"`

	// PromotionPolicy defines promotion policies
	PromotionPolicy []PromotionPolicy `json:"promotionPolicy,omitempty"`
}

// PromotionPolicy defines a promotion policy
type PromotionPolicy struct {
	// From is the source environment
	From string `json:"from"`

	// To is the target environment
	To string `json:"to"`

	// RequiredApprovals is the number of required approvals
	RequiredApprovals int `json:"requiredApprovals,omitempty"`

	// AutoPromoteAfter defines auto-promotion duration
	AutoPromoteAfter string `json:"autoPromoteAfter,omitempty"`
}

// DriftDetectionSpec defines drift detection configuration
type DriftDetectionSpec struct {
	// Enabled enables drift detection
	Enabled bool `json:"enabled"`

	// Interval is the drift detection interval
	// +kubebuilder:default:="5m"
	Interval string `json:"interval,omitempty"`

	// AutoRemediate enables automatic drift remediation
	AutoRemediate bool `json:"autoRemediate,omitempty"`

	// NotificationPolicy defines notification configuration
	NotificationPolicy *NotificationPolicy `json:"notificationPolicy,omitempty"`
}

// NotificationPolicy defines notification configuration
type NotificationPolicy struct {
	// Channels defines notification channels
	Channels []NotificationChannel `json:"channels,omitempty"`

	// Severity defines minimum severity for notifications
	// +kubebuilder:validation:Enum=Info;Warning;Error;Critical
	Severity string `json:"severity,omitempty"`
}

// NotificationChannel defines a notification channel
type NotificationChannel struct {
	// Type is the channel type
	// +kubebuilder:validation:Enum=Slack;Email;Webhook;Teams
	Type string `json:"type"`

	// Config contains channel-specific configuration
	Config map[string]string `json:"config"`
}

// RollbackSpec defines rollback configuration
type RollbackSpec struct {
	// Enabled enables automatic rollback
	Enabled bool `json:"enabled"`

	// FailureThreshold defines failure threshold for rollback
	FailureThreshold int `json:"failureThreshold,omitempty"`

	// Window defines the time window for failure detection
	Window string `json:"window,omitempty"`

	// RevisionHistoryLimit defines how many revisions to keep
	// +kubebuilder:default:=10
	RevisionHistoryLimit int `json:"revisionHistoryLimit,omitempty"`
}

// GitOpsIntegrationStatus defines the observed state of GitOps integration
type GitOpsIntegrationStatus struct {
	// Phase represents the current phase of the integration
	// +kubebuilder:validation:Enum=Pending;Initializing;Syncing;Synced;Failed;Unknown
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime is the last successful sync time
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// LastSyncRevision is the last synced Git revision
	LastSyncRevision string `json:"lastSyncRevision,omitempty"`

	// ObservedGeneration is the generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// EnvironmentStatuses contains status for each environment
	EnvironmentStatuses []EnvironmentStatus `json:"environmentStatuses,omitempty"`

	// DriftStatus contains drift detection status
	DriftStatus *DriftStatus `json:"driftStatus,omitempty"`
}

// EnvironmentStatus represents the status of an environment
type EnvironmentStatus struct {
	// Name is the environment name
	Name string `json:"name"`

	// Phase is the environment phase
	Phase string `json:"phase,omitempty"`

	// LastSyncTime is the last sync time for this environment
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Revision is the current revision
	Revision string `json:"revision,omitempty"`

	// Ready indicates if the environment is ready
	Ready bool `json:"ready"`

	// Message contains additional information
	Message string `json:"message,omitempty"`
}

// DriftStatus represents drift detection status
type DriftStatus struct {
	// Detected indicates if drift was detected
	Detected bool `json:"detected"`

	// LastCheck is the last drift check time
	LastCheck *metav1.Time `json:"lastCheck,omitempty"`

	// DriftedResources contains list of drifted resources
	DriftedResources []DriftedResource `json:"driftedResources,omitempty"`

	// RemediationStatus contains remediation status
	RemediationStatus string `json:"remediationStatus,omitempty"`
}

// DriftedResource represents a resource that has drifted
type DriftedResource struct {
	// APIVersion is the resource API version
	APIVersion string `json:"apiVersion"`

	// Kind is the resource kind
	Kind string `json:"kind"`

	// Name is the resource name
	Name string `json:"name"`

	// Namespace is the resource namespace
	Namespace string `json:"namespace,omitempty"`

	// DriftType describes the type of drift
	// +kubebuilder:validation:Enum=Modified;Deleted;Added
	DriftType string `json:"driftType"`

	// Details contains drift details
	Details string `json:"details,omitempty"`
}
