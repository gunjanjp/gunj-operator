# Gunj Operator RESTful API Design

**Version**: 1.0.0  
**Date**: June 12, 2025  
**Status**: Draft  
**Author**: Gunjan JP  

## Table of Contents
1. [Overview](#overview)
2. [API Principles](#api-principles)
3. [Base URL Structure](#base-url-structure)
4. [Authentication](#authentication)
5. [API Endpoints](#api-endpoints)
6. [Response Format](#response-format)
7. [Error Handling](#error-handling)
8. [Versioning Strategy](#versioning-strategy)

## Overview

The Gunj Operator RESTful API provides programmatic access to manage observability platforms in Kubernetes. This API follows REST principles and provides a consistent interface for all platform operations.

## API Principles

1. **RESTful Design**: Resources are nouns, HTTP methods are verbs
2. **Consistency**: Uniform interface across all endpoints
3. **Stateless**: Each request contains all necessary information
4. **Idempotent**: Safe operations can be repeated without side effects
5. **HATEOAS**: Responses include links to related resources
6. **JSON-First**: All requests and responses use JSON

## Base URL Structure

```
https://api.gunj-operator.{domain}/api/{version}
```

Example:
```
https://api.gunj-operator.example.com/api/v1
```

## Authentication

All API requests require authentication using JWT tokens:

```http
Authorization: Bearer <jwt-token>
```

## API Endpoints

### 1. Platform Management

#### 1.1 List Platforms
```http
GET /api/v1/platforms
```

Query Parameters:
- `namespace` (optional): Filter by namespace
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)
- `sort` (optional): Sort field (name, created, updated)
- `order` (optional): Sort order (asc, desc)
- `labels` (optional): Label selector (e.g., "env=prod,team=ops")

Response:
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "metadata": {
          "name": "production",
          "namespace": "monitoring",
          "uid": "123e4567-e89b-12d3-a456-426614174000",
          "creationTimestamp": "2025-06-12T10:00:00Z",
          "labels": {
            "env": "prod",
            "team": "platform"
          }
        },
        "spec": {
          "components": {
            "prometheus": {
              "enabled": true,
              "version": "v2.48.0"
            }
          }
        },
        "status": {
          "phase": "Ready",
          "message": "All components are running",
          "lastUpdated": "2025-06-12T10:05:00Z"
        }
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 45,
      "pages": 3
    }
  }
}
```

#### 1.2 Get Platform
```http
GET /api/v1/platforms/{name}?namespace={namespace}
```

Path Parameters:
- `name`: Platform name

Query Parameters:
- `namespace` (optional): Namespace (default: "default")

#### 1.3 Create Platform
```http
POST /api/v1/platforms
```

Request Body:
```json
{
  "metadata": {
    "name": "staging",
    "namespace": "staging",
    "labels": {
      "env": "staging",
      "team": "dev"
    }
  },
  "spec": {
    "components": {
      "prometheus": {
        "enabled": true,
        "version": "v2.48.0",
        "resources": {
          "requests": {
            "memory": "4Gi",
            "cpu": "1"
          },
          "limits": {
            "memory": "8Gi",
            "cpu": "2"
          }
        }
      },
      "grafana": {
        "enabled": true,
        "version": "10.2.0"
      }
    }
  }
}
```

#### 1.4 Update Platform
```http
PUT /api/v1/platforms/{name}?namespace={namespace}
```

#### 1.5 Patch Platform
```http
PATCH /api/v1/platforms/{name}?namespace={namespace}
```

Request Body (JSON Patch):
```json
[
  {
    "op": "replace",
    "path": "/spec/components/prometheus/version",
    "value": "v2.49.0"
  }
]
```

#### 1.6 Delete Platform
```http
DELETE /api/v1/platforms/{name}?namespace={namespace}
```

Query Parameters:
- `cascade` (optional): Delete dependent resources (default: true)
- `gracePeriod` (optional): Grace period in seconds (default: 30)

### 2. Component Management

#### 2.1 List Components
```http
GET /api/v1/platforms/{name}/components?namespace={namespace}
```

#### 2.2 Get Component
```http
GET /api/v1/platforms/{name}/components/{component}?namespace={namespace}
```

Component types: `prometheus`, `grafana`, `loki`, `tempo`

#### 2.3 Update Component
```http
PUT /api/v1/platforms/{name}/components/{component}?namespace={namespace}
```

#### 2.4 Enable/Disable Component
```http
POST /api/v1/platforms/{name}/components/{component}/enable?namespace={namespace}
POST /api/v1/platforms/{name}/components/{component}/disable?namespace={namespace}
```

### 3. Operations

#### 3.1 Backup Platform
```http
POST /api/v1/platforms/{name}/operations/backup?namespace={namespace}
```

Request Body:
```json
{
  "destination": "s3://backups/gunj-operator/",
  "includeData": true,
  "includeConfigs": true,
  "compression": "gzip"
}
```

#### 3.2 Restore Platform
```http
POST /api/v1/platforms/{name}/operations/restore?namespace={namespace}
```

Request Body:
```json
{
  "source": "s3://backups/gunj-operator/backup-20250612-100000.tar.gz",
  "overwrite": false,
  "components": ["prometheus", "grafana"]
}
```

#### 3.3 Upgrade Platform
```http
POST /api/v1/platforms/{name}/operations/upgrade?namespace={namespace}
```

Request Body:
```json
{
  "targetVersion": "v2.1.0",
  "strategy": "rolling",
  "backupFirst": true
}
```

#### 3.4 Scale Component
```http
POST /api/v1/platforms/{name}/components/{component}/scale?namespace={namespace}
```

Request Body:
```json
{
  "replicas": 3
}
```

### 4. Monitoring & Metrics

#### 4.1 Platform Metrics
```http
GET /api/v1/platforms/{name}/metrics?namespace={namespace}
```

Query Parameters:
- `start` (optional): Start time (ISO 8601)
- `end` (optional): End time (ISO 8601)
- `step` (optional): Query step (e.g., "5m", "1h")

#### 4.2 Platform Health
```http
GET /api/v1/platforms/{name}/health?namespace={namespace}
```

Response:
```json
{
  "success": true,
  "data": {
    "overall": "healthy",
    "components": {
      "prometheus": {
        "status": "healthy",
        "ready": true,
        "replicas": {
          "desired": 2,
          "ready": 2
        }
      },
      "grafana": {
        "status": "healthy",
        "ready": true,
        "replicas": {
          "desired": 1,
          "ready": 1
        }
      }
    },
    "lastCheck": "2025-06-12T10:30:00Z"
  }
}
```

#### 4.3 Platform Events
```http
GET /api/v1/platforms/{name}/events?namespace={namespace}
```

Query Parameters:
- `type` (optional): Event type filter
- `since` (optional): Events since timestamp
- `limit` (optional): Maximum events to return

### 5. Configuration Management

#### 5.1 Get Platform Configuration
```http
GET /api/v1/platforms/{name}/config?namespace={namespace}
```

#### 5.2 Update Platform Configuration
```http
PUT /api/v1/platforms/{name}/config?namespace={namespace}
```

#### 5.3 Validate Configuration
```http
POST /api/v1/platforms/validate
```

Request Body: Platform configuration to validate

### 6. Administrative Endpoints

#### 6.1 Operator Health
```http
GET /api/v1/health
```

#### 6.2 Operator Metrics
```http
GET /api/v1/metrics
```

Response: Prometheus format metrics

#### 6.3 Operator Version
```http
GET /api/v1/version
```

Response:
```json
{
  "version": "2.0.0",
  "gitCommit": "abc123def",
  "buildDate": "2025-06-12T08:00:00Z",
  "goVersion": "go1.21",
  "platform": "linux/amd64"
}
```

#### 6.4 API Documentation
```http
GET /api/v1/openapi
```

Response: OpenAPI 3.0 specification

### 7. User & Authentication

#### 7.1 Login
```http
POST /api/v1/auth/login
```

Request Body:
```json
{
  "username": "admin",
  "password": "password"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expiresIn": 3600,
    "refreshToken": "refresh_token_here"
  }
}
```

#### 7.2 Refresh Token
```http
POST /api/v1/auth/refresh
```

Request Body:
```json
{
  "refreshToken": "refresh_token_here"
}
```

#### 7.3 Logout
```http
POST /api/v1/auth/logout
```

#### 7.4 Get Current User
```http
GET /api/v1/auth/me
```

### 8. Webhooks

#### 8.1 List Webhooks
```http
GET /api/v1/webhooks
```

#### 8.2 Create Webhook
```http
POST /api/v1/webhooks
```

Request Body:
```json
{
  "name": "slack-notifications",
  "url": "https://hooks.slack.com/services/...",
  "events": ["platform.created", "platform.failed"],
  "active": true
}
```

#### 8.3 Update Webhook
```http
PUT /api/v1/webhooks/{id}
```

#### 8.4 Delete Webhook
```http
DELETE /api/v1/webhooks/{id}
```

### 9. Cost Management

#### 9.1 Get Cost Analysis
```http
GET /api/v1/platforms/{name}/cost?namespace={namespace}
```

Query Parameters:
- `period` (optional): Analysis period (day, week, month)
- `breakdown` (optional): Cost breakdown type (component, resource)

#### 9.2 Get Recommendations
```http
GET /api/v1/platforms/{name}/recommendations?namespace={namespace}
```

#### 9.3 Apply Recommendations
```http
POST /api/v1/platforms/{name}/recommendations/apply?namespace={namespace}
```

Request Body:
```json
{
  "recommendations": ["resource-optimization", "storage-cleanup"],
  "dryRun": false
}
```

## Response Format

All API responses follow a consistent format:

### Success Response
```json
{
  "success": true,
  "data": {
    // Response data
  },
  "meta": {
    "requestId": "req_123456",
    "timestamp": "2025-06-12T10:00:00Z",
    "version": "v1"
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "PLATFORM_NOT_FOUND",
    "message": "The requested platform does not exist",
    "details": {
      "platform": "production",
      "namespace": "monitoring"
    }
  },
  "meta": {
    "requestId": "req_123456",
    "timestamp": "2025-06-12T10:00:00Z",
    "version": "v1"
  }
}
```

## Error Handling

### HTTP Status Codes

- `200 OK`: Successful GET, PUT
- `201 Created`: Successful POST creating new resource
- `202 Accepted`: Request accepted for async processing
- `204 No Content`: Successful DELETE
- `400 Bad Request`: Invalid request format
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict (e.g., already exists)
- `422 Unprocessable Entity`: Validation error
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

### Error Codes

| Code | Description |
|------|-------------|
| `AUTH_INVALID` | Invalid authentication credentials |
| `AUTH_EXPIRED` | Authentication token expired |
| `PERMISSION_DENIED` | Insufficient permissions |
| `PLATFORM_NOT_FOUND` | Platform does not exist |
| `PLATFORM_EXISTS` | Platform already exists |
| `VALIDATION_ERROR` | Request validation failed |
| `OPERATION_FAILED` | Operation could not be completed |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `INTERNAL_ERROR` | Internal server error |

## Versioning Strategy

- API versions are specified in the URL path: `/api/v1`, `/api/v2`
- Breaking changes require a new major version
- Deprecated endpoints include `Sunset` header with deprecation date
- Minimum 6-month deprecation period for any endpoint
- Version compatibility matrix maintained in documentation

## Rate Limiting

- Rate limits applied per authenticated user
- Default: 1000 requests per hour
- Burst: 100 requests per minute
- Headers included in response:
  - `X-RateLimit-Limit`: Maximum requests
  - `X-RateLimit-Remaining`: Remaining requests
  - `X-RateLimit-Reset`: Reset timestamp

## Pagination

Standard pagination parameters:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20, max: 100)
- `sort`: Sort field
- `order`: Sort order (asc/desc)

Pagination metadata in response:
```json
{
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "pages": 5,
    "hasNext": true,
    "hasPrev": false
  }
}
```