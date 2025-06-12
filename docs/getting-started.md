# Getting Started with Gunj Operator

This guide will help you get the Gunj Operator up and running in your Kubernetes cluster in just a few minutes!

## Prerequisites

Before you begin, ensure you have:

- ✅ Kubernetes cluster (v1.26+)
- ✅ kubectl configured to access your cluster
- ✅ Helm 3.14+ installed
- ✅ 4 CPU cores and 8GB RAM available in your cluster

## Quick Installation

### 1. Add the Helm Repository

```bash
helm repo add gunj-operator https://gunjanjp.github.io/gunj-operator/charts
helm repo update
```

### 2. Install the Operator

```bash
# Create namespace
kubectl create namespace gunj-system

# Install operator
helm install gunj-operator gunj-operator/gunj-operator \
  --namespace gunj-system \
  --wait
```

### 3. Verify Installation

```bash
# Check operator pod
kubectl get pods -n gunj-system

# Output should show:
# NAME                            READY   STATUS    RESTARTS   AGE
# gunj-operator-7b9c5d4f6-xxxxx  1/1     Running   0          1m
```

## Deploy Your First Observability Platform

### 1. Create a Simple Platform

Create a file named `my-platform.yaml`:

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: demo-platform
  namespace: default
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      storage:
        size: 10Gi
    grafana:
      enabled: true
      version: "10.2.0"
      adminPassword: "MySecurePassword123!" # Change this!
```

### 2. Apply the Configuration

```bash
kubectl apply -f my-platform.yaml
```

### 3. Watch the Deployment

```bash
# Watch the platform status
kubectl get observabilityplatform demo-platform -w

# Output:
# NAME            PHASE        AGE
# demo-platform   Installing   10s
# demo-platform   Ready        2m
```

### 4. Access Your Components

Once the platform is ready, get the access information:

```bash
# Get Grafana URL
kubectl get svc -n default grafana-demo-platform

# Port forward to access locally
kubectl port-forward -n default svc/grafana-demo-platform 3000:3000
```

Now open http://localhost:3000 in your browser and login with:
- Username: `admin`
- Password: `MySecurePassword123!`

## What's Next?

### Explore More Features

<div class="grid cards" markdown>

-   :material-web: **[Access the Web UI](user-guide/ui/getting-started.md)**

    ---

    Use the graphical interface for easier management

-   :material-shield-check: **[Configure Security](user-guide/configuration/security.md)**

    ---

    Set up RBAC, network policies, and authentication

-   :material-chart-line: **[Add Monitoring](user-guide/configuration/components.md)**

    ---

    Configure Loki for logs and Tempo for traces

-   :material-git: **[GitOps Integration](user-guide/operations/gitops.md)**

    ---

    Integrate with ArgoCD or Flux

</div>

### Try Advanced Examples

#### High Availability Setup

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: ha-platform
spec:
  highAvailability: true
  components:
    prometheus:
      enabled: true
      replicas: 3
      resources:
        requests:
          memory: "4Gi"
          cpu: "1"
    grafana:
      enabled: true
      replicas: 2
```

#### Multi-Component Platform

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: full-platform
spec:
  components:
    prometheus:
      enabled: true
      retention: 30d
    grafana:
      enabled: true
    loki:
      enabled: true
      storage:
        size: 100Gi
    tempo:
      enabled: true
      storage:
        size: 50Gi
```

## Common Operations

### Check Platform Status

```bash
# Get detailed status
kubectl describe observabilityplatform demo-platform

# View events
kubectl get events --field-selector involvedObject.name=demo-platform
```

### Update Components

```bash
# Update Prometheus version
kubectl patch observabilityplatform demo-platform --type merge -p '
{
  "spec": {
    "components": {
      "prometheus": {
        "version": "v2.49.0"
      }
    }
  }
}'
```

### Scale Components

```bash
# Scale Prometheus to 3 replicas
kubectl patch observabilityplatform demo-platform --type merge -p '
{
  "spec": {
    "components": {
      "prometheus": {
        "replicas": 3
      }
    }
  }
}'
```

## Troubleshooting

### Platform Stuck in Installing

Check operator logs:
```bash
kubectl logs -n gunj-system deployment/gunj-operator
```

### Component Not Starting

Check component pods:
```bash
kubectl get pods -l app.kubernetes.io/managed-by=gunj-operator
```

### Resource Issues

Check resource availability:
```bash
kubectl describe nodes
kubectl top nodes
```

## Clean Up

To remove the demo platform:

```bash
# Delete the platform
kubectl delete observabilityplatform demo-platform

# Uninstall the operator (optional)
helm uninstall gunj-operator -n gunj-system

# Delete the namespace (optional)
kubectl delete namespace gunj-system
```

## Need Help?

- 📖 Read the [User Guide](user-guide/index.md)
- 💬 Ask in [GitHub Discussions](https://github.com/gunjanjp/gunj-operator/discussions)
- 🐛 Report issues on [GitHub](https://github.com/gunjanjp/gunj-operator/issues)
- 📧 Email: gunjanjp@gmail.com

---

**Congratulations!** 🎉 You've successfully deployed your first observability platform with the Gunj Operator. Continue exploring the documentation to learn about advanced features and best practices.
