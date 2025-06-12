# Rate Limiting Architecture Diagram

```mermaid
graph TB
    subgraph "Client Layer"
        A[Web UI]
        B[API Client]
        C[CLI Tool]
        D[CI/CD Pipeline]
    end
    
    subgraph "API Gateway Layer"
        E[Load Balancer]
        F[Rate Limit Middleware]
        G[Auth Middleware]
    end
    
    subgraph "Rate Limiting Core"
        H[Token Bucket Engine]
        I[Sliding Window Engine]
        J[Adaptive Engine]
        K[GraphQL Complexity]
    end
    
    subgraph "Storage Layer"
        L[(Redis Cluster)]
        M[(Local Cache)]
        N[(PostgreSQL)]
    end
    
    subgraph "Monitoring"
        O[Prometheus]
        P[Grafana]
        Q[AlertManager]
    end
    
    A --> E
    B --> E
    C --> E
    D --> E
    
    E --> F
    F --> G
    
    F --> H
    F --> I
    F --> J
    F --> K
    
    H --> L
    I --> L
    J --> L
    
    H --> M
    I --> M
    J --> M
    
    K --> N
    
    F --> O
    O --> P
    O --> Q
    
    L -.->|Failover| M
    
    style F fill:#ff9999
    style L fill:#99ccff
    style O fill:#99ff99
```

## Rate Limiting Flow

```mermaid
sequenceDiagram
    participant Client
    participant API Gateway
    participant Rate Limiter
    participant Redis
    participant API Service
    
    Client->>API Gateway: API Request
    API Gateway->>Rate Limiter: Check rate limit
    
    Rate Limiter->>Redis: Get token bucket
    
    alt Redis Available
        Redis-->>Rate Limiter: Current tokens
        Rate Limiter->>Rate Limiter: Calculate tokens
        
        alt Tokens Available
            Rate Limiter->>Redis: Update tokens
            Rate Limiter-->>API Gateway: Allowed
            API Gateway->>API Service: Forward request
            API Service-->>API Gateway: Response
            API Gateway-->>Client: 200 OK + Headers
        else No Tokens
            Rate Limiter-->>API Gateway: Denied
            API Gateway-->>Client: 429 Too Many Requests
        end
    else Redis Unavailable
        Rate Limiter->>Rate Limiter: Use local cache
        Note over Rate Limiter: 80% of normal limit
        Rate Limiter-->>API Gateway: Decision
    end
```

## Quota Management Flow

```mermaid
stateDiagram-v2
    [*] --> CheckQuota: Resource Request
    
    CheckQuota --> QuotaAvailable: Under Limit
    CheckQuota --> QuotaExceeded: Over Limit
    CheckQuota --> QuotaWarning: Near Limit (>80%)
    
    QuotaAvailable --> ConsumeQuota: Proceed
    QuotaWarning --> ConsumeQuota: Proceed + Alert
    QuotaExceeded --> [*]: Reject
    
    ConsumeQuota --> UpdateUsage: Success
    ConsumeQuota --> RollbackQuota: Failed
    
    UpdateUsage --> RecordMetrics
    RollbackQuota --> RecordMetrics
    
    RecordMetrics --> [*]
    
    state QuotaWarning {
        [*] --> NotifyUser
        NotifyUser --> SuggestUpgrade
    }
```

## Implementation Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Rate Limiting System                     │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐      │
│  │   Ingress   │  │   Strategy   │  │   GraphQL     │      │
│  │  Middleware │  │   Selector   │  │  Complexity   │      │
│  └──────┬──────┘  └──────┬───────┘  └───────┬───────┘      │
│         │                 │                   │               │
│         ▼                 ▼                   ▼               │
│  ┌─────────────────────────────────────────────────┐        │
│  │              Rate Limiter Core                   │        │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐│        │
│  │  │  Token  │ │ Sliding │ │  Fixed  │ │Adaptive││        │
│  │  │ Bucket  │ │ Window  │ │ Window  │ │ Limiter││        │
│  │  └─────────┘ └─────────┘ └─────────┘ └────────┘│        │
│  └─────────────────────────────────────────────────┘        │
│                           │                                   │
│                           ▼                                   │
│  ┌─────────────────────────────────────────────────┐        │
│  │              Storage Abstraction                 │        │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐│        │
│  │  │  Redis  │ │  Local  │ │Postgres │ │ Memory ││        │
│  │  │ Driver  │ │  Cache  │ │ Driver  │ │ Store  ││        │
│  │  └─────────┘ └─────────┘ └─────────┘ └────────┘│        │
│  └─────────────────────────────────────────────────┘        │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```