package istio

import (
	"fmt"
	"strings"
	"time"
	
	"google.golang.org/protobuf/types/known/durationpb"
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	securityv1beta1 "istio.io/api/security/v1beta1"
	telemetryv1alpha1 "istio.io/api/telemetry/v1alpha1"
	istioclientv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istioclientv1beta1security "istio.io/client-go/pkg/apis/security/v1beta1"
	istioclientv1alpha1telemetry "istio.io/client-go/pkg/apis/telemetry/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// GetServiceName extracts the service name from a host
func GetServiceName(host string) string {
	// Handle FQDN (e.g., service.namespace.svc.cluster.local)
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return host
}

// GetNamespace extracts the namespace from a host
func GetNamespace(host string) string {
	// Handle FQDN (e.g., service.namespace.svc.cluster.local)
	parts := strings.Split(host, ".")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// BuildFQDN builds a fully qualified domain name
func BuildFQDN(service, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", service, namespace)
}

// IsIstioNamespace checks if a namespace has Istio injection enabled
func IsIstioNamespace(labels map[string]string) bool {
	if labels == nil {
		return false
	}
	
	// Check for istio-injection label
	if val, ok := labels["istio-injection"]; ok && val == "enabled" {
		return true
	}
	
	// Check for istio.io/rev label (for revision-based injection)
	if _, ok := labels["istio.io/rev"]; ok {
		return true
	}
	
	return false
}

// ValidateMTLSMode validates if the mTLS mode is valid
func ValidateMTLSMode(mode string) error {
	validModes := map[string]bool{
		"UNSET":      true,
		"DISABLE":    true,
		"PERMISSIVE": true,
		"STRICT":     true,
	}
	
	if !validModes[mode] {
		return fmt.Errorf("invalid mTLS mode: %s, must be one of: UNSET, DISABLE, PERMISSIVE, STRICT", mode)
	}
	
	return nil
}

// ValidateLoadBalancerType validates if the load balancer type is valid
func ValidateLoadBalancerType(lbType string) error {
	validTypes := map[string]bool{
		"ROUND_ROBIN":   true,
		"LEAST_REQUEST": true,
		"RANDOM":        true,
		"PASSTHROUGH":   true,
	}
	
	if !validTypes[lbType] {
		return fmt.Errorf("invalid load balancer type: %s, must be one of: ROUND_ROBIN, LEAST_REQUEST, RANDOM, PASSTHROUGH", lbType)
	}
	
	return nil
}

// ConvertToIstioLabels converts generic labels to Istio-compatible labels
func ConvertToIstioLabels(genericLabels map[string]string) map[string]string {
	istioLabels := make(map[string]string)
	
	for k, v := range genericLabels {
		// Convert common labels to Istio conventions
		switch k {
		case "app.kubernetes.io/name":
			istioLabels["app"] = v
		case "app.kubernetes.io/version":
			istioLabels["version"] = v
		default:
			istioLabels[k] = v
		}
	}
	
	return istioLabels
}

// MergeMaps merges multiple maps, with later maps overriding earlier ones
func MergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	
	return result
}

// BuildWorkloadSelector builds a workload selector from labels
func BuildWorkloadSelector(matchLabels map[string]string) *networkingv1beta1.WorkloadSelector {
	if len(matchLabels) == 0 {
		return nil
	}
	
	return &networkingv1beta1.WorkloadSelector{
		MatchLabels: matchLabels,
	}
}

// ListOptions creates ListOptions for querying Istio resources
func ListOptions(namespace string, labelSelector map[string]string) metav1.ListOptions {
	opts := metav1.ListOptions{}
	
	if len(labelSelector) > 0 {
		selector := labels.SelectorFromSet(labelSelector)
		opts.LabelSelector = selector.String()
	}
	
	return opts
}

// GenerateResourceName generates a resource name with prefix
func GenerateResourceName(prefix, component, suffix string) string {
	parts := []string{prefix}
	
	if component != "" {
		parts = append(parts, component)
	}
	
	if suffix != "" {
		parts = append(parts, suffix)
	}
	
	return strings.Join(parts, "-")
}

// IstioResourceList represents a list of Istio resources
type IstioResourceList struct {
	VirtualServices      []*istioclientv1beta1.VirtualService
	DestinationRules     []*istioclientv1beta1.DestinationRule
	PeerAuthentications  []*istioclientv1beta1security.PeerAuthentication
	Telemetries         []*istioclientv1alpha1telemetry.Telemetry
}

// NewIstioResourceList creates a new IstioResourceList
func NewIstioResourceList() *IstioResourceList {
	return &IstioResourceList{
		VirtualServices:      []*istioclientv1beta1.VirtualService{},
		DestinationRules:     []*istioclientv1beta1.DestinationRule{},
		PeerAuthentications:  []*istioclientv1beta1security.PeerAuthentication{},
		Telemetries:         []*istioclientv1alpha1telemetry.Telemetry{},
	}
}

// Count returns the total number of resources
func (l *IstioResourceList) Count() int {
	return len(l.VirtualServices) + len(l.DestinationRules) + 
		len(l.PeerAuthentications) + len(l.Telemetries)
}

// IsEmpty returns true if there are no resources
func (l *IstioResourceList) IsEmpty() bool {
	return l.Count() == 0
}

// GetVirtualServiceByName returns a VirtualService by name
func (l *IstioResourceList) GetVirtualServiceByName(name string) *istioclientv1beta1.VirtualService {
	for _, vs := range l.VirtualServices {
		if vs.Name == name {
			return vs
		}
	}
	return nil
}

// GetDestinationRuleByHost returns a DestinationRule by host
func (l *IstioResourceList) GetDestinationRuleByHost(host string) *istioclientv1beta1.DestinationRule {
	for _, dr := range l.DestinationRules {
		if dr.Spec.Host == host {
			return dr
		}
	}
	return nil
}

// IstioConfig represents common Istio configuration patterns
type IstioConfig struct {
	// EnableAutoMTLS enables automatic mTLS for the mesh
	EnableAutoMTLS bool
	
	// DefaultRetryPolicy is the default retry policy
	DefaultRetryPolicy *networkingv1beta1.HTTPRetry
	
	// DefaultTimeout is the default timeout
	DefaultTimeout string
	
	// DefaultCircuitBreaker is the default circuit breaker
	DefaultCircuitBreaker *networkingv1beta1.OutlierDetection
	
	// PrometheusEnabled indicates if Prometheus metrics are enabled
	PrometheusEnabled bool
	
	// TracingEnabled indicates if distributed tracing is enabled
	TracingEnabled bool
	
	// TracingSamplingRate is the tracing sampling rate
	TracingSamplingRate float64
}

// DefaultIstioConfig returns a default Istio configuration
func DefaultIstioConfig() *IstioConfig {
	return &IstioConfig{
		EnableAutoMTLS: true,
		DefaultRetryPolicy: &networkingv1beta1.HTTPRetry{
			Attempts:      3,
			PerTryTimeout: durationpb.New(2 * time.Second),
			RetryOn:       "5xx,reset,connect-failure,refused-stream",
		},
		DefaultTimeout: "30s",
		DefaultCircuitBreaker: &networkingv1beta1.OutlierDetection{
			ConsecutiveErrors:  5,
			Interval:           durationpb.New(30 * time.Second),
			BaseEjectionTime:   durationpb.New(30 * time.Second),
			MaxEjectionPercent: 50,
		},
		PrometheusEnabled:   true,
		TracingEnabled:      true,
		TracingSamplingRate: 1.0,
	}
}

// Validation helpers

// ValidateVirtualService validates a VirtualService configuration
func ValidateVirtualService(vs *istioclientv1beta1.VirtualService) error {
	if vs.Name == "" {
		return fmt.Errorf("VirtualService name cannot be empty")
	}
	
	if len(vs.Spec.Hosts) == 0 {
		return fmt.Errorf("VirtualService must have at least one host")
	}
	
	if len(vs.Spec.Http) == 0 && len(vs.Spec.Tcp) == 0 && len(vs.Spec.Tls) == 0 {
		return fmt.Errorf("VirtualService must have at least one route (HTTP, TCP, or TLS)")
	}
	
	return nil
}

// ValidateDestinationRule validates a DestinationRule configuration
func ValidateDestinationRule(dr *istioclientv1beta1.DestinationRule) error {
	if dr.Name == "" {
		return fmt.Errorf("DestinationRule name cannot be empty")
	}
	
	if dr.Spec.Host == "" {
		return fmt.Errorf("DestinationRule host cannot be empty")
	}
	
	// Validate subsets
	subsetNames := make(map[string]bool)
	for _, subset := range dr.Spec.Subsets {
		if subset.Name == "" {
			return fmt.Errorf("subset name cannot be empty")
		}
		if subsetNames[subset.Name] {
			return fmt.Errorf("duplicate subset name: %s", subset.Name)
		}
		subsetNames[subset.Name] = true
		
		if len(subset.Labels) == 0 {
			return fmt.Errorf("subset %s must have at least one label", subset.Name)
		}
	}
	
	return nil
}

// ValidatePeerAuthentication validates a PeerAuthentication configuration
func ValidatePeerAuthentication(pa *istioclientv1beta1security.PeerAuthentication) error {
	if pa.Name == "" {
		return fmt.Errorf("PeerAuthentication name cannot be empty")
	}
	
	if pa.Spec.Mtls != nil {
		mode := pa.Spec.Mtls.Mode.String()
		if err := ValidateMTLSMode(mode); err != nil {
			return err
		}
	}
	
	return nil
}

// ValidateTelemetry validates a Telemetry configuration
func ValidateTelemetry(t *istioclientv1alpha1telemetry.Telemetry) error {
	if t.Name == "" {
		return fmt.Errorf("Telemetry name cannot be empty")
	}
	
	// Validate metrics
	for _, metric := range t.Spec.Metrics {
		if len(metric.Providers) == 0 {
			return fmt.Errorf("metrics configuration must have at least one provider")
		}
	}
	
	// Validate tracing
	for _, tracing := range t.Spec.Tracing {
		if len(tracing.Providers) == 0 {
			return fmt.Errorf("tracing configuration must have at least one provider")
		}
		if tracing.RandomSamplingPercentage != nil {
			rate := *tracing.RandomSamplingPercentage
			if rate < 0 || rate > 100 {
				return fmt.Errorf("tracing sampling rate must be between 0 and 100, got %f", rate)
			}
		}
	}
	
	// Validate access logging
	for _, logging := range t.Spec.AccessLogging {
		if len(logging.Providers) == 0 {
			return fmt.Errorf("access logging configuration must have at least one provider")
		}
	}
	
	return nil
}
