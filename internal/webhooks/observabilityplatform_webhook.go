package webhooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/deprecation"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-observability-io-v1beta1-observabilityplatform,mutating=false,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1beta1,name=vobservabilityplatform.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ObservabilityPlatformWebhook{}

var observabilityplatformlog = logf.Log.WithName("observabilityplatform-webhook")

// ObservabilityPlatformWebhook handles admission webhooks for ObservabilityPlatform
type ObservabilityPlatformWebhook struct {
	ConfigValidator      *ConfigurationValidator
	DeprecationHandler   *deprecation.WebhookHandler
}

// SetupWebhookWithManager sets up the webhook with the controller manager
func (w *ObservabilityPlatformWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// Initialize the configuration validator
	w.ConfigValidator = NewConfigurationValidator(observabilityplatformlog)
	
	// Initialize the deprecation handler
	w.DeprecationHandler = deprecation.NewWebhookHandler(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1beta1.ObservabilityPlatform{}).
		Complete()
}

// ValidateCreate implements webhook.Validator
func (w *ObservabilityPlatformWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	platform, ok := obj.(*v1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform but got %T", obj)
	}

	observabilityplatformlog.Info("validating create", "name", platform.Name, "namespace", platform.Namespace)

	// Perform configuration validation
	allErrs := w.ConfigValidator.ValidateConfiguration(ctx, platform)
	
	// Check for deprecations
	warnings := w.checkDeprecations(ctx, platform)
	
	if len(allErrs) > 0 {
		return warnings, errors.NewInvalid(
			schema.GroupKind{Group: "observability.io", Kind: "ObservabilityPlatform"},
			platform.Name,
			allErrs,
		)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator
func (w *ObservabilityPlatformWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldPlatform, ok := oldObj.(*v1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform but got %T", oldObj)
	}
	
	newPlatform, ok := newObj.(*v1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform but got %T", newObj)
	}

	observabilityplatformlog.Info("validating update", "name", newPlatform.Name, "namespace", newPlatform.Namespace)

	// Perform configuration validation
	allErrs := w.ConfigValidator.ValidateConfiguration(ctx, newPlatform)
	
	// Check for immutable field changes
	allErrs = append(allErrs, w.validateImmutableFields(oldPlatform, newPlatform)...)
	
	// Check for deprecations
	warnings := w.checkDeprecations(ctx, newPlatform)
	
	// Add warnings for changes from non-deprecated to deprecated values
	additionalWarnings := w.checkDeprecationTransitions(oldPlatform, newPlatform)
	warnings = append(warnings, additionalWarnings...)
	
	if len(allErrs) > 0 {
		return warnings, errors.NewInvalid(
			schema.GroupKind{Group: "observability.io", Kind: "ObservabilityPlatform"},
			newPlatform.Name,
			allErrs,
		)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator
func (w *ObservabilityPlatformWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	platform, ok := obj.(*v1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected ObservabilityPlatform but got %T", obj)
	}

	observabilityplatformlog.Info("validating delete", "name", platform.Name, "namespace", platform.Namespace)

	// No specific validation for delete at this time
	return nil, nil
}

// checkDeprecations checks for deprecated fields and values
func (w *ObservabilityPlatformWebhook) checkDeprecations(ctx context.Context, platform *v1beta1.ObservabilityPlatform) admission.Warnings {
	// Convert to unstructured for deprecation checking
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(platform)
	if err != nil {
		observabilityplatformlog.Error(err, "failed to convert to unstructured")
		return nil
	}

	u := &unstructured.Unstructured{Object: unstructuredObj}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "observability.io",
		Version: "v1beta1",
		Kind:    "ObservabilityPlatform",
	})

	// Check for deprecations
	result, err := w.DeprecationHandler.CheckAndRespond(ctx, u, u.GroupVersionKind())
	if err != nil {
		observabilityplatformlog.Error(err, "failed to check deprecations")
		return nil
	}

	return admission.Warnings(result.Warnings)
}

// checkDeprecationTransitions warns when updating from non-deprecated to deprecated values
func (w *ObservabilityPlatformWebhook) checkDeprecationTransitions(old, new *v1beta1.ObservabilityPlatform) admission.Warnings {
	var warnings admission.Warnings

	// Check if Prometheus version is being downgraded to a deprecated version
	if old.Spec.Components != nil && old.Spec.Components.Prometheus != nil &&
		new.Spec.Components != nil && new.Spec.Components.Prometheus != nil {
		
		oldVersion := old.Spec.Components.Prometheus.Version
		newVersion := new.Spec.Components.Prometheus.Version
		
		// Check if new version is deprecated
		if isVersionDeprecated(newVersion) && !isVersionDeprecated(oldVersion) {
			warnings = append(warnings, fmt.Sprintf(
				"[WARNING] Downgrading Prometheus from %s to deprecated version %s. "+
					"Consider using v2.48.0 or later for security and feature improvements.",
				oldVersion, newVersion))
		}
	}

	// Check for other component version downgrades
	warnings = append(warnings, w.checkComponentVersionTransitions(old, new)...)

	return warnings
}

// checkComponentVersionTransitions checks all component version transitions
func (w *ObservabilityPlatformWebhook) checkComponentVersionTransitions(old, new *v1beta1.ObservabilityPlatform) []string {
	var warnings []string

	// Helper function to check version transition
	checkVersion := func(componentName, oldVersion, newVersion string) {
		if oldVersion != "" && newVersion != "" && oldVersion != newVersion {
			if isOlderVersion(newVersion, oldVersion) {
				warnings = append(warnings, fmt.Sprintf(
					"[INFO] %s version changing from %s to %s. "+
						"Ensure this downgrade is intentional and review the migration guide.",
					componentName, oldVersion, newVersion))
			}
		}
	}

	if old.Spec.Components != nil && new.Spec.Components != nil {
		// Check Grafana
		if old.Spec.Components.Grafana != nil && new.Spec.Components.Grafana != nil {
			checkVersion("Grafana",
				old.Spec.Components.Grafana.Version,
				new.Spec.Components.Grafana.Version)
		}

		// Check Loki
		if old.Spec.Components.Loki != nil && new.Spec.Components.Loki != nil {
			checkVersion("Loki",
				old.Spec.Components.Loki.Version,
				new.Spec.Components.Loki.Version)
		}

		// Check Tempo
		if old.Spec.Components.Tempo != nil && new.Spec.Components.Tempo != nil {
			checkVersion("Tempo",
				old.Spec.Components.Tempo.Version,
				new.Spec.Components.Tempo.Version)
		}

		// Check OpenTelemetry Collector
		if old.Spec.Components.OpenTelemetryCollector != nil && new.Spec.Components.OpenTelemetryCollector != nil {
			checkVersion("OpenTelemetry Collector",
				old.Spec.Components.OpenTelemetryCollector.Version,
				new.Spec.Components.OpenTelemetryCollector.Version)
		}
	}

	return warnings
}

// validateImmutableFields checks for changes to immutable fields
func (w *ObservabilityPlatformWebhook) validateImmutableFields(old, new *v1beta1.ObservabilityPlatform) field.ErrorList {
	allErrs := field.ErrorList{}

	// Add any immutable field validations here
	// Example: storage class changes might be immutable

	return allErrs
}

// Helper functions

// isVersionDeprecated checks if a version is deprecated
func isVersionDeprecated(version string) bool {
	// List of deprecated versions
	deprecatedVersions := map[string]bool{
		"v2.30.0": true,
		"v2.31.0": true,
		"v2.32.0": true,
		"v2.33.0": true,
		"v2.34.0": true,
		"v2.35.0": true,
		"v2.36.0": true,
		"v2.37.0": true,
		"v2.38.0": true,
		"v2.39.0": true,
	}

	return deprecatedVersions[version]
}

// isOlderVersion compares two version strings
func isOlderVersion(v1, v2 string) bool {
	// Simple string comparison after normalizing
	// In production, use proper semantic version comparison
	v1 = normalizeVersion(v1)
	v2 = normalizeVersion(v2)
	return v1 < v2
}

// +kubebuilder:webhook:path=/mutate-observability-io-v1beta1-observabilityplatform,mutating=true,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1beta1,name=mobservabilityplatform.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ObservabilityPlatformMutatingWebhook{}

// ObservabilityPlatformMutatingWebhook handles mutating webhooks for ObservabilityPlatform
type ObservabilityPlatformMutatingWebhook struct {
	DeprecationHandler *deprecation.WebhookHandler
}

// SetupWebhookWithManager sets up the mutating webhook with the controller manager
func (w *ObservabilityPlatformMutatingWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// Initialize the deprecation handler
	w.DeprecationHandler = deprecation.NewWebhookHandler(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1beta1.ObservabilityPlatform{}).
		Complete()
}

// Default implements webhook.Defaulter
func (w *ObservabilityPlatformMutatingWebhook) Default(ctx context.Context, obj runtime.Object) error {
	platform, ok := obj.(*v1beta1.ObservabilityPlatform)
	if !ok {
		return fmt.Errorf("expected ObservabilityPlatform but got %T", obj)
	}

	observabilityplatformlog.Info("defaulting", "name", platform.Name, "namespace", platform.Namespace)

	// Set defaults
	w.setDefaults(platform)

	// Attempt auto-migration of deprecated fields
	if err := w.autoMigrateDeprecatedFields(ctx, platform); err != nil {
		observabilityplatformlog.Error(err, "failed to auto-migrate deprecated fields")
		// Don't fail the request, just log the error
	}

	return nil
}

// setDefaults sets default values for the platform
func (w *ObservabilityPlatformMutatingWebhook) setDefaults(platform *v1beta1.ObservabilityPlatform) {
	// Set component defaults
	if platform.Spec.Components != nil {
		// Prometheus defaults
		if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
			if platform.Spec.Components.Prometheus.Replicas == 0 {
				platform.Spec.Components.Prometheus.Replicas = 1
			}
			if platform.Spec.Components.Prometheus.Version == "" {
				platform.Spec.Components.Prometheus.Version = "v2.48.0"
			}
		}

		// Grafana defaults
		if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
			if platform.Spec.Components.Grafana.Replicas == 0 {
				platform.Spec.Components.Grafana.Replicas = 1
			}
			if platform.Spec.Components.Grafana.Version == "" {
				platform.Spec.Components.Grafana.Version = "10.2.0"
			}
			if platform.Spec.Components.Grafana.AdminUser == "" {
				platform.Spec.Components.Grafana.AdminUser = "admin"
			}
		}

		// Loki defaults
		if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
			if platform.Spec.Components.Loki.Replicas == 0 {
				platform.Spec.Components.Loki.Replicas = 1
			}
			if platform.Spec.Components.Loki.Version == "" {
				platform.Spec.Components.Loki.Version = "2.9.0"
			}
		}

		// Tempo defaults
		if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
			if platform.Spec.Components.Tempo.Replicas == 0 {
				platform.Spec.Components.Tempo.Replicas = 1
			}
			if platform.Spec.Components.Tempo.Version == "" {
				platform.Spec.Components.Tempo.Version = "2.3.0"
			}
		}

		// OpenTelemetry Collector defaults
		if platform.Spec.Components.OpenTelemetryCollector != nil && platform.Spec.Components.OpenTelemetryCollector.Enabled {
			if platform.Spec.Components.OpenTelemetryCollector.Replicas == 0 {
				platform.Spec.Components.OpenTelemetryCollector.Replicas = 1
			}
			if platform.Spec.Components.OpenTelemetryCollector.Version == "" {
				platform.Spec.Components.OpenTelemetryCollector.Version = "0.91.0"
			}
			if platform.Spec.Components.OpenTelemetryCollector.Mode == "" {
				platform.Spec.Components.OpenTelemetryCollector.Mode = "deployment"
			}
		}
	}
}

// autoMigrateDeprecatedFields attempts to automatically migrate deprecated fields
func (w *ObservabilityPlatformMutatingWebhook) autoMigrateDeprecatedFields(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
	// This is where you would implement auto-migration logic
	// For now, we'll just log that we're checking
	observabilityplatformlog.V(1).Info("checking for auto-migration opportunities", 
		"name", platform.Name, 
		"namespace", platform.Namespace)

	// Example: Auto-upgrade deprecated Prometheus versions
	if platform.Spec.Components != nil && platform.Spec.Components.Prometheus != nil {
		if isVersionDeprecated(platform.Spec.Components.Prometheus.Version) {
			observabilityplatformlog.Info("auto-upgrading deprecated Prometheus version",
				"old", platform.Spec.Components.Prometheus.Version,
				"new", "v2.48.0")
			platform.Spec.Components.Prometheus.Version = "v2.48.0"
		}
	}

	return nil
}
