#!/bin/bash

# Documentation Generation Script
# This script generates various documentation artifacts for the Gunj Operator

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DOCS_DIR="${PROJECT_ROOT}/docs"
OUTPUT_DIR="${DOCS_DIR}/generated"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to generate CRD documentation
generate_crd_docs() {
    log_info "Generating CRD documentation..."
    
    if command_exists crd-ref-docs; then
        crd-ref-docs \
            --source-path="${PROJECT_ROOT}/api" \
            --config="${DOCS_DIR}/crd-ref-docs.yaml" \
            --renderer=markdown \
            --output-path="${OUTPUT_DIR}/crd-reference.md"
        log_info "CRD documentation generated successfully"
    else
        log_warn "crd-ref-docs not found. Install with: go install github.com/elastic/crd-ref-docs@latest"
    fi
}

# Function to generate Go documentation
generate_go_docs() {
    log_info "Generating Go documentation..."
    
    if command_exists godoc; then
        # Start godoc server in background
        godoc -http=:6060 &
        GODOC_PID=$!
        
        # Wait for server to start
        sleep 3
        
        # Generate static HTML
        wget -r -np -k -E -p -erobots=off \
            --convert-links \
            --no-host-directories \
            --directory-prefix="${OUTPUT_DIR}/godoc" \
            "http://localhost:6060/pkg/github.com/gunjanjp/gunj-operator/"
        
        # Stop godoc server
        kill $GODOC_PID
        
        log_info "Go documentation generated successfully"
    else
        log_warn "godoc not found. Install with: go install golang.org/x/tools/cmd/godoc@latest"
    fi
}

# Function to generate OpenAPI documentation
generate_openapi_docs() {
    log_info "Generating OpenAPI documentation..."
    
    if [ -f "${PROJECT_ROOT}/api/openapi.yaml" ]; then
        if command_exists swagger-codegen; then
            swagger-codegen generate \
                -i "${PROJECT_ROOT}/api/openapi.yaml" \
                -l html2 \
                -o "${OUTPUT_DIR}/api-docs"
            log_info "OpenAPI documentation generated successfully"
        else
            log_warn "swagger-codegen not found. Install from: https://swagger.io/tools/swagger-codegen/"
        fi
    else
        log_warn "OpenAPI specification not found at api/openapi.yaml"
    fi
}

# Function to generate GraphQL documentation
generate_graphql_docs() {
    log_info "Generating GraphQL documentation..."
    
    if [ -f "${PROJECT_ROOT}/api/schema.graphql" ]; then
        if command_exists spectaql; then
            spectaql "${DOCS_DIR}/spectaql-config.yaml" \
                --target-dir "${OUTPUT_DIR}/graphql-docs"
            log_info "GraphQL documentation generated successfully"
        else
            log_warn "spectaql not found. Install with: npm install -g spectaql"
        fi
    else
        log_warn "GraphQL schema not found at api/schema.graphql"
    fi
}

# Function to generate metrics documentation
generate_metrics_docs() {
    log_info "Generating metrics documentation..."
    
    # Extract metrics from Go source files
    grep -r "prometheus.NewDesc\|prometheus.NewCounterVec\|prometheus.NewGaugeVec\|prometheus.NewHistogramVec" \
        "${PROJECT_ROOT}/controllers" \
        "${PROJECT_ROOT}/internal" \
        --include="*.go" | \
        awk -F'"' '{print "- `" $2 "`: " $4}' | \
        sort -u > "${OUTPUT_DIR}/metrics.md"
    
    # Add header
    cat > "${OUTPUT_DIR}/metrics.md.tmp" << EOF
# Gunj Operator Metrics

This document lists all metrics exposed by the Gunj Operator.

## Operator Metrics

EOF
    
    cat "${OUTPUT_DIR}/metrics.md" >> "${OUTPUT_DIR}/metrics.md.tmp"
    mv "${OUTPUT_DIR}/metrics.md.tmp" "${OUTPUT_DIR}/metrics.md"
    
    log_info "Metrics documentation generated successfully"
}

# Function to generate CLI documentation
generate_cli_docs() {
    log_info "Generating CLI documentation..."
    
    if [ -f "${PROJECT_ROOT}/cmd/cli/main.go" ]; then
        # Build the CLI first
        go build -o /tmp/gunj-cli "${PROJECT_ROOT}/cmd/cli/main.go"
        
        # Generate documentation
        /tmp/gunj-cli docs markdown --dir "${OUTPUT_DIR}/cli"
        
        # Clean up
        rm /tmp/gunj-cli
        
        log_info "CLI documentation generated successfully"
    else
        log_warn "CLI not found at cmd/cli/main.go"
    fi
}

# Function to generate architecture diagrams
generate_diagrams() {
    log_info "Generating architecture diagrams..."
    
    if command_exists mmdc; then
        # Find all mermaid diagram files
        find "${DOCS_DIR}" -name "*.mmd" -type f | while read -r diagram; do
            output_file="${OUTPUT_DIR}/diagrams/$(basename "${diagram%.mmd}.png")"
            mkdir -p "$(dirname "$output_file")"
            
            mmdc -i "$diagram" -o "$output_file" -t dark -b transparent
            log_info "Generated diagram: $(basename "$output_file")"
        done
    else
        log_warn "mmdc (mermaid-cli) not found. Install with: npm install -g @mermaid-js/mermaid-cli"
    fi
}

# Function to generate example configurations
generate_examples() {
    log_info "Generating example configurations..."
    
    mkdir -p "${OUTPUT_DIR}/examples"
    
    # Copy and process example files
    find "${PROJECT_ROOT}/examples" -name "*.yaml" -type f | while read -r example; do
        relative_path="${example#${PROJECT_ROOT}/examples/}"
        output_file="${OUTPUT_DIR}/examples/${relative_path}"
        mkdir -p "$(dirname "$output_file")"
        
        # Add header with description
        cat > "$output_file" << EOF
# This is an auto-generated example configuration
# Source: examples/${relative_path}
# Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

EOF
        cat "$example" >> "$output_file"
    done
    
    log_info "Example configurations generated successfully"
}

# Function to validate links in documentation
validate_links() {
    log_info "Validating documentation links..."
    
    if command_exists markdown-link-check; then
        find "${DOCS_DIR}" -name "*.md" -type f | while read -r file; do
            log_info "Checking links in $(basename "$file")..."
            markdown-link-check "$file" || true
        done
    else
        log_warn "markdown-link-check not found. Install with: npm install -g markdown-link-check"
    fi
}

# Function to build the documentation site
build_docs_site() {
    log_info "Building documentation site..."
    
    cd "${DOCS_DIR}"
    
    if [ -f "mkdocs.yml" ] && command_exists mkdocs; then
        log_info "Building with MkDocs..."
        mkdocs build
    elif [ -f "config.toml" ] && command_exists hugo; then
        log_info "Building with Hugo..."
        hugo
    else
        log_warn "No documentation site generator found (MkDocs or Hugo)"
    fi
    
    cd "${PROJECT_ROOT}"
}

# Main execution
main() {
    log_info "Starting documentation generation..."
    
    # Check for required tools
    if ! command_exists go; then
        log_error "Go is required but not installed"
        exit 1
    fi
    
    # Generate different documentation types
    generate_crd_docs
    generate_go_docs
    generate_openapi_docs
    generate_graphql_docs
    generate_metrics_docs
    generate_cli_docs
    generate_diagrams
    generate_examples
    
    # Validate documentation
    validate_links
    
    # Build documentation site
    build_docs_site
    
    log_info "Documentation generation completed!"
    log_info "Generated documentation can be found in: ${OUTPUT_DIR}"
}

# Run main function
main "$@"
