/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var log = logf.Log.WithName("conversion-webhook")

// ConversionWebhook handles conversion between different API versions
type ConversionWebhook struct {
	client  client.Client
	scheme  *runtime.Scheme
	decoder *conversion.Decoder
}

// NewConversionWebhook creates a new conversion webhook
func NewConversionWebhook(mgr ctrl.Manager) (*ConversionWebhook, error) {
	return &ConversionWebhook{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}, nil
}

// SetupWithManager sets up the conversion webhook with the manager
func (w *ConversionWebhook) SetupWithManager(mgr ctrl.Manager) error {
	// The conversion webhook is automatically set up by controller-runtime
	// when we implement the Convertible interface on our types
	log.Info("Conversion webhook setup completed")
	return nil
}

// ConversionReview handles the conversion review requests
// This is called by the Kubernetes API server when conversion is needed
func (w *ConversionWebhook) ConversionReview(ctx context.Context, req runtime.Object) (runtime.Object, error) {
	log.V(1).Info("Handling conversion review request")
	
	// The actual conversion is handled by the ConvertTo and ConvertFrom methods
	// implemented on the v1alpha1.ObservabilityPlatform type
	
	return nil, nil
}

// ValidateConversion validates that a conversion between versions is valid
func ValidateConversion(src, dst runtime.Object) error {
	switch src := src.(type) {
	case *v1alpha1.ObservabilityPlatform:
		switch dst := dst.(type) {
		case *v1beta1.ObservabilityPlatform:
			return validateV1Alpha1ToV1Beta1(src, dst)
		default:
			return fmt.Errorf("unsupported conversion from v1alpha1 to %T", dst)
		}
	case *v1beta1.ObservabilityPlatform:
		switch dst := dst.(type) {
		case *v1alpha1.ObservabilityPlatform:
			return validateV1Beta1ToV1Alpha1(src, dst)
		default:
			return fmt.Errorf("unsupported conversion from v1beta1 to %T", dst)
		}
	default:
		return fmt.Errorf("unsupported source type %T", src)
	}
}

// validateV1Alpha1ToV1Beta1 validates conversion from v1alpha1 to v1beta1
func validateV1Alpha1ToV1Beta1(src *v1alpha1.ObservabilityPlatform, dst *v1beta1.ObservabilityPlatform) error {
	// Validate that critical fields are preserved
	if src.Name != dst.Name {
		return fmt.Errorf("name must be preserved during conversion")
	}
	
	if src.Namespace != dst.Namespace {
		return fmt.Errorf("namespace must be preserved during conversion")
	}
	
	// Add more validation as needed
	return nil
}

// validateV1Beta1ToV1Alpha1 validates conversion from v1beta1 to v1alpha1
func validateV1Beta1ToV1Alpha1(src *v1beta1.ObservabilityPlatform, dst *v1alpha1.ObservabilityPlatform) error {
	// Validate that critical fields are preserved
	if src.Name != dst.Name {
		return fmt.Errorf("name must be preserved during conversion")
	}
	
	if src.Namespace != dst.Namespace {
		return fmt.Errorf("namespace must be preserved during conversion")
	}
	
	// Log warnings for fields that will be lost
	if src.Spec.Security != nil {
		log.Info("Security configuration will be lost during conversion from v1beta1 to v1alpha1", 
			"platform", src.Name)
	}
	
	if src.Spec.Grafana != nil && src.Spec.Grafana.SMTP != nil {
		log.Info("Grafana SMTP configuration will be lost during conversion from v1beta1 to v1alpha1", 
			"platform", src.Name)
	}
	
	if src.Spec.Grafana != nil && len(src.Spec.Grafana.Plugins) > 0 {
		log.Info("Grafana plugins configuration will be lost during conversion from v1beta1 to v1alpha1", 
			"platform", src.Name)
	}
	
	if src.Spec.Loki != nil && src.Spec.Loki.CompactorEnabled {
		log.Info("Loki compactor configuration will be lost during conversion from v1beta1 to v1alpha1", 
			"platform", src.Name)
	}
	
	if src.Spec.Tempo != nil && src.Spec.Tempo.SearchEnabled {
		log.Info("Tempo search configuration will be lost during conversion from v1beta1 to v1alpha1", 
			"platform", src.Name)
	}
	
	if src.Spec.Global.Affinity != nil {
		log.Info("Global affinity configuration will be lost during conversion from v1beta1 to v1alpha1", 
			"platform", src.Name)
	}
	
	if len(src.Spec.Global.ImagePullSecrets) > 0 {
		log.Info("Global image pull secrets will be lost during conversion from v1beta1 to v1alpha1", 
			"platform", src.Name)
	}
	
	return nil
}

// ConversionMetrics tracks conversion metrics
type ConversionMetrics struct {
	Total     int64
	Succeeded int64
	Failed    int64
}

var metrics = &ConversionMetrics{}

// RecordConversion records conversion metrics
func RecordConversion(success bool) {
	metrics.Total++
	if success {
		metrics.Succeeded++
	} else {
		metrics.Failed++
	}
	
	log.V(2).Info("Conversion metrics", 
		"total", metrics.Total,
		"succeeded", metrics.Succeeded,
		"failed", metrics.Failed)
}
