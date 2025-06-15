// Package servicemesh provides service mesh integration for the Gunj Operator
package servicemesh

import (
	"context"
	"fmt"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceMeshIntegration handles service mesh integration for the operator
type ServiceMeshIntegration struct {
	client   client.Client
	scheme   *runtime.Scheme
	log      logr.Logger
	recorder record.EventRecorder
	factory  ServiceMeshFactory
	utils    *ServiceMeshUtils
}

// NewServiceMeshIntegration creates a new service mesh integration
func NewServiceMeshIntegration(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) *ServiceMeshIntegration {
	log := ctrl.Log.WithName("service-mesh-integration")
	
	return &ServiceMeshIntegration{
		client:   client,
		scheme:   scheme,
		log:      log,
		recorder: recorder,
		factory:  NewServiceMeshFactory(client, log),
		utils:    NewServiceMeshUtils(client, log),
	}
}

// ReconcileServiceMesh reconciles service mesh for the observability platform
func (s *ServiceMeshIntegration) ReconcileServiceMesh(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := s.log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	
	// Get service mesh configuration
	config := GetServiceMeshConfig(platform)
	
	// Check if service mesh is enabled
	if !config.Enabled {
		log.V(1).Info("Service mesh integration is disabled")
		// Update status to reflect disabled state
		return s.updateServiceMeshStatus(ctx, platform, &ServiceMeshStatus{
			Phase:              ServiceMeshPhaseReady,
			Message:            "Service mesh integration is disabled",
			LastReconcileTime:  metav1.Now(),
			ObservedGeneration: platform.Generation,
		})
	}
	
	// Validate configuration
	if err := ValidateServiceMeshConfig(config); err != nil {
		log.Error(err, "Invalid service mesh configuration")
		s.recorder.Event(platform, "Warning", "InvalidConfiguration", fmt.Sprintf("Invalid service mesh configuration: %v", err))
		return s.updateServiceMeshStatus(ctx, platform, &ServiceMeshStatus{
			Phase:              ServiceMeshPhaseFailed,
			Message:            fmt.Sprintf("Invalid configuration: %v", err),
			LastReconcileTime:  metav1.Now(),
			ObservedGeneration: platform.Generation,
		})
	}
	
	// Check if service mesh is installed
	installed, err := s.utils.IsServiceMeshInstalled(ctx, config.Provider)
	if err != nil {
		log.Error(err, "Failed to check if service mesh is installed")
		return err
	}
	
	if !installed {
		log.Info("Service mesh is not installed", "provider", config.Provider)
		s.recorder.Event(platform, "Warning", "ServiceMeshNotInstalled", 
			fmt.Sprintf("Service mesh %s is not installed in the cluster", config.Provider))
		return s.updateServiceMeshStatus(ctx, platform, &ServiceMeshStatus{
			Phase:              ServiceMeshPhaseFailed,
			Message:            fmt.Sprintf("Service mesh %s is not installed", config.Provider),
			LastReconcileTime:  metav1.Now(),
			ObservedGeneration: platform.Generation,
		})
	}
	
	// Create service mesh manager
	manager, err := s.factory.CreateManager(config.Provider)
	if err != nil {
		log.Error(err, "Failed to create service mesh manager")
		return err
	}
	
	// Initialize service mesh
	log.Info("Initializing service mesh integration")
	if err := manager.Initialize(ctx, platform); err != nil {
		log.Error(err, "Failed to initialize service mesh")
		s.recorder.Event(platform, "Warning", "InitializationFailed", 
			fmt.Sprintf("Failed to initialize service mesh: %v", err))
		return s.updateServiceMeshStatus(ctx, platform, &ServiceMeshStatus{
			Phase:              ServiceMeshPhaseFailed,
			Message:            fmt.Sprintf("Initialization failed: %v", err),
			LastReconcileTime:  metav1.Now(),
			ObservedGeneration: platform.Generation,
		})
	}
	
	// Configure service mesh
	log.Info("Configuring service mesh")
	if err := manager.Configure(ctx, platform, config); err != nil {
		log.Error(err, "Failed to configure service mesh")
		s.recorder.Event(platform, "Warning", "ConfigurationFailed", 
			fmt.Sprintf("Failed to configure service mesh: %v", err))
		return s.updateServiceMeshStatus(ctx, platform, &ServiceMeshStatus{
			Phase:              ServiceMeshPhaseFailed,
			Message:            fmt.Sprintf("Configuration failed: %v", err),
			LastReconcileTime:  metav1.Now(),
			ObservedGeneration: platform.Generation,
		})
	}
	
	// Configure mTLS if enabled
	if config.MTLS.Mode != "" {
		log.Info("Configuring mTLS", "mode", config.MTLS.Mode)
		if err := manager.ConfigureMTLS(ctx, platform, config.MTLS); err != nil {
			log.Error(err, "Failed to configure mTLS")
			s.recorder.Event(platform, "Warning", "MTLSConfigurationFailed", 
				fmt.Sprintf("Failed to configure mTLS: %v", err))
			// Don't fail the whole reconciliation for mTLS issues
		}
	}
	
	// Enable observability features
	if config.Observability.Metrics.Enabled || config.Observability.Tracing.Enabled {
		log.Info("Enabling service mesh observability")
		if err := manager.EnableObservability(ctx, platform, config.Observability); err != nil {
			log.Error(err, "Failed to enable observability")
			s.recorder.Event(platform, "Warning", "ObservabilityConfigurationFailed", 
				fmt.Sprintf("Failed to enable observability: %v", err))
			// Don't fail the whole reconciliation for observability issues
		}
	}
	
	// Apply traffic policies if configured
	if err := s.applyTrafficPolicies(ctx, platform, manager); err != nil {
		log.Error(err, "Failed to apply traffic policies")
		s.recorder.Event(platform, "Warning", "TrafficPolicyFailed", 
			fmt.Sprintf("Failed to apply traffic policies: %v", err))
		// Don't fail the whole reconciliation for traffic policy issues
	}
	
	// Perform health check
	if err := manager.HealthCheck(ctx, platform); err != nil {
		log.Error(err, "Service mesh health check failed")
		s.recorder.Event(platform, "Warning", "HealthCheckFailed", 
			fmt.Sprintf("Service mesh health check failed: %v", err))
	}
	
	// Get and update status
	status, err := manager.GetStatus(ctx, platform)
	if err != nil {
		log.Error(err, "Failed to get service mesh status")
		return err
	}
	
	// Record successful configuration
	s.recorder.Event(platform, "Normal", "ServiceMeshConfigured", 
		fmt.Sprintf("Service mesh %s configured successfully", config.Provider))
	
	return s.updateServiceMeshStatus(ctx, platform, status)
}

// DeleteServiceMesh handles service mesh cleanup during platform deletion
func (s *ServiceMeshIntegration) DeleteServiceMesh(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := s.log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	
	// Get service mesh configuration
	config := GetServiceMeshConfig(platform)
	
	// Check if service mesh was enabled
	if !config.Enabled {
		log.V(1).Info("Service mesh integration was not enabled, skipping cleanup")
		return nil
	}
	
	// Create service mesh manager
	manager, err := s.factory.CreateManager(config.Provider)
	if err != nil {
		log.Error(err, "Failed to create service mesh manager for cleanup")
		return err
	}
	
	// Cleanup service mesh resources
	log.Info("Cleaning up service mesh resources")
	if err := manager.Cleanup(ctx, platform); err != nil {
		log.Error(err, "Failed to cleanup service mesh resources")
		s.recorder.Event(platform, "Warning", "CleanupFailed", 
			fmt.Sprintf("Failed to cleanup service mesh resources: %v", err))
		return err
	}
	
	s.recorder.Event(platform, "Normal", "ServiceMeshCleanedUp", 
		fmt.Sprintf("Service mesh %s resources cleaned up", config.Provider))
	
	return nil
}

// applyTrafficPolicies applies traffic management policies
func (s *ServiceMeshIntegration) applyTrafficPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, manager ServiceMeshManager) error {
	// This will be expanded based on platform configuration
	// For now, apply basic policies for each component
	
	components := []string{"prometheus", "grafana", "loki", "tempo"}
	
	for _, component := range components {
		// Check if component is enabled
		if !s.isComponentEnabled(platform, component) {
			continue
		}
		
		// Create basic traffic policy
		policy := TrafficPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-policy", platform.Name, component),
				Namespace: platform.Namespace,
			},
			Spec: TrafficPolicySpec{
				TargetService: fmt.Sprintf("%s-%s", platform.Name, component),
			},
		}
		
		// Apply retry policy from configuration
		config := GetServiceMeshConfig(platform)
		if config.TrafficManagement.RetryPolicy.Attempts > 0 {
			policy.Spec.RetryPolicy = &config.TrafficManagement.RetryPolicy
		}
		
		// Apply timeout
		if config.TrafficManagement.Timeout > 0 {
			policy.Spec.Timeout = &config.TrafficManagement.Timeout
		}
		
		// Apply circuit breaker
		if config.TrafficManagement.CircuitBreaker.ConsecutiveErrors > 0 {
			policy.Spec.CircuitBreaker = &config.TrafficManagement.CircuitBreaker
		}
		
		// Apply load balancer
		if config.TrafficManagement.LoadBalancer.Algorithm != "" {
			policy.Spec.LoadBalancer = &config.TrafficManagement.LoadBalancer
		}
		
		// Apply the policy
		if err := manager.ApplyTrafficPolicy(ctx, platform, policy); err != nil {
			s.log.Error(err, "Failed to apply traffic policy", "component", component)
			// Continue with other components
		}
	}
	
	return nil
}

// isComponentEnabled checks if a component is enabled
func (s *ServiceMeshIntegration) isComponentEnabled(platform *observabilityv1.ObservabilityPlatform, component string) bool {
	switch component {
	case "prometheus":
		return platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled
	case "grafana":
		return platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled
	case "loki":
		return platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled
	case "tempo":
		return platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled
	default:
		return false
	}
}

// updateServiceMeshStatus updates the service mesh status in the platform
func (s *ServiceMeshIntegration) updateServiceMeshStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, status *ServiceMeshStatus) error {
	// Update the platform status with service mesh information
	platform.Status.ServiceMesh = &observabilityv1.ServiceMeshStatus{
		Phase:              status.Phase,
		Message:            status.Message,
		LastReconcileTime:  status.LastReconcileTime,
		ObservedGeneration: status.ObservedGeneration,
	}
	
	// Add conditions
	for _, condition := range status.Conditions {
		meta.SetStatusCondition(&platform.Status.Conditions, condition)
	}
	
	// Update the status
	if err := s.client.Status().Update(ctx, platform); err != nil {
		s.log.Error(err, "Failed to update service mesh status")
		return err
	}
	
	return nil
}

// EnrichPodSpec enriches a pod spec with service mesh annotations and labels
func (s *ServiceMeshIntegration) EnrichPodSpec(platform *observabilityv1.ObservabilityPlatform, podSpec *corev1.PodSpec, component string) {
	// Get service mesh configuration
	config := GetServiceMeshConfig(platform)
	
	if !config.Enabled {
		return
	}
	
	// Generate labels and annotations
	labels := GenerateServiceMeshLabels(platform, component)
	annotations := GenerateServiceMeshAnnotations(platform, component)
	
	// Apply labels
	if podSpec.Labels == nil {
		podSpec.Labels = make(map[string]string)
	}
	for k, v := range labels {
		podSpec.Labels[k] = v
	}
	
	// Apply annotations
	if podSpec.Annotations == nil {
		podSpec.Annotations = make(map[string]string)
	}
	for k, v := range annotations {
		podSpec.Annotations[k] = v
	}
	
	// Add provider-specific configurations
	switch config.Provider {
	case IstioProvider:
		s.enrichForIstio(podSpec, config, component)
	case LinkerdProvider:
		s.enrichForLinkerd(podSpec, config, component)
	}
}

// enrichForIstio adds Istio-specific configurations
func (s *ServiceMeshIntegration) enrichForIstio(podSpec *corev1.PodSpec, config ServiceMeshConfig, component string) {
	// Add Istio-specific annotations
	if podSpec.Annotations == nil {
		podSpec.Annotations = make(map[string]string)
	}
	
	// Enable sidecar injection
	podSpec.Annotations["sidecar.istio.io/inject"] = "true"
	
	// Configure proxy resources
	podSpec.Annotations["sidecar.istio.io/proxyCPU"] = "100m"
	podSpec.Annotations["sidecar.istio.io/proxyMemory"] = "128Mi"
	podSpec.Annotations["sidecar.istio.io/proxyCPULimit"] = "200m"
	podSpec.Annotations["sidecar.istio.io/proxyMemoryLimit"] = "256Mi"
	
	// Configure telemetry
	if config.Observability.Metrics.Enabled {
		podSpec.Annotations["sidecar.istio.io/statsInclusionRegexps"] = ".*"
	}
	
	// Configure access logging
	if config.Observability.AccessLog.Enabled {
		podSpec.Annotations["sidecar.istio.io/accessLogFile"] = "/dev/stdout"
		if config.Observability.AccessLog.Format != "" {
			podSpec.Annotations["sidecar.istio.io/accessLogFormat"] = config.Observability.AccessLog.Format
		}
		if config.Observability.AccessLog.Encoding == LogEncodingJSON {
			podSpec.Annotations["sidecar.istio.io/accessLogEncoding"] = "JSON"
		}
	}
}

// enrichForLinkerd adds Linkerd-specific configurations
func (s *ServiceMeshIntegration) enrichForLinkerd(podSpec *corev1.PodSpec, config ServiceMeshConfig, component string) {
	// Add Linkerd-specific annotations
	if podSpec.Annotations == nil {
		podSpec.Annotations = make(map[string]string)
	}
	
	// Enable proxy injection
	podSpec.Annotations["linkerd.io/inject"] = "enabled"
	
	// Configure proxy resources
	podSpec.Annotations["config.linkerd.io/proxy-cpu-request"] = "100m"
	podSpec.Annotations["config.linkerd.io/proxy-memory-request"] = "128Mi"
	podSpec.Annotations["config.linkerd.io/proxy-cpu-limit"] = "200m"
	podSpec.Annotations["config.linkerd.io/proxy-memory-limit"] = "256Mi"
	
	// Configure telemetry
	if config.Observability.Metrics.Enabled {
		podSpec.Annotations["config.alpha.linkerd.io/proxy-enable-gateway"] = "true"
	}
}

// GetServiceMeshCapabilities returns the capabilities provided by the service mesh
func (s *ServiceMeshIntegration) GetServiceMeshCapabilities(platform *observabilityv1.ObservabilityPlatform) ServiceMeshCapabilities {
	config := GetServiceMeshConfig(platform)
	
	if !config.Enabled {
		return ServiceMeshCapabilities{}
	}
	
	return ServiceMeshCapabilities{
		Provider:        config.Provider,
		MTLS:            config.MTLS.Mode != MTLSModeDisable,
		TrafficManagement: true,
		Observability:   config.Observability.Metrics.Enabled || config.Observability.Tracing.Enabled,
		LoadBalancing:   config.TrafficManagement.LoadBalancer.Algorithm != "",
		CircuitBreaking: config.TrafficManagement.CircuitBreaker.ConsecutiveErrors > 0,
		Retries:         config.TrafficManagement.RetryPolicy.Attempts > 0,
	}
}

// ServiceMeshCapabilities represents the capabilities provided by the service mesh
type ServiceMeshCapabilities struct {
	Provider          MeshProvider
	MTLS              bool
	TrafficManagement bool
	Observability     bool
	LoadBalancing     bool
	CircuitBreaking   bool
	Retries           bool
}
