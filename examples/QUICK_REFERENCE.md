# Quick Reference: Example Selection Guide

Use this guide to quickly find the right ObservabilityPlatform example for your use case.

## 🎯 By Use Case

### I want to...

#### **Just try it out quickly**
→ Use [`basic/minimal-platform.yaml`](basic/minimal-platform.yaml)
- Simplest configuration
- Prometheus + Grafana only
- All defaults

#### **Set up a development environment**
→ Use [`basic/dev-platform.yaml`](basic/dev-platform.yaml)
- All components enabled
- Minimal resources
- Debug logging
- Short retention

#### **Deploy to production**
→ Use [`production/production-ha-platform.yaml`](production/production-ha-platform.yaml)
- High availability (3 replicas)
- Security hardening
- 90-day retention
- Full monitoring stack

#### **Support multiple teams**
→ Use [`multi-tenant/multi-tenant-platform.yaml`](multi-tenant/multi-tenant-platform.yaml)
- Namespace isolation
- Per-team quotas
- Separate Grafana orgs
- Cost allocation

#### **Integrate with GitOps**
→ Use [`advanced/gitops-platform.yaml`](advanced/gitops-platform.yaml)
- ArgoCD/Flux integration
- Config from Git
- Automated upgrades
- Progressive rollouts

#### **Optimize costs**
→ Use [`advanced/cost-optimized-platform.yaml`](advanced/cost-optimized-platform.yaml)
- Spot instance support
- Resource right-sizing
- Data tiering
- ~60% cost reduction

#### **Ensure disaster recovery**
→ Use [`advanced/disaster-recovery-platform.yaml`](advanced/disaster-recovery-platform.yaml)
- Multi-region setup
- 5-minute RPO
- Automated backups
- Fast failover

#### **Deploy in air-gapped environment**
→ Use [`advanced/airgapped-platform.yaml`](advanced/airgapped-platform.yaml)
- No internet required
- Internal registries
- FIPS compliance
- Enhanced security

## 🏢 By Environment

| Environment | Example | Key Features |
|------------|---------|--------------|
| **Local Testing** | `basic/minimal-platform.yaml` | Minimal resources, quick setup |
| **Development** | `basic/dev-platform.yaml` | All components, debug mode |
| **Staging** | `production/production-ha-platform.yaml` | Production-like, reduced resources |
| **Production** | `production/production-ha-platform.yaml` | HA, security, monitoring |
| **Enterprise** | `multi-tenant/multi-tenant-platform.yaml` | Multi-team, chargeback |
| **Regulated** | `advanced/airgapped-platform.yaml` | Compliance, audit, security |

## 💰 By Budget

| Budget | Example | Est. Monthly Cost |
|--------|---------|-------------------|
| **Minimal** | `basic/minimal-platform.yaml` | ~$50 |
| **Small** | `basic/dev-platform.yaml` | ~$150 |
| **Optimized** | `advanced/cost-optimized-platform.yaml` | ~$800 |
| **Standard** | `production/production-ha-platform.yaml` | ~$2,000 |
| **Enterprise** | `multi-tenant/multi-tenant-platform.yaml` | ~$100/team |

## 🔧 By Features

### Security Features

| Feature | Examples |
|---------|----------|
| **OAuth/OIDC** | `production/production-ha-platform.yaml` |
| **mTLS** | `production/production-ha-platform.yaml`, `advanced/airgapped-platform.yaml` |
| **RBAC** | All production examples |
| **Network Policies** | `production/production-ha-platform.yaml`, `multi-tenant/multi-tenant-platform.yaml` |
| **FIPS Compliance** | `advanced/airgapped-platform.yaml` |

### Storage Options

| Storage Type | Examples |
|--------------|----------|
| **Local Storage** | `basic/*.yaml` |
| **Cloud Storage (S3)** | `production/production-ha-platform.yaml` |
| **Multi-tier Storage** | `advanced/cost-optimized-platform.yaml` |
| **NFS** | `advanced/airgapped-platform.yaml` |

### Integration Options

| Integration | Examples |
|-------------|----------|
| **GitOps** | `advanced/gitops-platform.yaml` |
| **Remote Write** | `production/production-ha-platform.yaml` |
| **Service Mesh** | `production/production-ha-platform.yaml` |
| **CI/CD Webhooks** | `advanced/gitops-platform.yaml` |

## 📊 Component Configurations

### Prometheus Variants

| Configuration | CPU | Memory | Storage | Retention | Example |
|--------------|-----|--------|---------|-----------|---------|
| **Minimal** | 250m | 512Mi | 10Gi | 7d | `basic/dev-platform.yaml` |
| **Standard** | 1 | 4Gi | 100Gi | 30d | `production/production-ha-platform.yaml` |
| **High Performance** | 4 | 16Gi | 500Gi | 90d | `production/production-ha-platform.yaml` |
| **Cost Optimized** | 500m | 2Gi | 100Gi | 15d | `advanced/cost-optimized-platform.yaml` |

### Deployment Modes

| Mode | Components | Use Case | Example |
|------|------------|----------|---------|
| **All-in-One** | All in single namespace | Development | `basic/dev-platform.yaml` |
| **Distributed** | Separate component deployments | Production | `production/production-ha-platform.yaml` |
| **Multi-Region** | Cross-region deployment | DR/HA | `advanced/disaster-recovery-platform.yaml` |
| **Multi-Tenant** | Shared with isolation | Enterprise | `multi-tenant/multi-tenant-platform.yaml` |

## 🚀 Getting Started

1. **Choose an example** based on your use case
2. **Copy and customize** the YAML file
3. **Validate** with `kubectl apply --dry-run=client -f your-platform.yaml`
4. **Deploy** with `kubectl apply -f your-platform.yaml`
5. **Monitor** the deployment with `kubectl get observabilityplatform -w`

## 💡 Tips

- Start with a minimal configuration and scale up
- Always set resource limits in production
- Enable monitoring of the monitoring platform
- Test backup and restore procedures
- Use GitOps for production deployments

## 📚 Further Reading

- [Detailed Example Documentation](README.md)
- [Gunj Operator User Guide](https://gunjanjp.github.io/gunj-operator/docs/user-guide)
- [Best Practices](https://gunjanjp.github.io/gunj-operator/docs/best-practices)
