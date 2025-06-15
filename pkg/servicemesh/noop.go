// Package servicemesh provides service mesh integration for the Gunj Operator
package servicemesh

import (
	"context"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NoopServiceMeshManager is a no-operation implementation of ServiceMeshManager
// Used when service mesh integration is disabled
type NoopServiceMeshManager struct {
	client client.Client
	log    logr.Logger
}

// NewNoopServiceMeshManager creates a new noop service mesh manager
func NewNoopServiceMeshManager(client client.Client, log logr.Logger) ServiceMeshManager {
	return &NoopServiceMeshManager{
		client: client,
		log:    log.WithName("noop-service-mesh"),
	}
}

// Initialize initializes the service mesh integration (no-op)
func (n *NoopServiceMeshManager) Initialize(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	n.log.V(1).Info("Service mesh integration is disabled", "platform", platform.Name, "namespace", platform.Namespace)
	return nil
}

// Configure configures the service mesh for the platform (no-op)
func (n *NoopServiceMeshManager) Configure(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, config ServiceMeshConfig) error {
	n.log.V(1).Info("Service mesh configuration skipped - integration disabled", "platform", platform.Name)
	return nil
}

// ApplyTrafficPolicy applies traffic management policies (no-op)
func (n *NoopServiceMeshManager) ApplyTrafficPolicy(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, policy TrafficPolicy) error {
	n.log.V(1).Info("Traffic policy application skipped - service mesh disabled", "platform", platform.Name, "policy", policy.Name)
	return nil
}

// ConfigureMTLS configures mutual TLS (no-op)
func (n *NoopServiceMeshManager) ConfigureMTLS(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, config MTLSConfig) error {
	n.log.V(1).Info("mTLS configuration skipped - service mesh disabled", "platform", platform.Name)
	return nil
}

// EnableObservability enables observability features (no-op)
func (n *NoopServiceMeshManager) EnableObservability(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, config ObservabilityConfig) error {
	n.log.V(1).Info("Service mesh observability skipped - integration disabled", "platform", platform.Name)
	return nil
}

// GetStatus returns the current status of service mesh integration
func (n *NoopServiceMeshManager) GetStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*ServiceMeshStatus, error) {
	return &ServiceMeshStatus{
		Phase:              ServiceMeshPhaseReady,
		Message:            "Service mesh integration is disabled",
		LastReconcileTime:  metav1.Now(),
		ObservedGeneration: platform.Generation,
		Conditions: []metav1.Condition{
			{
				Type:               "ServiceMeshReady",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: platform.Generation,
				LastTransitionTime: metav1.Now(),
				Reason:             "Disabled",
				Message:            "Service mesh integration is disabled",
			},
		},
	}, nil
}

// Cleanup removes service mesh resources (no-op)
func (n *NoopServiceMeshManager) Cleanup(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	n.log.V(1).Info("Service mesh cleanup skipped - integration disabled", "platform", platform.Name)
	return nil
}

// HealthCheck performs health check on service mesh integration (no-op)
func (n *NoopServiceMeshManager) HealthCheck(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	n.log.V(1).Info("Service mesh health check skipped - integration disabled", "platform", platform.Name)
	return nil
}

// ValidateConfiguration validates the service mesh configuration (no-op)
func (n *NoopServiceMeshManager) ValidateConfiguration(config ServiceMeshConfig) error {
	if config.Enabled {
		n.log.Info("Service mesh is enabled in config but using noop manager")
	}
	return nil
}
