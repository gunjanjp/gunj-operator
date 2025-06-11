/*
Copyright 2025.

Licensed under the MIT License.
*/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// Components defines which observability components to deploy
	Components Components `json:"components"`

	// +kubebuilder:validation:Optional
	// Global configuration settings applied to all components
	Global GlobalConfig `json:"global,omitempty"`

	// +kubebuilder:validation:Optional
	// HighAvailability defines the HA configuration for the platform
	HighAvailability *HighAvailabilityConfig `json:"highAvailability,omitempty"`

	// +kubebuilder:validation:Optional
	// Backup configuration for the platform
	Backup *BackupConfig `json:"backup,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Paused indicates whether the operator should reconcile this resource
	Paused bool `json:"paused,omitempty"`

	// +kubebuilder:validation:Optional
	// Alerting configuration for the platform
	Alerting *AlertingConfig `json:"alerting,omitempty"`
}

// Components defines the observability components configuration
type Components struct {
	// +kubebuilder:validation:Optional
	// Prometheus configuration
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// +kubebuilder:validation:Optional
	// Grafana configuration
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// +kubebuilder:validation:Optional
	// Loki configuration for log aggregation
	Loki *LokiSpec `json:"loki,omitempty"`

	// +kubebuilder:validation:Optional
	// Tempo configuration for distributed tracing
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// +kubebuilder:validation:Optional
	// OpenTelemetry Collector configuration
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines the Prometheus component configuration
type PrometheusSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates whether Prometheus should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+$`
	// Version of Prometheus to deploy
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// Replicas is the number of Prometheus instances
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// Resources defines the compute resources for Prometheus
	Resources ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// Storage configuration for Prometheus data
	Storage *StorageConfig `json:"storage,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30d"
	// +kubebuilder:validation:Pattern=`^\d+[hdwmy]$`
	// Retention period for metrics data
	Retention string `json:"retention,omitempty"`

	// +kubebuilder:validation:Optional
	// CustomConfig allows overriding the default Prometheus configuration
	CustomConfig map[string]string `json:"customConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ServiceMonitorSelector defines which ServiceMonitors should be selected
	ServiceMonitorSelector *metav1.LabelSelector `json:"serviceMonitorSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// RemoteWrite configurations for sending metrics to external systems
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`
}

// GrafanaSpec defines the Grafana component configuration
type GrafanaSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates whether Grafana should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+$`
	// Version of Grafana to deploy
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// Replicas is the number of Grafana instances
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// Resources defines the compute resources for Grafana
	Resources ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// AdminPassword for the Grafana admin user
	AdminPassword string `json:"adminPassword,omitempty"`

	// +kubebuilder:validation:Optional
	// Ingress configuration for external access
	Ingress *IngressConfig `json:"ingress,omitempty"`

	// +kubebuilder:validation:Optional
	// DataSources to configure in Grafana
	DataSources []DataSourceConfig `json:"dataSources,omitempty"`

	// +kubebuilder:validation:Optional
	// Dashboards to provision in Grafana
	Dashboards []DashboardConfig `json:"dashboards,omitempty"`
}

// LokiSpec defines the Loki component configuration
type LokiSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates whether Loki should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+$`
	// Version of Loki to deploy
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// Resources defines the compute resources for Loki
	Resources ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// Storage configuration for Loki data
	Storage *StorageConfig `json:"storage,omitempty"`

	// +kubebuilder:validation:Optional
	// S3 configuration for log storage
	S3 *S3Config `json:"s3,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="7d"
	// +kubebuilder:validation:Pattern=`^\d+[hdwmy]$`
	// Retention period for log data
	Retention string `json:"retention,omitempty"`
}

// TempoSpec defines the Tempo component configuration
type TempoSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates whether Tempo should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+$`
	// Version of Tempo to deploy
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// Resources defines the compute resources for Tempo
	Resources ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// Storage configuration for Tempo data
	Storage *StorageConfig `json:"storage,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="24h"
	// +kubebuilder:validation:Pattern=`^\d+[hdwmy]$`
	// Retention period for trace data
	Retention string `json:"retention,omitempty"`
}

// OpenTelemetryCollectorSpec defines the OpenTelemetry Collector configuration
type OpenTelemetryCollectorSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates whether OpenTelemetry Collector should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Required
	// Version of OpenTelemetry Collector to deploy
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// Replicas is the number of collector instances
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// Resources defines the compute resources for the collector
	Resources ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// Config is the OpenTelemetry Collector configuration
	Config string `json:"config,omitempty"`
}

// GlobalConfig defines global settings for all components
type GlobalConfig struct {
	// +kubebuilder:validation:Optional
	// ExternalLabels to add to all metrics
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="info"
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// LogLevel for all components
	LogLevel string `json:"logLevel,omitempty"`

	// +kubebuilder:validation:Optional
	// NodeSelector for all component pods
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// Tolerations for all component pods
	Tolerations []Toleration `json:"tolerations,omitempty"`

	// +kubebuilder:validation:Optional
	// SecurityContext for all component pods
	SecurityContext *SecurityContext `json:"securityContext,omitempty"`
}

// HighAvailabilityConfig defines HA settings
type HighAvailabilityConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates whether HA mode is enabled
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=2
	// MinReplicas for HA components
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// +kubebuilder:validation:Optional
	// AntiAffinity rules for pod distribution
	AntiAffinity *AntiAffinityConfig `json:"antiAffinity,omitempty"`
}

// BackupConfig defines backup settings
type BackupConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled indicates whether backups are enabled
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0 2 * * *"
	// Schedule in cron format
	Schedule string `json:"schedule,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=7
	// +kubebuilder:validation:Minimum=1
	// RetentionDays for backup data
	RetentionDays int32 `json:"retentionDays,omitempty"`

	// +kubebuilder:validation:Optional
	// Destination for backup storage
	Destination BackupDestination `json:"destination,omitempty"`
}

// AlertingConfig defines alerting configuration
type AlertingConfig struct {
	// +kubebuilder:validation:Optional
	// AlertManager configuration
	AlertManager *AlertManagerSpec `json:"alertmanager,omitempty"`

	// +kubebuilder:validation:Optional
	// Rules for alerting
	Rules []AlertingRule `json:"rules,omitempty"`
}

// ObservabilityPlatformStatus defines the observed state of ObservabilityPlatform
type ObservabilityPlatformStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	// Phase represents the current phase of the platform
	// +kubebuilder:validation:Enum=Pending;Installing;Ready;Failed;Upgrading
	Phase string `json:"phase,omitempty"`

	// +kubebuilder:validation:Optional
	// Message provides additional information about the current phase
	Message string `json:"message,omitempty"`

	// +kubebuilder:validation:Optional
	// LastReconcileTime is the last time the resource was reconciled
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// +kubebuilder:validation:Optional
	// ObservedGeneration reflects the generation of the most recently observed spec
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +kubebuilder:validation:Optional
	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:Optional
	// ComponentStatus tracks the status of each component
	ComponentStatus map[string]ComponentStatus `json:"componentStatus,omitempty"`

	// +kubebuilder:validation:Optional
	// Endpoints provides access URLs for components
	Endpoints map[string]string `json:"endpoints,omitempty"`
}

// ComponentStatus represents the status of a single component
type ComponentStatus struct {
	// +kubebuilder:validation:Optional
	// Phase of the component
	// +kubebuilder:validation:Enum=Pending;Installing;Ready;Failed;Upgrading
	Phase string `json:"phase,omitempty"`

	// +kubebuilder:validation:Optional
	// Version currently deployed
	Version string `json:"version,omitempty"`

	// +kubebuilder:validation:Optional
	// ReadyReplicas number
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// +kubebuilder:validation:Optional
	// LastUpdateTime of the component
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=op;ops
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,priority=1

// ObservabilityPlatform is the Schema for the observabilityplatforms API
type ObservabilityPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObservabilityPlatformSpec   `json:"spec,omitempty"`
	Status ObservabilityPlatformStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ObservabilityPlatformList contains a list of ObservabilityPlatform
type ObservabilityPlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ObservabilityPlatform `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ObservabilityPlatform{}, &ObservabilityPlatformList{})
}
