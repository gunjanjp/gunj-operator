# Gunj Operator - Makefile
# Version: v2.0
# Purpose: Build automation and linting for the Gunj Operator project

# Variables
SHELL := /bin/bash
.DEFAULT_GOAL := help

# Go variables
GOCMD := go
GOMOD := $(GOCMD) mod
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOIMPORTS := goimports
GOLANGCI_LINT := golangci-lint
GOLANGCI_VERSION := v1.55.2

# Project variables
PROJECT_NAME := gunj-operator
OPERATOR_IMG ?= gunj-operator:latest
API_IMG ?= gunj-api:latest
UI_IMG ?= gunj-ui:latest
REGISTRY ?= docker.io/gunjanjp
VERSION ?= $(shell git describe --tags --always --dirty)

# Directories
ROOT_DIR := $(shell pwd)
OPERATOR_DIR := $(ROOT_DIR)
UI_DIR := $(ROOT_DIR)/ui
API_DIR := $(ROOT_DIR)/internal/api
DOCS_DIR := $(ROOT_DIR)/docs

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Linting

.PHONY: lint
lint: lint-go lint-ui lint-yaml lint-docker lint-markdown lint-shell ## Run all linters

.PHONY: lint-go
lint-go: golangci-lint ## Run Go linters
	@echo -e "$(BLUE)Running Go linters...$(NC)"
	@$(GOLANGCI_LINT) run --config .golangci.yml ./...
	@echo -e "$(GREEN)✓ Go linting completed$(NC)"

.PHONY: lint-go-fix
lint-go-fix: golangci-lint ## Run Go linters and fix issues
	@echo -e "$(BLUE)Running Go linters with fixes...$(NC)"
	@$(GOLANGCI_LINT) run --config .golangci.yml --fix ./...
	@echo -e "$(GREEN)✓ Go linting and fixes completed$(NC)"

.PHONY: lint-ui
lint-ui: ## Run UI (TypeScript/React) linters
	@echo -e "$(BLUE)Running UI linters...$(NC)"
	@cd $(UI_DIR) && npm run lint
	@echo -e "$(GREEN)✓ UI linting completed$(NC)"

.PHONY: lint-ui-fix
lint-ui-fix: ## Run UI linters and fix issues
	@echo -e "$(BLUE)Running UI linters with fixes...$(NC)"
	@cd $(UI_DIR) && npm run lint:fix
	@echo -e "$(GREEN)✓ UI linting and fixes completed$(NC)"

.PHONY: lint-yaml
lint-yaml: yamllint ## Run YAML linter
	@echo -e "$(BLUE)Running YAML linter...$(NC)"
	@yamllint -c .yamllint.yml .
	@echo -e "$(GREEN)✓ YAML linting completed$(NC)"

.PHONY: lint-docker
lint-docker: hadolint ## Run Dockerfile linter
	@echo -e "$(BLUE)Running Dockerfile linter...$(NC)"
	@find . -name "Dockerfile*" -not -path "./vendor/*" -not -path "./node_modules/*" | \
		xargs hadolint --config .hadolint.yaml
	@echo -e "$(GREEN)✓ Dockerfile linting completed$(NC)"

.PHONY: lint-markdown
lint-markdown: markdownlint ## Run Markdown linter
	@echo -e "$(BLUE)Running Markdown linter...$(NC)"
	@markdownlint '**/*.md' --config .markdownlint.json --ignore node_modules --ignore vendor
	@echo -e "$(GREEN)✓ Markdown linting completed$(NC)"

.PHONY: lint-markdown-fix
lint-markdown-fix: markdownlint ## Run Markdown linter and fix issues
	@echo -e "$(BLUE)Running Markdown linter with fixes...$(NC)"
	@markdownlint '**/*.md' --config .markdownlint.json --ignore node_modules --ignore vendor --fix
	@echo -e "$(GREEN)✓ Markdown linting and fixes completed$(NC)"

.PHONY: lint-shell
lint-shell: shellcheck ## Run shell script linter
	@echo -e "$(BLUE)Running shell script linter...$(NC)"
	@find . -name "*.sh" -not -path "./vendor/*" -not -path "./node_modules/*" | \
		xargs shellcheck -x
	@echo -e "$(GREEN)✓ Shell script linting completed$(NC)"

##@ Code Formatting

.PHONY: fmt
fmt: fmt-go fmt-ui ## Format all code

.PHONY: fmt-go
fmt-go: ## Format Go code
	@echo -e "$(BLUE)Formatting Go code...$(NC)"
	@$(GOFMT) -s -w .
	@$(GOIMPORTS) -w -local github.com/gunjanjp/gunj-operator .
	@echo -e "$(GREEN)✓ Go formatting completed$(NC)"

.PHONY: fmt-ui
fmt-ui: ## Format UI code with Prettier
	@echo -e "$(BLUE)Formatting UI code...$(NC)"
	@cd $(UI_DIR) && npm run format
	@echo -e "$(GREEN)✓ UI formatting completed$(NC)"

.PHONY: fmt-check
fmt-check: fmt-check-go fmt-check-ui ## Check code formatting

.PHONY: fmt-check-go
fmt-check-go: ## Check Go code formatting
	@echo -e "$(BLUE)Checking Go code formatting...$(NC)"
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo -e "$(RED)✗ Go code needs formatting. Run 'make fmt-go'$(NC)"; \
		$(GOFMT) -l .; \
		exit 1; \
	fi
	@echo -e "$(GREEN)✓ Go code formatting check passed$(NC)"

.PHONY: fmt-check-ui
fmt-check-ui: ## Check UI code formatting
	@echo -e "$(BLUE)Checking UI code formatting...$(NC)"
	@cd $(UI_DIR) && npm run format:check
	@echo -e "$(GREEN)✓ UI code formatting check passed$(NC)"

##@ Security

.PHONY: sec
sec: sec-go sec-ui sec-docker ## Run all security checks

.PHONY: sec-go
sec-go: ## Run Go security checks
	@echo -e "$(BLUE)Running Go security checks...$(NC)"
	@gosec -fmt=json -out=gosec-report.json -stdout ./...
	@echo -e "$(GREEN)✓ Go security check completed$(NC)"

.PHONY: sec-ui
sec-ui: ## Run UI security audit
	@echo -e "$(BLUE)Running UI security audit...$(NC)"
	@cd $(UI_DIR) && npm audit
	@echo -e "$(GREEN)✓ UI security audit completed$(NC)"

.PHONY: sec-docker
sec-docker: ## Run Docker image security scan
	@echo -e "$(BLUE)Running Docker security scan...$(NC)"
	@trivy image --severity HIGH,CRITICAL $(OPERATOR_IMG)
	@echo -e "$(GREEN)✓ Docker security scan completed$(NC)"

##@ Dependencies

.PHONY: deps
deps: deps-go deps-ui deps-tools ## Install all dependencies

.PHONY: deps-go
deps-go: ## Download Go dependencies
	@echo -e "$(BLUE)Downloading Go dependencies...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo -e "$(GREEN)✓ Go dependencies installed$(NC)"

.PHONY: deps-ui
deps-ui: ## Install UI dependencies
	@echo -e "$(BLUE)Installing UI dependencies...$(NC)"
	@cd $(UI_DIR) && npm ci
	@echo -e "$(GREEN)✓ UI dependencies installed$(NC)"

.PHONY: deps-tools
deps-tools: golangci-lint yamllint hadolint markdownlint shellcheck ## Install linting tools

##@ Tool Installation

.PHONY: golangci-lint
golangci-lint: ## Install golangci-lint
	@if ! command -v golangci-lint &> /dev/null; then \
		echo -e "$(BLUE)Installing golangci-lint $(GOLANGCI_VERSION)...$(NC)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
			sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_VERSION); \
		echo -e "$(GREEN)✓ golangci-lint installed$(NC)"; \
	else \
		echo -e "$(GREEN)✓ golangci-lint already installed$(NC)"; \
	fi

.PHONY: yamllint
yamllint: ## Install yamllint
	@if ! command -v yamllint &> /dev/null; then \
		echo -e "$(BLUE)Installing yamllint...$(NC)"; \
		pip install --user yamllint; \
		echo -e "$(GREEN)✓ yamllint installed$(NC)"; \
	else \
		echo -e "$(GREEN)✓ yamllint already installed$(NC)"; \
	fi

.PHONY: hadolint
hadolint: ## Install hadolint
	@if ! command -v hadolint &> /dev/null; then \
		echo -e "$(BLUE)Installing hadolint...$(NC)"; \
		curl -sSfL https://github.com/hadolint/hadolint/releases/latest/download/hadolint-Linux-x86_64 \
			-o /usr/local/bin/hadolint && chmod +x /usr/local/bin/hadolint; \
		echo -e "$(GREEN)✓ hadolint installed$(NC)"; \
	else \
		echo -e "$(GREEN)✓ hadolint already installed$(NC)"; \
	fi

.PHONY: markdownlint
markdownlint: ## Install markdownlint
	@if ! command -v markdownlint &> /dev/null; then \
		echo -e "$(BLUE)Installing markdownlint...$(NC)"; \
		npm install -g markdownlint-cli; \
		echo -e "$(GREEN)✓ markdownlint installed$(NC)"; \
	else \
		echo -e "$(GREEN)✓ markdownlint already installed$(NC)"; \
	fi

.PHONY: shellcheck
shellcheck: ## Install shellcheck
	@if ! command -v shellcheck &> /dev/null; then \
		echo -e "$(BLUE)Installing shellcheck...$(NC)"; \
		apt-get update && apt-get install -y shellcheck || \
			brew install shellcheck || \
			echo -e "$(RED)Please install shellcheck manually$(NC)"; \
		echo -e "$(GREEN)✓ shellcheck installed$(NC)"; \
	else \
		echo -e "$(GREEN)✓ shellcheck already installed$(NC)"; \
	fi

##@ Pre-commit Hooks

.PHONY: pre-commit-install
pre-commit-install: ## Install pre-commit hooks
	@echo -e "$(BLUE)Installing pre-commit hooks...$(NC)"
	@pre-commit install
	@pre-commit install --hook-type commit-msg
	@echo -e "$(GREEN)✓ Pre-commit hooks installed$(NC)"

.PHONY: pre-commit-run
pre-commit-run: ## Run pre-commit on all files
	@echo -e "$(BLUE)Running pre-commit on all files...$(NC)"
	@pre-commit run --all-files
	@echo -e "$(GREEN)✓ Pre-commit checks completed$(NC)"

##@ CI/CD

.PHONY: ci
ci: fmt-check lint sec ## Run all CI checks

.PHONY: ci-fix
ci-fix: fmt lint-fix ## Fix all auto-fixable issues

##@ Clean

.PHONY: clean
clean: ## Clean build artifacts and caches
	@echo -e "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf bin/ dist/ _output/ coverage/ .cache/
	@rm -f gosec-report.json
	@find . -name "*.test" -type f -delete
	@find . -name "*.out" -type f -delete
	@echo -e "$(GREEN)✓ Cleanup completed$(NC)"

.PHONY: clean-deps
clean-deps: ## Clean dependency caches
	@echo -e "$(BLUE)Cleaning dependency caches...$(NC)"
	@$(GOMOD) clean -modcache
	@cd $(UI_DIR) && rm -rf node_modules
	@echo -e "$(GREEN)✓ Dependency cleanup completed$(NC)"

##@ Utilities

.PHONY: check-tools
check-tools: ## Check if all required tools are installed
	@echo -e "$(BLUE)Checking required tools...$(NC)"
	@command -v go >/dev/null 2>&1 || { echo -e "$(RED)✗ go is not installed$(NC)"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo -e "$(RED)✗ node is not installed$(NC)"; exit 1; }
	@command -v npm >/dev/null 2>&1 || { echo -e "$(RED)✗ npm is not installed$(NC)"; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo -e "$(RED)✗ docker is not installed$(NC)"; exit 1; }
	@command -v kubectl >/dev/null 2>&1 || { echo -e "$(RED)✗ kubectl is not installed$(NC)"; exit 1; }
	@command -v helm >/dev/null 2>&1 || { echo -e "$(RED)✗ helm is not installed$(NC)"; exit 1; }
	@echo -e "$(GREEN)✓ All required tools are installed$(NC)"

.PHONY: version
version: ## Display project version
	@echo -e "$(BLUE)Gunj Operator $(VERSION)$(NC)"
