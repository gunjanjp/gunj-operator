/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/webhook/quota"
	"github.com/gunjanjp/gunj-operator/internal/webhook/security"
)

// log is for logging in this package.
var observabilityplatformlog = logf.Log.WithName("observabilityplatform-resource")

// +kubebuilder:webhook:path=/mutate-observability-io-v1beta1-observabilityplatform,mutating=true,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1beta1,name=mobservabilityplatform.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ObservabilityPlatformWebhook{}

// ObservabilityPlatformWebhook implements webhook for ObservabilityPlatform
type ObservabilityPlatformWebhook struct {
	client.Client
	QuotaValidator    *quota.QuotaValidator
	SecurityValidator *security.SecurityValidator
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ObservabilityPlatformWebhook) Default(ctx context.Context, obj runtime.Object) error {
	platform, ok := obj.(*observabilityv1beta1.ObservabilityPlatform)
	if !ok {
		return fmt.Errorf("expected ObservabilityPlatform but got %T", obj)
	}

	observabilityplatformlog.Info("default", "name", platform.Name)

	// Set default values for Prometheus
	if platform.Spec.Components.Prometheus != nil {
		if platform.Spec.Components.Prometheus.Replicas == 0 {
			platform.Spec.Components.Prometheus.Replicas = 1
		}
		if platform.Spec.Components.Prometheus.Retention == "" {
			platform.Spec.Components.Prometheus.Retention = "30d"
		}
		if platform.Spec.Components.Prometheus.Storage.Size.IsZero() {
			platform.Spec.Components.Prometheus.Storage.Size = resource.MustParse("10Gi")
		}
		// Set default resources if not specified
		if platform.Spec.Components.Prometheus.Resources.Requests == nil {
			platform.Spec.Components.Prometheus.Resources.Requests = map[string]resource.Quantity{
				"memory": resource.MustParse("2Gi"),
				"cpu":    resource.MustParse("500m"),
			}
		}
		if platform.Spec.Components.Prometheus.Resources.Limits == nil {
			platform.Spec.Components.Prometheus.Resources.Limits = map[string]resource.Quantity{
				"memory": resource.MustParse("4Gi"),
				"cpu":    resource.MustParse("1"),
			}
		}
	}

	// Set default values for Grafana
	if platform.Spec.Components.Grafana != nil {
		if platform.Spec.Components.Grafana.Replicas == 0 {
			platform.Spec.Components.Grafana.Replicas = 1
		}
		// Generate admin password if not provided
		if platform.Spec.Components.Grafana.AdminPassword == "" {
			platform.Spec.Components.Grafana.AdminPassword = generateSecurePassword()
		}
		// Set default resources if not specified
		if platform.Spec.Components.Grafana.Resources.Requests == nil {
			platform.Spec.Components.Grafana.Resources.Requests = map[string]resource.Quantity{
				"memory": resource.MustParse("512Mi"),
				"cpu":    resource.MustParse("100m"),
			}
		}
		if platform.Spec.Components.Grafana.Resources.Limits == nil {
			platform.Spec.Components.Grafana.Resources.Limits = map[string]resource.Quantity{
				"memory": resource.MustParse("1Gi"),
				"cpu":    resource.MustParse("500m"),
			}
		}
	}

	// Set default values for Loki
	if platform.Spec.Components.Loki != nil {
		if platform.Spec.Components.Loki.Replicas == 0 {
			platform.Spec.Components.Loki.Replicas = 1
		}
		if platform.Spec.Components.Loki.Retention == "" {
			platform.Spec.Components.Loki.Retention = "168h" // 7 days
		}
		if platform.Spec.Components.Loki.Storage.Size.IsZero() {
			platform.Spec.Components.Loki.Storage.Size = resource.MustParse("20Gi")
		}
		// Set default resources if not specified
		if platform.Spec.Components.Loki.Resources.Requests == nil {
			platform.Spec.Components.Loki.Resources.Requests = map[string]resource.Quantity{
				"memory": resource.MustParse("1Gi"),
				"cpu":    resource.MustParse("250m"),
			}
		}
		if platform.Spec.Components.Loki.Resources.Limits == nil {
			platform.Spec.Components.Loki.Resources.Limits = map[string]resource.Quantity{
				"memory": resource.MustParse("2Gi"),
				"cpu":    resource.MustParse("500m"),
			}
		}
	}

	// Set default values for Tempo
	if platform.Spec.Components.Tempo != nil {
		if platform.Spec.Components.Tempo.Replicas == 0 {
			platform.Spec.Components.Tempo.Replicas = 1
		}
		if platform.Spec.Components.Tempo.Retention == "" {
			platform.Spec.Components.Tempo.Retention = "336h" // 14 days
		}
		if platform.Spec.Components.Tempo.Storage.Size.IsZero() {
			platform.Spec.Components.Tempo.Storage.Size = resource.MustParse("10Gi")
		}
		// Set default resources if not specified
		if platform.Spec.Components.Tempo.Resources.Requests == nil {
			platform.Spec.Components.Tempo.Resources.Requests = map[string]resource.Quantity{
				"memory": resource.MustParse("1Gi"),
				"cpu":    resource.MustParse("250m"),
			}
		}
		if platform.Spec.Components.Tempo.Resources.Limits == nil {
			platform.Spec.Components.Tempo.Resources.Limits = map[string]resource.Quantity{
				"memory": resource.MustParse("2Gi"),
				"cpu":    resource.MustParse("500m"),
			}
		}
	}

	// Set default values for OpenTelemetry Collector
	if platform.Spec.Components.OpenTelemetryCollector != nil {
		if platform.Spec.Components.OpenTelemetryCollector.Replicas == 0 {
			platform.Spec.Components.OpenTelemetryCollector.Replicas = 2
		}
		// Set default resources if not specified
		if platform.Spec.Components.OpenTelemetryCollector.Resources.Requests == nil {
			platform.Spec.Components.OpenTelemetryCollector.Resources.Requests = map[string]resource.Quantity{
				"memory": resource.MustParse("512Mi"),
				"cpu":    resource.MustParse("100m"),
			}
		}
		if platform.Spec.Components.OpenTelemetryCollector.Resources.Limits == nil {
			platform.Spec.Components.OpenTelemetryCollector.Resources.Limits = map[string]resource.Quantity{
				"memory": resource.MustParse("1Gi"),
				"cpu":    resource.MustParse("500m"),
			}
		}
	}

	// Set default global configuration
	if platform.Spec.Global.LogLevel == "" {
		platform.Spec.Global.LogLevel = "info"
	}

	// Set default HA configuration
	if platform.Spec.HighAvailability != nil && platform.Spec.HighAvailability.Enabled {
		if platform.Spec.HighAvailability.MinReplicas < 3 {
			platform.Spec.HighAvailability.MinReplicas = 3
		}
		// Ensure components have appropriate replicas for HA
		if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Replicas < 3 {
			platform.Spec.Components.Prometheus.Replicas = 3
		}
		if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Replicas < 2 {
			platform.Spec.Components.Grafana.Replicas = 2
		}
		if platform.Spec.Alerting.Alertmanager != nil && platform.Spec.Alerting.Alertmanager.Replicas < 3 {
			platform.Spec.Alerting.Alertmanager.Replicas = 3
		}
	}

	// Set default backup configuration
	if platform.Spec.Backup != nil && platform.Spec.Backup.Enabled {
		if platform.Spec.Backup.Schedule == "" {
			platform.Spec.Backup.Schedule = "0 2 * * *" // Daily at 2 AM
		}
		if platform.Spec.Backup.RetentionDays == 0 {
			platform.Spec.Backup.RetentionDays = 7
		}
	}

	// Set default security configuration
	if platform.Spec.Security != nil {
		if platform.Spec.Security.TLS.Enabled && platform.Spec.Security.TLS.AutoTLS {
			// Mark for auto-TLS generation
			if platform.Annotations == nil {
				platform.Annotations = make(map[string]string)
			}
			platform.Annotations["observability.io/auto-tls"] = "true"
		}
	}

	return nil
}

// +kubebuilder:webhook:path=/validate-observability-io-v1beta1-observabilityplatform,mutating=false,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1beta1,name=vobservabilityplatform.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ObservabilityPlatformWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ObservabilityPlatformWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	platform, ok := obj.(*observabilityv1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform but got %T", obj)
	}

	observabilityplatformlog.Info("validate create", "name", platform.Name)

	var allErrs field.ErrorList

	// Validate component configuration
	if err := r.validateComponents(platform, &allErrs); err != nil {
		return nil, err
	}

	// Validate resource requirements
	if err := r.validateResources(platform, &allErrs); err != nil {
		return nil, err
	}

	// Validate storage configuration
	if err := r.validateStorage(platform, &allErrs); err != nil {
		return nil, err
	}

	// Validate networking configuration
	if err := r.validateNetworking(platform, &allErrs); err != nil {
		return nil, err
	}

	// Validate security configuration
	if err := r.validateSecurity(platform, &allErrs); err != nil {
		return nil, err
	}

	// Validate backup configuration
	if err := r.validateBackup(platform, &allErrs); err != nil {
		return nil, err
	}

	// Validate high availability configuration
	if err := r.validateHighAvailability(platform, &allErrs); err != nil {
		return nil, err
	}

	// Validate resource quotas if quota validator is configured
	if r.QuotaValidator != nil {
		quotaErrs := r.QuotaValidator.ValidateQuotas(ctx, platform)
		allErrs = append(allErrs, quotaErrs...)
	}

	// Validate security policies if security validator is configured
	if r.SecurityValidator != nil {
		securityErrs := r.SecurityValidator.ValidateSecurity(ctx, platform)
		allErrs = append(allErrs, securityErrs...)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "observability.io", Kind: "ObservabilityPlatform"},
			platform.Name,
			allErrs,
		)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ObservabilityPlatformWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldPlatform, ok := oldObj.(*observabilityv1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform but got %T", oldObj)
	}

	newPlatform, ok := newObj.(*observabilityv1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform but got %T", newObj)
	}

	observabilityplatformlog.Info("validate update", "name", newPlatform.Name)

	var allErrs field.ErrorList
	var warnings admission.Warnings

	// First, run all create validations on the new object
	if _, err := r.ValidateCreate(ctx, newObj); err != nil {
		return warnings, err
	}

	// Validate immutable fields
	if err := r.validateImmutableFields(oldPlatform, newPlatform, &allErrs); err != nil {
		return warnings, err
	}

	// Validate version upgrades
	if err := r.validateVersionUpgrades(oldPlatform, newPlatform, &allErrs, &warnings); err != nil {
		return warnings, err
	}

	// Validate scaling changes
	if err := r.validateScalingChanges(oldPlatform, newPlatform, &allErrs, &warnings); err != nil {
		return warnings, err
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{Group: "observability.io", Kind: "ObservabilityPlatform"},
			newPlatform.Name,
			allErrs,
		)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ObservabilityPlatformWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	platform, ok := obj.(*observabilityv1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform but got %T", obj)
	}

	observabilityplatformlog.Info("validate delete", "name", platform.Name)

	var warnings admission.Warnings

	// Check if platform has backup enabled and warn about data loss
	if platform.Spec.Backup != nil && platform.Spec.Backup.Enabled {
		warnings = append(warnings, fmt.Sprintf(
			"Platform %s has backups enabled. Ensure you have exported necessary data before deletion.",
			platform.Name,
		))
	}

	// Check if platform has persistent storage and warn about data loss
	hasStorage := false
	if platform.Spec.Components.Prometheus != nil && !platform.Spec.Components.Prometheus.Storage.Size.IsZero() {
		hasStorage = true
	}
	if platform.Spec.Components.Loki != nil && !platform.Spec.Components.Loki.Storage.Size.IsZero() {
		hasStorage = true
	}
	if platform.Spec.Components.Tempo != nil && !platform.Spec.Components.Tempo.Storage.Size.IsZero() {
		hasStorage = true
	}

	if hasStorage {
		warnings = append(warnings, fmt.Sprintf(
			"Platform %s has persistent storage configured. All data will be permanently deleted.",
			platform.Name,
		))
	}

	// Check for production-critical labels/annotations
	if platform.Labels != nil {
		if env, ok := platform.Labels["environment"]; ok && env == "production" {
			warnings = append(warnings, fmt.Sprintf(
				"WARNING: You are deleting a PRODUCTION platform %s. This action cannot be undone.",
				platform.Name,
			))
		}
	}

	return warnings, nil
}

// Validation helper methods

func (r *ObservabilityPlatformWebhook) validateComponents(platform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList) error {
	componentsPath := field.NewPath("spec", "components")

	// Ensure at least one component is enabled
	hasEnabledComponent := false
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		hasEnabledComponent = true
	}
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		hasEnabledComponent = true
	}
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		hasEnabledComponent = true
	}
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		hasEnabledComponent = true
	}

	if !hasEnabledComponent {
		*allErrs = append(*allErrs, field.Invalid(
			componentsPath,
			platform.Spec.Components,
			"at least one component must be enabled",
		))
	}

	// Validate Prometheus configuration
	if platform.Spec.Components.Prometheus != nil {
		prometheusPath := componentsPath.Child("prometheus")
		
		// Validate version format
		if err := validateVersion(platform.Spec.Components.Prometheus.Version); err != nil {
			*allErrs = append(*allErrs, field.Invalid(
				prometheusPath.Child("version"),
				platform.Spec.Components.Prometheus.Version,
				err.Error(),
			))
		}

		// Validate retention format
		if err := validateRetention(platform.Spec.Components.Prometheus.Retention); err != nil {
			*allErrs = append(*allErrs, field.Invalid(
				prometheusPath.Child("retention"),
				platform.Spec.Components.Prometheus.Retention,
				err.Error(),
			))
		}

		// Validate remote write URLs
		for i, rw := range platform.Spec.Components.Prometheus.RemoteWrite {
			if _, err := url.Parse(rw.URL); err != nil {
				*allErrs = append(*allErrs, field.Invalid(
					prometheusPath.Child("remoteWrite").Index(i).Child("url"),
					rw.URL,
					"invalid URL format",
				))
			}
		}
	}

	// Validate Grafana configuration
	if platform.Spec.Components.Grafana != nil {
		grafanaPath := componentsPath.Child("grafana")
		
		// Validate version format
		if err := validateGrafanaVersion(platform.Spec.Components.Grafana.Version); err != nil {
			*allErrs = append(*allErrs, field.Invalid(
				grafanaPath.Child("version"),
				platform.Spec.Components.Grafana.Version,
				err.Error(),
			))
		}

		// Validate datasources
		for i, ds := range platform.Spec.Components.Grafana.DataSources {
			if ds.Name == "" {
				*allErrs = append(*allErrs, field.Required(
					grafanaPath.Child("dataSources").Index(i).Child("name"),
					"datasource name is required",
				))
			}
			if ds.Type == "" {
				*allErrs = append(*allErrs, field.Required(
					grafanaPath.Child("dataSources").Index(i).Child("type"),
					"datasource type is required",
				))
			}
			if _, err := url.Parse(ds.URL); err != nil {
				*allErrs = append(*allErrs, field.Invalid(
					grafanaPath.Child("dataSources").Index(i).Child("url"),
					ds.URL,
					"invalid URL format",
				))
			}
		}
	}

	// Validate Loki configuration
	if platform.Spec.Components.Loki != nil {
		lokiPath := componentsPath.Child("loki")
		
		// Validate version format
		if err := validateVersion(platform.Spec.Components.Loki.Version); err != nil {
			*allErrs = append(*allErrs, field.Invalid(
				lokiPath.Child("version"),
				platform.Spec.Components.Loki.Version,
				err.Error(),
			))
		}

		// Validate S3 configuration if enabled
		if platform.Spec.Components.Loki.S3 != nil && platform.Spec.Components.Loki.S3.Enabled {
			s3Path := lokiPath.Child("s3")
			if platform.Spec.Components.Loki.S3.BucketName == "" {
				*allErrs = append(*allErrs, field.Required(
					s3Path.Child("bucketName"),
					"S3 bucket name is required when S3 is enabled",
				))
			}
			if platform.Spec.Components.Loki.S3.Region == "" {
				*allErrs = append(*allErrs, field.Required(
					s3Path.Child("region"),
					"S3 region is required when S3 is enabled",
				))
			}
		}
	}

	// Validate Tempo configuration
	if platform.Spec.Components.Tempo != nil {
		tempoPath := componentsPath.Child("tempo")
		
		// Validate version format
		if err := validateVersion(platform.Spec.Components.Tempo.Version); err != nil {
			*allErrs = append(*allErrs, field.Invalid(
				tempoPath.Child("version"),
				platform.Spec.Components.Tempo.Version,
				err.Error(),
			))
		}

		// Validate retention format
		if err := validateRetention(platform.Spec.Components.Tempo.Retention); err != nil {
			*allErrs = append(*allErrs, field.Invalid(
				tempoPath.Child("retention"),
				platform.Spec.Components.Tempo.Retention,
				err.Error(),
			))
		}
	}

	return nil
}

func (r *ObservabilityPlatformWebhook) validateResources(platform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList) error {
	// Helper function to validate resource requirements
	validateComponentResources := func(resources corev1.ResourceRequirements, path *field.Path) {
		// Check if requests are specified
		if resources.Requests != nil {
			// Validate CPU request
			if cpu, ok := resources.Requests["cpu"]; ok {
				if cpu.MilliValue() < 10 { // Minimum 10m CPU
					*allErrs = append(*allErrs, field.Invalid(
						path.Child("resources", "requests", "cpu"),
						cpu.String(),
						"CPU request must be at least 10m",
					))
				}
			}
			// Validate memory request
			if memory, ok := resources.Requests["memory"]; ok {
				if memory.Value() < 128*1024*1024 { // Minimum 128Mi
					*allErrs = append(*allErrs, field.Invalid(
						path.Child("resources", "requests", "memory"),
						memory.String(),
						"memory request must be at least 128Mi",
					))
				}
			}
		}

		// Check if limits are specified and validate against requests
		if resources.Limits != nil && resources.Requests != nil {
			// CPU limits should be >= requests
			if cpuLimit, ok := resources.Limits["cpu"]; ok {
				if cpuRequest, ok := resources.Requests["cpu"]; ok {
					if cpuLimit.MilliValue() < cpuRequest.MilliValue() {
						*allErrs = append(*allErrs, field.Invalid(
							path.Child("resources", "limits", "cpu"),
							cpuLimit.String(),
							"CPU limit must be greater than or equal to CPU request",
						))
					}
				}
			}
			// Memory limits should be >= requests
			if memLimit, ok := resources.Limits["memory"]; ok {
				if memRequest, ok := resources.Requests["memory"]; ok {
					if memLimit.Value() < memRequest.Value() {
						*allErrs = append(*allErrs, field.Invalid(
							path.Child("resources", "limits", "memory"),
							memLimit.String(),
							"memory limit must be greater than or equal to memory request",
						))
					}
				}
			}
		}
	}

	// Validate resources for each component
	if platform.Spec.Components.Prometheus != nil {
		validateComponentResources(
			platform.Spec.Components.Prometheus.Resources,
			field.NewPath("spec", "components", "prometheus"),
		)
	}

	if platform.Spec.Components.Grafana != nil {
		validateComponentResources(
			platform.Spec.Components.Grafana.Resources,
			field.NewPath("spec", "components", "grafana"),
		)
	}

	if platform.Spec.Components.Loki != nil {
		validateComponentResources(
			platform.Spec.Components.Loki.Resources,
			field.NewPath("spec", "components", "loki"),
		)
	}

	if platform.Spec.Components.Tempo != nil {
		validateComponentResources(
			platform.Spec.Components.Tempo.Resources,
			field.NewPath("spec", "components", "tempo"),
		)
	}

	if platform.Spec.Components.OpenTelemetryCollector != nil {
		validateComponentResources(
			platform.Spec.Components.OpenTelemetryCollector.Resources,
			field.NewPath("spec", "components", "openTelemetryCollector"),
		)
	}

	return nil
}

func (r *ObservabilityPlatformWebhook) validateStorage(platform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList) error {
	// Helper function to validate storage spec
	validateStorageSpec := func(storage observabilityv1beta1.StorageSpec, path *field.Path) {
		// Validate storage size
		if !storage.Size.IsZero() {
			minSize := resource.MustParse("1Gi")
			if storage.Size.Cmp(minSize) < 0 {
				*allErrs = append(*allErrs, field.Invalid(
					path.Child("storage", "size"),
					storage.Size.String(),
					"storage size must be at least 1Gi",
				))
			}
		}

		// If VolumeClaimTemplate is specified, validate it
		if storage.VolumeClaimTemplate != nil {
			if storage.VolumeClaimTemplate.Spec.Resources.Requests != nil {
				if size, ok := storage.VolumeClaimTemplate.Spec.Resources.Requests["storage"]; ok {
					minSize := resource.MustParse("1Gi")
					if size.Cmp(minSize) < 0 {
						*allErrs = append(*allErrs, field.Invalid(
							path.Child("storage", "volumeClaimTemplate", "spec", "resources", "requests", "storage"),
							size.String(),
							"storage size must be at least 1Gi",
						))
					}
				}
			}
		}
	}

	// Validate storage for each component
	if platform.Spec.Components.Prometheus != nil {
		validateStorageSpec(
			platform.Spec.Components.Prometheus.Storage,
			field.NewPath("spec", "components", "prometheus"),
		)
	}

	if platform.Spec.Components.Loki != nil {
		validateStorageSpec(
			platform.Spec.Components.Loki.Storage,
			field.NewPath("spec", "components", "loki"),
		)
	}

	if platform.Spec.Components.Tempo != nil {
		validateStorageSpec(
			platform.Spec.Components.Tempo.Storage,
			field.NewPath("spec", "components", "tempo"),
		)
	}

	return nil
}

func (r *ObservabilityPlatformWebhook) validateNetworking(platform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList) error {
	// Validate Grafana ingress if enabled
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Ingress != nil {
		ingress := platform.Spec.Components.Grafana.Ingress
		ingressPath := field.NewPath("spec", "components", "grafana", "ingress")

		if ingress.Enabled {
			// Host is required when ingress is enabled
			if ingress.Host == "" {
				*allErrs = append(*allErrs, field.Required(
					ingressPath.Child("host"),
					"host is required when ingress is enabled",
				))
			} else {
				// Validate host format
				if err := validateHostname(ingress.Host); err != nil {
					*allErrs = append(*allErrs, field.Invalid(
						ingressPath.Child("host"),
						ingress.Host,
						err.Error(),
					))
				}
			}

			// If TLS is enabled, validate TLS configuration
			if ingress.TLS != nil && ingress.TLS.Enabled {
				if ingress.TLS.SecretName == "" {
					*allErrs = append(*allErrs, field.Required(
						ingressPath.Child("tls", "secretName"),
						"TLS secret name is required when TLS is enabled",
					))
				}
			}

			// Validate annotations
			for key, value := range ingress.Annotations {
				// Check for common misconfigurations
				if strings.Contains(key, "nginx.ingress.kubernetes.io/ssl-redirect") && value == "false" && ingress.TLS != nil && ingress.TLS.Enabled {
					*allErrs = append(*allErrs, field.Invalid(
						ingressPath.Child("annotations", key),
						value,
						"SSL redirect should not be disabled when TLS is enabled",
					))
				}
			}
		}
	}

	return nil
}

func (r *ObservabilityPlatformWebhook) validateSecurity(platform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList) error {
	if platform.Spec.Security == nil {
		return nil
	}

	securityPath := field.NewPath("spec", "security")

	// Validate authentication configuration
	if platform.Spec.Security.Authentication.Type != "" {
		authPath := securityPath.Child("authentication")

		switch platform.Spec.Security.Authentication.Type {
		case "oidc":
			if platform.Spec.Security.Authentication.OIDC == nil {
				*allErrs = append(*allErrs, field.Required(
					authPath.Child("oidc"),
					"OIDC configuration is required when authentication type is 'oidc'",
				))
			} else {
				oidc := platform.Spec.Security.Authentication.OIDC
				if oidc.Issuer == "" {
					*allErrs = append(*allErrs, field.Required(
						authPath.Child("oidc", "issuer"),
						"OIDC issuer is required",
					))
				} else if _, err := url.Parse(oidc.Issuer); err != nil {
					*allErrs = append(*allErrs, field.Invalid(
						authPath.Child("oidc", "issuer"),
						oidc.Issuer,
						"invalid issuer URL",
					))
				}
				if oidc.ClientID == "" {
					*allErrs = append(*allErrs, field.Required(
						authPath.Child("oidc", "clientId"),
						"OIDC client ID is required",
					))
				}
			}
		case "ldap":
			if platform.Spec.Security.Authentication.LDAP == nil {
				*allErrs = append(*allErrs, field.Required(
					authPath.Child("ldap"),
					"LDAP configuration is required when authentication type is 'ldap'",
				))
			} else {
				ldap := platform.Spec.Security.Authentication.LDAP
				if ldap.Host == "" {
					*allErrs = append(*allErrs, field.Required(
						authPath.Child("ldap", "host"),
						"LDAP host is required",
					))
				}
				if ldap.BindDN == "" {
					*allErrs = append(*allErrs, field.Required(
						authPath.Child("ldap", "bindDn"),
						"LDAP bind DN is required",
					))
				}
				if ldap.UserBaseDN == "" {
					*allErrs = append(*allErrs, field.Required(
						authPath.Child("ldap", "userBaseDn"),
						"LDAP user base DN is required",
					))
				}
			}
		case "basic":
			if platform.Spec.Security.Authentication.Basic == nil {
				*allErrs = append(*allErrs, field.Required(
					authPath.Child("basic"),
					"Basic auth configuration is required when authentication type is 'basic'",
				))
			} else {
				if platform.Spec.Security.Authentication.Basic.Username == "" {
					*allErrs = append(*allErrs, field.Required(
						authPath.Child("basic", "username"),
						"username is required for basic auth",
					))
				}
			}
		default:
			*allErrs = append(*allErrs, field.Invalid(
				authPath.Child("type"),
				platform.Spec.Security.Authentication.Type,
				"authentication type must be one of: oidc, ldap, basic",
			))
		}
	}

	// Validate TLS configuration
	if platform.Spec.Security.TLS.Enabled {
		if !platform.Spec.Security.TLS.AutoTLS && platform.Spec.Security.TLS.CertSecret == "" {
			*allErrs = append(*allErrs, field.Required(
				securityPath.Child("tls", "certSecret"),
				"certificate secret is required when TLS is enabled and auto-TLS is disabled",
			))
		}
	}

	return nil
}

func (r *ObservabilityPlatformWebhook) validateBackup(platform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList) error {
	if platform.Spec.Backup == nil || !platform.Spec.Backup.Enabled {
		return nil
	}

	backupPath := field.NewPath("spec", "backup")

	// Validate cron schedule
	if platform.Spec.Backup.Schedule != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(platform.Spec.Backup.Schedule); err != nil {
			*allErrs = append(*allErrs, field.Invalid(
				backupPath.Child("schedule"),
				platform.Spec.Backup.Schedule,
				fmt.Sprintf("invalid cron schedule: %v", err),
			))
		}
	}

	// Validate retention days
	if platform.Spec.Backup.RetentionDays < 1 {
		*allErrs = append(*allErrs, field.Invalid(
			backupPath.Child("retentionDays"),
			platform.Spec.Backup.RetentionDays,
			"retention days must be at least 1",
		))
	}

	// Validate destination
	destPath := backupPath.Child("destination")
	if platform.Spec.Backup.Destination.Type == "" {
		*allErrs = append(*allErrs, field.Required(
			destPath.Child("type"),
			"backup destination type is required",
		))
	} else {
		switch platform.Spec.Backup.Destination.Type {
		case "s3":
			if platform.Spec.Backup.Destination.S3 == nil {
				*allErrs = append(*allErrs, field.Required(
					destPath.Child("s3"),
					"S3 configuration is required when destination type is 's3'",
				))
			} else {
				if platform.Spec.Backup.Destination.S3.BucketName == "" {
					*allErrs = append(*allErrs, field.Required(
						destPath.Child("s3", "bucketName"),
						"S3 bucket name is required",
					))
				}
				if platform.Spec.Backup.Destination.S3.Region == "" {
					*allErrs = append(*allErrs, field.Required(
						destPath.Child("s3", "region"),
						"S3 region is required",
					))
				}
			}
		case "gcs":
			if platform.Spec.Backup.Destination.GCS == nil {
				*allErrs = append(*allErrs, field.Required(
					destPath.Child("gcs"),
					"GCS configuration is required when destination type is 'gcs'",
				))
			} else {
				if platform.Spec.Backup.Destination.GCS.BucketName == "" {
					*allErrs = append(*allErrs, field.Required(
						destPath.Child("gcs", "bucketName"),
						"GCS bucket name is required",
					))
				}
			}
		case "azure":
			if platform.Spec.Backup.Destination.Azure == nil {
				*allErrs = append(*allErrs, field.Required(
					destPath.Child("azure"),
					"Azure configuration is required when destination type is 'azure'",
				))
			} else {
				if platform.Spec.Backup.Destination.Azure.ContainerName == "" {
					*allErrs = append(*allErrs, field.Required(
						destPath.Child("azure", "containerName"),
						"Azure container name is required",
					))
				}
				if platform.Spec.Backup.Destination.Azure.StorageAccount == "" {
					*allErrs = append(*allErrs, field.Required(
						destPath.Child("azure", "storageAccount"),
						"Azure storage account is required",
					))
				}
			}
		default:
			*allErrs = append(*allErrs, field.Invalid(
				destPath.Child("type"),
				platform.Spec.Backup.Destination.Type,
				"destination type must be one of: s3, gcs, azure",
			))
		}
	}

	return nil
}

func (r *ObservabilityPlatformWebhook) validateHighAvailability(platform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList) error {
	if platform.Spec.HighAvailability == nil || !platform.Spec.HighAvailability.Enabled {
		return nil
	}

	haPath := field.NewPath("spec", "highAvailability")

	// Validate minimum replicas
	if platform.Spec.HighAvailability.MinReplicas < 3 {
		*allErrs = append(*allErrs, field.Invalid(
			haPath.Child("minReplicas"),
			platform.Spec.HighAvailability.MinReplicas,
			"minimum replicas for HA must be at least 3",
		))
	}

	// Ensure components have appropriate replicas for HA
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		if platform.Spec.Components.Prometheus.Replicas < 3 {
			*allErrs = append(*allErrs, field.Invalid(
				field.NewPath("spec", "components", "prometheus", "replicas"),
				platform.Spec.Components.Prometheus.Replicas,
				"Prometheus replicas must be at least 3 when HA is enabled",
			))
		}
	}

	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		if platform.Spec.Components.Grafana.Replicas < 2 {
			*allErrs = append(*allErrs, field.Invalid(
				field.NewPath("spec", "components", "grafana", "replicas"),
				platform.Spec.Components.Grafana.Replicas,
				"Grafana replicas must be at least 2 when HA is enabled",
			))
		}
	}

	if platform.Spec.Alerting.Alertmanager != nil && platform.Spec.Alerting.Alertmanager.Enabled {
		if platform.Spec.Alerting.Alertmanager.Replicas < 3 {
			*allErrs = append(*allErrs, field.Invalid(
				field.NewPath("spec", "alerting", "alertmanager", "replicas"),
				platform.Spec.Alerting.Alertmanager.Replicas,
				"Alertmanager replicas must be at least 3 when HA is enabled",
			))
		}
	}

	return nil
}

func (r *ObservabilityPlatformWebhook) validateImmutableFields(oldPlatform, newPlatform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList) error {
	// Storage class cannot be changed once set
	if oldPlatform.Spec.Components.Prometheus != nil && newPlatform.Spec.Components.Prometheus != nil {
		if oldPlatform.Spec.Components.Prometheus.Storage.StorageClassName != "" &&
			oldPlatform.Spec.Components.Prometheus.Storage.StorageClassName != newPlatform.Spec.Components.Prometheus.Storage.StorageClassName {
			*allErrs = append(*allErrs, field.Forbidden(
				field.NewPath("spec", "components", "prometheus", "storage", "storageClassName"),
				"storage class cannot be changed once set",
			))
		}
	}

	// Similar checks for other components...

	return nil
}

func (r *ObservabilityPlatformWebhook) validateVersionUpgrades(oldPlatform, newPlatform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList, warnings *admission.Warnings) error {
	// Check Prometheus version upgrade
	if oldPlatform.Spec.Components.Prometheus != nil && newPlatform.Spec.Components.Prometheus != nil {
		oldVersion := oldPlatform.Spec.Components.Prometheus.Version
		newVersion := newPlatform.Spec.Components.Prometheus.Version

		if oldVersion != newVersion {
			if isDowngrade(oldVersion, newVersion) {
				*allErrs = append(*allErrs, field.Invalid(
					field.NewPath("spec", "components", "prometheus", "version"),
					newVersion,
					fmt.Sprintf("downgrade from %s to %s is not supported", oldVersion, newVersion),
				))
			} else if isMajorVersionChange(oldVersion, newVersion) {
				*warnings = append(*warnings, fmt.Sprintf(
					"Major version upgrade detected for Prometheus (%s -> %s). Please ensure you have reviewed the upgrade documentation and have appropriate backups.",
					oldVersion, newVersion,
				))
			}
		}
	}

	// Similar checks for other components...

	return nil
}

func (r *ObservabilityPlatformWebhook) validateScalingChanges(oldPlatform, newPlatform *observabilityv1beta1.ObservabilityPlatform, allErrs *field.ErrorList, warnings *admission.Warnings) error {
	// Check for drastic scaling changes
	if oldPlatform.Spec.Components.Prometheus != nil && newPlatform.Spec.Components.Prometheus != nil {
		oldReplicas := oldPlatform.Spec.Components.Prometheus.Replicas
		newReplicas := newPlatform.Spec.Components.Prometheus.Replicas

		// Warn about scaling down in HA mode
		if oldPlatform.Spec.HighAvailability != nil && oldPlatform.Spec.HighAvailability.Enabled {
			if newReplicas < oldReplicas && newReplicas < 3 {
				*warnings = append(*warnings,
					"Scaling down Prometheus below 3 replicas in HA mode may impact availability",
				)
			}
		}

		// Warn about large scale changes
		if float64(newReplicas) > float64(oldReplicas)*2 || float64(newReplicas) < float64(oldReplicas)/2 {
			*warnings = append(*warnings, fmt.Sprintf(
				"Large scaling change detected for Prometheus (%d -> %d). This may cause temporary disruption.",
				oldReplicas, newReplicas,
			))
		}
	}

	return nil
}

// Utility functions

func validateVersion(version string) error {
	// Version pattern: v2.45.0 or 2.45.0
	versionRegex := regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)
	if !versionRegex.MatchString(version) {
		return fmt.Errorf("version must follow semantic versioning (e.g., v2.45.0 or 2.45.0)")
	}
	return nil
}

func validateGrafanaVersion(version string) error {
	// Grafana version pattern: 10.0.0 (no v prefix)
	versionRegex := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !versionRegex.MatchString(version) {
		return fmt.Errorf("Grafana version must follow semantic versioning without 'v' prefix (e.g., 10.0.0)")
	}
	return nil
}

func validateRetention(retention string) error {
	// Retention format: 30d, 24h, 1w, etc.
	retentionRegex := regexp.MustCompile(`^\d+[hdwmy]$`)
	if !retentionRegex.MatchString(retention) {
		return fmt.Errorf("retention must be a duration (e.g., 30d, 24h, 1w)")
	}
	return nil
}

func validateHostname(hostname string) error {
	// Basic hostname validation
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	
	// Check for wildcard
	if strings.HasPrefix(hostname, "*.") {
		hostname = hostname[2:]
	}
	
	// Validate each part of the hostname
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return fmt.Errorf("hostname must be a fully qualified domain name")
	}
	
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return fmt.Errorf("each hostname label must be between 1 and 63 characters")
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`).MatchString(part) {
			return fmt.Errorf("hostname labels must start and end with alphanumeric characters")
		}
	}
	
	return nil
}

func isDowngrade(oldVersion, newVersion string) bool {
	// Simple version comparison - in production, use a proper semver library
	oldParts := strings.Split(strings.TrimPrefix(oldVersion, "v"), ".")
	newParts := strings.Split(strings.TrimPrefix(newVersion, "v"), ".")
	
	for i := 0; i < len(oldParts) && i < len(newParts); i++ {
		if oldParts[i] > newParts[i] {
			return true
		} else if oldParts[i] < newParts[i] {
			return false
		}
	}
	
	return false
}

func isMajorVersionChange(oldVersion, newVersion string) bool {
	oldMajor := strings.Split(strings.TrimPrefix(oldVersion, "v"), ".")[0]
	newMajor := strings.Split(strings.TrimPrefix(newVersion, "v"), ".")[0]
	return oldMajor != newMajor
}

func generateSecurePassword() string {
	// In production, use a cryptographically secure random generator
	// This is a placeholder implementation
	return fmt.Sprintf("gunj-%d", time.Now().Unix())
}

// SetupWebhookWithManager sets up the webhook with the Manager
func (r *ObservabilityPlatformWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// Initialize the webhook with client and validators
	r.Client = mgr.GetClient()
	r.QuotaValidator = quota.NewQuotaValidator(mgr.GetClient())
	r.SecurityValidator = security.NewSecurityValidator(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).
		For(&observabilityv1beta1.ObservabilityPlatform{}).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}
