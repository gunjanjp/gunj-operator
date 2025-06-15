/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

// ServiceMeshSpec defines the service mesh configuration for the platform
type ServiceMeshSpec struct {
	// Enabled determines if service mesh integration should be enabled
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Type of service mesh to use (istio, linkerd)
	// +optional
	// +kubebuilder:validation:Enum=istio;linkerd;auto
	// +kubebuilder:default="auto"
	Type string `json:"type,omitempty"`

	// MTLS configuration for mutual TLS
	// +optional
	MTLS *MTLSConfig `json:"mtls,omitempty"`

	// TrafficManagement defines traffic management policies
	// +optional
	TrafficManagement *TrafficManagementConfig `json:"trafficManagement,omitempty"`

	// Observability defines service mesh observability features
	// +optional
	Observability *ServiceMeshObservabilityConfig `json:"observability,omitempty"`

	// SidecarInjection configuration
	// +optional
	SidecarInjection *SidecarInjectionConfig `json:"sidecarInjection,omitempty"`

	// CircuitBreaker configuration
	// +optional
	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty"`

	// RateLimiting configuration
	// +optional
	RateLimiting *RateLimitingConfig `json:"rateLimiting,omitempty"`
}

// MTLSConfig defines mutual TLS configuration
type MTLSConfig struct {
	// Enabled determines if mTLS should be enabled
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Mode defines the mTLS mode (STRICT, PERMISSIVE, DISABLE)
	// +optional
	// +kubebuilder:validation:Enum=STRICT;PERMISSIVE;DISABLE
	// +kubebuilder:default="STRICT"
	Mode string `json:"mode,omitempty"`

	// CertificateRotation defines certificate rotation policy
	// +optional
	CertificateRotation *CertificateRotationConfig `json:"certificateRotation,omitempty"`
}

// CertificateRotationConfig defines certificate rotation configuration
type CertificateRotationConfig struct {
	// Enabled determines if automatic certificate rotation should be enabled
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// RotationInterval defines how often certificates should be rotated
	// +optional
	// +kubebuilder:default="720h"
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	RotationInterval string `json:"rotationInterval,omitempty"`
}

// TrafficManagementConfig defines traffic management policies
type TrafficManagementConfig struct {
	// LoadBalancing defines load balancing policy
	// +optional
	LoadBalancing *LoadBalancingConfig `json:"loadBalancing,omitempty"`

	// Retry configuration for failed requests
	// +optional
	Retry *RetryConfig `json:"retry,omitempty"`

	// Timeout configuration for requests
	// +optional
	Timeout *TimeoutConfig `json:"timeout,omitempty"`

	// FaultInjection for testing resilience
	// +optional
	FaultInjection *FaultInjectionConfig `json:"faultInjection,omitempty"`

	// TrafficSplitting for canary deployments
	// +optional
	TrafficSplitting *TrafficSplittingConfig `json:"trafficSplitting,omitempty"`
}

// LoadBalancingConfig defines load balancing configuration
type LoadBalancingConfig struct {
	// Algorithm defines the load balancing algorithm
	// +optional
	// +kubebuilder:validation:Enum=ROUND_ROBIN;LEAST_REQUEST;RANDOM;PASSTHROUGH
	// +kubebuilder:default="ROUND_ROBIN"
	Algorithm string `json:"algorithm,omitempty"`

	// ConsistentHash configuration for session affinity
	// +optional
	ConsistentHash *ConsistentHashConfig `json:"consistentHash,omitempty"`
}

// ConsistentHashConfig defines consistent hash configuration
type ConsistentHashConfig struct {
	// HTTPHeaderName to use for consistent hashing
	// +optional
	HTTPHeaderName string `json:"httpHeaderName,omitempty"`

	// HTTPCookie to use for consistent hashing
	// +optional
	HTTPCookie *HTTPCookieConfig `json:"httpCookie,omitempty"`

	// UseSourceIP enables source IP based hashing
	// +optional
	UseSourceIP bool `json:"useSourceIP,omitempty"`
}

// HTTPCookieConfig defines HTTP cookie configuration
type HTTPCookieConfig struct {
	// Name of the cookie
	Name string `json:"name"`

	// TTL for the cookie
	// +optional
	// +kubebuilder:default="3600s"
	TTL string `json:"ttl,omitempty"`
}

// RetryConfig defines retry configuration
type RetryConfig struct {
	// Attempts is the number of retry attempts
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	Attempts int32 `json:"attempts,omitempty"`

	// PerTryTimeout for each retry attempt
	// +optional
	// +kubebuilder:default="10s"
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	PerTryTimeout string `json:"perTryTimeout,omitempty"`

	// RetryOn conditions that trigger a retry
	// +optional
	RetryOn []string `json:"retryOn,omitempty"`

	// RetryBackoff configuration
	// +optional
	RetryBackoff *RetryBackoffConfig `json:"retryBackoff,omitempty"`
}

// RetryBackoffConfig defines retry backoff configuration
type RetryBackoffConfig struct {
	// InitialInterval between retries
	// +optional
	// +kubebuilder:default="1s"
	InitialInterval string `json:"initialInterval,omitempty"`

	// MaxInterval between retries
	// +optional
	// +kubebuilder:default="30s"
	MaxInterval string `json:"maxInterval,omitempty"`

	// Multiplier for exponential backoff
	// +optional
	// +kubebuilder:default=2
	Multiplier float64 `json:"multiplier,omitempty"`
}

// TimeoutConfig defines timeout configuration
type TimeoutConfig struct {
	// RequestTimeout for HTTP requests
	// +optional
	// +kubebuilder:default="30s"
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	RequestTimeout string `json:"requestTimeout,omitempty"`

	// IdleTimeout for idle connections
	// +optional
	// +kubebuilder:default="300s"
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	IdleTimeout string `json:"idleTimeout,omitempty"`
}

// FaultInjectionConfig defines fault injection configuration
type FaultInjectionConfig struct {
	// Delay injection configuration
	// +optional
	Delay *DelayInjectionConfig `json:"delay,omitempty"`

	// Abort injection configuration
	// +optional
	Abort *AbortInjectionConfig `json:"abort,omitempty"`
}

// DelayInjectionConfig defines delay injection configuration
type DelayInjectionConfig struct {
	// Percentage of requests to inject delay
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Percentage int32 `json:"percentage"`

	// FixedDelay duration
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	FixedDelay string `json:"fixedDelay"`
}

// AbortInjectionConfig defines abort injection configuration
type AbortInjectionConfig struct {
	// Percentage of requests to abort
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Percentage int32 `json:"percentage"`

	// HTTPStatus code to return
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=599
	HTTPStatus int32 `json:"httpStatus"`
}

// TrafficSplittingConfig defines traffic splitting configuration
type TrafficSplittingConfig struct {
	// Enabled determines if traffic splitting should be enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Splits defines the traffic splits
	// +optional
	Splits []TrafficSplit `json:"splits,omitempty"`
}

// TrafficSplit defines a single traffic split
type TrafficSplit struct {
	// Version or subset to route to
	Version string `json:"version"`

	// Weight percentage (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Weight int32 `json:"weight"`
}

// ServiceMeshObservabilityConfig defines service mesh observability features
type ServiceMeshObservabilityConfig struct {
	// Metrics configuration
	// +optional
	Metrics *ServiceMeshMetricsConfig `json:"metrics,omitempty"`

	// Tracing configuration
	// +optional
	Tracing *ServiceMeshTracingConfig `json:"tracing,omitempty"`

	// AccessLogging configuration
	// +optional
	AccessLogging *AccessLoggingConfig `json:"accessLogging,omitempty"`
}

// ServiceMeshMetricsConfig defines service mesh metrics configuration
type ServiceMeshMetricsConfig struct {
	// Enabled determines if metrics should be collected
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Providers defines metrics providers
	// +optional
	Providers []string `json:"providers,omitempty"`

	// CustomMetrics defines custom metrics to collect
	// +optional
	CustomMetrics []CustomMetric `json:"customMetrics,omitempty"`
}

// CustomMetric defines a custom metric
type CustomMetric struct {
	// Name of the metric
	Name string `json:"name"`

	// Dimensions for the metric
	// +optional
	Dimensions map[string]string `json:"dimensions,omitempty"`

	// Unit of the metric
	// +optional
	Unit string `json:"unit,omitempty"`
}

// ServiceMeshTracingConfig defines service mesh tracing configuration
type ServiceMeshTracingConfig struct {
	// Enabled determines if tracing should be enabled
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// SamplingRate for traces (0.0-100.0)
	// +optional
	// +kubebuilder:default=1.0
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	SamplingRate float64 `json:"samplingRate,omitempty"`

	// Provider for tracing (jaeger, zipkin, datadog)
	// +optional
	// +kubebuilder:validation:Enum=jaeger;zipkin;datadog;tempo
	// +kubebuilder:default="tempo"
	Provider string `json:"provider,omitempty"`

	// CustomTags to add to all traces
	// +optional
	CustomTags map[string]string `json:"customTags,omitempty"`
}

// AccessLoggingConfig defines access logging configuration
type AccessLoggingConfig struct {
	// Enabled determines if access logging should be enabled
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Format of access logs
	// +optional
	// +kubebuilder:validation:Enum=JSON;TEXT
	// +kubebuilder:default="JSON"
	Format string `json:"format,omitempty"`

	// Filters to apply to access logs
	// +optional
	Filters []AccessLogFilter `json:"filters,omitempty"`
}

// AccessLogFilter defines an access log filter
type AccessLogFilter struct {
	// Type of filter (status_code, duration, header)
	// +kubebuilder:validation:Enum=status_code;duration;header;path
	Type string `json:"type"`

	// Condition for the filter
	Condition string `json:"condition"`

	// Value to match
	Value string `json:"value"`
}

// SidecarInjectionConfig defines sidecar injection configuration
type SidecarInjectionConfig struct {
	// AutoInject determines if sidecars should be automatically injected
	// +optional
	// +kubebuilder:default=true
	AutoInject bool `json:"autoInject,omitempty"`

	// ExcludeNamespaces from sidecar injection
	// +optional
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`

	// ExcludeLabels from sidecar injection
	// +optional
	ExcludeLabels map[string]string `json:"excludeLabels,omitempty"`

	// ResourceLimits for sidecars
	// +optional
	ResourceLimits *ResourceRequirements `json:"resourceLimits,omitempty"`
}

// CircuitBreakerConfig defines circuit breaker configuration
type CircuitBreakerConfig struct {
	// Enabled determines if circuit breaker should be enabled
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// ConsecutiveErrors before opening circuit
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=5
	ConsecutiveErrors int32 `json:"consecutiveErrors,omitempty"`

	// Interval for error counting
	// +optional
	// +kubebuilder:default="30s"
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	Interval string `json:"interval,omitempty"`

	// BaseEjectionTime when circuit is open
	// +optional
	// +kubebuilder:default="30s"
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	BaseEjectionTime string `json:"baseEjectionTime,omitempty"`

	// MaxEjectionPercent of upstream hosts
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=50
	MaxEjectionPercent int32 `json:"maxEjectionPercent,omitempty"`
}

// RateLimitingConfig defines rate limiting configuration
type RateLimitingConfig struct {
	// Enabled determines if rate limiting should be enabled
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Rules defines rate limiting rules
	// +optional
	Rules []RateLimitRule `json:"rules,omitempty"`

	// GlobalRateLimit configuration
	// +optional
	GlobalRateLimit *GlobalRateLimitConfig `json:"globalRateLimit,omitempty"`
}

// RateLimitRule defines a rate limit rule
type RateLimitRule struct {
	// Name of the rule
	Name string `json:"name"`

	// Match conditions for the rule
	// +optional
	Match *RateLimitMatch `json:"match,omitempty"`

	// Limit configuration
	Limit *RateLimit `json:"limit"`
}

// RateLimitMatch defines match conditions for rate limiting
type RateLimitMatch struct {
	// Headers to match
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// Path patterns to match
	// +optional
	Paths []string `json:"paths,omitempty"`

	// Methods to match
	// +optional
	Methods []string `json:"methods,omitempty"`
}

// RateLimit defines rate limit configuration
type RateLimit struct {
	// RequestsPerUnit allowed
	// +kubebuilder:validation:Minimum=1
	RequestsPerUnit int32 `json:"requestsPerUnit"`

	// Unit of time (SECOND, MINUTE, HOUR)
	// +kubebuilder:validation:Enum=SECOND;MINUTE;HOUR
	Unit string `json:"unit"`
}

// GlobalRateLimitConfig defines global rate limiting configuration
type GlobalRateLimitConfig struct {
	// Enabled determines if global rate limiting should be enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Service endpoint for rate limit service
	// +optional
	Service string `json:"service,omitempty"`

	// FailureMode when rate limit service is unavailable (ALLOW, DENY)
	// +optional
	// +kubebuilder:validation:Enum=ALLOW;DENY
	// +kubebuilder:default="ALLOW"
	FailureMode string `json:"failureMode,omitempty"`
}

// ServiceMeshHealthCheck defines service mesh health check configuration
type ServiceMeshHealthCheck struct {
	// Enabled determines if health checks should be enabled
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Interval between health checks
	// +optional
	// +kubebuilder:default="10s"
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	Interval string `json:"interval,omitempty"`

	// Timeout for health checks
	// +optional
	// +kubebuilder:default="3s"
	// +kubebuilder:validation:Pattern=`^\d+[hms]$`
	Timeout string `json:"timeout,omitempty"`

	// UnhealthyThreshold before marking unhealthy
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=2
	UnhealthyThreshold int32 `json:"unhealthyThreshold,omitempty"`

	// HealthyThreshold before marking healthy
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	HealthyThreshold int32 `json:"healthyThreshold,omitempty"`
}

// ServiceMeshStatus defines the status of service mesh integration
type ServiceMeshStatus struct {
	// Type of service mesh detected/configured
	Type string `json:"type,omitempty"`

	// Ready indicates if service mesh is ready
	Ready bool `json:"ready"`

	// Version of the service mesh
	Version string `json:"version,omitempty"`

	// SidecarInjected indicates if sidecars are injected
	SidecarInjected bool `json:"sidecarInjected"`

	// Metrics contains service mesh metrics
	// +optional
	Metrics *ServiceMeshMetrics `json:"metrics,omitempty"`

	// LastSyncTime when service mesh was last synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Message provides additional status information
	// +optional
	Message string `json:"message,omitempty"`
}

// ServiceMeshMetrics contains runtime metrics from the service mesh
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

	// LastUpdated when metrics were last updated
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}
