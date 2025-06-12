# GraphQL Integration Architecture

## Overview

This document describes how the GraphQL API integrates with the Gunj Operator architecture.

## Architecture Diagram

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│                 │     │                  │     │                 │
│   Web UI        │────▶│  GraphQL API     │────▶│   Operator      │
│  (React App)    │     │  (Port 4000)     │     │  Controller     │
│                 │     │                  │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
         │                       │                         │
         │                       │                         │
         ▼                       ▼                         ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   WebSocket     │     │   DataLoader     │     │   Kubernetes    │
│  Subscriptions  │     │   (N+1 Cache)    │     │      API        │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

## Key Integration Points

### 1. GraphQL Server Setup

```go
// internal/api/graphql/server.go
package graphql

import (
    "github.com/99designs/gqlgen/graphql/handler"
    "github.com/99designs/gqlgen/graphql/handler/extension"
    "github.com/99designs/gqlgen/graphql/handler/lru"
    "github.com/99designs/gqlgen/graphql/handler/transport"
    "github.com/99designs/gqlgen/graphql/playground"
)

func NewServer(resolver *Resolver) *handler.Server {
    srv := handler.NewDefaultServer(NewExecutableSchema(Config{
        Resolvers: resolver,
    }))

    // Add transport options
    srv.AddTransport(&transport.Websocket{
        Upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                return true // Configure appropriately
            },
        },
        KeepAlivePingInterval: 10 * time.Second,
    })

    // Add extensions
    srv.Use(extension.Introspection{})
    srv.Use(extension.AutomaticPersistedQuery{
        Cache: lru.New(100),
    })

    // Add custom middleware
    srv.Use(NewComplexityLimit(1000))
    srv.Use(NewRateLimiter())
    srv.Use(NewAuthMiddleware())

    return srv
}
```

### 2. Resolver Implementation

```go
// internal/api/graphql/resolver.go
package graphql

import (
    "context"
    "sigs.k8s.io/controller-runtime/pkg/client"
    
    "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

type Resolver struct {
    client     client.Client
    dataloader *DataLoader
}

// Platform resolver
func (r *Resolver) Platform(ctx context.Context, name string, namespace string) (*Platform, error) {
    // Use DataLoader to batch Kubernetes API calls
    return r.dataloader.LoadPlatform(ctx, namespace, name)
}

// Platforms resolver with pagination
func (r *Resolver) Platforms(ctx context.Context, args PlatformsArgs) (*PlatformConnection, error) {
    var platforms v1beta1.ObservabilityPlatformList
    
    opts := []client.ListOption{
        client.InNamespace(args.Namespace),
    }
    
    if args.Labels != nil {
        opts = append(opts, client.MatchingLabels(args.Labels.MatchLabels))
    }
    
    if err := r.client.List(ctx, &platforms, opts...); err != nil {
        return nil, err
    }
    
    return toPlatformConnection(platforms, args.Pagination), nil
}
```

### 3. DataLoader Pattern

```go
// internal/api/graphql/dataloader.go
package graphql

import (
    "github.com/graph-gophers/dataloader"
)

type DataLoader struct {
    platformLoader *dataloader.Loader
    componentLoader *dataloader.Loader
}

func NewDataLoader(client client.Client) *DataLoader {
    return &DataLoader{
        platformLoader: dataloader.NewBatchedLoader(
            batchGetPlatforms(client),
            dataloader.WithCache(&dataloader.InMemoryCache{}),
            dataloader.WithWait(2*time.Millisecond),
        ),
        componentLoader: dataloader.NewBatchedLoader(
            batchGetComponents(client),
            dataloader.WithCache(&dataloader.InMemoryCache{}),
            dataloader.WithWait(2*time.Millisecond),
        ),
    }
}
```

### 4. Subscription Implementation

```go
// internal/api/graphql/subscriptions.go
package graphql

func (r *Resolver) PlatformStatus(ctx context.Context, name string, namespace string) (<-chan *PlatformStatus, error) {
    ch := make(chan *PlatformStatus, 1)
    
    // Create informer for the specific platform
    informer := r.createPlatformInformer(namespace, name)
    
    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        UpdateFunc: func(oldObj, newObj interface{}) {
            platform := newObj.(*v1beta1.ObservabilityPlatform)
            
            select {
            case ch <- toPlatformStatus(platform.Status):
            case <-ctx.Done():
                close(ch)
                return
            }
        },
    })
    
    go informer.Run(ctx.Done())
    
    return ch, nil
}
```

### 5. Authorization Integration

```go
// internal/api/graphql/auth.go
package graphql

func NewAuthMiddleware() graphql.ResponseMiddleware {
    return func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
        // Get field context
        fc := graphql.GetFieldContext(ctx)
        
        // Check directives
        for _, directive := range fc.Field.Definition.Directives {
            switch directive.Name {
            case "hasRole":
                if !hasRequiredRole(ctx, directive.Arguments) {
                    return graphql.ErrorResponse(ctx, "Insufficient permissions")
                }
            case "hasPermission":
                if !hasRequiredPermission(ctx, directive.Arguments) {
                    return graphql.ErrorResponse(ctx, "Insufficient permissions")
                }
            }
        }
        
        return next(ctx)
    }
}
```

## Integration Benefits

1. **Single Request**: Clients can fetch exactly what they need in one request
2. **Type Safety**: Strong typing ensures API contract consistency
3. **Real-time Updates**: Subscriptions provide live data updates
4. **Performance**: DataLoader prevents N+1 queries
5. **Flexibility**: Clients control the response shape
6. **Introspection**: Self-documenting API

## Implementation Timeline

- Week 1: Basic query implementation
- Week 2: Mutations and complex types
- Week 3: Subscriptions and real-time updates
- Week 4: Performance optimization and testing