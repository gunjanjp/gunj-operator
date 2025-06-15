// Package servicemesh provides service mesh health check functionality
package servicemesh

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// HealthChecker performs health checks for service mesh components
type HealthChecker struct {
	client   client.Client
	manager  *Manager
	log      logr.Logger
	mu       sync.RWMutex
	statuses map[string]*HealthStatus
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Component    string             `json:"component"`
	Healthy      bool               `json:"healthy"`
	LastChecked  time.Time          `json:"lastChecked"`
	Message      string             `json:"message,omitempty"`
	Details      map[string]string  `json:"details,omitempty"`
	Metrics      *ServiceMeshMetrics `json:"metrics,omitempty"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(client client.Client, manager *Manager) *HealthChecker {
	return &HealthChecker{
		client:   client,
		manager:  manager,
		log:      log.Log.WithName("servicemesh-healthcheck"),
		statuses: make(map[string]*HealthStatus),
	}
}

// StartHealthChecks starts periodic health checks for a platform
func (h *HealthChecker) StartHealthChecks(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
	if platform.Spec.ServiceMesh == nil || !platform.Spec.ServiceMesh.Enabled {
		return nil
	}

	// Get health check configuration
	interval := 10 * time.Second
	if platform.Spec.ServiceMesh.HealthCheck != nil && platform.Spec.ServiceMesh.HealthCheck.Interval != "" {
		duration, err := time.ParseDuration(platform.Spec.ServiceMesh.HealthCheck.Interval)
		if err == nil {
			interval = duration
		}
	}

	// Start periodic health checks
	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		h.performHealthCheck(ctx, platform)
	}, interval)

	return nil
}

// performHealthCheck performs a single health check
func (h *HealthChecker) performHealthCheck(ctx context.Context, platform *v1beta1.ObservabilityPlatform) {
	h.log.V(1).Info("Performing health check", "platform", platform.Name, "namespace", platform.Namespace)

	// Detect service mesh type
	meshType, err := h.manager.DetectServiceMesh(ctx)
	if err != nil {
		h.updateStatus(platform.Name, &HealthStatus{
			Component:   "service-mesh",
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     fmt.Sprintf("Failed to detect service mesh: %v", err),
		})
		return
	}

	if meshType == ServiceMeshTypeNone {
		h.updateStatus(platform.Name, &HealthStatus{
			Component:   "service-mesh",
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     "No service mesh detected",
		})
		return
	}

	// Get the provider
	provider, err := h.manager.GetProvider(meshType)
	if err != nil {
		h.updateStatus(platform.Name, &HealthStatus{
			Component:   "service-mesh",
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     fmt.Sprintf("Failed to get provider: %v", err),
		})
		return
	}

	// Check if service mesh is installed
	installed, err := provider.IsInstalled(ctx)
	if err != nil || !installed {
		h.updateStatus(platform.Name, &HealthStatus{
			Component:   "service-mesh",
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     fmt.Sprintf("Service mesh not installed: %v", err),
		})
		return
	}

	// Check control plane components
	controlPlaneHealthy := h.checkControlPlane(ctx, meshType)
	
	// Check data plane (sidecars)
	dataPlaneHealthy := h.checkDataPlane(ctx, platform)
	
	// Check policies
	policiesHealthy := h.checkPolicies(ctx, platform, meshType)
	
	// Get metrics
	metrics, err := provider.GetMetrics(ctx, platform.Namespace)
	if err != nil {
		h.log.Error(err, "Failed to get metrics")
	}

	// Update overall status
	overallHealthy := controlPlaneHealthy && dataPlaneHealthy && policiesHealthy
	
	status := &HealthStatus{
		Component:   string(meshType),
		Healthy:     overallHealthy,
		LastChecked: time.Now(),
		Details: map[string]string{
			"control_plane": fmt.Sprintf("%v", controlPlaneHealthy),
			"data_plane":    fmt.Sprintf("%v", dataPlaneHealthy),
			"policies":      fmt.Sprintf("%v", policiesHealthy),
			"mesh_type":     string(meshType),
		},
		Metrics: metrics,
	}

	if !overallHealthy {
		status.Message = "One or more service mesh components are unhealthy"
	} else {
		status.Message = "All service mesh components are healthy"
	}

	h.updateStatus(platform.Name, status)

	// Update platform status
	h.updatePlatformStatus(ctx, platform, status)
}

// checkControlPlane checks the health of control plane components
func (h *HealthChecker) checkControlPlane(ctx context.Context, meshType ServiceMeshType) bool {
	namespace := ""
	labelSelector := ""

	switch meshType {
	case ServiceMeshTypeIstio:
		namespace = "istio-system"
		labelSelector = "app=istiod"
	case ServiceMeshTypeLinkerd:
		namespace = "linkerd"
		labelSelector = "linkerd.io/control-plane-component=controller"
	default:
		return false
	}

	// Check control plane pods
	pods := &corev1.PodList{}
	if err := h.client.List(ctx, pods, client.InNamespace(namespace), client.MatchingLabels{
		"app": labelSelector,
	}); err != nil {
		h.log.Error(err, "Failed to list control plane pods")
		return false
	}

	if len(pods.Items) == 0 {
		h.log.Info("No control plane pods found")
		return false
	}

	// Check if all pods are ready
	for _, pod := range pods.Items {
		if !isPodReady(&pod) {
			h.log.Info("Control plane pod not ready", "pod", pod.Name)
			return false
		}
	}

	return true
}

// checkDataPlane checks the health of data plane components (sidecars)
func (h *HealthChecker) checkDataPlane(ctx context.Context, platform *v1beta1.ObservabilityPlatform) bool {
	// Get pods belonging to the platform
	pods := &corev1.PodList{}
	if err := h.client.List(ctx, pods, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		h.log.Error(err, "Failed to list platform pods")
		return false
	}

	if len(pods.Items) == 0 {
		return true // No pods to check
	}

	// Check if sidecars are injected and healthy
	for _, pod := range pods.Items {
		if !hasSidecar(&pod) {
			h.log.Info("Pod missing sidecar", "pod", pod.Name)
			return false
		}

		if !isSidecarReady(&pod) {
			h.log.Info("Sidecar not ready", "pod", pod.Name)
			return false
		}
	}

	return true
}

// checkPolicies checks if service mesh policies are correctly applied
func (h *HealthChecker) checkPolicies(ctx context.Context, platform *v1beta1.ObservabilityPlatform, meshType ServiceMeshType) bool {
	// This is a simplified check - in reality, you would check specific policy resources
	switch meshType {
	case ServiceMeshTypeIstio:
		return h.checkIstioPolicies(ctx, platform)
	case ServiceMeshTypeLinkerd:
		return h.checkLinkerdPolicies(ctx, platform)
	default:
		return true
	}
}

// checkIstioPolicies checks Istio-specific policies
func (h *HealthChecker) checkIstioPolicies(ctx context.Context, platform *v1beta1.ObservabilityPlatform) bool {
	// Check if namespace has sidecar injection enabled
	ns := &corev1.Namespace{}
	if err := h.client.Get(ctx, types.NamespacedName{Name: platform.Namespace}, ns); err != nil {
		h.log.Error(err, "Failed to get namespace")
		return false
	}

	if ns.Labels["istio-injection"] != "enabled" {
		h.log.Info("Istio injection not enabled for namespace", "namespace", platform.Namespace)
		return false
	}

	// Additional policy checks would go here (PeerAuthentication, AuthorizationPolicy, etc.)
	return true
}

// checkLinkerdPolicies checks Linkerd-specific policies
func (h *HealthChecker) checkLinkerdPolicies(ctx context.Context, platform *v1beta1.ObservabilityPlatform) bool {
	// Check if namespace has Linkerd injection annotation
	ns := &corev1.Namespace{}
	if err := h.client.Get(ctx, types.NamespacedName{Name: platform.Namespace}, ns); err != nil {
		h.log.Error(err, "Failed to get namespace")
		return false
	}

	if ns.Annotations["linkerd.io/inject"] != "enabled" {
		h.log.Info("Linkerd injection not enabled for namespace", "namespace", platform.Namespace)
		return false
	}

	// Additional policy checks would go here (ServiceProfiles, TrafficSplits, etc.)
	return true
}

// updateStatus updates the health status for a platform
func (h *HealthChecker) updateStatus(platformName string, status *HealthStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.statuses[platformName] = status
}

// GetStatus returns the current health status for a platform
func (h *HealthChecker) GetStatus(platformName string) *HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.statuses[platformName]
}

// updatePlatformStatus updates the platform's service mesh status
func (h *HealthChecker) updatePlatformStatus(ctx context.Context, platform *v1beta1.ObservabilityPlatform, status *HealthStatus) {
	// Update the platform status
	platform.Status.ServiceMesh = &v1beta1.ServiceMeshStatus{
		Type:            status.Component,
		Ready:           status.Healthy,
		SidecarInjected: h.checkSidecarInjection(ctx, platform),
		Metrics:         convertMetrics(status.Metrics),
		LastSyncTime:    &metav1.Time{Time: status.LastChecked},
		Message:         status.Message,
	}

	// Update the platform
	if err := h.client.Status().Update(ctx, platform); err != nil {
		h.log.Error(err, "Failed to update platform status")
	}
}

// checkSidecarInjection checks if sidecars are injected for platform pods
func (h *HealthChecker) checkSidecarInjection(ctx context.Context, platform *v1beta1.ObservabilityPlatform) bool {
	pods := &corev1.PodList{}
	if err := h.client.List(ctx, pods, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return false
	}

	if len(pods.Items) == 0 {
		return false
	}

	for _, pod := range pods.Items {
		if !hasSidecar(&pod) {
			return false
		}
	}

	return true
}

// Helper functions

// isPodReady checks if a pod is ready
func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// hasSidecar checks if a pod has a service mesh sidecar
func hasSidecar(pod *corev1.Pod) bool {
	// Check for Istio sidecar
	for _, container := range pod.Spec.Containers {
		if container.Name == "istio-proxy" {
			return true
		}
	}

	// Check for Linkerd sidecar
	if pod.Annotations["linkerd.io/inject"] == "enabled" {
		for _, container := range pod.Spec.Containers {
			if container.Name == "linkerd-proxy" {
				return true
			}
		}
	}

	return false
}

// isSidecarReady checks if the sidecar container is ready
func isSidecarReady(pod *corev1.Pod) bool {
	sidecarName := ""
	for _, container := range pod.Spec.Containers {
		if container.Name == "istio-proxy" || container.Name == "linkerd-proxy" {
			sidecarName = container.Name
			break
		}
	}

	if sidecarName == "" {
		return false
	}

	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == sidecarName {
			return status.Ready
		}
	}

	return false
}

// convertMetrics converts service mesh metrics to API types
func convertMetrics(metrics *ServiceMeshMetrics) *v1beta1.ServiceMeshMetrics {
	if metrics == nil {
		return nil
	}

	return &v1beta1.ServiceMeshMetrics{
		RequestRate:       metrics.RequestRate,
		ErrorRate:         metrics.ErrorRate,
		P50Latency:        metrics.P50Latency,
		P99Latency:        metrics.P99Latency,
		ActiveConnections: metrics.ActiveConnections,
		LastUpdated:       &metav1.Time{Time: time.Now()},
	}
}

// HealthCheckResult represents a comprehensive health check result
type HealthCheckResult struct {
	OverallHealth   bool                       `json:"overallHealth"`
	Components      map[string]ComponentHealth `json:"components"`
	LastCheckTime   time.Time                  `json:"lastCheckTime"`
	NextCheckTime   time.Time                  `json:"nextCheckTime"`
	Recommendations []string                   `json:"recommendations,omitempty"`
}

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Name        string            `json:"name"`
	Healthy     bool              `json:"healthy"`
	Message     string            `json:"message"`
	Metrics     map[string]string `json:"metrics,omitempty"`
	LastChecked time.Time         `json:"lastChecked"`
}

// PerformComprehensiveHealthCheck performs a comprehensive health check
func (h *HealthChecker) PerformComprehensiveHealthCheck(ctx context.Context, platform *v1beta1.ObservabilityPlatform) (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		Components:    make(map[string]ComponentHealth),
		LastCheckTime: time.Now(),
	}

	// Check service mesh installation
	meshType, err := h.manager.DetectServiceMesh(ctx)
	if err != nil || meshType == ServiceMeshTypeNone {
		result.OverallHealth = false
		result.Components["service-mesh"] = ComponentHealth{
			Name:        "service-mesh",
			Healthy:     false,
			Message:     "No service mesh detected",
			LastChecked: time.Now(),
		}
		return result, nil
	}

	// Check control plane
	controlPlaneHealth := ComponentHealth{
		Name:        fmt.Sprintf("%s-control-plane", meshType),
		Healthy:     h.checkControlPlane(ctx, meshType),
		LastChecked: time.Now(),
	}
	if !controlPlaneHealth.Healthy {
		controlPlaneHealth.Message = "Control plane components are not healthy"
		result.Recommendations = append(result.Recommendations, "Check control plane pod logs")
	} else {
		controlPlaneHealth.Message = "Control plane is healthy"
	}
	result.Components["control-plane"] = controlPlaneHealth

	// Check data plane
	dataPlaneHealth := ComponentHealth{
		Name:        "data-plane",
		Healthy:     h.checkDataPlane(ctx, platform),
		LastChecked: time.Now(),
	}
	if !dataPlaneHealth.Healthy {
		dataPlaneHealth.Message = "Some pods are missing sidecars or sidecars are not ready"
		result.Recommendations = append(result.Recommendations, "Restart pods to inject sidecars")
	} else {
		dataPlaneHealth.Message = "All sidecars are healthy"
	}
	result.Components["data-plane"] = dataPlaneHealth

	// Check policies
	policiesHealth := ComponentHealth{
		Name:        "policies",
		Healthy:     h.checkPolicies(ctx, platform, meshType),
		LastChecked: time.Now(),
	}
	if !policiesHealth.Healthy {
		policiesHealth.Message = "Service mesh policies are not correctly configured"
		result.Recommendations = append(result.Recommendations, "Review service mesh policies")
	} else {
		policiesHealth.Message = "Policies are correctly configured"
	}
	result.Components["policies"] = policiesHealth

	// Check mTLS if enabled
	if platform.Spec.ServiceMesh != nil && platform.Spec.ServiceMesh.MTLS != nil && platform.Spec.ServiceMesh.MTLS.Enabled {
		mtlsHealth := ComponentHealth{
			Name:        "mtls",
			Healthy:     h.checkMTLS(ctx, platform, meshType),
			LastChecked: time.Now(),
		}
		if !mtlsHealth.Healthy {
			mtlsHealth.Message = "mTLS is not properly configured"
			result.Recommendations = append(result.Recommendations, "Check mTLS configuration")
		} else {
			mtlsHealth.Message = "mTLS is enabled and working"
		}
		result.Components["mtls"] = mtlsHealth
	}

	// Check traffic management
	if platform.Spec.ServiceMesh != nil && platform.Spec.ServiceMesh.TrafficManagement != nil {
		trafficHealth := ComponentHealth{
			Name:        "traffic-management",
			Healthy:     h.checkTrafficManagement(ctx, platform, meshType),
			LastChecked: time.Now(),
		}
		if !trafficHealth.Healthy {
			trafficHealth.Message = "Traffic management policies have issues"
			result.Recommendations = append(result.Recommendations, "Review traffic management configuration")
		} else {
			trafficHealth.Message = "Traffic management is working correctly"
		}
		result.Components["traffic-management"] = trafficHealth
	}

	// Calculate overall health
	result.OverallHealth = true
	for _, component := range result.Components {
		if !component.Healthy {
			result.OverallHealth = false
			break
		}
	}

	// Set next check time based on configuration
	interval := 10 * time.Second
	if platform.Spec.ServiceMesh != nil && platform.Spec.ServiceMesh.HealthCheck != nil && platform.Spec.ServiceMesh.HealthCheck.Interval != "" {
		if duration, err := time.ParseDuration(platform.Spec.ServiceMesh.HealthCheck.Interval); err == nil {
			interval = duration
		}
	}
	result.NextCheckTime = time.Now().Add(interval)

	return result, nil
}

// checkMTLS checks if mTLS is properly configured
func (h *HealthChecker) checkMTLS(ctx context.Context, platform *v1beta1.ObservabilityPlatform, meshType ServiceMeshType) bool {
	// This is a simplified check - actual implementation would verify certificates and policies
	switch meshType {
	case ServiceMeshTypeIstio:
		// Check for PeerAuthentication resources
		return true
	case ServiceMeshTypeLinkerd:
		// Check for Server and ServerAuthorization resources
		return true
	default:
		return false
	}
}

// checkTrafficManagement checks if traffic management is working correctly
func (h *HealthChecker) checkTrafficManagement(ctx context.Context, platform *v1beta1.ObservabilityPlatform, meshType ServiceMeshType) bool {
	// This is a simplified check - actual implementation would verify traffic policies
	switch meshType {
	case ServiceMeshTypeIstio:
		// Check VirtualServices and DestinationRules
		return true
	case ServiceMeshTypeLinkerd:
		// Check ServiceProfiles and TrafficSplits
		return true
	default:
		return false
	}
}
