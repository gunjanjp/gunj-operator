#!/bin/bash
# Multi-arch Build Script for Gunj Operator
# Supports local development and testing of multi-architecture builds
# Version: 2.0

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
COMPONENTS=(operator api cli ui)
ARCHITECTURES=(linux/amd64 linux/arm64 linux/arm/v7)
REGISTRY="${REGISTRY:-docker.io}"
REGISTRY_USER="${REGISTRY_USER:-gunjanjp}"
VERSION="${VERSION:-dev}"
BUILD_TYPE="${BUILD_TYPE:-release}"
PUSH="${PUSH:-false}"
PARALLEL="${PARALLEL:-true}"
MAX_PARALLEL="${MAX_PARALLEL:-4}"

# Help function
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Multi-arch build script for Gunj Operator components

OPTIONS:
    -c, --components    Components to build (comma-separated: operator,api,cli,ui)
                       Default: all components
    -a, --architectures Architectures to build (comma-separated: amd64,arm64,arm/v7)
                       Default: all architectures
    -v, --version      Version tag to use
                       Default: dev
    -t, --type         Build type: release or debug
                       Default: release
    -p, --push         Push images to registry
                       Default: false
    -r, --registry     Container registry
                       Default: docker.io
    -u, --user         Registry username
                       Default: gunjanjp
    --no-parallel      Disable parallel builds
    -h, --help         Show this help message

EXAMPLES:
    # Build all components for all architectures
    $0

    # Build only operator for amd64 and arm64
    $0 -c operator -a amd64,arm64

    # Build and push release version
    $0 -v v2.0.0 -p

    # Debug build for local testing
    $0 -t debug -c operator -a amd64
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--components)
                IFS=',' read -ra COMPONENTS <<< "$2"
                shift 2
                ;;
            -a|--architectures)
                IFS=',' read -ra ARCH_INPUT <<< "$2"
                ARCHITECTURES=()
                for arch in "${ARCH_INPUT[@]}"; do
                    ARCHITECTURES+=("linux/$arch")
                done
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -t|--type)
                BUILD_TYPE="$2"
                shift 2
                ;;
            -p|--push)
                PUSH="true"
                shift
                ;;
            -r|--registry)
                REGISTRY="$2"
                shift 2
                ;;
            -u|--user)
                REGISTRY_USER="$2"
                shift 2
                ;;
            --no-parallel)
                PARALLEL="false"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                show_help
                exit 1
                ;;
        esac
    done
}

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"
    
    local missing=()
    
    # Check for required tools
    command -v docker >/dev/null 2>&1 || missing+=("docker")
    command -v go >/dev/null 2>&1 || missing+=("go")
    command -v npm >/dev/null 2>&1 || missing+=("npm")
    command -v make >/dev/null 2>&1 || missing+=("make")
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo -e "${RED}Missing required tools: ${missing[*]}${NC}"
        exit 1
    fi
    
    # Check Docker buildx
    if ! docker buildx version >/dev/null 2>&1; then
        echo -e "${YELLOW}Docker buildx not found. Installing...${NC}"
        docker buildx create --use --name gunj-builder
    fi
    
    # Enable experimental features if needed
    export DOCKER_CLI_EXPERIMENTAL=enabled
    
    echo -e "${GREEN}All prerequisites satisfied${NC}"
}

# Setup build environment
setup_environment() {
    echo -e "${BLUE}Setting up build environment...${NC}"
    
    # Create necessary directories
    mkdir -p dist/{operator,api,cli,ui}
    mkdir -p .cache/{go-mod,go-build,npm}
    
    # Set environment variables
    export GO111MODULE=on
    export GOCACHE="$(pwd)/.cache/go-build"
    export GOMODCACHE="$(pwd)/.cache/go-mod"
    export npm_config_cache="$(pwd)/.cache/npm"
    
    # Load build configuration
    if [[ -f .github/build-config.yml ]]; then
        echo -e "${GREEN}Build configuration loaded${NC}"
    else
        echo -e "${YELLOW}Warning: build-config.yml not found${NC}"
    fi
}

# Build Go component
build_go_component() {
    local component=$1
    local platform=$2
    local goos="${platform%%/*}"
    local goarch_full="${platform#*/}"
    local goarch="${goarch_full%/*}"
    local goarm=""
    
    # Handle ARM variants
    if [[ "$goarch_full" == "arm/v7" ]]; then
        goarch="arm"
        goarm="7"
    fi
    
    echo -e "${BLUE}Building $component for $platform...${NC}"
    
    # Set build environment
    export CGO_ENABLED=0
    export GOOS=$goos
    export GOARCH=$goarch
    [[ -n "$goarm" ]] && export GOARM=$goarm
    
    # Determine build flags based on type
    local build_flags="-a -trimpath"
    local ldflags="-w -s"
    
    if [[ "$BUILD_TYPE" == "release" ]]; then
        ldflags="$ldflags -X main.version=$VERSION -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    else
        build_flags=""
        ldflags="-X main.version=$VERSION-debug"
    fi
    
    # Component-specific settings
    local main_path=""
    local binary_name=""
    
    case $component in
        operator)
            main_path="./cmd/operator"
            binary_name="gunj-operator"
            ;;
        api)
            main_path="./cmd/api-server"
            binary_name="gunj-api-server"
            ;;
        cli)
            main_path="./cmd/cli"
            binary_name="gunj-cli"
            ;;
    esac
    
    # Build binary
    echo "Building binary..."
    go build $build_flags -ldflags "$ldflags" \
        -o "dist/$component/$platform/$binary_name" \
        "$main_path"
    
    # Build container image
    echo "Building container image..."
    docker buildx build \
        --platform "$platform" \
        --build-arg VERSION="$VERSION" \
        --build-arg TARGETPLATFORM="$platform" \
        --tag "$REGISTRY/$REGISTRY_USER/$component:$VERSION-${goarch}${goarm}" \
        --tag "$REGISTRY/$REGISTRY_USER/$component:latest-${goarch}${goarm}" \
        --cache-from "type=registry,ref=$REGISTRY/$REGISTRY_USER/$component:buildcache-${goarch}${goarm}" \
        --cache-to "type=registry,ref=$REGISTRY/$REGISTRY_USER/$component:buildcache-${goarch}${goarm},mode=max" \
        --file "Dockerfile.$component" \
        --load \
        .
    
    echo -e "${GREEN}Successfully built $component for $platform${NC}"
}

# Build UI component
build_ui_component() {
    local platform=$1
    
    echo -e "${BLUE}Building UI for $platform...${NC}"
    
    # Install dependencies if needed
    if [[ ! -d "ui/node_modules" ]]; then
        echo "Installing UI dependencies..."
        (cd ui && npm ci)
    fi
    
    # Build UI
    echo "Building UI assets..."
    (cd ui && REACT_APP_VERSION="$VERSION" npm run build)
    
    # Build container image
    echo "Building container image..."
    docker buildx build \
        --platform "$platform" \
        --build-arg VERSION="$VERSION" \
        --tag "$REGISTRY/$REGISTRY_USER/gunj-ui:$VERSION-${platform#*/}" \
        --tag "$REGISTRY/$REGISTRY_USER/gunj-ui:latest-${platform#*/}" \
        --file Dockerfile.ui \
        --load \
        .
    
    echo -e "${GREEN}Successfully built UI for $platform${NC}"
}

# Build component for all architectures
build_component() {
    local component=$1
    
    echo -e "${YELLOW}Building $component for architectures: ${ARCHITECTURES[*]}${NC}"
    
    if [[ "$PARALLEL" == "true" ]] && command -v parallel >/dev/null 2>&1; then
        # Use GNU parallel for concurrent builds
        export -f build_go_component build_ui_component
        export REGISTRY REGISTRY_USER VERSION BUILD_TYPE
        
        if [[ "$component" == "ui" ]]; then
            printf "%s\n" "${ARCHITECTURES[@]}" | \
                parallel -j "$MAX_PARALLEL" build_ui_component {}
        else
            printf "%s\n" "${ARCHITECTURES[@]}" | \
                parallel -j "$MAX_PARALLEL" build_go_component "$component" {}
        fi
    else
        # Sequential build
        for arch in "${ARCHITECTURES[@]}"; do
            if [[ "$component" == "ui" ]]; then
                build_ui_component "$arch"
            else
                build_go_component "$component" "$arch"
            fi
        done
    fi
}

# Create multi-arch manifest
create_manifest() {
    local component=$1
    
    echo -e "${BLUE}Creating multi-arch manifest for $component...${NC}"
    
    # Build manifest list
    local manifest_tags=()
    for arch in "${ARCHITECTURES[@]}"; do
        local arch_tag="${arch#*/}"
        arch_tag="${arch_tag//\//-}"  # Replace / with - for arm/v7
        manifest_tags+=("$REGISTRY/$REGISTRY_USER/$component:$VERSION-$arch_tag")
    done
    
    # Create and push manifest
    docker manifest create \
        "$REGISTRY/$REGISTRY_USER/$component:$VERSION" \
        "${manifest_tags[@]}"
    
    docker manifest create \
        "$REGISTRY/$REGISTRY_USER/$component:latest" \
        "${manifest_tags[@]}"
    
    if [[ "$PUSH" == "true" ]]; then
        echo "Pushing manifests..."
        docker manifest push "$REGISTRY/$REGISTRY_USER/$component:$VERSION"
        docker manifest push "$REGISTRY/$REGISTRY_USER/$component:latest"
    fi
    
    echo -e "${GREEN}Manifest created for $component${NC}"
}

# Main build process
main() {
    parse_args "$@"
    
    echo -e "${YELLOW}=== Gunj Operator Multi-arch Build ===${NC}"
    echo "Components: ${COMPONENTS[*]}"
    echo "Architectures: ${ARCHITECTURES[*]}"
    echo "Version: $VERSION"
    echo "Build Type: $BUILD_TYPE"
    echo "Registry: $REGISTRY/$REGISTRY_USER"
    echo "Push: $PUSH"
    echo ""
    
    check_prerequisites
    setup_environment
    
    # Build each component
    for component in "${COMPONENTS[@]}"; do
        echo -e "\n${YELLOW}=== Building $component ===${NC}"
        build_component "$component"
        create_manifest "$component"
    done
    
    # Summary
    echo -e "\n${GREEN}=== Build Complete ===${NC}"
    echo "Built components: ${COMPONENTS[*]}"
    echo "For architectures: ${ARCHITECTURES[*]}"
    
    if [[ "$PUSH" == "true" ]]; then
        echo -e "${GREEN}Images pushed to $REGISTRY/$REGISTRY_USER${NC}"
    else
        echo -e "${YELLOW}Images built locally. Use -p flag to push to registry${NC}"
    fi
}

# Run main function
main "$@"
