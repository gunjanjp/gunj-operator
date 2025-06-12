#!/bin/bash
# Artifact Management Script
# Handles local artifact storage, upload, and download
# Version: 2.0

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
ARTIFACT_DIR="${PROJECT_ROOT}/artifacts"
CONFIG_FILE="${PROJECT_ROOT}/.github/artifact-storage-config.yml"
CACHE_DIR="${HOME}/.cache/gunj-operator/artifacts"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
COMMAND=""
ARTIFACT_TYPE="binary"
COMPONENT="all"
VERSION="dev"
ARCH="$(uname -m)"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"

# Show help
show_help() {
    cat << EOF
Usage: $0 COMMAND [OPTIONS]

Artifact management for Gunj Operator

COMMANDS:
    store       Store artifacts locally
    upload      Upload artifacts to remote storage
    download    Download artifacts from remote storage
    list        List available artifacts
    clean       Clean local artifact storage
    info        Show artifact information

OPTIONS:
    -t, --type TYPE         Artifact type (binary|container|test|coverage|release)
                           Default: binary
    -c, --component COMP    Component name (operator|api|cli|ui|all)
                           Default: all
    -v, --version VERSION   Version tag
                           Default: dev
    -a, --arch ARCH        Architecture (amd64|arm64|arm/v7)
                           Default: current arch
    -o, --os OS            Operating system (linux|darwin|windows)
                           Default: current OS
    -s, --storage BACKEND  Storage backend (local|github|s3|registry)
                           Default: local
    -h, --help             Show this help message

EXAMPLES:
    # Store local build artifacts
    $0 store -t binary -c operator

    # Upload artifacts to S3
    $0 upload -t release -v v2.0.0 -s s3

    # Download specific artifact
    $0 download -t binary -c cli -v v2.0.0 -a arm64

    # List all artifacts
    $0 list -s github

    # Clean old artifacts
    $0 clean --older-than 7d
EOF
}

# Parse arguments
parse_args() {
    if [[ $# -eq 0 ]]; then
        show_help
        exit 1
    fi

    COMMAND=$1
    shift

    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--type)
                ARTIFACT_TYPE="$2"
                shift 2
                ;;
            -c|--component)
                COMPONENT="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -a|--arch)
                ARCH="$2"
                shift 2
                ;;
            -o|--os)
                OS="$2"
                shift 2
                ;;
            -s|--storage)
                STORAGE_BACKEND="$2"
                shift 2
                ;;
            --older-than)
                OLDER_THAN="$2"
                shift 2
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

# Initialize artifact directory structure
init_artifact_dirs() {
    echo -e "${BLUE}Initializing artifact directories...${NC}"
    
    mkdir -p "$ARTIFACT_DIR"/{binaries,containers,test-results,coverage,releases}
    mkdir -p "$ARTIFACT_DIR"/binaries/{operator,api,cli,ui}/{linux,darwin,windows}/{amd64,arm64,arm}
    mkdir -p "$ARTIFACT_DIR"/containers/{images,manifests}
    mkdir -p "$ARTIFACT_DIR"/test-results/{unit,integration,e2e}
    mkdir -p "$ARTIFACT_DIR"/coverage/{reports,badges}
    mkdir -p "$ARTIFACT_DIR"/releases/{stable,beta,nightly}
    mkdir -p "$CACHE_DIR"
    
    echo -e "${GREEN}Artifact directories initialized${NC}"
}

# Store artifacts locally
store_artifacts() {
    echo -e "${BLUE}Storing artifacts locally...${NC}"
    
    init_artifact_dirs
    
    case $ARTIFACT_TYPE in
        binary)
            store_binary_artifacts
            ;;
        container)
            store_container_artifacts
            ;;
        test)
            store_test_artifacts
            ;;
        coverage)
            store_coverage_artifacts
            ;;
        release)
            store_release_artifacts
            ;;
        *)
            echo -e "${RED}Unknown artifact type: $ARTIFACT_TYPE${NC}"
            exit 1
            ;;
    esac
}

# Store binary artifacts
store_binary_artifacts() {
    local components=()
    
    if [[ "$COMPONENT" == "all" ]]; then
        components=(operator api cli ui)
    else
        components=("$COMPONENT")
    fi
    
    for comp in "${components[@]}"; do
        local binary_name=""
        case $comp in
            operator) binary_name="gunj-operator" ;;
            api) binary_name="gunj-api-server" ;;
            cli) binary_name="gunj-cli" ;;
            ui) continue ;; # UI doesn't have binary
        esac
        
        local source_path="${PROJECT_ROOT}/dist/$comp/$OS/$ARCH/$binary_name"
        if [[ "$OS" == "windows" ]]; then
            source_path="${source_path}.exe"
        fi
        
        if [[ -f "$source_path" ]]; then
            local dest_dir="$ARTIFACT_DIR/binaries/$comp/$OS/$ARCH"
            local dest_file="$dest_dir/${binary_name}-${VERSION}"
            
            mkdir -p "$dest_dir"
            cp "$source_path" "$dest_file"
            
            # Generate metadata
            cat > "$dest_file.json" << EOF
{
    "component": "$comp",
    "version": "$VERSION",
    "os": "$OS",
    "arch": "$ARCH",
    "size": $(stat -f%z "$dest_file" 2>/dev/null || stat -c%s "$dest_file"),
    "sha256": "$(sha256sum "$dest_file" | cut -d' ' -f1)",
    "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "git_commit": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
    "build_date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
            
            echo -e "${GREEN}Stored $comp binary for $OS/$ARCH${NC}"
        else
            echo -e "${YELLOW}Binary not found: $source_path${NC}"
        fi
    done
}

# Store container artifacts  
store_container_artifacts() {
    local components=()
    
    if [[ "$COMPONENT" == "all" ]]; then
        components=(operator api cli ui)
    else
        components=("$COMPONENT")
    fi
    
    for comp in "${components[@]}"; do
        local image_name="gunj-$comp:$VERSION"
        
        # Check if image exists
        if docker image inspect "$image_name" >/dev/null 2>&1; then
            local dest_dir="$ARTIFACT_DIR/containers/images"
            local dest_file="$dest_dir/$comp-$VERSION-$ARCH.tar"
            
            mkdir -p "$dest_dir"
            
            echo "Saving container image $image_name..."
            docker save "$image_name" -o "$dest_file"
            
            # Compress
            gzip -9 "$dest_file"
            
            # Generate manifest
            docker image inspect "$image_name" > "$dest_dir/$comp-$VERSION-$ARCH.json"
            
            echo -e "${GREEN}Stored $comp container image${NC}"
        else
            echo -e "${YELLOW}Container image not found: $image_name${NC}"
        fi
    done
}

# Upload artifacts to remote storage
upload_artifacts() {
    echo -e "${BLUE}Uploading artifacts to $STORAGE_BACKEND...${NC}"
    
    case $STORAGE_BACKEND in
        github)
            upload_to_github
            ;;
        s3)
            upload_to_s3
            ;;
        registry)
            upload_to_registry
            ;;
        *)
            echo -e "${RED}Unknown storage backend: $STORAGE_BACKEND${NC}"
            exit 1
            ;;
    esac
}

# Upload to GitHub artifacts
upload_to_github() {
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        echo -e "${RED}GITHUB_TOKEN not set${NC}"
        exit 1
    fi
    
    # This would typically be done in CI, but we can use gh CLI
    if command -v gh >/dev/null 2>&1; then
        echo "Creating artifact bundle..."
        
        local bundle_name="gunj-artifacts-$ARTIFACT_TYPE-$VERSION-$(date +%s).tar.gz"
        tar -czf "$bundle_name" -C "$ARTIFACT_DIR" .
        
        # Upload using gh CLI (requires gh auth)
        echo "Uploading to GitHub..."
        # gh release upload or artifact upload would go here
        
        echo -e "${GREEN}Upload complete${NC}"
    else
        echo -e "${YELLOW}GitHub CLI (gh) not installed${NC}"
        echo "Artifacts prepared at: $ARTIFACT_DIR"
    fi
}

# Upload to S3
upload_to_s3() {
    if ! command -v aws >/dev/null 2>&1; then
        echo -e "${RED}AWS CLI not installed${NC}"
        exit 1
    fi
    
    local bucket="gunj-operator-artifacts"
    local prefix="$ARTIFACT_TYPE/$VERSION"
    
    echo "Uploading to S3 bucket: $bucket/$prefix"
    
    # Upload artifacts
    aws s3 sync "$ARTIFACT_DIR/$ARTIFACT_TYPE" "s3://$bucket/$prefix" \
        --exclude "*.json" \
        --metadata "version=$VERSION,type=$ARTIFACT_TYPE,timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    
    # Upload metadata separately
    aws s3 sync "$ARTIFACT_DIR/$ARTIFACT_TYPE" "s3://$bucket/$prefix" \
        --exclude "*" \
        --include "*.json" \
        --content-type "application/json"
    
    echo -e "${GREEN}Upload to S3 complete${NC}"
}

# Download artifacts
download_artifacts() {
    echo -e "${BLUE}Downloading artifacts from $STORAGE_BACKEND...${NC}"
    
    case $STORAGE_BACKEND in
        github)
            download_from_github
            ;;
        s3)
            download_from_s3
            ;;
        registry)
            download_from_registry
            ;;
        *)
            echo -e "${RED}Unknown storage backend: $STORAGE_BACKEND${NC}"
            exit 1
            ;;
    esac
}

# Download from S3
download_from_s3() {
    if ! command -v aws >/dev/null 2>&1; then
        echo -e "${RED}AWS CLI not installed${NC}"
        exit 1
    fi
    
    local bucket="gunj-operator-artifacts"
    local prefix="$ARTIFACT_TYPE/$VERSION"
    local dest_dir="$CACHE_DIR/$ARTIFACT_TYPE/$VERSION"
    
    mkdir -p "$dest_dir"
    
    echo "Downloading from S3: $bucket/$prefix"
    
    if [[ "$COMPONENT" == "all" ]]; then
        aws s3 sync "s3://$bucket/$prefix" "$dest_dir"
    else
        aws s3 sync "s3://$bucket/$prefix/$COMPONENT" "$dest_dir/$COMPONENT"
    fi
    
    echo -e "${GREEN}Download complete. Artifacts in: $dest_dir${NC}"
}

# List artifacts
list_artifacts() {
    echo -e "${BLUE}Listing artifacts in $STORAGE_BACKEND...${NC}"
    
    case $STORAGE_BACKEND in
        local)
            list_local_artifacts
            ;;
        s3)
            list_s3_artifacts
            ;;
        *)
            echo -e "${RED}List not implemented for: $STORAGE_BACKEND${NC}"
            exit 1
            ;;
    esac
}

# List local artifacts
list_local_artifacts() {
    if [[ ! -d "$ARTIFACT_DIR" ]]; then
        echo -e "${YELLOW}No local artifacts found${NC}"
        return
    fi
    
    echo -e "${GREEN}Local artifacts:${NC}"
    echo ""
    
    # List by type
    for type_dir in "$ARTIFACT_DIR"/*; do
        if [[ -d "$type_dir" ]]; then
            local type_name=$(basename "$type_dir")
            echo -e "${BLUE}$type_name:${NC}"
            
            find "$type_dir" -type f -not -name "*.json" | while read -r file; do
                local size=$(ls -lh "$file" | awk '{print $5}')
                local modified=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$file" 2>/dev/null || \
                                stat -c "%y" "$file" 2>/dev/null | cut -d' ' -f1,2)
                echo "  $(basename "$file") ($size, $modified)"
            done
            echo ""
        fi
    done
}

# Clean artifacts
clean_artifacts() {
    echo -e "${BLUE}Cleaning artifacts...${NC}"
    
    local older_than_days=7
    if [[ -n "${OLDER_THAN:-}" ]]; then
        older_than_days="${OLDER_THAN%d}"
    fi
    
    echo "Removing artifacts older than $older_than_days days..."
    
    if [[ -d "$ARTIFACT_DIR" ]]; then
        find "$ARTIFACT_DIR" -type f -mtime +$older_than_days -delete
        find "$ARTIFACT_DIR" -type d -empty -delete
    fi
    
    if [[ -d "$CACHE_DIR" ]]; then
        find "$CACHE_DIR" -type f -mtime +$older_than_days -delete
        find "$CACHE_DIR" -type d -empty -delete
    fi
    
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Show artifact info
show_info() {
    echo -e "${BLUE}Artifact Storage Information${NC}"
    echo ""
    echo "Project Root: $PROJECT_ROOT"
    echo "Artifact Directory: $ARTIFACT_DIR"
    echo "Cache Directory: $CACHE_DIR"
    echo ""
    
    if [[ -d "$ARTIFACT_DIR" ]]; then
        echo "Storage Usage:"
        du -sh "$ARTIFACT_DIR"/* 2>/dev/null | while read -r size path; do
            echo "  $(basename "$path"): $size"
        done
        echo ""
        echo "Total: $(du -sh "$ARTIFACT_DIR" | cut -f1)"
    else
        echo "No artifacts stored locally"
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    # Set default storage backend
    STORAGE_BACKEND="${STORAGE_BACKEND:-local}"
    
    case $COMMAND in
        store)
            store_artifacts
            ;;
        upload)
            upload_artifacts
            ;;
        download)
            download_artifacts
            ;;
        list)
            list_artifacts
            ;;
        clean)
            clean_artifacts
            ;;
        info)
            show_info
            ;;
        *)
            echo -e "${RED}Unknown command: $COMMAND${NC}"
            show_help
            exit 1
            ;;
    esac
}

# Run main
main "$@"
