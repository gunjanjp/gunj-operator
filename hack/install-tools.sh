#!/usr/bin/env bash
# Script to install all code quality tools

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "Installing code quality tools for Gunj Operator..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check if running as root
if [ "$EUID" -eq 0 ]; then
   error "Please do not run this script as root"
fi

# Install Go tools
info "Installing Go code quality tools..."

# golangci-lint
info "Installing golangci-lint..."
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

# goimports
info "Installing goimports..."
go install golang.org/x/tools/cmd/goimports@latest

# gosec
info "Installing gosec..."
go install github.com/securego/gosec/v2/cmd/gosec@latest

# go-licenses
info "Installing go-licenses..."
go install github.com/google/go-licenses@latest

# mockgen
info "Installing mockgen..."
go install go.uber.org/mock/mockgen@latest

# govulncheck
info "Installing govulncheck..."
go install golang.org/x/vuln/cmd/govulncheck@latest

# gocovmerge
info "Installing gocovmerge..."
go install github.com/wadey/gocovmerge@latest

# Install Node.js tools
info "Installing Node.js code quality tools..."

cd "${PROJECT_ROOT}/ui"

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    error "npm is not installed. Please install Node.js and npm first."
fi

# Install ESLint and related packages
info "Installing ESLint and plugins..."
npm install --save-dev \
    eslint@^8.56.0 \
    @typescript-eslint/eslint-plugin@^6.19.0 \
    @typescript-eslint/parser@^6.19.0 \
    eslint-config-prettier@^9.1.0 \
    eslint-plugin-react@^7.33.2 \
    eslint-plugin-react-hooks@^4.6.0 \
    eslint-plugin-jsx-a11y@^6.8.0 \
    eslint-plugin-import@^2.29.1 \
    eslint-plugin-jest@^27.6.3 \
    eslint-plugin-testing-library@^6.2.0 \
    eslint-plugin-cypress@^2.15.1 \
    eslint-plugin-storybook@^0.6.15 \
    eslint-plugin-security@^2.1.0 \
    eslint-plugin-unicorn@^50.0.1 \
    eslint-plugin-sonarjs@^0.23.0 \
    eslint-import-resolver-typescript@^3.6.1

# Install Prettier and plugins
info "Installing Prettier and plugins..."
npm install --save-dev \
    prettier@^3.1.1 \
    @trivago/prettier-plugin-sort-imports@^4.3.0

# Install other development tools
info "Installing additional development tools..."
npm install --save-dev \
    husky@^8.0.3 \
    lint-staged@^15.2.0 \
    @commitlint/cli@^18.4.4 \
    @commitlint/config-conventional@^18.4.4

cd "${PROJECT_ROOT}"

# Install Python tools (for pre-commit)
info "Installing Python tools..."

# Check if pip is installed
if ! command -v pip3 &> /dev/null; then
    warn "pip3 is not installed. Skipping Python tools installation."
    warn "To install pre-commit, run: pip3 install pre-commit"
else
    info "Installing pre-commit..."
    pip3 install --user pre-commit
    
    # Install pre-commit hooks
    if command -v pre-commit &> /dev/null; then
        info "Installing pre-commit hooks..."
        pre-commit install
        pre-commit install --hook-type commit-msg
    fi
fi

# Install security scanning tools
info "Installing security scanning tools..."

# Trivy
info "Installing Trivy..."
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
    echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -cs) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
    sudo apt-get update
    sudo apt-get install trivy
elif [[ "$OSTYPE" == "darwin"* ]]; then
    brew install trivy
else
    warn "Please install Trivy manually from https://github.com/aquasecurity/trivy"
fi

# Nancy (for Go dependency scanning)
info "Installing Nancy..."
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    curl -L -o nancy https://github.com/sonatype-nexus-community/nancy/releases/latest/download/nancy-linux.amd64
    chmod +x nancy
    sudo mv nancy /usr/local/bin/
elif [[ "$OSTYPE" == "darwin"* ]]; then
    brew tap sonatype-nexus-community/homebrew-nancy-tap
    brew install nancy
else
    warn "Please install Nancy manually from https://github.com/sonatype-nexus-community/nancy"
fi

# Create git hooks directory if it doesn't exist
mkdir -p "${PROJECT_ROOT}/.git/hooks"

# Setup commit-msg hook for conventional commits
info "Setting up commit-msg hook..."
cat > "${PROJECT_ROOT}/.git/hooks/commit-msg" << 'EOF'
#!/usr/bin/env bash
# Conventional Commits hook

commit_regex='^(feat|fix|docs|style|refactor|perf|test|chore|build|ci|revert)(\(.+\))?: .{1,72}$'
error_msg="Commit message does not follow Conventional Commits format.

Format: <type>(<scope>): <subject>

Example: feat(operator): add new reconciliation logic

Types: feat, fix, docs, style, refactor, perf, test, chore, build, ci, revert"

if ! grep -qE "$commit_regex" "$1"; then
    echo "$error_msg" >&2
    exit 1
fi
EOF

chmod +x "${PROJECT_ROOT}/.git/hooks/commit-msg"

# Create quality check scripts
info "Creating quality check scripts..."

# Create check script
cat > "${PROJECT_ROOT}/hack/check-quality.sh" << 'EOF'
#!/usr/bin/env bash
# Run all code quality checks

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${PROJECT_ROOT}"

echo "Running code quality checks..."

# Go checks
echo "Running Go linting..."
golangci-lint run --timeout=10m

echo "Running Go security scan..."
gosec -fmt=json -out=gosec-report.json ./...

echo "Running Go vulnerability check..."
govulncheck ./...

# TypeScript/React checks
echo "Running ESLint..."
cd ui && npm run lint

echo "Running Prettier check..."
npm run format:check

cd "${PROJECT_ROOT}"

# Security scanning
echo "Running Trivy scan..."
trivy fs --config=.trivy.yaml .

echo "All quality checks passed!"
EOF

chmod +x "${PROJECT_ROOT}/hack/check-quality.sh"

info "Code quality tools installation completed!"
info ""
info "Next steps:"
info "1. Run 'make lint' to check code quality"
info "2. Run 'make fmt' to format code"
info "3. Run 'make security-scan' to run security scans"
info "4. Commit messages must follow Conventional Commits format"
info ""
info "Pre-commit hooks have been installed and will run automatically."
