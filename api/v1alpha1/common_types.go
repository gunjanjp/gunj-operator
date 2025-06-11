/*
Copyright 2025.

Licensed under the MIT License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ResourceRequirements defines the compute resource requirements
type ResourceRequirements struct {
	// +kubebuilder:validation:Optional
	// Limits describes the maximum amount of compute resources allowed
	Limits corev1.ResourceList `json:"limits,omitempty"`

	// +kubebuilder:validation:Optional
	// Requests describes the minimum amount of compute resources required
	Requests corev1.ResourceList `json:"requests,omitempty"`
}

// StorageConfig defines storage configuration
type StorageConfig struct {
	// +kubebuilder:validation:Required
	// Size of the storage
	Size resource.Quantity `json:"size"`

	// +kubebuilder:validation:Optional
	// StorageClassName to use for the PVC
	StorageClassName *string `json:"storageClassName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="ReadWriteOnce"
	// +kubebuilder:validation:Enum=ReadWriteOnce;ReadWriteMany;ReadWriteOncePod
	// AccessMode for the storage
	AccessMode string `json:"accessMode,omitempty"`
}

// S3Config defines S3 storage configuration
type S3Config struct {
	// +kubebuilder:validation:Required
	// Enabled indicates whether S3 storage is enabled
	Enabled bool `json:"enabled"`

	// +kubebuilder:validation:Required
	// BucketName for storing data
	BucketName string `json:"bucketName"`

	// +kubebuilder:validation:Required
	// Region of the S3 bucket
	Region string `json:"region"`

	// +kubebuilder:validation:Optional
	// Endpoint for S3-compatible storage
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// SecretName containing AWS credentials
	SecretName string `json:"secretName,omitempty"`
}

// IngressConfig defines ingress configuration
type IngressConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled indicates whether ingress is enabled
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="nginx"
	// ClassName for the ingress controller
	ClassName string `json:"className,omitempty"`

	// +kubebuilder:validation:Required
	// Host for the ingress
	Host string `json:"host"`

	// +kubebuilder:validation:Optional
	// Path for the ingress
	Path string `json:"path,omitempty"`

	// +kubebuilder:validation:Optional
	// TLSSecret name for HTTPS
	TLSSecret string `json:"tlsSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// Annotations for the ingress
	Annotations map[string]string `json:"annotations,omitempty"`
}

// DataSourceConfig defines Grafana datasource configuration
type DataSourceConfig struct {
	// +kubebuilder:validation:Required
	// Name of the datasource
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=prometheus;loki;tempo;elasticsearch
	// Type of the datasource
	Type string `json:"type"`

	// +kubebuilder:validation:Required
	// URL of the datasource
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Default indicates if this is the default datasource
	Default bool `json:"default,omitempty"`

	// +kubebuilder:validation:Optional
	// Additional datasource configuration
	Config map[string]interface{} `json:"config,omitempty"`
}

// DashboardConfig defines Grafana dashboard configuration
type DashboardConfig struct {
	// +kubebuilder:validation:Required
	// Name of the dashboard
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// ConfigMapName containing the dashboard JSON
	ConfigMapName string `json:"configMapName,omitempty"`

	// +kubebuilder:validation:Optional
	// URL to download the dashboard
	URL string `json:"url,omitempty"`

	// +kubebuilder:validation:Optional
	// Folder to place the dashboard in
	Folder string `json:"folder,omitempty"`
}

// RemoteWriteSpec defines Prometheus remote write configuration
type RemoteWriteSpec struct {
	// +kubebuilder:validation:Required
	// URL of the remote write endpoint
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	// Name of the remote write config
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	// BasicAuth configuration
	BasicAuth *BasicAuth `json:"basicAuth,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerToken for authentication
	BearerToken string `json:"bearerToken,omitempty"`

	// +kubebuilder:validation:Optional
	// TLSConfig for the remote write endpoint
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// WriteRelabelConfigs for the remote write
	WriteRelabelConfigs []RelabelConfig `json:"writeRelabelConfigs,omitempty"`
}

// BasicAuth defines basic authentication configuration
type BasicAuth struct {
	// +kubebuilder:validation:Required
	// Username for basic auth
	Username string `json:"username"`

	// +kubebuilder:validation:Required
	// Password for basic auth (reference to secret)
	Password corev1.SecretKeySelector `json:"password"`
}

// TLSConfig defines TLS configuration
type TLSConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// InsecureSkipVerify disables TLS verification
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// +kubebuilder:validation:Optional
	// CAFile path for TLS verification
	CAFile string `json:"caFile,omitempty"`

	// +kubebuilder:validation:Optional
	// CertFile path for client certificate
	CertFile string `json:"certFile,omitempty"`

	// +kubebuilder:validation:Optional
	// KeyFile path for client key
	KeyFile string `json:"keyFile,omitempty"`
}

// RelabelConfig defines Prometheus relabel configuration
type RelabelConfig struct {
	// +kubebuilder:validation:Optional
	// SourceLabels to use for relabeling
	SourceLabels []string `json:"sourceLabels,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=";"
	// Separator to use between source labels
	Separator string `json:"separator,omitempty"`

	// +kubebuilder:validation:Optional
	// TargetLabel to set
	TargetLabel string `json:"targetLabel,omitempty"`

	// +kubebuilder:validation:Optional
	// Regex to match
	Regex string `json:"regex,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0
	// Modulus for hash-based relabeling
	Modulus uint64 `json:"modulus,omitempty"`

	// +kubebuilder:validation:Optional
	// Replacement value
	Replacement string `json:"replacement,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="replace"
	// +kubebuilder:validation:Enum=replace;keep;drop;hashmod;labelmap;labeldrop;labelkeep
	// Action to perform
	Action string `json:"action,omitempty"`
}

// Toleration defines pod toleration
type Toleration struct {
	// +kubebuilder:validation:Optional
	// Key of the toleration
	Key string `json:"key,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Exists;Equal
	// Operator of the toleration
	Operator string `json:"operator,omitempty"`

	// +kubebuilder:validation:Optional
	// Value of the toleration
	Value string `json:"value,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=NoSchedule;PreferNoSchedule;NoExecute
	// Effect of the toleration
	Effect string `json:"effect,omitempty"`

	// +kubebuilder:validation:Optional
	// TolerationSeconds for NoExecute effect
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty"`
}

// SecurityContext defines security settings for pods
type SecurityContext struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=65534
	// RunAsUser specifies the user ID
	RunAsUser *int64 `json:"runAsUser,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=65534
	// RunAsGroup specifies the group ID
	RunAsGroup *int64 `json:"runAsGroup,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// RunAsNonRoot indicates the container must run as non-root
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=65534
	// FSGroup specifies the filesystem group
	FSGroup *int64 `json:"fsGroup,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// ReadOnlyRootFilesystem mounts the root filesystem as read-only
	ReadOnlyRootFilesystem *bool `json:"readOnlyRootFilesystem,omitempty"`
}

// AntiAffinityConfig defines anti-affinity rules
type AntiAffinityConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="preferred"
	// +kubebuilder:validation:Enum=required;preferred
	// Type of anti-affinity
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="kubernetes.io/hostname"
	// TopologyKey for spreading pods
	TopologyKey string `json:"topologyKey,omitempty"`
}

// BackupDestination defines backup storage destination
type BackupDestination struct {
	// +kubebuilder:validation:Optional
	// S3 destination configuration
	S3 *S3BackupConfig `json:"s3,omitempty"`

	// +kubebuilder:validation:Optional
	// PVC destination configuration
	PVC *PVCBackupConfig `json:"pvc,omitempty"`
}

// S3BackupConfig defines S3 backup destination
type S3BackupConfig struct {
	// +kubebuilder:validation:Required
	// BucketName for backups
	BucketName string `json:"bucketName"`

	// +kubebuilder:validation:Required
	// Region of the S3 bucket
	Region string `json:"region"`

	// +kubebuilder:validation:Optional
	// Prefix for backup objects
	Prefix string `json:"prefix,omitempty"`

	// +kubebuilder:validation:Optional
	// SecretName containing AWS credentials
	SecretName string `json:"secretName,omitempty"`
}

// PVCBackupConfig defines PVC backup destination
type PVCBackupConfig struct {
	// +kubebuilder:validation:Required
	// ClaimName of the PVC
	ClaimName string `json:"claimName"`

	// +kubebuilder:validation:Optional
	// SubPath within the PVC
	SubPath string `json:"subPath,omitempty"`
}

// AlertManagerSpec defines AlertManager configuration
type AlertManagerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates whether AlertManager should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Required
	// Version of AlertManager to deploy
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// Replicas is the number of AlertManager instances
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// Resources defines the compute resources for AlertManager
	Resources ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// Config is the AlertManager configuration
	Config string `json:"config,omitempty"`
}

// AlertingRule defines alerting rule configuration
type AlertingRule struct {
	// +kubebuilder:validation:Required
	// Name of the alerting rule
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// Expr is the PromQL expression
	Expr string `json:"expr"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// For duration before firing
	For string `json:"for,omitempty"`

	// +kubebuilder:validation:Optional
	// Labels to add to the alert
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	// Annotations to add to the alert
	Annotations map[string]string `json:"annotations,omitempty"`
}
