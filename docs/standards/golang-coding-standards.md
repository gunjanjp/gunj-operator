# Golang Coding Standards - Gunj Operator

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Official  
**Applies To**: All Go code in the Gunj Operator project  

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Code Organization](#code-organization)
3. [Naming Conventions](#naming-conventions)
4. [Code Formatting](#code-formatting)
5. [Programming Practices](#programming-practices)
6. [Error Handling](#error-handling)
7. [Testing Standards](#testing-standards)
8. [Documentation](#documentation)
9. [Performance Guidelines](#performance-guidelines)
10. [Security Guidelines](#security-guidelines)
11. [Kubernetes Specific Guidelines](#kubernetes-specific-guidelines)
12. [Tools and Enforcement](#tools-and-enforcement)

---

## Overview

This document establishes the Golang coding standards for the Gunj Operator project. These standards ensure consistency, maintainability, and quality across all Go code in the project.

### Core Principles

1. **Clarity over Cleverness**: Write code that is easy to understand
2. **Consistency**: Follow established patterns throughout the codebase
3. **Simplicity**: Prefer simple solutions over complex ones
4. **Testability**: Write code that is easy to test
5. **Performance**: Consider performance implications but don't optimize prematurely

---

## Code Organization

### Package Structure

```go
// Package gunj provides the operator for managing observability platforms
// 
// The package implements a Kubernetes operator that manages the lifecycle of
// observability components including Prometheus, Grafana, Loki, and Tempo.
package gunj
```

### Import Organization

Imports must be organized in the following groups, separated by blank lines:

```go
import (
    // Standard library imports
    "context"
    "fmt"
    "time"
    
    // Third-party imports
    "github.com/go-logr/logr"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    
    // Internal imports
    observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
    "github.com/gunjanjp/gunj-operator/internal/utils"
)
```

### File Organization

Each file should contain:
1. Package declaration with documentation
2. Imports (organized as above)
3. Constants
4. Variables
5. Types (interfaces first, then structs)
6. Constructor/Factory functions
7. Methods (receiver methods grouped by type)
8. Functions (exported first, then unexported)

---

## Naming Conventions

### General Rules

- Use `camelCase` for variables and functions
- Use `PascalCase` for exported types, functions, and constants
- Use `ALL_CAPS` with underscores for environment variables
- Avoid abbreviations unless widely recognized (e.g., `URL`, `ID`)
- Be descriptive but concise

### Specific Conventions

#### Variables
```go
// Good
var reconcilerOptions ReconcilerOptions
var maxRetries int
var isReady bool

// Bad
var ro ReconcilerOptions  // Too abbreviated
var maximum_retries int   // Use camelCase, not snake_case
var ready bool           // Not descriptive enough for package-level
```

#### Constants
```go
// Good
const (
    // DefaultReconcileTimeout is the default timeout for reconciliation
    DefaultReconcileTimeout = 2 * time.Minute
    
    // MaxRetries defines the maximum number of retry attempts
    MaxRetries = 5
)

// Bad
const TIMEOUT = 120  // Use descriptive names and proper case
```

#### Functions and Methods
```go
// Good
func (r *Reconciler) ReconcilePlatform(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
func NewReconciler(client client.Client, scheme *runtime.Scheme) *Reconciler
func validatePlatformSpec(spec *v1beta1.ObservabilityPlatformSpec) error

// Bad
func (r *Reconciler) Reconcile(ctx context.Context, p *v1beta1.ObservabilityPlatform) error  // Too generic
func MakeReconciler() *Reconciler  // Use "New" prefix for constructors
```

#### Interfaces
```go
// Good
type PlatformManager interface {
    Deploy(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
    Update(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
    Delete(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
}

// Bad
type Manager interface {  // Too generic
    DoStuff()            // Not descriptive
}
```

---

## Code Formatting

### Standard Formatting

All code MUST be formatted using `gofmt` and `goimports`:

```bash
gofmt -w .
goimports -w .
```

### Line Length

- Prefer lines under 120 characters
- Break long function calls at appropriate points:

```go
// Good
deployment := r.buildDeployment(
    platform,
    component,
    r.getLabels(platform),
    r.getAnnotations(platform),
)

// Bad
deployment := r.buildDeployment(platform, component, r.getLabels(platform), r.getAnnotations(platform))
```

### Comments

#### Package Comments
```go
// Package controllers implements the Kubernetes controllers for the Gunj Operator.
// It provides reconciliation logic for ObservabilityPlatform resources and manages
// the lifecycle of observability components.
package controllers
```

#### Function Comments
```go
// ReconcilePlatform ensures the desired state of an ObservabilityPlatform.
// It creates, updates, or deletes the necessary Kubernetes resources to match
// the specification defined in the ObservabilityPlatform CR.
//
// The function returns an error if reconciliation fails, which will trigger
// a retry with exponential backoff.
func (r *Reconciler) ReconcilePlatform(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
    // Implementation
}
```

#### Inline Comments
```go
// Good
// Calculate the resource requirements based on the platform size
resources := r.calculateResources(platform.Spec.Size)

// Bad
resources := r.calculateResources(platform.Spec.Size) // calculate resources
```

---

## Programming Practices

### Variable Declaration

```go
// Prefer short declaration for local variables
isReady := checkReadiness(deployment)

// Use var for package-level or when zero value is needed
var (
    defaultTimeout = 30 * time.Second
    cache          = make(map[string]*Deployment)
)

// Group related declarations
var (
    ErrInvalidPlatform = errors.New("invalid platform specification")
    ErrTimeout         = errors.New("operation timed out")
    ErrNotReady        = errors.New("component not ready")
)
```

### Error Handling

#### Error Definition
```go
// Define sentinel errors at package level
var (
    ErrPlatformNotFound      = errors.New("platform not found")
    ErrInvalidConfiguration  = errors.New("invalid configuration")
    ErrReconciliationFailed  = errors.New("reconciliation failed")
)

// Use error wrapping for context
func (r *Reconciler) deployPrometheus(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
    deployment := r.buildPrometheusDeployment(platform)
    
    if err := r.Create(ctx, deployment); err != nil {
        return fmt.Errorf("creating Prometheus deployment: %w", err)
    }
    
    return nil
}
```

#### Error Checking
```go
// Always check errors immediately
result, err := someOperation()
if err != nil {
    return fmt.Errorf("performing operation: %w", err)
}

// Handle specific errors
if err := r.Client.Get(ctx, key, platform); err != nil {
    if errors.IsNotFound(err) {
        // Handle not found case
        return ctrl.Result{}, nil
    }
    // Handle other errors
    return ctrl.Result{}, fmt.Errorf("getting platform: %w", err)
}
```

### Context Usage

Always pass context as the first parameter:

```go
// Good
func (r *Reconciler) deployComponent(ctx context.Context, component Component) error

// Bad
func (r *Reconciler) deployComponent(component Component, ctx context.Context) error
```

### Defer Usage

```go
// Good - Simple cleanup
func processFile(filename string) error {
    f, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer f.Close()
    
    // Process file
    return nil
}

// Good - Multiple defers execute in LIFO order
func (r *Reconciler) withLock(fn func() error) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    start := time.Now()
    defer func() {
        r.metrics.RecordDuration(time.Since(start))
    }()
    
    return fn()
}
```

### Interface Design

```go
// Good - Small, focused interfaces
type Deployer interface {
    Deploy(ctx context.Context, obj runtime.Object) error
}

type Updater interface {
    Update(ctx context.Context, obj runtime.Object) error
}

// Can be composed
type Manager interface {
    Deployer
    Updater
    Delete(ctx context.Context, obj runtime.Object) error
}
```

---

## Testing Standards

### Test File Organization

```go
package controllers_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "sigs.k8s.io/controller-runtime/pkg/client/fake"
    
    observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
    "github.com/gunjanjp/gunj-operator/controllers"
)
```

### Table-Driven Tests

```go
func TestReconcilePlatform(t *testing.T) {
    tests := []struct {
        name      string
        platform  *v1beta1.ObservabilityPlatform
        objects   []runtime.Object
        wantErr   bool
        wantPhase string
        validate  func(t *testing.T, client client.Client)
    }{
        {
            name: "successful deployment",
            platform: &v1beta1.ObservabilityPlatform{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-platform",
                    Namespace: "default",
                },
                Spec: v1beta1.ObservabilityPlatformSpec{
                    Components: v1beta1.Components{
                        Prometheus: &v1beta1.PrometheusSpec{
                            Enabled: true,
                            Version: "v2.45.0",
                        },
                    },
                },
            },
            wantErr:   false,
            wantPhase: "Ready",
            validate: func(t *testing.T, c client.Client) {
                // Additional validation
                deployment := &appsv1.Deployment{}
                err := c.Get(context.Background(), types.NamespacedName{
                    Name:      "test-platform-prometheus",
                    Namespace: "default",
                }, deployment)
                require.NoError(t, err)
                assert.Equal(t, int32(1), *deployment.Spec.Replicas)
            },
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            ctx := context.Background()
            scheme := runtime.NewScheme()
            _ = observabilityv1.AddToScheme(scheme)
            
            objs := append(tt.objects, tt.platform)
            client := fake.NewClientBuilder().
                WithScheme(scheme).
                WithRuntimeObjects(objs...).
                Build()
            
            reconciler := controllers.NewReconciler(client, scheme)
            
            // Execute
            err := reconciler.ReconcilePlatform(ctx, tt.platform)
            
            // Assert
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            
            // Validate
            if tt.validate != nil {
                tt.validate(t, client)
            }
        })
    }
}
```

### Test Helpers

```go
// testutil/helpers.go
package testutil

// NewTestPlatform creates a test ObservabilityPlatform with defaults
func NewTestPlatform(name, namespace string, opts ...PlatformOption) *v1beta1.ObservabilityPlatform {
    platform := &v1beta1.ObservabilityPlatform{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: namespace,
        },
        Spec: v1beta1.ObservabilityPlatformSpec{
            Components: v1beta1.Components{},
        },
    }
    
    for _, opt := range opts {
        opt(platform)
    }
    
    return platform
}

type PlatformOption func(*v1beta1.ObservabilityPlatform)

func WithPrometheus(enabled bool) PlatformOption {
    return func(p *v1beta1.ObservabilityPlatform) {
        p.Spec.Components.Prometheus = &v1beta1.PrometheusSpec{
            Enabled: enabled,
            Version: "v2.45.0",
        }
    }
}
```

### Benchmark Tests

```go
func BenchmarkReconcile(b *testing.B) {
    ctx := context.Background()
    scheme := runtime.NewScheme()
    _ = observabilityv1.AddToScheme(scheme)
    
    platform := testutil.NewTestPlatform("bench", "default",
        testutil.WithPrometheus(true),
        testutil.WithGrafana(true),
    )
    
    client := fake.NewClientBuilder().
        WithScheme(scheme).
        WithRuntimeObjects(platform).
        Build()
    
    reconciler := controllers.NewReconciler(client, scheme)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = reconciler.ReconcilePlatform(ctx, platform)
    }
}
```

---

## Documentation

### Code Documentation Rules

1. **All exported types, functions, and methods MUST have documentation**
2. **Documentation should start with the name being documented**
3. **Use complete sentences with proper punctuation**
4. **Include examples for complex functionality**

```go
// ObservabilityPlatformReconciler reconciles a ObservabilityPlatform object.
// It manages the lifecycle of all observability components defined in the platform spec.
type ObservabilityPlatformReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Log    logr.Logger
}

// Reconcile implements the main reconciliation loop for ObservabilityPlatform resources.
// It ensures that the actual state of the cluster matches the desired state defined
// in the ObservabilityPlatform spec.
//
// The function performs the following steps:
//   1. Fetches the ObservabilityPlatform resource
//   2. Handles deletion if marked for deletion
//   3. Reconciles each enabled component
//   4. Updates the status with the current state
//
// Example:
//     result, err := r.Reconcile(ctx, ctrl.Request{
//         NamespacedName: types.NamespacedName{
//             Name:      "my-platform",
//             Namespace: "monitoring",
//         },
//     })
func (r *ObservabilityPlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Implementation
}
```

---

## Performance Guidelines

### Memory Management

```go
// Reuse slices when possible
func (r *Reconciler) processItems(items []Item) {
    // Good - Pre-allocate with capacity
    results := make([]Result, 0, len(items))
    
    for _, item := range items {
        results = append(results, r.process(item))
    }
}

// Use sync.Pool for frequently allocated objects
var deploymentPool = sync.Pool{
    New: func() interface{} {
        return &appsv1.Deployment{}
    },
}

func (r *Reconciler) getDeployment() *appsv1.Deployment {
    return deploymentPool.Get().(*appsv1.Deployment)
}

func (r *Reconciler) putDeployment(d *appsv1.Deployment) {
    // Reset the deployment
    *d = appsv1.Deployment{}
    deploymentPool.Put(d)
}
```

### Concurrency

```go
// Use goroutines with proper synchronization
func (r *Reconciler) reconcileComponents(ctx context.Context, components []Component) error {
    g, ctx := errgroup.WithContext(ctx)
    
    // Limit concurrent operations
    sem := make(chan struct{}, 5)
    
    for _, comp := range components {
        comp := comp // Capture loop variable
        g.Go(func() error {
            sem <- struct{}{}
            defer func() { <-sem }()
            
            return r.reconcileComponent(ctx, comp)
        })
    }
    
    return g.Wait()
}
```

---

## Security Guidelines

### Input Validation

```go
// Always validate input
func (r *Reconciler) validatePlatformSpec(spec *v1beta1.ObservabilityPlatformSpec) error {
    if spec == nil {
        return fmt.Errorf("spec cannot be nil")
    }
    
    // Validate component versions against allowed list
    if spec.Components.Prometheus != nil {
        if !isValidVersion(spec.Components.Prometheus.Version) {
            return fmt.Errorf("invalid Prometheus version: %s", spec.Components.Prometheus.Version)
        }
    }
    
    // Validate resource requirements
    if err := validateResources(spec.Resources); err != nil {
        return fmt.Errorf("invalid resources: %w", err)
    }
    
    return nil
}
```

### Secret Handling

```go
// Never log secrets
func (r *Reconciler) createSecret(ctx context.Context, data map[string][]byte) error {
    secret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "platform-secret",
            Namespace: "monitoring",
        },
        Data: data,
    }
    
    // Good - Don't log secret data
    r.Log.Info("Creating secret", "name", secret.Name)
    
    // Bad - Never do this
    // r.Log.Info("Creating secret", "data", secret.Data)
    
    return r.Create(ctx, secret)
}
```

---

## Kubernetes Specific Guidelines

### Client Usage

```go
// Use typed clients when possible
func (r *Reconciler) getDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
    deployment := &appsv1.Deployment{}
    key := client.ObjectKey{Name: name, Namespace: namespace}
    
    if err := r.Get(ctx, key, deployment); err != nil {
        return nil, fmt.Errorf("getting deployment %s/%s: %w", namespace, name, err)
    }
    
    return deployment, nil
}

// Use Server-Side Apply for better conflict resolution
func (r *Reconciler) applyDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
    return r.Patch(ctx, deployment, client.Apply, client.FieldOwner("gunj-operator"))
}
```

### Resource Management

```go
// Always set owner references
func (r *Reconciler) createOwnedResource(ctx context.Context, owner *v1beta1.ObservabilityPlatform, obj client.Object) error {
    if err := ctrl.SetControllerReference(owner, obj, r.Scheme); err != nil {
        return fmt.Errorf("setting controller reference: %w", err)
    }
    
    return r.Create(ctx, obj)
}

// Handle resource cleanup properly
func (r *Reconciler) handleDeletion(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
    if !controllerutil.ContainsFinalizer(platform, FinalizerName) {
        return nil
    }
    
    // Perform cleanup
    if err := r.cleanup(ctx, platform); err != nil {
        return fmt.Errorf("cleanup failed: %w", err)
    }
    
    // Remove finalizer
    controllerutil.RemoveFinalizer(platform, FinalizerName)
    return r.Update(ctx, platform)
}
```

### Status Updates

```go
// Update status separately from spec
func (r *Reconciler) updateStatus(ctx context.Context, platform *v1beta1.ObservabilityPlatform, phase string) error {
    platform.Status.Phase = phase
    platform.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
    platform.Status.ObservedGeneration = platform.Generation
    
    if err := r.Status().Update(ctx, platform); err != nil {
        return fmt.Errorf("updating status: %w", err)
    }
    
    return nil
}
```

---

## Tools and Enforcement

### Required Tools

All developers MUST have these tools installed:

```bash
# Install required tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

### Linting Configuration

Create `.golangci.yml`:

```yaml
linters:
  enable:
    - gofmt
    - goimports
    - golint
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - structcheck
    - varcheck
    - ineffassign
    - deadcode
    - typecheck
    - bodyclose
    - noctx
    - gosec
    - unconvert
    - dupl
    - misspell
    - lll
    - gocognit
    - goconst
    - gocyclo
    - maligned
    - prealloc

linters-settings:
  lll:
    line-length: 120
  gocognit:
    min-complexity: 20
  gocyclo:
    min-complexity: 15
  maligned:
    suggest-new: true
  dupl:
    threshold: 100
  goconst:
    min-len: 3
    min-occurrences: 3

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gocyclo
        - errcheck
        - dupl
        - gosec
```

### Pre-commit Hooks

Create `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: local
    hooks:
      - id: go-fmt
        name: go fmt
        entry: gofmt -w
        language: system
        files: \.go$
        pass_filenames: true
        
      - id: go-imports
        name: go imports
        entry: goimports -w
        language: system
        files: \.go$
        pass_filenames: true
        
      - id: golangci-lint
        name: golangci-lint
        entry: golangci-lint run
        language: system
        files: \.go$
        pass_filenames: false
        
      - id: go-test
        name: go test
        entry: go test ./...
        language: system
        files: \.go$
        pass_filenames: false
```

### Makefile Targets

```makefile
.PHONY: fmt lint test verify

# Format code
fmt:
	gofmt -w .
	goimports -w .

# Run linters
lint:
	golangci-lint run --timeout=10m

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run security scan
sec:
	gosec -fmt sarif -out gosec-results.sarif ./...

# Verify code quality
verify: fmt lint test sec
	@echo "All checks passed!"
```

---

## Appendix: Common Patterns

### Singleton Pattern

```go
var (
    instance *Manager
    once     sync.Once
)

func GetManager() *Manager {
    once.Do(func() {
        instance = &Manager{
            // Initialize
        }
    })
    return instance
}
```

### Builder Pattern

```go
type DeploymentBuilder struct {
    deployment *appsv1.Deployment
}

func NewDeploymentBuilder(name, namespace string) *DeploymentBuilder {
    return &DeploymentBuilder{
        deployment: &appsv1.Deployment{
            ObjectMeta: metav1.ObjectMeta{
                Name:      name,
                Namespace: namespace,
            },
        },
    }
}

func (b *DeploymentBuilder) WithReplicas(replicas int32) *DeploymentBuilder {
    b.deployment.Spec.Replicas = &replicas
    return b
}

func (b *DeploymentBuilder) WithImage(image string) *DeploymentBuilder {
    // Set image
    return b
}

func (b *DeploymentBuilder) Build() *appsv1.Deployment {
    return b.deployment
}
```

### Factory Pattern

```go
type ComponentFactory interface {
    CreateComponent(spec ComponentSpec) (Component, error)
}

type componentFactory struct {
    // fields
}

func NewComponentFactory() ComponentFactory {
    return &componentFactory{}
}

func (f *componentFactory) CreateComponent(spec ComponentSpec) (Component, error) {
    switch spec.Type {
    case "prometheus":
        return NewPrometheusComponent(spec), nil
    case "grafana":
        return NewGrafanaComponent(spec), nil
    default:
        return nil, fmt.Errorf("unknown component type: %s", spec.Type)
    }
}
```

---

**Document Status**: This is the authoritative guide for Golang coding standards in the Gunj Operator project. It will be reviewed and updated quarterly or as needed.
