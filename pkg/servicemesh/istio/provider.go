// Package istio provides Istio service mesh integration for the Gunj Operator
package istio

import (
	"context"
	"fmt"

	"istio.io/api/networking/v1beta1"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	istiosecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/pkg/servicemesh"
)

// Provider implements ServiceMeshProvider for Istio
type Provider struct {
	client client.Client
}

// NewProvider creates a new Istio provider
func NewProvider(client client.Client) *Provider {
	return &Provider{
		client: client,
	}
}

// IsInstalled checks if Istio is installed in the cluster
func (p *Provider) IsInstalled(ctx context.Context) (bool, error) {
	// Check if istio-system namespace exists
	namespace := &corev1.Namespace{}
	err := p.client.Get(ctx, types.NamespacedName{Name: "istio-system"}, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Check if istiod deployment exists
	deploymentList := &corev1.PodList{}
	err = p.client.List(ctx, deploymentList, client.InNamespace("istio-system"), client.MatchingLabels{
		"app": "istiod",
	})
	if err != nil {
		return false, err
	}

	return len(deploymentList.Items) > 0, nil
}

// Configure configures Istio for an observability platform
func (p *Provider) Configure(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)
	log.Info("Configuring Istio for observability platform", "platform", platform.Name)

	// Create ServiceEntry for external services
	if err := p.createServiceEntries(ctx, platform); err != nil {
		return fmt.Errorf("creating service entries: %w", err)
	}

	// Create VirtualService for traffic management
	if err := p.createVirtualServices(ctx, platform); err != nil {
		return fmt.Errorf("creating virtual services: %w", err)
	}

	// Create DestinationRule for load balancing
	if err := p.createDestinationRules(ctx, platform); err != nil {
		return fmt.Errorf("creating destination rules: %w", err)
	}

	return nil
}

// EnableSidecarInjection enables automatic sidecar injection for a namespace
func (p *Provider) EnableSidecarInjection(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{}
	err := p.client.Get(ctx, types.NamespacedName{Name: namespace}, ns)
	if err != nil {
		return fmt.Errorf("getting namespace: %w", err)
	}

	// Add sidecar injection label
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels["istio-injection"] = "enabled"

	err = p.client.Update(ctx, ns)
	if err != nil {
		return fmt.Errorf("updating namespace: %w", err)
	}

	return nil
}

// ConfigureMTLS configures mutual TLS for the platform components
func (p *Provider) ConfigureMTLS(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Create PeerAuthentication for mTLS
	pa := &istiosecurity.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-mtls", platform.Name),
			Namespace: platform.Namespace,
		},
		Spec: v1beta1.PeerAuthentication{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/platform":    platform.Name,
				},
			},
			Mtls: &v1beta1.PeerAuthentication_MutualTLS{
				Mode: v1beta1.PeerAuthentication_MutualTLS_STRICT,
			},
		},
	}

	// Set owner reference
	if err := client.SetOwnerReference(platform, pa, p.client.Scheme()); err != nil {
		return fmt.Errorf("setting owner reference: %w", err)
	}

	// Create or update PeerAuthentication
	if err := p.client.Create(ctx, pa); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing
			existingPA := &istiosecurity.PeerAuthentication{}
			if err := p.client.Get(ctx, types.NamespacedName{
				Name:      pa.Name,
				Namespace: pa.Namespace,
			}, existingPA); err != nil {
				return fmt.Errorf("getting existing peer authentication: %w", err)
			}
			existingPA.Spec = pa.Spec
			if err := p.client.Update(ctx, existingPA); err != nil {
				return fmt.Errorf("updating peer authentication: %w", err)
			}
		} else {
			return fmt.Errorf("creating peer authentication: %w", err)
		}
	}

	// Create AuthorizationPolicy for access control
	ap := &istiosecurity.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-authz", platform.Name),
			Namespace: platform.Namespace,
		},
		Spec: v1beta1.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/platform":    platform.Name,
				},
			},
			Action: v1beta1.AuthorizationPolicy_ALLOW,
			Rules: []*v1beta1.Rule{
				{
					From: []*v1beta1.Rule_From{
						{
							Source: &v1beta1.Source{
								Namespaces: []string{platform.Namespace},
							},
						},
					},
				},
			},
		},
	}

	// Set owner reference
	if err := client.SetOwnerReference(platform, ap, p.client.Scheme()); err != nil {
		return fmt.Errorf("setting owner reference: %w", err)
	}

	// Create or update AuthorizationPolicy
	if err := p.client.Create(ctx, ap); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing
			existingAP := &istiosecurity.AuthorizationPolicy{}
			if err := p.client.Get(ctx, types.NamespacedName{
				Name:      ap.Name,
				Namespace: ap.Namespace,
			}, existingAP); err != nil {
				return fmt.Errorf("getting existing authorization policy: %w", err)
			}
			existingAP.Spec = ap.Spec
			if err := p.client.Update(ctx, existingAP); err != nil {
				return fmt.Errorf("updating authorization policy: %w", err)
			}
		} else {
			return fmt.Errorf("creating authorization policy: %w", err)
		}
	}

	return nil
}

// CreateTrafficPolicies creates traffic management policies
func (p *Provider) CreateTrafficPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Create retry policies
	if err := p.createRetryPolicies(ctx, platform); err != nil {
		return fmt.Errorf("creating retry policies: %w", err)
	}

	// Create circuit breaker policies
	if err := p.createCircuitBreakerPolicies(ctx, platform); err != nil {
		return fmt.Errorf("creating circuit breaker policies: %w", err)
	}

	// Create timeout policies
	if err := p.createTimeoutPolicies(ctx, platform); err != nil {
		return fmt.Errorf("creating timeout policies: %w", err)
	}

	return nil
}

// GetMetrics retrieves service mesh metrics
func (p *Provider) GetMetrics(ctx context.Context, namespace string) (*servicemesh.ServiceMeshMetrics, error) {
	// This would typically query Prometheus or Istio telemetry API
	// For now, return mock data
	return &servicemesh.ServiceMeshMetrics{
		RequestRate:       1000.0,
		ErrorRate:         5.0,
		P50Latency:        25.0,
		P99Latency:        100.0,
		ActiveConnections: 150,
	}, nil
}

// Cleanup removes Istio configurations
func (p *Provider) Cleanup(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Delete VirtualServices
	vsList := &istionetworking.VirtualServiceList{}
	if err := p.client.List(ctx, vsList, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return fmt.Errorf("listing virtual services: %w", err)
	}
	for _, vs := range vsList.Items {
		if err := p.client.Delete(ctx, &vs); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting virtual service %s: %w", vs.Name, err)
		}
	}

	// Delete DestinationRules
	drList := &istionetworking.DestinationRuleList{}
	if err := p.client.List(ctx, drList, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return fmt.Errorf("listing destination rules: %w", err)
	}
	for _, dr := range drList.Items {
		if err := p.client.Delete(ctx, &dr); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting destination rule %s: %w", dr.Name, err)
		}
	}

	// Delete PeerAuthentication
	paList := &istiosecurity.PeerAuthenticationList{}
	if err := p.client.List(ctx, paList, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return fmt.Errorf("listing peer authentications: %w", err)
	}
	for _, pa := range paList.Items {
		if err := p.client.Delete(ctx, &pa); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting peer authentication %s: %w", pa.Name, err)
		}
	}

	// Delete AuthorizationPolicies
	apList := &istiosecurity.AuthorizationPolicyList{}
	if err := p.client.List(ctx, apList, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return fmt.Errorf("listing authorization policies: %w", err)
	}
	for _, ap := range apList.Items {
		if err := p.client.Delete(ctx, &ap); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting authorization policy %s: %w", ap.Name, err)
		}
	}

	return nil
}

// Helper functions

func (p *Provider) createServiceEntries(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Create ServiceEntry for external services if needed
	// This would be implemented based on platform configuration
	return nil
}

func (p *Provider) createVirtualServices(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Create VirtualService for Prometheus
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		vs := p.buildVirtualService(platform, "prometheus", 9090)
		if err := p.createOrUpdateVirtualService(ctx, vs); err != nil {
			return fmt.Errorf("creating prometheus virtual service: %w", err)
		}
	}

	// Create VirtualService for Grafana
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		vs := p.buildVirtualService(platform, "grafana", 3000)
		if err := p.createOrUpdateVirtualService(ctx, vs); err != nil {
			return fmt.Errorf("creating grafana virtual service: %w", err)
		}
	}

	return nil
}

func (p *Provider) buildVirtualService(platform *observabilityv1.ObservabilityPlatform, component string, port int) *istionetworking.VirtualService {
	return &istionetworking.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-vs", platform.Name, component),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"observability.io/platform":    platform.Name,
				"observability.io/component":   component,
			},
		},
		Spec: v1beta1.VirtualService{
			Hosts: []string{
				fmt.Sprintf("%s-%s", platform.Name, component),
				fmt.Sprintf("%s-%s.%s.svc.cluster.local", platform.Name, component, platform.Namespace),
			},
			Http: []*v1beta1.HTTPRoute{
				{
					Match: []*v1beta1.HTTPMatchRequest{
						{
							Port: uint32(port),
						},
					},
					Route: []*v1beta1.HTTPRouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: fmt.Sprintf("%s-%s.%s.svc.cluster.local", platform.Name, component, platform.Namespace),
								Port: &v1beta1.PortSelector{
									Number: uint32(port),
								},
							},
						},
					},
					Timeout: "30s",
					Retries: &v1beta1.HTTPRetry{
						Attempts:      3,
						PerTryTimeout: "10s",
					},
				},
			},
		},
	}
}

func (p *Provider) createOrUpdateVirtualService(ctx context.Context, vs *istionetworking.VirtualService) error {
	if err := p.client.Create(ctx, vs); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing
			existing := &istionetworking.VirtualService{}
			if err := p.client.Get(ctx, types.NamespacedName{
				Name:      vs.Name,
				Namespace: vs.Namespace,
			}, existing); err != nil {
				return err
			}
			existing.Spec = vs.Spec
			return p.client.Update(ctx, existing)
		}
		return err
	}
	return nil
}

func (p *Provider) createDestinationRules(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Create DestinationRule for each component
	components := []struct {
		name    string
		enabled bool
		port    int
	}{
		{"prometheus", platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled, 9090},
		{"grafana", platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled, 3000},
		{"loki", platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled, 3100},
		{"tempo", platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled, 3200},
	}

	for _, comp := range components {
		if !comp.enabled {
			continue
		}

		dr := &istionetworking.DestinationRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-dr", platform.Name, comp.name),
				Namespace: platform.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/platform":    platform.Name,
					"observability.io/component":   comp.name,
				},
			},
			Spec: v1beta1.DestinationRule{
				Host: fmt.Sprintf("%s-%s.%s.svc.cluster.local", platform.Name, comp.name, platform.Namespace),
				TrafficPolicy: &v1beta1.TrafficPolicy{
					ConnectionPool: &v1beta1.ConnectionPoolSettings{
						Tcp: &v1beta1.ConnectionPoolSettings_TCPSettings{
							MaxConnections: 100,
							ConnectTimeout: "30s",
						},
						Http: &v1beta1.ConnectionPoolSettings_HTTPSettings{
							Http1MaxPendingRequests:  10,
							Http2MaxRequests:         100,
							MaxRequestsPerConnection: 2,
						},
					},
					LoadBalancer: &v1beta1.LoadBalancerSettings{
						LbPolicy: &v1beta1.LoadBalancerSettings_Simple{
							Simple: v1beta1.LoadBalancerSettings_ROUND_ROBIN,
						},
					},
					OutlierDetection: &v1beta1.OutlierDetection{
						ConsecutiveErrors:       5,
						Interval:                "30s",
						BaseEjectionTime:        "30s",
						MaxEjectionPercent:      50,
						MinHealthPercent:        50,
						SplitExternalLocalOriginErrors: true,
					},
				},
			},
		}

		// Set owner reference
		if err := client.SetOwnerReference(platform, dr, p.client.Scheme()); err != nil {
			return fmt.Errorf("setting owner reference: %w", err)
		}

		if err := p.createOrUpdateDestinationRule(ctx, dr); err != nil {
			return fmt.Errorf("creating destination rule for %s: %w", comp.name, err)
		}
	}

	return nil
}

func (p *Provider) createOrUpdateDestinationRule(ctx context.Context, dr *istionetworking.DestinationRule) error {
	if err := p.client.Create(ctx, dr); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing
			existing := &istionetworking.DestinationRule{}
			if err := p.client.Get(ctx, types.NamespacedName{
				Name:      dr.Name,
				Namespace: dr.Namespace,
			}, existing); err != nil {
				return err
			}
			existing.Spec = dr.Spec
			return p.client.Update(ctx, existing)
		}
		return err
	}
	return nil
}

func (p *Provider) createRetryPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Retry policies are configured in VirtualServices
	return nil
}

func (p *Provider) createCircuitBreakerPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Circuit breaker policies are configured in DestinationRules
	return nil
}

func (p *Provider) createTimeoutPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Timeout policies are configured in VirtualServices
	return nil
}
