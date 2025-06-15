# [API Name] Reference

> **Version**: v1  
> **Base URL**: `https://api.gunj-operator.io`  
> **Authentication**: Bearer Token / API Key

## Overview

[Brief description of the API, its purpose, and main capabilities]

### Key Features

- Feature 1
- Feature 2
- Feature 3

### API Versioning

This API uses URL-based versioning. The current version is `v1`.

```
https://api.gunj-operator.io/api/v1/
```

## Authentication

### Bearer Token

Include the authentication token in the Authorization header:

```http
Authorization: Bearer <your-token>
```

### Example

```bash
curl -H "Authorization: Bearer ${TOKEN}" \
  https://api.gunj-operator.io/api/v1/platforms
```

## Common Headers

| Header | Description | Required |
|--------|-------------|----------|
| `Authorization` | Bearer token for authentication | Yes |
| `Content-Type` | Request content type (usually `application/json`) | Yes for POST/PUT |
| `Accept` | Response content type | No (default: `application/json`) |
| `X-Request-ID` | Unique request identifier for tracing | No |

## Rate Limiting

- **Rate Limit**: 1000 requests per hour
- **Rate Limit Headers**:
  - `X-RateLimit-Limit`: Maximum requests per hour
  - `X-RateLimit-Remaining`: Remaining requests
  - `X-RateLimit-Reset`: UTC epoch seconds when limit resets

## Error Responses

### Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "field": "Additional context"
    }
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Internal server error |

## Endpoints

### Resource: Platforms

#### List Platforms

Returns a list of all observability platforms.

**Endpoint**: `GET /api/v1/platforms`

**Query Parameters**:

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `namespace` | string | Filter by namespace | all |
| `labelSelector` | string | Kubernetes label selector | none |
| `page` | integer | Page number | 1 |
| `limit` | integer | Items per page | 20 |

**Response**:

```json
{
  "items": [
    {
      "name": "production",
      "namespace": "monitoring",
      "status": {
        "phase": "Ready",
        "message": "All components running"
      },
      "created_at": "2025-06-12T10:00:00Z",
      "updated_at": "2025-06-12T15:00:00Z"
    }
  ],
  "metadata": {
    "page": 1,
    "limit": 20,
    "total": 42
  }
}
```

**Example Request**:

```bash
curl -X GET \
  'https://api.gunj-operator.io/api/v1/platforms?namespace=production&limit=10' \
  -H 'Authorization: Bearer ${TOKEN}'
```

#### Get Platform

Retrieve details of a specific platform.

**Endpoint**: `GET /api/v1/platforms/{name}`

**Path Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Platform name |

**Query Parameters**:

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `namespace` | string | Platform namespace | default |

**Response**:

```json
{
  "name": "production",
  "namespace": "monitoring",
  "spec": {
    "components": {
      "prometheus": {
        "enabled": true,
        "version": "v2.48.0",
        "replicas": 3
      }
    }
  },
  "status": {
    "phase": "Ready",
    "conditions": [
      {
        "type": "Ready",
        "status": "True",
        "lastTransitionTime": "2025-06-12T15:00:00Z",
        "reason": "AllComponentsReady",
        "message": "All components are running"
      }
    ]
  }
}
```

**Example Request**:

```bash
curl -X GET \
  'https://api.gunj-operator.io/api/v1/platforms/production?namespace=monitoring' \
  -H 'Authorization: Bearer ${TOKEN}'
```

#### Create Platform

Create a new observability platform.

**Endpoint**: `POST /api/v1/platforms`

**Request Body**:

```json
{
  "name": "staging",
  "namespace": "staging",
  "spec": {
    "components": {
      "prometheus": {
        "enabled": true,
        "version": "v2.48.0",
        "replicas": 2,
        "resources": {
          "requests": {
            "memory": "2Gi",
            "cpu": "1"
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

**Response**: `201 Created`

```json
{
  "name": "staging",
  "namespace": "staging",
  "status": {
    "phase": "Creating",
    "message": "Platform is being created"
  },
  "created_at": "2025-06-12T16:00:00Z"
}
```

**Example Request**:

```bash
curl -X POST \
  'https://api.gunj-operator.io/api/v1/platforms' \
  -H 'Authorization: Bearer ${TOKEN}' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "staging",
    "namespace": "staging",
    "spec": {
      "components": {
        "prometheus": {
          "enabled": true
        }
      }
    }
  }'
```

#### Update Platform

Update an existing platform configuration.

**Endpoint**: `PUT /api/v1/platforms/{name}`

**Path Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Platform name |

**Query Parameters**:

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `namespace` | string | Platform namespace | default |

**Request Body**: Same as Create Platform

**Response**: `200 OK`

#### Delete Platform

Delete an observability platform.

**Endpoint**: `DELETE /api/v1/platforms/{name}`

**Path Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Platform name |

**Query Parameters**:

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `namespace` | string | Platform namespace | default |

**Response**: `204 No Content`

**Example Request**:

```bash
curl -X DELETE \
  'https://api.gunj-operator.io/api/v1/platforms/staging?namespace=staging' \
  -H 'Authorization: Bearer ${TOKEN}'
```

### Resource: Operations

#### Backup Platform

Create a backup of platform data.

**Endpoint**: `POST /api/v1/platforms/{name}/operations/backup`

**Request Body**:

```json
{
  "destination": "s3://backups/production-2025-06-12",
  "components": ["prometheus", "grafana"],
  "compression": true
}
```

**Response**: `202 Accepted`

```json
{
  "operation_id": "backup-550e8400",
  "status": "InProgress",
  "message": "Backup started"
}
```

## Pagination

List endpoints support pagination:

```json
{
  "items": [...],
  "metadata": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5,
    "next_page": "/api/v1/platforms?page=2",
    "previous_page": null
  }
}
```

## Filtering and Sorting

### Filtering

Use query parameters for filtering:

```
GET /api/v1/platforms?namespace=production&status=Ready
```

### Sorting

Use the `sort` parameter:

```
GET /api/v1/platforms?sort=name:asc,created_at:desc
```

## Webhooks

### Webhook Events

The API can send webhooks for these events:

- `platform.created`
- `platform.updated`
- `platform.deleted`
- `platform.status_changed`

### Webhook Payload

```json
{
  "event": "platform.status_changed",
  "timestamp": "2025-06-12T16:00:00Z",
  "data": {
    "platform": "production",
    "namespace": "monitoring",
    "old_status": "Creating",
    "new_status": "Ready"
  }
}
```

## SDK Examples

### Go

```go
import "github.com/gunjanjp/gunj-operator/pkg/client"

client := client.New("https://api.gunj-operator.io", token)

// List platforms
platforms, err := client.Platforms().List(ctx, "production")

// Create platform
platform := &api.Platform{
    Name: "staging",
    Spec: api.PlatformSpec{
        Components: api.Components{
            Prometheus: &api.PrometheusSpec{
                Enabled: true,
            },
        },
    },
}
created, err := client.Platforms().Create(ctx, platform)
```

### Python

```python
from gunj_operator import Client

client = Client(base_url="https://api.gunj-operator.io", token=token)

# List platforms
platforms = client.platforms.list(namespace="production")

# Create platform
platform = client.platforms.create(
    name="staging",
    namespace="staging",
    spec={
        "components": {
            "prometheus": {
                "enabled": True
            }
        }
    }
)
```

### JavaScript/TypeScript

```typescript
import { GunjClient } from '@gunjanjp/gunj-operator-sdk';

const client = new GunjClient({
  baseURL: 'https://api.gunj-operator.io',
  token: process.env.GUNJ_TOKEN
});

// List platforms
const platforms = await client.platforms.list({
  namespace: 'production'
});

// Create platform
const platform = await client.platforms.create({
  name: 'staging',
  namespace: 'staging',
  spec: {
    components: {
      prometheus: {
        enabled: true
      }
    }
  }
});
```

## Changelog

### v1.0.0 (2025-06-12)
- Initial API release
- Platform CRUD operations
- Basic authentication
- Webhook support

## Support

- **Documentation**: [https://docs.gunj-operator.io](https://docs.gunj-operator.io)
- **Issues**: [GitHub Issues](https://github.com/gunjanjp/gunj-operator/issues)
- **Community**: [Slack Channel](https://gunjanjp.slack.com)
- **Email**: api-support@gunjanjp.com
