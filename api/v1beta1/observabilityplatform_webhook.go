/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/gunjanjp/gunj-operator/internal/webhook/quota"
	"github.com/gunjanjp/gunj-operator/internal/webhooks"
)

// log is for logging in this package.
var observabilityplatformlog = logf.Log.WithName("observabilityplatform-resource")

// ObservabilityPlatformWebhook handles validation and mutation
type ObservabilityPlatformWebhook struct {
	Client                 client.Client
	QuotaValidator         *quota.ResourceQuotaValidator
	ConfigurationValidator *webhooks.ConfigurationValidator
}

// SetupWebhookWithManager sets up the webhook with the Manager
func (r *ObservabilityPlatform) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// Create quota validator with cluster limits (these could be configured)
	quotaValidator := &quota.ResourceQuotaValidator{
		Client: mgr.GetClient(),
		ClusterResourceLimits: &quota.ClusterResourceLimits{
			MaxCPUPerNamespace:    resource.MustParse("100"),
			MaxMemoryPerNamespace: resource.MustParse("200Gi"),
			MaxStoragePerPVC:      resource.MustParse("1Ti"),
			MaxPVCPerNamespace:    50,
			MaxPodsPerNamespace:   100,
		},
	}

	// Create configuration validator
	configValidator := webhooks.NewConfigurationValidator(mgr.GetLogger().WithName("configuration-validator"))

	// Store the validators in global variables for use in validation
	globalQuotaValidator = quotaValidator
	globalConfigValidator = configValidator

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// Global validator instances
var (
	globalQuotaValidator  *quota.ResourceQuotaValidator
	globalConfigValidator *webhooks.ConfigurationValidator
)

// +kubebuilder:webhook:path=/mutate-observability-io-v1beta1-observabilityplatform,mutating=true,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1beta1,name=mobservabilityplatform.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-observability-io-v1beta1-observabilityplatform,mutating=false,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1beta1,name=vobservabilityplatform.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ObservabilityPlatform{}
var _ webhook.Validator = &ObservabilityPlatform{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ObservabilityPlatform) Default() {
	observabilityplatformlog.Info("default", "name", r.Name)
	
	// Set default metadata
	r.defaultMetadata()
	
	// Initialize spec if nil
	if r.Spec.Components == nil {
		r.Spec.Components = &Components{}
	}
	
	// Set defaults for each component
	r.defaultPrometheus()
	r.defaultGrafana()
	r.defaultLoki()
	r.defaultTempo()
	
	// Set global defaults
	r.defaultGlobalSettings()
	
	// Set high availability defaults
	r.defaultHighAvailability()
	
	// Set backup defaults
	r.defaultBackup()
	
	// Set alerting defaults
	r.defaultAlerting()
}

// defaultMetadata sets default labels and annotations
func (r *ObservabilityPlatform) defaultMetadata() {
	// Initialize labels map if nil
	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	
	// Add default labels
	if _, ok := r.Labels["app.kubernetes.io/managed-by"]; !ok {
		r.Labels["app.kubernetes.io/managed-by"] = "gunj-operator"
	}
	if _, ok := r.Labels["app.kubernetes.io/name"]; !ok {
		r.Labels["app.kubernetes.io/name"] = "observability-platform"
	}
	if _, ok := r.Labels["app.kubernetes.io/instance"]; !ok {
		r.Labels["app.kubernetes.io/instance"] = r.Name
	}
	if _, ok := r.Labels["app.kubernetes.io/version"]; !ok {
		r.Labels["app.kubernetes.io/version"] = "v2.0.0"
	}
	
	// Initialize annotations map if nil
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	
	// Add default annotations
	if _, ok := r.Annotations["observability.io/created-by"]; !ok {
		r.Annotations["observability.io/created-by"] = "gunj-operator"
	}
}

// defaultPrometheus sets defaults for Prometheus component
func (r *ObservabilityPlatform) defaultPrometheus() {
	if r.Spec.Components.Prometheus == nil {
		return
	}
	
	prom := r.Spec.Components.Prometheus
	
	// Set default version if not specified
	if prom.Version == "" {
		prom.Version = "v2.48.0"
	}
	
	// Set default replicas
	if prom.Replicas == 0 {
		if r.Spec.HighAvailability != nil && r.Spec.HighAvailability.Enabled {
			prom.Replicas = 3
		} else {
			prom.Replicas = 1
		}
	}
	
	// Set default resources
	if prom.Resources == nil {
		prom.Resources = &ResourceRequirements{
			Requests: &ResourceList{
				CPU:    "250m",
				Memory: "512Mi",
			},
			Limits: &ResourceList{
				CPU:    "1",
				Memory: "2Gi",
			},
		}
	} else {
		// Ensure requests and limits are set
		if prom.Resources.Requests == nil {
			prom.Resources.Requests = &ResourceList{
				CPU:    "250m",
				Memory: "512Mi",
			}
		}
		if prom.Resources.Limits == nil {
			prom.Resources.Limits = &ResourceList{
				CPU:    "1",
				Memory: "2Gi",
			}
		}
	}
	
	// Set default storage
	if prom.Storage == nil {
		prom.Storage = &StorageSpec{
			Size:      "10Gi",
			Retention: "15d",
		}
	} else {
		if prom.Storage.Size == "" {
			prom.Storage.Size = "10Gi"
		}
		if prom.Storage.Retention == "" {
			prom.Storage.Retention = "15d"
		}
	}
	
	// Set default external labels
	if prom.ExternalLabels == nil {
		prom.ExternalLabels = make(map[string]string)
	}
	// Add cluster label if not present
	if _, ok := prom.ExternalLabels["cluster"]; !ok {
		prom.ExternalLabels["cluster"] = "default"
	}
}

// defaultGrafana sets defaults for Grafana component
func (r *ObservabilityPlatform) defaultGrafana() {
	if r.Spec.Components.Grafana == nil {
		return
	}
	
	grafana := r.Spec.Components.Grafana
	
	// Set default version
	if grafana.Version == "" {
		grafana.Version = "10.2.0"
	}
	
	// Set default replicas
	if grafana.Replicas == 0 {
		if r.Spec.HighAvailability != nil && r.Spec.HighAvailability.Enabled {
			grafana.Replicas = 2
		} else {
			grafana.Replicas = 1
		}
	}
	
	// Set default resources
	if grafana.Resources == nil {
		grafana.Resources = &ResourceRequirements{
			Requests: &ResourceList{
				CPU:    "100m",
				Memory: "256Mi",
			},
			Limits: &ResourceList{
				CPU:    "500m",
				Memory: "1Gi",
			},
		}
	}
	
	// Generate admin password if not provided
	if grafana.AdminPassword == "" {
		grafana.AdminPassword = generateSecurePassword(16)
		observabilityplatformlog.Info("Generated Grafana admin password", "name", r.Name)
	}
	
	// Set default admin user
	if grafana.AdminUser == "" {
		grafana.AdminUser = "admin"
	}
	
	// Set persistence defaults
	if grafana.Persistence == nil {
		grafana.Persistence = &PersistenceSpec{
			Enabled: true,
			Size:    "1Gi",
		}
	}
}

// defaultLoki sets defaults for Loki component
func (r *ObservabilityPlatform) defaultLoki() {
	if r.Spec.Components.Loki == nil {
		return
	}
	
	loki := r.Spec.Components.Loki
	
	// Set default version
	if loki.Version == "" {
		loki.Version = "2.9.0"
	}
	
	// Set default resources
	if loki.Resources == nil {
		loki.Resources = &ResourceRequirements{
			Requests: &ResourceList{
				CPU:    "100m",
				Memory: "256Mi",
			},
			Limits: &ResourceList{
				CPU:    "500m",
				Memory: "1Gi",
			},
		}
	}
	
	// Set default storage
	if loki.Storage == nil {
		loki.Storage = &LokiStorageSpec{
			Size: "10Gi",
		}
	} else if loki.Storage.Size == "" {
		loki.Storage.Size = "10Gi"
	}
	
	// Set default retention
	if loki.Retention == nil {
		loki.Retention = &RetentionSpec{
			Days:                   7,
			DeletesEnabled:         true,
			CompactionInterval:     "10m",
		}
	}
	
	// Set default replicas
	if loki.Replicas == 0 {
		loki.Replicas = 1
	}
}

// defaultTempo sets defaults for Tempo component
func (r *ObservabilityPlatform) defaultTempo() {
	if r.Spec.Components.Tempo == nil {
		return
	}
	
	tempo := r.Spec.Components.Tempo
	
	// Set default version
	if tempo.Version == "" {
		tempo.Version = "2.3.0"
	}
	
	// Set default resources
	if tempo.Resources == nil {
		tempo.Resources = &ResourceRequirements{
			Requests: &ResourceList{
				CPU:    "100m",
				Memory: "256Mi",
			},
			Limits: &ResourceList{
				CPU:    "500m",
				Memory: "1Gi",
			},
		}
	}
	
	// Set default storage
	if tempo.Storage == nil {
		tempo.Storage = &StorageSpec{
			Size: "5Gi",
		}
	} else if tempo.Storage.Size == "" {
		tempo.Storage.Size = "5Gi"
	}
	
	// Set default replicas
	if tempo.Replicas == 0 {
		tempo.Replicas = 1
	}
}

// defaultGlobalSettings sets global configuration defaults
func (r *ObservabilityPlatform) defaultGlobalSettings() {
	if r.Spec.Global == nil {
		r.Spec.Global = &GlobalSettings{}
	}
	
	// Set default log level
	if r.Spec.Global.LogLevel == "" {
		r.Spec.Global.LogLevel = "info"
	}
	
	// Set default external labels
	if r.Spec.Global.ExternalLabels == nil {
		r.Spec.Global.ExternalLabels = make(map[string]string)
	}
	
	// Add organization defaults
	if _, ok := r.Spec.Global.ExternalLabels["organization"]; !ok {
		r.Spec.Global.ExternalLabels["organization"] = "default"
	}
	if _, ok := r.Spec.Global.ExternalLabels["environment"]; !ok {
		// Try to infer environment from namespace
		switch r.Namespace {
		case "production", "prod":
			r.Spec.Global.ExternalLabels["environment"] = "production"
		case "staging", "stage":
			r.Spec.Global.ExternalLabels["environment"] = "staging"
		case "development", "dev":
			r.Spec.Global.ExternalLabels["environment"] = "development"
		default:
			r.Spec.Global.ExternalLabels["environment"] = "default"
		}
	}
	
	// Set default retention policies
	if r.Spec.Global.RetentionPolicies == nil {
		r.Spec.Global.RetentionPolicies = &RetentionPolicies{
			Metrics: "30d",
			Logs:    "7d",
			Traces:  "3d",
		}
	}
}

// defaultHighAvailability sets HA configuration defaults
func (r *ObservabilityPlatform) defaultHighAvailability() {
	if r.Spec.HighAvailability == nil {
		return
	}
	
	if !r.Spec.HighAvailability.Enabled {
		return
	}
	
	// Ensure minimum replicas for HA mode
	if r.Spec.Components.Prometheus != nil && r.Spec.Components.Prometheus.Enabled {
		if r.Spec.Components.Prometheus.Replicas < 3 {
			r.Spec.Components.Prometheus.Replicas = 3
		}
	}
	
	if r.Spec.Components.Grafana != nil && r.Spec.Components.Grafana.Enabled {
		if r.Spec.Components.Grafana.Replicas < 2 {
			r.Spec.Components.Grafana.Replicas = 2
		}
	}
	
	// Set anti-affinity rules
	if r.Spec.HighAvailability.AntiAffinity == nil {
		r.Spec.HighAvailability.AntiAffinity = &AntiAffinitySpec{
			Type: "soft",
		}
	}
}

// defaultBackup sets backup configuration defaults
func (r *ObservabilityPlatform) defaultBackup() {
	if r.Spec.Backup == nil {
		return
	}
	
	if !r.Spec.Backup.Enabled {
		return
	}
	
	// Set default schedule (daily at 2 AM)
	if r.Spec.Backup.Schedule == "" {
		r.Spec.Backup.Schedule = "0 2 * * *"
	}
	
	// Set default retention (7 days)
	if r.Spec.Backup.Retention == 0 {
		r.Spec.Backup.Retention = 7
	}
	
	// Set default storage type
	if r.Spec.Backup.StorageType == "" {
		r.Spec.Backup.StorageType = "s3"
	}
	
	// Set default storage location
	if r.Spec.Backup.StorageLocation == "" {
		r.Spec.Backup.StorageLocation = fmt.Sprintf("observability-backups/%s/%s", r.Namespace, r.Name)
	}
}

// defaultAlerting sets alerting configuration defaults
func (r *ObservabilityPlatform) defaultAlerting() {
	if r.Spec.Alerting == nil {
		return
	}
	
	if r.Spec.Alerting.Alertmanager == nil {
		return
	}
	
	am := r.Spec.Alerting.Alertmanager
	
	// Set default version
	if am.Version == "" {
		am.Version = "v0.26.0"
	}
	
	// Set default replicas
	if am.Replicas == 0 {
		if r.Spec.HighAvailability != nil && r.Spec.HighAvailability.Enabled {
			am.Replicas = 3
		} else {
			am.Replicas = 1
		}
	}
	
	// Set default resources
	if am.Resources == nil {
		am.Resources = &ResourceRequirements{
			Requests: &ResourceList{
				CPU:    "10m",
				Memory: "32Mi",
			},
			Limits: &ResourceList{
				CPU:    "100m",
				Memory: "128Mi",
			},
		}
	}
	
	// Set default storage
	if am.Storage == nil {
		am.Storage = &StorageSpec{
			Size: "1Gi",
		}
	}
	
	// Set default configuration if not provided
	if am.Config == nil {
		am.Config = &AlertmanagerConfig{
			Route: &Route{
				GroupBy:        []string{"alertname", "cluster", "service"},
				GroupWait:      "10s",
				GroupInterval:  "10s",
				RepeatInterval: "12h",
				Receiver:       "default-receiver",
			},
			Receivers: []Receiver{
				{
					Name: "default-receiver",
				},
			},
		}
	}
}

// generateSecurePassword generates a cryptographically secure password
func generateSecurePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	bytes := make([]byte, length)
	
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple default if random generation fails
		return "ChangeMe123!"
	}
	
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	
	return string(bytes)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ObservabilityPlatform) ValidateCreate() (admission.Warnings, error) {
	observabilityplatformlog.Info("validate create", "name", r.Name)
	
	var allErrs field.ErrorList
	var warnings admission.Warnings
	ctx := context.Background()
	
	// Use configuration validator for comprehensive validation
	if globalConfigValidator != nil {
		if err := globalConfigValidator.ValidateConfiguration(ctx, r); err != nil {
			allErrs = append(allErrs, err...)
		}
	} else {
		// Fallback to basic validation if configuration validator is not available
		if err := r.validateComponents(ctx); err != nil {
			allErrs = append(allErrs, err...)
		}
	}
	
	// Validate global settings
	if err := r.validateGlobalSettings(ctx); err != nil {
		allErrs = append(allErrs, err...)
	}
	
	// Validate high availability settings
	if err := r.validateHighAvailability(ctx); err != nil {
		allErrs = append(allErrs, err...)
	}
	
	// Validate backup settings
	if err := r.validateBackupSettings(ctx); err != nil {
		allErrs = append(allErrs, err...)
	}
	
	// Validate resource quotas
	if globalQuotaValidator != nil {
		if err := globalQuotaValidator.ValidateResourceQuota(ctx, r); err != nil {
			allErrs = append(allErrs, err...)
			
			// Add a warning with quota summary
			if summary, summaryErr := globalQuotaValidator.GetQuotaSummary(ctx, r.Namespace); summaryErr == nil {
				warnings = append(warnings, fmt.Sprintf("Resource quota validation failed. Current quota status:\n%s", summary))
			}
		}
	} else {
		observabilityplatformlog.V(1).Info("quota validator not initialized, skipping quota validation")
	}
	
	if len(allErrs) == 0 {
		return warnings, nil
	}
	
	return warnings, errors.NewInvalid(
		schema.GroupKind{Group: GroupVersion.Group, Kind: "ObservabilityPlatform"},
		r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ObservabilityPlatform) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	observabilityplatformlog.Info("validate update", "name", r.Name)
	
	oldObj, ok := old.(*ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform object")
	}
	
	var allErrs field.ErrorList
	var warnings admission.Warnings
	ctx := context.Background()
	
	// Validate immutable fields
	if err := r.validateImmutableFields(ctx, oldObj); err != nil {
		allErrs = append(allErrs, err...)
	}
	
	// Validate component version downgrades
	if warn := r.validateVersionChanges(ctx, oldObj); len(warn) > 0 {
		warnings = append(warnings, warn...)
	}
	
	// Validate resource quotas for scaled resources
	if globalQuotaValidator != nil {
		// Check if resource requirements have increased
		if r.hasIncreasedResources(oldObj) {
			if err := globalQuotaValidator.ValidateResourceQuota(ctx, r); err != nil {
				allErrs = append(allErrs, err...)
				
				// Add a warning with quota summary
				if summary, summaryErr := globalQuotaValidator.GetQuotaSummary(ctx, r.Namespace); summaryErr == nil {
					warnings = append(warnings, fmt.Sprintf("Resource quota validation failed during update. Current quota status:\n%s", summary))
				}
			}
		}
	}
	
	// Run create validations
	if createWarnings, err := r.ValidateCreate(); err != nil {
		if invalidErr, ok := err.(*errors.StatusError); ok {
			allErrs = append(allErrs, invalidErr.ErrStatus.Details.Causes...)
		}
	} else if len(createWarnings) > 0 {
		warnings = append(warnings, createWarnings...)
	}
	
	if len(allErrs) == 0 {
		return warnings, nil
	}
	
	return warnings, errors.NewInvalid(
		schema.GroupKind{Group: GroupVersion.Group, Kind: "ObservabilityPlatform"},
		r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ObservabilityPlatform) ValidateDelete() (admission.Warnings, error) {
	observabilityplatformlog.Info("validate delete", "name", r.Name)
	
	// Add any deletion validation logic here
	// For example, checking if there are dependent resources
	
	return nil, nil
}

// validateComponents validates the component configurations
func (r *ObservabilityPlatform) validateComponents(ctx context.Context) field.ErrorList {
	var allErrs field.ErrorList
	componentsPath := field.NewPath("spec").Child("components")
	
	// At least one component must be enabled
	if !r.hasEnabledComponent() {
		allErrs = append(allErrs, field.Required(componentsPath, "at least one component must be enabled"))
	}
	
	// Validate Prometheus
	if r.Spec.Components.Prometheus != nil && r.Spec.Components.Prometheus.Enabled {
		allErrs = append(allErrs, r.validatePrometheus(componentsPath.Child("prometheus"))...)
	}
	
	// Validate Grafana
	if r.Spec.Components.Grafana != nil && r.Spec.Components.Grafana.Enabled {
		allErrs = append(allErrs, r.validateGrafana(componentsPath.Child("grafana"))...)
	}
	
	// Validate Loki
	if r.Spec.Components.Loki != nil && r.Spec.Components.Loki.Enabled {
		allErrs = append(allErrs, r.validateLoki(componentsPath.Child("loki"))...)
	}
	
	// Validate Tempo
	if r.Spec.Components.Tempo != nil && r.Spec.Components.Tempo.Enabled {
		allErrs = append(allErrs, r.validateTempo(componentsPath.Child("tempo"))...)
	}
	
	return allErrs
}

// validatePrometheus validates Prometheus configuration
func (r *ObservabilityPlatform) validatePrometheus(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	prom := r.Spec.Components.Prometheus
	
	// Validate version format
	if prom.Version != "" {
		if !isValidVersion(prom.Version) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), prom.Version, "invalid version format, expected vX.Y.Z"))
		}
	}
	
	// Validate replicas
	if prom.Replicas < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("replicas"), prom.Replicas, "must be at least 1"))
	}
	
	// Validate resources
	if prom.Resources != nil {
		allErrs = append(allErrs, r.validateResourceRequirements(fldPath.Child("resources"), prom.Resources)...)
	}
	
	// Validate storage
	if prom.Storage != nil {
		if prom.Storage.Size != "" {
			if _, err := resource.ParseQuantity(prom.Storage.Size); err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("storage").Child("size"), prom.Storage.Size, "invalid quantity"))
			}
		}
		
		// Validate retention
		if prom.Storage.Retention != "" {
			if !isValidRetention(prom.Storage.Retention) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("storage").Child("retention"), prom.Storage.Retention, "invalid retention format, expected format like 30d, 12h"))
			}
		}
	}
	
	// Validate external labels
	if prom.ExternalLabels != nil {
		for k, v := range prom.ExternalLabels {
			if !isValidLabelName(k) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("externalLabels").Key(k), k, "invalid label name"))
			}
			if !isValidLabelValue(v) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("externalLabels").Key(k), v, "invalid label value"))
			}
		}
	}
	
	return allErrs
}

// validateGrafana validates Grafana configuration
func (r *ObservabilityPlatform) validateGrafana(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	grafana := r.Spec.Components.Grafana
	
	// Validate version
	if grafana.Version != "" {
		if !isValidVersion(grafana.Version) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), grafana.Version, "invalid version format"))
		}
	}
	
	// Validate replicas
	if grafana.Replicas < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("replicas"), grafana.Replicas, "must be at least 1"))
	}
	
	// Validate resources
	if grafana.Resources != nil {
		allErrs = append(allErrs, r.validateResourceRequirements(fldPath.Child("resources"), grafana.Resources)...)
	}
	
	// Validate ingress
	if grafana.Ingress != nil && grafana.Ingress.Enabled {
		if grafana.Ingress.Host == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("ingress").Child("host"), "host is required when ingress is enabled"))
		}
	}
	
	return allErrs
}

// validateLoki validates Loki configuration
func (r *ObservabilityPlatform) validateLoki(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	loki := r.Spec.Components.Loki
	
	// Validate version
	if loki.Version != "" {
		if !isValidVersion(loki.Version) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), loki.Version, "invalid version format"))
		}
	}
	
	// Validate storage
	if loki.Storage != nil {
		if loki.Storage.Size != "" {
			if _, err := resource.ParseQuantity(loki.Storage.Size); err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("storage").Child("size"), loki.Storage.Size, "invalid quantity"))
			}
		}
		
		// If S3 is enabled, validate S3 configuration
		if loki.Storage.S3 != nil && loki.Storage.S3.Enabled {
			if loki.Storage.S3.BucketName == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("storage").Child("s3").Child("bucketName"), "bucket name is required when S3 is enabled"))
			}
			if loki.Storage.S3.Region == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("storage").Child("s3").Child("region"), "region is required when S3 is enabled"))
			}
		}
	}
	
	return allErrs
}

// validateTempo validates Tempo configuration
func (r *ObservabilityPlatform) validateTempo(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	tempo := r.Spec.Components.Tempo
	
	// Validate version
	if tempo.Version != "" {
		if !isValidVersion(tempo.Version) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), tempo.Version, "invalid version format"))
		}
	}
	
	// Validate storage
	if tempo.Storage != nil && tempo.Storage.Size != "" {
		if _, err := resource.ParseQuantity(tempo.Storage.Size); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("storage").Child("size"), tempo.Storage.Size, "invalid quantity"))
		}
	}
	
	return allErrs
}

// validateGlobalSettings validates global configuration
func (r *ObservabilityPlatform) validateGlobalSettings(ctx context.Context) field.ErrorList {
	var allErrs field.ErrorList
	
	if r.Spec.Global == nil {
		return allErrs
	}
	
	globalPath := field.NewPath("spec").Child("global")
	
	// Validate external labels
	if r.Spec.Global.ExternalLabels != nil {
		for k, v := range r.Spec.Global.ExternalLabels {
			if !isValidLabelName(k) {
				allErrs = append(allErrs, field.Invalid(globalPath.Child("externalLabels").Key(k), k, "invalid label name"))
			}
			if !isValidLabelValue(v) {
				allErrs = append(allErrs, field.Invalid(globalPath.Child("externalLabels").Key(k), v, "invalid label value"))
			}
		}
	}
	
	// Validate log level
	if r.Spec.Global.LogLevel != "" {
		validLevels := []string{"debug", "info", "warn", "error"}
		if !contains(validLevels, r.Spec.Global.LogLevel) {
			allErrs = append(allErrs, field.NotSupported(globalPath.Child("logLevel"), r.Spec.Global.LogLevel, validLevels))
		}
	}
	
	return allErrs
}

// validateHighAvailability validates HA settings
func (r *ObservabilityPlatform) validateHighAvailability(ctx context.Context) field.ErrorList {
	var allErrs field.ErrorList
	
	if r.Spec.HighAvailability == nil || !r.Spec.HighAvailability.Enabled {
		return allErrs
	}
	
	haPath := field.NewPath("spec").Child("highAvailability")
	
	// When HA is enabled, certain components must have appropriate replica counts
	if r.Spec.Components.Prometheus != nil && r.Spec.Components.Prometheus.Enabled {
		if r.Spec.Components.Prometheus.Replicas < 2 {
			allErrs = append(allErrs, field.Invalid(haPath, r.Spec.Components.Prometheus.Replicas, "Prometheus must have at least 2 replicas when HA is enabled"))
		}
	}
	
	if r.Spec.Components.Grafana != nil && r.Spec.Components.Grafana.Enabled {
		if r.Spec.Components.Grafana.Replicas < 2 {
			allErrs = append(allErrs, field.Invalid(haPath, r.Spec.Components.Grafana.Replicas, "Grafana must have at least 2 replicas when HA is enabled"))
		}
	}
	
	return allErrs
}

// validateBackupSettings validates backup configuration
func (r *ObservabilityPlatform) validateBackupSettings(ctx context.Context) field.ErrorList {
	var allErrs field.ErrorList
	
	if r.Spec.Backup == nil || !r.Spec.Backup.Enabled {
		return allErrs
	}
	
	backupPath := field.NewPath("spec").Child("backup")
	
	// Validate schedule
	if r.Spec.Backup.Schedule == "" {
		allErrs = append(allErrs, field.Required(backupPath.Child("schedule"), "schedule is required when backup is enabled"))
	} else {
		// Validate cron expression
		if !isValidCronExpression(r.Spec.Backup.Schedule) {
			allErrs = append(allErrs, field.Invalid(backupPath.Child("schedule"), r.Spec.Backup.Schedule, "invalid cron expression"))
		}
	}
	
	// Validate retention
	if r.Spec.Backup.Retention < 1 {
		allErrs = append(allErrs, field.Invalid(backupPath.Child("retention"), r.Spec.Backup.Retention, "retention must be at least 1 day"))
	}
	
	// Validate storage backend
	if r.Spec.Backup.StorageType == "" {
		allErrs = append(allErrs, field.Required(backupPath.Child("storageType"), "storage type is required"))
	} else {
		validTypes := []string{"s3", "azure", "gcs", "local"}
		if !contains(validTypes, r.Spec.Backup.StorageType) {
			allErrs = append(allErrs, field.NotSupported(backupPath.Child("storageType"), r.Spec.Backup.StorageType, validTypes))
		}
	}
	
	return allErrs
}

// validateResourceRequirements validates resource requests and limits
func (r *ObservabilityPlatform) validateResourceRequirements(fldPath *field.Path, resources *ResourceRequirements) field.ErrorList {
	var allErrs field.ErrorList
	
	// Validate CPU
	if resources.Requests != nil && resources.Requests.CPU != "" {
		if _, err := resource.ParseQuantity(resources.Requests.CPU); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("requests").Child("cpu"), resources.Requests.CPU, "invalid CPU quantity"))
		}
	}
	
	if resources.Limits != nil && resources.Limits.CPU != "" {
		if _, err := resource.ParseQuantity(resources.Limits.CPU); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("limits").Child("cpu"), resources.Limits.CPU, "invalid CPU quantity"))
		}
	}
	
	// Validate Memory
	if resources.Requests != nil && resources.Requests.Memory != "" {
		if _, err := resource.ParseQuantity(resources.Requests.Memory); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("requests").Child("memory"), resources.Requests.Memory, "invalid memory quantity"))
		}
	}
	
	if resources.Limits != nil && resources.Limits.Memory != "" {
		if _, err := resource.ParseQuantity(resources.Limits.Memory); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("limits").Child("memory"), resources.Limits.Memory, "invalid memory quantity"))
		}
	}
	
	// Ensure requests don't exceed limits
	if resources.Requests != nil && resources.Limits != nil {
		if resources.Requests.CPU != "" && resources.Limits.CPU != "" {
			reqCPU, _ := resource.ParseQuantity(resources.Requests.CPU)
			limCPU, _ := resource.ParseQuantity(resources.Limits.CPU)
			if reqCPU.Cmp(limCPU) > 0 {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("requests").Child("cpu"), resources.Requests.CPU, "CPU request cannot exceed limit"))
			}
		}
		
		if resources.Requests.Memory != "" && resources.Limits.Memory != "" {
			reqMem, _ := resource.ParseQuantity(resources.Requests.Memory)
			limMem, _ := resource.ParseQuantity(resources.Limits.Memory)
			if reqMem.Cmp(limMem) > 0 {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("requests").Child("memory"), resources.Requests.Memory, "memory request cannot exceed limit"))
			}
		}
	}
	
	return allErrs
}

// validateImmutableFields checks that immutable fields haven't changed
func (r *ObservabilityPlatform) validateImmutableFields(ctx context.Context, old *ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList
	
	// Add validation for fields that shouldn't change after creation
	// For example, storage class shouldn't change after PVCs are created
	
	return allErrs
}

// validateVersionChanges checks for valid version transitions
func (r *ObservabilityPlatform) validateVersionChanges(ctx context.Context, old *ObservabilityPlatform) admission.Warnings {
	var warnings admission.Warnings
	
	// Check for version downgrades
	if old.Spec.Components.Prometheus != nil && r.Spec.Components.Prometheus != nil {
		if isVersionDowngrade(old.Spec.Components.Prometheus.Version, r.Spec.Components.Prometheus.Version) {
			warnings = append(warnings, fmt.Sprintf("Prometheus version downgrade detected: %s -> %s", old.Spec.Components.Prometheus.Version, r.Spec.Components.Prometheus.Version))
		}
	}
	
	if old.Spec.Components.Grafana != nil && r.Spec.Components.Grafana != nil {
		if isVersionDowngrade(old.Spec.Components.Grafana.Version, r.Spec.Components.Grafana.Version) {
			warnings = append(warnings, fmt.Sprintf("Grafana version downgrade detected: %s -> %s", old.Spec.Components.Grafana.Version, r.Spec.Components.Grafana.Version))
		}
	}
	
	return warnings
}

// Helper functions

func (r *ObservabilityPlatform) hasEnabledComponent() bool {
	return (r.Spec.Components.Prometheus != nil && r.Spec.Components.Prometheus.Enabled) ||
		(r.Spec.Components.Grafana != nil && r.Spec.Components.Grafana.Enabled) ||
		(r.Spec.Components.Loki != nil && r.Spec.Components.Loki.Enabled) ||
		(r.Spec.Components.Tempo != nil && r.Spec.Components.Tempo.Enabled)
}

func isValidVersion(version string) bool {
	// Match versions like v2.45.0 or 2.45.0
	versionRegex := regexp.MustCompile(`^v?\d+\.\d+\.\d+(-.*)?$`)
	return versionRegex.MatchString(version)
}

func isValidRetention(retention string) bool {
	// Match retention formats like 30d, 12h, 1w
	retentionRegex := regexp.MustCompile(`^\d+[hdwmy]$`)
	return retentionRegex.MatchString(retention)
}

func isValidLabelName(name string) bool {
	// Kubernetes label name validation
	labelRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	return len(name) <= 63 && labelRegex.MatchString(name)
}

func isValidLabelValue(value string) bool {
	// Kubernetes label value validation
	if value == "" {
		return true
	}
	labelValueRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	return len(value) <= 63 && labelValueRegex.MatchString(value)
}

func isValidCronExpression(cron string) bool {
	// Simple cron validation - can be enhanced
	parts := strings.Fields(cron)
	return len(parts) == 5 || len(parts) == 6
}

func isVersionDowngrade(oldVersion, newVersion string) bool {
	// Simple version comparison - can be enhanced with semver library
	oldVersion = strings.TrimPrefix(oldVersion, "v")
	newVersion = strings.TrimPrefix(newVersion, "v")
	
	return oldVersion > newVersion
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// hasIncreasedResources checks if the new platform has increased resource requirements
func (r *ObservabilityPlatform) hasIncreasedResources(old *ObservabilityPlatform) bool {
	// Check Prometheus
	if r.Spec.Components.Prometheus != nil && old.Spec.Components.Prometheus != nil {
		if r.Spec.Components.Prometheus.Replicas > old.Spec.Components.Prometheus.Replicas {
			return true
		}
		if hasIncreasedResourceRequirements(r.Spec.Components.Prometheus.Resources, old.Spec.Components.Prometheus.Resources) {
			return true
		}
		if hasIncreasedStorage(r.Spec.Components.Prometheus.Storage, old.Spec.Components.Prometheus.Storage) {
			return true
		}
	}
	
	// Check Grafana
	if r.Spec.Components.Grafana != nil && old.Spec.Components.Grafana != nil {
		if r.Spec.Components.Grafana.Replicas > old.Spec.Components.Grafana.Replicas {
			return true
		}
		if hasIncreasedResourceRequirements(r.Spec.Components.Grafana.Resources, old.Spec.Components.Grafana.Resources) {
			return true
		}
	}
	
	// Check Loki
	if r.Spec.Components.Loki != nil && old.Spec.Components.Loki != nil {
		if r.Spec.Components.Loki.Replicas > old.Spec.Components.Loki.Replicas {
			return true
		}
		if hasIncreasedResourceRequirements(r.Spec.Components.Loki.Resources, old.Spec.Components.Loki.Resources) {
			return true
		}
	}
	
	// Check Tempo
	if r.Spec.Components.Tempo != nil && old.Spec.Components.Tempo != nil {
		if r.Spec.Components.Tempo.Replicas > old.Spec.Components.Tempo.Replicas {
			return true
		}
		if hasIncreasedResourceRequirements(r.Spec.Components.Tempo.Resources, old.Spec.Components.Tempo.Resources) {
			return true
		}
	}
	
	// Check if new components are being added
	if r.Spec.Components.Prometheus != nil && r.Spec.Components.Prometheus.Enabled && 
		(old.Spec.Components.Prometheus == nil || !old.Spec.Components.Prometheus.Enabled) {
		return true
	}
	if r.Spec.Components.Grafana != nil && r.Spec.Components.Grafana.Enabled && 
		(old.Spec.Components.Grafana == nil || !old.Spec.Components.Grafana.Enabled) {
		return true
	}
	if r.Spec.Components.Loki != nil && r.Spec.Components.Loki.Enabled && 
		(old.Spec.Components.Loki == nil || !old.Spec.Components.Loki.Enabled) {
		return true
	}
	if r.Spec.Components.Tempo != nil && r.Spec.Components.Tempo.Enabled && 
		(old.Spec.Components.Tempo == nil || !old.Spec.Components.Tempo.Enabled) {
		return true
	}
	
	return false
}

// hasIncreasedResourceRequirements checks if resource requirements have increased
func hasIncreasedResourceRequirements(new, old *ResourceRequirements) bool {
	if new == nil || old == nil {
		return new != nil // Only increased if new is not nil and old was nil
	}
	
	// Check CPU requests
	if new.Requests != nil && old.Requests != nil {
		if new.Requests.CPU != "" && old.Requests.CPU != "" {
			newCPU, _ := resource.ParseQuantity(new.Requests.CPU)
			oldCPU, _ := resource.ParseQuantity(old.Requests.CPU)
			if newCPU.Cmp(oldCPU) > 0 {
				return true
			}
		}
		
		// Check Memory requests
		if new.Requests.Memory != "" && old.Requests.Memory != "" {
			newMem, _ := resource.ParseQuantity(new.Requests.Memory)
			oldMem, _ := resource.ParseQuantity(old.Requests.Memory)
			if newMem.Cmp(oldMem) > 0 {
				return true
			}
		}
	}
	
	return false
}

// hasIncreasedStorage checks if storage requirements have increased
func hasIncreasedStorage(new, old *StorageSpec) bool {
	if new == nil || old == nil {
		return new != nil // Only increased if new is not nil and old was nil
	}
	
	if new.Size != "" && old.Size != "" {
		newSize, _ := resource.ParseQuantity(new.Size)
		oldSize, _ := resource.ParseQuantity(old.Size)
		if newSize.Cmp(oldSize) > 0 {
			return true
		}
	}
	
	return false
}
