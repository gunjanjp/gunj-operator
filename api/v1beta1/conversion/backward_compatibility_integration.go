/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var integrationLog = ctrl.Log.WithName("backward-compatibility-integration")

// BackwardCompatibilityWebhook integrates all backward compatibility features
type BackwardCompatibilityWebhook struct {
	Client                client.Client
	Scheme                *runtime.Scheme
	CompatibilityManager  *BackwardCompatibilityManager
	VersionDiscovery      *VersionDiscovery
	LegacyClientSupport   *LegacyClientSupport
	VersionNegotiator     *VersionNegotiator
}

// NewBackwardCompatibilityWebhook creates a new integrated backward compatibility webhook
func NewBackwardCompatibilityWebhook(client client.Client, scheme *runtime.Scheme) *BackwardCompatibilityWebhook {
	bcm := NewBackwardCompatibilityManager(client, scheme)
	vd := NewVersionDiscovery()
	lcs := NewLegacyClientSupport()
	vn := NewVersionNegotiator(vd)

	return &BackwardCompatibilityWebhook{
		Client:               client,
		Scheme:               scheme,
		CompatibilityManager: bcm,
		VersionDiscovery:     vd,
		LegacyClientSupport:  lcs,
		VersionNegotiator:    vn,
	}
}

// Handle processes incoming requests with full backward compatibility support
func (bcw *BackwardCompatibilityWebhook) Handle(ctx context.Context, req webhook.Request) webhook.Response {
	integrationLog.V(1).Info("Processing request with backward compatibility",
		"namespace", req.Namespace,
		"name", req.Name,
		"operation", req.Operation,
		"userAgent", req.UserInfo.Username)

	// Create HTTP request from webhook request for client detection
	httpReq := bcw.createHTTPRequest(req)

	// 1. Detect client version
	clientVersion := bcw.LegacyClientSupport.DetectClient(httpReq)
	integrationLog.V(1).Info("Detected client", "version", clientVersion.String())

	// 2. Negotiate API version
	acceptVersions := bcw.extractAcceptVersions(req)
	negotiatedVersion, err := bcw.VersionNegotiator.NegotiateVersion(
		req.UserInfo.UID,
		req.UserInfo.Username,
		acceptVersions,
	)
	if err != nil {
		integrationLog.Error(err, "Failed to negotiate version")
		negotiatedVersion = "v1alpha1" // Fallback to most compatible
	}

	// 3. Process the request based on client capabilities
	response := webhook.Response{Allowed: true}
	
	switch req.Operation {
	case "CREATE", "UPDATE":
		response = bcw.handleMutation(ctx, req, clientVersion, negotiatedVersion)
	case "CONVERT":
		response = bcw.handleConversion(ctx, req, clientVersion, negotiatedVersion)
	default:
		// For other operations, just allow
		response.Allowed = true
	}

	// 4. Add compatibility headers to response
	if response.AdmissionResponse.Result != nil && response.AdmissionResponse.Result.Metadata == nil {
		response.AdmissionResponse.Result.Metadata = &runtime.RawExtension{}
	}
	
	headers := make(map[string]string)
	bcw.CompatibilityManager.AddCompatibilityHeaders(
		headers,
		clientVersion.String(),
		negotiatedVersion,
	)
	
	// Add headers as warnings (webhook responses don't have headers)
	for key, value := range headers {
		response.Warnings = append(response.Warnings, fmt.Sprintf("%s: %s", key, value))
	}

	return response
}

// handleMutation handles CREATE and UPDATE operations with backward compatibility
func (bcw *BackwardCompatibilityWebhook) handleMutation(
	ctx context.Context,
	req webhook.Request,
	clientVersion ClientVersion,
	negotiatedVersion string,
) webhook.Response {
	// Decode the object
	obj, err := bcw.decodeObject(req.Object.Raw)
	if err != nil {
		return webhook.Errored(http.StatusBadRequest, err)
	}

	// Handle unknown fields
	obj, unknownFields, err := bcw.CompatibilityManager.HandleUnknownFields(obj, negotiatedVersion)
	if err != nil {
		integrationLog.Error(err, "Failed to handle unknown fields")
	}
	
	if len(unknownFields) > 0 {
		integrationLog.Info("Preserved unknown fields", "count", len(unknownFields))
	}

	// Apply default values for missing fields
	if err := bcw.CompatibilityManager.ApplyDefaultValues(obj, negotiatedVersion); err != nil {
		integrationLog.Error(err, "Failed to apply default values")
	}

	// Check feature compatibility
	if platform, ok := obj.(*v1beta1.ObservabilityPlatform); ok {
		bcw.checkFeatureCompatibility(platform, clientVersion)
	}

	// Translate any errors for legacy clients
	if !clientVersion.SupportsFeature("detailed-errors") {
		// Simplify error messages for old clients
		bcw.simplifyValidationErrors(&req.AdmissionRequest)
	}

	// Create patched response
	marshaledObj, err := json.Marshal(obj)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, err)
	}

	return webhook.PatchResponseFromRaw(req.Object.Raw, marshaledObj)
}

// handleConversion handles conversion requests with backward compatibility
func (bcw *BackwardCompatibilityWebhook) handleConversion(
	ctx context.Context,
	req webhook.Request,
	clientVersion ClientVersion,
	negotiatedVersion string,
) webhook.Response {
	integrationLog.V(1).Info("Handling conversion",
		"fromVersion", req.OldObject.Raw,
		"toVersion", negotiatedVersion,
		"clientVersion", clientVersion.String())

	// Get migration path
	fromVersion := bcw.extractVersionFromObject(req.OldObject.Raw)
	migrationPath, err := bcw.VersionDiscovery.GetMigrationPath(fromVersion, negotiatedVersion)
	if err != nil {
		return webhook.Errored(http.StatusBadRequest, fmt.Errorf("no migration path: %w", err))
	}

	// Apply migrations
	obj, err := bcw.decodeObject(req.OldObject.Raw)
	if err != nil {
		return webhook.Errored(http.StatusBadRequest, err)
	}

	for _, transition := range migrationPath {
		integrationLog.V(2).Info("Applying transition", "from", transition.From, "to", transition.To)
		// Apply transition logic here
	}

	// Serialize for the specific client
	data, err := bcw.CompatibilityManager.SerializeForClient(obj, clientVersion.String())
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, err)
	}

	return webhook.Response{
		Allowed: true,
		Result: &runtime.RawExtension{
			Raw: data,
		},
	}
}

// Helper methods

func (bcw *BackwardCompatibilityWebhook) createHTTPRequest(req webhook.Request) *http.Request {
	httpReq, _ := http.NewRequest("POST", "/webhook", nil)
	
	// Add headers from webhook request
	if req.UserInfo.Username != "" {
		httpReq.Header.Set("User-Agent", req.UserInfo.Username)
	}
	
	// Add any extra info as headers
	for key, values := range req.UserInfo.Extra {
		if len(values) > 0 {
			httpReq.Header.Set("X-"+key, values[0])
		}
	}
	
	return httpReq
}

func (bcw *BackwardCompatibilityWebhook) extractAcceptVersions(req webhook.Request) []string {
	// Extract from user info or default
	if versions, ok := req.UserInfo.Extra["accept-versions"]; ok {
		return versions
	}
	return []string{"v1beta1", "v1alpha1"}
}

func (bcw *BackwardCompatibilityWebhook) decodeObject(raw []byte) (runtime.Object, error) {
	// Decode the raw object
	obj := &v1beta1.ObservabilityPlatform{}
	if err := json.Unmarshal(raw, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (bcw *BackwardCompatibilityWebhook) extractVersionFromObject(raw []byte) string {
	var meta struct {
		APIVersion string `json:"apiVersion"`
	}
	
	if err := json.Unmarshal(raw, &meta); err != nil {
		return "v1alpha1" // Default
	}
	
	parts := strings.Split(meta.APIVersion, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	
	return "v1alpha1"
}

func (bcw *BackwardCompatibilityWebhook) checkFeatureCompatibility(
	platform *v1beta1.ObservabilityPlatform,
	clientVersion ClientVersion,
) {
	// Check multi-cluster feature
	if platform.Spec.MultiCluster != nil && platform.Spec.MultiCluster.Enabled {
		compatible, degraded := bcw.CompatibilityManager.CheckFeatureCompatibility(
			"multiCluster",
			clientVersion.String(),
		)
		
		if !compatible {
			integrationLog.Info("Multi-cluster feature not compatible with client",
				"clientVersion", clientVersion.String())
			
			// Apply degradation
			if degraded != nil {
				platform.Spec.MultiCluster.Enabled = false
				if msg, ok := degraded.(map[string]interface{})["message"].(string); ok {
					if platform.Annotations == nil {
						platform.Annotations = make(map[string]string)
					}
					platform.Annotations["observability.io/feature-degradation"] = msg
				}
			}
		}
	}
	
	// Check cost optimization feature
	if platform.Spec.CostOptimization != nil && platform.Spec.CostOptimization.Enabled {
		compatible, _ := bcw.CompatibilityManager.CheckFeatureCompatibility(
			"costOptimization",
			clientVersion.String(),
		)
		
		if !compatible {
			platform.Spec.CostOptimization.Enabled = false
		}
	}
}

func (bcw *BackwardCompatibilityWebhook) simplifyValidationErrors(req *webhook.AdmissionRequest) {
	// Simplify error messages for legacy clients
	// This would be implemented based on specific error formats
}

// SetupWebhookWithManager sets up the webhook with the controller manager
func (bcw *BackwardCompatibilityWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// Register conversion webhook
	mgr.GetWebhookServer().Register(
		"/convert",
		&webhook.Admission{Handler: bcw},
	)
	
	// Register mutating webhook
	mgr.GetWebhookServer().Register(
		"/mutate",
		&webhook.Admission{Handler: bcw},
	)
	
	// Register version discovery endpoint
	mgr.GetWebhookServer().Register(
		"/api/versions",
		http.HandlerFunc(bcw.VersionDiscovery.HandleVersionDiscoveryRequest),
	)
	
	return nil
}

// Example usage in main.go or webhook setup:
/*
func main() {
	// ... setup code ...
	
	// Create backward compatibility webhook
	bcWebhook := conversion.NewBackwardCompatibilityWebhook(mgr.GetClient(), mgr.GetScheme())
	
	// Setup with manager
	if err := bcWebhook.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup backward compatibility webhook")
		os.Exit(1)
	}
	
	// ... rest of setup ...
}
*/

// HTTPMiddleware provides HTTP middleware for backward compatibility
func (bcw *BackwardCompatibilityWebhook) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Detect client
		clientVersion := bcw.LegacyClientSupport.DetectClient(r)
		
		// Add client info to context
		ctx := context.WithValue(r.Context(), "clientVersion", clientVersion)
		
		// Handle legacy request
		transformedReq, err := bcw.LegacyClientSupport.HandleLegacyRequest(ctx, r, clientVersion)
		if err != nil {
			integrationLog.Error(err, "Failed to handle legacy request")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		
		// Create response wrapper to capture and adapt response
		rw := &responseWrapper{
			ResponseWriter: w,
			clientVersion:  clientVersion,
			bcw:            bcw,
		}
		
		// Call next handler with transformed request
		next.ServeHTTP(rw, transformedReq.WithContext(ctx))
	})
}

// responseWrapper wraps http.ResponseWriter to adapt responses
type responseWrapper struct {
	http.ResponseWriter
	clientVersion ClientVersion
	bcw           *BackwardCompatibilityWebhook
	statusCode    int
	body          []byte
}

func (rw *responseWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	
	// Add compatibility headers
	headers := make(map[string]string)
	rw.bcw.CompatibilityManager.AddCompatibilityHeaders(
		headers,
		rw.clientVersion.String(),
		rw.clientVersion.APIVersion(),
	)
	
	for key, value := range headers {
		rw.Header().Set(key, value)
	}
	
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWrapper) Write(data []byte) (int, error) {
	// For error responses, translate the error
	if rw.statusCode >= 400 {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(data, &errorResp); err == nil {
			// Translate error for legacy client
			if errMsg, ok := errorResp["error"].(string); ok {
				translatedErr := rw.bcw.LegacyClientSupport.TranslateError(
					fmt.Errorf(errMsg),
					rw.clientVersion,
				)
				errorResp["error"] = translatedErr.Error()
				data, _ = json.Marshal(errorResp)
			}
		}
	}
	
	// Adapt response for legacy client
	var response interface{}
	if err := json.Unmarshal(data, &response); err == nil {
		adapted, err := rw.bcw.LegacyClientSupport.AdaptResponse(response, rw.clientVersion)
		if err == nil {
			data, _ = json.Marshal(adapted)
		}
	}
	
	return rw.ResponseWriter.Write(data)
}
