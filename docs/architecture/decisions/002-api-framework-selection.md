# API Framework Selection Decision

**Date**: June 12, 2025  
**Decision**: Gin Web Framework v1.9.1  
**Status**: Approved and Implemented  

## Summary

After evaluating Gin, Echo, and Fiber frameworks, **Gin v1.9.1** has been selected as the API framework for the Gunj Operator project.

## Decision Criteria

1. **Kubernetes Ecosystem Compatibility** (Weight: 30%)
   - Gin: ⭐⭐⭐⭐⭐ Standard choice in K8s projects
   - Echo: ⭐⭐⭐ Good compatibility
   - Fiber: ⭐⭐ Limited K8s adoption

2. **GraphQL Integration** (Weight: 25%)
   - Gin: ⭐⭐⭐⭐⭐ Excellent with gqlgen
   - Echo: ⭐⭐⭐ Basic support
   - Fiber: ⭐⭐ Limited ecosystem

3. **Performance** (Weight: 20%)
   - Gin: ⭐⭐⭐⭐ Excellent (meets <100ms requirement)
   - Echo: ⭐⭐⭐⭐ Excellent
   - Fiber: ⭐⭐⭐⭐⭐ Best (but overkill for our needs)

4. **Middleware Ecosystem** (Weight: 15%)
   - Gin: ⭐⭐⭐⭐⭐ Richest selection
   - Echo: ⭐⭐⭐⭐ Good built-in middleware
   - Fiber: ⭐⭐⭐ Growing but limited

5. **Community & Stability** (Weight: 10%)
   - Gin: ⭐⭐⭐⭐⭐ 74k+ stars, mature
   - Echo: ⭐⭐⭐⭐ 28k+ stars, stable
   - Fiber: ⭐⭐⭐ 30k+ stars, newer

## Technical Implementation

### Core Dependencies
```go
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/gin-contrib/cors v1.4.0
    github.com/gin-contrib/requestid v0.0.6
    github.com/99designs/gqlgen v0.17.40
    github.com/gorilla/websocket v1.5.1
    go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.46.1
)
```

### Architecture Integration
```
gunj-operator/
└── internal/
    └── api/
        ├── server.go          # Main API server with Gin
        ├── graphql.go         # GraphQL integration
        ├── websocket.go       # WebSocket handlers
        ├── sse.go            # Server-Sent Events
        ├── middleware/       # Gin middleware
        │   ├── auth.go
        │   ├── ratelimit.go
        │   └── logging.go
        └── handlers/         # REST API handlers
            ├── platform.go
            ├── component.go
            └── metrics.go
```

### Key Features Enabled

1. **RESTful API**: Full CRUD operations with consistent response format
2. **GraphQL API**: Query, Mutation, and Subscription support via gqlgen
3. **Real-time**: WebSocket and SSE integration for live updates
4. **Security**: JWT/OIDC authentication, RBAC authorization
5. **Observability**: OpenTelemetry integration, structured logging
6. **Performance**: Rate limiting, request ID tracking, efficient routing

## Risk Mitigation

- **Risk**: Gin uses standard net/http, not fasthttp
- **Mitigation**: Performance testing shows Gin meets all requirements
- **Fallback**: API layer is abstracted, framework can be changed if needed

## Next Steps

1. Complete middleware implementations
2. Create handler packages for each resource
3. Integrate with GraphQL schema generation
4. Set up WebSocket connection manager
5. Implement SSE event streaming

## References

- [Gin Documentation](https://gin-gonic.com/docs/)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [gqlgen GraphQL](https://gqlgen.com/)
