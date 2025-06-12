# Gunj Operator OpenAPI Specification Summary

**Version**: 1.0  
**Date**: June 12, 2025  
**File**: `api/openapi/gunj-operator-api-v1.yaml`

## Overview

The OpenAPI 3.0 specification has been created for the Gunj Operator API v1. This comprehensive specification provides the foundation for:

- API client code generation
- Interactive API documentation (Swagger UI)
- API validation and testing
- Contract-first development

## Key Features

### 1. Authentication
- JWT-based authentication with bearer tokens
- Login, refresh, and logout endpoints
- Support for OIDC/SAML integration

### 2. Platform Management
- Full CRUD operations for ObservabilityPlatform resources
- Support for namespace isolation
- Label-based filtering and sorting
- JSON Patch support for partial updates

### 3. Component Management
- Individual component configuration
- Support for Prometheus, Grafana, Loki, Tempo
- Version management and upgrades

### 4. Operations
- Backup and restore functionality
- Component upgrades with strategies
- Async operation tracking

### 5. Monitoring
- Platform metrics endpoint
- Health and readiness checks
- Component-level health status

## API Structure

### Base URLs
- Production: `https://api.gunj-operator.yourdomain.com/api/v1`
- Staging: `https://staging-api.gunj-operator.yourdomain.com/api/v1`
- Local: `http://localhost:8080/api/v1`

### Main Endpoints

#### Authentication
- `POST /auth/login` - User authentication
- `POST /auth/refresh` - Token refresh
- `POST /auth/logout` - Session termination

#### Platforms
- `GET /platforms` - List platforms
- `POST /platforms` - Create platform
- `GET /platforms/{name}` - Get platform
- `PUT /platforms/{name}` - Update platform
- `PATCH /platforms/{name}` - Patch platform
- `DELETE /platforms/{name}` - Delete platform

#### Components
- `GET /platforms/{name}/components` - List components
- `PUT /platforms/{name}/components/{component}` - Update component

#### Operations
- `POST /platforms/{name}/operations/backup` - Create backup
- `POST /platforms/{name}/operations/restore` - Restore from backup
- `POST /platforms/{name}/operations/upgrade` - Upgrade components

#### Monitoring
- `GET /platforms/{name}/metrics` - Get metrics
- `GET /platforms/{name}/health` - Get health status
- `GET /health` - API health check
- `GET /ready` - API readiness check

## Security

### Authentication
- Bearer token authentication (JWT)
- Token expiration and refresh mechanism
- Rate limiting with headers

### Rate Limiting
- Default: 100 requests per minute per user
- Burst: 200 requests
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

## Response Format

### Success Response
```json
{
  "apiVersion": "observability.io/v1beta1",
  "kind": "ObservabilityPlatform",
  "metadata": {...},
  "spec": {...},
  "status": {...}
}
```

### Error Response
```json
{
  "code": "PLATFORM_NOT_FOUND",
  "message": "The requested platform does not exist",
  "details": {
    "platform": "production-platform",
    "namespace": "monitoring"
  },
  "timestamp": "2025-06-12T10:30:00Z",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

## Next Steps

1. **Generate API Documentation**
   ```bash
   npx @redocly/openapi-cli preview-docs api/openapi/gunj-operator-api-v1.yaml
   ```

2. **Validate Specification**
   ```bash
   npx @redocly/openapi-cli lint api/openapi/gunj-operator-api-v1.yaml
   ```

3. **Generate Client SDKs**
   ```bash
   openapi-generator generate -i api/openapi/gunj-operator-api-v1.yaml -g go -o pkg/client
   ```

4. **Set up Swagger UI**
   ```bash
   docker run -p 8080:8080 -e SWAGGER_JSON=/api/gunj-operator-api-v1.yaml -v ${PWD}/api/openapi:/api swaggerapi/swagger-ui
   ```

## Integration with Development

The OpenAPI specification will be used for:

1. **Contract-First Development**: API implementation must match the specification
2. **Testing**: Contract testing to ensure compliance
3. **Documentation**: Auto-generated API docs
4. **Client Libraries**: Generated SDKs for various languages
5. **Mocking**: Mock servers for frontend development

This completes the API architecture design phase of the Gunj Operator project.
