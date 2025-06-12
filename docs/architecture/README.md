# Gunj Operator Architecture

This document provides a comprehensive overview of the Gunj Operator architecture, design principles, and technical implementation details.

## Table of Contents

1. [Overview](#overview)
2. [Architecture Principles](#architecture-principles)
3. [Component Architecture](#component-architecture)
4. [Data Flow](#data-flow)
5. [Security Architecture](#security-architecture)
6. [Scalability Design](#scalability-design)
7. [Integration Points](#integration-points)

## Overview

The Gunj Operator is a cloud-native Kubernetes operator that manages the complete lifecycle of observability platforms. It follows the operator pattern to provide declarative management of Prometheus, Grafana, Loki, and Tempo.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Users/Systems                              │
└────────────┬───────────────────────┬────────────────┬────────────┘
             │                       │                │
             ▼                       ▼                ▼
     ┌───────────────┐      ┌───────────────┐  ┌───────────────┐
     │   Web UI      │      │   REST API    │  │  GraphQL API  │
     │   (React)     │      │   (Gin)       │  │   (gqlgen)    │
     └───────┬───────┘      └───────┬───────┘  └───────┬───────┘
             │                       │                │
             └───────────────────────┴────────────────┘
                                    │
                        ┌───────────▼───────────┐
                        │   API Server          │
                        │   - Auth/AuthZ        │
                        │   - Rate Limiting     │
                        │   - Request Routing   │
                        └───────────┬───────────┘
                                    │
                        ┌───────────▼───────────┐
                        │   Operator Core       │
                        │   - Controllers       │
                        │   - Reconcilers      │
                        │   - Webhooks         │
                        └───────────┬───────────┘
                                    │
                ┌───────────────────┴───────────────────┐
                │          Kubernetes API               │
                └───────────────────┬───────────────────┘
                                    │
        ┌───────────┬───────────┬───┴───────┬───────────┐
        ▼           ▼           ▼           ▼           ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
   │Prometheus│ │ Grafana │ │  Loki   │ │  Tempo  │ │ Others  │
   └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘
```

## Architecture Principles

### 1. Cloud Native Design

- **Containerized**: All components run in containers
- **Kubernetes Native**: Built for and with Kubernetes
- **12-Factor App**: Follows cloud-native best practices
- **Stateless Operator**: State stored in Kubernetes

### 2. Declarative Management

- **GitOps Ready**: Full declarative configuration
- **Idempotent Operations**: Safe to retry
- **Self-Healing**: Automatic error recovery
- **Eventual Consistency**: Converges to desired state

### 3. Extensibility

- **Plugin Architecture**: Support for custom components
- **Webhook System**: External integrations
- **API First**: Everything accessible via API
- **Custom Resources**: Extensible via CRDs

### 4. Security First

- **Zero Trust**: Assume breach mentality
- **RBAC**: Fine-grained access control
- **Encryption**: TLS everywhere, encrypted storage
- **Audit Logging**: Complete audit trail

## Component Architecture

### Operator Core

The operator core implements the Kubernetes controller pattern:

```go
// Simplified controller architecture
type ObservabilityPlatformReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    
    // Component managers
    PrometheusManager Manager
    GrafanaManager    Manager
    LokiManager       Manager
    TempoManager      Manager
    
    // Utilities
    Recorder record.EventRecorder
    Cache    cache.Cache
}
```

**Key Components:**

1. **Controllers**: Watch and reconcile CRs
2. **Managers**: Handle component lifecycle
3. **Webhooks**: Validate and mutate resources
4. **Informers**: Cache Kubernetes objects
5. **Work Queue**: Process reconciliation requests

[Read more: Operator Internals](./operator-internals.md)

### API Server

The API server provides external access to operator functionality:

```
┌─────────────────────────────────────────┐
│           API Gateway                    │
├─────────────────────────────────────────┤
│  - Request Routing                      │
│  - Load Balancing                       │
│  - SSL Termination                      │
└────────────────┬────────────────────────┘
                 │
    ┌────────────┴────────────┐
    │                         │
┌───▼──────────┐      ┌──────▼──────────┐
│  REST API    │      │  GraphQL API    │
├──────────────┤      ├─────────────────┤
│ - CRUD Ops   │      │ - Queries       │
│ - Streaming  │      │ - Mutations     │
│ - Webhooks   │      │ - Subscriptions │
└──────┬───────┘      └────────┬────────┘
       │                       │
       └───────────┬───────────┘
                   │
         ┌─────────▼─────────┐
         │   Business Logic  │
         │   - Validation    │
         │   - Authorization │
         │   - Processing    │
         └───────────────────┘
```

[Read more: API Architecture](./api-architecture.md)

### Web UI

React-based single-page application:

```
┌─────────────────────────────────────┐
│          React Application          │
├─────────────────────────────────────┤
│  ┌─────────────────────────────┐   │
│  │      Component Layer         │   │
│  │  - Pages                    │   │
│  │  - Shared Components        │   │
│  │  - Layouts                  │   │
│  └──────────────┬──────────────┘   │
│                 │                   │
│  ┌──────────────▼──────────────┐   │
│  │      State Management       │   │
│  │  - Zustand Stores          │   │
│  │  - React Query             │   │
│  │  - Context Providers       │   │
│  └──────────────┬──────────────┘   │
│                 │                   │
│  ┌──────────────▼──────────────┐   │
│  │      Service Layer          │   │
│  │  - API Client              │   │
│  │  - WebSocket Client        │   │
│  │  - Error Handling          │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

[Read more: UI Architecture](./ui-architecture.md)

## Data Flow

### Resource Creation Flow

```
User/GitOps → API/kubectl → Kubernetes API → Operator Controller
                                                    │
                                                    ▼
                                            Reconciliation Loop
                                                    │
                    ┌───────────────────────────────┴───────────┐
                    │                                           │
                    ▼                                           ▼
            Validate Resources                          Create/Update Resources
                    │                                           │
                    ▼                                           ▼
            Apply Defaults                              Update Status
                    │                                           │
                    ▼                                           ▼
            Create K8s Objects                          Emit Events
```

### Monitoring Data Flow

```
Components → Metrics/Logs/Traces → Collection Layer → Storage Layer → Query Layer → UI
     │              │                     │               │              │           │
Prometheus      Promtail             Collectors      Prometheus      PromQL      Grafana
Grafana         Loki Agent           OTel Agent      Loki           LogQL       Web UI
Loki           OTel SDK              Fluentd         Tempo          TraceQL     API
Tempo
```

## Security Architecture

### Authentication & Authorization

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   User      │────▶│    Auth     │────▶│   AuthZ     │
└─────────────┘     └─────────────┘     └─────────────┘
                           │                     │
                    ┌──────┴──────┐       ┌─────┴─────┐
                    │             │       │           │
                    ▼             ▼       ▼           ▼
                 OIDC         Local    RBAC      Policies
                Provider      Auth    Rules
```

**Security Layers:**

1. **Network Security**
   - TLS encryption
   - Network policies
   - Service mesh integration

2. **Authentication**
   - OIDC/SAML support
   - mTLS for service-to-service
   - API key management

3. **Authorization**
   - Kubernetes RBAC
   - Custom policies
   - Multi-tenancy isolation

4. **Data Security**
   - Encryption at rest
   - Secrets management
   - Audit logging

[Read more: Security Architecture](./security-architecture.md)

## Scalability Design

### Horizontal Scaling

```
┌─────────────────────┐
│   Load Balancer     │
└──────────┬──────────┘
           │
    ┌──────┴──────┬──────────┬──────────┐
    ▼             ▼          ▼          ▼
┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐
│Operator│  │Operator│  │Operator│  │Operator│
│ Pod 1  │  │ Pod 2  │  │ Pod 3  │  │ Pod N  │
└────────┘  └────────┘  └────────┘  └────────┘
    │             │          │          │
    └─────────────┴──────────┴──────────┘
                  │
           Leader Election
                  │
                  ▼
           Active Leader
```

**Scaling Strategies:**

1. **Operator Scaling**
   - Leader election for single active
   - Horizontal pod autoscaling
   - Resource-based sharding

2. **Component Scaling**
   - Automated replica management
   - Resource optimization
   - Load distribution

3. **Data Scaling**
   - Retention policies
   - Data tiering
   - Compression strategies

[Read more: Scalability Design](./scalability.md)

## Integration Points

### External Integrations

```
┌─────────────────────────────────────────────────┐
│              Gunj Operator                       │
├─────────────────────────────────────────────────┤
│                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  APIs    │  │ Webhooks │  │ Events   │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
│       │              │              │           │
└───────┼──────────────┼──────────────┼───────────┘
        │              │              │
   ┌────▼────┐    ┌───▼────┐    ┌───▼────┐
   │ GitOps  │    │  SIEM  │    │  ITSM  │
   │ - Argo  │    │ - Splunk│    │ - SNOW │
   │ - Flux  │    │ - ELK   │    │ - Jira │
   └─────────┘    └────────┘    └────────┘
```

**Integration Types:**

1. **GitOps Integration**
   - ArgoCD support
   - Flux compatibility
   - Helm integration

2. **Monitoring Integration**
   - External Prometheus
   - Remote write/read
   - Federation support

3. **Enterprise Integration**
   - LDAP/AD authentication
   - SIEM forwarding
   - Ticketing systems

[Read more: Integration Architecture](./integrations.md)

## Related Documentation

- [Operator Internals](./operator-internals.md)
- [API Architecture](./api-architecture.md)
- [UI Architecture](./ui-architecture.md)
- [Security Architecture](./security-architecture.md)
- [Scalability Design](./scalability.md)
- [Integration Guide](./integrations.md)
- [Decision Records](./decisions/)
