# Gunj Operator 🚀

[![CI Pipeline](https://github.com/gunjanjp/gunj-operator/actions/workflows/ci.yml/badge.svg)](https://github.com/gunjanjp/gunj-operator/actions/workflows/ci.yml)
[![Security Scan](https://github.com/gunjanjp/gunj-operator/actions/workflows/security.yml/badge.svg)](https://github.com/gunjanjp/gunj-operator/actions/workflows/security.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Enterprise Observability Platform - Next-generation Kubernetes operator for deploying and managing Prometheus, Grafana, Loki, and Tempo.

## 🎯 Features

- **Kubernetes Native**: Built using the operator pattern
- **Complete Observability**: Metrics, logs, traces, and dashboards
- **API First**: RESTful and GraphQL APIs
- **Modern UI**: React-based management interface
- **GitOps Ready**: Seamless ArgoCD/Flux integration
- **Multi-Cluster**: Manage observability across clusters
- **Enterprise Ready**: RBAC, audit logging, and compliance

## 🚀 Quick Start

### Prerequisites

- Kubernetes 1.26+
- Helm 3.14+
- kubectl configured
- Docker Hub account (for development)

### Installation

```bash
# Add Helm repository
helm repo add gunj-operator https://gunjanjp.github.io/gunj-operator/charts
helm repo update

# Install operator
helm install gunj-operator gunj-operator/gunj-operator \
  --namespace gunj-system \
  --create-namespace

# Create your first observability platform
kubectl apply -f examples/basic-platform.yaml
```

## 🔧 Development Setup

### 1. Clone Repository

```bash
git clone https://github.com/gunjanjp/gunj-operator.git
cd gunj-operator
```

### 2. Set Up Secrets

```bash
# Run the interactive setup script
./setup-local-secrets.sh

# Load secrets into your shell
source ./load-secrets.sh

# Verify Docker Hub access
./test-docker-login.sh
```

### 3. Configure GitHub Secrets

For CI/CD to work, configure these secrets in your GitHub repository:

| Secret | Required | Description |
|--------|----------|-------------|
| `DOCKER_USERNAME` | ✅ | Docker Hub username |
| `DOCKER_PASSWORD` | ✅ | Docker Hub access token |
| `SLACK_WEBHOOK_URL` | ❌ | Slack notifications |
| `GPG_PRIVATE_KEY` | ❌ | Code signing |
| `SNYK_TOKEN` | ❌ | Security scanning |

See [Secret Management Guide](docs/security/secret-management.md) for detailed instructions.

### 4. Run Tests

```bash
# Verify secret configuration
gh workflow run secret-test.yml

# Run full CI pipeline
make test
```

## 🔒 Security

Security is a top priority for the Gunj Operator project.

### Security Features

- **Secret Management**: Automated rotation reminders and secure storage
- **Container Security**: Distroless images, non-root execution
- **Supply Chain**: SBOM generation, image signing
- **Vulnerability Scanning**: Integrated Snyk and Trivy scanning
- **Compliance**: CNCF best practices, CIS benchmarks

### Security Documentation

- [Secret Management Guide](docs/security/secret-management.md)
- [Security Checklist](docs/security/security-checklist.md)
- [Security Policy](SECURITY.md)

### Reporting Security Issues

**DO NOT** create public issues for security vulnerabilities.

Email: gunjanjp@gmail.com

We will respond within 24 hours and work with you to resolve the issue.

## 📚 Documentation

- [Installation Guide](docs/installation.md)
- [User Manual](docs/user-manual.md)
- [API Reference](docs/api-reference.md)
- [Development Guide](docs/development.md)
- [Architecture](docs/architecture.md)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md).

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests locally
5. Submit a pull request

### Code of Conduct

This project follows the [CNCF Code of Conduct](CODE_OF_CONDUCT.md).

## 🏗️ Project Structure

```
gunj-operator/
├── .github/              # GitHub Actions workflows
│   ├── workflows/       # CI/CD pipelines
│   └── README.md        # Workflow documentation
├── api/                 # CRD definitions
├── cmd/                 # Entry points
├── config/              # Kubernetes manifests
├── controllers/         # Operator logic
├── docs/               # Documentation
│   └── security/       # Security documentation
├── internal/           # Internal packages
├── pkg/                # Public packages
├── test/               # Test suites
├── ui/                 # React UI
└── setup-local-secrets.sh  # Local development setup
```

## 📊 Roadmap

- [x] Phase 1: Foundation & Architecture
- [ ] Phase 2: Operator Core Development
- [ ] Phase 3: API Development
- [ ] Phase 4: UI Development
- [ ] Phase 5: Testing Framework
- [ ] Phase 6: CI/CD Pipeline
- [ ] Phase 7: Documentation
- [ ] Phase 8: Migration & Rollout

See [ROADMAP.md](ROADMAP.md) for details.

## 📞 Support

- **Documentation**: https://gunjanjp.github.io/gunj-operator
- **Issues**: https://github.com/gunjanjp/gunj-operator/issues
- **Discussions**: https://github.com/gunjanjp/gunj-operator/discussions
- **Email**: gunjanjp@gmail.com

## 📄 License

This project is licensed under the MIT License - see [LICENSE](LICENSE) file.

## 🙏 Acknowledgments

- CNCF for the cloud-native ecosystem
- Kubernetes community for the operator pattern
- All contributors and users

---

**Current Phase**: Foundation & Architecture (Phase 1)  
**Version**: v2.0.0-alpha  
**Status**: In Development  

Made with ❤️ by the Gunj Operator Team