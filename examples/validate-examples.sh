#!/bin/bash
# validate-examples.sh - Validates all example CR manifests
# This script checks that all example YAML files are valid and can be applied

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
EXAMPLES_DIR="${SCRIPT_DIR}"
FAILED_FILES=()
PASSED_FILES=()

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 Validating ObservabilityPlatform example manifests..."
echo "=================================================="

# Function to validate a YAML file
validate_yaml() {
    local file=$1
    echo -n "Checking $(basename "$file")... "
    
    # Check if file exists
    if [[ ! -f "$file" ]]; then
        echo -e "${RED}✗ File not found${NC}"
        FAILED_FILES+=("$file")
        return 1
    fi
    
    # Basic YAML syntax check using kubectl
    if kubectl apply --dry-run=client -f "$file" &>/dev/null; then
        # Check for required fields
        if grep -q "apiVersion: observability.io/v1beta1" "$file" && \
           grep -q "kind: ObservabilityPlatform" "$file" && \
           grep -q "metadata:" "$file" && \
           grep -q "spec:" "$file"; then
            echo -e "${GREEN}✓ Valid${NC}"
            PASSED_FILES+=("$file")
            return 0
        else
            echo -e "${RED}✗ Missing required fields${NC}"
            FAILED_FILES+=("$file")
            return 1
        fi
    else
        echo -e "${RED}✗ Invalid YAML syntax${NC}"
        kubectl apply --dry-run=client -f "$file" 2>&1 | grep -E "error|Error" || true
        FAILED_FILES+=("$file")
        return 1
    fi
}

# Function to check example completeness
check_completeness() {
    local file=$1
    local warnings=()
    
    # Check for comments/documentation
    if ! grep -q "^#" "$file"; then
        warnings+=("No comments/documentation found")
    fi
    
    # Check for resource limits
    if grep -q "resources:" "$file" && ! grep -q "limits:" "$file"; then
        warnings+=("Resources defined but no limits specified")
    fi
    
    # Check for version specifications
    if ! grep -q "version:" "$file"; then
        warnings+=("No version specified for components")
    fi
    
    if [[ ${#warnings[@]} -gt 0 ]]; then
        echo -e "${YELLOW}  ⚠ Warnings:${NC}"
        for warning in "${warnings[@]}"; do
            echo "    - $warning"
        done
    fi
}

# Find all YAML files in examples directory
echo "Finding example files..."
mapfile -t YAML_FILES < <(find "$EXAMPLES_DIR" -name "*.yaml" -o -name "*.yml" | grep -v "node_modules" | sort)

echo "Found ${#YAML_FILES[@]} YAML files to validate"
echo ""

# Validate each file
for file in "${YAML_FILES[@]}"; do
    validate_yaml "$file"
    check_completeness "$file"
done

echo ""
echo "=================================================="
echo "Validation Summary:"
echo "  Total files: ${#YAML_FILES[@]}"
echo -e "  ${GREEN}Passed: ${#PASSED_FILES[@]}${NC}"
echo -e "  ${RED}Failed: ${#FAILED_FILES[@]}${NC}"

if [[ ${#FAILED_FILES[@]} -gt 0 ]]; then
    echo ""
    echo -e "${RED}Failed files:${NC}"
    for file in "${FAILED_FILES[@]}"; do
        echo "  - $file"
    done
    exit 1
else
    echo ""
    echo -e "${GREEN}✅ All example manifests are valid!${NC}"
fi

# Additional checks if CRD is installed
if kubectl get crd observabilityplatforms.observability.io &>/dev/null; then
    echo ""
    echo "📋 CRD is installed. Running additional validation..."
    
    for file in "${PASSED_FILES[@]}"; do
        echo -n "  Validating against CRD schema: $(basename "$file")... "
        if kubectl apply --dry-run=server -f "$file" &>/dev/null; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${YELLOW}⚠ Schema validation failed${NC}"
        fi
    done
else
    echo ""
    echo -e "${YELLOW}ℹ️  CRD not installed. Skipping schema validation.${NC}"
    echo "   Install the operator to enable full validation."
fi
