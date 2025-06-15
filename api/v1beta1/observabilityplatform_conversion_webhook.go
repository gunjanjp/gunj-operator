/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion"
)

var conversionlog = logf.Log.WithName("observabilityplatform-conversion")

// +kubebuilder:webhook:path=/convert,mutating=false,failurePolicy=fail,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1alpha1;v1beta1,name=cobservabilityplatform.kb.io,sideEffects=None,admissionReviewVersions=v1

// ConversionWebhook handles conversions with enhanced validation
type ConversionWebhook struct {
	client          client.Client
	decoder         *admission.Decoder
	validator       *conversion.FieldValidator
	dryRun          *conversion.DryRunConverter
	rollbackManager *conversion.RollbackManager
	metrics         *conversion.ConversionMetrics
	dataPreserver   *conversion.DataPreserver
	logger          logr.Logger
}

// NewConversionWebhook creates a new conversion webhook with enhanced features
func NewConversionWebhook(client client.Client, scheme *runtime.Scheme) *ConversionWebhook {
	logger := conversionlog.WithName("webhook")
	rollbackStore := conversion.NewInMemoryRollbackStore()
	
	return &ConversionWebhook{
		client:          client,
		validator:       conversion.NewFieldValidator(logger),
		dryRun:          conversion.NewDryRunConverter(logger),
		rollbackManager: conversion.NewRollbackManager(logger, client, rollbackStore),
		metrics:         conversion.GetMetrics(),
		dataPreserver:   conversion.NewDataPreserver(logger, client),
		logger:          logger,
	}
}

// Handle processes conversion requests with comprehensive validation
func (w *ConversionWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Check if this is a dry-run request
	isDryRun := req.DryRun != nil && *req.DryRun
	
	// Extract conversion review from request
	convertReview := &apiextensionsv1.ConversionReview{}
	if err := json.Unmarshal(req.Object.Raw, convertReview); err != nil {
		w.logger.Error(err, "Failed to unmarshal conversion review")
		return admission.Errored(http.StatusBadRequest, err)
	}
	
	// Process conversion request
	response := w.processConversion(ctx, convertReview.Request, isDryRun)
	
	// Build response
	convertReview.Response = response
	convertReview.Response.UID = convertReview.Request.UID
	
	// Marshal response
	responseBytes, err := json.Marshal(convertReview)
	if err != nil {
		w.logger.Error(err, "Failed to marshal conversion response")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	
	return admission.Response{
		AdmissionResponse: admissionv1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Status: metav1.StatusSuccess,
			},
			Patch: responseBytes,
			PatchType: func() *admissionv1.PatchType {
				pt := admissionv1.PatchTypeJSONPatch
				return &pt
			}(),
		},
	}
}

// processConversion handles the actual conversion with validation
func (w *ConversionWebhook) processConversion(ctx context.Context, req *apiextensionsv1.ConversionRequest, isDryRun bool) *apiextensionsv1.ConversionResponse {
	response := &apiextensionsv1.ConversionResponse{
		UID:              req.UID,
		ConvertedObjects: []runtime.RawExtension{},
		Result: metav1.Status{
			Status: metav1.StatusSuccess,
		},
	}
	
	// Start metrics timer
	timer := w.metrics.StartTimer(
		req.Objects[0].Object.GetObjectKind().GroupVersionKind().Version,
		req.DesiredAPIVersion,
		"ObservabilityPlatform",
	)
	defer timer.Complete(true)
	
	// Process each object
	for _, obj := range req.Objects {
		converted, err := w.convertObject(ctx, obj.Object, req.DesiredAPIVersion, isDryRun)
		if err != nil {
			w.logger.Error(err, "Conversion failed",
				"object", obj.Object.GetObjectKind(),
				"targetVersion", req.DesiredAPIVersion)
			
			response.Result = metav1.Status{
				Status:  metav1.StatusFailure,
				Message: err.Error(),
				Reason:  metav1.StatusReasonInvalid,
			}
			
			// Attempt auto-rollback if not dry-run
			if !isDryRun {
				key, _ := client.ObjectKeyFromObject(obj.Object)
				if rollbackErr := w.rollbackManager.AutoRollback(ctx, key, err); rollbackErr != nil {
					w.logger.Error(rollbackErr, "Auto-rollback failed")
				}
			}
			
			timer.Complete(false)
			return response
		}
		
		// Add to converted objects
		response.ConvertedObjects = append(response.ConvertedObjects, runtime.RawExtension{
			Object: converted,
		})
	}
	
	return response
}

// convertObject performs the conversion with validation for a single object
func (w *ConversionWebhook) convertObject(ctx context.Context, obj runtime.Object, targetVersion string, isDryRun bool) (runtime.Object, error) {
	// Determine source version
	sourceVersion := obj.GetObjectKind().GroupVersionKind().Version
	
	w.logger.V(1).Info("Converting object",
		"sourceVersion", sourceVersion,
		"targetVersion", targetVersion,
		"dryRun", isDryRun)
	
	// Create snapshot before conversion (unless dry-run)
	if !isDryRun {
		snapshot, err := w.rollbackManager.CreateSnapshot(ctx, obj, targetVersion, "webhook-conversion")
		if err != nil {
			w.logger.Error(err, "Failed to create snapshot")
			// Continue with conversion but log the error
		} else {
			w.logger.V(2).Info("Created conversion snapshot", "snapshotID", snapshot.SnapshotID)
		}
	}
	
	// Perform dry-run validation first
	dryRunResult, err := w.dryRun.DryRunConversion(ctx, obj, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("dry-run validation failed: %w", err)
	}
	
	// Check dry-run results
	if !dryRunResult.Success {
		// Build detailed error message
		errMsg := "Conversion validation failed:\n"
		
		// Add validation errors
		for _, err := range dryRunResult.ValidationResult.Errors {
			errMsg += fmt.Sprintf("- %s: %s\n", err.Field, err.Detail)
			
			// Record field validation error metric
			w.metrics.RecordFieldValidationError(
				sourceVersion,
				targetVersion,
				err.Field,
				string(err.Type),
			)
		}
		
		// Add warnings
		if len(dryRunResult.ValidationResult.Warnings) > 0 {
			errMsg += "\nWarnings:\n"
			for _, warning := range dryRunResult.ValidationResult.Warnings {
				errMsg += fmt.Sprintf("- %s\n", warning)
			}
		}
		
		// Add recommendations
		if len(dryRunResult.Recommendations) > 0 {
			errMsg += "\nRecommendations:\n"
			for _, rec := range dryRunResult.Recommendations {
				errMsg += fmt.Sprintf("- %s\n", rec)
			}
		}
		
		return nil, fmt.Errorf(errMsg)
	}
	
	// Log validation metrics
	w.logValidationMetrics(dryRunResult.ValidationResult, sourceVersion, targetVersion)
	
	// If this is a dry-run request, return the would-be converted object
	if isDryRun {
		// Extract converted object from dry-run result
		var converted runtime.Object
		if err := json.Unmarshal(dryRunResult.Converted, &converted); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dry-run result: %w", err)
		}
		return converted, nil
	}
	
	// Preserve data before conversion
	preservedData, err := w.dataPreserver.PreserveData(ctx, obj, targetVersion)
	if err != nil {
		w.logger.Error(err, "Failed to preserve data before conversion")
		// Continue with conversion but log the error
	}
	
	// Perform actual conversion using the existing conversion logic
	var converted runtime.Object
	var convErr error
	
	switch sourceVersion {
	case "v1alpha1":
		if targetVersion == "v1beta1" {
			// v1alpha1 to v1beta1
			src := obj.(*v1alpha1.ObservabilityPlatform)
			dst := &ObservabilityPlatform{}
			convErr = src.ConvertTo(dst)
			converted = dst
		}
		
	case "v1beta1":
		if targetVersion == "v1alpha1" {
			// v1beta1 to v1alpha1
			src := obj.(*ObservabilityPlatform)
			dst := &v1alpha1.ObservabilityPlatform{}
			convErr = dst.ConvertFrom(src)
			converted = dst
		}
		
	default:
		return nil, fmt.Errorf("unsupported source version: %s", sourceVersion)
	}
	
	if convErr != nil {
		return nil, fmt.Errorf("conversion failed: %w", convErr)
	}
	
	// Restore preserved data after conversion
	if preservedData != nil {
		if err := w.dataPreserver.RestoreData(ctx, converted, preservedData); err != nil {
			w.logger.Error(err, "Failed to restore preserved data after conversion")
			// Continue but log the error
		}
	}
	
	// Set the target version in the converted object
	converted.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "observability.io",
		Version: targetVersion,
		Kind:    "ObservabilityPlatform",
	})
	
	// Log successful conversion
	w.logger.Info("Successfully converted object",
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"from", sourceVersion,
		"to", targetVersion)
	
	return converted, nil
}

// logValidationMetrics logs validation metrics for monitoring
func (w *ConversionWebhook) logValidationMetrics(result *conversion.ValidationResult, sourceVersion, targetVersion string) {
	// Report metrics
	reporter := conversion.NewMetricsReporter(sourceVersion, targetVersion)
	reporter.ReportValidationResult(result)
	
	// Log summary
	w.logger.V(1).Info("Conversion validation complete",
		"fieldsValidated", result.Metrics.FieldsValidated,
		"errors", result.Metrics.FieldsWithErrors,
		"warnings", result.Metrics.FieldsWithWarnings,
		"deprecatedFields", result.Metrics.DeprecatedFields,
		"dataLossFields", result.Metrics.DataLossFields,
		"enhancedFields", result.Metrics.EnhancedFields,
	)
}

// SetupWebhookWithManager sets up the conversion webhook
func (w *ConversionWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&ObservabilityPlatform{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

// Default implements webhook.Defaulter
func (w *ConversionWebhook) Default(ctx context.Context, obj runtime.Object) error {
	// Defaulting is handled in the regular webhook
	return nil
}

// ValidateCreate implements webhook.Validator
func (w *ConversionWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	// Validation is handled in the regular webhook
	return nil
}

// ValidateUpdate implements webhook.Validator
func (w *ConversionWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	// Validation is handled in the regular webhook
	return nil
}

// ValidateDelete implements webhook.Validator
func (w *ConversionWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	// No validation needed for delete
	return nil
}

// ConversionReviewHandler creates an HTTP handler for conversion webhook
func ConversionReviewHandler(w *ConversionWebhook) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var body []byte
		if req.Body != nil {
			defer req.Body.Close()
			body, _ = ioutil.ReadAll(req.Body)
		}
		
		// Create admission request
		admissionReq := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UID:    types.UID(req.Header.Get("X-Request-UID")),
				Object: runtime.RawExtension{Raw: body},
			},
		}
		
		// Check for dry-run header
		if dryRun := req.Header.Get("X-Dry-Run"); dryRun == "true" {
			dryRunBool := true
			admissionReq.DryRun = &dryRunBool
		}
		
		// Handle the conversion
		resp := w.Handle(req.Context(), admissionReq)
		
		// Write response
		rw.Header().Set("Content-Type", "application/json")
		if resp.Allowed {
			rw.WriteHeader(http.StatusOK)
		} else {
			rw.WriteHeader(http.StatusBadRequest)
		}
		
		json.NewEncoder(rw).Encode(resp)
	})
}
