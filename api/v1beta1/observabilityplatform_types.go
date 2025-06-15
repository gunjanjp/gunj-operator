/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`

	// GitOps configuration for declarative management
	// +optional
	GitOps *GitOpsConfig `json:"gitOps,omitempty"`

	// ServiceMesh configuration for service mesh integration
	// +optional
	ServiceMesh *ServiceMeshSpec `json:"serviceMesh,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// Enabled determines if Prometheus should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Prometheus to deploy
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}


	// +kubebuilder:default="v2.48.0"
	Version string `json:"version"`

	// Replicas is the number of Prometheus instances to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources for Prometheus
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// RemoteWrite configurations
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// AdditionalScrapeConfigs allows adding custom scrape configurations
	// +optional
	AdditionalScrapeConfigs string `json:"additionalScrapeConfigs,omitempty"`

	// AdminUser for Prometheus (if auth is enabled)
	// +optional
	AdminUser string `json:"adminUser,omitempty"`
}

// GrafanaSpec defines Grafana configuration
type GrafanaSpec struct {
	// Enabled determines if Grafana should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Grafana to deploy
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// Enabled determines if Prometheus should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Prometheus to deploy
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}


	// +kubebuilder:default="v2.48.0"
	Version string `json:"version"`

	// Replicas is the number of Prometheus instances to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources for Prometheus
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// RemoteWrite configurations
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// AdditionalScrapeConfigs allows adding custom scrape configurations
	// +optional
	AdditionalScrapeConfigs string `json:"additionalScrapeConfigs,omitempty"`

	// AdminUser for Prometheus (if auth is enabled)
	// +optional
	AdminUser string `json:"adminUser,omitempty"`
}


	// +kubebuilder:default="10.2.0"
	Version string `json:"version"`

	// Replicas is the number of Grafana instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Ingress configuration for external access
	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// AdminUser for Grafana admin account
	// +optional
	// +kubebuilder:default="admin"
	AdminUser string `json:"adminUser,omitempty"`

	// AdminPassword for the admin user
	// If not set, a random password will be generated
	// +optional
	AdminPassword string `json:"adminPassword,omitempty"`

	// Persistence configuration
	// +optional
	Persistence *PersistenceSpec `json:"persistence,omitempty"`

	// DataSources to configure automatically
	// +optional
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
}

// LokiSpec defines Loki configuration
type LokiSpec struct {
	// Enabled determines if Loki should be deployed
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Version of Loki to deploy
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// Enabled determines if Prometheus should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Prometheus to deploy
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}


	// +kubebuilder:default="v2.48.0"
	Version string `json:"version"`

	// Replicas is the number of Prometheus instances to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources for Prometheus
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// RemoteWrite configurations
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// AdditionalScrapeConfigs allows adding custom scrape configurations
	// +optional
	AdditionalScrapeConfigs string `json:"additionalScrapeConfigs,omitempty"`

	// AdminUser for Prometheus (if auth is enabled)
	// +optional
	AdminUser string `json:"adminUser,omitempty"`
}

// GrafanaSpec defines Grafana configuration
type GrafanaSpec struct {
	// Enabled determines if Grafana should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Grafana to deploy
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// Enabled determines if Prometheus should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Prometheus to deploy
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}


	// +kubebuilder:default="v2.48.0"
	Version string `json:"version"`

	// Replicas is the number of Prometheus instances to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources for Prometheus
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// RemoteWrite configurations
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// AdditionalScrapeConfigs allows adding custom scrape configurations
	// +optional
	AdditionalScrapeConfigs string `json:"additionalScrapeConfigs,omitempty"`

	// AdminUser for Prometheus (if auth is enabled)
	// +optional
	AdminUser string `json:"adminUser,omitempty"`
}


	// +kubebuilder:default="10.2.0"
	Version string `json:"version"`

	// Replicas is the number of Grafana instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Ingress configuration for external access
	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// AdminUser for Grafana admin account
	// +optional
	// +kubebuilder:default="admin"
	AdminUser string `json:"adminUser,omitempty"`

	// AdminPassword for the admin user
	// If not set, a random password will be generated
	// +optional
	AdminPassword string `json:"adminPassword,omitempty"`

	// Persistence configuration
	// +optional
	Persistence *PersistenceSpec `json:"persistence,omitempty"`

	// DataSources to configure automatically
	// +optional
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
}


	// +kubebuilder:default="2.9.0"
	Version string `json:"version"`

	// Replicas is the number of Loki instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration
	// +optional
	Storage *LokiStorageSpec `json:"storage,omitempty"`

	// Retention configuration for logs
	// +optional
	Retention *RetentionSpec `json:"retention,omitempty"`
}

// TempoSpec defines Tempo configuration
type TempoSpec struct {
	// Enabled determines if Tempo should be deployed
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Version of Tempo to deploy
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// Enabled determines if Prometheus should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Prometheus to deploy
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}


	// +kubebuilder:default="v2.48.0"
	Version string `json:"version"`

	// Replicas is the number of Prometheus instances to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources for Prometheus
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// RemoteWrite configurations
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// AdditionalScrapeConfigs allows adding custom scrape configurations
	// +optional
	AdditionalScrapeConfigs string `json:"additionalScrapeConfigs,omitempty"`

	// AdminUser for Prometheus (if auth is enabled)
	// +optional
	AdminUser string `json:"adminUser,omitempty"`
}

// GrafanaSpec defines Grafana configuration
type GrafanaSpec struct {
	// Enabled determines if Grafana should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Grafana to deploy
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// Enabled determines if Prometheus should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Prometheus to deploy
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}


	// +kubebuilder:default="v2.48.0"
	Version string `json:"version"`

	// Replicas is the number of Prometheus instances to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources for Prometheus
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// RemoteWrite configurations
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// AdditionalScrapeConfigs allows adding custom scrape configurations
	// +optional
	AdditionalScrapeConfigs string `json:"additionalScrapeConfigs,omitempty"`

	// AdminUser for Prometheus (if auth is enabled)
	// +optional
	AdminUser string `json:"adminUser,omitempty"`
}


	// +kubebuilder:default="10.2.0"
	Version string `json:"version"`

	// Replicas is the number of Grafana instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Ingress configuration for external access
	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// AdminUser for Grafana admin account
	// +optional
	// +kubebuilder:default="admin"
	AdminUser string `json:"adminUser,omitempty"`

	// AdminPassword for the admin user
	// If not set, a random password will be generated
	// +optional
	AdminPassword string `json:"adminPassword,omitempty"`

	// Persistence configuration
	// +optional
	Persistence *PersistenceSpec `json:"persistence,omitempty"`

	// DataSources to configure automatically
	// +optional
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
}

// LokiSpec defines Loki configuration
type LokiSpec struct {
	// Enabled determines if Loki should be deployed
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Version of Loki to deploy
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// Enabled determines if Prometheus should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Prometheus to deploy
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}


	// +kubebuilder:default="v2.48.0"
	Version string `json:"version"`

	// Replicas is the number of Prometheus instances to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources for Prometheus
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// RemoteWrite configurations
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// AdditionalScrapeConfigs allows adding custom scrape configurations
	// +optional
	AdditionalScrapeConfigs string `json:"additionalScrapeConfigs,omitempty"`

	// AdminUser for Prometheus (if auth is enabled)
	// +optional
	AdminUser string `json:"adminUser,omitempty"`
}

// GrafanaSpec defines Grafana configuration
type GrafanaSpec struct {
	// Enabled determines if Grafana should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Grafana to deploy
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// Enabled determines if Prometheus should be deployed
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Version of Prometheus to deploy
	// +kubebuilder:validation:Pattern=`^v?\d+\.\d+\.\d+/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityPlatformSpec defines the desired state of ObservabilityPlatform
type ObservabilityPlatformSpec struct {
	// Components defines which observability components to deploy
	// +kubebuilder:validation:Required
	Components *Components `json:"components"`

	// Global configuration settings that apply to all components
	// +optional
	Global *GlobalSettings `json:"global,omitempty"`

	// HighAvailability enables HA mode for all components
	// +optional
	HighAvailability *HighAvailabilitySettings `json:"highAvailability,omitempty"`

	// Backup configuration for the platform
	// +optional
	Backup *BackupSettings `json:"backup,omitempty"`

	// Alerting configuration
	// +optional
	Alerting *AlertingSettings `json:"alerting,omitempty"`

	// NodeSelector for the entire platform (overrides global settings)
	// This is used for multi-tenant isolation
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Paused indicates whether the operator should reconcile this resource
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// Components defines the observability components to deploy
type Components struct {
	// Prometheus configuration
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Loki configuration for log aggregation
	// +optional
	Loki *LokiSpec `json:"loki,omitempty"`

	// Tempo configuration for distributed tracing
	// +optional
	Tempo *TempoSpec `json:"tempo,omitempty"`

	// OpenTelemetry Collector configuration
	// +optional
	OpenTelemetryCollector *OpenTelemetryCollectorSpec `json:"opentelemetryCollector,omitempty"`
}


	// +kubebuilder:default="v2.48.0"
	Version string `json:"version"`

	// Replicas is the number of Prometheus instances to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources for Prometheus
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// RemoteWrite configurations
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// AdditionalScrapeConfigs allows adding custom scrape configurations
	// +optional
	AdditionalScrapeConfigs string `json:"additionalScrapeConfigs,omitempty"`

	// AdminUser for Prometheus (if auth is enabled)
	// +optional
	AdminUser string `json:"adminUser,omitempty"`
}


	// +kubebuilder:default="10.2.0"
	Version string `json:"version"`

	// Replicas is the number of Grafana instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Ingress configuration for external access
	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// AdminUser for Grafana admin account
	// +optional
	// +kubebuilder:default="admin"
	AdminUser string `json:"adminUser,omitempty"`

	// AdminPassword for the admin user
	// If not set, a random password will be generated
	// +optional
	AdminPassword string `json:"adminPassword,omitempty"`

	// Persistence configuration
	// +optional
	Persistence *PersistenceSpec `json:"persistence,omitempty"`

	// DataSources to configure automatically
	// +optional
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
}


	// +kubebuilder:default="2.9.0"
	Version string `json:"version"`

	// Replicas is the number of Loki instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration
	// +optional
	Storage *LokiStorageSpec `json:"storage,omitempty"`

	// Retention configuration for logs
	// +optional
	Retention *RetentionSpec `json:"retention,omitempty"`
}


	// +kubebuilder:default="2.3.0"
	Version string `json:"version"`

	// Replicas is the number of Tempo instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`
}

// OpenTelemetryCollectorSpec defines OpenTelemetry Collector configuration
type OpenTelemetryCollectorSpec struct {
	// Enabled determines if OpenTelemetry Collector should be deployed
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Version of OpenTelemetry Collector to deploy
	// +kubebuilder:default="0.91.0"
	Version string `json:"version"`

	// Mode of deployment (deployment, daemonset, statefulset)
	// +kubebuilder:validation:Enum=deployment;daemonset;statefulset
	// +kubebuilder:default="deployment"
	Mode string `json:"mode"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`
}

// ResourceRequirements defines resource requests and limits
type ResourceRequirements struct {
	// Requests describes the minimum amount of compute resources required
	// +optional
	Requests *ResourceList `json:"requests,omitempty"`

	// Limits describes the maximum amount of compute resources allowed
	// +optional
	Limits *ResourceList `json:"limits,omitempty"`
}

// ResourceList defines CPU and memory resources
type ResourceList struct {
	// CPU resource quantity
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory resource quantity
	// +optional
	Memory string `json:"memory,omitempty"`
}

// StorageSpec defines storage configuration
type StorageSpec struct {
	// Size of the persistent volume
	// +kubebuilder:default="10Gi"
	Size string `json:"size"`

	// StorageClassName to use for the PVC
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// Retention period for data
	// +kubebuilder:default="30d"
	// +kubebuilder:validation:Pattern=`^\d+[hdwmy]$`
	Retention string `json:"retention,omitempty"`
}

// LokiStorageSpec extends StorageSpec with S3 configuration
type LokiStorageSpec struct {
	StorageSpec `json:",inline"`

	// S3 configuration for remote storage
	// +optional
	S3 *S3StorageSpec `json:"s3,omitempty"`
}

// S3StorageSpec defines S3 storage configuration
type S3StorageSpec struct {
	// Enabled determines if S3 storage should be used
	Enabled bool `json:"enabled"`

	// BucketName is the S3 bucket name
	BucketName string `json:"bucketName,omitempty"`

	// Region is the AWS region
	Region string `json:"region,omitempty"`

	// Endpoint for S3-compatible storage
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// SecretName containing AWS credentials
	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// IngressSpec defines ingress configuration
type IngressSpec struct {
	// Enabled determines if ingress should be created
	Enabled bool `json:"enabled"`

	// ClassName specifies the ingress class to use
	// +kubebuilder:default="nginx"
	ClassName string `json:"className,omitempty"`

	// Host is the hostname for the ingress
	Host string `json:"host,omitempty"`

	// TLS configuration
	// +optional
	TLS *TLSSpec `json:"tls,omitempty"`

	// Annotations to add to the ingress
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// TLSSpec defines TLS configuration for ingress
type TLSSpec struct {
	// Enabled determines if TLS should be enabled
	Enabled bool `json:"enabled"`

	// SecretName containing the TLS certificate
	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// PersistenceSpec defines persistence configuration
type PersistenceSpec struct {
	// Enabled determines if persistence should be enabled
	Enabled bool `json:"enabled"`

	// Size of the persistent volume
	// +kubebuilder:default="1Gi"
	Size string `json:"size"`

	// StorageClassName to use for the PVC
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
}

// DataSourceSpec defines a Grafana datasource
type DataSourceSpec struct {
	// Name of the datasource
	Name string `json:"name"`

	// Type of the datasource (prometheus, loki, tempo)
	// +kubebuilder:validation:Enum=prometheus;loki;tempo;elasticsearch
	Type string `json:"type"`

	// URL of the datasource
	URL string `json:"url"`

	// IsDefault marks this as the default datasource
	// +optional
	IsDefault bool `json:"isDefault,omitempty"`

	// Access mode (proxy or direct)
	// +kubebuilder:validation:Enum=proxy;direct
	// +kubebuilder:default="proxy"
	Access string `json:"access,omitempty"`
}

// RemoteWriteSpec defines Prometheus remote write configuration
type RemoteWriteSpec struct {
	// URL of the remote write endpoint
	URL string `json:"url"`

	// Name of the remote write config
	// +optional
	Name string `json:"name,omitempty"`

	// BasicAuth configuration
	// +optional
	BasicAuth *BasicAuthSpec `json:"basicAuth,omitempty"`

	// BearerToken for authentication
	// +optional
	BearerToken string `json:"bearerToken,omitempty"`

	// TLSConfig for the remote write endpoint
	// +optional
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`
}

// BasicAuthSpec defines basic authentication
type BasicAuthSpec struct {
	// Username for basic auth
	Username string `json:"username"`

	// Password for basic auth
	Password string `json:"password"`
}

// TLSConfig defines TLS configuration
type TLSConfig struct {
	// InsecureSkipVerify disables TLS certificate verification
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// CAFile is the path to the CA certificate
	// +optional
	CAFile string `json:"caFile,omitempty"`

	// CertFile is the path to the client certificate
	// +optional
	CertFile string `json:"certFile,omitempty"`

	// KeyFile is the path to the client key
	// +optional
	KeyFile string `json:"keyFile,omitempty"`
}

// GlobalSettings defines global configuration
type GlobalSettings struct {
	// ExternalLabels to add to all metrics
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// LogLevel for all components
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// +kubebuilder:default="info"
	LogLevel string `json:"logLevel,omitempty"`

	// NodeSelector for all components
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations for all components
	// +optional
	Tolerations []Toleration `json:"tolerations,omitempty"`

	// SecurityContext for all components
	// +optional
	SecurityContext *SecurityContextSpec `json:"securityContext,omitempty"`

	// RetentionPolicies defines data retention for each component
	// +optional
	RetentionPolicies *RetentionPolicies `json:"retentionPolicies,omitempty"`
}

// Toleration represents a Kubernetes toleration
type Toleration struct {
	// Key is the taint key that the toleration applies to
	// +optional
	Key string `json:"key,omitempty"`

	// Operator represents a key's relationship to the value
	// +kubebuilder:validation:Enum=Exists;Equal
	// +optional
	Operator string `json:"operator,omitempty"`

	// Value is the taint value the toleration matches to
	// +optional
	Value string `json:"value,omitempty"`

	// Effect indicates the taint effect to match
	// +kubebuilder:validation:Enum=NoSchedule;PreferNoSchedule;NoExecute
	// +optional
	Effect string `json:"effect,omitempty"`

	// TolerationSeconds represents the period of time the toleration tolerates the taint
	// +optional
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty"`
}

// SecurityContextSpec defines security context
type SecurityContextSpec struct {
	// RunAsNonRoot indicates that the container must run as a non-root user
	// +optional
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`

	// RunAsUser is the UID to run the entrypoint of the container process
	// +optional
	RunAsUser *int64 `json:"runAsUser,omitempty"`

	// RunAsGroup is the GID to run the entrypoint of the container process
	// +optional
	RunAsGroup *int64 `json:"runAsGroup,omitempty"`

	// FSGroup is the group ID for the volumes
	// +optional
	FSGroup *int64 `json:"fsGroup,omitempty"`
}

// HighAvailabilitySettings defines HA configuration
type HighAvailabilitySettings struct {
	// Enabled determines if HA mode should be enabled
	Enabled bool `json:"enabled"`

	// AntiAffinity level (hard, soft)
	// +kubebuilder:validation:Enum=hard;soft
	// +kubebuilder:default="soft"
	AntiAffinity string `json:"antiAffinity,omitempty"`
}

// BackupSettings defines backup configuration
type BackupSettings struct {
	// Enabled determines if backups should be enabled
	Enabled bool `json:"enabled"`

	// Schedule in cron format
	Schedule string `json:"schedule,omitempty"`

	// Retention in days
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=7
	Retention int32 `json:"retention"`

	// StorageType (s3, azure, gcs, local)
	// +kubebuilder:validation:Enum=s3;azure;gcs;local
	StorageType string `json:"storageType,omitempty"`

	// StorageConfig contains storage-specific configuration
	// +optional
	StorageConfig map[string]string `json:"storageConfig,omitempty"`
}

// AlertingSettings defines alerting configuration
type AlertingSettings struct {
	// Alertmanager configuration
	// +optional
	Alertmanager *AlertmanagerSpec `json:"alertmanager,omitempty"`

	// Rules defines alerting rules
	// +optional
	Rules []AlertingRule `json:"rules,omitempty"`
}

// AlertmanagerSpec defines Alertmanager configuration
type AlertmanagerSpec struct {
	// Enabled determines if Alertmanager should be deployed
	Enabled bool `json:"enabled"`

	// Version of Alertmanager to deploy
	// +kubebuilder:default="v0.26.0"
	Version string `json:"version"`

	// Replicas is the number of Alertmanager instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Resources defines the compute resources
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Storage configuration
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// Config is the Alertmanager configuration
	// +optional
	Config *AlertmanagerConfig `json:"config,omitempty"`
}

// AlertingRule defines an alerting rule
type AlertingRule struct {
	// Name of the rule
	Name string `json:"name"`

	// Expression is the PromQL expression
	Expression string `json:"expression"`

	// Duration before firing
	// +kubebuilder:default="5m"
	Duration string `json:"duration"`

	// Labels to add to the alert
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to add to the alert
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ObservabilityPlatformStatus defines the observed state of ObservabilityPlatform
type ObservabilityPlatformStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed spec
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase summarizes the overall status of the platform
	// +kubebuilder:validation:Enum=Pending;Installing;Ready;Failed;Upgrading;Deleting
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ComponentStatuses contains the status of each component
	ComponentStatuses map[string]ComponentStatus `json:"componentStatuses,omitempty"`

	// LastReconcileTime is the last time the resource was reconciled
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// Message provides additional information about the current status
	Message string `json:"message,omitempty"`

	// Endpoints contains the accessible endpoints for each component
	Endpoints map[string]string `json:"endpoints,omitempty"`

	// GitOps contains GitOps-related status information
	// +optional
	GitOps *GitOpsStatus `json:"gitOps,omitempty"`

	// ServiceMesh contains service mesh status information
	// +optional
	ServiceMesh *ServiceMeshStatus `json:"serviceMesh,omitempty"`
}

// ComponentStatus represents the status of a single component
type ComponentStatus struct {
	// Ready indicates if the component is ready
	Ready bool `json:"ready"`

	// Version is the deployed version
	Version string `json:"version"`

	// Replicas is the number of ready replicas
	Replicas int32 `json:"replicas"`

	// Message provides additional status information
	Message string `json:"message,omitempty"`

	// LastUpdateTime is when the component was last updated
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=op;ops
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.components.prometheus.version`,priority=1

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
