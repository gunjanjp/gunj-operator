// Package servicemesh provides service mesh integration for the Gunj Operator
package servicemesh

import (
	"fmt"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceMeshFactory creates service mesh managers based on provider
type ServiceMeshFactory interface {
	// CreateManager creates a service mesh manager for the specified provider
	CreateManager(provider MeshProvider) (ServiceMeshManager, error)
}

// DefaultServiceMeshFactory is the default implementation of ServiceMeshFactory
type DefaultServiceMeshFactory struct {
	client client.Client
	log    logr.Logger
}

// NewServiceMeshFactory creates a new service mesh factory
func NewServiceMeshFactory(client client.Client, log logr.Logger) ServiceMeshFactory {
	return &DefaultServiceMeshFactory{
		client: client,
		log:    log,
	}
}

// CreateManager creates a service mesh manager for the specified provider
func (f *DefaultServiceMeshFactory) CreateManager(provider MeshProvider) (ServiceMeshManager, error) {
	switch provider {
	case IstioProvider:
		// TODO: Implement in Part 2
		return nil, fmt.Errorf("Istio provider implementation pending")
	case LinkerdProvider:
		// TODO: Implement in Part 3
		return nil, fmt.Errorf("Linkerd provider implementation pending")
	case NoneProvider:
		return NewNoopServiceMeshManager(f.client, f.log), nil
	default:
		return nil, fmt.Errorf("unsupported service mesh provider: %s", provider)
	}
}

// GetServiceMeshProvider extracts the service mesh provider from the platform spec
func GetServiceMeshProvider(platform *observabilityv1.ObservabilityPlatform) MeshProvider {
	if platform == nil || platform.Spec.ServiceMesh == nil {
		return NoneProvider
	}
	
	if !platform.Spec.ServiceMesh.Enabled {
		return NoneProvider
	}
	
	switch platform.Spec.ServiceMesh.Provider {
	case string(IstioProvider):
		return IstioProvider
	case string(LinkerdProvider):
		return LinkerdProvider
	default:
		return NoneProvider
	}
}

// GetServiceMeshConfig extracts the service mesh configuration from the platform spec
func GetServiceMeshConfig(platform *observabilityv1.ObservabilityPlatform) ServiceMeshConfig {
	if platform == nil || platform.Spec.ServiceMesh == nil {
		return ServiceMeshConfig{
			Provider: NoneProvider,
			Enabled:  false,
		}
	}
	
	sm := platform.Spec.ServiceMesh
	config := ServiceMeshConfig{
		Provider:    GetServiceMeshProvider(platform),
		Enabled:     sm.Enabled,
		Namespace:   sm.Namespace,
		Labels:      sm.Labels,
		Annotations: sm.Annotations,
	}
	
	// Configure mTLS
	if sm.MTLS != nil {
		config.MTLS = MTLSConfig{
			Mode:                sm.MTLS.Mode,
			CertificateRotation: sm.MTLS.CertificateRotation,
			RootCertTTL:         sm.MTLS.RootCertTTL.Duration,
			WorkloadCertTTL:     sm.MTLS.WorkloadCertTTL.Duration,
		}
	}
	
	// Configure traffic management
	if sm.TrafficManagement != nil {
		config.TrafficManagement = TrafficConfig{
			Timeout: sm.TrafficManagement.Timeout.Duration,
		}
		
		// Retry policy
		if sm.TrafficManagement.RetryPolicy != nil {
			config.TrafficManagement.RetryPolicy = RetryPolicy{
				Attempts:      sm.TrafficManagement.RetryPolicy.Attempts,
				PerTryTimeout: sm.TrafficManagement.RetryPolicy.PerTryTimeout.Duration,
				RetryOn:       sm.TrafficManagement.RetryPolicy.RetryOn,
			}
		}
		
		// Circuit breaker
		if sm.TrafficManagement.CircuitBreaker != nil {
			config.TrafficManagement.CircuitBreaker = CircuitBreakerConfig{
				ConsecutiveErrors:  sm.TrafficManagement.CircuitBreaker.ConsecutiveErrors,
				Interval:           sm.TrafficManagement.CircuitBreaker.Interval.Duration,
				BaseEjectionTime:   sm.TrafficManagement.CircuitBreaker.BaseEjectionTime.Duration,
				MaxEjectionPercent: sm.TrafficManagement.CircuitBreaker.MaxEjectionPercent,
			}
		}
		
		// Load balancer
		if sm.TrafficManagement.LoadBalancer != nil {
			config.TrafficManagement.LoadBalancer = LoadBalancerConfig{
				Algorithm: sm.TrafficManagement.LoadBalancer.Algorithm,
			}
			
			if sm.TrafficManagement.LoadBalancer.ConsistentHash != nil {
				config.TrafficManagement.LoadBalancer.ConsistentHash = &ConsistentHashConfig{
					HTTPCookie:      sm.TrafficManagement.LoadBalancer.ConsistentHash.HTTPCookie,
					HTTPHeader:      sm.TrafficManagement.LoadBalancer.ConsistentHash.HTTPHeader,
					MinimumRingSize: sm.TrafficManagement.LoadBalancer.ConsistentHash.MinimumRingSize,
				}
			}
		}
	}
	
	// Configure observability
	if sm.Observability != nil {
		config.Observability = ObservabilityConfig{
			Metrics: MetricsConfig{
				Enabled:   sm.Observability.Metrics.Enabled,
				Providers: sm.Observability.Metrics.Providers,
				Tags:      sm.Observability.Metrics.Tags,
			},
			Tracing: TracingConfig{
				Enabled:      sm.Observability.Tracing.Enabled,
				Provider:     sm.Observability.Tracing.Provider,
				SamplingRate: sm.Observability.Tracing.SamplingRate,
				Tags:         sm.Observability.Tracing.Tags,
			},
			AccessLog: AccessLogConfig{
				Enabled:  sm.Observability.AccessLog.Enabled,
				Format:   sm.Observability.AccessLog.Format,
				Encoding: sm.Observability.AccessLog.Encoding,
			},
		}
	}
	
	return config
}

// ValidateServiceMeshConfig validates the service mesh configuration
func ValidateServiceMeshConfig(config ServiceMeshConfig) error {
	if config.Provider == "" {
		return fmt.Errorf("service mesh provider is required")
	}
	
	// Validate mTLS mode
	if config.MTLS.Mode != "" {
		switch config.MTLS.Mode {
		case MTLSModeStrict, MTLSModePermissive, MTLSModeDisable:
			// Valid modes
		default:
			return fmt.Errorf("invalid mTLS mode: %s", config.MTLS.Mode)
		}
	}
	
	// Validate load balancer algorithm
	if config.TrafficManagement.LoadBalancer.Algorithm != "" {
		switch config.TrafficManagement.LoadBalancer.Algorithm {
		case LoadBalancerRoundRobin, LoadBalancerLeastRequest, LoadBalancerRandom, LoadBalancerPassthrough:
			// Valid algorithms
		default:
			return fmt.Errorf("invalid load balancer algorithm: %s", config.TrafficManagement.LoadBalancer.Algorithm)
		}
	}
	
	// Validate retry policy
	if config.TrafficManagement.RetryPolicy.Attempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}
	
	// Validate circuit breaker
	if config.TrafficManagement.CircuitBreaker.ConsecutiveErrors < 0 {
		return fmt.Errorf("consecutive errors cannot be negative")
	}
	if config.TrafficManagement.CircuitBreaker.MaxEjectionPercent < 0 || config.TrafficManagement.CircuitBreaker.MaxEjectionPercent > 100 {
		return fmt.Errorf("max ejection percent must be between 0 and 100")
	}
	
	// Validate observability
	if config.Observability.Tracing.SamplingRate < 0 || config.Observability.Tracing.SamplingRate > 1 {
		return fmt.Errorf("tracing sampling rate must be between 0 and 1")
	}
	
	if config.Observability.AccessLog.Encoding != "" {
		switch config.Observability.AccessLog.Encoding {
		case LogEncodingText, LogEncodingJSON:
			// Valid encodings
		default:
			return fmt.Errorf("invalid log encoding: %s", config.Observability.AccessLog.Encoding)
		}
	}
	
	return nil
}

// GenerateServiceMeshLabels generates labels for service mesh resources
func GenerateServiceMeshLabels(platform *observabilityv1.ObservabilityPlatform, component string) map[string]string {
	labels := make(map[string]string)
	
	// Add standard labels
	labels["app.kubernetes.io/name"] = "gunj-operator"
	labels["app.kubernetes.io/instance"] = platform.Name
	labels["app.kubernetes.io/component"] = component
	labels["app.kubernetes.io/managed-by"] = "gunj-operator"
	labels["observability.io/platform"] = platform.Name
	
	// Add mesh-specific labels
	config := GetServiceMeshConfig(platform)
	if config.Provider != NoneProvider {
		labels["service-mesh.io/provider"] = string(config.Provider)
		labels["service-mesh.io/enabled"] = "true"
	}
	
	// Add custom labels from config
	for k, v := range config.Labels {
		labels[k] = v
	}
	
	return labels
}

// GenerateServiceMeshAnnotations generates annotations for service mesh resources
func GenerateServiceMeshAnnotations(platform *observabilityv1.ObservabilityPlatform, component string) map[string]string {
	annotations := make(map[string]string)
	
	// Add standard annotations
	annotations["observability.io/platform"] = platform.Name
	annotations["observability.io/component"] = component
	
	// Add mesh-specific annotations
	config := GetServiceMeshConfig(platform)
	
	// Add provider-specific annotations
	switch config.Provider {
	case IstioProvider:
		// Istio sidecar injection
		if config.Enabled {
			annotations["sidecar.istio.io/inject"] = "true"
		}
		// Istio telemetry
		if config.Observability.Metrics.Enabled {
			annotations["sidecar.istio.io/statsInclusionRegexps"] = ".*"
		}
	case LinkerdProvider:
		// Linkerd injection
		if config.Enabled {
			annotations["linkerd.io/inject"] = "enabled"
		}
	}
	
	// Add custom annotations from config
	for k, v := range config.Annotations {
		annotations[k] = v
	}
	
	return annotations
}
