# Service Mesh Integration Examples

This directory contains example configurations for integrating the Gunj Operator with service meshes like Istio and Linkerd.

## Prerequisites

Before using these examples, ensure you have a service mesh installed in your cluster:

### For Istio:
```bash
# Install Istio
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH
istioctl install --set values.pilot.env.PILOT_ENABLE_WORKLOAD_ENTRY_AUTOREGISTRATION=true
```

### For Linkerd:
```bash
# Install Linkerd CLI
curl --proto '=https' --tlsv1.2 -sSfL https://run.linkerd.io/install | sh
export PATH=$PATH:$HOME/.linkerd2/bin

# Install Linkerd control plane
linkerd install | kubectl apply -f -
linkerd check
```

## Examples

1. **Basic Service Mesh Integration** - `basic-servicemesh.yaml`
   - Enables automatic service mesh integration
   - Configures mTLS in STRICT mode
   - Sets up basic traffic management policies

2. **Advanced Istio Configuration** - `istio-advanced.yaml`
   - Custom retry and timeout policies
   - Circuit breaker configuration
   - Advanced load balancing with consistent hashing
   - Full observability integration

3. **Linkerd Configuration** - `linkerd-config.yaml`
   - Linkerd-specific settings
   - Service profiles
   - Retry budgets
   - Traffic splits for canary deployments

4. **Multi-Cluster Mesh** - `multi-cluster.yaml`
   - Multi-cluster service mesh configuration
   - Cross-cluster service discovery
   - Multi-cluster observability

## Usage

Apply any of the example configurations:

```bash
kubectl apply -f basic-servicemesh.yaml
```

Monitor the service mesh integration:

```bash
# Check if sidecar injection is enabled
kubectl get namespace monitoring -o yaml | grep injection

# Verify sidecars are injected
kubectl get pods -n monitoring -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].name}{"\n"}{end}'

# Check service mesh policies (Istio)
kubectl get virtualservices,destinationrules,peerauthentications -n monitoring

# Check service mesh policies (Linkerd)
kubectl get serviceprofiles -n monitoring
```

## Metrics and Observability

The service mesh integration automatically exposes additional metrics:

- Request rate and error rate
- Latency percentiles (P50, P90, P95, P99)
- Active connections
- Circuit breaker status
- Retry attempts

Access these metrics through Grafana dashboards or query Prometheus directly:

```promql
# Request rate
sum(rate(istio_request_total[5m])) by (destination_service_name)

# Error rate
sum(rate(istio_request_total{response_code=~"5.."}[5m])) by (destination_service_name)

# P99 latency
histogram_quantile(0.99, sum(rate(istio_request_duration_milliseconds_bucket[5m])) by (destination_service_name, le))
```

## Troubleshooting

### Sidecar not injected
- Ensure the namespace has the correct label/annotation
- Check if the service mesh webhook is running
- Restart pods after enabling injection

### mTLS issues
- Verify PeerAuthentication resources are created
- Check if all services have sidecars
- Look for certificate errors in sidecar logs

### Traffic policies not working
- Ensure VirtualServices/ServiceProfiles are correctly configured
- Check sidecar proxy configuration
- Verify service discovery is working

For more detailed troubleshooting, check the operator logs:

```bash
kubectl logs -n gunj-system deployment/gunj-operator -f | grep servicemesh
```
