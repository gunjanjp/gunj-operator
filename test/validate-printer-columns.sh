#!/bin/bash
# Test script for validating printer columns implementation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=== Testing Printer Columns for ObservabilityPlatform CRD ==="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    
    case $status in
        "success")
            echo -e "${GREEN}✓${NC} $message"
            ;;
        "error")
            echo -e "${RED}✗${NC} $message"
            ;;
        "info")
            echo -e "${YELLOW}→${NC} $message"
            ;;
    esac
}

# Function to check if a printer column exists in the CRD
check_printer_column() {
    local version=$1
    local column_name=$2
    local crd_file="${PROJECT_ROOT}/config/crd/bases/observability.io_observabilityplatforms.yaml"
    
    if grep -q "name: $column_name" "$crd_file"; then
        print_status "success" "Column '$column_name' found in $version"
        return 0
    else
        print_status "error" "Column '$column_name' NOT found in $version"
        return 1
    fi
}

# Function to validate printer column attributes
validate_column_attributes() {
    local version=$1
    local expected_columns=("${!2}")
    
    echo
    print_status "info" "Validating printer columns for $version"
    
    local failed=0
    for column in "${expected_columns[@]}"; do
        if ! check_printer_column "$version" "$column"; then
            ((failed++))
        fi
    done
    
    if [ $failed -eq 0 ]; then
        print_status "success" "All printer columns validated for $version"
    else
        print_status "error" "$failed printer columns missing for $version"
        return 1
    fi
}

# Expected columns for v1beta1
v1beta1_columns=(
    "Phase"
    "Health"
    "Components"
    "Version"
    "Prometheus"
    "Grafana"
    "Loki"
    "Tempo"
    "Message"
    "Age"
)

# Expected columns for v1alpha1
v1alpha1_columns=(
    "Phase"
    "Ready"
    "Prometheus"
    "Grafana"
    "Message"
    "Age"
)

# Main test execution
main() {
    print_status "info" "Starting printer columns validation"
    echo
    
    # Check if controller-gen is available
    if ! command -v controller-gen &> /dev/null; then
        print_status "error" "controller-gen not found. Please install it first."
        echo "Run: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest"
        exit 1
    fi
    
    # Generate CRDs
    print_status "info" "Generating CRDs with controller-gen"
    cd "$PROJECT_ROOT"
    
    # Create config directory if it doesn't exist
    mkdir -p config/crd/bases
    
    # Generate CRDs
    controller-gen crd paths="./api/..." output:crd:artifacts:config=config/crd/bases
    
    if [ $? -eq 0 ]; then
        print_status "success" "CRDs generated successfully"
    else
        print_status "error" "Failed to generate CRDs"
        exit 1
    fi
    
    # Check if CRD file exists
    local crd_file="${PROJECT_ROOT}/config/crd/bases/observability.io_observabilityplatforms.yaml"
    if [ ! -f "$crd_file" ]; then
        print_status "error" "CRD file not found: $crd_file"
        exit 1
    fi
    
    print_status "success" "CRD file found: $crd_file"
    
    # Validate v1beta1 columns
    validate_column_attributes "v1beta1" v1beta1_columns[@]
    
    # Validate v1alpha1 columns
    validate_column_attributes "v1alpha1" v1alpha1_columns[@]
    
    # Check for categories
    echo
    print_status "info" "Checking resource categories"
    if grep -q "categories:" "$crd_file"; then
        print_status "success" "Resource categories found"
        grep -A2 "categories:" "$crd_file" | sed 's/^/    /'
    else
        print_status "error" "Resource categories NOT found"
    fi
    
    # Check for shortNames
    echo
    print_status "info" "Checking resource shortNames"
    if grep -q "shortNames:" "$crd_file"; then
        print_status "success" "Resource shortNames found"
        grep -A2 "shortNames:" "$crd_file" | sed 's/^/    /'
    else
        print_status "error" "Resource shortNames NOT found"
    fi
    
    # Display sample kubectl output format
    echo
    print_status "info" "Sample kubectl output format:"
    cat << EOF

Standard output:
kubectl get observabilityplatforms
NAME                 PHASE        HEALTH     AGE
production-full      Ready        Healthy    5d
staging-installing   Installing   Unknown    2m

Wide output:
kubectl get observabilityplatforms -o wide
NAME                 PHASE        HEALTH     COMPONENTS   VERSION   PROMETHEUS   GRAFANA   LOKI      TEMPO     MESSAGE                                        AGE
production-full      Ready        Healthy    4            v2.0.0    Ready        Ready     Ready     Ready     All components are running successfully        5d
staging-installing   Installing   Unknown    0                      Deploying    Pending                       Installing Prometheus and Grafana components   2m

Using shortname:
kubectl get op
NAME                 PHASE        HEALTH     AGE
production-full      Ready        Healthy    5d
EOF
    
    echo
    print_status "success" "Printer columns validation completed!"
}

# Run main function
main "$@"
