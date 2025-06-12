# Gunj Operator Architecture

This document provides a detailed technical overview of the Gunj Operator architecture, covering all major components, design decisions, and implementation details.

## Table of Contents

1. [Overview](#overview)
2. [Core Components](#core-components)
3. [Design Principles](#design-principles)
4. [Component Details](#component-details)
5. [Data Flow](#data-flow)
6. [Security Architecture](#security-architecture)
7. [Scalability Considerations](#scalability-considerations)
8. [Future Enhancements](#future-enhancements)

## Overview

The Gunj Operator follows a modern cloud-native architecture built on Kubernetes operator pattern. It consists of:

- **Operator Core**: Kubernetes controller managing CRDs
- **API Server**: RESTful and GraphQL APIs for external access
- **Web UI**: React-based management interface
- **Component Managers**: Specialized managers for each observability component

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              External Access                              │
├─────────────────┬────────────────────┬─────────────────┬────────────────┤
│   kubectl/CLI   │    Web Browser     │   REST Client   │  GraphQL Client│
└────────┬────────┴─────────┬──────────┴────────┬────────┴────────┬───────┘
         │                  │                   │                 │
         │              ┌───▼───┐          ┌────▼────┐      ┌────▼────┐
         │              │Web UI │          │REST API │      │GraphQL  │
         │              │(React)│          │ (Gin)   │      │  API    │
         │              └───┬───┘          └────┬────┘      └────┬────┘
         │                  │                   │                 │
         │                  └───────────┬───────┴─────────────────┘
         │                              │
     ┌───▼────────────────────────────▼─┐
     │        Kubernetes API Server      │
     └───────────────┬───────────────────┘
                     │
         ┌───────────▼───────────┐
         │   Gunj Operator       │
         │  ┌─────────────────┐  │
         │  │  Controllers    │  │
         │  ├─────────────────┤  │
         │  │  Reconcilers    │  │
         │  ├─────────────────┤  │
         │  │  Webhooks       │  │
         │  ├─────────────────┤  │
         │  │Component Managers│  │
         │  └─────────────────┘  │
         └───────────┬───────────┘
                     │
     ┌───────────────┴───────────────────┐
     │         Managed Components         │
     ├─────────┬─────────┬───────┬───────┤
     │Prometheus│ Grafana │ Loki  │ Tempo │
     └─────────┴─────────┴───────┴───────┘
```

## Core Components

### 1. Operator Core

The operator core implements the Kubernetes controller pattern using controller-runtime.

**Key Responsibilities:**
- Watch ObservabilityPlatform CRDs
- Reconcile desired state with actual state
- Manage component lifecycle
- Handle upgrades and rollbacks
- Emit events and update status

**Implementation:**
```go
type ObservabilityPlatformReconciler struct {
    client.Client
    Scheme            *runtime.Scheme
    Recorder          record.EventRecorder
    ComponentManagers map[string]ComponentManager
}
```

### 2. API Server

Provides external API access to operator functionality.

**Features:**
- RESTful API for CRUD operations
- GraphQL API for flexible queries
- WebSocket support for real-time updates
- Authentication and authorization
- Rate limiting and quotas

**Endpoints:**
- `/api/v1/platforms` - Platform management
- `/api/v1/components` - Component operations
- `/graphql` - GraphQL endpoint
- `/ws` - WebSocket endpoint

### 3. Web UI

Modern React-based user interface.

**Technology Stack:**
- React 18 with TypeScript
- Material-UI components
- Zustand for state management
- React Query for data fetching
- Recharts for visualization

**Features:**
- Real-time platform status
- Interactive configuration editor
- Metrics and log viewing
- Cost analysis dashboard
- Multi-language support

### 4. Component Managers

Specialized managers for each observability component.

**Prometheus Manager:**
- StatefulSet deployment
- Configuration generation
- Service discovery setup
- Alert rule management

**Grafana Manager:**
- Deployment management
- Dashboard provisioning
- Data source configuration
- Plugin management

**Loki Manager:**
- StatefulSet deployment
- Storage configuration
- Compactor setup
- Retention policies

**Tempo Manager:**
- Deployment management
- Storage backend setup
- Ingester configuration
- Query frontend setup

## Design Principles

### 1. Declarative Configuration

All configuration is declarative through CRDs:

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
spec:
  # Entire desired state declared here
```

### 2. Eventual Consistency

The operator continuously reconciles towards desired state:

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Fetch current state
    // Compare with desired state
    // Take actions to converge
    // Requeue for continuous reconciliation
    return ctrl.Result{RequeueAfter: ReconcilePeriod}, nil
}
```

### 3. Idempotency

All operations are idempotent and safe to retry:

```go
func (r *Reconciler) ensureDeployment(ctx context.Context, desired *appsv1.Deployment) error {
    existing := &appsv1.Deployment{}
    err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
    
    if err != nil && errors.IsNotFound(err) {
        // Create new
        return r.Create(ctx, desired)
    } else if err != nil {
        return err
    }
    
    // Update existing
    existing.Spec = desired.Spec
    return r.Update(ctx, existing)
}
```

### 4. Extensibility

Plugin architecture for custom components:

```go
type ComponentManager interface {
    Deploy(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
    Update(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
    Delete(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error
    GetStatus(ctx context.Context, platform *v1beta1.ObservabilityPlatform) (*ComponentStatus, error)
}
```

## Component Details

### Controller Architecture

```go
// Main controller structure
type Controller struct {
    // Kubernetes client
    client.Client
    
    // Scheme for type registration
    Scheme *runtime.Scheme
    
    // Component managers
    Managers map[string]ComponentManager
    
    // Event recorder
    Recorder record.EventRecorder
    
    // Metrics collector
    Metrics *MetricsCollector
    
    // Configuration
    Config *OperatorConfig
}
```

### Reconciliation Loop

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := ctrl.LoggerFrom(ctx)
    
    // 1. Fetch the ObservabilityPlatform
    platform := &v1beta1.ObservabilityPlatform{}
    if err := r.Get(ctx, req.NamespacedName, platform); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // 2. Handle deletion
    if !platform.DeletionTimestamp.IsZero() {
        return r.handleDeletion(ctx, platform)
    }
    
    // 3. Ensure finalizer
    if !controllerutil.ContainsFinalizer(platform, FinalizerName) {
        controllerutil.AddFinalizer(platform, FinalizerName)
        if err := r.Update(ctx, platform); err != nil {
            return ctrl.Result{}, err
        }
    }
    
    // 4. Reconcile components
    if err := r.reconcileComponents(ctx, platform); err != nil {
        platform.Status.Phase = v1beta1.PhaseFailed
        platform.Status.Message = err.Error()
        _ = r.Status().Update(ctx, platform)
        return ctrl.Result{}, err
    }
    
    // 5. Update status
    platform.Status.Phase = v1beta1.PhaseReady
    platform.Status.Message = "All components are running"
    if err := r.Status().Update(ctx, platform); err != nil {
        return ctrl.Result{}, err
    }
    
    // 6. Requeue for continuous reconciliation
    return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}
```

### Webhook Implementation

```go
// Validating webhook
func (v *PlatformValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
    platform := obj.(*v1beta1.ObservabilityPlatform)
    
    // Validate component versions
    if platform.Spec.Components.Prometheus != nil {
        if err := validateVersion(platform.Spec.Components.Prometheus.Version); err != nil {
            return err
        }
    }
    
    // Validate resource requirements
    if err := validateResources(platform); err != nil {
        return err
    }
    
    return nil
}

// Mutating webhook
func (m *PlatformMutator) Default(ctx context.Context, obj runtime.Object) error {
    platform := obj.(*v1beta1.ObservabilityPlatform)
    
    // Set defaults
    if platform.Spec.Components.Prometheus != nil {
        if platform.Spec.Components.Prometheus.Retention == "" {
            platform.Spec.Components.Prometheus.Retention = "15d"
        }
    }
    
    return nil
}
```

## Data Flow

### 1. Resource Creation Flow

```
User → kubectl/API → K8s API Server → Admission Webhooks → etcd
                                              ↓
                                        Operator Controller
                                              ↓
                                     Component Reconciliation
                                              ↓
                                    Create K8s Resources
                                              ↓
                                      Update Status
```

### 2. Metrics Collection Flow

```
Components → Prometheus → Aggregation → Storage
     ↓                                      ↓
Service Discovery                    Remote Write
     ↓                                      ↓
Scrape Targets                      Long-term Storage
```

### 3. Query Flow

```
User Query → Grafana → Data Source Proxy → Prometheus/Loki/Tempo
                              ↓                      ↓
                        Authentication          Query Engine
                              ↓                      ↓
                          Results              Response Cache
```

## Security Architecture

### 1. Authentication Layers

```
External Users → OIDC/SAML → API Gateway → Service
                                  ↓
                           JWT Validation
                                  ↓
                            RBAC Check
```

### 2. Network Security

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: operator-network-policy
spec:
  podSelector:
    matchLabels:
      app: gunj-operator
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: gunj-system
    ports:
    - protocol: TCP
      port: 8443  # Webhook port
  egress:
  - to:
    - namespaceSelector: {}  # Allow all namespaces
    ports:
    - protocol: TCP
      port: 443  # Kubernetes API
```

### 3. Secret Management

```go
// Automatic secret rotation
func (r *Reconciler) rotateSecrets(ctx context.Context) error {
    secrets, err := r.listManagedSecrets(ctx)
    if err != nil {
        return err
    }
    
    for _, secret := range secrets {
        if shouldRotate(secret) {
            newSecret := generateNewSecret()
            if err := r.updateSecret(ctx, secret, newSecret); err != nil {
                return err
            }
        }
    }
    
    return nil
}
```

## Scalability Considerations

### 1. Horizontal Scaling

The operator supports horizontal scaling through leader election:

```go
func main() {
    // Setup leader election
    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
        LeaderElection:          true,
        LeaderElectionID:        "gunj-operator-leader",
        LeaderElectionNamespace: "gunj-system",
    })
}
```

### 2. Resource Optimization

```go
// Dynamic resource allocation
func (r *Reconciler) calculateResources(platform *v1beta1.ObservabilityPlatform) corev1.ResourceRequirements {
    // Base resources
    resources := corev1.ResourceRequirements{
        Requests: corev1.ResourceList{
            corev1.ResourceCPU:    resource.MustParse("100m"),
            corev1.ResourceMemory: resource.MustParse("128Mi"),
        },
    }
    
    // Scale based on workload
    if platform.Spec.WorkloadSize == "large" {
        resources.Requests[corev1.ResourceCPU] = resource.MustParse("1000m")
        resources.Requests[corev1.ResourceMemory] = resource.MustParse("2Gi")
    }
    
    return resources
}
```

### 3. Performance Optimizations

- **Caching**: Kubernetes client cache for reduced API calls
- **Batch Operations**: Group multiple updates
- **Async Processing**: Non-blocking reconciliation
- **Resource Pooling**: Reuse connections and clients

## Future Enhancements

### 1. Machine Learning Integration

- Anomaly detection in metrics
- Predictive scaling
- Automated remediation

### 2. Advanced Multi-tenancy

- Namespace isolation
- Resource quotas per tenant
- Cost allocation

### 3. Service Mesh Integration

- Automatic sidecar injection
- Traffic management
- Circuit breaking

### 4. Edge Deployment

- Lightweight components
- Offline operation
- Data aggregation

## Conclusion

The Gunj Operator architecture is designed to be:

- **Extensible**: Easy to add new components
- **Scalable**: Handles large deployments
- **Secure**: Multiple security layers
- **Reliable**: Self-healing and fault-tolerant
- **User-friendly**: Simple to operate

For more detailed information on specific components, see:

- [Operator Internals](./operator-internals.md)
- [API Architecture](./api-architecture.md)
- [Security Architecture](./security-architecture.md)
- [UI Architecture](./ui-architecture.md)
