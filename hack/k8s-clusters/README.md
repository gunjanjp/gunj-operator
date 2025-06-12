# Kubernetes Cluster Setup for Gunj Operator

This directory contains configurations and scripts for setting up local Kubernetes clusters for Gunj Operator development.

## Overview

We support multiple Kubernetes environments:

1. **Docker Desktop Kubernetes** (Recommended for Windows/macOS)
2. **Kind (Kubernetes in Docker)** - For CI/CD and multi-cluster testing
3. **Minikube** - Alternative local development

## Docker Desktop Setup

Since you already have Docker Desktop with Kubernetes enabled, you can use the following scripts:

### Verify Docker Desktop Kubernetes

```bash
# On Linux/macOS
./hack/k8s-clusters/verify-docker-desktop.sh

# On Windows PowerShell
# Run the script directly or use bash through WSL
```

### Setup Development Environment

```bash
# This will install all required components
./hack/k8s-clusters/setup-docker-desktop.sh
```

This script will:
- Install NGINX Ingress Controller
- Install metrics-server
- Install cert-manager
- Create development namespaces
- Set up local Docker registry
- Configure storage classes

## Kind Cluster Management

Kind is useful for:
- CI/CD testing
- Multi-node scenarios
- Clean isolated environments
- Testing different Kubernetes versions

### Available Cluster Types

1. **dev** - Single-node development cluster with port mappings
2. **ha** - Multi-node HA cluster (3 control planes, 3 workers)
3. **ci** - Minimal cluster optimized for CI/CD speed

### Using the Cluster Management Script

#### On Linux/macOS:

```bash
# Create a cluster
./hack/k8s-clusters/kind-cluster.sh create dev

# List all clusters
./hack/k8s-clusters/kind-cluster.sh list

# Switch context
./hack/k8s-clusters/kind-cluster.sh switch dev

# Setup cluster components
./hack/k8s-clusters/kind-cluster.sh setup dev

# Delete cluster
./hack/k8s-clusters/kind-cluster.sh delete dev
```

#### On Windows PowerShell:

```powershell
# Create a cluster
.\hack\k8s-clusters\kind-cluster.ps1 -Command create -ClusterType dev

# List all clusters
.\hack\k8s-clusters\kind-cluster.ps1 -Command list

# Switch context
.\hack\k8s-clusters\kind-cluster.ps1 -Command switch -ClusterType dev

# Setup cluster components
.\hack\k8s-clusters\kind-cluster.ps1 -Command setup -ClusterType dev

# Delete cluster
.\hack\k8s-clusters\kind-cluster.ps1 -Command delete -ClusterType dev
```

## Multi-Cluster Setup

For testing federation and multi-cluster features:

```bash
# Create all three clusters (hub, member-1, member-2)
kind create cluster --config=./hack/k8s-clusters/kind-multi-cluster.yaml --name=gunj-operator-hub
kind create cluster --config=./hack/k8s-clusters/kind-multi-cluster.yaml --name=gunj-operator-member-1
kind create cluster --config=./hack/k8s-clusters/kind-multi-cluster.yaml --name=gunj-operator-member-2

# Switch between clusters
kubectl config use-context kind-gunj-operator-hub
kubectl config use-context kind-gunj-operator-member-1
kubectl config use-context kind-gunj-operator-member-2
```

## Port Mappings

### Docker Desktop
- Ingress HTTP: `localhost:80`
- Ingress HTTPS: `localhost:443`
- Local Registry: `localhost:30500`

### Kind Dev Cluster
- Ingress HTTP: `localhost:80`
- Ingress HTTPS: `localhost:443`
- API Server: `localhost:8443`
- Prometheus: `localhost:9090`
- Grafana: `localhost:3000`
- Local Registry: `localhost:5000`

### Kind Multi-Cluster
- Hub Ingress: `localhost:8080` (HTTP), `localhost:8443` (HTTPS)
- Member clusters: No external ports (access via kubectl)

## Accessing Services

### Using Ingress

After deploying the operator, services will be available at:

```bash
# Docker Desktop or Kind
http://gunj-operator.localhost       # UI
http://api.gunj-operator.localhost   # API
http://grafana.localhost            # Grafana (when platform deployed)
http://prometheus.localhost         # Prometheus (when platform deployed)
```

### Using Port-Forward

For services without ingress:

```bash
# Forward operator API
kubectl port-forward -n gunj-operator-system svc/gunj-operator-api 8080:80

# Forward Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090

# Forward Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

## Local Registry

### Docker Desktop

The setup script creates a local registry at `localhost:30500`.

To use it:

```bash
# Tag image
docker tag gunj-operator:latest localhost:30500/gunj-operator:latest

# Push to local registry
docker push localhost:30500/gunj-operator:latest

# Use in Kubernetes
# Update image in deployment to: localhost:30500/gunj-operator:latest
```

### Kind

Kind clusters are configured to use a local registry at `localhost:5000`.

```bash
# Create registry container (one time)
docker run -d --restart=always -p 5000:5000 --name kind-registry registry:2

# Connect to kind network
docker network connect kind kind-registry

# Use in builds
docker tag gunj-operator:latest localhost:5000/gunj-operator:latest
docker push localhost:5000/gunj-operator:latest
```

## Troubleshooting

### Docker Desktop Issues

1. **Kubernetes not starting**
   - Check Docker Desktop settings
   - Reset Kubernetes in Docker Desktop settings
   - Increase Docker Desktop memory allocation

2. **Can't access services**
   - Check if ingress controller is running: `kubectl get pods -n ingress-nginx`
   - Verify service endpoints: `kubectl get endpoints -A`

### Kind Issues

1. **Port already in use**
   - Check for conflicting services
   - Use different ports in kind config
   - Stop conflicting containers: `docker ps` and `docker stop <container>`

2. **Cluster creation fails**
   - Check Docker daemon is running
   - Ensure sufficient disk space
   - Check Docker resource limits

3. **Can't connect to cluster**
   - Verify cluster is running: `kind get clusters`
   - Check kubeconfig: `kubectl config view`
   - Re-get credentials: `kind get kubeconfig --name=gunj-operator-dev`

## Best Practices

1. **Development Workflow**
   - Use Docker Desktop for day-to-day development
   - Use Kind for testing specific scenarios
   - Always clean up unused clusters to save resources

2. **Resource Management**
   - Limit resource usage in development
   - Use resource quotas in test namespaces
   - Monitor cluster resource usage

3. **Testing**
   - Test on both Docker Desktop and Kind
   - Use CI cluster for automated tests
   - Test multi-cluster scenarios regularly

## Next Steps

After setting up your cluster:

1. Build the operator: `make docker-build`
2. Deploy to cluster: `make deploy`
3. Create a sample platform: `kubectl apply -f config/samples/`
4. Access the UI: http://gunj-operator.localhost

For more information, see the main project documentation.
