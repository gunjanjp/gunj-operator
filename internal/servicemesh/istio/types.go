package istio

import (
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	securityv1beta1 "istio.io/api/security/v1beta1"
	telemetryv1alpha1 "istio.io/api/telemetry/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	"github.com/gunjanjp/gunj-operator/internal/servicemesh"
)

// VirtualServiceBuilder helps build VirtualService resources
type VirtualServiceBuilder struct {
	name      string
	namespace string
	spec      *servicemesh.VirtualServiceSpec
}

// NewVirtualServiceBuilder creates a new VirtualService builder
func NewVirtualServiceBuilder(name, namespace string) *VirtualServiceBuilder {
	return &VirtualServiceBuilder{
		name:      name,
		namespace: namespace,
		spec: &servicemesh.VirtualServiceSpec{
			HTTP: []servicemesh.HTTPRouteSpec{},
		},
	}
}

// WithHosts sets the hosts
func (b *VirtualServiceBuilder) WithHosts(hosts ...string) *VirtualServiceBuilder {
	b.spec.Hosts = hosts
	return b
}

// WithGateways sets the gateways
func (b *VirtualServiceBuilder) WithGateways(gateways ...string) *VirtualServiceBuilder {
	b.spec.Gateways = gateways
	return b
}

// AddHTTPRoute adds an HTTP route
func (b *VirtualServiceBuilder) AddHTTPRoute(route servicemesh.HTTPRouteSpec) *VirtualServiceBuilder {
	b.spec.HTTP = append(b.spec.HTTP, route)
	return b
}

// AddCanaryRoute adds a canary deployment route
func (b *VirtualServiceBuilder) AddCanaryRoute(stableHost, canaryHost string, canaryWeight int32) *VirtualServiceBuilder {
	route := servicemesh.HTTPRouteSpec{
		Route: []servicemesh.HTTPRouteDestination{
			{
				Destination: &servicemesh.Destination{
					Host: stableHost,
				},
				Weight: 100 - canaryWeight,
			},
			{
				Destination: &servicemesh.Destination{
					Host: canaryHost,
				},
				Weight: canaryWeight,
			},
		},
	}
	return b.AddHTTPRoute(route)
}

// AddRetryPolicy adds a retry policy to the last route
func (b *VirtualServiceBuilder) AddRetryPolicy(attempts int32, perTryTimeout string) *VirtualServiceBuilder {
	if len(b.spec.HTTP) > 0 {
		b.spec.HTTP[len(b.spec.HTTP)-1].Retry = &servicemesh.RetryPolicy{
			Attempts:      attempts,
			PerTryTimeout: perTryTimeout,
			RetryOn:       "5xx,reset,connect-failure,refused-stream",
		}
	}
	return b
}

// AddTimeout adds a timeout to the last route
func (b *VirtualServiceBuilder) AddTimeout(timeout string) *VirtualServiceBuilder {
	if len(b.spec.HTTP) > 0 {
		b.spec.HTTP[len(b.spec.HTTP)-1].Timeout = timeout
	}
	return b
}

// AddFaultInjection adds fault injection to the last route
func (b *VirtualServiceBuilder) AddFaultInjection(delayPercentage float64, delayDuration string) *VirtualServiceBuilder {
	if len(b.spec.HTTP) > 0 {
		b.spec.HTTP[len(b.spec.HTTP)-1].Fault = &servicemesh.HTTPFaultInjection{
			Delay: &servicemesh.HTTPFaultDelay{
				Percentage: &servicemesh.Percentage{Value: delayPercentage},
				FixedDelay: delayDuration,
			},
		}
	}
	return b
}

// Build returns the VirtualServiceSpec
func (b *VirtualServiceBuilder) Build() *servicemesh.VirtualServiceSpec {
	return b.spec
}

// DestinationRuleBuilder helps build DestinationRule resources
type DestinationRuleBuilder struct {
	name      string
	namespace string
	spec      *servicemesh.DestinationRuleSpec
}

// NewDestinationRuleBuilder creates a new DestinationRule builder
func NewDestinationRuleBuilder(name, namespace, host string) *DestinationRuleBuilder {
	return &DestinationRuleBuilder{
		name:      name,
		namespace: namespace,
		spec: &servicemesh.DestinationRuleSpec{
			Host:    host,
			Subsets: []servicemesh.Subset{},
		},
	}
}

// WithLoadBalancer sets the load balancer policy
func (b *DestinationRuleBuilder) WithLoadBalancer(lbType string) *DestinationRuleBuilder {
	if b.spec.TrafficPolicy == nil {
		b.spec.TrafficPolicy = &servicemesh.TrafficPolicy{}
	}
	b.spec.TrafficPolicy.LoadBalancer = &servicemesh.LoadBalancerSettings{
		Simple: lbType,
	}
	return b
}

// WithConsistentHash sets consistent hash load balancing
func (b *DestinationRuleBuilder) WithConsistentHash(headerName string) *DestinationRuleBuilder {
	if b.spec.TrafficPolicy == nil {
		b.spec.TrafficPolicy = &servicemesh.TrafficPolicy{}
	}
	b.spec.TrafficPolicy.LoadBalancer = &servicemesh.LoadBalancerSettings{
		ConsistentHash: &servicemesh.ConsistentHashLB{
			HTTPHeaderName: headerName,
		},
	}
	return b
}

// WithConnectionPool sets connection pool settings
func (b *DestinationRuleBuilder) WithConnectionPool(maxConnections int32, connectTimeout string) *DestinationRuleBuilder {
	if b.spec.TrafficPolicy == nil {
		b.spec.TrafficPolicy = &servicemesh.TrafficPolicy{}
	}
	b.spec.TrafficPolicy.ConnectionPool = &servicemesh.ConnectionPoolSettings{
		TCP: &servicemesh.TCPSettings{
			MaxConnections: maxConnections,
			ConnectTimeout: connectTimeout,
		},
	}
	return b
}

// WithHTTPConnectionPool sets HTTP connection pool settings
func (b *DestinationRuleBuilder) WithHTTPConnectionPool(http1MaxPending, http2MaxRequests int32) *DestinationRuleBuilder {
	if b.spec.TrafficPolicy == nil {
		b.spec.TrafficPolicy = &servicemesh.TrafficPolicy{}
	}
	if b.spec.TrafficPolicy.ConnectionPool == nil {
		b.spec.TrafficPolicy.ConnectionPool = &servicemesh.ConnectionPoolSettings{}
	}
	b.spec.TrafficPolicy.ConnectionPool.HTTP = &servicemesh.HTTPSettings{
		HTTP1MaxPendingRequests: http1MaxPending,
		HTTP2MaxRequests:        http2MaxRequests,
	}
	return b
}

// WithOutlierDetection sets outlier detection
func (b *DestinationRuleBuilder) WithOutlierDetection(consecutiveErrors int32, interval, baseEjectionTime string) *DestinationRuleBuilder {
	if b.spec.TrafficPolicy == nil {
		b.spec.TrafficPolicy = &servicemesh.TrafficPolicy{}
	}
	b.spec.TrafficPolicy.OutlierDetection = &servicemesh.OutlierDetection{
		ConsecutiveErrors: consecutiveErrors,
		Interval:          interval,
		BaseEjectionTime:  baseEjectionTime,
		MaxEjectionPercent: 50,
	}
	return b
}

// AddSubset adds a subset
func (b *DestinationRuleBuilder) AddSubset(name string, labels map[string]string) *DestinationRuleBuilder {
	subset := servicemesh.Subset{
		Name:   name,
		Labels: labels,
	}
	b.spec.Subsets = append(b.spec.Subsets, subset)
	return b
}

// AddSubsetWithPolicy adds a subset with specific traffic policy
func (b *DestinationRuleBuilder) AddSubsetWithPolicy(name string, labels map[string]string, policy *servicemesh.TrafficPolicy) *DestinationRuleBuilder {
	subset := servicemesh.Subset{
		Name:          name,
		Labels:        labels,
		TrafficPolicy: policy,
	}
	b.spec.Subsets = append(b.spec.Subsets, subset)
	return b
}

// Build returns the DestinationRuleSpec
func (b *DestinationRuleBuilder) Build() *servicemesh.DestinationRuleSpec {
	return b.spec
}

// PeerAuthenticationBuilder helps build PeerAuthentication resources
type PeerAuthenticationBuilder struct {
	name      string
	namespace string
	spec      *servicemesh.PeerAuthenticationSpec
}

// NewPeerAuthenticationBuilder creates a new PeerAuthentication builder
func NewPeerAuthenticationBuilder(name, namespace string) *PeerAuthenticationBuilder {
	return &PeerAuthenticationBuilder{
		name:      name,
		namespace: namespace,
		spec:      &servicemesh.PeerAuthenticationSpec{},
	}
}

// WithWorkloadSelector sets the workload selector
func (b *PeerAuthenticationBuilder) WithWorkloadSelector(labels map[string]string) *PeerAuthenticationBuilder {
	b.spec.Selector = &servicemesh.WorkloadSelector{
		MatchLabels: labels,
	}
	return b
}

// WithMTLSMode sets the mTLS mode
func (b *PeerAuthenticationBuilder) WithMTLSMode(mode string) *PeerAuthenticationBuilder {
	b.spec.MutualTLS = &servicemesh.MutualTLS{
		Mode: mode,
	}
	return b
}

// EnableStrictMTLS enables strict mTLS
func (b *PeerAuthenticationBuilder) EnableStrictMTLS() *PeerAuthenticationBuilder {
	return b.WithMTLSMode("STRICT")
}

// EnablePermissiveMTLS enables permissive mTLS
func (b *PeerAuthenticationBuilder) EnablePermissiveMTLS() *PeerAuthenticationBuilder {
	return b.WithMTLSMode("PERMISSIVE")
}

// Build returns the PeerAuthenticationSpec
func (b *PeerAuthenticationBuilder) Build() *servicemesh.PeerAuthenticationSpec {
	return b.spec
}

// TelemetryBuilder helps build Telemetry resources
type TelemetryBuilder struct {
	name      string
	namespace string
	spec      *servicemesh.TelemetrySpec
}

// NewTelemetryBuilder creates a new Telemetry builder
func NewTelemetryBuilder(name, namespace string) *TelemetryBuilder {
	return &TelemetryBuilder{
		name:      name,
		namespace: namespace,
		spec: &servicemesh.TelemetrySpec{
			Metrics:       []servicemesh.MetricSpec{},
			Tracing:       []servicemesh.TracingSpec{},
			AccessLogging: []servicemesh.AccessLoggingSpec{},
		},
	}
}

// AddMetrics adds metrics configuration
func (b *TelemetryBuilder) AddMetrics(providers []string, dimensions map[string]string) *TelemetryBuilder {
	providerRefs := make([]servicemesh.ProviderRef, 0, len(providers))
	for _, p := range providers {
		providerRefs = append(providerRefs, servicemesh.ProviderRef{Name: p})
	}
	
	metric := servicemesh.MetricSpec{
		Providers:  providerRefs,
		Dimensions: dimensions,
	}
	b.spec.Metrics = append(b.spec.Metrics, metric)
	return b
}

// AddTracing adds tracing configuration
func (b *TelemetryBuilder) AddTracing(provider string, samplingRate float64) *TelemetryBuilder {
	tracing := servicemesh.TracingSpec{
		Providers: []servicemesh.ProviderRef{
			{Name: provider},
		},
		RandomSamplingPercentage: samplingRate,
	}
	b.spec.Tracing = append(b.spec.Tracing, tracing)
	return b
}

// AddAccessLogging adds access logging configuration
func (b *TelemetryBuilder) AddAccessLogging(providers []string) *TelemetryBuilder {
	providerRefs := make([]servicemesh.ProviderRef, 0, len(providers))
	for _, p := range providers {
		providerRefs = append(providerRefs, servicemesh.ProviderRef{Name: p})
	}
	
	accessLog := servicemesh.AccessLoggingSpec{
		Providers: providerRefs,
	}
	b.spec.AccessLogging = append(b.spec.AccessLogging, accessLog)
	return b
}

// EnableDefaultMetrics enables default metrics with Prometheus
func (b *TelemetryBuilder) EnableDefaultMetrics() *TelemetryBuilder {
	return b.AddMetrics([]string{"prometheus"}, map[string]string{
		"request_protocol": "request.protocol | \"unknown\"",
		"response_code":    "response.code | 200",
	})
}

// EnableDefaultTracing enables default tracing with Jaeger
func (b *TelemetryBuilder) EnableDefaultTracing(samplingRate float64) *TelemetryBuilder {
	return b.AddTracing("jaeger", samplingRate)
}

// EnableDefaultAccessLogging enables default access logging
func (b *TelemetryBuilder) EnableDefaultAccessLogging() *TelemetryBuilder {
	return b.AddAccessLogging([]string{"otel"})
}

// Build returns the TelemetrySpec
func (b *TelemetryBuilder) Build() *servicemesh.TelemetrySpec {
	return b.spec
}

// Common Istio resource helpers

// CreateDefaultLabels creates default labels for Istio resources
func CreateDefaultLabels(component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "gunj-operator",
		"app.kubernetes.io/component":  component,
		"service-mesh.io/type":        "istio",
	}
}

// CreatePrometheusServiceMonitor creates a ServiceMonitor for Istio metrics
func CreatePrometheusServiceMonitor(name, namespace string, selector map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name + "-istio-metrics",
		Namespace: namespace,
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "gunj-operator",
			"app.kubernetes.io/component":  "monitoring",
			"service-mesh.io/type":        "istio",
			"prometheus.io/scrape":        "true",
		},
	}
}

// LoadBalancerTypes defines available load balancer algorithms
var LoadBalancerTypes = struct {
	RoundRobin   string
	LeastRequest string
	Random       string
	Passthrough  string
}{
	RoundRobin:   "ROUND_ROBIN",
	LeastRequest: "LEAST_REQUEST",
	Random:       "RANDOM",
	Passthrough:  "PASSTHROUGH",
}

// MTLSModes defines available mTLS modes
var MTLSModes = struct {
	Disable     string
	Permissive  string
	Strict      string
	Unset       string
}{
	Disable:    "DISABLE",
	Permissive: "PERMISSIVE",
	Strict:     "STRICT",
	Unset:      "UNSET",
}

// CommonRetryPolicies provides common retry configurations
var CommonRetryPolicies = struct {
	Default     servicemesh.RetryPolicy
	Aggressive  servicemesh.RetryPolicy
	Conservative servicemesh.RetryPolicy
}{
	Default: servicemesh.RetryPolicy{
		Attempts:      3,
		PerTryTimeout: "2s",
		RetryOn:       "5xx,reset,connect-failure,refused-stream",
	},
	Aggressive: servicemesh.RetryPolicy{
		Attempts:      5,
		PerTryTimeout: "5s",
		RetryOn:       "5xx,reset,connect-failure,refused-stream,retriable-4xx",
	},
	Conservative: servicemesh.RetryPolicy{
		Attempts:      2,
		PerTryTimeout: "1s",
		RetryOn:       "5xx",
	},
}

// CommonCircuitBreakers provides common circuit breaker configurations
var CommonCircuitBreakers = struct {
	Default     servicemesh.OutlierDetection
	Aggressive  servicemesh.OutlierDetection
	Conservative servicemesh.OutlierDetection
}{
	Default: servicemesh.OutlierDetection{
		ConsecutiveErrors:  5,
		Interval:           "30s",
		BaseEjectionTime:   "30s",
		MaxEjectionPercent: 50,
	},
	Aggressive: servicemesh.OutlierDetection{
		ConsecutiveErrors:  3,
		Interval:           "10s",
		BaseEjectionTime:   "60s",
		MaxEjectionPercent: 70,
	},
	Conservative: servicemesh.OutlierDetection{
		ConsecutiveErrors:  10,
		Interval:           "60s",
		BaseEjectionTime:   "15s",
		MaxEjectionPercent: 30,
	},
}
