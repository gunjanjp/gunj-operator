#!/bin/bash

# Accessibility Testing Script
# Gunj Operator Project
# Version: 1.0

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test results
PASSED=0
FAILED=0
WARNINGS=0

echo "╔══════════════════════════════════════════════════════════╗"
echo "║       Accessibility Testing - Gunj Operator UI           ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "Date: $(date)"
echo ""

# Function to run accessibility test
run_test() {
    local test_name=$1
    local command=$2
    local expected=$3
    
    echo -n "Testing $test_name... "
    
    if eval "$command" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗ FAILED${NC}"
        FAILED=$((FAILED + 1))
        echo "  Command: $command"
    fi
}

# Check for required dependencies
echo -e "\n${BLUE}=== Checking Dependencies ===${NC}"

run_test "Node.js installed" "which node" "exists"
run_test "npm installed" "which npm" "exists"
run_test "UI directory exists" "[ -d ui ]" "exists"

# Install accessibility testing tools if needed
echo -e "\n${BLUE}=== Setting Up Testing Tools ===${NC}"

if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install --save-dev \
        eslint-plugin-jsx-a11y \
        jest-axe \
        @testing-library/react \
        @testing-library/jest-dom \
        lighthouse \
        pa11y \
        axe-core
fi

# Static Analysis
echo -e "\n${BLUE}=== Static Accessibility Analysis ===${NC}"

# ESLint with jsx-a11y plugin
if [ -f ".eslintrc.json" ]; then
    echo "Running ESLint accessibility checks..."
    npx eslint --ext .jsx,.js,.tsx,.ts ui/src \
        --plugin jsx-a11y \
        --rule 'jsx-a11y/alt-text: error' \
        --rule 'jsx-a11y/aria-props: error' \
        --rule 'jsx-a11y/aria-role: error' \
        --format json > eslint-a11y-report.json || true
    
    # Count issues
    ESLINT_ERRORS=$(jq '[.[] | .errorCount] | add' eslint-a11y-report.json || echo "0")
    ESLINT_WARNINGS=$(jq '[.[] | .warningCount] | add' eslint-a11y-report.json || echo "0")
    
    if [ "$ESLINT_ERRORS" -eq 0 ]; then
        echo -e "${GREEN}✓ No accessibility errors found${NC}"
    else
        echo -e "${RED}✗ Found $ESLINT_ERRORS accessibility errors${NC}"
        FAILED=$((FAILED + ESLINT_ERRORS))
    fi
    
    if [ "$ESLINT_WARNINGS" -gt 0 ]; then
        echo -e "${YELLOW}⚠ Found $ESLINT_WARNINGS accessibility warnings${NC}"
        WARNINGS=$((WARNINGS + ESLINT_WARNINGS))
    fi
fi

# Color Contrast Check
echo -e "\n${BLUE}=== Color Contrast Analysis ===${NC}"

# Extract colors from CSS
if [ -f "ui/src/styles/colors.css" ] || [ -f "ui/src/index.css" ]; then
    echo "Analyzing color contrast ratios..."
    
    # Create temporary HTML file for contrast testing
    cat > temp-contrast-test.html <<EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <link rel="stylesheet" href="ui/src/index.css">
</head>
<body>
    <div style="background: var(--color-bg); color: var(--color-text)">Normal text</div>
    <div style="background: var(--color-primary); color: white">Button text</div>
    <div style="background: var(--color-error); color: white">Error text</div>
</body>
</html>
EOF
    
    # Use pa11y for contrast checking
    if which pa11y >/dev/null 2>&1; then
        pa11y temp-contrast-test.html \
            --standard WCAG2AA \
            --reporter json > contrast-report.json || true
        
        CONTRAST_ISSUES=$(jq 'length' contrast-report.json || echo "0")
        
        if [ "$CONTRAST_ISSUES" -eq 0 ]; then
            echo -e "${GREEN}✓ All color contrasts meet WCAG AA${NC}"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}✗ Found $CONTRAST_ISSUES contrast issues${NC}"
            FAILED=$((FAILED + CONTRAST_ISSUES))
        fi
    fi
    
    rm -f temp-contrast-test.html
fi

# Component Testing
echo -e "\n${BLUE}=== Component Accessibility Tests ===${NC}"

# Run Jest tests with axe
if [ -f "ui/package.json" ]; then
    cd ui
    
    # Run accessibility tests
    echo "Running component accessibility tests..."
    npm test -- --testPathPattern=".*\\.a11y\\.test\\.(js|jsx|ts|tsx)$" \
        --coverage \
        --coverageDirectory=../coverage/a11y \
        --json \
        --outputFile=../jest-a11y-results.json || true
    
    cd ..
    
    # Parse results
    if [ -f "jest-a11y-results.json" ]; then
        JEST_PASSED=$(jq '.numPassedTests' jest-a11y-results.json || echo "0")
        JEST_FAILED=$(jq '.numFailedTests' jest-a11y-results.json || echo "0")
        
        PASSED=$((PASSED + JEST_PASSED))
        FAILED=$((FAILED + JEST_FAILED))
        
        echo -e "Component tests: ${GREEN}$JEST_PASSED passed${NC}, ${RED}$JEST_FAILED failed${NC}"
    fi
fi

# Lighthouse Audit
echo -e "\n${BLUE}=== Lighthouse Accessibility Audit ===${NC}"

# Check if development server is running
if curl -s http://localhost:3000 > /dev/null; then
    echo "Running Lighthouse audit..."
    
    npx lighthouse http://localhost:3000 \
        --only-categories=accessibility \
        --output=json \
        --output-path=./lighthouse-a11y-report.json \
        --chrome-flags="--headless" || true
    
    if [ -f "lighthouse-a11y-report.json" ]; then
        LIGHTHOUSE_SCORE=$(jq '.categories.accessibility.score * 100' lighthouse-a11y-report.json || echo "0")
        echo -e "Lighthouse Accessibility Score: ${YELLOW}${LIGHTHOUSE_SCORE}%${NC}"
        
        if (( $(echo "$LIGHTHOUSE_SCORE >= 90" | bc -l) )); then
            echo -e "${GREEN}✓ Excellent accessibility score${NC}"
            PASSED=$((PASSED + 1))
        elif (( $(echo "$LIGHTHOUSE_SCORE >= 70" | bc -l) )); then
            echo -e "${YELLOW}⚠ Good accessibility score, but improvements needed${NC}"
            WARNINGS=$((WARNINGS + 1))
        else
            echo -e "${RED}✗ Poor accessibility score${NC}"
            FAILED=$((FAILED + 1))
        fi
    fi
else
    echo -e "${YELLOW}⚠ Development server not running, skipping Lighthouse audit${NC}"
    echo "  Start the dev server with: cd ui && npm start"
fi

# ARIA Validation
echo -e "\n${BLUE}=== ARIA Usage Validation ===${NC}"

# Check for common ARIA mistakes
echo "Checking for ARIA anti-patterns..."

# Check for aria-label on elements that don't support it
ARIA_LABEL_MISUSE=$(grep -r 'aria-label=' ui/src --include="*.jsx" --include="*.tsx" | \
    grep -E '<(div|span|p|section).*aria-label=' | wc -l || echo "0")

if [ "$ARIA_LABEL_MISUSE" -eq 0 ]; then
    echo -e "${GREEN}✓ No aria-label misuse detected${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}⚠ Found $ARIA_LABEL_MISUSE potential aria-label misuses${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

# Check for redundant ARIA roles
REDUNDANT_ROLES=$(grep -r 'role=' ui/src --include="*.jsx" --include="*.tsx" | \
    grep -E '<(button.*role="button"|a.*role="link"|nav.*role="navigation")' | wc -l || echo "0")

if [ "$REDUNDANT_ROLES" -eq 0 ]; then
    echo -e "${GREEN}✓ No redundant ARIA roles detected${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}⚠ Found $REDUNDANT_ROLES redundant ARIA roles${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

# Keyboard Navigation Check
echo -e "\n${BLUE}=== Keyboard Navigation Check ===${NC}"

# Check for proper tabindex usage
POSITIVE_TABINDEX=$(grep -r 'tabIndex=' ui/src --include="*.jsx" --include="*.tsx" | \
    grep -E 'tabIndex="[1-9]|tabIndex={[1-9]' | wc -l || echo "0")

if [ "$POSITIVE_TABINDEX" -eq 0 ]; then
    echo -e "${GREEN}✓ No positive tabindex values found${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ Found $POSITIVE_TABINDEX positive tabindex values${NC}"
    FAILED=$((FAILED + 1))
fi

# Generate Summary Report
echo -e "\n${BLUE}=== Generating Accessibility Report ===${NC}"

cat > accessibility-test-report.json <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "project": "gunj-operator-ui",
  "summary": {
    "total_tests": $((PASSED + FAILED)),
    "passed": $PASSED,
    "failed": $FAILED,
    "warnings": $WARNINGS,
    "compliance_percentage": $(( (PASSED * 100) / (PASSED + FAILED + 1) ))
  },
  "wcag_compliance": {
    "level_a": $([ $FAILED -eq 0 ] && echo "true" || echo "false"),
    "level_aa": $([ $FAILED -eq 0 ] && [ $WARNINGS -lt 5 ] && echo "true" || echo "false")
  },
  "audits": {
    "eslint": {
      "errors": ${ESLINT_ERRORS:-0},
      "warnings": ${ESLINT_WARNINGS:-0}
    },
    "lighthouse": {
      "score": ${LIGHTHOUSE_SCORE:-0}
    },
    "contrast": {
      "issues": ${CONTRAST_ISSUES:-0}
    }
  }
}
EOF

# Generate HTML Report
cat > accessibility-test-report.html <<EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Accessibility Test Report - Gunj Operator</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .pass { color: green; }
        .fail { color: red; }
        .warn { color: orange; }
        .summary { background: #f0f0f0; padding: 20px; border-radius: 8px; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>Accessibility Test Report</h1>
    <p>Generated: $(date)</p>
    
    <div class="summary">
        <h2>Summary</h2>
        <p>Tests Passed: <span class="pass">$PASSED</span></p>
        <p>Tests Failed: <span class="fail">$FAILED</span></p>
        <p>Warnings: <span class="warn">$WARNINGS</span></p>
        <p>Compliance: $(( (PASSED * 100) / (PASSED + FAILED + 1) ))%</p>
    </div>
    
    <h2>Detailed Results</h2>
    <table>
        <tr>
            <th>Category</th>
            <th>Status</th>
            <th>Details</th>
        </tr>
        <tr>
            <td>ESLint Accessibility</td>
            <td>$([ "${ESLINT_ERRORS:-0}" -eq 0 ] && echo '<span class="pass">PASS</span>' || echo '<span class="fail">FAIL</span>')</td>
            <td>${ESLINT_ERRORS:-0} errors, ${ESLINT_WARNINGS:-0} warnings</td>
        </tr>
        <tr>
            <td>Color Contrast</td>
            <td>$([ "${CONTRAST_ISSUES:-0}" -eq 0 ] && echo '<span class="pass">PASS</span>' || echo '<span class="fail">FAIL</span>')</td>
            <td>${CONTRAST_ISSUES:-0} issues</td>
        </tr>
        <tr>
            <td>Lighthouse Score</td>
            <td>$([ "${LIGHTHOUSE_SCORE:-0}" -ge 90 ] && echo '<span class="pass">PASS</span>' || echo '<span class="warn">NEEDS IMPROVEMENT</span>')</td>
            <td>${LIGHTHOUSE_SCORE:-0}%</td>
        </tr>
    </table>
    
    <h2>Next Steps</h2>
    <ul>
        <li>Review detailed reports in individual JSON files</li>
        <li>Fix all failing tests before merging</li>
        <li>Address warnings to improve accessibility</li>
        <li>Run manual screen reader testing</li>
    </ul>
</body>
</html>
EOF

# Print Summary
echo -e "\n${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                   Test Summary                           ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"

echo -e "\n${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo -e "${YELLOW}Warnings: $WARNINGS${NC}"
echo -e "\nCompliance: $(( (PASSED * 100) / (PASSED + FAILED + 1) ))%"

echo -e "\n${GREEN}Reports generated:${NC}"
echo "  - accessibility-test-report.json"
echo "  - accessibility-test-report.html"
echo "  - eslint-a11y-report.json"
echo "  - lighthouse-a11y-report.json"

# Exit with appropriate code
if [ $FAILED -gt 0 ]; then
    echo -e "\n${RED}❌ Accessibility tests failed. Please fix issues before proceeding.${NC}"
    exit 1
else
    echo -e "\n${GREEN}✅ All accessibility tests passed!${NC}"
    exit 0
fi
