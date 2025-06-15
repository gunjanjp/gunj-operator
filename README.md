# Gunj Operator

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Kubernetes Version](https://img.shields.io/badge/Kubernetes-1.26+-blue.svg)](https://kubernetes.io)
[![CNCF Status](https://img.shields.io/badge/CNCF-Sandbox%20Ready-brightgreen.svg)](https://www.cncf.io)

A next-generation Kubernetes operator for deploying and managing enterprise observability platforms. The Gunj Operator simplifies the deployment, configuration, and lifecycle management of observability stacks including Prometheus, Grafana, Loki, and Tempo.

## 🚀 Features

- **🎯 Kubernetes Native**: Full operator pattern implementation following CNCF best practices
- **📊 Complete Observability Stack**: Automated deployment of Prometheus, Grafana, Loki, and Tempo
- **🔄 GitOps Integration**: Native support for ArgoCD and Flux with automatic rollback
- **🌐 Multi-Environment Support**: Manage dev, staging, and production with automated promotion
- **💻 Web UI**: Beautiful React-based management interface
- **🔌 API First**: RESTful and GraphQL APIs for automation
- **🔒 Enterprise Security**: OIDC, SAML, LDAP integration with RBAC
- **📈 Auto-scaling**: Resource optimization and automatic scaling
- **💾 Backup & Restore**: Automated backup with multiple storage backends
- **🏥 Self-Healing**: Automatic failure detection and recovery

## 📋 Table of Contents

- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [GitOps Integration](#gitops-integration)
- [API Reference](#api-reference)
- [Web UI](#web-ui)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## 🏗️ Architecture

The Gunj Operator follows the Kubernetes operator pattern and consists of:

- **Operator Core**: Manages the lifecycle of observability components
- **CRDs**: Custom Resource Definitions for platform configuration
- **API Server**: RESTful and GraphQL APIs for external integration
- **Web UI**: React-based management interface
- **GitOps Manager**: Integration with ArgoCD and Flux

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Kubernetes    │     │   Gunj API      │     │    Gunj UI      │
│   API Server    │◄────┤     Server      │◄────┤   (React SPA)   │
└────────┬────────┘     └─────────────────┘     └─────────────────┘
         │                        ▲
         ▼                        │
┌─────────────────┐              │
│  Gunj Operator  │──────────────┘
│                 │
│  ┌───────────┐  │     ┌─────────────────────────────────────────┐
│  │Controller │  │     │         Observability Platform          │
│  └─────┬─────┘  │     │                                         │
│        │        │     │  ┌────────────┐    ┌────────────┐      │
│  ┌─────▼─────┐  │     │  │ Prometheus │    │  Grafana   │      │
│  │ Managers  │  │────▶│  └────────────┘    └────────────┘      │
│  └───────────┘  │     │                                         │
│                 │     │  ┌────────────┐    ┌────────────┐      │
└─────────────────┘     │  │    Loki    │    │   Tempo    │      │
                        │  └────────────┘    └────────────┘      │
                        └─────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (v1.26+)
- kubectl configured
- Helm 3.14+ (optional)

### Install the Operator

```bash
# Using kubectl
kubectl apply -f https://github.com/gunjanjp/gunj-operator/releases/latest/download/install.yaml

# Or using Helm
helm repo add gunj-operator https://gunjanjp.github.io/gunj-operator/charts
helm install gunj-operator gunj-operator/gunj-operator \
  --namespace gunj-system \
  --create-namespace
```

### Deploy Your First Platform

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
    grafana:
      enabled: true
      version: "10.2.0"
    loki:
      enabled: true
      version: "2.9.0"
    tempo:
      enabled: true
      version: "2.3.0"
```

```bash
kubectl apply -f platform.yaml
```

## 📦 Installation

### Production Installation

For production deployments, see our [Installation Guide](docs/installation.md) which covers:

- High availability configuration
- Security hardening
- Resource sizing
- Network policies
- Backup configuration

## ⚙️ Configuration

### Basic Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
  namespace: monitoring
spec:
  # Component configuration
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      resources:
        requests:
          memory: "4Gi"
          cpu: "1"
      storage:
        size: 100Gi
      retention: 30d
    
    grafana:
      enabled: true
      version: "10.2.0"
      ingress:
        enabled: true
        host: grafana.example.com
  
  # High Availability
  highAvailability:
    enabled: true
    minReplicas: 3
  
  # Security
  security:
    tls:
      enabled: true
      autoTLS: true
    authentication:
      type: oidc
      oidc:
        issuer: https://auth.example.com
        clientId: gunj-operator
```

For detailed configuration options, see [Configuration Guide](docs/configuration.md).

## 🔄 GitOps Integration

The Gunj Operator provides native GitOps integration with ArgoCD and Flux, enabling:

- **Declarative Configuration**: Store platform configs in Git
- **Multi-Environment Management**: Automated promotion between environments
- **Automatic Rollback**: Detect failures and rollback automatically
- **Drift Detection**: Ensure configuration matches desired state

### ArgoCD Example

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
  namespace: observability
spec:
  components:
    # ... component configuration ...
  
  gitOps:
    provider: argocd
    repository:
      url: https://github.com/your-org/observability-configs.git
      branch: main
      path: platforms
    
    environments:
      - name: dev
        namespace: observability-dev
        branch: develop
        
      - name: staging
        namespace: observability-staging
        promotionPolicy:
          autoPromotion: true
          dependsOn: dev
          promoteAfter: 1h
      
      - name: production
        namespace: observability-prod
        promotionPolicy:
          approvalRequired: true
          dependsOn: staging
    
    rollbackConfig:
      autoRollback: true
      failureThreshold: 3
      window: 30m
```

See [GitOps Examples](examples/gitops/) for more detailed examples.

## 🔌 API Reference

### REST API

The operator provides a comprehensive REST API:

```bash
# Get all platforms
curl -H "Authorization: Bearer $TOKEN" \
  https://api.gunj-operator.example.com/api/v1/platforms

# Create a platform
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @platform.json \
  https://api.gunj-operator.example.com/api/v1/platforms

# Trigger GitOps sync
curl -X POST -H "Authorization: Bearer $TOKEN" \
  https://api.gunj-operator.example.com/api/v1/platforms/production/sync
```

### GraphQL API

```graphql
query GetPlatforms {
  platforms {
    name
    namespace
    status {
      phase
      health
      components {
        name
        ready
      }
    }
  }
}

mutation PromoteEnvironment {
  promoteEnvironment(
    platform: "production",
    from: "staging",
    to: "production"
  ) {
    success
    message
  }
}
```

Full API documentation: [API Reference](docs/api-reference.md)

## 💻 Web UI

The Gunj Operator includes a modern React-based web interface:

- **Dashboard**: Real-time platform status and health
- **Platform Management**: Create, update, and delete platforms
- **GitOps Control**: Manage deployments and promotions
- **Monitoring**: Built-in dashboards and metrics
- **Configuration Editor**: Visual configuration with validation

Access the UI at `https://gunj-operator.example.com` after installation.

## 🛠️ Development

### Prerequisites

- Go 1.21+
- Node.js 20+
- Docker
- kind or minikube
- Make

### Setup Development Environment

```bash
# Clone the repository
git clone https://github.com/gunjanjp/gunj-operator.git
cd gunj-operator

# Install dependencies
make install-deps

# Run locally
make run

# Run tests
make test

# Build images
make docker-build
```

See [Development Guide](docs/development.md) for detailed instructions.

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- Code of Conduct
- Development workflow
- Coding standards
- Testing requirements
- Pull request process

## 📚 Documentation

- [User Guide](docs/user-guide/)
- [Administrator Guide](docs/admin-guide/)
- [API Reference](docs/api-reference/)
- [Architecture](docs/architecture/)
- [Troubleshooting](docs/troubleshooting.md)

## 🔒 Security

For security issues, please email gunjanjp@gmail.com directly instead of using the issue tracker.

See [SECURITY.md](SECURITY.md) for our security policy.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- The Kubernetes community for the operator pattern
- Prometheus, Grafana, Loki, and Tempo projects
- CNCF for guidance and best practices
- All our contributors and users

## 📞 Contact

- **Project Lead**: Gunjan Patil (gunjanjp@gmail.com)
- **Slack**: #gunj-operator on CNCF Slack
- **Website**: https://gunjanjp.github.io/gunj-operator

---

**Made with ❤️ by the Gunj Operator Community**
