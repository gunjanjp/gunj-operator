package istio

import (
	"context"
	"fmt"
	"time"
	
	"google.golang.org/protobuf/types/known/durationpb"
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	securityv1beta1 "istio.io/api/security/v1beta1"
	telemetryv1alpha1 "istio.io/api/telemetry/v1alpha1"
	istioclientv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istioclientv1beta1security "istio.io/client-go/pkg/apis/security/v1beta1"
	istioclientv1alpha1telemetry "istio.io/client-go/pkg/apis/telemetry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	
	"github.com/gunjanjp/gunj-operator/internal/servicemesh"
)

// IstioManager implements the Manager interface for Istio
type IstioManager struct {
	client client.Client
}

// NewIstioManager creates a new Istio manager
func NewIstioManager(client client.Client) servicemesh.Manager {
	return &IstioManager{
		client: client,
	}
}

// Configure applies Istio configuration to the platform
func (m *IstioManager) Configure(ctx context.Context, namespace string, config *servicemesh.ServiceMeshConfig) error {
	log := log.FromContext(ctx).WithValues("namespace", namespace, "meshType", "istio")
	log.V(1).Info("Configuring Istio for namespace")
	
	// Enable sidecar injection for the namespace
	if err := m.enableSidecarInjection(ctx, namespace, config); err != nil {
		return fmt.Errorf("enabling sidecar injection: %w", err)
	}
	
	// Configure default PeerAuthentication if mTLS is enabled
	if config.MTLS.Strict || config.MTLS.Mode != "" {
		if err := m.configureDefaultMTLS(ctx, namespace, config); err != nil {
			return fmt.Errorf("configuring default mTLS: %w", err)
		}
	}
	
	// Configure default telemetry if enabled
	if config.Telemetry.Metrics.Enabled || config.Telemetry.Tracing.Enabled || config.Telemetry.AccessLogging.Enabled {
		if err := m.configureDefaultTelemetry(ctx, namespace, config); err != nil {
			return fmt.Errorf("configuring default telemetry: %w", err)
		}
	}
	
	// Configure default destination rules for traffic management
	if config.TrafficManagement.CircuitBreaker != nil || config.TrafficManagement.LoadBalancer != nil {
		if err := m.configureDefaultTrafficManagement(ctx, namespace, config); err != nil {
			return fmt.Errorf("configuring default traffic management: %w", err)
		}
	}
	
	return nil
}

// enableSidecarInjection enables automatic sidecar injection for the namespace
func (m *IstioManager) enableSidecarInjection(ctx context.Context, namespace string, config *servicemesh.ServiceMeshConfig) error {
	log := log.FromContext(ctx)
	
	// Get the namespace
	ns := &corev1.Namespace{}
	if err := m.client.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
		return fmt.Errorf("getting namespace: %w", err)
	}
	
	// Add sidecar injection label
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	
	injectLabel := "istio-injection"
	if config.IstioConfig != nil && config.IstioConfig.Sidecar != nil && config.IstioConfig.Sidecar.InjectAnnotation != "" {
		injectLabel = config.IstioConfig.Sidecar.InjectAnnotation
	}
	
	ns.Labels[injectLabel] = "enabled"
	
	// Update namespace
	if err := m.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("updating namespace labels: %w", err)
	}
	
	log.V(1).Info("Enabled sidecar injection for namespace", "label", injectLabel)
	return nil
}

// configureDefaultMTLS creates default PeerAuthentication for mTLS
func (m *IstioManager) configureDefaultMTLS(ctx context.Context, namespace string, config *servicemesh.ServiceMeshConfig) error {
	log := log.FromContext(ctx)
	
	mode := "PERMISSIVE"
	if config.MTLS.Strict {
		mode = "STRICT"
	} else if config.MTLS.Mode != "" {
		mode = config.MTLS.Mode
	}
	
	pa := &istioclientv1beta1security.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"app.kubernetes.io/component":  "service-mesh",
			},
		},
		Spec: securityv1beta1.PeerAuthentication{
			Mtls: &securityv1beta1.PeerAuthentication_MutualTLS{
				Mode: securityv1beta1.PeerAuthentication_MutualTLS_Mode(
					securityv1beta1.PeerAuthentication_MutualTLS_Mode_value[mode],
				),
			},
		},
	}
	
	if err := m.client.Create(ctx, pa); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating default PeerAuthentication: %w", err)
		}
		// Update existing
		existing := &istioclientv1beta1security.PeerAuthentication{}
		if err := m.client.Get(ctx, client.ObjectKeyFromObject(pa), existing); err != nil {
			return fmt.Errorf("getting existing PeerAuthentication: %w", err)
		}
		existing.Spec = pa.Spec
		if err := m.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("updating PeerAuthentication: %w", err)
		}
	}
	
	log.V(1).Info("Configured default mTLS", "mode", mode)
	return nil
}

// configureDefaultTelemetry creates default telemetry configuration
func (m *IstioManager) configureDefaultTelemetry(ctx context.Context, namespace string, config *servicemesh.ServiceMeshConfig) error {
	log := log.FromContext(ctx)
	
	telemetry := &istioclientv1alpha1telemetry.Telemetry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"app.kubernetes.io/component":  "service-mesh",
			},
		},
		Spec: telemetryv1alpha1.Telemetry{},
	}
	
	// Configure metrics
	if config.Telemetry.Metrics.Enabled {
		telemetry.Spec.Metrics = []*telemetryv1alpha1.Metrics{
			{
				Providers: m.buildProviderRefs(config.Telemetry.Metrics.Providers),
			},
		}
		
		// Add dimensions if specified
		if len(config.Telemetry.Metrics.Dimensions) > 0 {
			telemetry.Spec.Metrics[0].Dimensions = config.Telemetry.Metrics.Dimensions
		}
	}
	
	// Configure tracing
	if config.Telemetry.Tracing.Enabled {
		telemetry.Spec.Tracing = []*telemetryv1alpha1.Tracing{
			{
				Providers:                m.buildProviderRefs([]string{config.Telemetry.Tracing.Provider}),
				RandomSamplingPercentage: &config.Telemetry.Tracing.SamplingRate,
			},
		}
	}
	
	// Configure access logging
	if config.Telemetry.AccessLogging.Enabled {
		telemetry.Spec.AccessLogging = []*telemetryv1alpha1.AccessLogging{
			{
				Providers: m.buildProviderRefs(config.Telemetry.AccessLogging.Providers),
			},
		}
	}
	
	if err := m.client.Create(ctx, telemetry); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating default Telemetry: %w", err)
		}
		// Update existing
		existing := &istioclientv1alpha1telemetry.Telemetry{}
		if err := m.client.Get(ctx, client.ObjectKeyFromObject(telemetry), existing); err != nil {
			return fmt.Errorf("getting existing Telemetry: %w", err)
		}
		existing.Spec = telemetry.Spec
		if err := m.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("updating Telemetry: %w", err)
		}
	}
	
	log.V(1).Info("Configured default telemetry")
	return nil
}

// configureDefaultTrafficManagement creates default traffic management rules
func (m *IstioManager) configureDefaultTrafficManagement(ctx context.Context, namespace string, config *servicemesh.ServiceMeshConfig) error {
	log := log.FromContext(ctx)
	
	// This would typically create a default DestinationRule for services in the namespace
	// For now, we'll skip this as it's usually better to create specific rules per service
	
	log.V(1).Info("Traffic management configuration would be applied per service")
	return nil
}

// CreateVirtualService creates a virtual service for traffic management
func (m *IstioManager) CreateVirtualService(ctx context.Context, namespace, name string, spec *servicemesh.VirtualServiceSpec) error {
	log := log.FromContext(ctx).WithValues("namespace", namespace, "virtualService", name)
	log.V(1).Info("Creating VirtualService")
	
	vs := &istioclientv1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"app.kubernetes.io/component":  "service-mesh",
			},
		},
		Spec: networkingv1beta1.VirtualService{
			Hosts:    spec.Hosts,
			Gateways: spec.Gateways,
			Http:     m.convertHTTPRoutes(spec.HTTP),
		},
	}
	
	if err := m.client.Create(ctx, vs); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating VirtualService: %w", err)
		}
		// Update existing
		existing := &istioclientv1beta1.VirtualService{}
		if err := m.client.Get(ctx, client.ObjectKeyFromObject(vs), existing); err != nil {
			return fmt.Errorf("getting existing VirtualService: %w", err)
		}
		existing.Spec = vs.Spec
		if err := m.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("updating VirtualService: %w", err)
		}
	}
	
	log.Info("Created VirtualService successfully")
	return nil
}

// CreateDestinationRule creates a destination rule for traffic management
func (m *IstioManager) CreateDestinationRule(ctx context.Context, namespace, name string, spec *servicemesh.DestinationRuleSpec) error {
	log := log.FromContext(ctx).WithValues("namespace", namespace, "destinationRule", name)
	log.V(1).Info("Creating DestinationRule")
	
	dr := &istioclientv1beta1.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"app.kubernetes.io/component":  "service-mesh",
			},
		},
		Spec: networkingv1beta1.DestinationRule{
			Host:          spec.Host,
			TrafficPolicy: m.convertTrafficPolicy(spec.TrafficPolicy),
			Subsets:       m.convertSubsets(spec.Subsets),
		},
	}
	
	if err := m.client.Create(ctx, dr); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating DestinationRule: %w", err)
		}
		// Update existing
		existing := &istioclientv1beta1.DestinationRule{}
		if err := m.client.Get(ctx, client.ObjectKeyFromObject(dr), existing); err != nil {
			return fmt.Errorf("getting existing DestinationRule: %w", err)
		}
		existing.Spec = dr.Spec
		if err := m.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("updating DestinationRule: %w", err)
		}
	}
	
	log.Info("Created DestinationRule successfully")
	return nil
}

// CreatePeerAuthentication creates peer authentication for mTLS
func (m *IstioManager) CreatePeerAuthentication(ctx context.Context, namespace, name string, spec *servicemesh.PeerAuthenticationSpec) error {
	log := log.FromContext(ctx).WithValues("namespace", namespace, "peerAuthentication", name)
	log.V(1).Info("Creating PeerAuthentication")
	
	pa := &istioclientv1beta1security.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"app.kubernetes.io/component":  "service-mesh",
			},
		},
		Spec: securityv1beta1.PeerAuthentication{},
	}
	
	// Set selector if provided
	if spec.Selector != nil && len(spec.Selector.MatchLabels) > 0 {
		pa.Spec.Selector = &networkingv1beta1.WorkloadSelector{
			MatchLabels: spec.Selector.MatchLabels,
		}
	}
	
	// Set mTLS mode
	if spec.MutualTLS != nil && spec.MutualTLS.Mode != "" {
		pa.Spec.Mtls = &securityv1beta1.PeerAuthentication_MutualTLS{
			Mode: securityv1beta1.PeerAuthentication_MutualTLS_Mode(
				securityv1beta1.PeerAuthentication_MutualTLS_Mode_value[spec.MutualTLS.Mode],
			),
		}
	}
	
	if err := m.client.Create(ctx, pa); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating PeerAuthentication: %w", err)
		}
		// Update existing
		existing := &istioclientv1beta1security.PeerAuthentication{}
		if err := m.client.Get(ctx, client.ObjectKeyFromObject(pa), existing); err != nil {
			return fmt.Errorf("getting existing PeerAuthentication: %w", err)
		}
		existing.Spec = pa.Spec
		if err := m.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("updating PeerAuthentication: %w", err)
		}
	}
	
	log.Info("Created PeerAuthentication successfully")
	return nil
}

// CreateTelemetry creates telemetry configuration
func (m *IstioManager) CreateTelemetry(ctx context.Context, namespace, name string, spec *servicemesh.TelemetrySpec) error {
	log := log.FromContext(ctx).WithValues("namespace", namespace, "telemetry", name)
	log.V(1).Info("Creating Telemetry")
	
	telemetry := &istioclientv1alpha1telemetry.Telemetry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"app.kubernetes.io/component":  "service-mesh",
			},
		},
		Spec: telemetryv1alpha1.Telemetry{},
	}
	
	// Configure metrics
	if len(spec.Metrics) > 0 {
		telemetry.Spec.Metrics = make([]*telemetryv1alpha1.Metrics, 0, len(spec.Metrics))
		for _, m := range spec.Metrics {
			metric := &telemetryv1alpha1.Metrics{
				Providers: m.buildProviderRefs(m.convertProviderRefs(m.Providers)),
			}
			if len(m.Dimensions) > 0 {
				metric.Dimensions = m.Dimensions
			}
			telemetry.Spec.Metrics = append(telemetry.Spec.Metrics, metric)
		}
	}
	
	// Configure tracing
	if len(spec.Tracing) > 0 {
		telemetry.Spec.Tracing = make([]*telemetryv1alpha1.Tracing, 0, len(spec.Tracing))
		for _, t := range spec.Tracing {
			tracing := &telemetryv1alpha1.Tracing{
				Providers:                m.buildProviderRefs(m.convertProviderRefs(t.Providers)),
				RandomSamplingPercentage: &t.RandomSamplingPercentage,
			}
			telemetry.Spec.Tracing = append(telemetry.Spec.Tracing, tracing)
		}
	}
	
	// Configure access logging
	if len(spec.AccessLogging) > 0 {
		telemetry.Spec.AccessLogging = make([]*telemetryv1alpha1.AccessLogging, 0, len(spec.AccessLogging))
		for _, al := range spec.AccessLogging {
			telemetry.Spec.AccessLogging = append(telemetry.Spec.AccessLogging, &telemetryv1alpha1.AccessLogging{
				Providers: m.buildProviderRefs(m.convertProviderRefs(al.Providers)),
			})
		}
	}
	
	if err := m.client.Create(ctx, telemetry); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating Telemetry: %w", err)
		}
		// Update existing
		existing := &istioclientv1alpha1telemetry.Telemetry{}
		if err := m.client.Get(ctx, client.ObjectKeyFromObject(telemetry), existing); err != nil {
			return fmt.Errorf("getting existing Telemetry: %w", err)
		}
		existing.Spec = telemetry.Spec
		if err := m.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("updating Telemetry: %w", err)
		}
	}
	
	log.Info("Created Telemetry successfully")
	return nil
}

// Cleanup removes Istio resources
func (m *IstioManager) Cleanup(ctx context.Context, namespace string) error {
	log := log.FromContext(ctx).WithValues("namespace", namespace)
	log.V(1).Info("Cleaning up Istio resources")
	
	// Remove sidecar injection label from namespace
	ns := &corev1.Namespace{}
	if err := m.client.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("getting namespace: %w", err)
		}
		return nil
	}
	
	// Remove Istio labels
	delete(ns.Labels, "istio-injection")
	if err := m.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("updating namespace: %w", err)
	}
	
	// Delete default configurations
	resources := []client.Object{
		&istioclientv1beta1security.PeerAuthentication{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: namespace},
		},
		&istioclientv1alpha1telemetry.Telemetry{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: namespace},
		},
	}
	
	for _, res := range resources {
		if err := m.client.Delete(ctx, res); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete resource", "resource", res.GetObjectKind())
		}
	}
	
	log.Info("Cleaned up Istio resources")
	return nil
}

// GetStatus returns the Istio status
func (m *IstioManager) GetStatus(ctx context.Context, namespace string) (*servicemesh.ServiceMeshStatus, error) {
	status := &servicemesh.ServiceMeshStatus{
		Ready:      true,
		Message:    "Istio is configured",
		Components: make(map[string]servicemesh.ComponentStatus),
	}
	
	// Check if namespace has sidecar injection enabled
	ns := &corev1.Namespace{}
	if err := m.client.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
		status.Ready = false
		status.Message = fmt.Sprintf("Failed to get namespace: %v", err)
		return status, nil
	}
	
	if ns.Labels["istio-injection"] != "enabled" {
		status.Ready = false
		status.Message = "Sidecar injection not enabled"
	}
	
	// Check for default configurations
	pa := &istioclientv1beta1security.PeerAuthentication{}
	if err := m.client.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, pa); err != nil {
		if !errors.IsNotFound(err) {
			status.Components["mTLS"] = servicemesh.ComponentStatus{
				Ready:   false,
				Message: fmt.Sprintf("Failed to get PeerAuthentication: %v", err),
			}
		}
	} else {
		status.Components["mTLS"] = servicemesh.ComponentStatus{
			Ready:   true,
			Message: "PeerAuthentication configured",
		}
	}
	
	telemetry := &istioclientv1alpha1telemetry.Telemetry{}
	if err := m.client.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, telemetry); err != nil {
		if !errors.IsNotFound(err) {
			status.Components["telemetry"] = servicemesh.ComponentStatus{
				Ready:   false,
				Message: fmt.Sprintf("Failed to get Telemetry: %v", err),
			}
		}
	} else {
		status.Components["telemetry"] = servicemesh.ComponentStatus{
			Ready:   true,
			Message: "Telemetry configured",
		}
	}
	
	return status, nil
}

// Helper functions for converting between our types and Istio types

func (m *IstioManager) convertHTTPRoutes(routes []servicemesh.HTTPRouteSpec) []*networkingv1beta1.HTTPRoute {
	result := make([]*networkingv1beta1.HTTPRoute, 0, len(routes))
	for _, r := range routes {
		route := &networkingv1beta1.HTTPRoute{
			Match:   m.convertHTTPMatchRequests(r.Match),
			Route:   m.convertHTTPRouteDestinations(r.Route),
			Timeout: m.convertDuration(r.Timeout),
		}
		
		if r.Retry != nil {
			route.Retries = &networkingv1beta1.HTTPRetry{
				Attempts:      r.Retry.Attempts,
				PerTryTimeout: m.convertDuration(r.Retry.PerTryTimeout),
				RetryOn:       r.Retry.RetryOn,
			}
		}
		
		if r.Fault != nil {
			route.Fault = m.convertHTTPFaultInjection(r.Fault)
		}
		
		result = append(result, route)
	}
	return result
}

func (m *IstioManager) convertHTTPMatchRequests(matches []servicemesh.HTTPMatchRequest) []*networkingv1beta1.HTTPMatchRequest {
	result := make([]*networkingv1beta1.HTTPMatchRequest, 0, len(matches))
	for _, match := range matches {
		m := &networkingv1beta1.HTTPMatchRequest{
			Uri:     m.convertStringMatch(match.URI),
			Headers: m.convertStringMatchMap(match.Headers),
			Method:  m.convertStringMatch(match.Method),
		}
		result = append(result, m)
	}
	return result
}

func (m *IstioManager) convertStringMatch(sm *servicemesh.StringMatch) *networkingv1beta1.StringMatch {
	if sm == nil {
		return nil
	}
	
	result := &networkingv1beta1.StringMatch{}
	if sm.Exact != "" {
		result.MatchType = &networkingv1beta1.StringMatch_Exact{Exact: sm.Exact}
	} else if sm.Prefix != "" {
		result.MatchType = &networkingv1beta1.StringMatch_Prefix{Prefix: sm.Prefix}
	} else if sm.Regex != "" {
		result.MatchType = &networkingv1beta1.StringMatch_Regex{Regex: sm.Regex}
	}
	return result
}

func (m *IstioManager) convertStringMatchMap(headers map[string]*servicemesh.StringMatch) map[string]*networkingv1beta1.StringMatch {
	if len(headers) == 0 {
		return nil
	}
	
	result := make(map[string]*networkingv1beta1.StringMatch)
	for k, v := range headers {
		result[k] = m.convertStringMatch(v)
	}
	return result
}

func (m *IstioManager) convertHTTPRouteDestinations(destinations []servicemesh.HTTPRouteDestination) []*networkingv1beta1.HTTPRouteDestination {
	result := make([]*networkingv1beta1.HTTPRouteDestination, 0, len(destinations))
	for _, dest := range destinations {
		d := &networkingv1beta1.HTTPRouteDestination{
			Weight: dest.Weight,
		}
		
		if dest.Destination != nil {
			d.Destination = &networkingv1beta1.Destination{
				Host:   dest.Destination.Host,
				Subset: dest.Destination.Subset,
			}
			if dest.Destination.Port != nil {
				d.Destination.Port = &networkingv1beta1.PortSelector{
					Number: dest.Destination.Port.Number,
				}
			}
		}
		
		result = append(result, d)
	}
	return result
}

func (m *IstioManager) convertHTTPFaultInjection(fault *servicemesh.HTTPFaultInjection) *networkingv1beta1.HTTPFaultInjection {
	if fault == nil {
		return nil
	}
	
	result := &networkingv1beta1.HTTPFaultInjection{}
	
	if fault.Delay != nil {
		result.Delay = &networkingv1beta1.HTTPFaultInjection_Delay{}
		if fault.Delay.FixedDelay != "" {
			result.Delay.HttpDelayType = &networkingv1beta1.HTTPFaultInjection_Delay_FixedDelay{
				FixedDelay: m.convertDuration(fault.Delay.FixedDelay),
			}
		}
		if fault.Delay.Percentage != nil {
			result.Delay.Percentage = &networkingv1beta1.Percent{
				Value: fault.Delay.Percentage.Value,
			}
		}
	}
	
	if fault.Abort != nil {
		result.Abort = &networkingv1beta1.HTTPFaultInjection_Abort{}
		if fault.Abort.HTTPStatus != 0 {
			result.Abort.ErrorType = &networkingv1beta1.HTTPFaultInjection_Abort_HttpStatus{
				HttpStatus: fault.Abort.HTTPStatus,
			}
		}
		if fault.Abort.Percentage != nil {
			result.Abort.Percentage = &networkingv1beta1.Percent{
				Value: fault.Abort.Percentage.Value,
			}
		}
	}
	
	return result
}

func (m *IstioManager) convertTrafficPolicy(tp *servicemesh.TrafficPolicy) *networkingv1beta1.TrafficPolicy {
	if tp == nil {
		return nil
	}
	
	result := &networkingv1beta1.TrafficPolicy{}
	
	if tp.LoadBalancer != nil {
		result.LoadBalancer = m.convertLoadBalancerSettings(tp.LoadBalancer)
	}
	
	if tp.ConnectionPool != nil {
		result.ConnectionPool = m.convertConnectionPoolSettings(tp.ConnectionPool)
	}
	
	if tp.OutlierDetection != nil {
		result.OutlierDetection = m.convertOutlierDetection(tp.OutlierDetection)
	}
	
	return result
}

func (m *IstioManager) convertLoadBalancerSettings(lb *servicemesh.LoadBalancerSettings) *networkingv1beta1.LoadBalancerSettings {
	if lb == nil {
		return nil
	}
	
	result := &networkingv1beta1.LoadBalancerSettings{}
	
	if lb.Simple != "" {
		switch lb.Simple {
		case "ROUND_ROBIN":
			result.LbPolicy = &networkingv1beta1.LoadBalancerSettings_Simple{
				Simple: networkingv1beta1.LoadBalancerSettings_ROUND_ROBIN,
			}
		case "LEAST_REQUEST":
			result.LbPolicy = &networkingv1beta1.LoadBalancerSettings_Simple{
				Simple: networkingv1beta1.LoadBalancerSettings_LEAST_REQUEST,
			}
		case "RANDOM":
			result.LbPolicy = &networkingv1beta1.LoadBalancerSettings_Simple{
				Simple: networkingv1beta1.LoadBalancerSettings_RANDOM,
			}
		case "PASSTHROUGH":
			result.LbPolicy = &networkingv1beta1.LoadBalancerSettings_Simple{
				Simple: networkingv1beta1.LoadBalancerSettings_PASSTHROUGH,
			}
		}
	} else if lb.ConsistentHash != nil {
		ch := &networkingv1beta1.LoadBalancerSettings_ConsistentHashLB{}
		
		if lb.ConsistentHash.HTTPHeaderName != "" {
			ch.HashKey = &networkingv1beta1.LoadBalancerSettings_ConsistentHashLB_HttpHeaderName{
				HttpHeaderName: lb.ConsistentHash.HTTPHeaderName,
			}
		} else if lb.ConsistentHash.HTTPCookie != nil {
			ch.HashKey = &networkingv1beta1.LoadBalancerSettings_ConsistentHashLB_HttpCookie{
				HttpCookie: &networkingv1beta1.LoadBalancerSettings_ConsistentHashLB_HTTPCookie{
					Name: lb.ConsistentHash.HTTPCookie.Name,
					Path: lb.ConsistentHash.HTTPCookie.Path,
					Ttl:  m.convertDuration(lb.ConsistentHash.HTTPCookie.TTL),
				},
			}
		} else if lb.ConsistentHash.UseSourceIP {
			ch.HashKey = &networkingv1beta1.LoadBalancerSettings_ConsistentHashLB_UseSourceIp{
				UseSourceIp: true,
			}
		}
		
		result.LbPolicy = &networkingv1beta1.LoadBalancerSettings_ConsistentHash{
			ConsistentHash: ch,
		}
	}
	
	return result
}

func (m *IstioManager) convertConnectionPoolSettings(cp *servicemesh.ConnectionPoolSettings) *networkingv1beta1.ConnectionPoolSettings {
	if cp == nil {
		return nil
	}
	
	result := &networkingv1beta1.ConnectionPoolSettings{}
	
	if cp.TCP != nil {
		result.Tcp = &networkingv1beta1.ConnectionPoolSettings_TCPSettings{
			MaxConnections: cp.TCP.MaxConnections,
			ConnectTimeout: m.convertDuration(cp.TCP.ConnectTimeout),
		}
	}
	
	if cp.HTTP != nil {
		result.Http = &networkingv1beta1.ConnectionPoolSettings_HTTPSettings{
			Http1MaxPendingRequests:  cp.HTTP.HTTP1MaxPendingRequests,
			Http2MaxRequests:         cp.HTTP.HTTP2MaxRequests,
			MaxRequestsPerConnection: cp.HTTP.MaxRequestsPerConnection,
		}
	}
	
	return result
}

func (m *IstioManager) convertOutlierDetection(od *servicemesh.OutlierDetection) *networkingv1beta1.OutlierDetection {
	if od == nil {
		return nil
	}
	
	return &networkingv1beta1.OutlierDetection{
		ConsecutiveErrors:  od.ConsecutiveErrors,
		Interval:           m.convertDuration(od.Interval),
		BaseEjectionTime:   m.convertDuration(od.BaseEjectionTime),
		MaxEjectionPercent: od.MaxEjectionPercent,
	}
}

func (m *IstioManager) convertSubsets(subsets []servicemesh.Subset) []*networkingv1beta1.Subset {
	result := make([]*networkingv1beta1.Subset, 0, len(subsets))
	for _, subset := range subsets {
		s := &networkingv1beta1.Subset{
			Name:          subset.Name,
			Labels:        subset.Labels,
			TrafficPolicy: m.convertTrafficPolicy(subset.TrafficPolicy),
		}
		result = append(result, s)
	}
	return result
}

func (m *IstioManager) buildProviderRefs(providers []string) []*telemetryv1alpha1.ProviderRef {
	if len(providers) == 0 {
		return nil
	}
	
	refs := make([]*telemetryv1alpha1.ProviderRef, 0, len(providers))
	for _, p := range providers {
		refs = append(refs, &telemetryv1alpha1.ProviderRef{
			Name: p,
		})
	}
	return refs
}

func (m *IstioManager) convertProviderRefs(refs []servicemesh.ProviderRef) []string {
	result := make([]string, 0, len(refs))
	for _, ref := range refs {
		result = append(result, ref.Name)
	}
	return result
}

func (m *IstioManager) convertDuration(duration string) *durationpb.Duration {
	if duration == "" {
		return nil
	}
	
	d, err := time.ParseDuration(duration)
	if err != nil {
		// Default to 30s if parsing fails
		d = 30 * time.Second
	}
	
	return durationpb.New(d)
}
