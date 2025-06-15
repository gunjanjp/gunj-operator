# Service Mesh Integration Examples

This directory contains examples of how to configure service mesh integration with the Gunj Operator.

## Examples

1. **basic-istio.yaml** - Basic Istio integration with mTLS
2. **basic-linkerd.yaml** - Basic Linkerd integration
3. **advanced-traffic-management.yaml** - Advanced traffic management features
4. **canary-deployment.yaml** - Canary deployment configuration
5. **fault-injection.yaml** - Fault injection for testing
6. **rate-limiting.yaml** - Rate limiting configuration
7. **circuit-breaker.yaml** - Circuit breaker configuration
8. **observability-mesh.yaml** - Service mesh observability features

## Prerequisites

Before using these examples, ensure you have:

1. A Kubernetes cluster (v1.26+)
2. Gunj Operator installed
3. Either Istio or Linkerd installed (depending on the example)

### Installing Istio

```bash
# Download and install Istio
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# Install Istio with demo configuration
istioctl install --set profile=demo -y

# Enable Istio injection for your namespace
kubectl label namespace <your-namespace> istio-injection=enabled
```

### Installing Linkerd

```bash
# Install Linkerd CLI
curl --proto '=https' --tlsv1.2 -sSfL https://run.linkerd.io/install | sh
export PATH=$PATH:$HOME/.linkerd2/bin

# Install Linkerd control plane
linkerd install | kubectl apply -f -
linkerd check

# Enable Linkerd injection for your namespace
kubectl annotate namespace <your-namespace> linkerd.io/inject=enabled
```

## Usage

1. Choose an example that fits your needs
2. Modify the namespace and other parameters as needed
3. Apply the configuration:

```bash
kubectl apply -f <example-file>.yaml
```

4. Monitor the platform status:

```bash
kubectl get observabilityplatform -n <namespace> -w
```

5. Check service mesh status:

```bash
kubectl describe observabilityplatform <platform-name> -n <namespace>
```

## Verifying Service Mesh Integration

### For Istio

```bash
# Check if sidecars are injected
kubectl get pods -n <namespace> -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].name}{"\n"}{end}'

# Check mTLS status
istioctl authn tls-check <pod-name>.<namespace>

# View service mesh metrics
kubectl exec -n <namespace> -c prometheus <prometheus-pod> -- curl -s localhost:9090/api/v1/query?query=istio_request_total
```

### For Linkerd

```bash
# Check if proxies are injected
linkerd -n <namespace> check --proxy

# View service mesh metrics
linkerd -n <namespace> stat deploy

# Check traffic routes
linkerd -n <namespace> routes deploy/<deployment-name>
```

## Troubleshooting

### Common Issues

1. **Sidecars not injected**
   - Ensure namespace has proper labels/annotations
   - Restart pods after enabling injection

2. **mTLS not working**
   - Check if PeerAuthentication resources are created
   - Verify certificates are valid

3. **Traffic policies not applied**
   - Check VirtualService/DestinationRule (Istio)
   - Check ServiceProfile (Linkerd)
   - Verify label selectors match

4. **Health checks failing**
   - Check control plane status
   - Verify all pods have sidecars
   - Review operator logs

### Getting Help

For issues or questions:
1. Check the operator logs: `kubectl logs -n gunj-system -l app=gunj-operator`
2. Review service mesh documentation
3. Open an issue on GitHub
