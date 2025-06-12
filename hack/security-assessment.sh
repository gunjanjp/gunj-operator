#!/bin/bash

# Security Compliance Assessment Script
# Gunj Operator Project
# Version: 1.0

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Initialize scores
TOTAL_SCORE=0
MAX_SCORE=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# Results array
declare -A RESULTS
declare -A FINDINGS

echo "╔══════════════════════════════════════════════════════════╗"
echo "║      Security Compliance Assessment - Gunj Operator      ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "Assessment Date: $(date)"
echo ""

# Function to check security requirement
check_security() {
    local category=$1
    local check_name=$2
    local command=$3
    local expected=$4
    local severity=$5
    local points=$6
    
    MAX_SCORE=$((MAX_SCORE + points))
    
    echo -n "Checking $category - $check_name... "
    
    if eval "$command" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC} (+$points points)"
        TOTAL_SCORE=$((TOTAL_SCORE + points))
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        RESULTS["$category.$check_name"]="PASSED"
    else
        if [ "$severity" == "critical" ]; then
            echo -e "${RED}✗ FAILED${NC} (Critical)"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            FINDINGS["critical"]="${FINDINGS[critical]}\n- $category: $check_name"
        else
            echo -e "${YELLOW}⚠ WARNING${NC}"
            WARNINGS=$((WARNINGS + 1))
            FINDINGS["warning"]="${FINDINGS[warning]}\n- $category: $check_name"
        fi
        RESULTS["$category.$check_name"]="FAILED"
    fi
}

# Container Security Checks
echo -e "\n${BLUE}=== Container Security ===${NC}"

check_security "Container" "Dockerfile exists" \
    "[ -f Dockerfile ]" \
    "exists" "critical" 5

check_security "Container" "Non-root user" \
    "grep -q 'USER.*[0-9]' Dockerfile 2>/dev/null" \
    "found" "critical" 10

check_security "Container" "Distroless/minimal base" \
    "grep -qE '(distroless|alpine|scratch)' Dockerfile 2>/dev/null" \
    "found" "high" 5

check_security "Container" "No latest tag" \
    "! grep -q ':latest' Dockerfile 2>/dev/null" \
    "not found" "medium" 3

check_security "Container" "Multi-stage build" \
    "grep -q 'FROM.*AS' Dockerfile 2>/dev/null" \
    "found" "medium" 5

# Kubernetes Security Checks
echo -e "\n${BLUE}=== Kubernetes Security ===${NC}"

check_security "Kubernetes" "Security Context defined" \
    "find . -name '*.yaml' -o -name '*.yml' | xargs grep -l 'securityContext' 2>/dev/null | grep -q ." \
    "found" "critical" 10

check_security "Kubernetes" "Network Policies" \
    "find . -name '*.yaml' -o -name '*.yml' | xargs grep -l 'kind: NetworkPolicy' 2>/dev/null | grep -q ." \
    "found" "high" 8

check_security "Kubernetes" "RBAC configured" \
    "find . -name '*.yaml' -o -name '*.yml' | xargs grep -l 'kind: ClusterRole' 2>/dev/null | grep -q ." \
    "found" "critical" 10

check_security "Kubernetes" "Resource limits" \
    "find . -name '*.yaml' -o -name '*.yml' | xargs grep -l 'resources:' 2>/dev/null | grep -q ." \
    "found" "high" 5

check_security "Kubernetes" "Pod Security Standards" \
    "find . -name '*.yaml' -o -name '*.yml' | xargs grep -l 'securityContext' 2>/dev/null | xargs grep -q 'runAsNonRoot: true'" \
    "found" "critical" 8

# Secret Management Checks
echo -e "\n${BLUE}=== Secret Management ===${NC}"

check_security "Secrets" "No hardcoded secrets" \
    "! grep -rE '(password|secret|key|token)\\s*[:=]\\s*[\"'\''][^\"'\'']+[\"'\'']' --include='*.go' --include='*.js' --include='*.ts' --include='*.py' . 2>/dev/null | grep -v -E '(example|test|fake|dummy)' | grep -q ." \
    "not found" "critical" 15

check_security "Secrets" "Environment variables used" \
    "grep -r 'os.Getenv\\|process.env' --include='*.go' --include='*.js' --include='*.ts' . 2>/dev/null | grep -q ." \
    "found" "high" 5

check_security "Secrets" ".gitignore for secrets" \
    "grep -qE '(\\.env|secrets|credentials)' .gitignore 2>/dev/null" \
    "found" "critical" 5

# Code Security Checks
echo -e "\n${BLUE}=== Code Security ===${NC}"

check_security "Code" "Input validation" \
    "grep -r 'validate\\|Validate\\|validation' --include='*.go' . 2>/dev/null | grep -q ." \
    "found" "high" 8

check_security "Code" "Error handling" \
    "grep -r 'if err != nil' --include='*.go' . 2>/dev/null | grep -q ." \
    "found" "medium" 5

check_security "Code" "Secure TLS config" \
    "grep -r 'tls\\.Config' --include='*.go' . 2>/dev/null | grep -q 'MinVersion'" \
    "found" "high" 5

check_security "Code" "No SQL injection" \
    "! grep -r 'fmt\\.Sprintf.*\\(.*SELECT\\|INSERT\\|UPDATE\\|DELETE' --include='*.go' . 2>/dev/null | grep -q ." \
    "not found" "critical" 10

# CI/CD Security Checks
echo -e "\n${BLUE}=== CI/CD Security ===${NC}"

check_security "CI/CD" "Security scanning in pipeline" \
    "find .github .gitlab-ci.yml Jenkinsfile -name '*.yml' -o -name '*.yaml' 2>/dev/null | xargs grep -l 'trivy\\|snyk\\|gosec' 2>/dev/null | grep -q ." \
    "found" "high" 8

check_security "CI/CD" "Dependency scanning" \
    "find .github .gitlab-ci.yml Jenkinsfile -name '*.yml' -o -name '*.yaml' 2>/dev/null | xargs grep -l 'nancy\\|dependabot\\|renovate' 2>/dev/null | grep -q ." \
    "found" "medium" 5

check_security "CI/CD" "SAST implementation" \
    "find .github .gitlab-ci.yml Jenkinsfile -name '*.yml' -o -name '*.yaml' 2>/dev/null | xargs grep -l 'semgrep\\|sonarqube\\|codeql' 2>/dev/null | grep -q ." \
    "found" "medium" 5

# Authentication & Authorization Checks
echo -e "\n${BLUE}=== Authentication & Authorization ===${NC}"

check_security "Auth" "JWT implementation" \
    "grep -r 'jwt\\|JWT' --include='*.go' --include='*.js' --include='*.ts' . 2>/dev/null | grep -q ." \
    "found" "critical" 10

check_security "Auth" "RBAC implementation" \
    "grep -r 'rbac\\|RBAC\\|role\\|permission' --include='*.go' . 2>/dev/null | grep -q ." \
    "found" "critical" 10

check_security "Auth" "Rate limiting" \
    "grep -r 'rate.*limit\\|RateLimit' --include='*.go' --include='*.js' . 2>/dev/null | grep -q ." \
    "found" "high" 5

# Supply Chain Security
echo -e "\n${BLUE}=== Supply Chain Security ===${NC}"

check_security "Supply Chain" "Go modules" \
    "[ -f go.mod ] && [ -f go.sum ]" \
    "exists" "critical" 5

check_security "Supply Chain" "Package lock files" \
    "[ -f package-lock.json ] || [ -f yarn.lock ]" \
    "exists" "high" 3

check_security "Supply Chain" "Signed commits" \
    "git config --get commit.gpgsign 2>/dev/null | grep -q true" \
    "enabled" "medium" 3

# Generate JSON report
generate_json_report() {
    cat > security-assessment-report.json <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "project": "gunj-operator",
  "summary": {
    "total_score": $TOTAL_SCORE,
    "max_score": $MAX_SCORE,
    "percentage": $(( (TOTAL_SCORE * 100) / MAX_SCORE )),
    "passed_checks": $PASSED_CHECKS,
    "failed_checks": $FAILED_CHECKS,
    "warnings": $WARNINGS
  },
  "compliance": {
    "cis_kubernetes": $(( (TOTAL_SCORE * 100) / MAX_SCORE ))
  },
  "findings": {
    "critical": [$(echo -e "${FINDINGS[critical]}" | sed 's/^/"/;s/$/",/' | tr '\n' ' ' | sed 's/, $//')],
    "warning": [$(echo -e "${FINDINGS[warning]}" | sed 's/^/"/;s/$/",/' | tr '\n' ' ' | sed 's/, $//')],
  },
  "recommendations": []
}
EOF
}

# Calculate compliance level
calculate_compliance() {
    local percentage=$(( (TOTAL_SCORE * 100) / MAX_SCORE ))
    
    if [ $percentage -ge 90 ]; then
        echo "Excellent"
    elif [ $percentage -ge 75 ]; then
        echo "Good"
    elif [ $percentage -ge 60 ]; then
        echo "Fair"
    elif [ $percentage -ge 40 ]; then
        echo "Poor"
    else
        echo "Critical"
    fi
}

# Generate recommendations
generate_recommendations() {
    echo -e "\n${YELLOW}Recommendations:${NC}"
    
    if [ ${FAILED_CHECKS} -gt 0 ]; then
        echo "├─ Address $FAILED_CHECKS critical security failures immediately"
    fi
    
    if [ ${WARNINGS} -gt 0 ]; then
        echo "├─ Review and fix $WARNINGS security warnings"
    fi
    
    if [ -n "${FINDINGS[critical]}" ]; then
        echo "├─ Critical findings require immediate attention:"
        echo -e "${FINDINGS[critical]}" | sed 's/^/│  /'
    fi
    
    echo "└─ Run security scanning tools in CI/CD pipeline"
}

# Print summary
echo -e "\n${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    Assessment Summary                     ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"

echo -e "\n${YELLOW}Security Score:${NC} $TOTAL_SCORE/$MAX_SCORE ($(( (TOTAL_SCORE * 100) / MAX_SCORE ))%)"
echo -e "${YELLOW}Compliance Level:${NC} $(calculate_compliance)"
echo -e "${YELLOW}Checks:${NC} ${GREEN}$PASSED_CHECKS passed${NC}, ${RED}$FAILED_CHECKS failed${NC}, ${YELLOW}$WARNINGS warnings${NC}"

# Generate recommendations
generate_recommendations

# Generate reports
generate_json_report

echo -e "\n${GREEN}Assessment complete! Reports generated:${NC}"
echo "- security-assessment-report.json"

# Exit with appropriate code
if [ $FAILED_CHECKS -gt 0 ]; then
    exit 1
else
    exit 0
fi
