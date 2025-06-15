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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GitOpsDeploymentSpec defines the desired state of GitOpsDeployment
type GitOpsDeploymentSpec struct {
	// GitProvider specifies the type of Git provider
	// +kubebuilder:validation:Enum=github;gitlab;bitbucket;generic
	GitProvider string `json:"gitProvider"`

	// Repository is the Git repository configuration
	Repository GitRepository `json:"repository"`

	// GitOpsEngine specifies which GitOps tool to use
	// +kubebuilder:validation:Enum=argocd;flux
	// +kubebuilder:default=argocd
	GitOpsEngine string `json:"gitOpsEngine,omitempty"`

	// ArgoCD specific configuration
	// +optional
	ArgoCD *ArgoCDConfig `json:"argocd,omitempty"`

	// Flux specific configuration
	// +optional
	Flux *FluxConfig `json:"flux,omitempty"`

	// Environments defines the environments and their promotion flow
	Environments []Environment `json:"environments"`

	// AutoSync enables automatic synchronization
	// +kubebuilder:default=true
	AutoSync bool `json:"autoSync,omitempty"`

	// SyncPolicy defines the sync policy
	// +optional
	SyncPolicy *SyncPolicy `json:"syncPolicy,omitempty"`

	// Rollback configuration for automatic rollbacks
	// +optional
	Rollback *RollbackConfig `json:"rollback,omitempty"`

	// DriftDetection configuration
	// +optional
	DriftDetection *DriftDetectionConfig `json:"driftDetection,omitempty"`
}

// GitRepository defines Git repository configuration
type GitRepository struct {
	// URL is the repository URL
	URL string `json:"url"`

	// Branch is the target branch
	// +kubebuilder:default=main
	Branch string `json:"branch,omitempty"`

	// Path is the path within the repository
	// +kubebuilder:default="/"
	Path string `json:"path,omitempty"`

	// SecretRef is the reference to the secret containing credentials
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// PollInterval is how often to check for changes
	// +kubebuilder:default="1m"
	PollInterval string `json:"pollInterval,omitempty"`

	// Webhook configuration for real-time updates
	// +optional
	Webhook *WebhookConfig `json:"webhook,omitempty"`
}

// ArgoCDConfig defines ArgoCD-specific configuration
type ArgoCDConfig struct {
	// ApplicationName is the name of the ArgoCD Application
	ApplicationName string `json:"applicationName"`

	// Project is the ArgoCD project name
	// +kubebuilder:default=default
	Project string `json:"project,omitempty"`

	// SyncOptions for ArgoCD
	// +optional
	SyncOptions []string `json:"syncOptions,omitempty"`

	// IgnoreDifferences configuration
	// +optional
	IgnoreDifferences []ResourceIgnoreDifferences `json:"ignoreDifferences,omitempty"`

	// RetryPolicy for sync operations
	// +optional
	RetryPolicy *ArgoCDRetryPolicy `json:"retryPolicy,omitempty"`
}

// FluxConfig defines Flux-specific configuration
type FluxConfig struct {
	// KustomizationName is the name of the Flux Kustomization
	KustomizationName string `json:"kustomizationName"`

	// ServiceAccount to use for reconciliation
	// +kubebuilder:default=default
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// Interval at which to reconcile
	// +kubebuilder:default="5m"
	Interval string `json:"interval,omitempty"`

	// Timeout for reconciliation operations
	// +kubebuilder:default="10m"
	Timeout string `json:"timeout,omitempty"`

	// Prune enables garbage collection
	// +kubebuilder:default=true
	Prune bool `json:"prune,omitempty"`

	// HealthChecks configuration
	// +optional
	HealthChecks []FluxHealthCheck `json:"healthChecks,omitempty"`
}

// Environment defines a deployment environment
type Environment struct {
	// Name is the environment name
	Name string `json:"name"`

	// Namespace is the target namespace
	Namespace string `json:"namespace"`

	// Branch is the Git branch for this environment
	// +optional
	Branch string `json:"branch,omitempty"`

	// Path is the path within the repository for this environment
	// +optional
	Path string `json:"path,omitempty"`

	// PromotionPolicy defines how to promote to this environment
	// +optional
	PromotionPolicy *PromotionPolicy `json:"promotionPolicy,omitempty"`

	// Values for environment-specific configuration
	// +optional
	Values map[string]string `json:"values,omitempty"`

	// PreSync hooks to run before sync
	// +optional
	PreSync []SyncHook `json:"preSync,omitempty"`

	// PostSync hooks to run after sync
	// +optional
	PostSync []SyncHook `json:"postSync,omitempty"`
}

// PromotionPolicy defines how deployments are promoted between environments
type PromotionPolicy struct {
	// AutoPromotion enables automatic promotion
	// +kubebuilder:default=false
	AutoPromotion bool `json:"autoPromotion,omitempty"`

	// FromEnvironment is the source environment
	FromEnvironment string `json:"fromEnvironment,omitempty"`

	// ApprovalRequired indicates if manual approval is needed
	// +kubebuilder:default=true
	ApprovalRequired bool `json:"approvalRequired,omitempty"`

	// Conditions that must be met for promotion
	// +optional
	Conditions []PromotionCondition `json:"conditions,omitempty"`
}

// SyncPolicy defines synchronization policies
type SyncPolicy struct {
	// SyncOptions is a list of sync options
	// +optional
	SyncOptions []string `json:"syncOptions,omitempty"`

	// Retry configuration
	// +optional
	Retry *RetryPolicy `json:"retry,omitempty"`

	// Automated sync policy
	// +optional
	Automated *AutomatedSyncPolicy `json:"automated,omitempty"`
}

// RollbackConfig defines rollback configuration
type RollbackConfig struct {
	// Enabled enables automatic rollback on failures
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// MaxRetries before rollback
	// +kubebuilder:default=3
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// FailureThreshold percentage to trigger rollback
	// +kubebuilder:default=50
	FailureThreshold int32 `json:"failureThreshold,omitempty"`

	// RevisionHistoryLimit number of revisions to keep
	// +kubebuilder:default=10
	RevisionHistoryLimit int32 `json:"revisionHistoryLimit,omitempty"`
}

// DriftDetectionConfig defines drift detection configuration
type DriftDetectionConfig struct {
	// Enabled enables drift detection
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// CheckInterval is how often to check for drift
	// +kubebuilder:default="5m"
	CheckInterval string `json:"checkInterval,omitempty"`

	// AutoRemediate enables automatic remediation of drift
	// +kubebuilder:default=false
	AutoRemediate bool `json:"autoRemediate,omitempty"`

	// IgnoreFields lists fields to ignore during drift detection
	// +optional
	IgnoreFields []string `json:"ignoreFields,omitempty"`
}

// GitOpsDeploymentStatus defines the observed state of GitOpsDeployment
type GitOpsDeploymentStatus struct {
	// Phase represents the current phase of the deployment
	// +kubebuilder:validation:Enum=Pending;Initializing;Syncing;Synced;Failed;Unknown
	Phase string `json:"phase,omitempty"`

	// LastSyncTime is the time of the last successful sync
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// LastSyncRevision is the Git revision of the last sync
	// +optional
	LastSyncRevision string `json:"lastSyncRevision,omitempty"`

	// Environments status
	// +optional
	Environments []EnvironmentStatus `json:"environments,omitempty"`

	// SyncStatus represents the current sync status
	// +optional
	SyncStatus *SyncStatus `json:"syncStatus,omitempty"`

	// HealthStatus represents the health of the deployment
	// +optional
	HealthStatus *HealthStatus `json:"healthStatus,omitempty"`

	// DriftStatus represents drift detection status
	// +optional
	DriftStatus *DriftStatus `json:"driftStatus,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// EnvironmentStatus represents the status of a specific environment
type EnvironmentStatus struct {
	// Name is the environment name
	Name string `json:"name"`

	// Phase is the current phase
	Phase string `json:"phase"`

	// LastSyncTime is the last sync time for this environment
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Revision is the current Git revision
	// +optional
	Revision string `json:"revision,omitempty"`

	// Resources deployed in this environment
	// +optional
	Resources []ResourceStatus `json:"resources,omitempty"`
}

// ResourceStatus represents the status of a deployed resource
type ResourceStatus struct {
	// Group is the API group
	Group string `json:"group"`

	// Version is the API version
	Version string `json:"version"`

	// Kind is the resource kind
	Kind string `json:"kind"`

	// Name is the resource name
	Name string `json:"name"`

	// Namespace is the resource namespace
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Status is the resource status
	Status string `json:"status"`

	// Health is the resource health
	// +optional
	Health string `json:"health,omitempty"`
}

// SyncStatus represents synchronization status
type SyncStatus struct {
	// Status is the sync status
	// +kubebuilder:validation:Enum=Synced;OutOfSync;Unknown
	Status string `json:"status"`

	// Revision is the target revision
	Revision string `json:"revision,omitempty"`

	// Message provides additional information
	Message string `json:"message,omitempty"`
}

// HealthStatus represents health status
type HealthStatus struct {
	// Status is the health status
	// +kubebuilder:validation:Enum=Healthy;Progressing;Degraded;Suspended;Missing;Unknown
	Status string `json:"status"`

	// Message provides additional information
	Message string `json:"message,omitempty"`
}

// DriftStatus represents drift detection status
type DriftStatus struct {
	// Detected indicates if drift was detected
	Detected bool `json:"detected"`

	// LastCheck is the time of the last drift check
	// +optional
	LastCheck *metav1.Time `json:"lastCheck,omitempty"`

	// DriftedResources lists resources that have drifted
	// +optional
	DriftedResources []DriftedResource `json:"driftedResources,omitempty"`
}

// DriftedResource represents a resource that has drifted from desired state
type DriftedResource struct {
	// Resource identification
	ResourceStatus `json:",inline"`

	// DriftType indicates the type of drift
	// +kubebuilder:validation:Enum=Modified;Added;Removed
	DriftType string `json:"driftType"`

	// Details provides drift details
	// +optional
	Details string `json:"details,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gitops;god
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.gitOpsEngine`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Sync",type=string,JSONPath=`.status.syncStatus.status`
// +kubebuilder:printcolumn:name="Health",type=string,JSONPath=`.status.healthStatus.status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GitOpsDeployment is the Schema for the gitopsdeployments API
type GitOpsDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitOpsDeploymentSpec   `json:"spec,omitempty"`
	Status GitOpsDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitOpsDeploymentList contains a list of GitOpsDeployment
type GitOpsDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitOpsDeployment `json:"items"`
}

// Helper types

// SecretReference references a secret
type SecretReference struct {
	// Name is the secret name
	Name string `json:"name"`

	// Key is the key within the secret
	// +optional
	Key string `json:"key,omitempty"`
}

// WebhookConfig defines webhook configuration
type WebhookConfig struct {
	// Enabled enables webhook support
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Secret for webhook validation
	// +optional
	Secret string `json:"secret,omitempty"`
}

// ResourceIgnoreDifferences defines differences to ignore
type ResourceIgnoreDifferences struct {
	// Group is the API group
	// +optional
	Group string `json:"group,omitempty"`

	// Kind is the resource kind
	Kind string `json:"kind"`

	// Name is the resource name
	// +optional
	Name string `json:"name,omitempty"`

	// JSONPointers lists JSON pointers to ignore
	// +optional
	JSONPointers []string `json:"jsonPointers,omitempty"`

	// JQPathExpressions lists JQ path expressions to ignore
	// +optional
	JQPathExpressions []string `json:"jqPathExpressions,omitempty"`
}

// ArgoCDRetryPolicy defines ArgoCD retry policy
type ArgoCDRetryPolicy struct {
	// Limit is the maximum number of attempts
	// +optional
	Limit *int64 `json:"limit,omitempty"`

	// Backoff configuration
	// +optional
	Backoff *Backoff `json:"backoff,omitempty"`
}

// Backoff defines backoff configuration
type Backoff struct {
	// Duration is the base duration
	// +optional
	Duration string `json:"duration,omitempty"`

	// Factor is the multiplication factor
	// +optional
	Factor *int64 `json:"factor,omitempty"`

	// MaxDuration is the maximum duration
	// +optional
	MaxDuration string `json:"maxDuration,omitempty"`
}

// FluxHealthCheck defines Flux health check
type FluxHealthCheck struct {
	// Type is the health check type
	Type string `json:"type"`

	// Name is the health check name
	Name string `json:"name"`

	// Namespace is the resource namespace
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// PromotionCondition defines a condition for promotion
type PromotionCondition struct {
	// Type is the condition type
	Type string `json:"type"`

	// Status is the required status
	Status string `json:"status"`

	// Reason is the required reason
	// +optional
	Reason string `json:"reason,omitempty"`
}

// SyncHook defines a sync hook
type SyncHook struct {
	// Name is the hook name
	Name string `json:"name"`

	// Type is the hook type
	// +kubebuilder:validation:Enum=Job;Webhook;Script
	Type string `json:"type"`

	// Config is the hook configuration
	// +kubebuilder:pruning:PreserveUnknownFields
	Config map[string]interface{} `json:"config,omitempty"`
}

// RetryPolicy defines retry configuration
type RetryPolicy struct {
	// Limit is the maximum number of retries
	// +optional
	Limit *int64 `json:"limit,omitempty"`

	// Backoff configuration
	// +optional
	Backoff *Backoff `json:"backoff,omitempty"`
}

// AutomatedSyncPolicy defines automated sync configuration
type AutomatedSyncPolicy struct {
	// Prune enables pruning of resources
	// +kubebuilder:default=false
	Prune bool `json:"prune,omitempty"`

	// SelfHeal enables self-healing
	// +kubebuilder:default=false
	SelfHeal bool `json:"selfHeal,omitempty"`

	// AllowEmpty allows syncing empty directories
	// +kubebuilder:default=false
	AllowEmpty bool `json:"allowEmpty,omitempty"`
}

func init() {
	SchemeBuilder.Register(&GitOpsDeployment{}, &GitOpsDeploymentList{})
}
