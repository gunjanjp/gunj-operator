#!/bin/bash
# test-examples.sh - Deploy and test example configurations in a kind cluster
# This script creates a local Kubernetes cluster and tests the example manifests

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CLUSTER_NAME="gunj-operator-test"
NAMESPACE="test-monitoring"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    local missing_tools=()
    
    for tool in kind kubectl helm; do
        if ! command -v $tool &> /dev/null; then
            missing_tools+=($tool)
        fi
    done
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        echo "Please install the missing tools and try again."
        exit 1
    fi
    
    print_success "All prerequisites satisfied"
}

# Create kind cluster
create_cluster() {
    print_info "Creating kind cluster '$CLUSTER_NAME'..."
    
    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        print_warning "Cluster '$CLUSTER_NAME' already exists. Deleting..."
        kind delete cluster --name "$CLUSTER_NAME"
    fi
    
    # Create cluster with extra mounts for persistent storage
    cat <<EOF | kind create cluster --name "$CLUSTER_NAME" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
  - hostPath: /tmp/gunj-operator-test
    containerPath: /data
- role: worker
  extraMounts:
  - hostPath: /tmp/gunj-operator-test
    containerPath: /data
- role: worker
  extraMounts:
  - hostPath: /tmp/gunj-operator-test
    containerPath: /data
EOF
    
    print_success "Cluster created successfully"
}

# Install the operator (mock for testing)
install_operator() {
    print_info "Installing Gunj Operator..."
    
    # For now, we'll just create the CRD
    # In real scenario, this would install the full operator
    kubectl apply -f - <<EOF
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: observabilityplatforms.observability.io
spec:
  group: observability.io
  versions:
  - name: v1beta1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
          status:
            type: object
    additionalPrinterColumns:
    - name: Phase
      type: string
      jsonPath: .status.phase
    - name: Age
      type: date
      jsonPath: .metadata.creationTimestamp
  scope: Namespaced
  names:
    plural: observabilityplatforms
    singular: observabilityplatform
    kind: ObservabilityPlatform
    shortNames:
    - op
EOF
    
    print_success "Operator installed"
}

# Test an example manifest
test_example() {
    local example_file=$1
    local test_name=$(basename "$example_file" .yaml)
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print_info "Testing: $test_name"
    echo "File: $example_file"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Create namespace
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply the example
    print_info "Applying manifest..."
    if kubectl apply -f "$example_file" -n "$NAMESPACE"; then
        print_success "Manifest applied successfully"
        
        # Get the resource
        print_info "Checking resource status..."
        kubectl get observabilityplatform -n "$NAMESPACE" -o wide
        
        # Get detailed info
        kubectl describe observabilityplatform -n "$NAMESPACE"
        
        # Clean up
        print_info "Cleaning up..."
        kubectl delete -f "$example_file" -n "$NAMESPACE" --wait=false
        
        print_success "Test completed for $test_name"
    else
        print_error "Failed to apply manifest"
        return 1
    fi
}

# Main test function
run_tests() {
    local test_filter=${1:-""}
    
    print_info "Running tests..."
    
    # Test specific example or all
    if [ -n "$test_filter" ]; then
        if [ -f "$test_filter" ]; then
            test_example "$test_filter"
        else
            print_error "File not found: $test_filter"
            exit 1
        fi
    else
        # Test all examples in order of complexity
        local test_order=(
            "$SCRIPT_DIR/basic/minimal-platform.yaml"
            "$SCRIPT_DIR/basic/dev-platform.yaml"
            "$SCRIPT_DIR/production/production-ha-platform.yaml"
            "$SCRIPT_DIR/multi-tenant/multi-tenant-platform.yaml"
            "$SCRIPT_DIR/advanced/gitops-platform.yaml"
            "$SCRIPT_DIR/advanced/cost-optimized-platform.yaml"
            "$SCRIPT_DIR/advanced/disaster-recovery-platform.yaml"
            "$SCRIPT_DIR/advanced/airgapped-platform.yaml"
        )
        
        local passed=0
        local failed=0
        
        for example in "${test_order[@]}"; do
            if [ -f "$example" ]; then
                if test_example "$example"; then
                    ((passed++))
                else
                    ((failed++))
                fi
            else
                print_warning "Example not found: $example"
            fi
        done
        
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "Test Summary:"
        echo "  Passed: $passed"
        echo "  Failed: $failed"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        if [ $failed -gt 0 ]; then
            print_error "Some tests failed"
            exit 1
        else
            print_success "All tests passed!"
        fi
    fi
}

# Cleanup function
cleanup() {
    print_info "Cleaning up..."
    
    # Delete namespace
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
    
    # Delete cluster
    if [ "${KEEP_CLUSTER:-false}" != "true" ]; then
        kind delete cluster --name "$CLUSTER_NAME"
        print_success "Cluster deleted"
    else
        print_info "Cluster kept for debugging. Delete with: kind delete cluster --name $CLUSTER_NAME"
    fi
}

# Main execution
main() {
    echo "🧪 Gunj Operator Example Test Suite"
    echo "=================================="
    
    # Parse arguments
    local test_file=""
    while [[ $# -gt 0 ]]; do
        case $1 in
            --keep-cluster)
                export KEEP_CLUSTER=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [example-file] [--keep-cluster]"
                echo ""
                echo "Options:"
                echo "  example-file    Test a specific example file"
                echo "  --keep-cluster  Keep the kind cluster after tests"
                echo ""
                echo "Examples:"
                echo "  $0                                    # Test all examples"
                echo "  $0 basic/minimal-platform.yaml        # Test specific example"
                echo "  $0 --keep-cluster                     # Keep cluster for debugging"
                exit 0
                ;;
            *)
                test_file=$1
                shift
                ;;
        esac
    done
    
    # Set up trap for cleanup
    trap cleanup EXIT
    
    # Run the test workflow
    check_prerequisites
    create_cluster
    
    # Wait for cluster to be ready
    print_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=ready node --all --timeout=60s
    
    install_operator
    run_tests "$test_file"
}

# Run main function
main "$@"
