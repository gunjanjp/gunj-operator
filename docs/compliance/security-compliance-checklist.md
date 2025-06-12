# Security Compliance Checklist
## Gunj Operator - Enterprise Observability Platform

**Document Version**: 1.0  
**Date**: June 12, 2025  
**Classification**: Security Compliance  
**Status**: Phase 1.4.2 - CNCF Compliance Planning  

---

## Executive Summary

This document establishes a comprehensive security compliance framework for the Gunj Operator project, mapping requirements to industry standards and providing actionable checklists for implementation and verification.

---

## Security Standards Mapping

### 1. CIS Kubernetes Benchmark v1.8.0

#### Control Plane Security
- [ ] **1.1.1** Ensure API server pod specification file permissions (644 or restrictive)
- [ ] **1.1.2** Ensure API server pod specification file ownership (root:root)
- [ ] **1.2.1** Ensure --anonymous-auth is set to false
- [ ] **1.2.2** Ensure --token-auth-file is not set
- [ ] **1.2.3** Ensure --DenyServiceExternalIPs is not set
- [ ] **1.2.4** Ensure --kubelet-https is set to true
- [ ] **1.2.5** Ensure --kubelet-client-certificate and --kubelet-client-key are set
- [ ] **1.2.6** Ensure --kubelet-certificate-authority is set
- [ ] **1.2.7** Ensure --authorization-mode includes RBAC
- [ ] **1.2.8** Ensure --authorization-mode includes Node
- [ ] **1.2.9** Ensure --enable-admission-plugins includes EventRateLimit
- [ ] **1.2.10** Ensure --enable-admission-plugins includes AlwaysPullImages

#### Pod Security Standards
- [ ] **5.1.1** Ensure cluster uses Pod Security Standards
- [ ] **5.1.2** Minimize container image privileges
- [ ] **5.1.3** Minimize host network usage
- [ ] **5.1.4** Ensure containers do not run as root
- [ ] **5.1.5** Ensure default seccomp profile is set
- [ ] **5.1.6** Apply Security Context to pods and containers
- [ ] **5.1.7** Ensure Network Policies are applied
- [ ] **5.1.8** Avoid use of privileged containers

### 2. NIST Cybersecurity Framework

#### Identify (ID)
- [ ] **ID.AM-1** Physical devices and systems inventoried
  ```yaml
  # Implementation: Asset inventory
  inventory:
    operator:
      components:
        - name: controller
          version: v2.0.0
          criticality: high
        - name: api-server
          version: v2.0.0
          criticality: high
  ```
- [ ] **ID.AM-2** Software platforms and applications inventoried
- [ ] **ID.AM-3** Organizational communication mapped
- [ ] **ID.AM-4** External information systems catalogued
- [ ] **ID.AM-5** Resources prioritized based on criticality
- [ ] **ID.AM-6** Cybersecurity roles and responsibilities established

#### Protect (PR)
- [ ] **PR.AC-1** Identities and credentials managed
  ```yaml
  # RBAC configuration
  apiVersion: rbac.authorization.k8s.io/v1
  kind: ClusterRole
  metadata:
    name: gunj-operator-role
  rules:
  - apiGroups: [""]
    resources: ["pods", "services"]
    verbs: ["get", "list", "watch"]
  ```
- [ ] **PR.AC-2** Physical access to assets managed
- [ ] **PR.AC-3** Remote access managed
- [ ] **PR.AC-4** Access permissions managed (least privilege)
- [ ] **PR.AC-5** Network integrity protected
- [ ] **PR.AC-6** Identities proofed and bound to credentials
- [ ] **PR.AC-7** Authentication mechanisms used

#### Detect (DE)
- [ ] **DE.AE-1** Baseline of operations established
- [ ] **DE.AE-2** Detected events analyzed
- [ ] **DE.AE-3** Event data aggregated and correlated
- [ ] **DE.AE-4** Impact of events determined
- [ ] **DE.AE-5** Incident alert thresholds established

#### Respond (RS)
- [ ] **RS.RP-1** Response plan executed during/after incident
- [ ] **RS.CO-1** Personnel know roles and operations
- [ ] **RS.CO-2** Incident information shared
- [ ] **RS.CO-3** Coordination with stakeholders
- [ ] **RS.AN-1** Notifications from detection systems investigated

#### Recover (RC)
- [ ] **RC.RP-1** Recovery plan executed
- [ ] **RC.IM-1** Lessons learned incorporated
- [ ] **RC.IM-2** Recovery strategies updated
- [ ] **RC.CO-1** Public relations managed
- [ ] **RC.CO-2** Reputation restored

### 3. OWASP Top 10 API Security 2023

#### API1:2023 - Broken Object Level Authorization
- [ ] **Implementation**: Object-level authorization checks
  ```go
  func (h *Handler) GetPlatform(c *gin.Context) {
      platformID := c.Param("id")
      userID := c.GetString("userID")
      
      // Check authorization
      if !h.authz.CanAccess(userID, platformID) {
          c.JSON(403, gin.H{"error": "forbidden"})
          return
      }
  }
  ```

#### API2:2023 - Broken Authentication
- [ ] **Strong authentication mechanisms**
- [ ] **Token expiration and rotation**
- [ ] **Multi-factor authentication support**
- [ ] **Password complexity requirements**
- [ ] **Account lockout mechanisms**

#### API3:2023 - Broken Object Property Level Authorization
- [ ] **Field-level authorization**
- [ ] **Data filtering based on user roles**
- [ ] **Sensitive field masking**

#### API4:2023 - Unrestricted Resource Consumption
- [ ] **Rate limiting implementation**
- [ ] **Request size limits**
- [ ] **Pagination for list operations**
- [ ] **Timeout configurations**

#### API5:2023 - Broken Function Level Authorization
- [ ] **Function-level access control**
- [ ] **Admin endpoint protection**
- [ ] **Role-based function access**

#### API6:2023 - Unrestricted Access to Sensitive Business Flows
- [ ] **Business flow rate limiting**
- [ ] **CAPTCHA for sensitive operations**
- [ ] **Progressive delays**

#### API7:2023 - Server Side Request Forgery
- [ ] **URL validation and whitelisting**
- [ ] **Network segmentation**
- [ ] **Disable unnecessary URL schemas**

#### API8:2023 - Security Misconfiguration
- [ ] **Secure defaults**
- [ ] **Security headers**
- [ ] **Error handling without stack traces**
- [ ] **Updated dependencies**

#### API9:2023 - Improper Inventory Management
- [ ] **API documentation**
- [ ] **Version management**
- [ ] **Endpoint inventory**
- [ ] **Deprecation process**

#### API10:2023 - Unsafe Consumption of APIs
- [ ] **Input validation**
- [ ] **TLS verification**
- [ ] **API response validation**
- [ ] **Timeout and retry logic**

---

## Container Security Checklist

### Image Security
- [ ] **Base Image Selection**
  ```dockerfile
  # GOOD: Distroless base
  FROM gcr.io/distroless/static:nonroot
  
  # GOOD: Minimal Alpine
  FROM alpine:3.19
  RUN apk --no-cache add ca-certificates
  
  # BAD: Full OS image
  # FROM ubuntu:latest
  ```

- [ ] **Multi-stage Builds**
  ```dockerfile
  # Build stage
  FROM golang:1.21-alpine AS builder
  WORKDIR /build
  COPY . .
  RUN CGO_ENABLED=0 go build -o operator
  
  # Runtime stage
  FROM scratch
  COPY --from=builder /build/operator /
  ENTRYPOINT ["/operator"]
  ```

- [ ] **Non-root User**
  ```dockerfile
  USER 65532:65532
  ```

- [ ] **Read-only Root Filesystem**
  ```yaml
  securityContext:
    readOnlyRootFilesystem: true
  ```

### Build Security
- [ ] **Image Scanning in CI**
  ```yaml
  - name: Run Trivy vulnerability scanner
    uses: aquasecurity/trivy-action@master
    with:
      image-ref: ${{ env.IMAGE }}
      format: 'sarif'
      exit-code: '1'
      severity: 'CRITICAL,HIGH'
  ```

- [ ] **SBOM Generation**
  ```yaml
  - name: Generate SBOM
    uses: anchore/sbom-action@v0
    with:
      image: ${{ env.IMAGE }}
      format: spdx-json
  ```

- [ ] **Image Signing**
  ```bash
  # Sign with Cosign
  cosign sign --key cosign.key ${IMAGE}
  
  # Verify signature
  cosign verify --key cosign.pub ${IMAGE}
  ```

### Runtime Security
- [ ] **Pod Security Context**
  ```yaml
  apiVersion: v1
  kind: Pod
  spec:
    securityContext:
      runAsNonRoot: true
      runAsUser: 65532
      fsGroup: 65532
      seccompProfile:
        type: RuntimeDefault
    containers:
    - name: operator
      securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities:
          drop:
          - ALL
  ```

- [ ] **Resource Limits**
  ```yaml
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "512Mi"
      cpu: "500m"
  ```

- [ ] **Network Policies**
  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: NetworkPolicy
  metadata:
    name: gunj-operator-netpol
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
        port: 8443
    egress:
    - to:
      - namespaceSelector: {}
      ports:
      - protocol: TCP
        port: 443
  ```

---

## Secrets Management

### Secret Storage
- [ ] **No Hardcoded Secrets**
  ```go
  // BAD
  const apiKey = "sk-1234567890abcdef"
  
  // GOOD
  apiKey := os.Getenv("API_KEY")
  ```

- [ ] **Kubernetes Secrets**
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: api-credentials
  type: Opaque
  stringData:
    api-key: ${API_KEY}
  ```

- [ ] **External Secret Operators**
  ```yaml
  apiVersion: external-secrets.io/v1beta1
  kind: ExternalSecret
  metadata:
    name: vault-secret
  spec:
    secretStoreRef:
      name: vault-backend
      kind: SecretStore
    target:
      name: api-credentials
    data:
    - secretKey: api-key
      remoteRef:
        key: secret/data/api
        property: key
  ```

### Secret Rotation
- [ ] **Automated Rotation**
- [ ] **Rotation Notifications**
- [ ] **Grace Period Handling**
- [ ] **Audit Trail**

---

## Network Security

### TLS Configuration
- [ ] **TLS 1.2+ Only**
  ```go
  tlsConfig := &tls.Config{
      MinVersion: tls.VersionTLS12,
      CipherSuites: []uint16{
          tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
          tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
          tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
          tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
      },
  }
  ```

- [ ] **Certificate Management**
  ```yaml
  apiVersion: cert-manager.io/v1
  kind: Certificate
  metadata:
    name: gunj-operator-tls
  spec:
    secretName: gunj-operator-tls
    issuerRef:
      name: letsencrypt-prod
      kind: ClusterIssuer
    dnsNames:
    - gunj-operator.example.com
  ```

### Service Mesh Integration
- [ ] **mTLS Between Services**
  ```yaml
  apiVersion: security.istio.io/v1beta1
  kind: PeerAuthentication
  metadata:
    name: default
  spec:
    mtls:
      mode: STRICT
  ```

- [ ] **Authorization Policies**
  ```yaml
  apiVersion: security.istio.io/v1beta1
  kind: AuthorizationPolicy
  metadata:
    name: gunj-operator
  spec:
    selector:
      matchLabels:
        app: gunj-operator
    rules:
    - from:
      - source:
          principals: ["cluster.local/ns/gunj-system/sa/gunj-api"]
      to:
      - operation:
          methods: ["GET", "POST"]
  ```

---

## Authentication & Authorization

### Authentication Methods
- [ ] **JWT Implementation**
  ```go
  func ValidateJWT(tokenString string) (*Claims, error) {
      token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
          if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
              return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
          }
          return []byte(jwtSecret), nil
      })
      
      if claims, ok := token.Claims.(*Claims); ok && token.Valid {
          return claims, nil
      }
      return nil, err
  }
  ```

- [ ] **OAuth2/OIDC**
  ```yaml
  auth:
    oidc:
      issuer: https://auth.example.com
      clientId: gunj-operator
      clientSecret: ${OIDC_CLIENT_SECRET}
      redirectUrl: https://gunj-operator.example.com/callback
      scopes:
        - openid
        - profile
        - email
  ```

- [ ] **mTLS Client Certificates**
  ```go
  tlsConfig := &tls.Config{
      ClientAuth: tls.RequireAndVerifyClientCert,
      ClientCAs:  clientCAPool,
  }
  ```

### Authorization Framework
- [ ] **RBAC Implementation**
  ```go
  type Permission struct {
      Resource string
      Action   string
  }
  
  type Role struct {
      Name        string
      Permissions []Permission
  }
  
  func (a *Authorizer) Authorize(user User, resource string, action string) bool {
      for _, role := range user.Roles {
          for _, perm := range role.Permissions {
              if perm.Resource == resource && perm.Action == action {
                  return true
              }
          }
      }
      return false
  }
  ```

- [ ] **Policy Engine Integration**
  ```rego
  # OPA Policy
  package gunj.authz
  
  default allow = false
  
  allow {
      input.method == "GET"
      input.path == "/api/v1/platforms"
      input.user.roles[_] == "viewer"
  }
  
  allow {
      input.method == "POST"
      input.path == "/api/v1/platforms"
      input.user.roles[_] == "admin"
  }
  ```

---

## Supply Chain Security

### Dependency Management
- [ ] **Dependency Scanning**
  ```yaml
  - name: Run Nancy dependency scan
    run: |
      go list -json -deps | nancy sleuth
  ```

- [ ] **License Compliance**
  ```yaml
  - name: Check licenses
    run: |
      go-licenses check ./... --allowed_licenses=MIT,Apache-2.0,BSD-3-Clause
  ```

- [ ] **Dependency Updates**
  ```yaml
  - name: Update dependencies
    uses: renovatebot/github-action@v39.0.0
    with:
      configurationFile: .github/renovate.json
  ```

### Build Pipeline Security
- [ ] **Signed Commits**
  ```yaml
  - name: Verify commit signature
    run: |
      git verify-commit HEAD
  ```

- [ ] **Protected Branches**
  ```yaml
  branch_protection:
    required_status_checks:
      strict: true
      contexts:
        - continuous-integration/travis-ci
    enforce_admins: true
    required_pull_request_reviews:
      required_approving_review_count: 2
    restrictions:
      users: []
      teams: ["maintainers"]
  ```

- [ ] **Artifact Provenance**
  ```yaml
  - name: Generate provenance
    uses: slsa-framework/slsa-github-generator@v1.9.0
    with:
      subject-name: ${{ env.IMAGE }}
      subject-digest: ${{ steps.image.outputs.digest }}
      push-to-registry: true
  ```

---

## Compliance Automation

### Security Scanning Pipeline
```yaml
name: Security Scan

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 0 * * *'

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    # Static Analysis
    - name: Run Gosec Security Scanner
      uses: securego/gosec@master
      with:
        args: '-fmt sarif -out gosec-results.sarif ./...'
    
    # Dependency Scanning
    - name: Run Nancy
      run: |
        go list -json -deps | nancy sleuth
    
    # Container Scanning
    - name: Run Trivy
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'sarif'
        output: 'trivy-results.sarif'
    
    # Secret Scanning
    - name: Run Gitleaks
      uses: gitleaks/gitleaks-action@v2
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    
    # License Scanning
    - name: Check Licenses
      run: |
        go-licenses check ./...
    
    # SAST
    - name: Run Semgrep
      uses: returntocorp/semgrep-action@v1
      with:
        config: >-
          p/security-audit
          p/owasp-top-ten
          p/golang
    
    # Upload results
    - name: Upload SARIF files
      uses: github/codeql-action/upload-sarif@v3
      with:
        sarif_file: '.'
```

### Compliance Reporting
```go
package compliance

import (
    "encoding/json"
    "time"
)

type ComplianceReport struct {
    Timestamp   time.Time              `json:"timestamp"`
    Version     string                 `json:"version"`
    Standards   map[string]Standard    `json:"standards"`
    Summary     Summary                `json:"summary"`
    Findings    []Finding              `json:"findings"`
}

type Standard struct {
    Name        string                 `json:"name"`
    Version     string                 `json:"version"`
    Controls    []Control              `json:"controls"`
    Compliance  float64                `json:"compliance_percentage"`
}

type Control struct {
    ID          string                 `json:"id"`
    Description string                 `json:"description"`
    Status      string                 `json:"status"` // passed, failed, not_applicable
    Evidence    []Evidence             `json:"evidence"`
    Remediation string                 `json:"remediation,omitempty"`
}

type Finding struct {
    Severity    string                 `json:"severity"`
    Standard    string                 `json:"standard"`
    ControlID   string                 `json:"control_id"`
    Description string                 `json:"description"`
    Resource    string                 `json:"resource"`
    Remediation string                 `json:"remediation"`
}

func GenerateComplianceReport() (*ComplianceReport, error) {
    report := &ComplianceReport{
        Timestamp: time.Now(),
        Version:   "2.0.0",
        Standards: make(map[string]Standard),
    }
    
    // Run CIS Kubernetes Benchmark checks
    cisResults := runCISBenchmark()
    report.Standards["CIS-Kubernetes"] = cisResults
    
    // Run NIST checks
    nistResults := runNISTChecks()
    report.Standards["NIST-CSF"] = nistResults
    
    // Run OWASP checks
    owaspResults := runOWASPChecks()
    report.Standards["OWASP-API"] = owaspResults
    
    // Calculate summary
    report.Summary = calculateSummary(report.Standards)
    
    // Identify findings
    report.Findings = identifyFindings(report.Standards)
    
    return report, nil
}
```

---

## Security Monitoring

### Metrics and Alerts
```yaml
groups:
  - name: security_alerts
    interval: 1m
    rules:
      # Authentication failures
      - alert: HighAuthenticationFailureRate
        expr: |
          rate(authentication_failures_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High authentication failure rate detected"
          description: "{{ $value }} failures per second"
      
      # Unauthorized access attempts
      - alert: UnauthorizedAccessAttempts
        expr: |
          rate(authorization_denied_total[5m]) > 5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Multiple unauthorized access attempts"
      
      # Certificate expiry
      - alert: CertificateExpiringSoon
        expr: |
          (cert_expiry_timestamp - time()) / 86400 < 30
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Certificate expiring in {{ $value }} days"
      
      # Vulnerability detection
      - alert: CriticalVulnerabilityDetected
        expr: |
          vulnerability_score{severity="critical"} > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Critical vulnerability detected"
```

### Security Dashboard
```json
{
  "dashboard": {
    "title": "Security Compliance Dashboard",
    "panels": [
      {
        "title": "Compliance Score",
        "type": "gauge",
        "targets": [{
          "expr": "compliance_score_percentage"
        }]
      },
      {
        "title": "Vulnerabilities by Severity",
        "type": "piechart",
        "targets": [{
          "expr": "sum by (severity) (vulnerabilities_total)"
        }]
      },
      {
        "title": "Authentication Metrics",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(authentication_success_total[5m])",
            "legendFormat": "Success"
          },
          {
            "expr": "rate(authentication_failures_total[5m])",
            "legendFormat": "Failures"
          }
        ]
      },
      {
        "title": "Security Events",
        "type": "table",
        "targets": [{
          "expr": "security_events_total"
        }]
      }
    ]
  }
}
```

---

## Incident Response

### Incident Response Plan
1. **Detection and Analysis**
   - Alert received
   - Initial triage
   - Severity assessment
   - Impact analysis

2. **Containment**
   - Isolate affected systems
   - Preserve evidence
   - Prevent spread

3. **Eradication**
   - Remove threat
   - Patch vulnerabilities
   - Update configurations

4. **Recovery**
   - Restore services
   - Verify functionality
   - Monitor for recurrence

5. **Post-Incident**
   - Document lessons learned
   - Update procedures
   - Improve controls

### Incident Response Automation
```go
package incident

type IncidentHandler struct {
    alertManager AlertManager
    pagerDuty    PagerDuty
    slack        SlackClient
}

func (h *IncidentHandler) HandleSecurityIncident(incident SecurityIncident) error {
    // 1. Log incident
    log.WithFields(log.Fields{
        "severity": incident.Severity,
        "type":     incident.Type,
        "source":   incident.Source,
    }).Error("Security incident detected")
    
    // 2. Create incident ticket
    ticket := h.createIncidentTicket(incident)
    
    // 3. Notify on-call
    if incident.Severity == "critical" {
        h.pagerDuty.TriggerIncident(ticket)
    }
    
    // 4. Post to Slack
    h.slack.PostToChannel("#security-incidents", formatIncident(incident))
    
    // 5. Execute automated response
    return h.executeResponse(incident)
}
```

---

## Compliance Evidence

### Evidence Collection
```bash
#!/bin/bash
# collect-compliance-evidence.sh

EVIDENCE_DIR="compliance-evidence-$(date +%Y%m%d)"
mkdir -p $EVIDENCE_DIR

# Collect configuration files
cp -r config/ $EVIDENCE_DIR/configs/

# Run security scans
trivy fs . > $EVIDENCE_DIR/trivy-scan.txt
gosec ./... > $EVIDENCE_DIR/gosec-scan.txt
nancy sleuth > $EVIDENCE_DIR/dependency-scan.txt

# Collect RBAC policies
kubectl get clusterroles,clusterrolebindings,roles,rolebindings --all-namespaces -o yaml > $EVIDENCE_DIR/rbac-policies.yaml

# Collect network policies
kubectl get networkpolicies --all-namespaces -o yaml > $EVIDENCE_DIR/network-policies.yaml

# Generate compliance report
./generate-compliance-report.sh > $EVIDENCE_DIR/compliance-report.json

# Create archive
tar -czf compliance-evidence-$(date +%Y%m%d).tar.gz $EVIDENCE_DIR/
```

---

## Appendix: Security Tools

### Required Security Tools
1. **Container Scanning**: Trivy, Grype, Clair
2. **SAST**: Gosec, Semgrep, SonarQube
3. **Dependency Scanning**: Nancy, Snyk, OWASP Dependency Check
4. **Secret Scanning**: Gitleaks, TruffleHog
5. **License Scanning**: go-licenses, FOSSA
6. **Runtime Security**: Falco, Sysdig
7. **Policy Engine**: Open Policy Agent (OPA)
8. **Certificate Management**: cert-manager
9. **Secret Management**: Vault, Sealed Secrets

### Integration Examples
```yaml
# ArgoCD Application with Security Policies
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gunj-operator
spec:
  source:
    helm:
      parameters:
      - name: security.podSecurityPolicy.enabled
        value: "true"
      - name: security.networkPolicy.enabled
        value: "true"
      - name: security.rbac.create
        value: "true"
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - Validate=true
    - CreateNamespace=false
```

---

**This security compliance checklist must be reviewed quarterly and updated to reflect new threats and standards.**
