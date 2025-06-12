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
