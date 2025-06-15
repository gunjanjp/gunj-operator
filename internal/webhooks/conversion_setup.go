/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package webhooks

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion"
)

// SetupConversionWebhook sets up the conversion webhook with enhanced validation
func SetupConversionWebhook(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("conversion-webhook-setup")
	
	// Create conversion webhook
	conversionWebhook := observabilityv1beta1.NewConversionWebhook(mgr.GetClient(), mgr.GetScheme())
	
	// Register webhook paths
	mgr.GetWebhookServer().Register(
		"/convert",
		observabilityv1beta1.ConversionReviewHandler(conversionWebhook),
	)
	
	// Register dry-run endpoint
	mgr.GetWebhookServer().Register(
		"/convert-dryrun",
		&webhook.Admission{Handler: &DryRunHandler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Logger:    logger,
			Converter: conversion.NewDryRunConverter(logger),
		}},
	)
	
	// Register rollback endpoint
	rollbackStore := conversion.NewInMemoryRollbackStore()
	rollbackManager := conversion.NewRollbackManager(logger, mgr.GetClient(), rollbackStore)
	
	mgr.GetWebhookServer().Register(
		"/rollback",
		&webhook.Admission{Handler: &RollbackHandler{
			Client:          mgr.GetClient(),
			RollbackManager: rollbackManager,
			Logger:          logger,
		}},
	)
	
	// Start background cleanup for old snapshots
	go startSnapshotCleanup(mgr.GetContext(), rollbackManager, logger)
	
	logger.Info("Conversion webhook setup complete")
	return nil
}

// DryRunHandler handles dry-run conversion requests
type DryRunHandler struct {
	Client    client.Client
	Scheme    *runtime.Scheme
	Logger    logr.Logger
	Converter *conversion.DryRunConverter
}

// Handle processes dry-run requests
func (h *DryRunHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	h.Logger.V(1).Info("Processing dry-run conversion request",
		"name", req.Name,
		"namespace", req.Namespace,
		"targetVersion", req.UserInfo.Extra["targetVersion"])
	
	// Extract object from request
	obj, err := h.extractObject(req)
	if err != nil {
		return webhook.Errored(400, err)
	}
	
	// Get target version
	targetVersion := ""
	if versions, ok := req.UserInfo.Extra["targetVersion"]; ok && len(versions) > 0 {
		targetVersion = versions[0]
	}
	
	if targetVersion == "" {
		return webhook.Errored(400, fmt.Errorf("targetVersion not specified"))
	}
	
	// Perform dry-run conversion
	result, err := h.Converter.DryRunConversion(ctx, obj, targetVersion)
	if err != nil {
		return webhook.Errored(500, err)
	}
	
	// Return result as JSON in response
	return webhook.Allowed("Dry-run complete").WithPatch(webhook.JSONPatchOp{
		Operation: "add",
		Path:      "/metadata/annotations/conversion-dryrun-result",
		Value:     result,
	})
}

// extractObject extracts the object from admission request
func (h *DryRunHandler) extractObject(req webhook.AdmissionRequest) (runtime.Object, error) {
	// Implementation would decode the object from req.Object.Raw
	// This is simplified for the example
	return nil, nil
}

// RollbackHandler handles rollback requests
type RollbackHandler struct {
	Client          client.Client
	RollbackManager *conversion.RollbackManager
	Logger          logr.Logger
}

// Handle processes rollback requests
func (h *RollbackHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	h.Logger.Info("Processing rollback request",
		"name", req.Name,
		"namespace", req.Namespace)
	
	// Extract rollback request from annotations
	rollbackReq, err := h.extractRollbackRequest(req)
	if err != nil {
		return webhook.Errored(400, err)
	}
	
	// Perform rollback
	result, err := h.RollbackManager.Rollback(ctx, rollbackReq)
	if err != nil {
		return webhook.Errored(500, err)
	}
	
	if !result.Success {
		return webhook.Denied(result.Error)
	}
	
	return webhook.Allowed(fmt.Sprintf("Rolled back to version %s", result.RolledBackTo))
}

// extractRollbackRequest extracts rollback request from admission request
func (h *RollbackHandler) extractRollbackRequest(req webhook.AdmissionRequest) (conversion.RollbackRequest, error) {
	// Implementation would extract from annotations or request body
	// This is simplified for the example
	return conversion.RollbackRequest{}, nil
}

// startSnapshotCleanup starts the background cleanup process
func startSnapshotCleanup(ctx context.Context, manager *conversion.RollbackManager, logger logr.Logger) {
	ticker := time.NewTicker(24 * time.Hour) // Daily cleanup
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping snapshot cleanup")
			return
		case <-ticker.C:
			logger.V(1).Info("Running snapshot cleanup")
			if err := manager.CleanupOldSnapshots(ctx, 7*24*time.Hour); err != nil {
				logger.Error(err, "Failed to cleanup old snapshots")
			}
		}
	}
}

// ConversionWebhookConfig contains configuration for the conversion webhook
type ConversionWebhookConfig struct {
	// MaxSnapshotAge is the maximum age of snapshots to keep
	MaxSnapshotAge time.Duration
	
	// EnableDryRun enables the dry-run endpoint
	EnableDryRun bool
	
	// EnableRollback enables the rollback endpoint
	EnableRollback bool
	
	// MetricsEnabled enables conversion metrics
	MetricsEnabled bool
	
	// ValidationLevel controls how strict validation is
	ValidationLevel string // strict, normal, lenient
}

// DefaultConversionWebhookConfig returns default configuration
func DefaultConversionWebhookConfig() ConversionWebhookConfig {
	return ConversionWebhookConfig{
		MaxSnapshotAge:  7 * 24 * time.Hour,
		EnableDryRun:    true,
		EnableRollback:  true,
		MetricsEnabled:  true,
		ValidationLevel: "normal",
	}
}

// SetupWithConfig sets up the conversion webhook with custom configuration
func SetupWithConfig(mgr ctrl.Manager, config ConversionWebhookConfig) error {
	logger := mgr.GetLogger().WithName("conversion-webhook-setup")
	
	// Validate config
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}
	
	// Create webhook with config
	webhook := createConversionWebhook(mgr, config, logger)
	
	// Register endpoints based on config
	if err := registerEndpoints(mgr, webhook, config); err != nil {
		return fmt.Errorf("registering endpoints: %w", err)
	}
	
	logger.Info("Conversion webhook setup complete with custom config",
		"config", config)
	
	return nil
}

// Validate validates the webhook configuration
func (c ConversionWebhookConfig) Validate() error {
	if c.MaxSnapshotAge < time.Hour {
		return fmt.Errorf("max snapshot age must be at least 1 hour")
	}
	
	validLevels := map[string]bool{
		"strict":  true,
		"normal":  true,
		"lenient": true,
	}
	
	if !validLevels[c.ValidationLevel] {
		return fmt.Errorf("invalid validation level: %s", c.ValidationLevel)
	}
	
	return nil
}

// createConversionWebhook creates a webhook with the given configuration
func createConversionWebhook(mgr ctrl.Manager, config ConversionWebhookConfig, logger logr.Logger) *observabilityv1beta1.ConversionWebhook {
	// Create webhook with configuration
	webhook := observabilityv1beta1.NewConversionWebhook(mgr.GetClient(), mgr.GetScheme())
	
	// Apply configuration
	// This would configure the webhook based on the config
	
	return webhook
}

// registerEndpoints registers webhook endpoints based on configuration
func registerEndpoints(mgr ctrl.Manager, webhook *observabilityv1beta1.ConversionWebhook, config ConversionWebhookConfig) error {
	server := mgr.GetWebhookServer()
	
	// Always register main conversion endpoint
	server.Register("/convert", observabilityv1beta1.ConversionReviewHandler(webhook))
	
	// Register optional endpoints
	if config.EnableDryRun {
		// Register dry-run endpoint
		server.Register("/convert-dryrun", createDryRunHandler(mgr, config))
	}
	
	if config.EnableRollback {
		// Register rollback endpoint
		server.Register("/rollback", createRollbackHandler(mgr, config))
	}
	
	return nil
}

// createDryRunHandler creates a dry-run handler
func createDryRunHandler(mgr ctrl.Manager, config ConversionWebhookConfig) http.Handler {
	logger := mgr.GetLogger().WithName("dryrun-handler")
	converter := conversion.NewDryRunConverter(logger)
	
	return &webhook.Admission{
		Handler: &DryRunHandler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Logger:    logger,
			Converter: converter,
		},
	}
}

// createRollbackHandler creates a rollback handler
func createRollbackHandler(mgr ctrl.Manager, config ConversionWebhookConfig) http.Handler {
	logger := mgr.GetLogger().WithName("rollback-handler")
	store := conversion.NewInMemoryRollbackStore()
	rollbackManager := conversion.NewRollbackManager(logger, mgr.GetClient(), store)
	
	// Start cleanup based on config
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		
		ctx := mgr.GetContext()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rollbackManager.CleanupOldSnapshots(ctx, config.MaxSnapshotAge)
			}
		}
	}()
	
	return &webhook.Admission{
		Handler: &RollbackHandler{
			Client:          mgr.GetClient(),
			RollbackManager: rollbackManager,
			Logger:          logger,
		},
	}
}
