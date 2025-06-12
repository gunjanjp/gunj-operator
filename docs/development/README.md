# Gunj Operator Development Documentation

Welcome to the Gunj Operator development documentation! This folder contains comprehensive guides, standards, and references for developers working on the Gunj Operator project.

## 📚 Documentation Structure

### Getting Started
- **[Developer Getting Started Guide](./getting-started.md)** - Set up your development environment and start contributing
- **[Architecture Overview](../architecture/README.md)** - Understand the system architecture

### Coding Standards
- **[Go Coding Standards](./coding-standards-go.md)** - Guidelines for Go code
- **[TypeScript/React Standards](./coding-standards-typescript.md)** - Guidelines for UI development

### Development Guides
- **[Operator Development](./operator/README.md)** - Working with controllers and CRDs
- **[API Development](./api/README.md)** - Building REST and GraphQL APIs
- **[UI Development](./ui/README.md)** - React component development

### Contributing
- **[Contributing Guidelines](../../CONTRIBUTING.md)** - How to contribute to the project
- **[Code of Conduct](../../CODE_OF_CONDUCT.md)** - Community standards

## 🚀 Quick Start

### Prerequisites

```bash
# Required tools
go version           # Go 1.21+
node --version       # Node.js 20+
docker --version     # Docker 24+
kubectl version      # Kubernetes 1.26+
helm version         # Helm 3.14+
```

### Development Setup

```bash
# Clone the repository
git clone https://github.com/gunjanjp/gunj-operator.git
cd gunj-operator

# Install dependencies
make install-deps

# Set up pre-commit hooks
make setup-hooks

# Start local development
make dev
```

## 🏗️ Project Structure

```
gunj-operator/
├── api/                    # API definitions (CRDs)
├── cmd/                    # Entry points
│   ├── operator/          # Operator main
│   ├── api-server/        # API server main
│   └── cli/              # CLI tool main
├── config/                # Kubernetes configurations
├── controllers/           # Operator controllers
├── docs/                  # User documentation
├── hack/                  # Development scripts
├── internal/              # Internal packages
│   ├── api/              # API implementation
│   ├── managers/         # Component managers
│   └── webhooks/         # Webhook handlers
├── pkg/                   # Public packages
├── test/                  # Test suites
├── ui/                    # React UI application
└── Makefile              # Build automation
```

## 🔧 Common Development Tasks

### Building

```bash
# Build all components
make build

# Build specific component
make build-operator
make build-api
make build-ui

# Build Docker images
make docker-build
```

### Testing

```bash
# Run all tests
make test

# Run specific test suites
make test-unit
make test-integration
make test-e2e

# Run with coverage
make test-coverage
```

### Running Locally

```bash
# Run operator outside cluster
make run

# Run with local Kubernetes (kind)
make kind-create
make deploy-local

# Run UI development server
make ui-dev
```

### Code Quality

```bash
# Format code
make fmt

# Run linters
make lint

# Run security scan
make security-scan

# Generate code
make generate
```

## 📋 Development Workflow

### 1. Create Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes

Follow the coding standards:
- [Go Standards](./coding-standards-go.md)
- [TypeScript Standards](./coding-standards-typescript.md)

### 3. Test Your Changes

```bash
# Run tests
make test

# Test in local cluster
make deploy-local
```

### 4. Submit Pull Request

- Use [conventional commits](https://www.conventionalcommits.org/)
- Update documentation
- Ensure all CI checks pass

## 🛠️ Key Development Areas

### Operator Development

Working on the Kubernetes operator:

```go
// Example: Adding a new reconciler
func (r *Reconciler) ReconcileNewComponent(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
    // Implementation
}
```

See [Operator Development Guide](./operator/README.md)

### API Development

Adding new API endpoints:

```go
// Example: New REST endpoint
router.GET("/api/v1/platforms/:name/metrics", h.GetPlatformMetrics)
```

See [API Development Guide](./api/README.md)

### UI Development

Creating React components:

```typescript
// Example: New component
export const PlatformMetrics: React.FC<Props> = ({ platform }) => {
    // Implementation
}
```

See [UI Development Guide](./ui/README.md)

## 🧪 Testing Guidelines

### Unit Tests

```go
func TestReconciler_DeployPrometheus(t *testing.T) {
    // Table-driven tests
    tests := []struct {
        name    string
        // ...
    }{
        // Test cases
    }
}
```

### Integration Tests

```go
var _ = Describe("Platform Controller", func() {
    It("should create all components", func() {
        // Ginkgo/Gomega tests
    })
})
```

### E2E Tests

```go
func TestE2EPlatformLifecycle(t *testing.T) {
    // Full lifecycle tests
}
```

## 📊 Performance Guidelines

- Use caching to reduce API calls
- Implement proper pagination
- Profile code for bottlenecks
- Monitor resource usage

## 🔒 Security Guidelines

- Never log sensitive information
- Validate all inputs
- Use least privilege RBAC
- Keep dependencies updated

## 🐛 Debugging

### Operator Debugging

```bash
# View operator logs
kubectl logs -n gunj-system deployment/gunj-operator -f

# Enable debug logging
kubectl set env -n gunj-system deployment/gunj-operator LOG_LEVEL=debug
```

### Remote Debugging

```bash
# Deploy with debug support
make deploy-debug

# Forward debug port
kubectl port-forward -n gunj-system deployment/gunj-operator 2345:2345
```

## 📚 Additional Resources

### Internal Documentation
- [Architecture Details](../architecture/)
- [API Reference](../api/)
- [User Guide](../user-guide/)

### External Resources
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Operator SDK](https://sdk.operatorframework.io/)
- [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime)

### Community
- [GitHub Discussions](https://github.com/gunjanjp/gunj-operator/discussions)
- [Slack Channel](https://kubernetes.slack.com/archives/gunj-operator)
- [Contributing Guide](../../CONTRIBUTING.md)

## 🤝 Getting Help

- **Documentation**: You're reading it!
- **GitHub Issues**: [Report bugs](https://github.com/gunjanjp/gunj-operator/issues)
- **Discussions**: [Ask questions](https://github.com/gunjanjp/gunj-operator/discussions)
- **Email**: gunjanjp@gmail.com

## 📈 Development Metrics

We track several metrics to ensure code quality:

- **Code Coverage**: Target > 80%
- **Linting**: Zero violations
- **Build Time**: < 5 minutes
- **Test Execution**: < 10 minutes

## 🎯 Development Principles

1. **Quality First**: Write clean, tested, documented code
2. **User Experience**: Consider the operator's perspective
3. **Performance**: Optimize for large-scale deployments
4. **Security**: Follow security best practices
5. **Community**: Be welcoming and helpful

---

Happy coding! If you have any questions or suggestions for improving the development experience, please don't hesitate to reach out.
