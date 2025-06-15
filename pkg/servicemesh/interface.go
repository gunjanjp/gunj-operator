// Package servicemesh provides service mesh integration for the Gunj Operator
package servicemesh

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ServiceMeshProvider represents a service mesh implementation
type ServiceMeshProvider interface {
	// IsInstalled checks if the service mesh is installed in the cluster
	IsInstalled(ctx context.Context) (bool, error)

	// Configure configures the service mesh for an observability platform
	Configure(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error

	// EnableSidecarInjection enables automatic sidecar injection for a namespace
	EnableSidecarInjection(ctx context.Context, namespace string) error

	// ConfigureMTLS configures mutual TLS for the platform components
	ConfigureMTLS(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error

	// CreateTrafficPolicies creates traffic management policies
	CreateTrafficPolicies(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error

	// GetMetrics retrieves service mesh metrics
	GetMetrics(ctx context.Context, namespace string) (*ServiceMeshMetrics, error)

	// Cleanup removes service mesh configurations
	Cleanup(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
}

// ServiceMeshType represents the type of service mesh
type ServiceMeshType string

const (
	// ServiceMeshTypeIstio represents Istio service mesh
	ServiceMeshTypeIstio ServiceMeshType = "istio"
	// ServiceMeshTypeLinkerd represents Linkerd service mesh
	ServiceMeshTypeLinkerd ServiceMeshType = "linkerd"
	// ServiceMeshTypeNone represents no service mesh
	ServiceMeshTypeNone ServiceMeshType = "none"
)

// ServiceMeshMetrics represents metrics from the service mesh
type ServiceMeshMetrics struct {
	// RequestRate is the rate of requests per second
	RequestRate float64 `json:"requestRate"`
	// ErrorRate is the rate of errors per second
	ErrorRate float64 `json:"errorRate"`
	// P50Latency is the 50th percentile latency in milliseconds
	P50Latency float64 `json:"p50Latency"`
	// P99Latency is the 99th percentile latency in milliseconds
	P99Latency float64 `json:"p99Latency"`
	// ActiveConnections is the number of active connections
	ActiveConnections int `json:"activeConnections"`
}

// Manager manages service mesh integrations
type Manager struct {
	client       client.Client
	providers    map[ServiceMeshType]ServiceMeshProvider
	healthChecker *HealthChecker
	mu           sync.RWMutex
}

// NewManager creates a new service mesh manager
func NewManager(client client.Client) *Manager {
	m := &Manager{
		client:    client,
		providers: make(map[ServiceMeshType]ServiceMeshProvider),
	}
	m.healthChecker = NewHealthChecker(client, m)
	return m
}

// RegisterProvider registers a service mesh provider
func (m *Manager) RegisterProvider(meshType ServiceMeshType, provider ServiceMeshProvider) {
	m.providers[meshType] = provider
}

// GetProvider returns the service mesh provider for the given type
func (m *Manager) GetProvider(meshType ServiceMeshType) (ServiceMeshProvider, error) {
	provider, ok := m.providers[meshType]
	if !ok {
		return nil, fmt.Errorf("service mesh provider %s not found", meshType)
	}
	return provider, nil
}

// DetectServiceMesh detects which service mesh is installed in the cluster
func (m *Manager) DetectServiceMesh(ctx context.Context) (ServiceMeshType, error) {
	// Check for Istio
	if istioProvider, ok := m.providers[ServiceMeshTypeIstio]; ok {
		if installed, err := istioProvider.IsInstalled(ctx); err == nil && installed {
			return ServiceMeshTypeIstio, nil
		}
	}

	// Check for Linkerd
	if linkerdProvider, ok := m.providers[ServiceMeshTypeLinkerd]; ok {
		if installed, err := linkerdProvider.IsInstalled(ctx); err == nil && installed {
			return ServiceMeshTypeLinkerd, nil
		}
	}

	return ServiceMeshTypeNone, nil
}

// ConfigurePlatform configures service mesh for an observability platform
func (m *Manager) ConfigurePlatform(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
	// Detect service mesh type
	meshType, err := m.DetectServiceMesh(ctx)
	if err != nil {
		return fmt.Errorf("detecting service mesh: %w", err)
	}

	if meshType == ServiceMeshTypeNone {
		// No service mesh detected, skip configuration
		return nil
	}

	// Get the appropriate provider
	provider, err := m.GetProvider(meshType)
	if err != nil {
		return fmt.Errorf("getting service mesh provider: %w", err)
	}

	// Configure the service mesh
	if err := provider.Configure(ctx, platform); err != nil {
		return fmt.Errorf("configuring service mesh: %w", err)
	}

	// Enable sidecar injection
	if err := provider.EnableSidecarInjection(ctx, platform.Namespace); err != nil {
		return fmt.Errorf("enabling sidecar injection: %w", err)
	}

	// Configure mTLS if enabled
	if platform.Spec.ServiceMesh != nil && platform.Spec.ServiceMesh.MTLS.Enabled {
		if err := provider.ConfigureMTLS(ctx, platform); err != nil {
			return fmt.Errorf("configuring mTLS: %w", err)
		}
	}

	// Create traffic policies
	if err := provider.CreateTrafficPolicies(ctx, platform); err != nil {
		return fmt.Errorf("creating traffic policies: %w", err)
	}

	// Start health checks if enabled
	if platform.Spec.ServiceMesh != nil && platform.Spec.ServiceMesh.HealthCheck != nil && platform.Spec.ServiceMesh.HealthCheck.Enabled {
		if err := m.healthChecker.StartHealthChecks(ctx, platform); err != nil {
			return fmt.Errorf("starting health checks: %w", err)
		}
	}

	return nil
}
