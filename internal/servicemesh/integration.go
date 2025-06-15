package servicemesh

import (
	"context"
	"fmt"
	
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Integration handles service mesh integration for the Gunj operator
type Integration struct {
	client  client.Client
	factory ManagerFactory
}

// NewIntegration creates a new service mesh integration
func NewIntegration(client client.Client, factory ManagerFactory) *Integration {
	return &Integration{
		client:  client,
		factory: factory,
	}
}

// ConfigureServiceMesh configures service mesh for a platform
func (i *Integration) ConfigureServiceMesh(ctx context.Context, namespace string, config *ServiceMeshConfig) error {
	log := log.FromContext(ctx).WithValues("namespace", namespace, "meshType", config.Type)
	
	if !config.Enabled {
		log.V(1).Info("Service mesh integration is disabled")
		return nil
	}
	
	// Create appropriate manager
	manager, err := i.factory.CreateManager(config.Type, i.client)
	if err != nil {
		return fmt.Errorf("creating service mesh manager: %w", err)
	}
	
	// Configure the service mesh
	if err := manager.Configure(ctx, namespace, config); err != nil {
		return fmt.Errorf("configuring service mesh: %w", err)
	}
	
	log.Info("Service mesh configured successfully")
	return nil
}

// CreateTrafficManagement creates traffic management resources
func (i *Integration) CreateTrafficManagement(ctx context.Context, namespace string, config *ServiceMeshConfig, serviceName string) error {
	if !config.Enabled {
		return nil
	}
	
	manager, err := i.factory.CreateManager(config.Type, i.client)
	if err != nil {
		return fmt.Errorf("creating service mesh manager: %w", err)
	}
	
	// Create VirtualService for basic routing
	vsSpec := &VirtualServiceSpec{
		Hosts: []string{serviceName},
		HTTP: []HTTPRouteSpec{
			{
				Route: []HTTPRouteDestination{
					{
						Destination: &Destination{
							Host: serviceName,
						},
						Weight: 100,
					},
				},
			},
		},
	}
	
	// Add retry policy if configured
	if config.TrafficManagement.RetryPolicy != nil {
		vsSpec.HTTP[0].Retry = config.TrafficManagement.RetryPolicy
	}
	
	// Add timeout if configured
	if config.TrafficManagement.Timeout != "" {
		vsSpec.HTTP[0].Timeout = config.TrafficManagement.Timeout
	}
	
	if err := manager.CreateVirtualService(ctx, namespace, serviceName+"-vs", vsSpec); err != nil {
		return fmt.Errorf("creating virtual service: %w", err)
	}
	
	// Create DestinationRule for load balancing and circuit breaking
	drSpec := &DestinationRuleSpec{
		Host: serviceName,
	}
	
	// Configure traffic policy
	if config.TrafficManagement.LoadBalancer != nil || config.TrafficManagement.CircuitBreaker != nil {
		drSpec.TrafficPolicy = &TrafficPolicy{}
		
		// Load balancer
		if config.TrafficManagement.LoadBalancer != nil {
			drSpec.TrafficPolicy.LoadBalancer = &LoadBalancerSettings{
				Simple: config.TrafficManagement.LoadBalancer.Type,
			}
			
			if config.TrafficManagement.LoadBalancer.ConsistentHash != nil {
				drSpec.TrafficPolicy.LoadBalancer = &LoadBalancerSettings{
					ConsistentHash: &ConsistentHashLB{
						HTTPHeaderName: config.TrafficManagement.LoadBalancer.ConsistentHash.HTTPHeaderName,
						HTTPCookie:     convertHTTPCookie(config.TrafficManagement.LoadBalancer.ConsistentHash.HTTPCookie),
						UseSourceIP:    config.TrafficManagement.LoadBalancer.ConsistentHash.UseSourceIP,
					},
				}
			}
		}
		
		// Circuit breaker
		if config.TrafficManagement.CircuitBreaker != nil {
			drSpec.TrafficPolicy.OutlierDetection = &OutlierDetection{
				ConsecutiveErrors:  config.TrafficManagement.CircuitBreaker.ConsecutiveErrors,
				Interval:           config.TrafficManagement.CircuitBreaker.Interval,
				BaseEjectionTime:   config.TrafficManagement.CircuitBreaker.BaseEjectionTime,
				MaxEjectionPercent: config.TrafficManagement.CircuitBreaker.MaxEjectionPercent,
			}
		}
	}
	
	if err := manager.CreateDestinationRule(ctx, namespace, serviceName+"-dr", drSpec); err != nil {
		return fmt.Errorf("creating destination rule: %w", err)
	}
	
	return nil
}

// CreateSecurityPolicies creates security policies for the service mesh
func (i *Integration) CreateSecurityPolicies(ctx context.Context, namespace string, config *ServiceMeshConfig, serviceName string, labels map[string]string) error {
	if !config.Enabled {
		return nil
	}
	
	manager, err := i.factory.CreateManager(config.Type, i.client)
	if err != nil {
		return fmt.Errorf("creating service mesh manager: %w", err)
	}
	
	// Create PeerAuthentication for mTLS
	if config.MTLS.Mode != "" || config.MTLS.Strict {
		paSpec := &PeerAuthenticationSpec{
			MutualTLS: &MutualTLS{
				Mode: config.MTLS.Mode,
			},
		}
		
		if config.MTLS.Strict {
			paSpec.MutualTLS.Mode = "STRICT"
		}
		
		// Apply to specific workload if labels provided
		if len(labels) > 0 {
			paSpec.Selector = &WorkloadSelector{
				MatchLabels: labels,
			}
		}
		
		if err := manager.CreatePeerAuthentication(ctx, namespace, serviceName+"-mtls", paSpec); err != nil {
			return fmt.Errorf("creating peer authentication: %w", err)
		}
	}
	
	return nil
}

// CreateTelemetryConfiguration creates telemetry configuration
func (i *Integration) CreateTelemetryConfiguration(ctx context.Context, namespace string, config *ServiceMeshConfig, serviceName string) error {
	if !config.Enabled {
		return nil
	}
	
	manager, err := i.factory.CreateManager(config.Type, i.client)
	if err != nil {
		return fmt.Errorf("creating service mesh manager: %w", err)
	}
	
	telemetrySpec := &TelemetrySpec{}
	
	// Configure metrics
	if config.Telemetry.Metrics.Enabled {
		metricSpec := MetricSpec{
			Dimensions: config.Telemetry.Metrics.Dimensions,
		}
		
		for _, provider := range config.Telemetry.Metrics.Providers {
			metricSpec.Providers = append(metricSpec.Providers, ProviderRef{Name: provider})
		}
		
		telemetrySpec.Metrics = []MetricSpec{metricSpec}
	}
	
	// Configure tracing
	if config.Telemetry.Tracing.Enabled {
		tracingSpec := TracingSpec{
			RandomSamplingPercentage: config.Telemetry.Tracing.SamplingRate,
		}
		
		if config.Telemetry.Tracing.Provider != "" {
			tracingSpec.Providers = []ProviderRef{{Name: config.Telemetry.Tracing.Provider}}
		}
		
		telemetrySpec.Tracing = []TracingSpec{tracingSpec}
	}
	
	// Configure access logging
	if config.Telemetry.AccessLogging.Enabled {
		accessLogSpec := AccessLoggingSpec{}
		
		for _, provider := range config.Telemetry.AccessLogging.Providers {
			accessLogSpec.Providers = append(accessLogSpec.Providers, ProviderRef{Name: provider})
		}
		
		telemetrySpec.AccessLogging = []AccessLoggingSpec{accessLogSpec}
	}
	
	if err := manager.CreateTelemetry(ctx, namespace, serviceName+"-telemetry", telemetrySpec); err != nil {
		return fmt.Errorf("creating telemetry configuration: %w", err)
	}
	
	return nil
}

// GetStatus returns the service mesh status
func (i *Integration) GetStatus(ctx context.Context, namespace string, meshType ServiceMeshType) (*ServiceMeshStatus, error) {
	manager, err := i.factory.CreateManager(meshType, i.client)
	if err != nil {
		return nil, fmt.Errorf("creating service mesh manager: %w", err)
	}
	
	return manager.GetStatus(ctx, namespace)
}

// Cleanup removes service mesh resources
func (i *Integration) Cleanup(ctx context.Context, namespace string, meshType ServiceMeshType) error {
	manager, err := i.factory.CreateManager(meshType, i.client)
	if err != nil {
		return fmt.Errorf("creating service mesh manager: %w", err)
	}
	
	return manager.Cleanup(ctx, namespace)
}

// Helper functions

func convertHTTPCookie(cookie *HTTPCookie) *HTTPCookieLB {
	if cookie == nil {
		return nil
	}
	
	return &HTTPCookieLB{
		Name: cookie.Name,
		Path: cookie.Path,
		TTL:  cookie.TTL,
	}
}

// DefaultServiceMeshConfig returns a default service mesh configuration
func DefaultServiceMeshConfig(meshType ServiceMeshType) *ServiceMeshConfig {
	return &ServiceMeshConfig{
		Type:    meshType,
		Enabled: true,
		MTLS: MTLSConfig{
			Mode:   "PERMISSIVE",
			Strict: false,
		},
		TrafficManagement: TrafficManagementConfig{
			RetryPolicy: &RetryPolicy{
				Attempts:      3,
				PerTryTimeout: "2s",
				RetryOn:       "5xx,reset,connect-failure,refused-stream",
			},
			CircuitBreaker: &CircuitBreakerConfig{
				ConsecutiveErrors:  5,
				Interval:           "30s",
				BaseEjectionTime:   "30s",
				MaxEjectionPercent: 50,
			},
			LoadBalancer: &LoadBalancerConfig{
				Type: "ROUND_ROBIN",
			},
			Timeout: "30s",
		},
		Telemetry: TelemetryConfig{
			Metrics: MetricsConfig{
				Enabled:   true,
				Providers: []string{"prometheus"},
			},
			Tracing: TracingConfig{
				Enabled:      true,
				SamplingRate: 1.0,
				Provider:     "jaeger",
			},
			AccessLogging: AccessLoggingConfig{
				Enabled:   true,
				Providers: []string{"otel"},
			},
		},
	}
}
