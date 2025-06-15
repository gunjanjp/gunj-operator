package factory

import (
	"context"
	"fmt"
	
	"github.com/gunjanjp/gunj-operator/internal/servicemesh"
	"github.com/gunjanjp/gunj-operator/internal/servicemesh/istio"
	"github.com/gunjanjp/gunj-operator/internal/servicemesh/linkerd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultFactory is the default implementation of ManagerFactory
type DefaultFactory struct{}

// NewDefaultFactory creates a new default factory
func NewDefaultFactory() servicemesh.ManagerFactory {
	return &DefaultFactory{}
}

// CreateManager creates a service mesh manager based on type
func (f *DefaultFactory) CreateManager(meshType servicemesh.ServiceMeshType, client client.Client) (servicemesh.Manager, error) {
	switch meshType {
	case servicemesh.ServiceMeshTypeIstio:
		return istio.NewIstioManager(client), nil
	case servicemesh.ServiceMeshTypeLinkerd:
		return linkerd.NewLinkerdManager(client), nil
	case servicemesh.ServiceMeshTypeNone:
		return NewNoopManager(), nil
	default:
		return nil, fmt.Errorf("unsupported service mesh type: %s", meshType)
	}
}

// NoopManager is a no-op implementation of the Manager interface
type NoopManager struct{}

// NewNoopManager creates a new no-op manager
func NewNoopManager() servicemesh.Manager {
	return &NoopManager{}
}

// Configure is a no-op
func (m *NoopManager) Configure(ctx context.Context, namespace string, config *servicemesh.ServiceMeshConfig) error {
	return nil
}

// CreateVirtualService is a no-op
func (m *NoopManager) CreateVirtualService(ctx context.Context, namespace, name string, spec *servicemesh.VirtualServiceSpec) error {
	return nil
}

// CreateDestinationRule is a no-op
func (m *NoopManager) CreateDestinationRule(ctx context.Context, namespace, name string, spec *servicemesh.DestinationRuleSpec) error {
	return nil
}

// CreatePeerAuthentication is a no-op
func (m *NoopManager) CreatePeerAuthentication(ctx context.Context, namespace, name string, spec *servicemesh.PeerAuthenticationSpec) error {
	return nil
}

// CreateTelemetry is a no-op
func (m *NoopManager) CreateTelemetry(ctx context.Context, namespace, name string, spec *servicemesh.TelemetrySpec) error {
	return nil
}

// Cleanup is a no-op
func (m *NoopManager) Cleanup(ctx context.Context, namespace string) error {
	return nil
}

// GetStatus returns a ready status
func (m *NoopManager) GetStatus(ctx context.Context, namespace string) (*servicemesh.ServiceMeshStatus, error) {
	return &servicemesh.ServiceMeshStatus{
		Ready:   true,
		Message: "No service mesh configured",
	}, nil
}
