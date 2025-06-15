package istio

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/gunjanjp/gunj-operator/internal/servicemesh"
)

func TestVirtualServiceBuilder(t *testing.T) {
	t.Run("basic virtual service", func(t *testing.T) {
		spec := NewVirtualServiceBuilder("test-vs", "default").
			WithHosts("myapp").
			WithGateways("myapp-gateway").
			AddHTTPRoute(servicemesh.HTTPRouteSpec{
				Route: []servicemesh.HTTPRouteDestination{
					{
						Destination: &servicemesh.Destination{
							Host: "myapp",
						},
						Weight: 100,
					},
				},
			}).
			Build()
		
		assert.Len(t, spec.Hosts, 1)
		assert.Equal(t, "myapp", spec.Hosts[0])
		assert.Len(t, spec.Gateways, 1)
		assert.Equal(t, "myapp-gateway", spec.Gateways[0])
		assert.Len(t, spec.HTTP, 1)
	})
	
	t.Run("canary deployment", func(t *testing.T) {
		spec := NewVirtualServiceBuilder("canary-vs", "default").
			WithHosts("myapp").
			AddCanaryRoute("myapp-stable", "myapp-canary", 20).
			Build()
		
		assert.Len(t, spec.HTTP, 1)
		assert.Len(t, spec.HTTP[0].Route, 2)
		assert.Equal(t, int32(80), spec.HTTP[0].Route[0].Weight)
		assert.Equal(t, int32(20), spec.HTTP[0].Route[1].Weight)
	})
	
	t.Run("with retry policy", func(t *testing.T) {
		spec := NewVirtualServiceBuilder("retry-vs", "default").
			WithHosts("myapp").
			AddHTTPRoute(servicemesh.HTTPRouteSpec{
				Route: []servicemesh.HTTPRouteDestination{
					{
						Destination: &servicemesh.Destination{
							Host: "myapp",
						},
						Weight: 100,
					},
				},
			}).
			AddRetryPolicy(5, "3s").
			Build()
		
		require.NotNil(t, spec.HTTP[0].Retry)
		assert.Equal(t, int32(5), spec.HTTP[0].Retry.Attempts)
		assert.Equal(t, "3s", spec.HTTP[0].Retry.PerTryTimeout)
	})
	
	t.Run("with timeout", func(t *testing.T) {
		spec := NewVirtualServiceBuilder("timeout-vs", "default").
			WithHosts("myapp").
			AddHTTPRoute(servicemesh.HTTPRouteSpec{
				Route: []servicemesh.HTTPRouteDestination{
					{
						Destination: &servicemesh.Destination{
							Host: "myapp",
						},
						Weight: 100,
					},
				},
			}).
			AddTimeout("10s").
			Build()
		
		assert.Equal(t, "10s", spec.HTTP[0].Timeout)
	})
	
	t.Run("with fault injection", func(t *testing.T) {
		spec := NewVirtualServiceBuilder("fault-vs", "default").
			WithHosts("myapp").
			AddHTTPRoute(servicemesh.HTTPRouteSpec{
				Route: []servicemesh.HTTPRouteDestination{
					{
						Destination: &servicemesh.Destination{
							Host: "myapp",
						},
						Weight: 100,
					},
				},
			}).
			AddFaultInjection(10.0, "5s").
			Build()
		
		require.NotNil(t, spec.HTTP[0].Fault)
		require.NotNil(t, spec.HTTP[0].Fault.Delay)
		assert.Equal(t, 10.0, spec.HTTP[0].Fault.Delay.Percentage.Value)
		assert.Equal(t, "5s", spec.HTTP[0].Fault.Delay.FixedDelay)
	})
}

func TestDestinationRuleBuilder(t *testing.T) {
	t.Run("basic destination rule", func(t *testing.T) {
		spec := NewDestinationRuleBuilder("test-dr", "default", "myapp").
			WithLoadBalancer("ROUND_ROBIN").
			Build()
		
		assert.Equal(t, "myapp", spec.Host)
		require.NotNil(t, spec.TrafficPolicy)
		require.NotNil(t, spec.TrafficPolicy.LoadBalancer)
		assert.Equal(t, "ROUND_ROBIN", spec.TrafficPolicy.LoadBalancer.Simple)
	})
	
	t.Run("with consistent hash", func(t *testing.T) {
		spec := NewDestinationRuleBuilder("hash-dr", "default", "myapp").
			WithConsistentHash("x-session-id").
			Build()
		
		require.NotNil(t, spec.TrafficPolicy)
		require.NotNil(t, spec.TrafficPolicy.LoadBalancer)
		require.NotNil(t, spec.TrafficPolicy.LoadBalancer.ConsistentHash)
		assert.Equal(t, "x-session-id", spec.TrafficPolicy.LoadBalancer.ConsistentHash.HTTPHeaderName)
	})
	
	t.Run("with connection pool", func(t *testing.T) {
		spec := NewDestinationRuleBuilder("pool-dr", "default", "myapp").
			WithConnectionPool(100, "10s").
			Build()
		
		require.NotNil(t, spec.TrafficPolicy)
		require.NotNil(t, spec.TrafficPolicy.ConnectionPool)
		require.NotNil(t, spec.TrafficPolicy.ConnectionPool.TCP)
		assert.Equal(t, int32(100), spec.TrafficPolicy.ConnectionPool.TCP.MaxConnections)
		assert.Equal(t, "10s", spec.TrafficPolicy.ConnectionPool.TCP.ConnectTimeout)
	})
	
	t.Run("with HTTP connection pool", func(t *testing.T) {
		spec := NewDestinationRuleBuilder("http-pool-dr", "default", "myapp").
			WithHTTPConnectionPool(1024, 1024).
			Build()
		
		require.NotNil(t, spec.TrafficPolicy)
		require.NotNil(t, spec.TrafficPolicy.ConnectionPool)
		require.NotNil(t, spec.TrafficPolicy.ConnectionPool.HTTP)
		assert.Equal(t, int32(1024), spec.TrafficPolicy.ConnectionPool.HTTP.HTTP1MaxPendingRequests)
		assert.Equal(t, int32(1024), spec.TrafficPolicy.ConnectionPool.HTTP.HTTP2MaxRequests)
	})
	
	t.Run("with outlier detection", func(t *testing.T) {
		spec := NewDestinationRuleBuilder("outlier-dr", "default", "myapp").
			WithOutlierDetection(5, "30s", "30s").
			Build()
		
		require.NotNil(t, spec.TrafficPolicy)
		require.NotNil(t, spec.TrafficPolicy.OutlierDetection)
		assert.Equal(t, int32(5), spec.TrafficPolicy.OutlierDetection.ConsecutiveErrors)
		assert.Equal(t, "30s", spec.TrafficPolicy.OutlierDetection.Interval)
		assert.Equal(t, "30s", spec.TrafficPolicy.OutlierDetection.BaseEjectionTime)
		assert.Equal(t, int32(50), spec.TrafficPolicy.OutlierDetection.MaxEjectionPercent)
	})
	
	t.Run("with subsets", func(t *testing.T) {
		spec := NewDestinationRuleBuilder("subset-dr", "default", "myapp").
			AddSubset("stable", map[string]string{"version": "stable"}).
			AddSubset("canary", map[string]string{"version": "canary"}).
			Build()
		
		assert.Len(t, spec.Subsets, 2)
		assert.Equal(t, "stable", spec.Subsets[0].Name)
		assert.Equal(t, "stable", spec.Subsets[0].Labels["version"])
		assert.Equal(t, "canary", spec.Subsets[1].Name)
		assert.Equal(t, "canary", spec.Subsets[1].Labels["version"])
	})
	
	t.Run("with subset and policy", func(t *testing.T) {
		policy := &servicemesh.TrafficPolicy{
			LoadBalancer: &servicemesh.LoadBalancerSettings{
				Simple: "RANDOM",
			},
		}
		
		spec := NewDestinationRuleBuilder("subset-policy-dr", "default", "myapp").
			AddSubsetWithPolicy("special", map[string]string{"tier": "premium"}, policy).
			Build()
		
		assert.Len(t, spec.Subsets, 1)
		assert.Equal(t, "special", spec.Subsets[0].Name)
		require.NotNil(t, spec.Subsets[0].TrafficPolicy)
		assert.Equal(t, "RANDOM", spec.Subsets[0].TrafficPolicy.LoadBalancer.Simple)
	})
}

func TestPeerAuthenticationBuilder(t *testing.T) {
	t.Run("strict mTLS", func(t *testing.T) {
		spec := NewPeerAuthenticationBuilder("strict-pa", "default").
			EnableStrictMTLS().
			Build()
		
		require.NotNil(t, spec.MutualTLS)
		assert.Equal(t, "STRICT", spec.MutualTLS.Mode)
	})
	
	t.Run("permissive mTLS", func(t *testing.T) {
		spec := NewPeerAuthenticationBuilder("permissive-pa", "default").
			EnablePermissiveMTLS().
			Build()
		
		require.NotNil(t, spec.MutualTLS)
		assert.Equal(t, "PERMISSIVE", spec.MutualTLS.Mode)
	})
	
	t.Run("with workload selector", func(t *testing.T) {
		spec := NewPeerAuthenticationBuilder("workload-pa", "default").
			WithWorkloadSelector(map[string]string{
				"app":     "myapp",
				"version": "v1",
			}).
			WithMTLSMode("DISABLE").
			Build()
		
		require.NotNil(t, spec.Selector)
		assert.Len(t, spec.Selector.MatchLabels, 2)
		assert.Equal(t, "myapp", spec.Selector.MatchLabels["app"])
		assert.Equal(t, "v1", spec.Selector.MatchLabels["version"])
		assert.Equal(t, "DISABLE", spec.MutualTLS.Mode)
	})
}

func TestTelemetryBuilder(t *testing.T) {
	t.Run("with metrics", func(t *testing.T) {
		spec := NewTelemetryBuilder("metrics-telemetry", "default").
			AddMetrics([]string{"prometheus", "stackdriver"}, map[string]string{
				"request_protocol": "request.protocol",
				"response_code":    "response.code | 200",
			}).
			Build()
		
		assert.Len(t, spec.Metrics, 1)
		assert.Len(t, spec.Metrics[0].Providers, 2)
		assert.Equal(t, "prometheus", spec.Metrics[0].Providers[0].Name)
		assert.Equal(t, "stackdriver", spec.Metrics[0].Providers[1].Name)
		assert.Len(t, spec.Metrics[0].Dimensions, 2)
	})
	
	t.Run("with tracing", func(t *testing.T) {
		spec := NewTelemetryBuilder("tracing-telemetry", "default").
			AddTracing("jaeger", 1.0).
			Build()
		
		assert.Len(t, spec.Tracing, 1)
		assert.Len(t, spec.Tracing[0].Providers, 1)
		assert.Equal(t, "jaeger", spec.Tracing[0].Providers[0].Name)
		assert.Equal(t, 1.0, spec.Tracing[0].RandomSamplingPercentage)
	})
	
	t.Run("with access logging", func(t *testing.T) {
		spec := NewTelemetryBuilder("logging-telemetry", "default").
			AddAccessLogging([]string{"otel", "file"}).
			Build()
		
		assert.Len(t, spec.AccessLogging, 1)
		assert.Len(t, spec.AccessLogging[0].Providers, 2)
		assert.Equal(t, "otel", spec.AccessLogging[0].Providers[0].Name)
		assert.Equal(t, "file", spec.AccessLogging[0].Providers[1].Name)
	})
	
	t.Run("with default configurations", func(t *testing.T) {
		spec := NewTelemetryBuilder("default-telemetry", "default").
			EnableDefaultMetrics().
			EnableDefaultTracing(0.1).
			EnableDefaultAccessLogging().
			Build()
		
		// Check metrics
		assert.Len(t, spec.Metrics, 1)
		assert.Equal(t, "prometheus", spec.Metrics[0].Providers[0].Name)
		assert.Contains(t, spec.Metrics[0].Dimensions, "request_protocol")
		assert.Contains(t, spec.Metrics[0].Dimensions, "response_code")
		
		// Check tracing
		assert.Len(t, spec.Tracing, 1)
		assert.Equal(t, "jaeger", spec.Tracing[0].Providers[0].Name)
		assert.Equal(t, 0.1, spec.Tracing[0].RandomSamplingPercentage)
		
		// Check access logging
		assert.Len(t, spec.AccessLogging, 1)
		assert.Equal(t, "otel", spec.AccessLogging[0].Providers[0].Name)
	})
}

func TestCommonConfigurations(t *testing.T) {
	t.Run("load balancer types", func(t *testing.T) {
		assert.Equal(t, "ROUND_ROBIN", LoadBalancerTypes.RoundRobin)
		assert.Equal(t, "LEAST_REQUEST", LoadBalancerTypes.LeastRequest)
		assert.Equal(t, "RANDOM", LoadBalancerTypes.Random)
		assert.Equal(t, "PASSTHROUGH", LoadBalancerTypes.Passthrough)
	})
	
	t.Run("mTLS modes", func(t *testing.T) {
		assert.Equal(t, "DISABLE", MTLSModes.Disable)
		assert.Equal(t, "PERMISSIVE", MTLSModes.Permissive)
		assert.Equal(t, "STRICT", MTLSModes.Strict)
		assert.Equal(t, "UNSET", MTLSModes.Unset)
	})
	
	t.Run("common retry policies", func(t *testing.T) {
		// Test default policy
		assert.Equal(t, int32(3), CommonRetryPolicies.Default.Attempts)
		assert.Equal(t, "2s", CommonRetryPolicies.Default.PerTryTimeout)
		assert.Contains(t, CommonRetryPolicies.Default.RetryOn, "5xx")
		
		// Test aggressive policy
		assert.Equal(t, int32(5), CommonRetryPolicies.Aggressive.Attempts)
		assert.Equal(t, "5s", CommonRetryPolicies.Aggressive.PerTryTimeout)
		assert.Contains(t, CommonRetryPolicies.Aggressive.RetryOn, "retriable-4xx")
		
		// Test conservative policy
		assert.Equal(t, int32(2), CommonRetryPolicies.Conservative.Attempts)
		assert.Equal(t, "1s", CommonRetryPolicies.Conservative.PerTryTimeout)
		assert.Equal(t, "5xx", CommonRetryPolicies.Conservative.RetryOn)
	})
	
	t.Run("common circuit breakers", func(t *testing.T) {
		// Test default circuit breaker
		assert.Equal(t, int32(5), CommonCircuitBreakers.Default.ConsecutiveErrors)
		assert.Equal(t, "30s", CommonCircuitBreakers.Default.Interval)
		assert.Equal(t, "30s", CommonCircuitBreakers.Default.BaseEjectionTime)
		assert.Equal(t, int32(50), CommonCircuitBreakers.Default.MaxEjectionPercent)
		
		// Test aggressive circuit breaker
		assert.Equal(t, int32(3), CommonCircuitBreakers.Aggressive.ConsecutiveErrors)
		assert.Equal(t, "10s", CommonCircuitBreakers.Aggressive.Interval)
		assert.Equal(t, "60s", CommonCircuitBreakers.Aggressive.BaseEjectionTime)
		assert.Equal(t, int32(70), CommonCircuitBreakers.Aggressive.MaxEjectionPercent)
		
		// Test conservative circuit breaker
		assert.Equal(t, int32(10), CommonCircuitBreakers.Conservative.ConsecutiveErrors)
		assert.Equal(t, "60s", CommonCircuitBreakers.Conservative.Interval)
		assert.Equal(t, "15s", CommonCircuitBreakers.Conservative.BaseEjectionTime)
		assert.Equal(t, int32(30), CommonCircuitBreakers.Conservative.MaxEjectionPercent)
	})
}

func TestCreateDefaultLabels(t *testing.T) {
	labels := CreateDefaultLabels("prometheus")
	
	assert.Equal(t, "gunj-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "prometheus", labels["app.kubernetes.io/component"])
	assert.Equal(t, "istio", labels["service-mesh.io/type"])
}
