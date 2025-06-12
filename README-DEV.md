# Gunj Operator Development Environment

This directory contains the complete development environment setup for the Gunj Operator project.

## 🚀 Quick Start

```bash
# Start the development environment
make quickstart

# Enter the development container
make dev-shell

# Start operator with hot reload
make dev-operator-watch

# In another terminal, start API with hot reload
make dev-api-watch
```

## 📋 Prerequisites

- Docker Desktop or Docker Engine with Docker Compose
- VS Code with Remote Containers extension (optional but recommended)
- kubectl configured for your Kubernetes cluster (optional)

## 🏗️ Architecture

The development environment includes:

### Core Services
- **dev**: Main development container with all tools
- **postgres**: PostgreSQL database for API data
- **redis**: Redis for caching and queuing
- **ui-dev**: Node.js container for React development

### Observability Stack (for testing)
- **prometheus**: Metrics collection
- **grafana**: Visualization 
- **loki**: Log aggregation
- **tempo**: Distributed tracing
- **otel-collector**: OpenTelemetry collector

### Supporting Services
- **docs**: Documentation server (MkDocs)
- **nginx**: Reverse proxy (prod testing only)

## 🛠️ Development Workflows

### 1. Using VS Code Dev Containers

1. Open VS Code
2. Install "Remote - Containers" extension
3. Open the project folder
4. Click "Reopen in Container" when prompted
5. VS Code will build and connect to the dev container

### 2. Using Docker Compose Directly

```bash
# Start all services
make dev-up

# View logs
make dev-logs

# Enter development shell
make dev-shell

# Stop all services
make dev-down
```

### 3. Hot Reload Development

Both the operator and API support hot reload:

```bash
# Terminal 1: Operator hot reload
make dev-operator-watch

# Terminal 2: API hot reload  
make dev-api-watch

# Terminal 3: UI development (automatic)
# The UI container already runs with hot reload
```

## 🔧 Common Tasks

### Database Operations

```bash
# Connect to PostgreSQL
make db-shell

# Run migrations
make db-migrate

# Rollback migrations
make db-rollback
```

### Testing

```bash
# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run with coverage
make test-coverage
```

### Kubernetes Development

```bash
# Create local kind cluster
make kind-create

# Load images into kind
make kind-load

# Delete kind cluster
make kind-delete
```

## 📁 Project Structure

```
.devcontainer/       # VS Code dev container configuration
hack/               # Development scripts and configs
  db/              # Database scripts
  prometheus/      # Prometheus config
  grafana/         # Grafana provisioning
  loki/           # Loki config
  tempo/          # Tempo config
  otel/           # OpenTelemetry config
docker-compose.yml   # Development services
Dockerfile.dev      # Development container image
Makefile           # Development automation
```

## 🌐 Service URLs

When running, services are available at:

- **API Server**: http://localhost:8081
- **React UI**: http://localhost:3000
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001 (admin/admin)
- **Documentation**: http://localhost:8000

## 🔍 Troubleshooting

### Container won't start

```bash
# Check logs
docker-compose logs dev

# Rebuild containers
docker-compose build --no-cache

# Clean everything and start fresh
make dev-clean
make quickstart
```

### Permission issues

```bash
# Fix ownership
docker-compose exec dev chown -R developer:developer /workspace
```

### Port conflicts

```bash
# Check what's using ports
netstat -tlnp | grep -E ':(3000|8080|8081|9090)'

# Or use different ports in docker-compose.override.yml
```

## 🔐 Security Notes

- Development containers run as non-root user `developer`
- Secrets are not committed - use `.env.local` for local secrets
- Default passwords are for development only
- SSL/TLS is disabled in development mode

## 🚀 Next Steps

1. Review the [Development Guidelines](docs/development/README.md)
2. Check the [Architecture Overview](docs/architecture/README.md) 
3. Start with a simple task from the [issue tracker](https://github.com/gunjanjp/gunj-operator/issues)
4. Join our [Slack channel](https://gunj-operator.slack.com) for help

Happy coding! 🎉
