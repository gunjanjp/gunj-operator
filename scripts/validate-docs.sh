#!/bin/bash
# Documentation validation script for Gunj Operator

set -e

echo "🔍 Validating documentation..."

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Counters
ERRORS=0
WARNINGS=0

# Function to report errors
error() {
    echo -e "${RED}❌ ERROR: $1${NC}"
    ((ERRORS++))
}

# Function to report warnings
warning() {
    echo -e "${YELLOW}⚠️  WARNING: $1${NC}"
    ((WARNINGS++))
}

# Function to report success
success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Check if required tools are installed
check_tools() {
    echo "Checking required tools..."
    
    local required_tools=("markdownlint" "linkcheck" "vale")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is not installed"
        else
            success "$tool is installed"
        fi
    done
}

# Validate markdown files
validate_markdown() {
    echo -e "\n📝 Validating Markdown files..."
    
    if command -v markdownlint &> /dev/null; then
        if markdownlint '**/*.md' --config .markdownlint.json --ignore node_modules --ignore vendor; then
            success "Markdown validation passed"
        else
            error "Markdown validation failed"
        fi
    fi
}

# Check for broken links
check_links() {
    echo -e "\n🔗 Checking for broken links..."
    
    local md_files=$(find docs -name "*.md" -type f)
    local broken_links=0
    
    for file in $md_files; do
        echo "Checking $file..."
        
        # Extract links from markdown
        local links=$(grep -oP '\[([^\]]+)\]\(([^)]+)\)' "$file" | grep -oP '(?<=\()([^)]+)(?=\))')
        
        for link in $links; do
            # Skip external links for now
            if [[ $link == http* ]]; then
                continue
            fi
            
            # Check internal links
            if [[ $link == /* ]]; then
                # Absolute path from docs root
                local target="docs${link}"
            else
                # Relative path
                local dir=$(dirname "$file")
                local target="$dir/$link"
            fi
            
            # Remove anchor if present
            target=${target%%#*}
            
            # Check if file exists
            if [[ ! -f "$target" ]] && [[ ! -f "${target}.md" ]]; then
                error "Broken link in $file: $link"
                ((broken_links++))
            fi
        done
    done
    
    if [[ $broken_links -eq 0 ]]; then
        success "No broken internal links found"
    fi
}

# Validate code examples
validate_code_examples() {
    echo -e "\n💻 Validating code examples..."
    
    # Extract and validate YAML examples
    local yaml_blocks=$(find docs -name "*.md" -type f -exec grep -l '```yaml' {} \;)
    local yaml_errors=0
    
    for file in $yaml_blocks; do
        # Extract YAML blocks and validate
        awk '/```yaml/{flag=1; next} /```/{flag=0} flag' "$file" > /tmp/temp.yaml
        
        if [[ -s /tmp/temp.yaml ]]; then
            if ! yamllint -c .yamllint.yml /tmp/temp.yaml > /dev/null 2>&1; then
                warning "Invalid YAML in $file"
                ((yaml_errors++))
            fi
        fi
    done
    
    if [[ $yaml_errors -eq 0 ]]; then
        success "All YAML examples are valid"
    fi
}

# Check documentation structure
check_structure() {
    echo -e "\n📁 Checking documentation structure..."
    
    local required_files=(
        "docs/index.md"
        "docs/getting-started/installation.md"
        "docs/user-guide/configuration.md"
        "docs/api/rest-api.md"
        "docs/architecture/overview.md"
    )
    
    for file in "${required_files[@]}"; do
        if [[ -f "$file" ]]; then
            success "$file exists"
        else
            error "$file is missing"
        fi
    done
}

# Validate Vale style
validate_style() {
    echo -e "\n✍️  Validating writing style..."
    
    if command -v vale &> /dev/null; then
        if [[ -f ".vale.ini" ]]; then
            if vale docs/; then
                success "Style validation passed"
            else
                warning "Style issues found"
            fi
        else
            warning ".vale.ini configuration not found"
        fi
    fi
}

# Check for required sections in docs
check_required_sections() {
    echo -e "\n📋 Checking for required sections..."
    
    local user_guides=$(find docs/user-guide -name "*.md" -type f)
    
    for file in $user_guides; do
        local required_sections=("Overview" "Prerequisites" "Configuration" "Troubleshooting")
        
        for section in "${required_sections[@]}"; do
            if ! grep -q "^#.*$section" "$file"; then
                warning "$file is missing section: $section"
            fi
        done
    done
}

# Check for outdated content
check_outdated() {
    echo -e "\n📅 Checking for outdated content..."
    
    # Check for old version references
    local old_versions=$(grep -r "v1\." docs/ --include="*.md" | grep -v "v1\.0" | wc -l)
    if [[ $old_versions -gt 0 ]]; then
        warning "Found $old_versions references to old versions"
    fi
    
    # Check for TODO items
    local todos=$(grep -r "TODO\|FIXME" docs/ --include="*.md" | wc -l)
    if [[ $todos -gt 0 ]]; then
        warning "Found $todos TODO/FIXME items in documentation"
    fi
}

# Generate documentation report
generate_report() {
    echo -e "\n📊 Documentation Report"
    echo "======================="
    
    # Count documentation files
    local total_files=$(find docs -name "*.md" -type f | wc -l)
    local total_words=$(find docs -name "*.md" -type f -exec wc -w {} + | tail -1 | awk '{print $1}')
    
    echo "Total documentation files: $total_files"
    echo "Total word count: $total_words"
    echo "Errors found: $ERRORS"
    echo "Warnings found: $WARNINGS"
    
    # Generate coverage report
    local api_docs=$(find docs/api -name "*.md" -type f | wc -l)
    local user_docs=$(find docs/user-guide -name "*.md" -type f | wc -l)
    local dev_docs=$(find docs/development -name "*.md" -type f | wc -l)
    
    echo -e "\nDocumentation coverage:"
    echo "  API documentation: $api_docs files"
    echo "  User guides: $user_docs files"
    echo "  Developer docs: $dev_docs files"
}

# Main execution
main() {
    echo "📚 Gunj Operator Documentation Validation"
    echo "========================================"
    
    check_tools
    validate_markdown
    check_links
    validate_code_examples
    check_structure
    validate_style
    check_required_sections
    check_outdated
    generate_report
    
    echo -e "\n"
    if [[ $ERRORS -gt 0 ]]; then
        echo -e "${RED}❌ Documentation validation failed with $ERRORS errors${NC}"
        exit 1
    elif [[ $WARNINGS -gt 0 ]]; then
        echo -e "${YELLOW}⚠️  Documentation validation passed with $WARNINGS warnings${NC}"
        exit 0
    else
        echo -e "${GREEN}✅ Documentation validation passed!${NC}"
        exit 0
    fi
}

# Run main function
main
