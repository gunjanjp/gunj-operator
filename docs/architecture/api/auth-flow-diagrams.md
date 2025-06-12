# Authentication Flow Diagrams

This document provides visual representations of the authentication and authorization flows in the Gunj Operator.

## 1. JWT Authentication Flow

```mermaid
sequenceDiagram
    participant Client
    participant API Gateway
    participant Auth Service
    participant User Store
    participant Token Store
    participant API Service
    
    Client->>API Gateway: POST /auth/login {username, password}
    API Gateway->>Auth Service: Forward auth request
    Auth Service->>User Store: Validate credentials
    User Store-->>Auth Service: User details + roles
    
    alt Valid credentials
        Auth Service->>Auth Service: Generate JWT tokens
        Auth Service->>Token Store: Store refresh token
        Auth Service-->>API Gateway: {access_token, refresh_token}
        API Gateway-->>Client: 200 OK with tokens
    else Invalid credentials
        Auth Service-->>API Gateway: 401 Unauthorized
        API Gateway-->>Client: 401 {error: "Invalid credentials"}
    end
    
    Note over Client: Subsequent API calls
    
    Client->>API Gateway: GET /api/v1/platforms
    Note right of Client: Authorization: Bearer {access_token}
    
    API Gateway->>API Gateway: Validate JWT signature
    API Gateway->>API Gateway: Check token expiration
    API Gateway->>API Service: Forward with user context
    API Service->>API Service: Check permissions
    API Service-->>API Gateway: Response data
    API Gateway-->>Client: 200 OK with data
```

## 2. OIDC Authentication Flow

```mermaid
sequenceDiagram
    participant Browser
    participant Gunj UI
    participant Gunj API
    participant OIDC Provider
    participant Token Service
    
    Browser->>Gunj UI: Click "Login with Okta"
    Gunj UI->>Gunj API: GET /auth/oidc/authorize?provider=okta
    Gunj API->>Gunj API: Generate state & nonce
    Gunj API-->>Browser: 302 Redirect to OIDC
    
    Browser->>OIDC Provider: GET /authorize?client_id=...&state=...
    OIDC Provider->>Browser: Show login page
    Browser->>OIDC Provider: Submit credentials
    OIDC Provider->>OIDC Provider: Validate user
    OIDC Provider-->>Browser: 302 Redirect with code
    
    Browser->>Gunj API: GET /auth/oidc/callback?code=...&state=...
    Gunj API->>Gunj API: Validate state
    Gunj API->>OIDC Provider: POST /token {code, client_secret}
    OIDC Provider-->>Gunj API: {id_token, access_token}
    
    Gunj API->>Gunj API: Validate ID token
    Gunj API->>Gunj API: Extract user info
    Gunj API->>Gunj API: Map groups to roles
    Gunj API->>Token Service: Generate internal JWT
    Gunj API-->>Browser: Set cookie + redirect
    
    Browser->>Gunj UI: Load with auth cookie
    Gunj UI->>Gunj API: API calls with JWT
```

## 3. API Key Authentication Flow

```mermaid
flowchart TB
    A[API Request] --> B{Has API Key?}
    B -->|No| C[401 Unauthorized]
    B -->|Yes| D[Parse API Key]
    
    D --> E{Valid Format?}
    E -->|No| F[400 Bad Request]
    E -->|Yes| G[Calculate Hash]
    
    G --> H[(Database Lookup)]
    H --> I{Key Found?}
    I -->|No| J[401 Invalid Key]
    I -->|Yes| K{Key Active?}
    
    K -->|No| L[401 Key Expired/Revoked]
    K -->|Yes| M[Check IP Whitelist]
    
    M --> N{IP Allowed?}
    N -->|No| O[403 IP Not Allowed]
    N -->|Yes| P[Check Rate Limit]
    
    P --> Q{Within Limit?}
    Q -->|No| R[429 Too Many Requests]
    Q -->|Yes| S[Load Permissions]
    
    S --> T[Authorize Request]
    T --> U{Has Permission?}
    U -->|No| V[403 Forbidden]
    U -->|Yes| W[Process Request]
    
    W --> X[Update Usage Stats]
    X --> Y[Return Response]
```

## 4. Token Refresh Flow

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Auth Service
    participant Token Store
    participant Blacklist
    
    Note over Client: Access token expired
    
    Client->>API: POST /auth/refresh
    Note right of Client: {refresh_token: "xxx"}
    
    API->>Auth Service: Validate refresh token
    Auth Service->>Token Store: Lookup token
    
    alt Token not found
        Token Store-->>Auth Service: Not found
        Auth Service-->>API: 401 Invalid token
        API-->>Client: 401 Unauthorized
    else Token found
        Token Store-->>Auth Service: Token metadata
        Auth Service->>Auth Service: Check expiration
        
        alt Token expired
            Auth Service-->>API: 401 Token expired
            API-->>Client: 401 Unauthorized
        else Token valid
            Auth Service->>Blacklist: Check if revoked
            
            alt Token revoked
                Blacklist-->>Auth Service: Revoked
                Auth Service-->>API: 401 Token revoked
                API-->>Client: 401 Unauthorized
            else Token active
                Auth Service->>Auth Service: Generate new tokens
                Auth Service->>Token Store: Store new refresh token
                Auth Service->>Token Store: Revoke old refresh token
                Auth Service-->>API: New token pair
                API-->>Client: 200 {access_token, refresh_token}
            end
        end
    end
```

## 5. RBAC Authorization Flow

```mermaid
flowchart TB
    A[Authenticated Request] --> B[Extract User Context]
    B --> C{Is Admin?}
    C -->|Yes| D[Allow All Actions]
    
    C -->|No| E[Get Resource & Action]
    E --> F[Build Permission String]
    
    F --> G{Check Direct Permissions}
    G -->|Found| H[Allow]
    G -->|Not Found| I{Check Role Permissions}
    
    I -->|Found| H
    I -->|Not Found| J{Check Namespace Scope}
    
    J -->|Has Access| K[Check Namespace Permissions]
    K -->|Allowed| H
    K -->|Denied| L[Deny]
    J -->|No Access| L
    
    H --> M[Audit Success]
    L --> N[Audit Failure]
    
    M --> O[Process Request]
    N --> P[403 Forbidden]
    
    style D fill:#90EE90
    style H fill:#90EE90
    style L fill:#FFB6C1
    style P fill:#FFB6C1
```

## 6. Service Account Authentication

```mermaid
sequenceDiagram
    participant Pod
    participant Gunj API
    participant K8s API
    participant RBAC
    
    Pod->>Pod: Mount SA token
    Note over Pod: /var/run/secrets/kubernetes.io/serviceaccount/token
    
    Pod->>Gunj API: API Request
    Note right of Pod: Authorization: Bearer {sa-token}
    
    Gunj API->>K8s API: TokenReview
    Note right of Gunj API: Validate SA token
    
    K8s API->>K8s API: Verify token signature
    K8s API->>K8s API: Check token expiration
    K8s API-->>Gunj API: Authentication result
    
    alt Token valid
        Gunj API->>RBAC: Get SA permissions
        RBAC-->>Gunj API: Roles and bindings
        Gunj API->>Gunj API: Map to internal permissions
        Gunj API->>Gunj API: Process request
        Gunj API-->>Pod: 200 OK
    else Token invalid
        Gunj API-->>Pod: 401 Unauthorized
    end
```

## 7. Multi-Factor Authentication Flow

```mermaid
stateDiagram-v2
    [*] --> UsernamePassword: User login
    
    UsernamePassword --> CheckRiskScore: Valid credentials
    UsernamePassword --> [*]: Invalid credentials
    
    CheckRiskScore --> DirectLogin: Low risk (<25)
    CheckRiskScore --> RequireMFA: High risk (≥25)
    
    RequireMFA --> SelectMethod: Choose MFA method
    
    SelectMethod --> TOTP: Authenticator app
    SelectMethod --> SMS: Text message
    SelectMethod --> Email: Email code
    SelectMethod --> WebAuthn: Security key
    
    TOTP --> VerifyCode: Enter 6-digit code
    SMS --> VerifyCode: Enter SMS code
    Email --> VerifyCode: Enter email code
    WebAuthn --> VerifyKey: Touch security key
    
    VerifyCode --> MFASuccess: Valid code
    VerifyCode --> MFAFailure: Invalid code
    VerifyKey --> MFASuccess: Valid signature
    VerifyKey --> MFAFailure: Invalid signature
    
    MFAFailure --> SelectMethod: Retry
    MFASuccess --> DirectLogin: MFA passed
    
    DirectLogin --> GenerateTokens: Create session
    GenerateTokens --> [*]: Logged in
    
    state CheckRiskScore {
        [*] --> CalculateRisk
        CalculateRisk --> EvaluateFactors
        
        state EvaluateFactors {
            NewDevice: +10 points
            NewLocation: +20 points
            ImpossibleTravel: +50 points
            SuspiciousIP: +30 points
        }
    }
```

## 8. Cross-Service Authentication

```mermaid
graph TB
    subgraph "Gunj Operator Cluster"
        A[User] --> B[Gunj UI]
        B --> C[Gunj API Gateway]
        C --> D[Auth Service]
        
        C --> E[Platform Service]
        C --> F[Metrics Service]
        C --> G[Cost Service]
    end
    
    subgraph "Observability Cluster"
        H[Prometheus]
        I[Grafana]
        J[Loki]
    end
    
    D -.->|Issue JWT| B
    B -.->|Pass JWT| C
    
    E -->|Service Token| H
    F -->|Service Token| H
    F -->|Service Token| I
    G -->|Service Token| H
    
    style D fill:#FFE4B5
    style H fill:#E6E6FA
    style I fill:#E6E6FA
    style J fill:#E6E6FA
```

## 9. Session Management Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Created: User login
    
    Created --> Active: First use
    
    Active --> Active: Activity within timeout
    Active --> Warning: Approaching timeout
    Active --> Expired: Timeout reached
    Active --> Revoked: Admin action
    Active --> Locked: Security event
    
    Warning --> Active: User activity
    Warning --> Expired: No activity
    
    Locked --> Active: Admin unlock
    Locked --> Revoked: Permanent block
    
    Expired --> [*]: Session end
    Revoked --> [*]: Session end
    
    note right of Active
        - Sliding window timeout
        - Activity tracking
        - Concurrent session limits
    end note
    
    note right of Locked
        - Too many failed attempts
        - Suspicious activity detected
        - Geographic anomaly
    end note
```

## 10. Zero Trust Security Model

```mermaid
graph TB
    subgraph "External"
        A[User]
        B[API Client]
        C[CI/CD Pipeline]
    end
    
    subgraph "Edge Security"
        D[WAF]
        E[DDoS Protection]
        F[Rate Limiter]
    end
    
    subgraph "Authentication Layer"
        G[Identity Provider]
        H[MFA Service]
        I[Token Service]
    end
    
    subgraph "Authorization Layer"
        J[Policy Engine]
        K[RBAC Service]
        L[Audit Logger]
    end
    
    subgraph "API Layer"
        M[API Gateway]
        N[Service Mesh]
    end
    
    subgraph "Services"
        O[Platform Service]
        P[Metrics Service]
        Q[Cost Service]
    end
    
    A --> D
    B --> D
    C --> D
    
    D --> E
    E --> F
    F --> M
    
    M --> G
    G --> H
    H --> I
    
    I --> J
    J --> K
    K --> L
    
    M --> N
    N --> O
    N --> P
    N --> Q
    
    style G fill:#FFE4B5
    style J fill:#E6E6FA
    style M fill:#98FB98
```

## Implementation Notes

1. **Token Storage**: All tokens are stored encrypted at rest
2. **Transport Security**: All communication uses TLS 1.3
3. **Audit Logging**: Every authentication event is logged
4. **Rate Limiting**: Applied at multiple layers
5. **Session Management**: Configurable timeout with sliding window
6. **Key Rotation**: Automatic rotation for signing keys
7. **Monitoring**: Real-time alerts for auth anomalies