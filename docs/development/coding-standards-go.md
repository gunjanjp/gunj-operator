# Go Coding Standards

This document outlines the Go coding standards for the Gunj Operator project. All Go code must follow these guidelines to ensure consistency, readability, and maintainability.

## Table of Contents

1. [General Guidelines](#general-guidelines)
2. [Code Organization](#code-organization)
3. [Naming Conventions](#naming-conventions)
4. [Error Handling](#error-handling)
5. [Testing Standards](#testing-standards)
6. [Documentation](#documentation)
7. [Performance Guidelines](#performance-guidelines)
8. [Security Guidelines](#security-guidelines)

## General Guidelines

### Code Formatting

- **Always use `gofmt`**: Run `gofmt -s` on all code
- **Use `goimports`**: Organize imports automatically
- **Line length**: Prefer lines under 120 characters
- **Use `golangci-lint`**: Run with project configuration

```bash
# Format code
make fmt

# Run linters
make lint
```

### Go Version

- Target Go 1.21+
- Use modern Go features where appropriate
- Avoid deprecated patterns

## Code Organization

### Package Structure

```go
// Package controllers implements the Kubernetes controllers for the Gunj Operator.
// It follows the controller-runtime patterns and manages the lifecycle of
// observability platform components.
package controllers

import (
    "context"
    "fmt"
    "time"
    
    // Standard library imports first
    
    // Third-party imports (alphabetically)
    "github.com/go-logr/logr"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    
    // Internal imports last
    observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
    "github.com/gunjanjp/gunj-operator/internal/managers"
)
```

### File Organization

```
controllers/
├── observabilityplatform_controller.go    # Main controller
├── observabilityplatform_controller_test.go
├── prometheus_reconciler.go               # Component reconcilers
├── prometheus_reconciler_test.go
├── grafana_reconciler.go
├── grafana_reconciler_test.go
├── helpers.go                             # Shared helpers
├── helpers_test.go
└── suite_test.go                          # Test suite setup
```

## Naming Conventions

### Variables and Functions

```go
// Good: Clear, descriptive names
func (r *ObservabilityPlatformReconciler) reconcilePrometheus(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
    prometheusConfig := r.buildPrometheusConfig(platform)
    
    // Use short names for common variables
    for i, rule := range platform.Spec.AlertingRules {
        if err := r.validateRule(rule); err != nil {
            return fmt.Errorf("validating rule %d: %w", i, err)
        }
    }
    
    return nil
}

// Bad: Unclear abbreviations
func (r *OPR) recProm(c context.Context, p *v1beta1.OP) error {
    pc := r.bldPromCfg(p)
    // ...
}
```

### Constants and Enums

```go
// Good: Clear constant naming
const (
    // DefaultPrometheusPort is the default port for Prometheus
    DefaultPrometheusPort = 9090
    
    // DefaultGrafanaPort is the default port for Grafana
    DefaultGrafanaPort = 3000
    
    // ReconcileTimeout is the maximum time for a reconciliation
    ReconcileTimeout = 5 * time.Minute
)

// Good: Type-safe enums
type Phase string

const (
    PhasePending   Phase = "Pending"
    PhaseInstalling Phase = "Installing"
    PhaseReady     Phase = "Ready"
    PhaseFailed    Phase = "Failed"
    PhaseUpgrading Phase = "Upgrading"
)
```

### Interfaces

```go
// Good: Interface names end with -er
type Reconciler interface {
    Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error)
}

type ComponentManager interface {
    Deploy(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
    Update(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
    Delete(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
    GetStatus(ctx context.Context, platform *v1beta1.ObservabilityPlatform) (*ComponentStatus, error)
}
```

## Error Handling

### Error Wrapping

```go
// Good: Wrap errors with context
func (r *Reconciler) deployComponent(ctx context.Context, name string) error {
    deployment := r.buildDeployment(name)
    
    if err := r.Create(ctx, deployment); err != nil {
        if !errors.IsAlreadyExists(err) {
            return fmt.Errorf("creating deployment %s: %w", name, err)
        }
        
        // Handle already exists case
        existing := &appsv1.Deployment{}
        if err := r.Get(ctx, client.ObjectKeyFromObject(deployment), existing); err != nil {
            return fmt.Errorf("getting existing deployment %s: %w", name, err)
        }
        
        existing.Spec = deployment.Spec
        if err := r.Update(ctx, existing); err != nil {
            return fmt.Errorf("updating deployment %s: %w", name, err)
        }
    }
    
    return nil
}
```

### Custom Errors

```go
// Good: Define custom error types for specific conditions
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s with value %v: %s", e.Field, e.Value, e.Message)
}

// Usage
func validateReplicas(replicas int32) error {
    if replicas < 1 || replicas > 10 {
        return ValidationError{
            Field:   "replicas",
            Value:   replicas,
            Message: "must be between 1 and 10",
        }
    }
    return nil
}
```

### Error Handling Patterns

```go
// Good: Early returns for errors
func (r *Reconciler) processRequest(ctx context.Context, req Request) error {
    // Validate input
    if err := req.Validate(); err != nil {
        return fmt.Errorf("invalid request: %w", err)
    }
    
    // Get resource
    resource, err := r.getResource(ctx, req.Name)
    if err != nil {
        return fmt.Errorf("getting resource %s: %w", req.Name, err)
    }
    
    // Process resource
    if err := r.process(ctx, resource); err != nil {
        return fmt.Errorf("processing resource %s: %w", req.Name, err)
    }
    
    return nil
}
```

## Testing Standards

### Test Structure

```go
func TestReconciler_DeployPrometheus(t *testing.T) {
    // Table-driven tests
    tests := []struct {
        name     string
        platform *v1beta1.ObservabilityPlatform
        objects  []runtime.Object
        wantErr  bool
        verify   func(t *testing.T, client client.Client)
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
            wantErr: false,
            verify: func(t *testing.T, c client.Client) {
                // Verify deployment created
                deployment := &appsv1.Deployment{}
                err := c.Get(context.Background(), types.NamespacedName{
                    Name:      "prometheus",
                    Namespace: "default",
                }, deployment)
                assert.NoError(t, err)
                assert.Equal(t, "v2.45.0", deployment.Labels["version"])
            },
        },
        {
            name: "handle existing deployment",
            platform: &v1beta1.ObservabilityPlatform{
                // ... platform definition
            },
            objects: []runtime.Object{
                // Existing deployment
                &appsv1.Deployment{
                    ObjectMeta: metav1.ObjectMeta{
                        Name:      "prometheus",
                        Namespace: "default",
                    },
                },
            },
            wantErr: false,
            verify: func(t *testing.T, c client.Client) {
                // Verify deployment updated
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            scheme := runtime.NewScheme()
            _ = v1beta1.AddToScheme(scheme)
            _ = appsv1.AddToScheme(scheme)
            
            builder := fake.NewClientBuilder().WithScheme(scheme)
            if tt.objects != nil {
                builder = builder.WithRuntimeObjects(tt.objects...)
            }
            
            c := builder.Build()
            r := &Reconciler{Client: c, Scheme: scheme}
            
            // Execute
            err := r.deployPrometheus(context.Background(), tt.platform)
            
            // Assert
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            
            if tt.verify != nil {
                tt.verify(t, c)
            }
        })
    }
}
```

### Mock Usage

```go
// Good: Use interfaces for mocking
type MetricsCollector interface {
    CollectMetrics(ctx context.Context, target string) (*Metrics, error)
}

// In tests
type mockMetricsCollector struct {
    metrics *Metrics
    err     error
}

func (m *mockMetricsCollector) CollectMetrics(ctx context.Context, target string) (*Metrics, error) {
    return m.metrics, m.err
}
```

## Documentation

### Package Documentation

```go
// Package managers provides component lifecycle management for the Gunj Operator.
//
// The managers package contains implementations for managing individual observability
// components like Prometheus, Grafana, Loki, and Tempo. Each manager handles the
// complete lifecycle including deployment, configuration, upgrades, and deletion.
//
// Example usage:
//
//	mgr := managers.NewPrometheusManager(client, scheme)
//	err := mgr.Deploy(ctx, platform)
//	if err != nil {
//	    log.Error(err, "Failed to deploy Prometheus")
//	}
package managers
```

### Function Documentation

```go
// ReconcilePrometheus ensures the Prometheus deployment matches the desired state
// specified in the ObservabilityPlatform resource.
//
// It performs the following operations:
//   - Creates or updates the Prometheus StatefulSet
//   - Configures Prometheus with the specified settings
//   - Creates necessary Services and ServiceMonitors
//   - Updates the platform status with Prometheus state
//
// Returns an error if any operation fails, nil otherwise.
func (r *Reconciler) ReconcilePrometheus(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
    // Implementation
}
```

### Inline Comments

```go
// Good: Explain why, not what
func (r *Reconciler) calculateReplicas(platform *v1beta1.ObservabilityPlatform) int32 {
    replicas := platform.Spec.Components.Prometheus.Replicas
    
    // Ensure odd number of replicas for quorum in HA mode
    if platform.Spec.HighAvailability && replicas%2 == 0 {
        replicas++
    }
    
    // Cap at MaxReplicas to prevent resource exhaustion
    if replicas > MaxReplicas {
        replicas = MaxReplicas
    }
    
    return replicas
}
```

## Performance Guidelines

### Efficient Resource Usage

```go
// Good: Reuse allocations
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processData(data []byte) string {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    buf.Write(data)
    return buf.String()
}
```

### Concurrent Operations

```go
// Good: Proper concurrency control
func (r *Reconciler) reconcileComponents(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
    g, ctx := errgroup.WithContext(ctx)
    
    // Limit concurrent operations
    sem := make(chan struct{}, 3)
    
    components := []struct {
        name      string
        reconcile func(context.Context, *v1beta1.ObservabilityPlatform) error
    }{
        {"prometheus", r.reconcilePrometheus},
        {"grafana", r.reconcileGrafana},
        {"loki", r.reconcileLoki},
        {"tempo", r.reconcileTempo},
    }
    
    for _, comp := range components {
        comp := comp // Capture loop variable
        g.Go(func() error {
            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release
            
            if err := comp.reconcile(ctx, platform); err != nil {
                return fmt.Errorf("reconciling %s: %w", comp.name, err)
            }
            return nil
        })
    }
    
    return g.Wait()
}
```

## Security Guidelines

### Input Validation

```go
// Good: Validate all inputs
func (r *Reconciler) validatePlatform(platform *v1beta1.ObservabilityPlatform) error {
    // Validate name
    if errs := validation.IsDNS1123Label(platform.Name); len(errs) > 0 {
        return fmt.Errorf("invalid platform name: %s", strings.Join(errs, ", "))
    }
    
    // Validate namespace
    if platform.Namespace == "" {
        return fmt.Errorf("namespace is required")
    }
    
    // Validate component versions
    if platform.Spec.Components.Prometheus != nil {
        if !isValidVersion(platform.Spec.Components.Prometheus.Version) {
            return fmt.Errorf("invalid Prometheus version: %s", platform.Spec.Components.Prometheus.Version)
        }
    }
    
    return nil
}
```

### Secret Handling

```go
// Good: Never log secrets
func (r *Reconciler) createSecret(ctx context.Context, name string, data map[string][]byte) error {
    log := ctrl.LoggerFrom(ctx)
    
    // Never log secret data
    log.V(1).Info("Creating secret", "name", name, "keys", mapKeys(data))
    
    secret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: r.Namespace,
        },
        Type: corev1.SecretTypeOpaque,
        Data: data,
    }
    
    return r.Create(ctx, secret)
}

func mapKeys(m map[string][]byte) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}
```

## Code Review Checklist

Before submitting code for review, ensure:

- [ ] Code passes `gofmt -s`
- [ ] Code passes `goimports`
- [ ] Code passes `golangci-lint`
- [ ] All tests pass
- [ ] Test coverage is adequate (>80%)
- [ ] Documentation is complete
- [ ] Error messages are helpful
- [ ] No sensitive data in logs
- [ ] Performance impact considered
- [ ] Security implications reviewed

## Tools and Commands

```bash
# Format code
make fmt

# Run linters
make lint

# Run tests with coverage
make test-coverage

# Run security scan
make security-scan

# Check for outdated dependencies
make check-deps
```

## Resources

- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Go Proverbs](https://go-proverbs.github.io/)

---

By following these standards, we ensure our Go code is consistent, maintainable, and of high quality.
