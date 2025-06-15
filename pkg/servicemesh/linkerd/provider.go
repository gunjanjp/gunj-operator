// Package linkerd provides Linkerd service mesh integration for the Gunj Operator
package linkerd

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/pkg/servicemesh"
)

// Provider implements ServiceMeshProvider for Linkerd
type Provider struct {
	client client.Client
}

// NewProvider creates a new Linkerd provider
func NewProvider(client client.Client) *Provider {
	return &Provider{
		client: client,
	}
}

// IsInstalled checks if Linkerd is installed in the cluster
func (p *Provider) IsInstalled(ctx context.Context) (bool, error) {
	// Check if linkerd namespace exists
	namespace := &corev1.Namespace{}
	err := p.client.Get(ctx, types.NamespacedName{Name: "linkerd"}, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Check if linkerd-destination deployment exists
	deploymentList := &corev1.PodList{}
	err = p.client.List(ctx, deploymentList, client.InNamespace("linkerd"), client.MatchingLabels{
		"linkerd.io/control-plane-component": "destination",
	})
	if err != nil {
		return false, err
	}

	return len(deploymentList.Items) > 0, nil
}

// Configure configures Linkerd for an observability platform
func (p *Provider) Configure(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)
	log.Info("Configuring Linkerd for observability platform", "platform", platform.Name)

	// Create ServiceProfiles for components
	if err := p.createServiceProfiles(ctx, platform); err != nil {
		return fmt.Errorf("creating service profiles: %w", err)
	}

	// Create TrafficSplits for canary deployments if needed
	if err := p.createTrafficSplits(ctx, platform); err != nil {
		return fmt.Errorf("creating traffic splits: %w", err)
	}

	// Configure retry and timeout policies
	if err := p.configureServicePolicies(ctx, platform); err != nil {
		return fmt.Errorf("configuring service policies: %w", err)
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

	// Add Linkerd injection annotation
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	ns.Annotations["linkerd.io/inject"] = "enabled"

	err = p.client.Update(ctx, ns)
	if err != nil {
		return fmt.Errorf("updating namespace: %w", err)
	}

	return nil
}

// ConfigureMTLS configures mutual TLS for the platform components
func (p *Provider) ConfigureMTLS(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Linkerd provides automatic mTLS, but we can configure policies
	
	// Create Server for mTLS enforcement
	server := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-linkerd-server", platform.Name),
			Namespace: platform.Namespace,
		},
		Data: map[string]string{
			"server.yaml": fmt.Sprintf(`
apiVersion: policy.linkerd.io/v1beta1
kind: Server
metadata:
  name: %s-server
  namespace: %s
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: gunj-operator
      observability.io/platform: %s
  port: all
  proxyProtocol: "HTTP/2"`, platform.Name, platform.Namespace, platform.Name),
		},
	}

	// Set owner reference
	if err := client.SetOwnerReference(platform, server, p.client.Scheme()); err != nil {
		return fmt.Errorf("setting owner reference: %w", err)
	}

	// Create or update Server configuration
	if err := p.client.Create(ctx, server); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing
			existingServer := &corev1.ConfigMap{}
			if err := p.client.Get(ctx, types.NamespacedName{
				Name:      server.Name,
				Namespace: server.Namespace,
			}, existingServer); err != nil {
				return fmt.Errorf("getting existing server config: %w", err)
			}
			existingServer.Data = server.Data
			if err := p.client.Update(ctx, existingServer); err != nil {
				return fmt.Errorf("updating server config: %w", err)
			}
		} else {
			return fmt.Errorf("creating server config: %w", err)
		}
	}

	// Create ServerAuthorization for access control
	serverAuth := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-linkerd-serverauth", platform.Name),
			Namespace: platform.Namespace,
		},
		Data: map[string]string{
			"serverauth.yaml": fmt.Sprintf(`
apiVersion: policy.linkerd.io/v1beta1
kind: ServerAuthorization
metadata:
  name: %s-serverauth
  namespace: %s
spec:
  server:
    name: %s-server
  client:
    meshTLS:
      identities:
      - "%s.%s.serviceaccount.identity.linkerd.cluster.local"`, 
				platform.Name, platform.Namespace, platform.Name, platform.Name, platform.Namespace),
		},
	}

	// Set owner reference
	if err := client.SetOwnerReference(platform, serverAuth, p.client.Scheme()); err != nil {
		return fmt.Errorf("setting owner reference: %w", err)
	}

	// Create or update ServerAuthorization
	if err := p.client.Create(ctx, serverAuth); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing
			existingAuth := &corev1.ConfigMap{}
			if err := p.client.Get(ctx, types.NamespacedName{
				Name:      serverAuth.Name,
				Namespace: serverAuth.Namespace,
			}, existingAuth); err != nil {
				return fmt.Errorf("getting existing server auth: %w", err)
			}
			existingAuth.Data = serverAuth.Data
			if err := p.client.Update(ctx, existingAuth); err != nil {
				return fmt.Errorf("updating server auth: %w", err)
			}
		} else {
			return fmt.Errorf("creating server auth: %w", err)
		}
	}

	return nil
}

// CreateTrafficPolicies creates traffic management policies
func (p *Provider) CreateTrafficPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Create retry budgets
	if err := p.createRetryBudgets(ctx, platform); err != nil {
		return fmt.Errorf("creating retry budgets: %w", err)
	}

	// Create circuit breaker configuration
	if err := p.createCircuitBreakers(ctx, platform); err != nil {
		return fmt.Errorf("creating circuit breakers: %w", err)
	}

	// Create timeout policies
	if err := p.createTimeoutPolicies(ctx, platform); err != nil {
		return fmt.Errorf("creating timeout policies: %w", err)
	}

	return nil
}

// GetMetrics retrieves service mesh metrics
func (p *Provider) GetMetrics(ctx context.Context, namespace string) (*servicemesh.ServiceMeshMetrics, error) {
	// This would typically query Prometheus with Linkerd metrics
	// For now, return mock data
	return &servicemesh.ServiceMeshMetrics{
		RequestRate:       800.0,
		ErrorRate:         3.0,
		P50Latency:        20.0,
		P99Latency:        85.0,
		ActiveConnections: 120,
	}, nil
}

// Cleanup removes Linkerd configurations
func (p *Provider) Cleanup(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Delete ServiceProfiles
	spList := &corev1.ConfigMapList{}
	if err := p.client.List(ctx, spList, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
		"linkerd.io/resource-type":  "service-profile",
	}); err != nil {
		return fmt.Errorf("listing service profiles: %w", err)
	}
	for _, sp := range spList.Items {
		if err := p.client.Delete(ctx, &sp); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting service profile %s: %w", sp.Name, err)
		}
	}

	// Delete TrafficSplits
	tsList := &corev1.ConfigMapList{}
	if err := p.client.List(ctx, tsList, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
		"linkerd.io/resource-type":  "traffic-split",
	}); err != nil {
		return fmt.Errorf("listing traffic splits: %w", err)
	}
	for _, ts := range tsList.Items {
		if err := p.client.Delete(ctx, &ts); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting traffic split %s: %w", ts.Name, err)
		}
	}

	// Remove injection annotation from namespace
	ns := &corev1.Namespace{}
	if err := p.client.Get(ctx, types.NamespacedName{Name: platform.Namespace}, ns); err != nil {
		return fmt.Errorf("getting namespace: %w", err)
	}
	
	if ns.Annotations != nil {
		delete(ns.Annotations, "linkerd.io/inject")
		if err := p.client.Update(ctx, ns); err != nil {
			return fmt.Errorf("updating namespace: %w", err)
		}
	}

	return nil
}

// Helper functions

func (p *Provider) createServiceProfiles(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Create ServiceProfile for each component
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

		sp := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-sp", platform.Name, comp.name),
				Namespace: platform.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/platform":    platform.Name,
					"observability.io/component":   comp.name,
					"linkerd.io/resource-type":     "service-profile",
				},
			},
			Data: map[string]string{
				"service-profile.yaml": fmt.Sprintf(`
apiVersion: linkerd.io/v1alpha2
kind: ServiceProfile
metadata:
  name: %s-%s.%s.svc.cluster.local
  namespace: %s
spec:
  routes:
  - name: metrics
    condition:
      method: GET
      pathRegex: "/metrics"
    timeout: 30s
  - name: api
    condition:
      pathRegex: "/api/.*"
    timeout: 15s
    isRetryable: true
    retryBudget:
      retryRatio: 0.2
      minRetriesPerSecond: 10
      ttl: 10s
  retryBudget:
    retryRatio: 0.2
    minRetriesPerSecond: 10
    ttl: 10s`, platform.Name, comp.name, platform.Namespace, platform.Namespace),
			},
		}

		// Set owner reference
		if err := client.SetOwnerReference(platform, sp, p.client.Scheme()); err != nil {
			return fmt.Errorf("setting owner reference: %w", err)
		}

		if err := p.createOrUpdateConfigMap(ctx, sp); err != nil {
			return fmt.Errorf("creating service profile for %s: %w", comp.name, err)
		}
	}

	return nil
}

func (p *Provider) createTrafficSplits(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Traffic splits would be created if canary deployments are configured
	// For now, this is a placeholder
	return nil
}

func (p *Provider) configureServicePolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Configure retry and timeout policies through ServiceProfiles
	// Already handled in createServiceProfiles
	return nil
}

func (p *Provider) createRetryBudgets(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Retry budgets are configured in ServiceProfiles
	// Already handled in createServiceProfiles
	return nil
}

func (p *Provider) createCircuitBreakers(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Linkerd doesn't have explicit circuit breakers like Istio
	// It uses retry budgets and timeouts for similar functionality
	return nil
}

func (p *Provider) createTimeoutPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Timeout policies are configured in ServiceProfiles
	// Already handled in createServiceProfiles
	return nil
}

func (p *Provider) createOrUpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	if err := p.client.Create(ctx, cm); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing
			existing := &corev1.ConfigMap{}
			if err := p.client.Get(ctx, types.NamespacedName{
				Name:      cm.Name,
				Namespace: cm.Namespace,
			}, existing); err != nil {
				return err
			}
			existing.Data = cm.Data
			return p.client.Update(ctx, existing)
		}
		return err
	}
	return nil
}

// createNetworkPolicies creates network policies for zero-trust networking
func (p *Provider) createNetworkPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Create default deny-all policy
	denyAll := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-deny-all", platform.Name),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"observability.io/platform":    platform.Name,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"observability.io/platform": platform.Name,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	// Set owner reference
	if err := client.SetOwnerReference(platform, denyAll, p.client.Scheme()); err != nil {
		return fmt.Errorf("setting owner reference: %w", err)
	}

	if err := p.createOrUpdateNetworkPolicy(ctx, denyAll); err != nil {
		return fmt.Errorf("creating deny-all network policy: %w", err)
	}

	// Create allow policies for each component
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

		np := &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-allow", platform.Name, comp.name),
				Namespace: platform.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/platform":    platform.Name,
					"observability.io/component":   comp.name,
				},
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"observability.io/platform":  platform.Name,
						"observability.io/component": comp.name,
					},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeIngress,
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"observability.io/platform": platform.Name,
									},
								},
							},
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"linkerd.io/control-plane-ns": "linkerd",
									},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: func() *corev1.Protocol { p := corev1.ProtocolTCP; return &p }(),
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: int32(comp.port)},
							},
						},
					},
				},
			},
		}

		// Set owner reference
		if err := client.SetOwnerReference(platform, np, p.client.Scheme()); err != nil {
			return fmt.Errorf("setting owner reference: %w", err)
		}

		if err := p.createOrUpdateNetworkPolicy(ctx, np); err != nil {
			return fmt.Errorf("creating network policy for %s: %w", comp.name, err)
		}
	}

	return nil
}

func (p *Provider) createOrUpdateNetworkPolicy(ctx context.Context, np *networkingv1.NetworkPolicy) error {
	if err := p.client.Create(ctx, np); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing
			existing := &networkingv1.NetworkPolicy{}
			if err := p.client.Get(ctx, types.NamespacedName{
				Name:      np.Name,
				Namespace: np.Namespace,
			}, existing); err != nil {
				return err
			}
			existing.Spec = np.Spec
			return p.client.Update(ctx, existing)
		}
		return err
	}
	return nil
}
