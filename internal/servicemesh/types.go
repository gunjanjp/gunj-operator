package servicemesh

import (
	"context"
	
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

// ServiceMeshConfig contains configuration for service mesh integration
type ServiceMeshConfig struct {
	// Type specifies the service mesh type
	Type ServiceMeshType `json:"type,omitempty"`
	
	// Enabled indicates if service mesh integration is enabled
	Enabled bool `json:"enabled,omitempty"`
	
	// Namespace is the service mesh control plane namespace
	Namespace string `json:"namespace,omitempty"`
	
	// MTLS configuration
	MTLS MTLSConfig `json:"mtls,omitempty"`
	
	// TrafficManagement configuration
	TrafficManagement TrafficManagementConfig `json:"trafficManagement,omitempty"`
	
	// Telemetry configuration
	Telemetry TelemetryConfig `json:"telemetry,omitempty"`
	
	// IstioConfig contains Istio-specific configuration
	IstioConfig *IstioConfig `json:"istioConfig,omitempty"`
	
	// LinkerdConfig contains Linkerd-specific configuration
	LinkerdConfig *LinkerdConfig `json:"linkerdConfig,omitempty"`
}

// MTLSConfig contains mTLS configuration
type MTLSConfig struct {
	// Mode specifies the mTLS mode
	Mode string `json:"mode,omitempty"`
	
	// Strict enables strict mTLS
	Strict bool `json:"strict,omitempty"`
}

// TrafficManagementConfig contains traffic management configuration
type TrafficManagementConfig struct {
	// RetryPolicy configuration
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
	
	// CircuitBreaker configuration
	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty"`
	
	// LoadBalancer configuration
	LoadBalancer *LoadBalancerConfig `json:"loadBalancer,omitempty"`
	
	// Timeout configuration
	Timeout string `json:"timeout,omitempty"`
}

// RetryPolicy defines retry configuration
type RetryPolicy struct {
	// Attempts is the number of retry attempts
	Attempts int32 `json:"attempts,omitempty"`
	
	// PerTryTimeout is the timeout per retry attempt
	PerTryTimeout string `json:"perTryTimeout,omitempty"`
	
	// RetryOn specifies the conditions under which retry takes place
	RetryOn string `json:"retryOn,omitempty"`
}

// CircuitBreakerConfig defines circuit breaker configuration
type CircuitBreakerConfig struct {
	// ConsecutiveErrors before opening the circuit
	ConsecutiveErrors int32 `json:"consecutiveErrors,omitempty"`
	
	// Interval for error counting
	Interval string `json:"interval,omitempty"`
	
	// BaseEjectionTime is the minimum ejection duration
	BaseEjectionTime string `json:"baseEjectionTime,omitempty"`
	
	// MaxEjectionPercent is the maximum percentage of upstream hosts ejected
	MaxEjectionPercent int32 `json:"maxEjectionPercent,omitempty"`
}

// LoadBalancerConfig defines load balancer configuration
type LoadBalancerConfig struct {
	// Type specifies the load balancer algorithm
	Type string `json:"type,omitempty"`
	
	// ConsistentHash configuration
	ConsistentHash *ConsistentHashConfig `json:"consistentHash,omitempty"`
}

// ConsistentHashConfig defines consistent hash load balancing
type ConsistentHashConfig struct {
	// HTTPHeaderName for hash-based load balancing
	HTTPHeaderName string `json:"httpHeaderName,omitempty"`
	
	// HTTPCookie for hash-based load balancing
	HTTPCookie *HTTPCookie `json:"httpCookie,omitempty"`
	
	// UseSourceIP enables source IP hashing
	UseSourceIP bool `json:"useSourceIP,omitempty"`
}

// HTTPCookie defines HTTP cookie configuration
type HTTPCookie struct {
	// Name of the cookie
	Name string `json:"name,omitempty"`
	
	// Path of the cookie
	Path string `json:"path,omitempty"`
	
	// TTL of the cookie
	TTL string `json:"ttl,omitempty"`
}

// TelemetryConfig contains telemetry configuration
type TelemetryConfig struct {
	// Metrics configuration
	Metrics MetricsConfig `json:"metrics,omitempty"`
	
	// Tracing configuration
	Tracing TracingConfig `json:"tracing,omitempty"`
	
	// AccessLogging configuration
	AccessLogging AccessLoggingConfig `json:"accessLogging,omitempty"`
}

// MetricsConfig defines metrics configuration
type MetricsConfig struct {
	// Enabled indicates if metrics are enabled
	Enabled bool `json:"enabled,omitempty"`
	
	// Providers list
	Providers []string `json:"providers,omitempty"`
	
	// Dimensions to include
	Dimensions map[string]string `json:"dimensions,omitempty"`
}

// TracingConfig defines tracing configuration
type TracingConfig struct {
	// Enabled indicates if tracing is enabled
	Enabled bool `json:"enabled,omitempty"`
	
	// SamplingRate for traces
	SamplingRate float64 `json:"samplingRate,omitempty"`
	
	// Provider name
	Provider string `json:"provider,omitempty"`
}

// AccessLoggingConfig defines access logging configuration
type AccessLoggingConfig struct {
	// Enabled indicates if access logging is enabled
	Enabled bool `json:"enabled,omitempty"`
	
	// Providers list
	Providers []string `json:"providers,omitempty"`
}

// IstioConfig contains Istio-specific configuration
type IstioConfig struct {
	// Sidecar configuration
	Sidecar *SidecarConfig `json:"sidecar,omitempty"`
	
	// Gateway configuration
	Gateway *GatewayConfig `json:"gateway,omitempty"`
	
	// Pilot configuration
	Pilot *PilotConfig `json:"pilot,omitempty"`
}

// SidecarConfig defines Istio sidecar configuration
type SidecarConfig struct {
	// InjectAnnotation controls sidecar injection
	InjectAnnotation string `json:"injectAnnotation,omitempty"`
	
	// Resources for the sidecar
	Resources *ResourceRequirements `json:"resources,omitempty"`
	
	// ProxyConfig overrides
	ProxyConfig map[string]interface{} `json:"proxyConfig,omitempty"`
}

// GatewayConfig defines Istio gateway configuration
type GatewayConfig struct {
	// Name of the gateway
	Name string `json:"name,omitempty"`
	
	// Selector for the gateway
	Selector map[string]string `json:"selector,omitempty"`
}

// PilotConfig defines Istio pilot configuration
type PilotConfig struct {
	// Trace sampling
	TraceSampling float64 `json:"traceSampling,omitempty"`
}

// LinkerdConfig contains Linkerd-specific configuration
type LinkerdConfig struct {
	// ProfileName for the Linkerd profile
	ProfileName string `json:"profileName,omitempty"`
	
	// ServiceProfile configuration
	ServiceProfile *ServiceProfileConfig `json:"serviceProfile,omitempty"`
}

// ServiceProfileConfig defines Linkerd service profile configuration
type ServiceProfileConfig struct {
	// Routes definition
	Routes []RouteConfig `json:"routes,omitempty"`
	
	// RetryBudget configuration
	RetryBudget *RetryBudgetConfig `json:"retryBudget,omitempty"`
}

// RouteConfig defines route configuration
type RouteConfig struct {
	// Name of the route
	Name string `json:"name,omitempty"`
	
	// Condition for the route
	Condition string `json:"condition,omitempty"`
	
	// Timeout for the route
	Timeout string `json:"timeout,omitempty"`
}

// RetryBudgetConfig defines retry budget configuration
type RetryBudgetConfig struct {
	// RetryRatio
	RetryRatio float64 `json:"retryRatio,omitempty"`
	
	// MinRetriesPerSecond
	MinRetriesPerSecond int32 `json:"minRetriesPerSecond,omitempty"`
	
	// TTL for the retry budget
	TTL string `json:"ttl,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	// CPU requirements
	CPU string `json:"cpu,omitempty"`
	
	// Memory requirements
	Memory string `json:"memory,omitempty"`
}

// Manager is the interface for service mesh managers
type Manager interface {
	// Configure applies service mesh configuration to the platform
	Configure(ctx context.Context, namespace string, config *ServiceMeshConfig) error
	
	// CreateVirtualService creates a virtual service for traffic management
	CreateVirtualService(ctx context.Context, namespace, name string, spec *VirtualServiceSpec) error
	
	// CreateDestinationRule creates a destination rule for traffic management
	CreateDestinationRule(ctx context.Context, namespace, name string, spec *DestinationRuleSpec) error
	
	// CreatePeerAuthentication creates peer authentication for mTLS
	CreatePeerAuthentication(ctx context.Context, namespace, name string, spec *PeerAuthenticationSpec) error
	
	// CreateTelemetry creates telemetry configuration
	CreateTelemetry(ctx context.Context, namespace, name string, spec *TelemetrySpec) error
	
	// Cleanup removes service mesh resources
	Cleanup(ctx context.Context, namespace string) error
	
	// GetStatus returns the service mesh status
	GetStatus(ctx context.Context, namespace string) (*ServiceMeshStatus, error)
}

// VirtualServiceSpec defines virtual service specification
type VirtualServiceSpec struct {
	// Hosts for the virtual service
	Hosts []string `json:"hosts,omitempty"`
	
	// Gateways for the virtual service
	Gateways []string `json:"gateways,omitempty"`
	
	// HTTP routes
	HTTP []HTTPRouteSpec `json:"http,omitempty"`
}

// HTTPRouteSpec defines HTTP route specification
type HTTPRouteSpec struct {
	// Match conditions
	Match []HTTPMatchRequest `json:"match,omitempty"`
	
	// Route destinations
	Route []HTTPRouteDestination `json:"route,omitempty"`
	
	// Timeout
	Timeout string `json:"timeout,omitempty"`
	
	// Retry policy
	Retry *RetryPolicy `json:"retry,omitempty"`
	
	// Fault injection
	Fault *HTTPFaultInjection `json:"fault,omitempty"`
}

// HTTPMatchRequest defines HTTP match conditions
type HTTPMatchRequest struct {
	// URI match
	URI *StringMatch `json:"uri,omitempty"`
	
	// Headers match
	Headers map[string]*StringMatch `json:"headers,omitempty"`
	
	// Method match
	Method *StringMatch `json:"method,omitempty"`
}

// StringMatch defines string matching
type StringMatch struct {
	// Exact match
	Exact string `json:"exact,omitempty"`
	
	// Prefix match
	Prefix string `json:"prefix,omitempty"`
	
	// Regex match
	Regex string `json:"regex,omitempty"`
}

// HTTPRouteDestination defines route destination
type HTTPRouteDestination struct {
	// Destination
	Destination *Destination `json:"destination,omitempty"`
	
	// Weight
	Weight int32 `json:"weight,omitempty"`
}

// Destination defines destination
type Destination struct {
	// Host
	Host string `json:"host,omitempty"`
	
	// Subset
	Subset string `json:"subset,omitempty"`
	
	// Port
	Port *PortSelector `json:"port,omitempty"`
}

// PortSelector defines port selector
type PortSelector struct {
	// Number
	Number uint32 `json:"number,omitempty"`
}

// HTTPFaultInjection defines fault injection
type HTTPFaultInjection struct {
	// Delay injection
	Delay *HTTPFaultDelay `json:"delay,omitempty"`
	
	// Abort injection
	Abort *HTTPFaultAbort `json:"abort,omitempty"`
}

// HTTPFaultDelay defines delay injection
type HTTPFaultDelay struct {
	// Percentage
	Percentage *Percentage `json:"percentage,omitempty"`
	
	// FixedDelay
	FixedDelay string `json:"fixedDelay,omitempty"`
}

// HTTPFaultAbort defines abort injection
type HTTPFaultAbort struct {
	// Percentage
	Percentage *Percentage `json:"percentage,omitempty"`
	
	// HTTPStatus
	HTTPStatus int32 `json:"httpStatus,omitempty"`
}

// Percentage defines percentage value
type Percentage struct {
	// Value
	Value float64 `json:"value,omitempty"`
}

// DestinationRuleSpec defines destination rule specification
type DestinationRuleSpec struct {
	// Host
	Host string `json:"host,omitempty"`
	
	// TrafficPolicy
	TrafficPolicy *TrafficPolicy `json:"trafficPolicy,omitempty"`
	
	// Subsets
	Subsets []Subset `json:"subsets,omitempty"`
}

// TrafficPolicy defines traffic policy
type TrafficPolicy struct {
	// LoadBalancer
	LoadBalancer *LoadBalancerSettings `json:"loadBalancer,omitempty"`
	
	// ConnectionPool
	ConnectionPool *ConnectionPoolSettings `json:"connectionPool,omitempty"`
	
	// OutlierDetection
	OutlierDetection *OutlierDetection `json:"outlierDetection,omitempty"`
}

// LoadBalancerSettings defines load balancer settings
type LoadBalancerSettings struct {
	// Simple load balancer
	Simple string `json:"simple,omitempty"`
	
	// ConsistentHash
	ConsistentHash *ConsistentHashLB `json:"consistentHash,omitempty"`
}

// ConsistentHashLB defines consistent hash load balancing
type ConsistentHashLB struct {
	// HTTPHeaderName
	HTTPHeaderName string `json:"httpHeaderName,omitempty"`
	
	// HTTPCookie
	HTTPCookie *HTTPCookieLB `json:"httpCookie,omitempty"`
	
	// UseSourceIP
	UseSourceIP bool `json:"useSourceIp,omitempty"`
}

// HTTPCookieLB defines HTTP cookie for load balancing
type HTTPCookieLB struct {
	// Name
	Name string `json:"name,omitempty"`
	
	// Path
	Path string `json:"path,omitempty"`
	
	// TTL
	TTL string `json:"ttl,omitempty"`
}

// ConnectionPoolSettings defines connection pool settings
type ConnectionPoolSettings struct {
	// TCP settings
	TCP *TCPSettings `json:"tcp,omitempty"`
	
	// HTTP settings
	HTTP *HTTPSettings `json:"http,omitempty"`
}

// TCPSettings defines TCP settings
type TCPSettings struct {
	// MaxConnections
	MaxConnections int32 `json:"maxConnections,omitempty"`
	
	// ConnectTimeout
	ConnectTimeout string `json:"connectTimeout,omitempty"`
}

// HTTPSettings defines HTTP settings
type HTTPSettings struct {
	// HTTP1MaxPendingRequests
	HTTP1MaxPendingRequests int32 `json:"http1MaxPendingRequests,omitempty"`
	
	// HTTP2MaxRequests
	HTTP2MaxRequests int32 `json:"http2MaxRequests,omitempty"`
	
	// MaxRequestsPerConnection
	MaxRequestsPerConnection int32 `json:"maxRequestsPerConnection,omitempty"`
}

// OutlierDetection defines outlier detection
type OutlierDetection struct {
	// ConsecutiveErrors
	ConsecutiveErrors int32 `json:"consecutiveErrors,omitempty"`
	
	// Interval
	Interval string `json:"interval,omitempty"`
	
	// BaseEjectionTime
	BaseEjectionTime string `json:"baseEjectionTime,omitempty"`
	
	// MaxEjectionPercent
	MaxEjectionPercent int32 `json:"maxEjectionPercent,omitempty"`
}

// Subset defines subset
type Subset struct {
	// Name
	Name string `json:"name,omitempty"`
	
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
	
	// TrafficPolicy
	TrafficPolicy *TrafficPolicy `json:"trafficPolicy,omitempty"`
}

// PeerAuthenticationSpec defines peer authentication specification
type PeerAuthenticationSpec struct {
	// Selector
	Selector *WorkloadSelector `json:"selector,omitempty"`
	
	// MutualTLS
	MutualTLS *MutualTLS `json:"mtls,omitempty"`
}

// WorkloadSelector defines workload selector
type WorkloadSelector struct {
	// MatchLabels
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// MutualTLS defines mutual TLS settings
type MutualTLS struct {
	// Mode
	Mode string `json:"mode,omitempty"`
}

// TelemetrySpec defines telemetry specification
type TelemetrySpec struct {
	// Metrics
	Metrics []MetricSpec `json:"metrics,omitempty"`
	
	// Tracing
	Tracing []TracingSpec `json:"tracing,omitempty"`
	
	// AccessLogging
	AccessLogging []AccessLoggingSpec `json:"accessLogging,omitempty"`
}

// MetricSpec defines metric specification
type MetricSpec struct {
	// Providers
	Providers []ProviderRef `json:"providers,omitempty"`
	
	// Dimensions
	Dimensions map[string]string `json:"dimensions,omitempty"`
}

// TracingSpec defines tracing specification
type TracingSpec struct {
	// Providers
	Providers []ProviderRef `json:"providers,omitempty"`
	
	// RandomSamplingPercentage
	RandomSamplingPercentage float64 `json:"randomSamplingPercentage,omitempty"`
}

// AccessLoggingSpec defines access logging specification
type AccessLoggingSpec struct {
	// Providers
	Providers []ProviderRef `json:"providers,omitempty"`
}

// ProviderRef defines provider reference
type ProviderRef struct {
	// Name
	Name string `json:"name,omitempty"`
}

// ServiceMeshStatus defines service mesh status
type ServiceMeshStatus struct {
	// Ready indicates if service mesh is ready
	Ready bool `json:"ready,omitempty"`
	
	// Message provides status message
	Message string `json:"message,omitempty"`
	
	// Components status
	Components map[string]ComponentStatus `json:"components,omitempty"`
}

// ComponentStatus defines component status
type ComponentStatus struct {
	// Ready
	Ready bool `json:"ready,omitempty"`
	
	// Version
	Version string `json:"version,omitempty"`
	
	// Message
	Message string `json:"message,omitempty"`
}

// IstioResources contains Istio-specific resources
type IstioResources struct {
	// VirtualServices
	VirtualServices []*v1beta1.VirtualService
	
	// DestinationRules
	DestinationRules []*v1beta1.DestinationRule
	
	// PeerAuthentications
	PeerAuthentications []runtime.Object
	
	// Telemetries
	Telemetries []runtime.Object
}

// ManagerFactory creates service mesh managers
type ManagerFactory interface {
	// CreateManager creates a service mesh manager based on type
	CreateManager(meshType ServiceMeshType, client client.Client) (Manager, error)
}
