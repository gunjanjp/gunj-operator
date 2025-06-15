package servicemesh

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockManager is a mock implementation of the Manager interface
type MockManager struct {
	mock.Mock
}

func (m *MockManager) Configure(ctx context.Context, namespace string, config *ServiceMeshConfig) error {
	args := m.Called(ctx, namespace, config)
	return args.Error(0)
}

func (m *MockManager) CreateVirtualService(ctx context.Context, namespace, name string, spec *VirtualServiceSpec) error {
	args := m.Called(ctx, namespace, name, spec)
	return args.Error(0)
}

func (m *MockManager) CreateDestinationRule(ctx context.Context, namespace, name string, spec *DestinationRuleSpec) error {
	args := m.Called(ctx, namespace, name, spec)
	return args.Error(0)
}

func (m *MockManager) CreatePeerAuthentication(ctx context.Context, namespace, name string, spec *PeerAuthenticationSpec) error {
	args := m.Called(ctx, namespace, name, spec)
	return args.Error(0)
}

func (m *MockManager) CreateTelemetry(ctx context.Context, namespace, name string, spec *TelemetrySpec) error {
	args := m.Called(ctx, namespace, name, spec)
	return args.Error(0)
}

func (m *MockManager) Cleanup(ctx context.Context, namespace string) error {
	args := m.Called(ctx, namespace)
	return args.Error(0)
}

func (m *MockManager) GetStatus(ctx context.Context, namespace string) (*ServiceMeshStatus, error) {
	args := m.Called(ctx, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ServiceMeshStatus), args.Error(1)
}

// MockFactory is a mock implementation of ManagerFactory
type MockFactory struct {
	mock.Mock
}

func (f *MockFactory) CreateManager(meshType ServiceMeshType, client client.Client) (Manager, error) {
	args := f.Called(meshType, client)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(Manager), args.Error(1)
}

func TestIntegration_ConfigureServiceMesh(t *testing.T) {
	tests := []struct {
		name      string
		config    *ServiceMeshConfig
		setupMock func(*MockFactory, *MockManager)
		wantErr   bool
	}{
		{
			name: "service mesh disabled",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: false,
			},
			setupMock: func(f *MockFactory, m *MockManager) {
				// Should not call factory when disabled
			},
			wantErr: false,
		},
		{
			name: "configure istio successfully",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: true,
			},
			setupMock: func(f *MockFactory, m *MockManager) {
				f.On("CreateManager", ServiceMeshTypeIstio, mock.Anything).Return(m, nil)
				m.On("Configure", mock.Anything, "test-ns", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "factory creation error",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: true,
			},
			setupMock: func(f *MockFactory, m *MockManager) {
				f.On("CreateManager", ServiceMeshTypeIstio, mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "configuration error",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: true,
			},
			setupMock: func(f *MockFactory, m *MockManager) {
				f.On("CreateManager", ServiceMeshTypeIstio, mock.Anything).Return(m, nil)
				m.On("Configure", mock.Anything, "test-ns", mock.Anything).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().Build()
			factory := new(MockFactory)
			manager := new(MockManager)
			
			if tt.setupMock != nil {
				tt.setupMock(factory, manager)
			}
			
			integration := NewIntegration(client, factory)
			err := integration.ConfigureServiceMesh(context.Background(), "test-ns", tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			factory.AssertExpectations(t)
			manager.AssertExpectations(t)
		})
	}
}

func TestIntegration_CreateTrafficManagement(t *testing.T) {
	tests := []struct {
		name        string
		config      *ServiceMeshConfig
		serviceName string
		setupMock   func(*MockFactory, *MockManager)
		wantErr     bool
	}{
		{
			name: "create traffic management with retry and timeout",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: true,
				TrafficManagement: TrafficManagementConfig{
					RetryPolicy: &RetryPolicy{
						Attempts:      3,
						PerTryTimeout: "2s",
						RetryOn:       "5xx",
					},
					Timeout: "10s",
				},
			},
			serviceName: "test-service",
			setupMock: func(f *MockFactory, m *MockManager) {
				f.On("CreateManager", ServiceMeshTypeIstio, mock.Anything).Return(m, nil)
				
				// Expect VirtualService creation
				m.On("CreateVirtualService", mock.Anything, "test-ns", "test-service-vs", mock.MatchedBy(func(spec *VirtualServiceSpec) bool {
					return len(spec.HTTP) > 0 &&
						spec.HTTP[0].Retry != nil &&
						spec.HTTP[0].Retry.Attempts == 3 &&
						spec.HTTP[0].Timeout == "10s"
				})).Return(nil)
				
				// Expect DestinationRule creation
				m.On("CreateDestinationRule", mock.Anything, "test-ns", "test-service-dr", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "create traffic management with circuit breaker",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: true,
				TrafficManagement: TrafficManagementConfig{
					CircuitBreaker: &CircuitBreakerConfig{
						ConsecutiveErrors:  5,
						Interval:           "30s",
						BaseEjectionTime:   "30s",
						MaxEjectionPercent: 50,
					},
				},
			},
			serviceName: "test-service",
			setupMock: func(f *MockFactory, m *MockManager) {
				f.On("CreateManager", ServiceMeshTypeIstio, mock.Anything).Return(m, nil)
				
				// Expect VirtualService creation
				m.On("CreateVirtualService", mock.Anything, "test-ns", "test-service-vs", mock.Anything).Return(nil)
				
				// Expect DestinationRule creation with circuit breaker
				m.On("CreateDestinationRule", mock.Anything, "test-ns", "test-service-dr", mock.MatchedBy(func(spec *DestinationRuleSpec) bool {
					return spec.TrafficPolicy != nil &&
						spec.TrafficPolicy.OutlierDetection != nil &&
						spec.TrafficPolicy.OutlierDetection.ConsecutiveErrors == 5
				})).Return(nil)
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().Build()
			factory := new(MockFactory)
			manager := new(MockManager)
			
			if tt.setupMock != nil {
				tt.setupMock(factory, manager)
			}
			
			integration := NewIntegration(client, factory)
			err := integration.CreateTrafficManagement(context.Background(), "test-ns", tt.config, tt.serviceName)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			factory.AssertExpectations(t)
			manager.AssertExpectations(t)
		})
	}
}

func TestIntegration_CreateSecurityPolicies(t *testing.T) {
	tests := []struct {
		name        string
		config      *ServiceMeshConfig
		serviceName string
		labels      map[string]string
		setupMock   func(*MockFactory, *MockManager)
		wantErr     bool
	}{
		{
			name: "create strict mTLS",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: true,
				MTLS: MTLSConfig{
					Strict: true,
				},
			},
			serviceName: "test-service",
			labels:      map[string]string{"app": "test"},
			setupMock: func(f *MockFactory, m *MockManager) {
				f.On("CreateManager", ServiceMeshTypeIstio, mock.Anything).Return(m, nil)
				
				// Expect PeerAuthentication creation
				m.On("CreatePeerAuthentication", mock.Anything, "test-ns", "test-service-mtls", mock.MatchedBy(func(spec *PeerAuthenticationSpec) bool {
					return spec.MutualTLS != nil &&
						spec.MutualTLS.Mode == "STRICT" &&
						spec.Selector != nil &&
						spec.Selector.MatchLabels["app"] == "test"
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "create permissive mTLS",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: true,
				MTLS: MTLSConfig{
					Mode: "PERMISSIVE",
				},
			},
			serviceName: "test-service",
			labels:      nil,
			setupMock: func(f *MockFactory, m *MockManager) {
				f.On("CreateManager", ServiceMeshTypeIstio, mock.Anything).Return(m, nil)
				
				// Expect PeerAuthentication creation without selector
				m.On("CreatePeerAuthentication", mock.Anything, "test-ns", "test-service-mtls", mock.MatchedBy(func(spec *PeerAuthenticationSpec) bool {
					return spec.MutualTLS != nil &&
						spec.MutualTLS.Mode == "PERMISSIVE" &&
						spec.Selector == nil
				})).Return(nil)
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().Build()
			factory := new(MockFactory)
			manager := new(MockManager)
			
			if tt.setupMock != nil {
				tt.setupMock(factory, manager)
			}
			
			integration := NewIntegration(client, factory)
			err := integration.CreateSecurityPolicies(context.Background(), "test-ns", tt.config, tt.serviceName, tt.labels)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			factory.AssertExpectations(t)
			manager.AssertExpectations(t)
		})
	}
}

func TestIntegration_CreateTelemetryConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      *ServiceMeshConfig
		serviceName string
		setupMock   func(*MockFactory, *MockManager)
		wantErr     bool
	}{
		{
			name: "create full telemetry configuration",
			config: &ServiceMeshConfig{
				Type:    ServiceMeshTypeIstio,
				Enabled: true,
				Telemetry: TelemetryConfig{
					Metrics: MetricsConfig{
						Enabled:   true,
						Providers: []string{"prometheus"},
						Dimensions: map[string]string{
							"source": "source.workload.name",
						},
					},
					Tracing: TracingConfig{
						Enabled:      true,
						Provider:     "jaeger",
						SamplingRate: 1.0,
					},
					AccessLogging: AccessLoggingConfig{
						Enabled:   true,
						Providers: []string{"otel"},
					},
				},
			},
			serviceName: "test-service",
			setupMock: func(f *MockFactory, m *MockManager) {
				f.On("CreateManager", ServiceMeshTypeIstio, mock.Anything).Return(m, nil)
				
				// Expect Telemetry creation with all features
				m.On("CreateTelemetry", mock.Anything, "test-ns", "test-service-telemetry", mock.MatchedBy(func(spec *TelemetrySpec) bool {
					return len(spec.Metrics) > 0 &&
						len(spec.Tracing) > 0 &&
						len(spec.AccessLogging) > 0 &&
						spec.Tracing[0].RandomSamplingPercentage == 1.0
				})).Return(nil)
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().Build()
			factory := new(MockFactory)
			manager := new(MockManager)
			
			if tt.setupMock != nil {
				tt.setupMock(factory, manager)
			}
			
			integration := NewIntegration(client, factory)
			err := integration.CreateTelemetryConfiguration(context.Background(), "test-ns", tt.config, tt.serviceName)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			factory.AssertExpectations(t)
			manager.AssertExpectations(t)
		})
	}
}

func TestDefaultServiceMeshConfig(t *testing.T) {
	config := DefaultServiceMeshConfig(ServiceMeshTypeIstio)
	
	assert.Equal(t, ServiceMeshTypeIstio, config.Type)
	assert.True(t, config.Enabled)
	assert.Equal(t, "PERMISSIVE", config.MTLS.Mode)
	assert.False(t, config.MTLS.Strict)
	
	// Check traffic management defaults
	assert.NotNil(t, config.TrafficManagement.RetryPolicy)
	assert.Equal(t, int32(3), config.TrafficManagement.RetryPolicy.Attempts)
	assert.NotNil(t, config.TrafficManagement.CircuitBreaker)
	assert.Equal(t, int32(5), config.TrafficManagement.CircuitBreaker.ConsecutiveErrors)
	assert.NotNil(t, config.TrafficManagement.LoadBalancer)
	assert.Equal(t, "ROUND_ROBIN", config.TrafficManagement.LoadBalancer.Type)
	
	// Check telemetry defaults
	assert.True(t, config.Telemetry.Metrics.Enabled)
	assert.Contains(t, config.Telemetry.Metrics.Providers, "prometheus")
	assert.True(t, config.Telemetry.Tracing.Enabled)
	assert.Equal(t, 1.0, config.Telemetry.Tracing.SamplingRate)
	assert.Equal(t, "jaeger", config.Telemetry.Tracing.Provider)
	assert.True(t, config.Telemetry.AccessLogging.Enabled)
	assert.Contains(t, config.Telemetry.AccessLogging.Providers, "otel")
}
