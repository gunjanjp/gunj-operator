# Developer Getting Started Guide

Welcome to the Gunj Operator development team! This guide will help you set up your development environment and start contributing to the project.

## Prerequisites

Before you begin, ensure you have the following tools installed:

### Required Tools

| Tool | Version | Installation Guide |
|------|---------|-------------------|
| Go | 1.21+ | [golang.org/dl](https://golang.org/dl/) |
| Node.js | 20+ | [nodejs.org](https://nodejs.org/) |
| Docker | 24+ | [docker.com/get-docker](https://docker.com/get-docker) |
| kubectl | 1.26+ | [kubernetes.io/docs/tasks/tools](https://kubernetes.io/docs/tasks/tools/) |
| Helm | 3.14+ | [helm.sh/docs/intro/install](https://helm.sh/docs/intro/install/) |
| Kind | 0.20+ | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/) |
| Git | 2.40+ | [git-scm.com](https://git-scm.com/) |

### Optional Tools

- **make**: Build automation (usually pre-installed on Linux/Mac)
- **jq**: JSON processing
- **yq**: YAML processing
- **stern**: Multi-pod log tailing
- **k9s**: Kubernetes CLI UI

## Environment Setup

### 1. Clone the Repository

```bash
git clone https://github.com/gunjanjp/gunj-operator.git
cd gunj-operator
```

### 2. Install Development Dependencies

```bash
# Install Go dependencies
make install-go-deps

# Install Node.js dependencies for UI
cd ui && npm install && cd ..

# Install development tools
make install-tools
```

### 3. Set Up Pre-commit Hooks

```bash
make setup-hooks
```

This will install hooks for:
- Code formatting
- Linting
- Commit message validation
- Security scanning

### 4. Configure Your IDE

#### VS Code (Recommended)

1. Install recommended extensions:
   ```bash
   code --install-extension golang.go
   code --install-extension ms-kubernetes-tools.vscode-kubernetes-tools
   code --install-extension esbenp.prettier-vscode
   code --install-extension dbaeumer.vscode-eslint
   ```

2. Copy the workspace settings:
   ```bash
   cp .vscode/settings.json.example .vscode/settings.json
   ```

#### GoLand/IntelliJ

1. Import the project as a Go module
2. Configure the Go SDK (1.21+)
3. Enable Go modules support
4. Configure code style from `.editorconfig`

## Local Development

### Starting the Development Environment

```bash
# Start local Kubernetes cluster
make kind-create

# Build and load operator image
make docker-build
make kind-load

# Deploy operator to local cluster
make deploy-local

# Start UI development server
make ui-dev
```

### Running the Operator Locally

For rapid development, you can run the operator outside the cluster:

```bash
# Set up RBAC and CRDs
make install

# Run operator locally (with debugger support)
make run
```

### Accessing the UI

After starting the UI dev server:
- UI: http://localhost:3000
- API: http://localhost:8080
- GraphQL Playground: http://localhost:8080/graphql

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Your Changes

Follow our coding standards:
- [Go Coding Standards](./coding-standards-go.md)
- [TypeScript/React Standards](./coding-standards-typescript.md)

### 3. Run Tests

```bash
# Run all tests
make test

# Run specific test suites
make test-unit        # Unit tests only
make test-integration # Integration tests
make test-e2e        # End-to-end tests

# Run UI tests
make ui-test
```

### 4. Build and Verify

```bash
# Build all components
make build

# Verify your changes
make verify
```

### 5. Commit Your Changes

Use conventional commits:
```bash
git commit -m "feat(operator): add new reconciliation logic"
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### 6. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## Debugging

### Operator Debugging

1. **VS Code Debugging**:
   - Use the provided launch configuration
   - Set breakpoints in your code
   - Press F5 to start debugging

2. **Remote Debugging**:
   ```bash
   # Deploy operator with debug support
   make deploy-debug
   
   # Forward debug port
   kubectl port-forward -n gunj-system deployment/gunj-operator 2345:2345
   
   # Connect your debugger to localhost:2345
   ```

3. **Logging**:
   ```bash
   # View operator logs
   kubectl logs -n gunj-system -l app=gunj-operator -f
   
   # Increase log verbosity
   kubectl set env -n gunj-system deployment/gunj-operator LOG_LEVEL=debug
   ```

### UI Debugging

1. **Browser DevTools**: Use React Developer Tools extension
2. **Network Debugging**: Monitor API calls in Network tab
3. **State Debugging**: Use Redux DevTools (if applicable)

## Common Tasks

### Adding a New CRD Field

1. Update the API types in `api/v1beta1/`
2. Run code generation: `make generate`
3. Update validation and defaults
4. Add tests for the new field
5. Update documentation

### Adding a New API Endpoint

1. Define the endpoint in `internal/api/routes.go`
2. Implement the handler in `internal/api/handlers/`
3. Add request/response types
4. Write tests
5. Update OpenAPI spec
6. Update API documentation

### Adding a New UI Component

1. Create component in `ui/src/components/`
2. Add TypeScript types
3. Write unit tests
4. Add to Storybook (if applicable)
5. Update component documentation

## Troubleshooting

### Common Issues

#### Kind cluster issues
```bash
# Reset Kind cluster
make kind-delete
make kind-create
```

#### Port conflicts
```bash
# Check for port usage
lsof -i :3000  # UI port
lsof -i :8080  # API port
```

#### Module issues
```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

### Getting Help

- Check [Troubleshooting Guide](../troubleshooting.md)
- Ask in #gunj-operator-dev Slack channel
- Create a GitHub issue
- Join our community calls

## Next Steps

1. Read the [Architecture Overview](../architecture/README.md)
2. Review [Contribution Guidelines](../../CONTRIBUTING.md)
3. Explore the [API Documentation](../api/README.md)
4. Join the next community call

## Useful Commands

```bash
# Development
make run            # Run operator locally
make ui-dev        # Start UI dev server
make test          # Run all tests
make lint          # Run linters
make fmt           # Format code

# Building
make build         # Build all binaries
make docker-build  # Build Docker images
make helm-package  # Package Helm chart

# Deployment
make deploy-local  # Deploy to local cluster
make undeploy     # Remove from cluster
make install-crd  # Install CRDs only

# Utilities
make generate     # Generate code
make manifests    # Generate manifests
make docs         # Generate documentation
make clean        # Clean build artifacts
```

Welcome aboard! We're excited to have you contributing to the Gunj Operator project! 🚀
