# Makefile for Gunj Operator
# Supports multi-architecture builds and comprehensive testing
# Version: 2.0

# Build variables
REGISTRY ?= docker.io
REGISTRY_USER ?= gunjanjp
VERSION ?= $(shell git describe --tags --always --dirty)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go variables
GO ?= go
GOOS ?= linux
GOARCH ?= amd64
GOARM ?=
CGO_ENABLED ?= 0
GOFLAGS ?= -mod=readonly

# Components
COMPONENTS := operator api cli ui
GO_COMPONENTS := operator api cli

# Architectures
ALL_ARCHITECTURES := linux/amd64 linux/arm64 linux/arm/v7 darwin/amd64 darwin/arm64 windows/amd64
LINUX_ARCHITECTURES := linux/amd64 linux/arm64 linux/arm/v7
CONTAINER_ARCHITECTURES := linux/amd64 linux/arm64 linux/arm/v7

# Build settings
LDFLAGS := -w -s \
	-X main.version=$(VERSION) \
	-X main.gitCommit=$(GIT_COMMIT) \
	-X main.buildDate=$(BUILD_DATE)

# Docker settings
DOCKER := docker
DOCKERFILE_OPERATOR := Dockerfile
DOCKERFILE_API := Dockerfile.api
DOCKERFILE_CLI := Dockerfile.cli
DOCKERFILE_UI := Dockerfile.ui

# Tools
TOOLS_DIR := $(shell pwd)/bin
CONTROLLER_GEN := $(TOOLS_DIR)/controller-gen
KUSTOMIZE := $(TOOLS_DIR)/kustomize
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint
KIND := $(TOOLS_DIR)/kind
GINKGO := $(TOOLS_DIR)/ginkgo

# Directories
OUTPUT_DIR := dist
COVERAGE_DIR := coverage
TEST_RESULTS_DIR := test-results

.PHONY: all
all: test build ## Run tests and build all components

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: install-deps
install-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	@mkdir -p $(TOOLS_DIR)
	@GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	@GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/kustomize/kustomize/v5@latest
	@GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/kind@latest
	@GOBIN=$(TOOLS_DIR) go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@cd ui && npm ci

.PHONY: setup
setup: install-deps ## Complete development environment setup
	@echo "Setting up development environment..."
	@./scripts/setup-dev-env.sh

##@ Build

.PHONY: build
build: build-operator build-api build-cli build-ui ## Build all components

.PHONY: build-operator
build-operator: generate ## Build operator binary
	@echo "Building operator..."
	@mkdir -p $(OUTPUT_DIR)/operator/$(GOOS)/$(GOARCH)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/operator/$(GOOS)/$(GOARCH)/gunj-operator ./cmd/operator

.PHONY: build-api
build-api: ## Build API server binary
	@echo "Building API server..."
	@mkdir -p $(OUTPUT_DIR)/api/$(GOOS)/$(GOARCH)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/api/$(GOOS)/$(GOARCH)/gunj-api-server ./cmd/api-server

.PHONY: build-cli
build-cli: ## Build CLI binary
	@echo "Building CLI..."
	@mkdir -p $(OUTPUT_DIR)/cli/$(GOOS)/$(GOARCH)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/cli/$(GOOS)/$(GOARCH)/gunj-cli ./cmd/cli

.PHONY: build-ui
build-ui: ## Build UI
	@echo "Building UI..."
	@cd ui && npm run build

##@ Multi-Architecture Build

.PHONY: build-all-arch
build-all-arch: ## Build all components for all architectures
	@echo "Building all components for all architectures..."
	@./scripts/build-multiarch.sh -c operator,api,cli,ui -a amd64,arm64,arm/v7

.PHONY: build-linux
build-linux: ## Build all components for Linux architectures
	@for arch in amd64 arm64 arm/v7; do \
		echo "Building for linux/$$arch..."; \
		$(MAKE) build GOOS=linux GOARCH=$${arch%/*} GOARM=$${arch##*/}; \
	done

.PHONY: docker-build
docker-build: docker-build-operator docker-build-api docker-build-ui ## Build all Docker images

.PHONY: docker-build-%
docker-build-%: ## Build Docker image for specific component
	@echo "Building Docker image for $*..."
	@$(DOCKER) buildx build \
		--platform $(CONTAINER_ARCHITECTURES) \
		--tag $(REGISTRY)/$(REGISTRY_USER)/gunj-$*:$(VERSION) \
		--tag $(REGISTRY)/$(REGISTRY_USER)/gunj-$*:latest \
		--file $(DOCKERFILE_$(shell echo $* | tr a-z A-Z)) \
		.

.PHONY: docker-build-all-arch
docker-build-all-arch: ## Build all images for all architectures
	@for component in $(COMPONENTS); do \
		echo "Building $$component for all architectures..."; \
		$(DOCKER) buildx build \
			--platform $(shell echo $(CONTAINER_ARCHITECTURES) | tr ' ' ',') \
			--tag $(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:$(VERSION) \
			--tag $(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:latest \
			--file Dockerfile.$$component \
			--push \
			.; \
	done

.PHONY: docker-push
docker-push: ## Push all Docker images
	@for component in $(COMPONENTS); do \
		for arch in amd64 arm64 arm-v7; do \
			echo "Pushing $$component:$(VERSION)-$$arch..."; \
			$(DOCKER) push $(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:$(VERSION)-$$arch; \
		done; \
		echo "Creating and pushing manifest for $$component..."; \
		$(DOCKER) manifest create $(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:$(VERSION) \
			$(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:$(VERSION)-amd64 \
			$(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:$(VERSION)-arm64 \
			$(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:$(VERSION)-arm-v7; \
		$(DOCKER) manifest push $(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:$(VERSION); \
	done

##@ Testing

.PHONY: test
test: test-unit test-integration ## Run all tests

.PHONY: test-unit
test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GO) test -v -coverprofile=$(COVERAGE_DIR)/unit.out ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@$(GINKGO) -v --label-filter="integration" ./test/...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@echo "Running e2e tests..."
	@$(GINKGO) -v ./test/e2e/...

.PHONY: test-arch
test-arch: ## Test on all architectures
	@echo "Running architecture-specific tests..."
	@for arch in $(ALL_ARCHITECTURES); do \
		echo "Testing on $$arch..."; \
		GOOS=$${arch%/*} GOARCH=$${arch#*/} $(GO) test -v ./...; \
	done

.PHONY: test-operator-arch
test-operator-arch: ## Test operator on specific architecture
	@echo "Testing operator on $(ARCH)..."
	@GOOS=$(shell echo $(ARCH) | cut -d'/' -f1) \
	 GOARCH=$(shell echo $(ARCH) | cut -d'/' -f2) \
	 $(GO) test -v ./controllers/...

.PHONY: benchmark-arch
benchmark-arch: ## Run benchmarks for specific architecture
	@echo "Running benchmarks on $(ARCH)..."
	@GOOS=$(shell echo $(ARCH) | cut -d'/' -f1) \
	 GOARCH=$(shell echo $(ARCH) | cut -d'/' -f2) \
	 $(GO) test -bench=. -benchmem ./...

##@ Code Generation

.PHONY: generate
generate: controller-gen ## Generate code
	@echo "Generating code..."
	@$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen ## Generate manifests
	@echo "Generating manifests..."
	@$(CONTROLLER_GEN) crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

##@ Linting

.PHONY: lint
lint: lint-go lint-ui ## Run all linters

.PHONY: lint-go
lint-go: golangci-lint ## Run Go linter
	@echo "Linting Go code..."
	@$(GOLANGCI_LINT) run --timeout=10m

.PHONY: lint-ui
lint-ui: ## Run UI linter
	@echo "Linting UI code..."
	@cd ui && npm run lint

##@ Local Development

.PHONY: run
run: generate ## Run operator locally
	@echo "Running operator locally..."
	@$(GO) run ./cmd/operator --config=config/dev/config.yaml

.PHONY: kind-create
kind-create: kind ## Create kind cluster
	@echo "Creating kind cluster..."
	@$(KIND) create cluster --config=test/kind-config.yaml

.PHONY: kind-load
kind-load: docker-build ## Load images into kind
	@echo "Loading images into kind..."
	@for component in $(COMPONENTS); do \
		$(KIND) load docker-image $(REGISTRY)/$(REGISTRY_USER)/gunj-$$component:$(VERSION); \
	done

.PHONY: deploy-local
deploy-local: kind-load ## Deploy to local kind cluster
	@echo "Deploying to kind cluster..."
	@kubectl apply -k config/default

##@ Tools

.PHONY: controller-gen
controller-gen: ## Install controller-gen
	@test -s $(CONTROLLER_GEN) || GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

.PHONY: kustomize
kustomize: ## Install kustomize
	@test -s $(KUSTOMIZE) || GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/kustomize/kustomize/v5@latest

.PHONY: golangci-lint
golangci-lint: ## Install golangci-lint
	@test -s $(GOLANGCI_LINT) || GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: kind
kind: ## Install kind
	@test -s $(KIND) || GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/kind@latest

.PHONY: ginkgo
ginkgo: ## Install ginkgo
	@test -s $(GINKGO) || GOBIN=$(TOOLS_DIR) go install github.com/onsi/ginkgo/v2/ginkgo@latest

##@ Cleanup

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUTPUT_DIR) $(COVERAGE_DIR) $(TEST_RESULTS_DIR)
	@rm -rf ui/dist ui/node_modules
	@$(DOCKER) system prune -f

.PHONY: kind-delete
kind-delete: ## Delete kind cluster
	@echo "Deleting kind cluster..."
	@$(KIND) delete cluster

##@ Release

.PHONY: release
release: ## Create a new release
	@echo "Creating release $(VERSION)..."
	@./scripts/release.sh $(VERSION)

.PHONY: changelog
changelog: ## Generate changelog
	@echo "Generating changelog..."
	@./scripts/generate-changelog.sh
