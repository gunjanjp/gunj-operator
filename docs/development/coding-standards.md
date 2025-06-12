# Coding Standards

This document defines the coding standards for the Gunj Operator project. All contributors must follow these standards to ensure code quality, maintainability, and consistency.

## Table of Contents

1. [General Principles](#general-principles)
2. [Go Standards](#go-standards)
3. [TypeScript/React Standards](#typescriptreact-standards)
4. [Kubernetes YAML Standards](#kubernetes-yaml-standards)
5. [API Design Standards](#api-design-standards)
6. [Testing Standards](#testing-standards)
7. [Documentation Standards](#documentation-standards)
8. [Security Standards](#security-standards)
9. [Performance Standards](#performance-standards)
10. [Tools and Enforcement](#tools-and-enforcement)

## General Principles

### Core Values

1. **Clarity over Cleverness**: Write code that is easy to understand
2. **Consistency**: Follow established patterns throughout the codebase
3. **Simplicity**: Choose simple solutions over complex ones
4. **Testability**: Write code that is easy to test
5. **Maintainability**: Consider future developers (including yourself)

### Universal Rules

- Use meaningful variable and function names
- Keep functions small and focused (single responsibility)
- Handle errors explicitly and gracefully
- Add comments for complex logic
- Remove dead code and TODO comments before committing
- Prefer composition over inheritance

## Go Standards

### Project Structure

```
package-name/
├── doc.go           # Package documentation
├── types.go         # Type definitions
├── interfaces.go    # Interface definitions
├── implementation.go # Main implementation
├── helpers.go       # Helper functions
└── tests/          # Test files
    ├── unit/
    └── integration/
```

### File Naming

- Use lowercase with underscores: `platform_controller.go`
- Test files: `platform_controller_test.go`
- Integration tests: `platform_integration_test.go`
- Build tags for special files: `platform_linux.go`

### Package Guidelines

```go
// Package controllers implements Kubernetes controllers for the Gunj operator.
// It provides reconciliation logic for ObservabilityPlatform resources.
package controllers

// Imports should be organized in groups
import (
    // Standard library
    "context"
    "fmt"
    "time"
    
    // External packages (alphabetically)
    "github.com/go-logr/logr"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    
    // Internal packages
    "github.com/gunjanjp/gunj-operator/api/v1beta1"
    "github.com/gunjanjp/gunj-operator/internal/utils"
)
```

### Naming Conventions

```go
// Constants - Use CamelCase for exported, UPPER_SNAKE_CASE discouraged
const DefaultTimeout = 30 * time.Second
const maxRetries = 3

// Variables - Use camelCase
var (
    errInvalidConfig = errors.New("invalid configuration")
    DefaultLogger    = logr.Discard()
)

// Types - Use CamelCase
type PlatformReconciler struct {
    Client client.Client
    Log    logr.Logger
}

// Interfaces - Use -er suffix where appropriate
type Reconciler interface {
    Reconcile(ctx context.Context, req Request) (Result, error)
}

// Functions - Use CamelCase for exported, camelCase for unexported
func NewReconciler(client client.Client) *Reconciler {
    return &Reconciler{Client: client}
}

func validateConfig(config *Config) error {
    // validation logic
}
```

### Error Handling

```go
// Always wrap errors with context
func deployPrometheus(ctx context.Context, platform *v1beta1.Platform) error {
    deployment := buildDeployment(platform)
    if err := r.Create(ctx, deployment); err != nil {
        return fmt.Errorf("creating prometheus deployment: %w", err)
    }
    return nil
}

// Define sentinel errors
var (
    ErrNotFound      = errors.New("resource not found")
    ErrInvalidConfig = errors.New("invalid configuration")
)

// Check errors appropriately
if err := r.Client.Get(ctx, key, platform); err != nil {
    if errors.IsNotFound(err) {
        // Handle not found case
        return ctrl.Result{}, nil
    }
    // Handle other errors
    return ctrl.Result{}, fmt.Errorf("getting platform: %w", err)
}
```

### Logging

```go
// Use structured logging with logr
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := r.Log.WithValues("platform", req.NamespacedName)
    
    // Info level for important operations
    log.Info("Starting reconciliation")
    
    // V(1) for detailed operational info
    log.V(1).Info("Checking platform status", "phase", platform.Status.Phase)
    
    // Error logging with context
    if err := r.deployComponent(ctx, platform); err != nil {
        log.Error(err, "Failed to deploy component", 
            "component", "prometheus",
            "namespace", platform.Namespace)
        return ctrl.Result{}, err
    }
    
    return ctrl.Result{}, nil
}
```

### Context Usage

```go
// Always accept context as first parameter
func (r *Reconciler) deployPlatform(ctx context.Context, platform *v1beta1.Platform) error {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Pass context through
    if err := r.deployPrometheus(ctx, platform); err != nil {
        return err
    }
    
    return nil
}
```

### Testing

```go
// Table-driven tests
func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        wantErr bool
    }{
        {
            name: "valid config",
            config: &Config{
                Name:    "test",
                Version: "v1.0.0",
            },
            wantErr: false,
        },
        {
            name: "missing name",
            config: &Config{
                Version: "v1.0.0",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateConfig(tt.config)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

// Use test helpers
func TestReconciler(t *testing.T) {
    g := NewWithT(t)
    
    // Setup
    platform := newTestPlatform()
    client := fake.NewClientBuilder().
        WithScheme(scheme).
        WithObjects(platform).
        Build()
    
    // Test
    reconciler := &Reconciler{Client: client}
    result, err := reconciler.Reconcile(ctx, request)
    
    // Assert
    g.Expect(err).NotTo(HaveOccurred())
    g.Expect(result.Requeue).To(BeFalse())
}
```

## TypeScript/React Standards

### Project Structure

```
src/
├── components/          # React components
│   ├── common/         # Shared components
│   ├── features/       # Feature-specific components
│   └── layouts/        # Layout components
├── hooks/              # Custom hooks
├── store/              # State management
├── services/           # API services
├── utils/              # Utility functions
├── types/              # TypeScript type definitions
└── __tests__/          # Test files
```

### TypeScript Guidelines

```typescript
// Always use explicit types
interface Platform {
  id: string;
  name: string;
  namespace: string;
  status: PlatformStatus;
  createdAt: Date;
  updatedAt: Date;
}

// Use enums for constants
enum PlatformStatus {
  Pending = 'Pending',
  Installing = 'Installing',
  Ready = 'Ready',
  Failed = 'Failed',
}

// Use type for unions and intersections
type ID = string | number;
type PlatformWithMetadata = Platform & {
  metadata: Record<string, unknown>;
};

// Avoid any - use unknown if type is truly unknown
function processData(data: unknown): void {
  if (typeof data === 'string') {
    // Process string
  }
}

// Use generics appropriately
function createResource<T extends Resource>(resource: T): Promise<T> {
  return api.post('/resources', resource);
}
```

### React Component Standards

```typescript
// Use functional components with TypeScript
import React, { useState, useCallback, useMemo, memo } from 'react';
import { Box, Button, CircularProgress } from '@mui/material';
import { usePlatform } from '@/hooks/usePlatform';
import type { Platform } from '@/types';

// Define props interface
interface PlatformCardProps {
  platform: Platform;
  onEdit?: (platform: Platform) => void;
  onDelete?: (id: string) => Promise<void>;
  className?: string;
  disabled?: boolean;
}

// Export named components with memo for performance
export const PlatformCard = memo<PlatformCardProps>(({
  platform,
  onEdit,
  onDelete,
  className,
  disabled = false,
}) => {
  // State hooks at the top
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Custom hooks
  const { refreshPlatform } = usePlatform(platform.id);
  
  // Memoized values
  const statusColor = useMemo(() => {
    switch (platform.status) {
      case PlatformStatus.Ready:
        return 'success';
      case PlatformStatus.Failed:
        return 'error';
      default:
        return 'warning';
    }
  }, [platform.status]);
  
  // Callbacks with useCallback
  const handleEdit = useCallback(() => {
    onEdit?.(platform);
  }, [platform, onEdit]);
  
  const handleDelete = useCallback(async () => {
    if (!onDelete || !window.confirm(`Delete ${platform.name}?`)) {
      return;
    }
    
    setIsDeleting(true);
    setError(null);
    
    try {
      await onDelete(platform.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed');
    } finally {
      setIsDeleting(false);
    }
  }, [platform, onDelete]);
  
  // Early returns for edge cases
  if (!platform) {
    return null;
  }
  
  // Main render
  return (
    <Box className={className}>
      {/* Component JSX */}
      {error && <Alert severity="error">{error}</Alert>}
    </Box>
  );
});

// Display name for debugging
PlatformCard.displayName = 'PlatformCard';

// Default props if needed
PlatformCard.defaultProps = {
  disabled: false,
};
```

### State Management (Zustand)

```typescript
import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';
import type { Platform } from '@/types';

interface PlatformState {
  // State
  platforms: Platform[];
  selectedId: string | null;
  isLoading: boolean;
  error: string | null;
  
  // Actions
  setPlatforms: (platforms: Platform[]) => void;
  selectPlatform: (id: string | null) => void;
  updatePlatform: (id: string, updates: Partial<Platform>) => void;
  deletePlatform: (id: string) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  
  // Selectors
  getSelectedPlatform: () => Platform | undefined;
}

export const usePlatformStore = create<PlatformState>()(
  devtools(
    persist(
      immer((set, get) => ({
        // Initial state
        platforms: [],
        selectedId: null,
        isLoading: false,
        error: null,
        
        // Actions using immer for immutability
        setPlatforms: (platforms) =>
          set((state) => {
            state.platforms = platforms;
          }),
          
        selectPlatform: (id) =>
          set((state) => {
            state.selectedId = id;
          }),
          
        updatePlatform: (id, updates) =>
          set((state) => {
            const index = state.platforms.findIndex((p) => p.id === id);
            if (index !== -1) {
              Object.assign(state.platforms[index], updates);
            }
          }),
          
        deletePlatform: (id) =>
          set((state) => {
            state.platforms = state.platforms.filter((p) => p.id !== id);
            if (state.selectedId === id) {
              state.selectedId = null;
            }
          }),
          
        setLoading: (loading) =>
          set((state) => {
            state.isLoading = loading;
          }),
          
        setError: (error) =>
          set((state) => {
            state.error = error;
          }),
          
        // Selectors
        getSelectedPlatform: () => {
          const { platforms, selectedId } = get();
          return platforms.find((p) => p.id === selectedId);
        },
      })),
      {
        name: 'platform-store',
        partialize: (state) => ({
          selectedId: state.selectedId,
        }),
      }
    ),
    {
      name: 'PlatformStore',
    }
  )
);
```

### Custom Hooks

```typescript
// hooks/usePlatform.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { platformAPI } from '@/services/api';
import { usePlatformStore } from '@/store/platform';
import type { Platform } from '@/types';

export function usePlatform(platformId?: string) {
  const queryClient = useQueryClient();
  const { setPlatforms, setError } = usePlatformStore();
  
  // Query for single platform
  const platformQuery = useQuery({
    queryKey: ['platform', platformId],
    queryFn: () => platformAPI.getById(platformId!),
    enabled: !!platformId,
    staleTime: 30000, // 30 seconds
  });
  
  // Query for all platforms
  const platformsQuery = useQuery({
    queryKey: ['platforms'],
    queryFn: platformAPI.getAll,
    onSuccess: (data) => {
      setPlatforms(data);
    },
    onError: (error) => {
      setError(error instanceof Error ? error.message : 'Failed to fetch platforms');
    },
  });
  
  // Mutations
  const createMutation = useMutation({
    mutationFn: platformAPI.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['platforms'] });
    },
  });
  
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Platform> }) =>
      platformAPI.update(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['platform', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['platforms'] });
    },
  });
  
  // Helper functions
  const refreshPlatform = useCallback(() => {
    if (platformId) {
      queryClient.invalidateQueries({ queryKey: ['platform', platformId] });
    }
  }, [platformId, queryClient]);
  
  return {
    platform: platformQuery.data,
    platforms: platformsQuery.data ?? [],
    isLoading: platformQuery.isLoading || platformsQuery.isLoading,
    error: platformQuery.error || platformsQuery.error,
    createPlatform: createMutation.mutate,
    updatePlatform: updateMutation.mutate,
    refreshPlatform,
  };
}
```

## Kubernetes YAML Standards

### General YAML Guidelines

```yaml
# Always include API version and kind
apiVersion: apps/v1
kind: Deployment

# Metadata should be descriptive
metadata:
  name: prometheus-server  # Use kebab-case
  namespace: monitoring
  labels:
    app.kubernetes.io/name: prometheus
    app.kubernetes.io/instance: server
    app.kubernetes.io/component: metrics
    app.kubernetes.io/part-of: observability-platform
    app.kubernetes.io/managed-by: gunj-operator
    app.kubernetes.io/version: "2.48.0"
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"

# Spec should be well-organized
spec:
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/name: prometheus
      app.kubernetes.io/instance: server
      
  template:
    metadata:
      labels:
        app.kubernetes.io/name: prometheus
        app.kubernetes.io/instance: server
        
    spec:
      # Security context for pod
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
        
      # Service account
      serviceAccountName: prometheus
      
      # Containers
      containers:
      - name: prometheus
        image: prom/prometheus:v2.48.0
        imagePullPolicy: IfNotPresent
        
        # Command and args
        args:
        - --config.file=/etc/prometheus/prometheus.yml
        - --storage.tsdb.path=/prometheus
        - --web.console.libraries=/usr/share/prometheus/console_libraries
        - --web.console.templates=/usr/share/prometheus/consoles
        
        # Ports
        ports:
        - name: http
          containerPort: 9090
          protocol: TCP
          
        # Probes
        livenessProbe:
          httpGet:
            path: /-/healthy
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
          
        readinessProbe:
          httpGet:
            path: /-/ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          
        # Resources
        resources:
          requests:
            memory: "2Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "1000m"
            
        # Volume mounts
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
        - name: storage
          mountPath: /prometheus
          
        # Security context for container
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
            
      # Volumes
      volumes:
      - name: config
        configMap:
          name: prometheus-config
      - name: storage
        persistentVolumeClaim:
          claimName: prometheus-storage
```

### CRD Standards

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production-platform
  namespace: monitoring
  # Finalizers for cleanup
  finalizers:
  - observability.io/platform-cleanup
  
spec:
  # Use clear structure
  components:
    prometheus:
      enabled: true
      version: v2.48.0  # Always specify versions
      
      # Resources with clear units
      resources:
        requests:
          memory: "4Gi"  # Not 4G or 4096Mi
          cpu: "1"       # Not 1000m when possible
        limits:
          memory: "8Gi"
          cpu: "2"
          
      # Storage configuration
      storage:
        size: 100Gi
        storageClassName: fast-ssd
        
      # Clear time units
      retention: 30d  # Not 720h
      
      # High availability
      replicas: 3
      
  # Global settings
  global:
    # External labels for federation
    externalLabels:
      cluster: production
      region: us-east-1
      environment: prod
      
    # Resource defaults
    defaultResources:
      requests:
        memory: "512Mi"
        cpu: "100m"
        
  # Alerting configuration
  alerting:
    enabled: true
    contactPoints:
    - name: oncall-team
      type: webhook
      url: https://alerts.example.com/webhook
      
# Status is managed by the operator
status:
  phase: Ready
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2025-06-12T10:00:00Z"
    reason: AllComponentsReady
    message: All components are running successfully
```

## API Design Standards

### RESTful API

```go
// Use proper HTTP methods and status codes
func (h *Handler) setupRoutes(r *gin.Engine) {
    api := r.Group("/api/v1")
    {
        // Platforms
        api.GET("/platforms", h.listPlatforms)       // 200, 400
        api.POST("/platforms", h.createPlatform)     // 201, 400, 409
        api.GET("/platforms/:id", h.getPlatform)     // 200, 404
        api.PUT("/platforms/:id", h.updatePlatform)  // 200, 400, 404
        api.DELETE("/platforms/:id", h.deletePlatform) // 204, 404
        
        // Sub-resources
        api.GET("/platforms/:id/components", h.listComponents)
        api.PUT("/platforms/:id/components/:component", h.updateComponent)
        
        // Actions
        api.POST("/platforms/:id/actions/backup", h.backupPlatform)
        api.POST("/platforms/:id/actions/restore", h.restorePlatform)
    }
}

// Consistent response format
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
    Meta    *APIMeta    `json:"meta,omitempty"`
}

type APIError struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}

// Pagination
type PaginationParams struct {
    Page     int    `form:"page" binding:"min=1"`
    PageSize int    `form:"pageSize" binding:"min=1,max=100"`
    Sort     string `form:"sort" binding:"omitempty,oneof=name created updated"`
    Order    string `form:"order" binding:"omitempty,oneof=asc desc"`
}
```

### GraphQL API

```graphql
# Use clear type definitions
type Platform {
  id: ID!
  name: String!
  namespace: String!
  status: PlatformStatus!
  components: [Component!]!
  createdAt: DateTime!
  updatedAt: DateTime!
}

# Enums for fixed values
enum PlatformStatus {
  PENDING
  INSTALLING
  READY
  FAILED
  UPGRADING
}

# Input types for mutations
input CreatePlatformInput {
  name: String!
  namespace: String!
  components: ComponentsInput!
}

# Clear query structure
type Query {
  # Single resource
  platform(id: ID!): Platform
  
  # List with filtering
  platforms(
    namespace: String
    status: PlatformStatus
    page: Int = 1
    pageSize: Int = 20
  ): PlatformConnection!
  
  # Nested queries
  platformMetrics(id: ID!, range: TimeRange!): Metrics!
}

# Mutations follow naming conventions
type Mutation {
  createPlatform(input: CreatePlatformInput!): Platform!
  updatePlatform(id: ID!, input: UpdatePlatformInput!): Platform!
  deletePlatform(id: ID!): Boolean!
  
  # Actions as mutations
  backupPlatform(id: ID!, destination: String!): BackupResult!
  restorePlatform(id: ID!, backupId: ID!): Platform!
}

# Subscriptions for real-time updates
type Subscription {
  platformUpdated(id: ID!): Platform!
  platformStatusChanged(namespace: String): PlatformStatusEvent!
}
```

## Testing Standards

### Test Organization

```
tests/
├── unit/           # Unit tests
├── integration/    # Integration tests
├── e2e/           # End-to-end tests
├── performance/   # Performance tests
├── fixtures/      # Test data
└── helpers/       # Test utilities
```

### Unit Testing

```go
// Name tests clearly
func TestPlatformController_Reconcile(t *testing.T) {
    // Use table-driven tests
    tests := []struct {
        name      string
        platform  *v1beta1.ObservabilityPlatform
        objects   []runtime.Object
        want      ctrl.Result
        wantErr   bool
        wantPhase string
    }{
        {
            name: "new platform starts installation",
            platform: newTestPlatform("test", "default"),
            objects:  []runtime.Object{},
            want:     ctrl.Result{RequeueAfter: 30 * time.Second},
            wantErr:  false,
            wantPhase: "Installing",
        },
        {
            name: "existing platform checks status",
            platform: newTestPlatform("test", "default"),
            objects: []runtime.Object{
                newTestDeployment("prometheus", "default"),
            },
            want:      ctrl.Result{RequeueAfter: 60 * time.Second},
            wantErr:   false,
            wantPhase: "Ready",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            ctx := context.Background()
            client := fake.NewClientBuilder().
                WithScheme(scheme).
                WithObjects(append(tt.objects, tt.platform)...).
                WithStatusSubresource(tt.platform).
                Build()
                
            reconciler := &PlatformReconciler{
                Client: client,
                Log:    ctrl.Log.WithName("test"),
                Scheme: scheme,
            }
            
            // Execute
            got, err := reconciler.Reconcile(ctx, ctrl.Request{
                NamespacedName: types.NamespacedName{
                    Name:      tt.platform.Name,
                    Namespace: tt.platform.Namespace,
                },
            })
            
            // Assert
            if (err != nil) != tt.wantErr {
                t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Reconcile() = %v, want %v", got, tt.want)
            }
            
            // Check status
            updated := &v1beta1.ObservabilityPlatform{}
            if err := client.Get(ctx, types.NamespacedName{
                Name:      tt.platform.Name,
                Namespace: tt.platform.Namespace,
            }, updated); err != nil {
                t.Fatalf("Failed to get updated platform: %v", err)
            }
            
            if updated.Status.Phase != tt.wantPhase {
                t.Errorf("Status.Phase = %v, want %v", updated.Status.Phase, tt.wantPhase)
            }
        })
    }
}

// Test helpers
func newTestPlatform(name, namespace string) *v1beta1.ObservabilityPlatform {
    return &v1beta1.ObservabilityPlatform{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: namespace,
        },
        Spec: v1beta1.ObservabilityPlatformSpec{
            Components: v1beta1.Components{
                Prometheus: &v1beta1.PrometheusSpec{
                    Enabled: true,
                    Version: "v2.48.0",
                },
            },
        },
    }
}
```

### React Testing

```typescript
// Component tests
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PlatformCard } from '@/components/PlatformCard';
import { mockPlatform } from '@/tests/fixtures';

// Test wrapper with providers
const TestWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  
  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
};

describe('PlatformCard', () => {
  const mockOnEdit = jest.fn();
  const mockOnDelete = jest.fn();
  
  beforeEach(() => {
    jest.clearAllMocks();
  });
  
  it('renders platform information correctly', () => {
    render(
      <TestWrapper>
        <PlatformCard
          platform={mockPlatform}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      </TestWrapper>
    );
    
    expect(screen.getByText(mockPlatform.name)).toBeInTheDocument();
    expect(screen.getByText(mockPlatform.status)).toBeInTheDocument();
  });
  
  it('calls onEdit when edit button is clicked', async () => {
    const user = userEvent.setup();
    
    render(
      <TestWrapper>
        <PlatformCard
          platform={mockPlatform}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      </TestWrapper>
    );
    
    const editButton = screen.getByRole('button', { name: /edit/i });
    await user.click(editButton);
    
    expect(mockOnEdit).toHaveBeenCalledWith(mockPlatform);
  });
  
  it('confirms before deletion', async () => {
    const user = userEvent.setup();
    window.confirm = jest.fn(() => true);
    
    render(
      <TestWrapper>
        <PlatformCard
          platform={mockPlatform}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      </TestWrapper>
    );
    
    const deleteButton = screen.getByRole('button', { name: /delete/i });
    await user.click(deleteButton);
    
    expect(window.confirm).toHaveBeenCalledWith(`Delete ${mockPlatform.name}?`);
    expect(mockOnDelete).toHaveBeenCalledWith(mockPlatform.id);
  });
  
  it('shows loading state during deletion', async () => {
    const user = userEvent.setup();
    window.confirm = jest.fn(() => true);
    
    // Mock delete to be slow
    mockOnDelete.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));
    
    render(
      <TestWrapper>
        <PlatformCard
          platform={mockPlatform}
          onEdit={mockOnEdit}
          onDelete={mockOnDelete}
        />
      </TestWrapper>
    );
    
    const deleteButton = screen.getByRole('button', { name: /delete/i });
    await user.click(deleteButton);
    
    // Should show loading
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
    
    // Wait for completion
    await waitFor(() => {
      expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
    });
  });
});
```

## Documentation Standards

### Code Comments

```go
// Package platform provides controllers for managing observability platforms.
// It implements the Kubernetes controller pattern to reconcile ObservabilityPlatform
// custom resources with their desired state.
package platform

// PlatformReconciler reconciles ObservabilityPlatform objects.
// It manages the lifecycle of Prometheus, Grafana, Loki, and Tempo components.
type PlatformReconciler struct {
    // Client is the Kubernetes client for API operations
    client.Client
    
    // Log is the logger instance for this reconciler
    Log logr.Logger
    
    // Scheme is the scheme for this reconciler
    Scheme *runtime.Scheme
}

// Reconcile is the main reconciliation loop for ObservabilityPlatform resources.
// It ensures that the actual state of the platform matches the desired state
// specified in the custom resource.
//
// The reconciliation process:
// 1. Fetches the ObservabilityPlatform resource
// 2. Handles deletion if marked for deletion
// 3. Ensures all components are deployed and configured
// 4. Updates the platform status
// 5. Requeues for periodic reconciliation
func (r *PlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Implementation
}

// TODO(username): Implement backup functionality - Issue #123
// NOTE: This uses eventual consistency for status updates
// DEPRECATED: Use ReconcileV2 instead (will be removed in v3.0.0)
```

### API Documentation

```go
// Package api provides the REST API for the Gunj operator.
//
// The API follows RESTful principles and provides endpoints for:
//   - Platform management (CRUD operations)
//   - Component configuration
//   - Monitoring and metrics
//   - Backup and restore operations
//
// Authentication:
// All endpoints require authentication via Bearer token in the Authorization header.
// Tokens can be obtained from the /auth/login endpoint.
//
// Error Responses:
// All endpoints return consistent error responses in the format:
//   {
//     "success": false,
//     "error": {
//       "code": "ERROR_CODE",
//       "message": "Human readable message",
//       "details": {}
//     }
//   }
//
// Pagination:
// List endpoints support pagination via query parameters:
//   - page: Page number (default: 1)
//   - pageSize: Items per page (default: 20, max: 100)
//   - sort: Sort field (name, created, updated)
//   - order: Sort order (asc, desc)
package api

// CreatePlatform creates a new observability platform.
//
// @Summary Create a new platform
// @Description Create a new observability platform with the specified configuration
// @Tags platforms
// @Accept json
// @Produce json
// @Param platform body CreatePlatformRequest true "Platform configuration"
// @Success 201 {object} APIResponse{data=Platform} "Platform created successfully"
// @Failure 400 {object} APIResponse{error=APIError} "Invalid request"
// @Failure 409 {object} APIResponse{error=APIError} "Platform already exists"
// @Router /api/v1/platforms [post]
// @Security Bearer
func (h *Handler) CreatePlatform(c *gin.Context) {
    // Implementation
}
```

## Security Standards

### Input Validation

```go
// Always validate and sanitize input
type CreatePlatformRequest struct {
    Name      string `json:"name" binding:"required,min=3,max=63,alphanum"`
    Namespace string `json:"namespace" binding:"required,min=3,max=63,alphanum"`
    Components Components `json:"components" binding:"required"`
}

func (r CreatePlatformRequest) Validate() error {
    // Additional validation beyond struct tags
    if !isValidKubernetesName(r.Name) {
        return fmt.Errorf("invalid platform name: must be a valid Kubernetes name")
    }
    
    if r.Components.Prometheus != nil {
        if err := r.Components.Prometheus.Validate(); err != nil {
            return fmt.Errorf("invalid prometheus config: %w", err)
        }
    }
    
    return nil
}
```

### Authentication & Authorization

```go
// Use middleware for auth
func AuthMiddleware(authService AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, APIResponse{
                Success: false,
                Error: &APIError{
                    Code:    "UNAUTHORIZED",
                    Message: "Missing authorization header",
                },
            })
            return
        }
        
        // Validate token
        claims, err := authService.ValidateToken(strings.TrimPrefix(token, "Bearer "))
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, APIResponse{
                Success: false,
                Error: &APIError{
                    Code:    "INVALID_TOKEN",
                    Message: "Invalid or expired token",
                },
            })
            return
        }
        
        // Check permissions
        if !authService.HasPermission(claims, c.Request.Method, c.Request.URL.Path) {
            c.AbortWithStatusJSON(http.StatusForbidden, APIResponse{
                Success: false,
                Error: &APIError{
                    Code:    "FORBIDDEN",
                    Message: "Insufficient permissions",
                },
            })
            return
        }
        
        c.Set("user", claims)
        c.Next()
    }
}
```

### Secret Management

```go
// Never log secrets
func (r *Reconciler) createSecret(ctx context.Context, platform *v1beta1.Platform) error {
    secret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      platform.Name + "-credentials",
            Namespace: platform.Namespace,
        },
        Type: corev1.SecretTypeOpaque,
        Data: map[string][]byte{
            "password": []byte(generatePassword()),
        },
    }
    
    // Log without exposing secret
    r.Log.Info("Creating credentials secret", "name", secret.Name)
    // Never: r.Log.Info("Creating secret", "data", secret.Data)
    
    return r.Create(ctx, secret)
}
```

## Performance Standards

### Resource Management

```go
// Use context timeouts
func (s *Service) GetPlatformMetrics(ctx context.Context, id string) (*Metrics, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Implementation
}

// Implement caching
type PlatformCache struct {
    mu    sync.RWMutex
    cache map[string]*Platform
    ttl   time.Duration
}

func (c *PlatformCache) Get(key string) (*Platform, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    item, found := c.cache[key]
    return item, found
}

// Use pagination
func (s *Service) ListPlatforms(ctx context.Context, params PaginationParams) (*PlatformList, error) {
    // Always enforce reasonable limits
    if params.PageSize == 0 {
        params.PageSize = 20
    }
    if params.PageSize > 100 {
        params.PageSize = 100
    }
    
    // Implementation
}
```

### Database Optimization

```go
// Use connection pooling
func NewDB(config DBConfig) (*sql.DB, error) {
    db, err := sql.Open("postgres", config.DSN)
    if err != nil {
        return nil, err
    }
    
    // Configure pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    return db, nil
}

// Use prepared statements
func (r *Repository) GetPlatform(ctx context.Context, id string) (*Platform, error) {
    stmt, err := r.db.PrepareContext(ctx, `
        SELECT id, name, namespace, status, created_at, updated_at
        FROM platforms
        WHERE id = $1
    `)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()
    
    var p Platform
    err = stmt.QueryRowContext(ctx, id).Scan(
        &p.ID, &p.Name, &p.Namespace, &p.Status, &p.CreatedAt, &p.UpdatedAt,
    )
    
    return &p, err
}
```

## Tools and Enforcement

### Linting Configuration

`.golangci.yml`:
```yaml
linters:
  enable:
  - gofmt
  - goimports
  - golint
  - govet
  - ineffassign
  - misspell
  - unconvert
  - goconst
  - gocyclo
  - gosec
  - megacheck
  - structcheck
  - varcheck
  - typecheck
  - gosimple
  - staticcheck

linters-settings:
  gocyclo:
    min-complexity: 15
  goconst:
    min-len: 3
    min-occurrences: 3

issues:
  exclude-rules:
  - path: _test\.go
    linters:
    - gocyclo
    - goconst
```

### Pre-commit Hooks

`.pre-commit-config.yaml`:
```yaml
repos:
- repo: https://github.com/pre-commit/pre-commit-hooks
  rev: v4.4.0
  hooks:
  - id: trailing-whitespace
  - id: end-of-file-fixer
  - id: check-yaml
  - id: check-added-large-files

- repo: https://github.com/dnephin/pre-commit-golang
  rev: v0.5.1
  hooks:
  - id: go-fmt
  - id: go-imports
  - id: go-vet
  - id: go-unit-tests
  - id: go-mod-tidy

- repo: https://github.com/pre-commit/mirrors-prettier
  rev: v3.0.0
  hooks:
  - id: prettier
    types_or: [typescript, tsx, javascript, jsx, json, yaml]
```

### Editor Configuration

`.editorconfig`:
```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[*.{ts,tsx,js,jsx,json}]
indent_style = space
indent_size = 2

[*.{yaml,yml}]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab
```

---

By following these coding standards, we ensure that the Gunj Operator codebase remains clean, consistent, and maintainable. All contributors are expected to adhere to these standards.
