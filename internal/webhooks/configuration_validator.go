/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package webhooks

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// ConfigurationValidator validates ObservabilityPlatform configurations
type ConfigurationValidator struct {
	Log                logr.Logger
	VersionCompatibility *VersionCompatibilityMatrix
}

// VersionCompatibilityMatrix defines compatible versions between components
type VersionCompatibilityMatrix struct {
	// Prometheus version -> compatible versions map
	Prometheus map[string]ComponentVersions `json:"prometheus"`
	// Grafana version -> compatible versions map
	Grafana map[string]ComponentVersions `json:"grafana"`
	// Loki version -> compatible versions map
	Loki map[string]ComponentVersions `json:"loki"`
	// Tempo version -> compatible versions map
	Tempo map[string]ComponentVersions `json:"tempo"`
	// OpenTelemetry Collector version -> compatible versions map
	OTelCollector map[string]ComponentVersions `json:"otelCollector"`
}

// ComponentVersions defines compatible versions for other components
type ComponentVersions struct {
	Prometheus    []string `json:"prometheus,omitempty"`
	Grafana       []string `json:"grafana,omitempty"`
	Loki          []string `json:"loki,omitempty"`
	Tempo         []string `json:"tempo,omitempty"`
	OTelCollector []string `json:"otelCollector,omitempty"`
}

// DeprecatedConfig tracks deprecated configuration options
type DeprecatedConfig struct {
	Field       string
	Since       string
	Alternative string
	RemovalDate string
}

// NewConfigurationValidator creates a new configuration validator
func NewConfigurationValidator(log logr.Logger) *ConfigurationValidator {
	return &ConfigurationValidator{
		Log:                  log,
		VersionCompatibility: getDefaultVersionCompatibilityMatrix(),
	}
}

// ValidateConfiguration validates the entire ObservabilityPlatform configuration
func (v *ConfigurationValidator) ValidateConfiguration(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	allErrs := field.ErrorList{}
	log := v.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)

	// Validate component configurations
	if platform.Spec.Components != nil {
		allErrs = append(allErrs, v.validateComponents(platform.Spec.Components, field.NewPath("spec", "components"))...)
	}

	// Validate version compatibility
	allErrs = append(allErrs, v.validateVersionCompatibility(platform, field.NewPath("spec", "components"))...)

	// Check configuration conflicts
	allErrs = append(allErrs, v.validateConfigurationConflicts(platform, field.NewPath("spec"))...)

	// Validate configuration completeness
	allErrs = append(allErrs, v.validateConfigurationCompleteness(platform, field.NewPath("spec"))...)

	// Check for deprecated configurations
	warnings := v.checkDeprecatedConfigurations(platform)
	for _, warning := range warnings {
		log.Info("Deprecated configuration detected", "warning", warning)
	}

	// Validate configuration dependencies
	allErrs = append(allErrs, v.validateConfigurationDependencies(platform, field.NewPath("spec"))...)

	return allErrs
}

// validateComponents validates individual component configurations
func (v *ConfigurationValidator) validateComponents(components *observabilityv1beta1.Components, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate Prometheus configuration
	if components.Prometheus != nil && components.Prometheus.Enabled {
		allErrs = append(allErrs, v.validatePrometheusConfig(components.Prometheus, fldPath.Child("prometheus"))...)
	}

	// Validate Grafana configuration
	if components.Grafana != nil && components.Grafana.Enabled {
		allErrs = append(allErrs, v.validateGrafanaConfig(components.Grafana, fldPath.Child("grafana"))...)
	}

	// Validate Loki configuration
	if components.Loki != nil && components.Loki.Enabled {
		allErrs = append(allErrs, v.validateLokiConfig(components.Loki, fldPath.Child("loki"))...)
	}

	// Validate Tempo configuration
	if components.Tempo != nil && components.Tempo.Enabled {
		allErrs = append(allErrs, v.validateTempoConfig(components.Tempo, fldPath.Child("tempo"))...)
	}

	// Validate OpenTelemetry Collector configuration
	if components.OpenTelemetryCollector != nil && components.OpenTelemetryCollector.Enabled {
		allErrs = append(allErrs, v.validateOTelCollectorConfig(components.OpenTelemetryCollector, fldPath.Child("opentelemetryCollector"))...)
	}

	return allErrs
}

// validatePrometheusConfig validates Prometheus-specific configuration
func (v *ConfigurationValidator) validatePrometheusConfig(prometheus *observabilityv1beta1.PrometheusSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate version format
	if !isValidVersion(prometheus.Version) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), prometheus.Version, "invalid version format"))
	}

	// Validate resources
	if prometheus.Resources != nil {
		allErrs = append(allErrs, v.validateResourceRequirements(prometheus.Resources, fldPath.Child("resources"))...)
	}

	// Validate storage
	if prometheus.Storage != nil {
		allErrs = append(allErrs, v.validateStorageSpec(prometheus.Storage, fldPath.Child("storage"))...)
	}

	// Validate remote write configurations
	for i, rw := range prometheus.RemoteWrite {
		allErrs = append(allErrs, v.validateRemoteWriteSpec(rw, fldPath.Child("remoteWrite").Index(i))...)
	}

	// Validate additional scrape configs
	if prometheus.AdditionalScrapeConfigs != "" {
		if err := validateYAML(prometheus.AdditionalScrapeConfigs); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("additionalScrapeConfigs"), 
				"<scrape configs>", fmt.Sprintf("invalid YAML: %v", err)))
		}
	}

	// Validate external labels
	for k, v := range prometheus.ExternalLabels {
		if errs := validation.IsQualifiedName(k); len(errs) > 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("externalLabels").Key(k), k, 
				fmt.Sprintf("invalid label key: %s", strings.Join(errs, ", "))))
		}
		if errs := validation.IsValidLabelValue(v); len(errs) > 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("externalLabels").Key(k), v, 
				fmt.Sprintf("invalid label value: %s", strings.Join(errs, ", "))))
		}
	}

	return allErrs
}

// validateGrafanaConfig validates Grafana-specific configuration
func (v *ConfigurationValidator) validateGrafanaConfig(grafana *observabilityv1beta1.GrafanaSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate version format
	if !isValidVersion(grafana.Version) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), grafana.Version, "invalid version format"))
	}

	// Validate resources
	if grafana.Resources != nil {
		allErrs = append(allErrs, v.validateResourceRequirements(grafana.Resources, fldPath.Child("resources"))...)
	}

	// Validate ingress
	if grafana.Ingress != nil && grafana.Ingress.Enabled {
		allErrs = append(allErrs, v.validateIngressSpec(grafana.Ingress, fldPath.Child("ingress"))...)
	}

	// Validate admin credentials
	if grafana.AdminUser == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("adminUser"), "admin user is required"))
	}

	// Validate persistence
	if grafana.Persistence != nil && grafana.Persistence.Enabled {
		allErrs = append(allErrs, v.validatePersistenceSpec(grafana.Persistence, fldPath.Child("persistence"))...)
	}

	// Validate data sources
	for i, ds := range grafana.DataSources {
		allErrs = append(allErrs, v.validateDataSourceSpec(ds, fldPath.Child("dataSources").Index(i))...)
	}

	return allErrs
}

// validateLokiConfig validates Loki-specific configuration
func (v *ConfigurationValidator) validateLokiConfig(loki *observabilityv1beta1.LokiSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate version format
	if !isValidVersion(loki.Version) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), loki.Version, "invalid version format"))
	}

	// Validate resources
	if loki.Resources != nil {
		allErrs = append(allErrs, v.validateResourceRequirements(loki.Resources, fldPath.Child("resources"))...)
	}

	// Validate storage
	if loki.Storage != nil {
		allErrs = append(allErrs, v.validateLokiStorageSpec(loki.Storage, fldPath.Child("storage"))...)
	}

	// Validate retention
	if loki.Retention != nil {
		allErrs = append(allErrs, v.validateRetentionSpec(loki.Retention, fldPath.Child("retention"))...)
	}

	return allErrs
}

// validateTempoConfig validates Tempo-specific configuration
func (v *ConfigurationValidator) validateTempoConfig(tempo *observabilityv1beta1.TempoSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate version format
	if !isValidVersion(tempo.Version) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), tempo.Version, "invalid version format"))
	}

	// Validate resources
	if tempo.Resources != nil {
		allErrs = append(allErrs, v.validateResourceRequirements(tempo.Resources, fldPath.Child("resources"))...)
	}

	// Validate storage
	if tempo.Storage != nil {
		allErrs = append(allErrs, v.validateStorageSpec(tempo.Storage, fldPath.Child("storage"))...)
	}

	return allErrs
}

// validateOTelCollectorConfig validates OpenTelemetry Collector configuration
func (v *ConfigurationValidator) validateOTelCollectorConfig(otel *observabilityv1beta1.OpenTelemetryCollectorSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate version format
	if !isValidVersion(otel.Version) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), otel.Version, "invalid version format"))
	}

	// Validate mode
	validModes := map[string]bool{"deployment": true, "daemonset": true, "statefulset": true}
	if !validModes[otel.Mode] {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("mode"), otel.Mode, []string{"deployment", "daemonset", "statefulset"}))
	}

	// Validate resources
	if otel.Resources != nil {
		allErrs = append(allErrs, v.validateResourceRequirements(otel.Resources, fldPath.Child("resources"))...)
	}

	return allErrs
}

// validateVersionCompatibility checks version compatibility between components
func (v *ConfigurationValidator) validateVersionCompatibility(platform *observabilityv1beta1.ObservabilityPlatform, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	components := platform.Spec.Components

	if components == nil {
		return allErrs
	}

	// Check Prometheus compatibility with other components
	if components.Prometheus != nil && components.Prometheus.Enabled {
		prometheusVersion := normalizeVersion(components.Prometheus.Version)
		compatVersions, exists := v.VersionCompatibility.Prometheus[prometheusVersion]
		if exists {
			// Check Grafana compatibility
			if components.Grafana != nil && components.Grafana.Enabled {
				grafanaVersion := normalizeVersion(components.Grafana.Version)
				if !contains(compatVersions.Grafana, grafanaVersion) {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("grafana", "version"), 
						components.Grafana.Version, 
						fmt.Sprintf("Grafana version %s is not compatible with Prometheus version %s", 
							components.Grafana.Version, components.Prometheus.Version)))
				}
			}

			// Check Loki compatibility
			if components.Loki != nil && components.Loki.Enabled {
				lokiVersion := normalizeVersion(components.Loki.Version)
				if !contains(compatVersions.Loki, lokiVersion) {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("loki", "version"), 
						components.Loki.Version, 
						fmt.Sprintf("Loki version %s is not compatible with Prometheus version %s", 
							components.Loki.Version, components.Prometheus.Version)))
				}
			}
		}
	}

	// Check Grafana data source compatibility
	if components.Grafana != nil && components.Grafana.Enabled {
		for i, ds := range components.Grafana.DataSources {
			switch ds.Type {
			case "prometheus":
				if components.Prometheus == nil || !components.Prometheus.Enabled {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("grafana", "dataSources").Index(i), 
						ds.Type, "Prometheus data source requires Prometheus to be enabled"))
				}
			case "loki":
				if components.Loki == nil || !components.Loki.Enabled {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("grafana", "dataSources").Index(i), 
						ds.Type, "Loki data source requires Loki to be enabled"))
				}
			case "tempo":
				if components.Tempo == nil || !components.Tempo.Enabled {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("grafana", "dataSources").Index(i), 
						ds.Type, "Tempo data source requires Tempo to be enabled"))
				}
			}
		}
	}

	return allErrs
}

// validateConfigurationConflicts checks for conflicting configurations
func (v *ConfigurationValidator) validateConfigurationConflicts(platform *observabilityv1beta1.ObservabilityPlatform, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	components := platform.Spec.Components

	if components == nil {
		return allErrs
	}

	// Check resource conflicts in HA mode
	if platform.Spec.HighAvailability != nil && platform.Spec.HighAvailability.Enabled {
		// Ensure replicas are appropriate for HA
		if components.Prometheus != nil && components.Prometheus.Enabled && components.Prometheus.Replicas < 2 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("components", "prometheus", "replicas"), 
				components.Prometheus.Replicas, "HA mode requires at least 2 replicas"))
		}

		if components.Grafana != nil && components.Grafana.Enabled && components.Grafana.Replicas < 2 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("components", "grafana", "replicas"), 
				components.Grafana.Replicas, "HA mode requires at least 2 replicas"))
		}

		// Check storage requirements for HA
		if components.Prometheus != nil && components.Prometheus.Enabled {
			if components.Prometheus.Storage == nil {
				allErrs = append(allErrs, field.Required(fldPath.Child("components", "prometheus", "storage"), 
					"storage is required for Prometheus in HA mode"))
			}
		}
	}

	// Check for conflicting global vs component settings
	if platform.Spec.Global != nil {
		// Check node selector conflicts
		if len(platform.Spec.Global.NodeSelector) > 0 && len(platform.Spec.NodeSelector) > 0 {
			for k := range platform.Spec.NodeSelector {
				if _, exists := platform.Spec.Global.NodeSelector[k]; exists {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("nodeSelector").Key(k), 
						platform.Spec.NodeSelector[k], 
						"conflicting node selector in global and platform settings"))
				}
			}
		}
	}

	// Check for storage class conflicts
	storageClasses := make(map[string]bool)
	if components.Prometheus != nil && components.Prometheus.Storage != nil && components.Prometheus.Storage.StorageClassName != "" {
		storageClasses[components.Prometheus.Storage.StorageClassName] = true
	}
	if components.Loki != nil && components.Loki.Storage != nil && components.Loki.Storage.StorageClassName != "" {
		if storageClasses[components.Loki.Storage.StorageClassName] {
			v.Log.Info("Multiple components using same storage class", 
				"storageClass", components.Loki.Storage.StorageClassName)
		}
	}

	return allErrs
}

// validateConfigurationCompleteness ensures all required configurations are present
func (v *ConfigurationValidator) validateConfigurationCompleteness(platform *observabilityv1beta1.ObservabilityPlatform, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	components := platform.Spec.Components

	// Ensure at least one component is enabled
	if components == nil || !hasEnabledComponent(components) {
		allErrs = append(allErrs, field.Required(fldPath.Child("components"), 
			"at least one component must be enabled"))
	}

	// Check backup configuration completeness
	if platform.Spec.Backup != nil && platform.Spec.Backup.Enabled {
		if platform.Spec.Backup.Schedule == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("backup", "schedule"), 
				"backup schedule is required when backup is enabled"))
		}
		if platform.Spec.Backup.StorageType == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("backup", "storageType"), 
				"storage type is required when backup is enabled"))
		}
		// Validate storage config based on type
		switch platform.Spec.Backup.StorageType {
		case "s3":
			if platform.Spec.Backup.StorageConfig == nil {
				allErrs = append(allErrs, field.Required(fldPath.Child("backup", "storageConfig"), 
					"storage config is required for S3 backup"))
			} else {
				requiredS3Fields := []string{"bucket", "region"}
				for _, field := range requiredS3Fields {
					if platform.Spec.Backup.StorageConfig[field] == "" {
						allErrs = append(allErrs, field.Required(fldPath.Child("backup", "storageConfig").Key(field), 
							fmt.Sprintf("%s is required for S3 backup", field)))
					}
				}
			}
		}
	}

	// Check alerting configuration completeness
	if platform.Spec.Alerting != nil && platform.Spec.Alerting.Alertmanager != nil && platform.Spec.Alerting.Alertmanager.Enabled {
		// Ensure Prometheus is enabled for alerting
		if components == nil || components.Prometheus == nil || !components.Prometheus.Enabled {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("alerting", "alertmanager"), 
				"enabled", "Alertmanager requires Prometheus to be enabled"))
		}
	}

	return allErrs
}

// checkDeprecatedConfigurations checks for deprecated configuration options
func (v *ConfigurationValidator) checkDeprecatedConfigurations(platform *observabilityv1beta1.ObservabilityPlatform) []string {
	warnings := []string{}
	deprecatedConfigs := getDeprecatedConfigurations()

	// Check for deprecated fields (this is a placeholder - in real implementation, 
	// you would use reflection or specific checks based on your deprecation policy)
	for _, deprecated := range deprecatedConfigs {
		warning := fmt.Sprintf("Field '%s' is deprecated since %s. Use '%s' instead. Will be removed on %s",
			deprecated.Field, deprecated.Since, deprecated.Alternative, deprecated.RemovalDate)
		
		// Check if deprecated field is used (simplified check)
		if fieldExists(platform, deprecated.Field) {
			warnings = append(warnings, warning)
		}
	}

	return warnings
}

// validateConfigurationDependencies validates configuration dependencies
func (v *ConfigurationValidator) validateConfigurationDependencies(platform *observabilityv1beta1.ObservabilityPlatform, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	components := platform.Spec.Components

	if components == nil {
		return allErrs
	}

	// OpenTelemetry Collector dependencies
	if components.OpenTelemetryCollector != nil && components.OpenTelemetryCollector.Enabled {
		// If OTel is enabled, at least one backend should be enabled
		hasBackend := false
		if components.Prometheus != nil && components.Prometheus.Enabled {
			hasBackend = true
		}
		if components.Loki != nil && components.Loki.Enabled {
			hasBackend = true
		}
		if components.Tempo != nil && components.Tempo.Enabled {
			hasBackend = true
		}

		if !hasBackend {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("components", "opentelemetryCollector"), 
				"enabled", "OpenTelemetry Collector requires at least one backend (Prometheus, Loki, or Tempo) to be enabled"))
		}
	}

	// Grafana dependencies
	if components.Grafana != nil && components.Grafana.Enabled {
		// Check if any data source is configured
		if len(components.Grafana.DataSources) == 0 {
			v.Log.Info("Grafana enabled without data sources configured")
		}
	}

	// Remote write dependencies
	if components.Prometheus != nil && components.Prometheus.Enabled {
		for i, rw := range components.Prometheus.RemoteWrite {
			// Check authentication dependencies
			authMethods := 0
			if rw.BasicAuth != nil {
				authMethods++
			}
			if rw.BearerToken != "" {
				authMethods++
			}
			if authMethods > 1 {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("components", "prometheus", "remoteWrite").Index(i), 
					"authentication", "only one authentication method should be specified"))
			}
		}
	}

	return allErrs
}

// Helper functions

// validateResourceRequirements validates resource requests and limits
func (v *ConfigurationValidator) validateResourceRequirements(resources *observabilityv1beta1.ResourceRequirements, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate requests
	if resources.Requests != nil {
		allErrs = append(allErrs, v.validateResourceList(resources.Requests, fldPath.Child("requests"))...)
	}

	// Validate limits
	if resources.Limits != nil {
		allErrs = append(allErrs, v.validateResourceList(resources.Limits, fldPath.Child("limits"))...)
	}

	// Ensure limits >= requests
	if resources.Requests != nil && resources.Limits != nil {
		if resources.Requests.CPU != "" && resources.Limits.CPU != "" {
			reqCPU, err1 := resource.ParseQuantity(resources.Requests.CPU)
			limCPU, err2 := resource.ParseQuantity(resources.Limits.CPU)
			if err1 == nil && err2 == nil && reqCPU.Cmp(limCPU) > 0 {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("requests", "cpu"), 
					resources.Requests.CPU, "CPU request cannot exceed limit"))
			}
		}

		if resources.Requests.Memory != "" && resources.Limits.Memory != "" {
			reqMem, err1 := resource.ParseQuantity(resources.Requests.Memory)
			limMem, err2 := resource.ParseQuantity(resources.Limits.Memory)
			if err1 == nil && err2 == nil && reqMem.Cmp(limMem) > 0 {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("requests", "memory"), 
					resources.Requests.Memory, "Memory request cannot exceed limit"))
			}
		}
	}

	return allErrs
}

// validateResourceList validates CPU and memory resource specifications
func (v *ConfigurationValidator) validateResourceList(resources *observabilityv1beta1.ResourceList, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if resources.CPU != "" {
		if _, err := resource.ParseQuantity(resources.CPU); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("cpu"), resources.CPU, 
				fmt.Sprintf("invalid CPU quantity: %v", err)))
		}
	}

	if resources.Memory != "" {
		if _, err := resource.ParseQuantity(resources.Memory); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("memory"), resources.Memory, 
				fmt.Sprintf("invalid memory quantity: %v", err)))
		}
	}

	return allErrs
}

// validateStorageSpec validates storage configuration
func (v *ConfigurationValidator) validateStorageSpec(storage *observabilityv1beta1.StorageSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate storage size
	if _, err := resource.ParseQuantity(storage.Size); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("size"), storage.Size, 
			fmt.Sprintf("invalid storage size: %v", err)))
	}

	// Validate retention format
	if storage.Retention != "" {
		if !isValidRetention(storage.Retention) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("retention"), storage.Retention, 
				"invalid retention format, must match pattern: ^\\d+[hdwmy]$"))
		}
	}

	return allErrs
}

// validateLokiStorageSpec validates Loki-specific storage configuration
func (v *ConfigurationValidator) validateLokiStorageSpec(storage *observabilityv1beta1.LokiStorageSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate base storage spec
	allErrs = append(allErrs, v.validateStorageSpec(&storage.StorageSpec, fldPath)...)

	// Validate S3 configuration if enabled
	if storage.S3 != nil && storage.S3.Enabled {
		if storage.S3.BucketName == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("s3", "bucketName"), 
				"bucket name is required when S3 is enabled"))
		}
		if storage.S3.Region == "" && storage.S3.Endpoint == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("s3", "region"), 
				"region is required when S3 is enabled (unless custom endpoint is specified)"))
		}
	}

	return allErrs
}

// validateIngressSpec validates ingress configuration
func (v *ConfigurationValidator) validateIngressSpec(ingress *observabilityv1beta1.IngressSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if ingress.Host == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("host"), 
			"host is required when ingress is enabled"))
	}

	// Validate TLS configuration
	if ingress.TLS != nil && ingress.TLS.Enabled {
		if ingress.TLS.SecretName == "" {
			v.Log.Info("TLS enabled without secret name, expecting cert-manager or similar to provide certificate")
		}
	}

	// Validate annotations
	for k, v := range ingress.Annotations {
		if strings.TrimSpace(k) == "" {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("annotations"), k, 
				"annotation key cannot be empty"))
		}
		if strings.TrimSpace(v) == "" {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("annotations").Key(k), v, 
				"annotation value cannot be empty"))
		}
	}

	return allErrs
}

// validatePersistenceSpec validates persistence configuration
func (v *ConfigurationValidator) validatePersistenceSpec(persistence *observabilityv1beta1.PersistenceSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if _, err := resource.ParseQuantity(persistence.Size); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("size"), persistence.Size, 
			fmt.Sprintf("invalid storage size: %v", err)))
	}

	return allErrs
}

// validateDataSourceSpec validates Grafana data source configuration
func (v *ConfigurationValidator) validateDataSourceSpec(ds observabilityv1beta1.DataSourceSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if ds.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "data source name is required"))
	}

	if ds.URL == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("url"), "data source URL is required"))
	}

	validTypes := map[string]bool{
		"prometheus":    true,
		"loki":          true,
		"tempo":         true,
		"elasticsearch": true,
	}
	if !validTypes[ds.Type] {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("type"), ds.Type, 
			[]string{"prometheus", "loki", "tempo", "elasticsearch"}))
	}

	validAccessModes := map[string]bool{"proxy": true, "direct": true}
	if !validAccessModes[ds.Access] {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("access"), ds.Access, 
			[]string{"proxy", "direct"}))
	}

	return allErrs
}

// validateRemoteWriteSpec validates Prometheus remote write configuration
func (v *ConfigurationValidator) validateRemoteWriteSpec(rw observabilityv1beta1.RemoteWriteSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if rw.URL == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("url"), "remote write URL is required"))
	}

	// Validate basic auth if provided
	if rw.BasicAuth != nil {
		if rw.BasicAuth.Username == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("basicAuth", "username"), 
				"username is required for basic auth"))
		}
		if rw.BasicAuth.Password == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("basicAuth", "password"), 
				"password is required for basic auth"))
		}
	}

	// Validate TLS config if provided
	if rw.TLSConfig != nil {
		if rw.TLSConfig.CertFile != "" && rw.TLSConfig.KeyFile == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("tlsConfig", "keyFile"), 
				"key file is required when cert file is specified"))
		}
		if rw.TLSConfig.KeyFile != "" && rw.TLSConfig.CertFile == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("tlsConfig", "certFile"), 
				"cert file is required when key file is specified"))
		}
	}

	return allErrs
}

// validateRetentionSpec validates retention configuration
func (v *ConfigurationValidator) validateRetentionSpec(retention *observabilityv1beta1.RetentionSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Add retention-specific validation here
	// This is a placeholder for when RetentionSpec is fully defined

	return allErrs
}

// Utility functions

// isValidVersion checks if a version string is valid
func isValidVersion(version string) bool {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")
	
	// Split by dots
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	// Each part should be a number
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

// normalizeVersion removes 'v' prefix and returns major.minor version
func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

// isValidRetention checks if retention format is valid (e.g., 30d, 12h, 52w)
func isValidRetention(retention string) bool {
	if len(retention) < 2 {
		return false
	}

	// Check suffix
	suffix := retention[len(retention)-1:]
	if !strings.Contains("hdwmy", suffix) {
		return false
	}

	// Check numeric part
	numPart := retention[:len(retention)-1]
	if _, err := strconv.Atoi(numPart); err != nil {
		return false
	}

	return true
}

// validateYAML checks if a string is valid YAML
func validateYAML(yamlStr string) error {
	// This is a simplified check - in production, you would use a YAML parser
	if strings.TrimSpace(yamlStr) == "" {
		return fmt.Errorf("empty YAML")
	}
	// Add actual YAML validation here
	return nil
}

// hasEnabledComponent checks if at least one component is enabled
func hasEnabledComponent(components *observabilityv1beta1.Components) bool {
	if components.Prometheus != nil && components.Prometheus.Enabled {
		return true
	}
	if components.Grafana != nil && components.Grafana.Enabled {
		return true
	}
	if components.Loki != nil && components.Loki.Enabled {
		return true
	}
	if components.Tempo != nil && components.Tempo.Enabled {
		return true
	}
	if components.OpenTelemetryCollector != nil && components.OpenTelemetryCollector.Enabled {
		return true
	}
	return false
}

// contains checks if a string slice contains a specific value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// fieldExists checks if a field exists in the platform spec (simplified)
func fieldExists(platform *observabilityv1beta1.ObservabilityPlatform, fieldPath string) bool {
	// This is a simplified implementation
	// In a real implementation, you would use reflection to check field existence
	return false
}

// getDefaultVersionCompatibilityMatrix returns the default version compatibility matrix
func getDefaultVersionCompatibilityMatrix() *VersionCompatibilityMatrix {
	return &VersionCompatibilityMatrix{
		Prometheus: map[string]ComponentVersions{
			"2.48": {
				Grafana:       []string{"10.0", "10.1", "10.2"},
				Loki:          []string{"2.9"},
				Tempo:         []string{"2.3"},
				OTelCollector: []string{"0.91", "0.92"},
			},
			"2.47": {
				Grafana:       []string{"9.5", "10.0", "10.1"},
				Loki:          []string{"2.8", "2.9"},
				Tempo:         []string{"2.2", "2.3"},
				OTelCollector: []string{"0.88", "0.89", "0.90", "0.91"},
			},
			"2.46": {
				Grafana:       []string{"9.4", "9.5", "10.0"},
				Loki:          []string{"2.8"},
				Tempo:         []string{"2.2"},
				OTelCollector: []string{"0.86", "0.87", "0.88"},
			},
		},
		Grafana: map[string]ComponentVersions{
			"10.2": {
				Prometheus: []string{"2.46", "2.47", "2.48"},
				Loki:       []string{"2.8", "2.9"},
				Tempo:      []string{"2.2", "2.3"},
			},
			"10.1": {
				Prometheus: []string{"2.45", "2.46", "2.47", "2.48"},
				Loki:       []string{"2.8", "2.9"},
				Tempo:      []string{"2.2", "2.3"},
			},
			"10.0": {
				Prometheus: []string{"2.44", "2.45", "2.46", "2.47", "2.48"},
				Loki:       []string{"2.7", "2.8", "2.9"},
				Tempo:      []string{"2.1", "2.2", "2.3"},
			},
		},
		Loki: map[string]ComponentVersions{
			"2.9": {
				Prometheus:    []string{"2.46", "2.47", "2.48"},
				Grafana:       []string{"10.0", "10.1", "10.2"},
				OTelCollector: []string{"0.89", "0.90", "0.91"},
			},
			"2.8": {
				Prometheus:    []string{"2.44", "2.45", "2.46", "2.47"},
				Grafana:       []string{"9.5", "10.0", "10.1"},
				OTelCollector: []string{"0.85", "0.86", "0.87", "0.88"},
			},
		},
		Tempo: map[string]ComponentVersions{
			"2.3": {
				Prometheus:    []string{"2.46", "2.47", "2.48"},
				Grafana:       []string{"10.0", "10.1", "10.2"},
				OTelCollector: []string{"0.89", "0.90", "0.91"},
			},
			"2.2": {
				Prometheus:    []string{"2.44", "2.45", "2.46", "2.47"},
				Grafana:       []string{"9.5", "10.0", "10.1"},
				OTelCollector: []string{"0.85", "0.86", "0.87", "0.88"},
			},
		},
		OTelCollector: map[string]ComponentVersions{
			"0.91": {
				Prometheus: []string{"2.47", "2.48"},
				Loki:       []string{"2.9"},
				Tempo:      []string{"2.3"},
			},
			"0.90": {
				Prometheus: []string{"2.46", "2.47", "2.48"},
				Loki:       []string{"2.8", "2.9"},
				Tempo:      []string{"2.2", "2.3"},
			},
		},
	}
}

// getDeprecatedConfigurations returns list of deprecated configurations
func getDeprecatedConfigurations() []DeprecatedConfig {
	return []DeprecatedConfig{
		{
			Field:       "spec.components.prometheus.adminUser",
			Since:       "v2.1.0",
			Alternative: "spec.components.prometheus.auth.adminUser",
			RemovalDate: "v3.0.0",
		},
		{
			Field:       "spec.components.grafana.adminPassword",
			Since:       "v2.1.0",
			Alternative: "spec.components.grafana.auth.adminPassword",
			RemovalDate: "v3.0.0",
		},
		// Add more deprecated configurations as needed
	}
}
