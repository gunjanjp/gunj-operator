#!/bin/bash
# Documentation generation script for Gunj Operator

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"
DOCS_GEN_DIR="$PROJECT_ROOT/docs-gen"
OUTPUT_DIR="$DOCS_GEN_DIR/output"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check required tools
check_requirements() {
    log_info "Checking requirements..."
    
    local requirements=(
        "go:Go compiler"
        "node:Node.js runtime"
        "npm:Node package manager"
        "python3:Python 3"
        "pip3:Python package manager"
    )
    
    for req in "${requirements[@]}"; do
        IFS=':' read -r cmd desc <<< "$req"
        if ! command -v "$cmd" &> /dev/null; then
            log_error "$desc ($cmd) is not installed"
            exit 1
        fi
    done
    
    log_info "All requirements satisfied"
}

# Install documentation tools
install_tools() {
    log_info "Installing documentation tools..."
    
    # Install MkDocs and plugins
    log_info "Installing MkDocs..."
    pip3 install --user mkdocs mkdocs-material mkdocs-minify-plugin \
        mkdocs-redirects mkdocs-git-revision-date-localized-plugin \
        mike pymdown-extensions
    
    # Install TypeDoc
    log_info "Installing TypeDoc..."
    cd "$PROJECT_ROOT/ui" && npm install --save-dev typedoc typedoc-plugin-markdown typedoc-plugin-missing-exports
    
    # Install godoc
    log_info "Installing godoc..."
    go install golang.org/x/tools/cmd/godoc@latest
    
    # Install Swagger UI
    log_info "Installing Swagger UI..."
    npm install -g @apidevtools/swagger-cli
    
    log_info "Documentation tools installed"
}

# Generate Go documentation
generate_go_docs() {
    log_info "Generating Go documentation..."
    
    mkdir -p "$OUTPUT_DIR/go"
    
    # Generate godoc HTML
    cd "$PROJECT_ROOT"
    godoc -http=:6060 &
    GODOC_PID=$!
    sleep 5
    
    # Download package documentation
    wget -r -np -k -E -p -erobots=off \
        --accept-regex="/pkg/github.com/gunjanjp/gunj-operator" \
        -P "$OUTPUT_DIR/go" \
        http://localhost:6060/pkg/github.com/gunjanjp/gunj-operator/
    
    kill $GODOC_PID
    
    log_info "Go documentation generated"
}

# Generate TypeScript documentation
generate_typescript_docs() {
    log_info "Generating TypeScript documentation..."
    
    cd "$DOCS_GEN_DIR"
    npx typedoc
    
    log_info "TypeScript documentation generated"
}

# Generate API documentation
generate_api_docs() {
    log_info "Generating API documentation..."
    
    mkdir -p "$OUTPUT_DIR/api"
    
    # Validate OpenAPI spec
    if [ -f "$PROJECT_ROOT/api/openapi.yaml" ]; then
        swagger-cli validate "$PROJECT_ROOT/api/openapi.yaml"
        
        # Generate HTML documentation
        npx @redocly/openapi-cli build-docs \
            "$PROJECT_ROOT/api/openapi.yaml" \
            -o "$OUTPUT_DIR/api/rest.html"
    else
        log_warn "OpenAPI spec not found"
    fi
    
    # Generate GraphQL documentation if schema exists
    if [ -f "$PROJECT_ROOT/api/schema.graphql" ]; then
        npx spectaql "$PROJECT_ROOT/api/schema.graphql" \
            -t "$OUTPUT_DIR/api/graphql"
    else
        log_warn "GraphQL schema not found"
    fi
    
    log_info "API documentation generated"
}

# Build MkDocs site
build_mkdocs() {
    log_info "Building MkDocs site..."
    
    cd "$DOCS_GEN_DIR"
    
    # Copy documentation files
    mkdir -p site_content
    cp -r "$PROJECT_ROOT/docs/"* site_content/
    
    # Build the site
    mkdocs build -d "$OUTPUT_DIR/site"
    
    log_info "MkDocs site built"
}

# Generate architecture diagrams
generate_diagrams() {
    log_info "Generating architecture diagrams..."
    
    mkdir -p "$OUTPUT_DIR/diagrams"
    
    # Generate PlantUML diagrams if any exist
    if command -v plantuml &> /dev/null; then
        find "$PROJECT_ROOT/docs" -name "*.puml" -exec \
            plantuml -tsvg -o "$OUTPUT_DIR/diagrams" {} \;
    else
        log_warn "PlantUML not installed, skipping diagram generation"
    fi
    
    log_info "Architecture diagrams generated"
}

# Create documentation index
create_index() {
    log_info "Creating documentation index..."
    
    cat > "$OUTPUT_DIR/index.html" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gunj Operator Documentation</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 { color: #2c3e50; }
        .section {
            background: #f4f4f4;
            padding: 20px;
            margin: 20px 0;
            border-radius: 8px;
        }
        .section h2 { margin-top: 0; }
        .links { display: flex; flex-wrap: wrap; gap: 10px; }
        .link {
            background: #3498db;
            color: white;
            padding: 10px 20px;
            text-decoration: none;
            border-radius: 5px;
            transition: background 0.3s;
        }
        .link:hover { background: #2980b9; }
    </style>
</head>
<body>
    <h1>Gunj Operator Documentation</h1>
    
    <div class="section">
        <h2>User Documentation</h2>
        <div class="links">
            <a href="site/index.html" class="link">User Guide</a>
            <a href="site/getting-started/index.html" class="link">Getting Started</a>
            <a href="site/api/index.html" class="link">API Reference</a>
        </div>
    </div>
    
    <div class="section">
        <h2>Developer Documentation</h2>
        <div class="links">
            <a href="go/pkg/github.com/gunjanjp/gunj-operator/index.html" class="link">Go Documentation</a>
            <a href="typescript/index.html" class="link">TypeScript Documentation</a>
            <a href="api/rest.html" class="link">REST API</a>
            <a href="api/graphql/index.html" class="link">GraphQL API</a>
        </div>
    </div>
    
    <div class="section">
        <h2>Architecture</h2>
        <div class="links">
            <a href="site/architecture/index.html" class="link">Architecture Overview</a>
            <a href="diagrams/index.html" class="link">Architecture Diagrams</a>
        </div>
    </div>
    
    <div class="section">
        <h2>External Links</h2>
        <div class="links">
            <a href="https://github.com/gunjanjp/gunj-operator" class="link">GitHub Repository</a>
            <a href="https://github.com/gunjanjp/gunj-operator/issues" class="link">Issue Tracker</a>
            <a href="https://github.com/gunjanjp/gunj-operator/discussions" class="link">Discussions</a>
        </div>
    </div>
</body>
</html>
EOF
    
    log_info "Documentation index created"
}

# Main execution
main() {
    log_info "Starting documentation generation..."
    
    # Check requirements
    check_requirements
    
    # Parse arguments
    case "${1:-all}" in
        install)
            install_tools
            ;;
        go)
            generate_go_docs
            ;;
        typescript)
            generate_typescript_docs
            ;;
        api)
            generate_api_docs
            ;;
        mkdocs)
            build_mkdocs
            ;;
        diagrams)
            generate_diagrams
            ;;
        all)
            # Clean output directory
            rm -rf "$OUTPUT_DIR"
            mkdir -p "$OUTPUT_DIR"
            
            # Generate all documentation
            generate_go_docs
            generate_typescript_docs
            generate_api_docs
            build_mkdocs
            generate_diagrams
            create_index
            
            log_info "All documentation generated successfully!"
            log_info "Output directory: $OUTPUT_DIR"
            ;;
        *)
            echo "Usage: $0 [install|go|typescript|api|mkdocs|diagrams|all]"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
