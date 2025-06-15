# GitOps Integration Examples

This directory contains example configurations for using the GitOps integration features of the Gunj Operator. The operator supports both ArgoCD and Flux for GitOps deployments with advanced features like:

- Multi-environment management
- Automated rollback on failures
- Drift detection and remediation
- Environment promotion workflows
- Git webhook integration

## Examples

1. **[argocd-basic.yaml](argocd-basic.yaml)** - Basic ArgoCD integration for a single environment
2. **[argocd-multi-env.yaml](argocd-multi-env.yaml)** - Multi-environment setup with ArgoCD
3. **[flux-basic.yaml](flux-basic.yaml)** - Basic Flux integration
4. **[flux-multi-env.yaml](flux-multi-env.yaml)** - Multi-environment setup with Flux
5. **[promotion-workflow.yaml](promotion-workflow.yaml)** - Automated promotion between environments
6. **[drift-detection.yaml](drift-detection.yaml)** - Drift detection and auto-remediation

## Prerequisites

### For ArgoCD Integration

1. Install ArgoCD in your cluster:
```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

2. Create Git credentials secret:
```bash
kubectl create secret generic git-credentials \
  --from-literal=username=YOUR_USERNAME \
  --from-literal=password=YOUR_TOKEN \
  -n monitoring
```

### For Flux Integration

1. Install Flux in your cluster:
```bash
flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=$GITHUB_REPO \
  --branch=main \
  --path=./clusters/my-cluster \
  --personal
```

2. Create Git credentials secret (if using private repos):
```bash
kubectl create secret generic git-credentials \
  --from-literal=username=git \
  --from-file=identity=/path/to/private-key \
  --from-file=known_hosts=/path/to/known_hosts \
  -n monitoring
```

## Quick Start

1. Choose your GitOps engine (ArgoCD or Flux)
2. Apply the appropriate example:
```bash
# For ArgoCD
kubectl apply -f argocd-basic.yaml

# For Flux
kubectl apply -f flux-basic.yaml
```

3. Monitor the deployment:
```bash
kubectl get gitopsdeployment -n monitoring
kubectl describe gitopsdeployment observability-platform -n monitoring
```

## Features Demonstrated

### Multi-Environment Support
- Development, staging, and production environments
- Different configurations per environment
- Environment-specific namespaces

### Automated Promotion
- Promotion from dev → staging → production
- Configurable promotion policies
- Manual approval gates

### Rollback Capabilities
- Automatic rollback on failures
- Configurable failure thresholds
- Revision history tracking

### Drift Detection
- Periodic drift checks
- Auto-remediation options
- Configurable ignore patterns

### Git Integration
- Webhook support for real-time updates
- Branch-based deployments
- Path-based configurations

## Monitoring

Check deployment status:
```bash
# Get overall status
kubectl get gitopsdeployment -A

# Get detailed status
kubectl describe gitopsdeployment observability-platform -n monitoring

# Watch events
kubectl get events -n monitoring --field-selector involvedObject.kind=GitOpsDeployment
```

## Troubleshooting

1. **Deployment stuck in "Initializing"**
   - Check if GitOps engine (ArgoCD/Flux) is installed
   - Verify Git credentials are correct
   - Check operator logs: `kubectl logs -n gunj-system -l app=gunj-operator`

2. **Sync failures**
   - Verify Git repository access
   - Check branch/path configuration
   - Review GitOps engine logs

3. **Promotion not working**
   - Ensure source environment is healthy
   - Check promotion conditions
   - Verify approval workflow if configured

4. **Drift detection issues**
   - Check drift detection interval
   - Review ignored fields configuration
   - Verify RBAC permissions

## Advanced Usage

See the individual example files for detailed configuration options and advanced features.
