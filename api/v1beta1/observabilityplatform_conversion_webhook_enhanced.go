/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admission/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion/preservation"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/migration"
)

var conversionlog = logf.Log.WithName("observabilityplatform-conversion")

// +kubebuilder:webhook:path=/convert,mutating=false,failurePolicy=fail,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1alpha1;v1beta1,name=cobservabilityplatform.kb.io,sideEffects=None,admissionReviewVersions=v1

// ConversionWebhook handles conversions with migration helpers integration
type ConversionWebhook struct {
	client          client.Client
	decoder         *admission.Decoder
	validator       *conversion.FieldValidator
	dryRun          *conversion.DryRunConverter
	rollbackManager *conversion.RollbackManager
	metrics         *conversion.ConversionMetrics
	dataPreserver   *conversion.DataPreserverEnhanced
	
	// Migration helpers
	schemaTracker   *migration.SchemaEvolutionTracker
	optimizer       *migration.ConversionOptimizer
	lifecycleMgr    *migration.LifecycleIntegrationManager
	statusReporter  *migration.MigrationStatusReporter
	
	logger          logr.Logger
}

// NewConversionWebhook creates a new conversion webhook with enhanced features
func NewConversionWebhook(client client.Client, scheme *runtime.Scheme) *ConversionWebhook {
	logger := conversionlog.WithName("webhook")
	rollbackStore := conversion.NewInMemoryRollbackStore()
	
	// Create preservation policy config
	policyConfig := &preservation.PolicyConfig{
		Policies:                preservation.DefaultPolicies(),
		DefaultStrategy:         "deep-copy",
		EnableMetrics:           true,
		EnableAlerting:          false,
		FieldSizeThreshold:      1024,
		MaxPreservationAttempts: 3,
		PreservationTimeout:     "30s",
	}
	
	// Create enhanced data preserver
	dataPreserver, err := conversion.NewDataPreserverEnhanced(logger, client, policyConfig)
	if err != nil {
		// Fallback to basic data preserver if enhanced fails
		logger.Error(err, "Failed to create enhanced data preserver, using basic version")
		dataPreserver = nil
	}
	
	return &ConversionWebhook{
		client:          client,
		validator:       conversion.NewFieldValidator(logger),
		dryRun:          conversion.NewDryRunConverter(logger),
		rollbackManager: conversion.NewRollbackManager(logger, client, rollbackStore),
		metrics:         conversion.GetMetrics(),
		dataPreserver:   dataPreserver,
		
		// Initialize migration helpers
		schemaTracker:   migration.NewSchemaEvolutionTracker(logger),
		optimizer:       migration.NewConversionOptimizer(logger),
		lifecycleMgr:    migration.NewLifecycleIntegrationManager(client, logger),
		statusReporter:  migration.NewMigrationStatusReporter(logger),
		
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
	
	// Get source and target versions
	sourceVersion := ""
	if len(req.Objects) > 0 {
		sourceVersion = req.Objects[0].Object.GetObjectKind().GroupVersionKind().Version
	}
	targetVersion := req.DesiredAPIVersion
	
	// Check migration path
	migrationPath, err := w.schemaTracker.GetMigrationPath(sourceVersion, targetVersion)
	if err != nil {
		w.logger.Error(err, "No migration path found",
			"sourceVersion", sourceVersion,
			"targetVersion", targetVersion)
		response.Result = metav1.Status{
			Status:  metav1.StatusFailure,
			Message: fmt.Sprintf("No migration path from %s to %s", sourceVersion, targetVersion),
			Reason:  metav1.StatusReasonInvalid,
		}
		return response
	}
	
	// Warn about data loss risk
	if migrationPath.DataLossRisk && !isDryRun {
		w.logger.Warn("Migration may result in data loss",
			"sourceVersion", sourceVersion,
			"targetVersion", targetVersion)
	}
	
	// Start metrics timer
	timer := w.metrics.StartTimer(sourceVersion, targetVersion, "ObservabilityPlatform")
	defer timer.Complete(true)
	
	// Process each object
	for _, obj := range req.Objects {
		converted, err := w.convertObjectWithMigration(ctx, obj.Object, targetVersion, isDryRun)
		if err != nil {
			w.logger.Error(err, "Conversion failed",
				"object", obj.Object.GetObjectKind(),
				"targetVersion", targetVersion)
			
			response.Result = metav1.Status{
				Status:  metav1.StatusFailure,
				Message: err.Error(),
				Reason:  metav1.StatusReasonInvalid,
			}
			
			// Record failure in schema tracker
			w.schemaTracker.RecordMigrationFailure(sourceVersion, targetVersion, err)
			
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
	
	// Record successful migration
	if !isDryRun {
		duration := timer.Duration()
		w.schemaTracker.RecordMigrationSuccess(sourceVersion, targetVersion, duration)
	}
	
	return response
}

// convertObjectWithMigration performs conversion with migration helpers
func (w *ConversionWebhook) convertObjectWithMigration(ctx context.Context, obj runtime.Object, targetVersion string, isDryRun bool) (runtime.Object, error) {
	// Determine source version
	sourceVersion := obj.GetObjectKind().GroupVersionKind().Version
	
	w.logger.V(1).Info("Converting object with migration helpers",
		"sourceVersion", sourceVersion,
		"targetVersion", targetVersion,
		"dryRun", isDryRun)
	
	// Get object key
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to get object key: %w", err)
	}
	
	// Track migration
	w.schemaTracker.RecordMigration(sourceVersion, targetVersion, key)
	
	// Apply optimizations
	if !isDryRun {
		if unstructuredObj, ok := obj.(*unstructured.Unstructured); ok {
			optimized, err := w.optimizer.OptimizeConversion(unstructuredObj, targetVersion)
			if err == nil && optimized != nil {
				obj = optimized
			}
		}
	}
	
	// Perform lifecycle pre-migration check
	if !isDryRun {
		if err := w.lifecycleMgr.PreMigrationCheck(ctx, key, targetVersion); err != nil {
			return nil, fmt.Errorf("pre-migration check failed: %w", err)
		}
	}
	
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
		errMsg := w.buildErrorMessage(dryRunResult, sourceVersion, targetVersion)
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
	var preservedData interface{}
	if w.dataPreserver != nil {
		enhancedData, err := w.dataPreserver.PreserveDataEnhanced(ctx, obj, targetVersion)
		if err != nil {
			w.logger.Error(err, "Failed to preserve data before conversion")
			// Continue with conversion but log the error
		} else {
			preservedData = enhancedData
		}
	}
	
	// Perform actual conversion using the existing conversion logic
	converted, err := w.performConversion(obj, sourceVersion, targetVersion)
	if err != nil {
		// Rollback on failure
		if rollbackErr := w.lifecycleMgr.RollbackMigration(ctx, key); rollbackErr != nil {
			w.logger.Error(rollbackErr, "Failed to rollback migration")
		}
		return nil, err
	}
	
	// Restore preserved data after conversion
	if preservedData != nil && w.dataPreserver != nil {
		if enhanced, ok := preservedData.(*conversion.PreservedDataEnhanced); ok {
			if err := w.dataPreserver.RestoreDataEnhanced(ctx, converted, enhanced); err != nil {
				w.logger.Error(err, "Failed to restore preserved data after conversion")
				// Continue but log the error
			}
		}
	}
	
	// Apply lifecycle hooks
	if err := w.lifecycleMgr.ApplyMigrationHooks(ctx, converted); err != nil {
		return nil, fmt.Errorf("failed to apply migration hooks: %w", err)
	}
	
	// Perform post-migration validation
	if err := w.lifecycleMgr.PostMigrationValidation(ctx, key, targetVersion); err != nil {
		// Rollback on validation failure
		if rollbackErr := w.lifecycleMgr.RollbackMigration(ctx, key); rollbackErr != nil {
			w.logger.Error(rollbackErr, "Failed to rollback after validation failure")
		}
		return nil, fmt.Errorf("post-migration validation failed: %w", err)
	}
	
	// Log successful conversion
	w.logger.Info("Successfully converted object with migration helpers",
		"name", key.Name,
		"namespace", key.Namespace,
		"from", sourceVersion,
		"to", targetVersion)
	
	return converted, nil
}

// performConversion performs the actual conversion logic
func (w *ConversionWebhook) performConversion(obj runtime.Object, sourceVersion, targetVersion string) (runtime.Object, error) {
	var converted runtime.Object
	var convErr error
	
	switch sourceVersion {
	case "v1alpha1":
		if targetVersion == "observability.io/v1beta1" {
			// v1alpha1 to v1beta1
			src := obj.(*v1alpha1.ObservabilityPlatform)
			dst := &ObservabilityPlatform{}
			convErr = src.ConvertTo(dst)
			converted = dst
		} else {
			return nil, fmt.Errorf("unsupported target version from v1alpha1: %s", targetVersion)
		}
		
	case "v1beta1":
		if targetVersion == "observability.io/v1alpha1" {
			// v1beta1 to v1alpha1
			src := obj.(*ObservabilityPlatform)
			dst := &v1alpha1.ObservabilityPlatform{}
			convErr = src.ConvertTo(dst)
			converted = dst
		} else {
			return nil, fmt.Errorf("unsupported target version from v1beta1: %s", targetVersion)
		}
		
	default:
		return nil, fmt.Errorf("unsupported source version: %s", sourceVersion)
	}
	
	if convErr != nil {
		return nil, fmt.Errorf("conversion failed: %w", convErr)
	}
	
	// Set the target version in the converted object
	gvk := schema.GroupVersionKind{
		Group:   "observability.io",
		Kind:    "ObservabilityPlatform",
	}
	
	// Extract version from target
	if targetVersion == "observability.io/v1beta1" {
		gvk.Version = "v1beta1"
	} else if targetVersion == "observability.io/v1alpha1" {
		gvk.Version = "v1alpha1"
	} else {
		gvk.Version = targetVersion
	}
	
	converted.GetObjectKind().SetGroupVersionKind(gvk)
	
	return converted, nil
}

// buildErrorMessage builds a detailed error message from dry-run results
func (w *ConversionWebhook) buildErrorMessage(result *conversion.DryRunResult, sourceVersion, targetVersion string) string {
	errMsg := "Conversion validation failed:\n"
	
	// Add validation errors
	for _, err := range result.ValidationResult.Errors {
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
	if len(result.ValidationResult.Warnings) > 0 {
		errMsg += "\nWarnings:\n"
		for _, warning := range result.ValidationResult.Warnings {
			errMsg += fmt.Sprintf("- %s\n", warning)
		}
	}
	
	// Add recommendations
	if len(result.Recommendations) > 0 {
		errMsg += "\nRecommendations:\n"
		for _, rec := range result.Recommendations {
			errMsg += fmt.Sprintf("- %s\n", rec)
		}
	}
	
	return errMsg
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
