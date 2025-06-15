# Multi-Cluster Observability Examples

This directory contains example configurations for setting up multi-cluster observability with the Gunj Operator.

## Examples

1. **cluster-registration.yaml** - Register multiple Kubernetes clusters
2. **federation-config.yaml** - Configure cluster federation
3. **resource-sync.yaml** - Set up cross-cluster resource synchronization
4. **network-mesh.yaml** - Configure cross-cluster networking
5. **failover-config.yaml** - Set up automatic failover
6. **federated-platform.yaml** - Deploy observability platform across clusters

## Quick Start

1. First, ensure the Gunj Operator is installed on your hub cluster:
```bash
helm install gunj-operator gunj-operator/gunj-operator \
  --namespace gunj-system \
  --create-namespace
```

2. Register your clusters:
```bash
kubectl apply -f cluster-registration.yaml
```

3. Configure federation:
```bash
kubectl apply -f federation-config.yaml
```

4. Deploy a federated observability platform:
```bash
kubectl apply -f federated-platform.yaml
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Hub Cluster   │     │  Spoke Cluster  │     │  Spoke Cluster  │
│                 │     │       US-1      │     │       EU-1      │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │   Gunj    │  │────▶│  │   Gunj    │  │────▶│  │   Gunj    │  │
│  │ Operator  │  │     │  │ Operator  │  │     │  │ Operator  │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
│                 │     │                 │     │                 │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │Federation │  │     │  │Prometheus │  │     │  │Prometheus │  │
│  │Controller │  │     │  │ Grafana   │  │     │  │ Grafana   │  │
│  └───────────┘  │     │  │   Loki    │  │     │  │   Loki    │  │
│                 │     │  └───────────┘  │     │  └───────────┘  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Use Cases

### 1. Multi-Region Deployment
Deploy observability stacks across multiple regions with automatic failover.

### 2. Development to Production Sync
Sync configurations from development to production clusters.

### 3. Disaster Recovery
Automatic failover when a cluster becomes unavailable.

### 4. Global Dashboards
Aggregate metrics and logs from all clusters in a single view.

## Best Practices

1. **Security**: Always use encrypted connections between clusters
2. **Resource Limits**: Set appropriate resource limits for each cluster
3. **Network Policies**: Implement strict network policies for cross-cluster communication
4. **Monitoring**: Monitor the federation system itself
5. **Backup**: Regular backups of cluster configurations and data
