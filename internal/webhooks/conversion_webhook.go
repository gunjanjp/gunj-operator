/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// ConversionWebhook handles conversion requests for ObservabilityPlatform resources
type ConversionWebhook struct {
	scheme  *runtime.Scheme
	decoder *conversion.Decoder
	log     logr.Logger
}

// NewConversionWebhook creates a new conversion webhook
func NewConversionWebhook(mgr manager.Manager, log logr.Logger) (*ConversionWebhook, error) {
	scheme := mgr.GetScheme()
	
	// Ensure all versions are registered
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding v1alpha1 to scheme: %w", err)
	}
	if err := v1beta1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding v1beta1 to scheme: %w", err)
	}
	
	decoder, err := conversion.NewDecoder(scheme)
	if err != nil {
		return nil, fmt.Errorf("creating decoder: %w", err)
	}
	
	return &ConversionWebhook{
		scheme:  scheme,
		decoder: decoder,
		log:     log,
	}, nil
}

// ServeHTTP handles the HTTP request for conversion
func (w *ConversionWebhook) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var body []byte
	var err error
	ctx := req.Context()
	
	if body, err = readRequestBody(req); err != nil {
		w.log.Error(err, "failed to read request body")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	
	// Decode the conversion review
	obj, gvk, err := w.decoder.Decode(body)
	if err != nil {
		w.log.Error(err, "failed to decode request")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	
	var responseObj runtime.Object
	switch *gvk {
	case apiextensionsv1.SchemeGroupVersion.WithKind("ConversionReview"):
		conversionReview := obj.(*apiextensionsv1.ConversionReview)
		responseObj = w.handleConversionReview(ctx, conversionReview)
	default:
		w.log.Error(nil, "unexpected request kind", "kind", gvk.String())
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	
	// Write response
	if err := writeResponse(rw, responseObj); err != nil {
		w.log.Error(err, "failed to write response")
	}
}

// handleConversionReview processes the conversion review request
func (w *ConversionWebhook) handleConversionReview(ctx context.Context, review *apiextensionsv1.ConversionReview) *apiextensionsv1.ConversionReview {
	if review.Request == nil {
		return errorResponse(review, "request is nil")
	}
	
	// Log the conversion request
	w.log.Info("handling conversion",
		"desiredAPIVersion", review.Request.DesiredAPIVersion,
		"objectCount", len(review.Request.Objects),
		"uid", review.Request.UID)
	
	// Prepare response
	response := &apiextensionsv1.ConversionResponse{
		UID:              review.Request.UID,
		ConvertedObjects: []runtime.RawExtension{},
		Result:           metav1.Status{Status: metav1.StatusSuccess},
	}
	
	// Get the target GVK
	targetGVK, err := parseGroupVersionKind(review.Request.DesiredAPIVersion)
	if err != nil {
		return errorResponseWithMessage(review, fmt.Sprintf("failed to parse desired API version: %v", err))
	}
	
	// Convert each object
	for _, obj := range review.Request.Objects {
		converted, err := w.convertObject(ctx, obj.Raw, targetGVK)
		if err != nil {
			w.log.Error(err, "failed to convert object")
			return errorResponseWithMessage(review, fmt.Sprintf("conversion failed: %v", err))
		}
		
		response.ConvertedObjects = append(response.ConvertedObjects, runtime.RawExtension{
			Raw: converted,
		})
	}
	
	review.Response = response
	return review
}

// convertObject converts a single object to the target version
func (w *ConversionWebhook) convertObject(ctx context.Context, raw []byte, targetGVK schema.GroupVersionKind) ([]byte, error) {
	// Decode the source object
	srcObj, srcGVK, err := w.decoder.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("decoding source object: %w", err)
	}
	
	w.log.V(1).Info("converting object",
		"from", srcGVK.String(),
		"to", targetGVK.String())
	
	// If source and target are the same, return as-is
	if srcGVK.GroupVersion() == targetGVK.GroupVersion() {
		return raw, nil
	}
	
	// Handle specific conversions
	switch srcGVK.GroupVersion().String() {
	case "observability.io/v1alpha1":
		switch targetGVK.GroupVersion().String() {
		case "observability.io/v1beta1":
			return w.convertV1Alpha1ToV1Beta1(srcObj)
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", srcGVK.GroupVersion(), targetGVK.GroupVersion())
		}
	case "observability.io/v1beta1":
		switch targetGVK.GroupVersion().String() {
		case "observability.io/v1alpha1":
			return w.convertV1Beta1ToV1Alpha1(srcObj)
		default:
			return nil, fmt.Errorf("unsupported conversion from %s to %s", srcGVK.GroupVersion(), targetGVK.GroupVersion())
		}
	default:
		return nil, fmt.Errorf("unsupported source version: %s", srcGVK.GroupVersion())
	}
}

// convertV1Alpha1ToV1Beta1 converts from v1alpha1 to v1beta1
func (w *ConversionWebhook) convertV1Alpha1ToV1Beta1(srcObj runtime.Object) ([]byte, error) {
	src, ok := srcObj.(*v1alpha1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected v1alpha1.ObservabilityPlatform, got %T", srcObj)
	}
	
	// Create v1beta1 instance
	dst := &v1beta1.ObservabilityPlatform{}
	
	// Use the conversion implementation
	if err := src.ConvertTo(dst); err != nil {
		return nil, fmt.Errorf("converting to v1beta1: %w", err)
	}
	
	// Preserve annotations that indicate conversion
	if dst.Annotations == nil {
		dst.Annotations = make(map[string]string)
	}
	dst.Annotations["observability.io/converted-from"] = "v1alpha1"
	dst.Annotations["observability.io/conversion-timestamp"] = metav1.Now().Format(metav1.RFC3339)
	
	// Encode the result
	return json.Marshal(dst)
}

// convertV1Beta1ToV1Alpha1 converts from v1beta1 to v1alpha1
func (w *ConversionWebhook) convertV1Beta1ToV1Alpha1(srcObj runtime.Object) ([]byte, error) {
	src, ok := srcObj.(*v1beta1.ObservabilityPlatform)
	if !ok {
		return nil, fmt.Errorf("expected v1beta1.ObservabilityPlatform, got %T", srcObj)
	}
	
	// Create v1alpha1 instance
	dst := &v1alpha1.ObservabilityPlatform{}
	
	// Use the conversion implementation
	if err := dst.ConvertFrom(src); err != nil {
		return nil, fmt.Errorf("converting from v1beta1: %w", err)
	}
	
	// Preserve annotations that indicate conversion
	if dst.Annotations == nil {
		dst.Annotations = make(map[string]string)
	}
	dst.Annotations["observability.io/converted-from"] = "v1beta1"
	dst.Annotations["observability.io/conversion-timestamp"] = metav1.Now().Format(metav1.RFC3339)
	
	// Log warnings for lost fields
	w.logLostFields(src, dst)
	
	// Encode the result
	return json.Marshal(dst)
}

// logLostFields logs warnings about fields that are lost during conversion
func (w *ConversionWebhook) logLostFields(src *v1beta1.ObservabilityPlatform, dst *v1alpha1.ObservabilityPlatform) {
	// Check for fields that exist in v1beta1 but not in v1alpha1
	if src.Spec.Security != nil {
		w.log.Info("field lost during conversion", "field", "spec.security", "name", src.Name)
	}
	
	if src.Spec.Components.Prometheus != nil {
		if len(src.Spec.Components.Prometheus.ExternalLabels) > 0 {
			w.log.Info("field lost during conversion", "field", "spec.components.prometheus.externalLabels", "name", src.Name)
		}
		if src.Spec.Components.Prometheus.AdditionalScrapeConfigs != "" {
			w.log.Info("field lost during conversion", "field", "spec.components.prometheus.additionalScrapeConfigs", "name", src.Name)
		}
	}
	
	if src.Spec.Components.Grafana != nil {
		if len(src.Spec.Components.Grafana.Plugins) > 0 {
			w.log.Info("field lost during conversion", "field", "spec.components.grafana.plugins", "name", src.Name)
		}
		if src.Spec.Components.Grafana.SMTP != nil {
			w.log.Info("field lost during conversion", "field", "spec.components.grafana.smtp", "name", src.Name)
		}
	}
	
	if src.Spec.Components.Loki != nil && src.Spec.Components.Loki.CompactorEnabled {
		w.log.Info("field lost during conversion", "field", "spec.components.loki.compactorEnabled", "name", src.Name)
	}
	
	if src.Spec.Components.Tempo != nil && src.Spec.Components.Tempo.SearchEnabled {
		w.log.Info("field lost during conversion", "field", "spec.components.tempo.searchEnabled", "name", src.Name)
	}
	
	if src.Spec.Global != nil {
		if src.Spec.Global.Affinity != nil {
			w.log.Info("field lost during conversion", "field", "spec.global.affinity", "name", src.Name)
		}
		if len(src.Spec.Global.ImagePullSecrets) > 0 {
			w.log.Info("field lost during conversion", "field", "spec.global.imagePullSecrets", "name", src.Name)
		}
	}
}

// SetupWebhookWithManager sets up the conversion webhook with the manager
func SetupWebhookWithManager(mgr manager.Manager) error {
	log := mgr.GetLogger().WithName("conversion-webhook")
	
	webhook, err := NewConversionWebhook(mgr, log)
	if err != nil {
		return fmt.Errorf("creating conversion webhook: %w", err)
	}
	
	// Register the webhook handler
	mgr.GetWebhookServer().Register("/convert", webhook)
	
	log.Info("conversion webhook registered")
	return nil
}

// Helper functions

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()
	
	body := make([]byte, r.ContentLength)
	if _, err := r.Body.Read(body); err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}
	
	return body, nil
}

func writeResponse(w http.ResponseWriter, obj runtime.Object) error {
	w.Header().Set("Content-Type", "application/json")
	
	resp, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	
	if _, err := w.Write(resp); err != nil {
		return fmt.Errorf("writing response: %w", err)
	}
	
	return nil
}

func parseGroupVersionKind(apiVersion string) (schema.GroupVersionKind, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	
	// For our CRD, the kind is always ObservabilityPlatform
	return gv.WithKind("ObservabilityPlatform"), nil
}

func errorResponse(review *apiextensionsv1.ConversionReview, reason string) *apiextensionsv1.ConversionReview {
	return errorResponseWithMessage(review, reason)
}

func errorResponseWithMessage(review *apiextensionsv1.ConversionReview, message string) *apiextensionsv1.ConversionReview {
	review.Response = &apiextensionsv1.ConversionResponse{
		UID: review.Request.UID,
		Result: metav1.Status{
			Status:  metav1.StatusFailure,
			Message: message,
		},
	}
	return review
}
