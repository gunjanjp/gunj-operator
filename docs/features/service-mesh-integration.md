# Service Mesh Integration

The Gunj Operator provides seamless integration with popular service meshes like Istio and Linkerd, enabling advanced traffic management, security, and observability features for your observability platform components.

## Overview

Service mesh integration enhances your observability platform with:

- **Automatic mTLS**: Zero-trust security between all components
- **Traffic Management**: Retries, timeouts, circuit breakers, and load balancing
- **Enhanced Observability**: Distributed tracing, service-level metrics, and access logs
- **Multi-cluster Support**: Federated observability across multiple Kubernetes clusters
- **Canary Deployments**: Safe rollouts with traffic splitting
- **Security Policies**: Fine-grained access control and network policies

## Supported Service Meshes

### Istio
- Version: 1.17+ recommended
- Features: Full support for all traffic management features
- Best for: Complex environments with advanced routing needs

### Linkerd
- Version: stable-2.14+ recommended
- Features: Lightweight with automatic mTLS and excellent observability
- Best for: Simplicity and performance-focused deployments

## Configuration

### Basic Configuration

Enable service mesh integration in your ObservabilityPlatform spec:

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
  namespace: monitoring
spec:
  serviceMesh:
    enabled: true
    type: auto  # auto-detect, or specify: istio, linkerd
    mtls:
      enabled: true
      mode: STRICT  # STRICT, PERMISSIVE, or DISABLE
```

### Advanced Configuration

```yaml
spec:
  serviceMesh:
    enabled: true
    type: istio
    
    # mTLS Configuration
    mtls:
      enabled: true
      mode: STRICT
    
    # Traffic Management
    trafficManagement:
      retry:
        attempts: 3
        perTryTimeout: "10s"
        retryOn: "5xx,reset,connect-failure"
      
      circuitBreaker:
        consecutiveErrors: 5
        interval: "30s"
        baseEjectionTime: "30s"
        maxEjectionPercent: 50
      
      timeout:
        request: "30s"
        idle: "900s"
      
      loadBalancing:
        algorithm: ROUND_ROBIN
        consistentHash:
          httpHeaderName: "x-session-id"
    
    # Observability
    observability:
      metrics:
        enabled: true
      tracing:
        enabled: true
        samplingRate: 0.01
      accessLogs:
        enabled: true
        format: JSON
```

## How It Works

### 1. Auto-Detection

When `type: auto` is configured, the operator:
- Checks for Istio by looking for `istio-system` namespace
- Checks for Linkerd by looking for `linkerd` namespace
- Configures the appropriate provider automatically

### 2. Sidecar Injection

The operator automatically:
- Enables sidecar injection for the platform namespace
- Adds appropriate labels/annotations for the service mesh
- Ensures all pods receive sidecar proxies

### 3. Security Configuration

#### mTLS Setup
- Creates PeerAuthentication (Istio) or Server policies (Linkerd)
- Configures authorization policies for service-to-service communication
- Ensures encrypted communication between all components

#### Network Policies
- Creates deny-all default policies
- Allows specific communication patterns between components
- Integrates with service mesh policies

### 4. Traffic Management

#### Retry Policies
- Configures automatic retries for transient failures
- Prevents retry storms with budgets (Linkerd) or limits (Istio)
- Customizable retry conditions

#### Circuit Breakers
- Prevents cascading failures
- Automatically ejects unhealthy endpoints
- Configurable thresholds and recovery times

#### Load Balancing
- Multiple algorithms: Round Robin, Least Request, Random
- Consistent hashing for session affinity
- Locality-aware routing for multi-zone deployments

## Multi-Cluster Configuration

Enable cross-cluster observability:

```yaml
spec:
  serviceMesh:
    multiCluster:
      enabled: true
      clusterName: "primary"
      network: "network1"
      remoteClusters:
        - name: "secondary"
          endpoint: "https://secondary-api.example.com:6443"
          network: "network2"
```

This enables:
- Cross-cluster service discovery
- Federated metrics collection
- Distributed trace correlation
- Multi-cluster dashboards

## Metrics and Monitoring

### Service Mesh Metrics

The integration automatically exposes:

```promql
# Request rate by service
sum(rate(istio_request_total[5m])) by (destination_service_name)

# P99 latency
histogram_quantile(0.99, 
  sum(rate(istio_request_duration_milliseconds_bucket[5m])) 
  by (destination_service_name, le)
)

# Error rate
sum(rate(istio_request_total{response_code=~"5.."}[5m])) 
by (destination_service_name)

# Circuit breaker status
sum(envoy_cluster_circuit_breakers_default_rq_open) 
by (cluster_name)
```

### Custom Dashboards

The operator creates service mesh dashboards for:
- Service overview with golden signals
- Traffic flow visualization
- Circuit breaker status
- mTLS certificate information
- Cross-cluster communication

## Troubleshooting

### Common Issues

#### Sidecars Not Injected
```bash
# Check namespace labels
kubectl get namespace monitoring -o yaml | grep -E "(istio-injection|linkerd.io/inject)"

# Restart pods to trigger injection
kubectl rollout restart deployment -n monitoring
```

#### mTLS Errors
```bash
# Check Istio policies
kubectl get peerauthentication,authorizationpolicy -n monitoring

# Check Linkerd policies  
kubectl get server,serverauthorization -n monitoring

# Verify certificates
istioctl proxy-config secret deployment/prometheus -n monitoring
```

#### Traffic Not Flowing
```bash
# Check virtual services (Istio)
kubectl get virtualservice,destinationrule -n monitoring

# Check service profiles (Linkerd)
kubectl get serviceprofile -n monitoring

# Test connectivity
kubectl exec -n monitoring deployment/prometheus -- \
  curl -v http://grafana:3000/api/health
```

### Debug Commands

#### Istio
```bash
# Analyze configuration
istioctl analyze -n monitoring

# Check proxy configuration
istioctl proxy-config all deployment/prometheus -n monitoring

# View proxy logs
kubectl logs -n monitoring deployment/prometheus -c istio-proxy
```

#### Linkerd
```bash
# Check data plane
linkerd check --proxy -n monitoring

# View traffic
linkerd viz tap -n monitoring deployment/prometheus

# Check routes
linkerd viz routes -n monitoring deployment/prometheus
```

## Best Practices

### 1. Resource Planning
- Sidecars add ~100Mi memory and ~100m CPU per pod
- Plan for additional resource overhead
- Use resource limits on sidecars

### 2. Security
- Always use STRICT mTLS in production
- Implement proper RBAC policies
- Regular certificate rotation

### 3. Observability
- Set appropriate sampling rates (0.01-0.1 for production)
- Use access logs sparingly (performance impact)
- Monitor sidecar resource usage

### 4. Traffic Management
- Start with conservative retry policies
- Test circuit breakers in staging
- Monitor for retry storms

### 5. Multi-cluster
- Use unique cluster names
- Ensure network connectivity between clusters
- Monitor cross-cluster latency

## Examples

See the [examples/servicemesh](../../examples/servicemesh) directory for:
- Basic service mesh integration
- Advanced Istio configuration
- Linkerd-specific setup
- Multi-cluster federation

## API Reference

### ServiceMeshSpec

| Field | Type | Description |
|-------|------|-------------|
| enabled | bool | Enable service mesh integration |
| type | string | Service mesh type: auto, istio, linkerd |
| mtls | MTLSConfig | mTLS configuration |
| trafficManagement | TrafficManagementConfig | Traffic policies |
| observability | ServiceMeshObservabilityConfig | Observability settings |
| multiCluster | MultiClusterConfig | Multi-cluster configuration |

### ServiceMeshStatus

| Field | Type | Description |
|-------|------|-------------|
| type | string | Detected service mesh type |
| enabled | bool | Whether integration is active |
| ready | bool | Integration health status |
| version | string | Service mesh version |
| metrics | ServiceMeshMetrics | Current metrics |

## Integration with Components

### Prometheus
- Automatically scrapes sidecar metrics
- Federates metrics in multi-cluster setups
- No additional configuration needed

### Grafana
- Pre-configured service mesh dashboards
- Multi-cluster data sources
- Traffic flow visualization

### Loki
- Collects sidecar logs if enabled
- Parses structured access logs
- Correlates with traces

### Tempo
- Receives traces from sidecars
- Correlates cross-service calls
- Multi-cluster trace aggregation

## Performance Considerations

### Resource Overhead
- Sidecars: ~100Mi RAM, ~100m CPU per pod
- Control plane: Varies by mesh and cluster size
- Network: Minimal latency addition (<1ms typically)

### Optimization Tips
1. Tune sidecar resource limits
2. Disable unused features (e.g., access logs)
3. Adjust telemetry sampling rates
4. Use locality-aware routing

## Future Enhancements

- Support for additional service meshes (Consul, Kuma)
- Automated canary deployments
- Service mesh migration tools
- Advanced security policies
- AI-driven traffic management
