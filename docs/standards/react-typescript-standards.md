# React/TypeScript Coding Standards v1.0

**Project**: Gunj Operator - Enterprise Observability Platform  
**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Official Frontend Standards  

---

## 📋 Table of Contents

1. [General Principles](#general-principles)
2. [TypeScript Standards](#typescript-standards)
3. [React Standards](#react-standards)
4. [Component Architecture](#component-architecture)
5. [State Management](#state-management)
6. [Styling Standards](#styling-standards)
7. [Testing Standards](#testing-standards)
8. [Performance Standards](#performance-standards)
9. [Security Standards](#security-standards)
10. [Accessibility Standards](#accessibility-standards)

---

## 🎯 General Principles

### Core Values

1. **Type Safety First**: Leverage TypeScript to catch errors at compile time
2. **Component Reusability**: Build once, use everywhere
3. **Performance by Default**: Every component should be optimized
4. **Accessibility Always**: WCAG 2.1 AA compliance minimum
5. **Developer Experience**: Code should be self-documenting and intuitive

### File Organization

```
ui/
├── src/
│   ├── components/        # Reusable components
│   │   ├── common/       # Basic UI components
│   │   ├── platform/     # Platform-specific components
│   │   └── layout/       # Layout components
│   ├── pages/            # Page-level components
│   ├── hooks/            # Custom React hooks
│   ├── store/            # State management (Zustand)
│   ├── api/              # API client and types
│   ├── utils/            # Utility functions
│   ├── types/            # TypeScript type definitions
│   ├── constants/        # Application constants
│   ├── styles/           # Global styles and themes
│   └── tests/            # Test utilities and mocks
```

### Naming Conventions

```typescript
// Files and folders: kebab-case
platform-card.tsx
use-platform-data.ts
api-client.ts

// React components: PascalCase
export const PlatformCard: React.FC = () => {}

// Functions and variables: camelCase
const handleSubmit = () => {}
const platformData = usePlatformData()

// Constants: UPPER_SNAKE_CASE
const MAX_RETRY_ATTEMPTS = 3
const API_BASE_URL = '/api/v1'

// Types and interfaces: PascalCase with 'I' prefix for interfaces (optional)
type Platform = {}
interface IPlatformProps {}
interface PlatformState {} // 'I' prefix optional, be consistent

// Enums: PascalCase with singular names
enum PlatformStatus {
  Pending = 'PENDING',
  Ready = 'READY',
  Failed = 'FAILED'
}
```

---

## 📘 TypeScript Standards

### Type Definitions

```typescript
// CORRECT: Explicit types with proper documentation
/**
 * Represents an observability platform configuration
 */
export interface Platform {
  /** Unique identifier for the platform */
  id: string;
  
  /** Platform metadata following Kubernetes conventions */
  metadata: {
    name: string;
    namespace: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
    creationTimestamp: string;
    uid: string;
  };
  
  /** Platform specification */
  spec: PlatformSpec;
  
  /** Current platform status */
  status: PlatformStatus;
}

// CORRECT: Use discriminated unions for variants
export type ComponentType = 
  | { type: 'prometheus'; config: PrometheusConfig }
  | { type: 'grafana'; config: GrafanaConfig }
  | { type: 'loki'; config: LokiConfig }
  | { type: 'tempo'; config: TempoConfig };

// CORRECT: Avoid 'any' - use 'unknown' or specific types
// BAD
const processData = (data: any) => {}

// GOOD
const processData = <T extends Record<string, unknown>>(data: T) => {}
```

### Utility Types

```typescript
// Create reusable utility types
export type Nullable<T> = T | null;
export type Optional<T> = T | undefined;
export type AsyncState<T> = {
  data: Nullable<T>;
  isLoading: boolean;
  error: Nullable<Error>;
};

// Use built-in utility types effectively
type ReadonlyPlatform = Readonly<Platform>;
type PartialPlatform = Partial<Platform>;
type PlatformKeys = keyof Platform;
type PlatformWithoutStatus = Omit<Platform, 'status'>;
```

### Strict Type Checking

```typescript
// tsconfig.json requirements
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "strictFunctionTypes": true,
    "strictBindCallApply": true,
    "strictPropertyInitialization": true,
    "noImplicitThis": true,
    "alwaysStrict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedIndexedAccess": true
  }
}
```

---

## ⚛️ React Standards

### Component Structure

```typescript
// CORRECT: Functional component with proper typing
import React, { useState, useCallback, useMemo } from 'react';
import { Box, Card, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';

import { Platform } from '@/types/platform';
import { usePlatformStore } from '@/store/platform';
import { StatusIndicator } from '@/components/common';

/**
 * Props for the PlatformCard component
 */
interface PlatformCardProps {
  /** Platform data to display */
  platform: Platform;
  
  /** Callback when platform is selected */
  onSelect?: (platform: Platform) => void;
  
  /** Whether the card is in a loading state */
  loading?: boolean;
  
  /** Whether to show detailed information */
  detailed?: boolean;
  
  /** Custom CSS classes */
  className?: string;
}

/**
 * Displays a platform summary card with status and actions
 * 
 * @example
 * <PlatformCard
 *   platform={platformData}
 *   onSelect={handleSelect}
 *   detailed
 * />
 */
export const PlatformCard: React.FC<PlatformCardProps> = React.memo(({
  platform,
  onSelect,
  loading = false,
  detailed = false,
  className,
}) => {
  // Hooks must be at the top level
  const { t } = useTranslation();
  const { updatePlatform } = usePlatformStore();
  
  // Local state
  const [isExpanded, setIsExpanded] = useState(false);
  
  // Memoized values
  const platformHealth = useMemo(() => 
    calculateHealth(platform), [platform]
  );
  
  // Callbacks
  const handleClick = useCallback(() => {
    onSelect?.(platform);
  }, [platform, onSelect]);
  
  const handleToggleExpand = useCallback(() => {
    setIsExpanded(prev => !prev);
  }, []);
  
  // Early returns for edge cases
  if (loading) {
    return <PlatformCardSkeleton />;
  }
  
  if (!platform) {
    return null;
  }
  
  // Main render
  return (
    <Card 
      className={className}
      onClick={handleClick}
      sx={{ 
        cursor: onSelect ? 'pointer' : 'default',
        transition: 'all 0.3s ease',
        '&:hover': {
          boxShadow: onSelect ? 4 : 1,
        },
      }}
    >
      <CardContent>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Typography variant="h6" component="h2">
            {platform.metadata.name}
          </Typography>
          <StatusIndicator 
            status={platform.status.phase} 
            health={platformHealth}
          />
        </Box>
        
        {detailed && (
          <PlatformDetails 
            platform={platform} 
            expanded={isExpanded}
            onToggle={handleToggleExpand}
          />
        )}
      </CardContent>
    </Card>
  );
});

// Display name for debugging
PlatformCard.displayName = 'PlatformCard';
```

### Hooks Usage

```typescript
// CORRECT: Custom hook with proper typing
import { useState, useEffect, useCallback, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

import { platformAPI } from '@/api/platform';
import { Platform } from '@/types/platform';

interface UsePlatformOptions {
  namespace?: string;
  autoRefresh?: boolean;
  refreshInterval?: number;
}

interface UsePlatformReturn {
  platforms: Platform[];
  isLoading: boolean;
  error: Error | null;
  refetch: () => void;
  createPlatform: (data: CreatePlatformData) => Promise<Platform>;
  updatePlatform: (id: string, data: UpdatePlatformData) => Promise<Platform>;
  deletePlatform: (id: string) => Promise<void>;
}

/**
 * Custom hook for platform management
 * 
 * @example
 * const { platforms, isLoading, createPlatform } = usePlatform({
 *   namespace: 'monitoring',
 *   autoRefresh: true
 * });
 */
export function usePlatform(options: UsePlatformOptions = {}): UsePlatformReturn {
  const {
    namespace = 'default',
    autoRefresh = false,
    refreshInterval = 30000,
  } = options;
  
  const queryClient = useQueryClient();
  const abortControllerRef = useRef<AbortController>();
  
  // Query for fetching platforms
  const {
    data: platforms = [],
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['platforms', namespace],
    queryFn: ({ signal }) => platformAPI.list(namespace, { signal }),
    refetchInterval: autoRefresh ? refreshInterval : false,
    refetchOnWindowFocus: false,
    retry: 3,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
  
  // Mutations
  const createMutation = useMutation({
    mutationFn: (data: CreatePlatformData) => 
      platformAPI.create(namespace, data),
    onSuccess: (newPlatform) => {
      queryClient.invalidateQueries(['platforms', namespace]);
      queryClient.setQueryData(['platform', newPlatform.id], newPlatform);
    },
  });
  
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdatePlatformData }) =>
      platformAPI.update(namespace, id, data),
    onSuccess: (updatedPlatform) => {
      queryClient.invalidateQueries(['platforms', namespace]);
      queryClient.setQueryData(['platform', updatedPlatform.id], updatedPlatform);
    },
  });
  
  const deleteMutation = useMutation({
    mutationFn: (id: string) => platformAPI.delete(namespace, id),
    onSuccess: (_, deletedId) => {
      queryClient.invalidateQueries(['platforms', namespace]);
      queryClient.removeQueries(['platform', deletedId]);
    },
  });
  
  // Cleanup on unmount
  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
    };
  }, []);
  
  return {
    platforms,
    isLoading,
    error,
    refetch,
    createPlatform: createMutation.mutateAsync,
    updatePlatform: (id, data) => updateMutation.mutateAsync({ id, data }),
    deletePlatform: deleteMutation.mutateAsync,
  };
}
```

### Event Handling

```typescript
// CORRECT: Proper event handling with types
import React, { FormEvent, ChangeEvent, KeyboardEvent } from 'react';

interface FormProps {
  onSubmit: (data: FormData) => void;
}

export const PlatformForm: React.FC<FormProps> = ({ onSubmit }) => {
  const [formData, setFormData] = useState<FormData>({
    name: '',
    namespace: 'default',
  });
  
  // Type-safe event handlers
  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };
  
  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onSubmit(formData);
  };
  
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && e.ctrlKey) {
      handleSubmit(e as any);
    }
  };
  
  return (
    <form onSubmit={handleSubmit}>
      <input
        name="name"
        value={formData.name}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        required
      />
    </form>
  );
};
```

---

## 🏗️ Component Architecture

### Component Categories

```typescript
// 1. Presentational Components (Pure, no side effects)
export const Button: React.FC<ButtonProps> = ({ 
  children, 
  onClick, 
  variant = 'primary' 
}) => (
  <button className={`btn btn-${variant}`} onClick={onClick}>
    {children}
  </button>
);

// 2. Container Components (Connected to state/API)
export const PlatformListContainer: React.FC = () => {
  const { platforms, isLoading } = usePlatform();
  
  if (isLoading) return <Spinner />;
  
  return <PlatformList platforms={platforms} />;
};

// 3. Layout Components
export const DashboardLayout: React.FC<PropsWithChildren> = ({ children }) => (
  <div className="dashboard-layout">
    <Header />
    <Sidebar />
    <main>{children}</main>
  </div>
);

// 4. Page Components
export const PlatformsPage: React.FC = () => {
  useDocumentTitle('Platforms | Gunj Operator');
  
  return (
    <DashboardLayout>
      <PlatformListContainer />
    </DashboardLayout>
  );
};
```

### Component Composition

```typescript
// CORRECT: Composition over inheritance
interface CardProps {
  title: string;
  actions?: React.ReactNode;
  children: React.ReactNode;
}

const Card: React.FC<CardProps> = ({ title, actions, children }) => (
  <div className="card">
    <div className="card-header">
      <h3>{title}</h3>
      {actions && <div className="card-actions">{actions}</div>}
    </div>
    <div className="card-body">{children}</div>
  </div>
);

// Specialized cards through composition
const PlatformCard: React.FC<{ platform: Platform }> = ({ platform }) => (
  <Card
    title={platform.name}
    actions={
      <>
        <IconButton icon="edit" />
        <IconButton icon="delete" />
      </>
    }
  >
    <PlatformDetails platform={platform} />
  </Card>
);
```

---

## 🗄️ State Management

### Zustand Store Structure

```typescript
// store/platform.store.ts
import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';

interface Platform {
  id: string;
  name: string;
  namespace: string;
  status: PlatformStatus;
}

interface PlatformState {
  // State
  platforms: Platform[];
  selectedPlatformId: string | null;
  filters: {
    namespace?: string;
    status?: PlatformStatus;
    search?: string;
  };
  
  // Computed values (using selectors)
  get selectedPlatform(): Platform | undefined;
  get filteredPlatforms(): Platform[];
  
  // Actions
  setPlatforms: (platforms: Platform[]) => void;
  addPlatform: (platform: Platform) => void;
  updatePlatform: (id: string, updates: Partial<Platform>) => void;
  deletePlatform: (id: string) => void;
  selectPlatform: (id: string | null) => void;
  setFilters: (filters: Partial<PlatformState['filters']>) => void;
  clearFilters: () => void;
}

export const usePlatformStore = create<PlatformState>()(
  devtools(
    persist(
      immer((set, get) => ({
        // Initial state
        platforms: [],
        selectedPlatformId: null,
        filters: {},
        
        // Computed values
        get selectedPlatform() {
          const state = get();
          return state.platforms.find(p => p.id === state.selectedPlatformId);
        },
        
        get filteredPlatforms() {
          const state = get();
          return state.platforms.filter(platform => {
            if (state.filters.namespace && platform.namespace !== state.filters.namespace) {
              return false;
            }
            if (state.filters.status && platform.status !== state.filters.status) {
              return false;
            }
            if (state.filters.search) {
              const search = state.filters.search.toLowerCase();
              return platform.name.toLowerCase().includes(search);
            }
            return true;
          });
        },
        
        // Actions
        setPlatforms: (platforms) =>
          set(state => {
            state.platforms = platforms;
          }),
          
        addPlatform: (platform) =>
          set(state => {
            state.platforms.push(platform);
          }),
          
        updatePlatform: (id, updates) =>
          set(state => {
            const index = state.platforms.findIndex(p => p.id === id);
            if (index !== -1) {
              Object.assign(state.platforms[index], updates);
            }
          }),
          
        deletePlatform: (id) =>
          set(state => {
            state.platforms = state.platforms.filter(p => p.id !== id);
            if (state.selectedPlatformId === id) {
              state.selectedPlatformId = null;
            }
          }),
          
        selectPlatform: (id) =>
          set(state => {
            state.selectedPlatformId = id;
          }),
          
        setFilters: (filters) =>
          set(state => {
            Object.assign(state.filters, filters);
          }),
          
        clearFilters: () =>
          set(state => {
            state.filters = {};
          }),
      })),
      {
        name: 'platform-store',
        partialize: (state) => ({
          selectedPlatformId: state.selectedPlatformId,
          filters: state.filters,
        }),
      }
    ),
    {
      name: 'GunjOperatorPlatforms',
    }
  )
);

// Selectors for optimization
export const usePlatformById = (id: string) => 
  usePlatformStore(state => state.platforms.find(p => p.id === id));

export const useFilteredPlatforms = () =>
  usePlatformStore(state => state.filteredPlatforms);
```

---

## 🎨 Styling Standards

### CSS-in-JS with Emotion/MUI

```typescript
// CORRECT: Theme-aware styling with MUI
import { styled } from '@mui/material/styles';
import { Box, Paper } from '@mui/material';

// Styled components
export const StyledCard = styled(Paper)(({ theme }) => ({
  padding: theme.spacing(3),
  borderRadius: theme.shape.borderRadius,
  transition: theme.transitions.create(['box-shadow', 'transform'], {
    duration: theme.transitions.duration.short,
  }),
  '&:hover': {
    boxShadow: theme.shadows[4],
    transform: 'translateY(-2px)',
  },
  [theme.breakpoints.down('sm')]: {
    padding: theme.spacing(2),
  },
}));

// Using sx prop for one-off styles
<Box
  sx={{
    display: 'flex',
    flexDirection: { xs: 'column', md: 'row' },
    gap: 2,
    p: { xs: 2, md: 3 },
    bgcolor: 'background.paper',
    borderRadius: 1,
  }}
>
  {children}
</Box>
```

### CSS Modules for Complex Components

```scss
// PlatformCard.module.scss
.card {
  @apply rounded-lg shadow-md bg-white dark:bg-gray-800;
  transition: all 0.3s ease;
  
  &:hover {
    @apply shadow-lg;
    transform: translateY(-2px);
  }
  
  &.selected {
    @apply ring-2 ring-primary-500;
  }
}

.header {
  @apply flex justify-between items-center p-4 border-b border-gray-200 dark:border-gray-700;
}

.title {
  @apply text-lg font-semibold text-gray-900 dark:text-white;
}

.status {
  @apply inline-flex items-center px-2 py-1 rounded-full text-xs font-medium;
  
  &.ready {
    @apply bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200;
  }
  
  &.pending {
    @apply bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200;
  }
  
  &.failed {
    @apply bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200;
  }
}
```

```typescript
// Using CSS Modules
import styles from './PlatformCard.module.scss';

export const PlatformCard: React.FC<Props> = ({ platform, selected }) => (
  <div className={`${styles.card} ${selected ? styles.selected : ''}`}>
    <div className={styles.header}>
      <h3 className={styles.title}>{platform.name}</h3>
      <span className={`${styles.status} ${styles[platform.status]}`}>
        {platform.status}
      </span>
    </div>
  </div>
);
```

---

## 🧪 Testing Standards

### Component Testing

```typescript
// PlatformCard.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';

import { PlatformCard } from '@/components/platform/PlatformCard';
import { TestWrapper } from '@/tests/utils/TestWrapper';
import { mockPlatform } from '@/tests/mocks/platform';

describe('PlatformCard', () => {
  const defaultProps = {
    platform: mockPlatform,
    onSelect: vi.fn(),
  };
  
  beforeEach(() => {
    vi.clearAllMocks();
  });
  
  it('should render platform information correctly', () => {
    render(
      <TestWrapper>
        <PlatformCard {...defaultProps} />
      </TestWrapper>
    );
    
    expect(screen.getByText(mockPlatform.metadata.name)).toBeInTheDocument();
    expect(screen.getByText(mockPlatform.status.phase)).toBeInTheDocument();
  });
  
  it('should call onSelect when clicked', async () => {
    const user = userEvent.setup();
    
    render(
      <TestWrapper>
        <PlatformCard {...defaultProps} />
      </TestWrapper>
    );
    
    await user.click(screen.getByRole('article'));
    
    expect(defaultProps.onSelect).toHaveBeenCalledWith(mockPlatform);
  });
  
  it('should show loading skeleton when loading', () => {
    render(
      <TestWrapper>
        <PlatformCard {...defaultProps} loading />
      </TestWrapper>
    );
    
    expect(screen.getByTestId('platform-card-skeleton')).toBeInTheDocument();
  });
  
  it('should expand details when detailed prop is true', async () => {
    const user = userEvent.setup();
    
    render(
      <TestWrapper>
        <PlatformCard {...defaultProps} detailed />
      </TestWrapper>
    );
    
    const expandButton = screen.getByRole('button', { name: /expand/i });
    await user.click(expandButton);
    
    await waitFor(() => {
      expect(screen.getByTestId('platform-details')).toBeInTheDocument();
    });
  });
});
```

### Hook Testing

```typescript
// usePlatform.test.ts
import { renderHook, waitFor } from '@testing-library/react';
import { vi } from 'vitest';

import { usePlatform } from '@/hooks/usePlatform';
import { TestWrapper } from '@/tests/utils/TestWrapper';
import { platformAPI } from '@/api/platform';

vi.mock('@/api/platform');

describe('usePlatform', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });
  
  it('should fetch platforms on mount', async () => {
    const mockPlatforms = [
      { id: '1', name: 'platform-1' },
      { id: '2', name: 'platform-2' },
    ];
    
    (platformAPI.list as jest.Mock).mockResolvedValue(mockPlatforms);
    
    const { result } = renderHook(() => usePlatform(), {
      wrapper: TestWrapper,
    });
    
    expect(result.current.isLoading).toBe(true);
    
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    
    expect(result.current.platforms).toEqual(mockPlatforms);
    expect(platformAPI.list).toHaveBeenCalledWith('default', expect.any(Object));
  });
  
  it('should create a platform', async () => {
    const newPlatform = { id: '3', name: 'new-platform' };
    (platformAPI.create as jest.Mock).mockResolvedValue(newPlatform);
    
    const { result } = renderHook(() => usePlatform(), {
      wrapper: TestWrapper,
    });
    
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    
    await result.current.createPlatform({
      name: 'new-platform',
      namespace: 'default',
    });
    
    expect(platformAPI.create).toHaveBeenCalledWith('default', {
      name: 'new-platform',
      namespace: 'default',
    });
  });
});
```

---

## ⚡ Performance Standards

### Component Optimization

```typescript
// CORRECT: Optimized component with memoization
import React, { memo, useMemo, useCallback } from 'react';
import { FixedSizeList as List } from 'react-window';

interface LargePlatformListProps {
  platforms: Platform[];
  onSelectPlatform: (platform: Platform) => void;
}

// Memoize expensive computations
const PlatformListItem = memo<{ platform: Platform; onSelect: () => void }>(
  ({ platform, onSelect }) => (
    <div onClick={onSelect} className="platform-item">
      {platform.name}
    </div>
  ),
  (prevProps, nextProps) => {
    // Custom comparison for better performance
    return (
      prevProps.platform.id === nextProps.platform.id &&
      prevProps.platform.status === nextProps.platform.status
    );
  }
);

export const LargePlatformList: React.FC<LargePlatformListProps> = memo(({
  platforms,
  onSelectPlatform,
}) => {
  // Memoize the row renderer
  const Row = useCallback(
    ({ index, style }: { index: number; style: React.CSSProperties }) => {
      const platform = platforms[index];
      return (
        <div style={style}>
          <PlatformListItem
            platform={platform}
            onSelect={() => onSelectPlatform(platform)}
          />
        </div>
      );
    },
    [platforms, onSelectPlatform]
  );
  
  // Memoize expensive calculations
  const itemData = useMemo(
    () => ({ platforms, onSelectPlatform }),
    [platforms, onSelectPlatform]
  );
  
  return (
    <List
      height={600}
      itemCount={platforms.length}
      itemSize={80}
      width="100%"
      itemData={itemData}
    >
      {Row}
    </List>
  );
});
```

### Code Splitting

```typescript
// CORRECT: Lazy loading with Suspense
import React, { lazy, Suspense } from 'react';
import { Routes, Route } from 'react-router-dom';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';
import { PageLoader } from '@/components/common/PageLoader';

// Lazy load pages
const PlatformsPage = lazy(() => import('@/pages/PlatformsPage'));
const PlatformDetailPage = lazy(() => import('@/pages/PlatformDetailPage'));
const SettingsPage = lazy(() => import('@/pages/SettingsPage'));

// Lazy load heavy components
const MetricsDashboard = lazy(() => 
  import('@/components/dashboard/MetricsDashboard')
);

export const AppRoutes: React.FC = () => (
  <ErrorBoundary>
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route path="/platforms" element={<PlatformsPage />} />
        <Route path="/platforms/:id" element={<PlatformDetailPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route 
          path="/metrics" 
          element={
            <Suspense fallback={<PageLoader />}>
              <MetricsDashboard />
            </Suspense>
          } 
        />
      </Routes>
    </Suspense>
  </ErrorBoundary>
);
```

---

## 🔒 Security Standards

### Input Sanitization

```typescript
// CORRECT: Proper input sanitization
import DOMPurify from 'isomorphic-dompurify';

interface UserContentProps {
  content: string;
  allowedTags?: string[];
}

export const UserContent: React.FC<UserContentProps> = ({ 
  content, 
  allowedTags = ['p', 'br', 'strong', 'em', 'a'] 
}) => {
  const sanitizedContent = useMemo(
    () => DOMPurify.sanitize(content, { ALLOWED_TAGS: allowedTags }),
    [content, allowedTags]
  );
  
  return (
    <div 
      dangerouslySetInnerHTML={{ __html: sanitizedContent }}
      className="user-content"
    />
  );
};
```

### Authentication & Authorization

```typescript
// CORRECT: Protected route component
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requiredRole?: string;
  requiredPermissions?: string[];
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  children,
  requiredRole,
  requiredPermissions = [],
}) => {
  const { user, isLoading } = useAuth();
  const location = useLocation();
  
  if (isLoading) {
    return <PageLoader />;
  }
  
  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  
  if (requiredRole && user.role !== requiredRole) {
    return <Navigate to="/unauthorized" replace />;
  }
  
  if (requiredPermissions.length > 0) {
    const hasPermissions = requiredPermissions.every(
      permission => user.permissions.includes(permission)
    );
    
    if (!hasPermissions) {
      return <Navigate to="/unauthorized" replace />;
    }
  }
  
  return <>{children}</>;
};
```

---

## ♿ Accessibility Standards

### ARIA and Semantic HTML

```typescript
// CORRECT: Accessible component implementation
interface AccessibleModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

export const AccessibleModal: React.FC<AccessibleModalProps> = ({
  isOpen,
  onClose,
  title,
  children,
}) => {
  const titleId = useId();
  const descriptionId = useId();
  
  // Trap focus within modal
  const modalRef = useRef<HTMLDivElement>(null);
  useFocusTrap(modalRef, isOpen);
  
  // Handle escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);
  
  if (!isOpen) return null;
  
  return (
    <Portal>
      <div
        className="modal-overlay"
        onClick={onClose}
        aria-hidden="true"
      />
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descriptionId}
        className="modal"
      >
        <h2 id={titleId} className="modal-title">
          {title}
        </h2>
        <button
          onClick={onClose}
          className="modal-close"
          aria-label="Close modal"
        >
          <CloseIcon aria-hidden="true" />
        </button>
        <div id={descriptionId} className="modal-content">
          {children}
        </div>
      </div>
    </Portal>
  );
};
```

### Keyboard Navigation

```typescript
// CORRECT: Keyboard-navigable list
export const KeyboardNavigableList: React.FC<{ items: Item[] }> = ({ items }) => {
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const itemRefs = useRef<(HTMLLIElement | null)[]>([]);
  
  const handleKeyDown = (e: KeyboardEvent<HTMLUListElement>) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setFocusedIndex(prev => 
          prev < items.length - 1 ? prev + 1 : prev
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setFocusedIndex(prev => prev > 0 ? prev - 1 : prev);
        break;
      case 'Home':
        e.preventDefault();
        setFocusedIndex(0);
        break;
      case 'End':
        e.preventDefault();
        setFocusedIndex(items.length - 1);
        break;
    }
  };
  
  useEffect(() => {
    if (focusedIndex >= 0 && itemRefs.current[focusedIndex]) {
      itemRefs.current[focusedIndex]?.focus();
    }
  }, [focusedIndex]);
  
  return (
    <ul
      role="list"
      onKeyDown={handleKeyDown}
      className="keyboard-list"
    >
      {items.map((item, index) => (
        <li
          key={item.id}
          ref={el => itemRefs.current[index] = el}
          tabIndex={focusedIndex === index ? 0 : -1}
          role="listitem"
          aria-selected={focusedIndex === index}
          className="keyboard-list-item"
        >
          {item.name}
        </li>
      ))}
    </ul>
  );
};
```

---

## 📁 File Templates

### Component Template

```typescript
// components/[ComponentName]/[ComponentName].tsx
import React, { memo } from 'react';
import { Box } from '@mui/material';

import { useStyles } from './[ComponentName].styles';
import { [ComponentName]Props } from './[ComponentName].types';

/**
 * Brief description of what this component does
 * 
 * @example
 * <[ComponentName] prop1="value" />
 */
export const [ComponentName]: React.FC<[ComponentName]Props> = memo(({
  // Destructure props here
}) => {
  const { classes } = useStyles();
  
  // Component logic here
  
  return (
    <Box className={classes.root}>
      {/* Component JSX */}
    </Box>
  );
});

[ComponentName].displayName = '[ComponentName]';
```

### Hook Template

```typescript
// hooks/use[HookName].ts
import { useState, useEffect, useCallback } from 'react';

/**
 * Brief description of what this hook does
 * 
 * @example
 * const { data, loading } = use[HookName]();
 */
export function use[HookName]<T = unknown>(
  initialValue?: T
): [HookName]Return<T> {
  const [state, setState] = useState<T | undefined>(initialValue);
  
  // Hook logic here
  
  return {
    state,
    setState,
  };
}

interface [HookName]Return<T> {
  state: T | undefined;
  setState: (value: T) => void;
}
```

### Test Template

```typescript
// __tests__/[ComponentName].test.tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';

import { [ComponentName] } from '@/components/[ComponentName]';
import { TestWrapper } from '@/tests/utils';

describe('[ComponentName]', () => {
  const defaultProps = {
    // Default props here
  };
  
  beforeEach(() => {
    vi.clearAllMocks();
  });
  
  it('should render correctly', () => {
    render(
      <TestWrapper>
        <[ComponentName] {...defaultProps} />
      </TestWrapper>
    );
    
    expect(screen.getByRole('...')).toBeInTheDocument();
  });
  
  it('should handle user interactions', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();
    
    render(
      <TestWrapper>
        <[ComponentName] {...defaultProps} onClick={handleClick} />
      </TestWrapper>
    );
    
    await user.click(screen.getByRole('button'));
    
    expect(handleClick).toHaveBeenCalled();
  });
});
```

---

## 📝 Linting Configuration

### ESLint Configuration

```javascript
// .eslintrc.js
module.exports = {
  extends: [
    'eslint:recommended',
    'plugin:react/recommended',
    'plugin:react-hooks/recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking',
    'plugin:jsx-a11y/recommended',
    'plugin:import/recommended',
    'plugin:import/typescript',
    'prettier'
  ],
  plugins: [
    'react',
    'react-hooks',
    '@typescript-eslint',
    'jsx-a11y',
    'import',
    'unused-imports'
  ],
  rules: {
    // React rules
    'react/prop-types': 'off',
    'react/react-in-jsx-scope': 'off',
    'react/display-name': 'warn',
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'warn',
    
    // TypeScript rules
    '@typescript-eslint/explicit-module-boundary-types': 'off',
    '@typescript-eslint/no-unused-vars': ['error', { 
      argsIgnorePattern: '^_',
      varsIgnorePattern: '^_'
    }],
    '@typescript-eslint/no-explicit-any': 'error',
    '@typescript-eslint/consistent-type-imports': 'error',
    
    // Import rules
    'import/order': ['error', {
      groups: [
        'builtin',
        'external',
        'internal',
        'parent',
        'sibling',
        'index'
      ],
      'newlines-between': 'always',
      alphabetize: { order: 'asc' }
    }],
    
    // Accessibility
    'jsx-a11y/anchor-is-valid': 'warn',
    
    // General
    'no-console': ['warn', { allow: ['warn', 'error'] }],
    'unused-imports/no-unused-imports': 'error'
  }
};
```

### Prettier Configuration

```javascript
// .prettierrc.js
module.exports = {
  semi: true,
  trailingComma: 'es5',
  singleQuote: true,
  printWidth: 80,
  tabWidth: 2,
  useTabs: false,
  arrowParens: 'always',
  endOfLine: 'lf'
};
```

---

**This document establishes the definitive React/TypeScript standards for the Gunj Operator UI development.**