# Kubebuilder Setup Guide for Gunj Operator

## Prerequisites

1. **Go Installation**
   ```bash
   # Install Go 1.21+
   wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin
   ```

2. **Kubebuilder Installation**
   ```bash
   # Download and install Kubebuilder
   curl -L -o kubebuilder "https://go.kubebuilder.io/dl/v3.14.0/$(go env GOOS)/$(go env GOARCH)"
   chmod +x kubebuilder
   sudo mv kubebuilder /usr/local/bin/
   ```

3. **Required Tools**
   ```bash
   # Install controller-gen
   go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
   
   # Install kustomize
   curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
   sudo mv kustomize /usr/local/bin/
   
   # Install setup-envtest
   go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
   ```

## Project Initialization

```bash
# Create project directory
mkdir -p gunj-operator
cd gunj-operator

# Initialize Kubebuilder project
kubebuilder init \
  --domain observability.io \
  --repo github.com/gunjanjp/gunj-operator \
  --project-name gunj-operator \
  --component-config \
  --license mit \
  --owner "Gunjan JP"

# Create the main API
kubebuilder create api \
  --group observability \
  --version v1beta1 \
  --kind ObservabilityPlatform \
  --resource \
  --controller

# Add webhooks
kubebuilder create webhook \
  --group observability \
  --version v1beta1 \
  --kind ObservabilityPlatform \
  --defaulting \
  --validation \
  --conversion
```

## Project Structure Overview

```
gunj-operator/
├── Dockerfile                 # Multi-stage build for operator
├── Makefile                  # Build automation
├── PROJECT                   # Kubebuilder project metadata
├── api/
│   └── v1beta1/
│       ├── groupversion_info.go
│       ├── observabilityplatform_types.go
│       ├── observabilityplatform_webhook.go
│       └── zz_generated.deepcopy.go
├── bin/                      # Build artifacts
├── cmd/
│   └── main.go              # Operator entry point
├── config/                   # Kustomize configurations
│   ├── crd/                 # CRD manifests
│   ├── default/             # Default Kustomization
│   ├── manager/             # Controller manager configs
│   ├── prometheus/          # ServiceMonitor for metrics
│   ├── rbac/               # RBAC manifests
│   └── webhook/            # Webhook configurations
├── hack/
│   └── boilerplate.go.txt   # License header
├── internal/
│   └── controller/
│       ├── observabilityplatform_controller.go
│       └── suite_test.go
└── go.mod                   # Go dependencies
```

## Key Kubebuilder Features We'll Use

1. **Multi-Version API Support**
   - v1alpha1 → v1beta1 → v1 progression
   - Conversion webhooks for seamless upgrades

2. **Advanced Webhooks**
   - Defaulting webhooks for optional fields
   - Validation webhooks for business logic
   - Conversion webhooks for API evolution

3. **Controller Patterns**
   - Finalizers for cleanup logic
   - Owner references for garbage collection
   - Status subresource for condition reporting

4. **Testing Framework**
   - Envtest for integration testing
   - Ginkgo/Gomega for BDD-style tests
   - Controller-runtime fake client for unit tests

5. **Metrics & Monitoring**
   - Prometheus metrics auto-exposed
   - Custom metrics via controller-runtime
   - Health/readiness probes

## Next Steps

1. Set up the development environment
2. Define the CRD schema
3. Implement the controller logic
4. Add comprehensive tests
5. Create webhook implementations
