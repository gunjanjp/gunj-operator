#!/bin/bash

# Cloud Native Maturity Assessment Script
# Gunj Operator Project
# Version: 1.0

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Initialize scores
TOTAL_SCORE=0
LEVEL1_SCORE=0
LEVEL2_SCORE=0
LEVEL3_SCORE=0
LEVEL4_SCORE=0
LEVEL5_SCORE=0

# Assessment results
declare -A RESULTS

echo "╔══════════════════════════════════════════════════════════╗"
echo "║     Cloud Native Maturity Assessment - Gunj Operator     ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

# Function to add score
add_score() {
    local level=$1
    local points=$2
    local description=$3
    
    TOTAL_SCORE=$((TOTAL_SCORE + points))
    
    case $level in
        1) LEVEL1_SCORE=$((LEVEL1_SCORE + points)) ;;
        2) LEVEL2_SCORE=$((LEVEL2_SCORE + points)) ;;
        3) LEVEL3_SCORE=$((LEVEL3_SCORE + points)) ;;
        4) LEVEL4_SCORE=$((LEVEL4_SCORE + points)) ;;
        5) LEVEL5_SCORE=$((LEVEL5_SCORE + points)) ;;
    esac
    
    echo -e "${GREEN}✓${NC} $description (+$points points)"
}

# Function to report missing requirement
report_missing() {
    local level=$1
    local description=$2
    
    echo -e "${RED}✗${NC} $description (Level $level requirement)"
    RESULTS["missing_$level"]+="- $description\n"
}

# Level 1: Containerization Assessment
echo -e "\n${BLUE}=== Level 1: Containerization ===${NC}"

# Check for Dockerfile
if [ -f "Dockerfile" ]; then
    add_score 1 5 "Dockerfile exists"
    
    # Check for multi-stage build
    if grep -q "FROM.*AS.*builder" Dockerfile 2>/dev/null; then
        add_score 1 5 "Multi-stage build pattern detected"
    else
        report_missing 1 "Multi-stage build not implemented"
    fi
    
    # Check for non-root user
    if grep -q "USER.*[0-9]" Dockerfile 2>/dev/null; then
        add_score 1 5 "Non-root user configured"
    else
        report_missing 1 "Container runs as root user"
    fi
    
    # Check for distroless or minimal base
    if grep -qE "(distroless|alpine|scratch)" Dockerfile 2>/dev/null; then
        add_score 1 5 "Minimal base image used"
    else
        report_missing 1 "Not using minimal base image"
    fi
else
    report_missing 1 "Dockerfile not found"
fi

# Check for .dockerignore
if [ -f ".dockerignore" ]; then
    add_score 1 2 ".dockerignore file present"
else
    report_missing 1 ".dockerignore file missing"
fi

# Level 2: Dynamic Orchestration Assessment
echo -e "\n${BLUE}=== Level 2: Dynamic Orchestration ===${NC}"

# Check for Kubernetes manifests
if [ -d "config" ]; then
    if [ -d "config/crd" ]; then
        add_score 2 10 "Custom Resource Definitions found"
    else
        report_missing 2 "No CRDs defined"
    fi
    
    if [ -d "config/rbac" ]; then
        add_score 2 5 "RBAC configuration present"
    else
        report_missing 2 "RBAC not configured"
    fi
    
    if [ -d "config/manager" ]; then
        add_score 2 5 "Manager configuration present"
    else
        report_missing 2 "Manager configuration missing"
    fi
fi

# Check for controller implementation
if [ -d "controllers" ]; then
    if ls controllers/*_controller.go 1> /dev/null 2>&1; then
        add_score 2 10 "Controller implementation found"
    else
        report_missing 2 "No controller implementation"
    fi
fi

# Check for Helm chart
if [ -d "charts" ] || [ -d "helm" ]; then
    add_score 2 5 "Helm chart present"
else
    report_missing 2 "No Helm chart found"
fi

# Level 3: Microservices Oriented Assessment
echo -e "\n${BLUE}=== Level 3: Microservices Oriented ===${NC}"

# Check for API implementation
if [ -d "api" ] || [ -d "pkg/api" ] || [ -d "internal/api" ]; then
    add_score 3 10 "API implementation found"
else
    report_missing 3 "No API implementation"
fi

# Check for service separation
if [ -d "cmd" ]; then
    service_count=$(find cmd -type d -mindepth 1 | wc -l)
    if [ $service_count -gt 1 ]; then
        add_score 3 10 "Multiple services detected ($service_count services)"
    else
        report_missing 3 "Monolithic architecture detected"
    fi
fi

# Check for observability
if grep -r "prometheus\|opentelemetry\|tracing" . --include="*.go" 2>/dev/null | grep -q .; then
    add_score 3 5 "Observability instrumentation detected"
else
    report_missing 3 "No observability instrumentation"
fi

# Level 4: Cloud Native Services Assessment
echo -e "\n${BLUE}=== Level 4: Cloud Native Services ===${NC}"

# Check for cloud provider integration
if grep -r "aws\|azure\|gcp" . --include="*.go" 2>/dev/null | grep -q .; then
    add_score 4 10 "Cloud provider integration detected"
else
    report_missing 4 "No cloud provider integration"
fi

# Check for external service integration
if [ -f "go.mod" ]; then
    if grep -q "github.com/aws/aws-sdk-go\|cloud.google.com/go\|github.com/Azure" go.mod; then
        add_score 4 5 "Cloud SDK dependencies found"
    else
        report_missing 4 "No cloud SDK dependencies"
    fi
fi

# Level 5: Cloud Native Operations Assessment
echo -e "\n${BLUE}=== Level 5: Cloud Native Operations ===${NC}"

# Check for GitOps configuration
if [ -d ".github/workflows" ] || [ -d ".gitlab-ci" ] || [ -f ".drone.yml" ]; then
    add_score 5 10 "CI/CD configuration found"
else
    report_missing 5 "No CI/CD configuration"
fi

# Check for automated testing
if [ -d "test" ] || [ -d "tests" ]; then
    add_score 5 5 "Test directory found"
    
    # Check for e2e tests
    if find . -name "*e2e*" -type f | grep -q .; then
        add_score 5 5 "End-to-end tests present"
    else
        report_missing 5 "No e2e tests found"
    fi
fi

# Calculate maturity level
calculate_maturity_level() {
    local score=$1
    
    if [ $score -ge 90 ]; then
        echo "5 - Cloud Native Operations"
    elif [ $score -ge 70 ]; then
        echo "4 - Cloud Native Services"
    elif [ $score -ge 50 ]; then
        echo "3 - Microservices Oriented"
    elif [ $score -ge 30 ]; then
        echo "2 - Dynamically Orchestrated"
    elif [ $score -ge 10 ]; then
        echo "1 - Containerized"
    else
        echo "0 - Traditional"
    fi
}

# Generate report
echo -e "\n${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    Assessment Results                    ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"

echo -e "\n${YELLOW}Score Breakdown:${NC}"
echo "├─ Level 1 (Containerization):        $LEVEL1_SCORE/20 points"
echo "├─ Level 2 (Dynamic Orchestration):   $LEVEL2_SCORE/35 points"
echo "├─ Level 3 (Microservices):          $LEVEL3_SCORE/25 points"
echo "├─ Level 4 (Cloud Native Services):   $LEVEL4_SCORE/15 points"
echo "└─ Level 5 (Cloud Native Operations): $LEVEL5_SCORE/20 points"

echo -e "\n${YELLOW}Overall Assessment:${NC}"
echo "├─ Total Score: $TOTAL_SCORE/115 points"
PERCENTAGE=$((TOTAL_SCORE * 100 / 115))
echo "├─ Percentage: $PERCENTAGE%"
MATURITY_LEVEL=$(calculate_maturity_level $PERCENTAGE)
echo "└─ Maturity Level: $MATURITY_LEVEL"

# Generate recommendations
echo -e "\n${YELLOW}Recommendations:${NC}"

if [ $LEVEL1_SCORE -lt 20 ]; then
    echo "├─ Focus on completing Level 1 containerization requirements"
fi

if [ $LEVEL2_SCORE -lt 35 ] && [ $LEVEL1_SCORE -ge 15 ]; then
    echo "├─ Implement Kubernetes orchestration features"
fi

if [ $LEVEL3_SCORE -lt 25 ] && [ $LEVEL2_SCORE -ge 25 ]; then
    echo "├─ Adopt microservices architecture patterns"
fi

if [ $LEVEL4_SCORE -lt 15 ] && [ $LEVEL3_SCORE -ge 15 ]; then
    echo "├─ Integrate with cloud native services"
fi

if [ $LEVEL5_SCORE -lt 20 ] && [ $LEVEL4_SCORE -ge 10 ]; then
    echo "└─ Implement advanced automation and operations"
fi

# Generate JSON report
cat > maturity-assessment-report.json <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "project": "gunj-operator",
  "scores": {
    "total": $TOTAL_SCORE,
    "percentage": $PERCENTAGE,
    "level1": $LEVEL1_SCORE,
    "level2": $LEVEL2_SCORE,
    "level3": $LEVEL3_SCORE,
    "level4": $LEVEL4_SCORE,
    "level5": $LEVEL5_SCORE
  },
  "maturityLevel": "$MATURITY_LEVEL",
  "recommendations": []
}
EOF

echo -e "\n${GREEN}Assessment complete! Report saved to maturity-assessment-report.json${NC}"
