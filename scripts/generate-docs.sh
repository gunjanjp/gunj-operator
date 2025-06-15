#!/bin/bash
# Documentation generation script for Gunj Operator

set -e

echo "📚 Generating documentation for Gunj Operator..."

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to report info
info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Function to report success
success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function to report error
error() {
    echo -e "${RED}❌ ERROR: $1${NC}"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    echo "Checking prerequisites..."
    
    # Check for Go
    if ! command -v go &> /dev/null; then
        error "Go is not installed"
    fi
    
    # Check for controller-gen
    if ! command -v controller-gen &> /dev/null; then
        info "Installing controller-gen..."
        go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
    fi
    
    # Check for gen-crd-api-reference-docs
    if ! command -v gen-crd-api-reference-docs &> /dev/null; then
        info "Installing gen-crd-api-reference-docs..."
        go install github.com/ahmetb/gen-crd-api-reference-docs@latest
    fi
    
    success "Prerequisites checked"
}

# Generate CRD documentation
generate_crd_docs() {
    echo -e "\n📋 Generating CRD documentation..."
    
    # Generate CRD YAML files
    info "Generating CRD manifests..."
    controller-gen crd:maxDescLen=0 paths="./api/..." output:crd:artifacts:config=config/crd/bases
    
    # Generate CRD reference documentation
    info "Generating CRD reference docs..."
    gen-crd-api-reference-docs \
        -api-dir=./api/v1beta1 \
        -config=./docs/api/crd-gen-config.json \
        -template-dir=./docs/api/templates \
        -out-file=./docs/api/crd-reference.md
    
    success "CRD documentation generated"
}

# Generate Go API documentation
generate_go_docs() {
    echo -e "\n🐹 Generating Go API documentation..."
    
    # Create temporary directory for godoc
    local temp_dir=$(mktemp -d)
    
    # Generate godoc HTML
    info "Generating Go documentation..."
    godoc -http=:6060 &
    local godoc_pid=$!
    
    # Wait for godoc to start
    sleep 3
    
    # Download the documentation
    wget -r -np -k -E -p -erobots=off \
        --convert-links \
        --domains localhost \
        --no-host-directories \
        --directory-prefix="$temp_dir" \
        http://localhost:6060/pkg/github.com/gunjanjp/gunj-operator/ || true
    
    # Kill godoc server
    kill $godoc_pid
    
    # Convert to markdown
    info "Converting to Markdown..."
    find "$temp_dir" -name "*.html" -type f | while read -r file; do
        local md_file="${file%.html}.md"
        pandoc -f html -t markdown "$file" -o "$md_file" || true
    done
    
    # Copy to documentation directory
    mkdir -p docs/api/go-api
    find "$temp_dir" -name "*.md" -type f -exec cp {} docs/api/go-api/ \;
    
    # Cleanup
    rm -rf "$temp_dir"
    
    success "Go API documentation generated"
}

# Generate OpenAPI specification
generate_openapi() {
    echo -e "\n📜 Generating OpenAPI specification..."
    
    info "Extracting OpenAPI annotations..."
    
    # Run the API server with docs generation flag
    go run cmd/api-server/main.go --generate-docs --output=docs/api/openapi.yaml || true
    
    # Generate HTML documentation from OpenAPI
    if command -v redoc-cli &> /dev/null; then
        info "Generating HTML documentation..."
        redoc-cli bundle docs/api/openapi.yaml -o docs/api/api-reference.html
    else
        info "Install redoc-cli for HTML generation: npm install -g redoc-cli"
    fi
    
    success "OpenAPI documentation generated"
}

# Generate metrics documentation
generate_metrics_docs() {
    echo -e "\n📊 Generating metrics documentation..."
    
    cat > docs/reference/metrics.md << 'EOF'
# Metrics Reference

This document lists all metrics exposed by the Gunj Operator.

## Operator Metrics

### Controller Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `gunj_operator_reconcile_total` | Counter | Total number of reconciliations | `controller`, `result` |
| `gunj_operator_reconcile_duration_seconds` | Histogram | Time taken for reconciliation | `controller` |
| `gunj_operator_reconcile_errors_total` | Counter | Total number of reconciliation errors | `controller`, `error` |
| `gunj_operator_platforms_total` | Gauge | Total number of platforms | `phase` |

### API Server Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `gunj_api_requests_total` | Counter | Total API requests | `method`, `endpoint`, `status` |
| `gunj_api_request_duration_seconds` | Histogram | API request duration | `method`, `endpoint` |
| `gunj_api_active_connections` | Gauge | Active API connections | `type` |

### Component Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `gunj_component_status` | Gauge | Component status (1=ready, 0=not ready) | `platform`, `component` |
| `gunj_component_restarts_total` | Counter | Component restart count | `platform`, `component` |
| `gunj_component_resource_usage` | Gauge | Resource usage | `platform`, `component`, `resource` |

## Platform Metrics

These metrics are collected from managed platforms:

### Prometheus Metrics
- All standard Prometheus metrics
- Custom application metrics

### Grafana Metrics
- Dashboard load times
- Active users
- Query performance

### Loki Metrics
- Log ingestion rate
- Query performance
- Storage usage

### Tempo Metrics
- Trace ingestion rate
- Query performance
- Storage usage

## Usage Examples

### Prometheus Query Examples

```promql
# Reconciliation error rate
rate(gunj_operator_reconcile_errors_total[5m])

# Average reconciliation duration
histogram_quantile(0.95, gunj_operator_reconcile_duration_seconds)

# Platforms by phase
gunj_operator_platforms_total
```

### Grafana Dashboard

Import the provided dashboard from `dashboards/gunj-operator.json` for a complete view of all metrics.

## Alerting Rules

See [Alerting Configuration](../user-guide/alerting.md) for pre-configured alerting rules based on these metrics.
EOF
    
    success "Metrics documentation generated"
}

# Generate CLI documentation
generate_cli_docs() {
    echo -e "\n🖥️  Generating CLI documentation..."
    
    info "Generating CLI reference..."
    
    # Build the CLI tool
    go build -o bin/gunj-cli ./cmd/cli
    
    # Generate documentation
    ./bin/gunj-cli docs markdown --dir docs/reference/cli/
    
    # Create main CLI reference
    cat > docs/reference/cli.md << 'EOF'
# CLI Reference

The Gunj CLI (`gunj`) provides command-line access to manage observability platforms.

## Installation

```bash
# Download the latest release
curl -LO https://github.com/gunjanjp/gunj-operator/releases/latest/download/gunj-cli
chmod +x gunj-cli
sudo mv gunj-cli /usr/local/bin/gunj

# Verify installation
gunj version
```

## Configuration

The CLI reads configuration from:
1. `$HOME/.gunj/config.yaml`
2. Environment variables (prefix: `GUNJ_`)
3. Command-line flags

### Configuration File

```yaml
# ~/.gunj/config.yaml
context: production
contexts:
  production:
    server: https://api.gunj-operator.io
    token: ${GUNJ_TOKEN}
  development:
    server: http://localhost:8080
    token: dev-token
```

## Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--context` | Context to use | `default` |
| `--output, -o` | Output format (json, yaml, table) | `table` |
| `--namespace, -n` | Kubernetes namespace | `default` |
| `--verbose, -v` | Verbose output | `false` |

## Commands

See the following pages for detailed command documentation:
- [gunj platform](cli/gunj_platform.md) - Manage platforms
- [gunj component](cli/gunj_component.md) - Manage components
- [gunj backup](cli/gunj_backup.md) - Backup operations
- [gunj config](cli/gunj_config.md) - Manage CLI configuration

## Examples

### Platform Management

```bash
# List all platforms
gunj platform list

# Create a new platform
gunj platform create my-platform -f platform.yaml

# Get platform details
gunj platform get my-platform -o yaml

# Delete a platform
gunj platform delete my-platform
```

### Component Operations

```bash
# Scale Prometheus
gunj component scale prometheus --platform my-platform --replicas 3

# Update component version
gunj component update grafana --platform my-platform --version 10.2.0

# Get component logs
gunj component logs prometheus --platform my-platform --follow
```

### Backup and Restore

```bash
# Create backup
gunj backup create --platform my-platform --destination s3://backups/

# List backups
gunj backup list --platform my-platform

# Restore from backup
gunj backup restore --platform my-platform --backup backup-20250612
```

## Shell Completion

Enable shell completion for better CLI experience:

```bash
# Bash
gunj completion bash > /etc/bash_completion.d/gunj

# Zsh
gunj completion zsh > "${fpath[1]}/_gunj"

# Fish
gunj completion fish > ~/.config/fish/completions/gunj.fish
```
EOF
    
    success "CLI documentation generated"
}

# Generate architecture diagrams
generate_diagrams() {
    echo -e "\n🎨 Generating architecture diagrams..."
    
    # Create diagrams directory
    mkdir -p docs/images/diagrams
    
    # Generate PlantUML diagrams if available
    if command -v plantuml &> /dev/null; then
        info "Generating PlantUML diagrams..."
        find docs -name "*.puml" -type f | while read -r file; do
            plantuml -tpng "$file" -o "$(dirname "$file")/../images/diagrams/"
        done
    else
        info "Install PlantUML for diagram generation"
    fi
    
    # Generate Mermaid diagrams if mermaid-cli is available
    if command -v mmdc &> /dev/null; then
        info "Generating Mermaid diagrams..."
        find docs -name "*.mmd" -type f | while read -r file; do
            mmdc -i "$file" -o "${file%.mmd}.png" -t dark -b transparent
        done
    else
        info "Install @mermaid-js/mermaid-cli for Mermaid diagram generation"
    fi
    
    success "Diagrams generated"
}

# Update documentation index
update_index() {
    echo -e "\n📑 Updating documentation index..."
    
    # Generate table of contents
    info "Generating table of contents..."
    
    # Create index file
    cat > docs/index.md << 'EOF'
# Gunj Operator Documentation

<p align="center">
  <img src="assets/logo.png" alt="Gunj Operator Logo" width="200">
</p>

<p align="center">
  <strong>Enterprise Observability Platform for Kubernetes</strong>
</p>

<p align="center">
  <a href="https://github.com/gunjanjp/gunj-operator/releases">
    <img src="https://img.shields.io/github/v/release/gunjanjp/gunj-operator" alt="Release">
  </a>
  <a href="https://github.com/gunjanjp/gunj-operator/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/gunjanjp/gunj-operator" alt="License">
  </a>
  <a href="https://gunjanjp.slack.com">
    <img src="https://img.shields.io/badge/slack-join-brightgreen" alt="Slack">
  </a>
</p>

## Welcome

The Gunj Operator simplifies the deployment and management of observability platforms in Kubernetes environments. It provides a unified way to deploy and configure Prometheus, Grafana, Loki, and Tempo.

## Quick Links

<div class="grid cards" markdown>

- :material-rocket-launch: **[Quick Start](getting-started/quick-start.md)**  
  Get up and running in 5 minutes

- :material-book-open-variant: **[User Guide](user-guide/index.md)**  
  Complete guide for operators

- :material-api: **[API Reference](api/index.md)**  
  REST and GraphQL API docs

- :material-school: **[Tutorials](tutorials/index.md)**  
  Step-by-step learning guides

</div>

## Features

- ✨ **Kubernetes Native** - Built using the operator pattern
- 🚀 **Easy Deployment** - Single CRD to deploy entire observability stack
- 🔧 **Highly Configurable** - Customize every aspect of your platform
- 📊 **Multi-Component** - Prometheus, Grafana, Loki, and Tempo
- 🔄 **GitOps Ready** - Declarative configuration management
- 🌐 **Multi-Cluster** - Manage platforms across clusters
- 🔒 **Secure by Default** - RBAC, TLS, and authentication
- 📈 **Production Ready** - HA, backups, and monitoring

## Latest Updates

See [What's New](changelog.md) for the latest features and improvements.

## Get Started

1. [Install the operator](getting-started/installation.md)
2. [Create your first platform](getting-started/first-platform.md)
3. [Explore the UI](user-guide/ui-overview.md)
4. [Configure components](user-guide/configuration.md)

## Community

- 💬 [Slack Channel](https://gunjanjp.slack.com)
- 🐛 [Issue Tracker](https://github.com/gunjanjp/gunj-operator/issues)
- 💡 [Discussions](https://github.com/gunjanjp/gunj-operator/discussions)
- 📧 [Mailing List](mailto:gunj-operator@googlegroups.com)

## Contributing

We welcome contributions! See our [Contributing Guide](development/contributing.md) to get started.

## License

Gunj Operator is licensed under the [MIT License](https://github.com/gunjanjp/gunj-operator/blob/main/LICENSE).
EOF
    
    success "Documentation index updated"
}

# Build documentation site
build_docs_site() {
    echo -e "\n🏗️  Building documentation site..."
    
    if command -v mkdocs &> /dev/null; then
        info "Building with MkDocs..."
        mkdocs build --clean --strict
        success "Documentation site built in 'site/' directory"
    else
        info "Install MkDocs to build the documentation site: pip install mkdocs-material"
    fi
}

# Main execution
main() {
    echo "📚 Gunj Operator Documentation Generation"
    echo "========================================"
    
    check_prerequisites
    generate_crd_docs
    generate_go_docs
    generate_openapi
    generate_metrics_docs
    generate_cli_docs
    generate_diagrams
    update_index
    build_docs_site
    
    echo -e "\n"
    success "Documentation generation complete!"
    
    echo -e "\nNext steps:"
    echo "  - Review generated documentation in docs/"
    echo "  - Run 'mkdocs serve' to preview the site"
    echo "  - Run './scripts/validate-docs.sh' to validate"
}

# Run main function
main
