# TypeScript/React Coding Standards

This document outlines the TypeScript and React coding standards for the Gunj Operator UI project. All TypeScript and React code must follow these guidelines to ensure consistency, type safety, and maintainability.

## Table of Contents

1. [General Guidelines](#general-guidelines)
2. [TypeScript Standards](#typescript-standards)
3. [React Standards](#react-standards)
4. [State Management](#state-management)
5. [Styling Guidelines](#styling-guidelines)
6. [Testing Standards](#testing-standards)
7. [Performance Guidelines](#performance-guidelines)
8. [Accessibility Standards](#accessibility-standards)

## General Guidelines

### Code Formatting

- **Use Prettier**: All code must be formatted with Prettier
- **Use ESLint**: Follow the project ESLint configuration
- **Line length**: Maximum 100 characters
- **File naming**: Use kebab-case for files, PascalCase for components

```bash
# Format code
npm run format

# Run linters
npm run lint

# Type check
npm run type-check
```

### TypeScript Version

- Target TypeScript 5.0+
- Use strict mode
- Enable all strict checks

```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "strictFunctionTypes": true,
    "strictBindCallApply": true,
    "strictPropertyInitialization": true,
    "noImplicitThis": true,
    "alwaysStrict": true
  }
}
```

## TypeScript Standards

### Type Definitions

```typescript
// Good: Explicit interfaces for props
interface PlatformCardProps {
  /** The platform to display */
  platform: Platform;
  
  /** Called when platform is selected */
  onSelect?: (platform: Platform) => void;
  
  /** Whether the card is in loading state */
  loading?: boolean;
  
  /** Additional CSS classes */
  className?: string;
}

// Good: Use type for unions and intersections
type PlatformStatus = 'pending' | 'installing' | 'ready' | 'failed' | 'upgrading';

type PlatformWithMetrics = Platform & {
  metrics: PlatformMetrics;
};

// Bad: Using 'any' type
const processData = (data: any) => { // ❌ Avoid any
  return data.value;
};

// Good: Proper typing
const processData = <T extends { value: unknown }>(data: T): T['value'] => {
  return data.value;
};
```

### Enums and Constants

```typescript
// Good: Use const assertions for literal types
export const ComponentType = {
  PROMETHEUS: 'prometheus',
  GRAFANA: 'grafana',
  LOKI: 'loki',
  TEMPO: 'tempo',
} as const;

export type ComponentType = typeof ComponentType[keyof typeof ComponentType];

// Good: Enum for numeric values
export enum HttpStatus {
  OK = 200,
  BadRequest = 400,
  Unauthorized = 401,
  NotFound = 404,
  InternalServerError = 500,
}
```

### Utility Types

```typescript
// Good: Use utility types effectively
type PartialPlatform = Partial<Platform>;
type RequiredPlatform = Required<Platform>;
type PlatformKeys = keyof Platform;

// Custom utility types
type Nullable<T> = T | null;
type Optional<T> = T | undefined;
type AsyncFunction<T = void> = () => Promise<T>;

// Good: Type guards
export function isPlatform(value: unknown): value is Platform {
  return (
    typeof value === 'object' &&
    value !== null &&
    'metadata' in value &&
    'spec' in value
  );
}
```

### Generics

```typescript
// Good: Meaningful generic names
interface ApiResponse<TData, TError = ApiError> {
  data?: TData;
  error?: TError;
  loading: boolean;
}

// Good: Generic constraints
function updateEntity<T extends { id: string }>(
  entities: T[],
  id: string,
  updates: Partial<T>
): T[] {
  return entities.map(entity =>
    entity.id === id ? { ...entity, ...updates } : entity
  );
}
```

## React Standards

### Component Structure

```typescript
// Good: Functional component with proper typing
import React, { useState, useCallback, useMemo } from 'react';
import { Box, Card, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';

import { Platform } from '@/types';
import { usePlatformStatus } from '@/hooks';
import { StatusIndicator } from '@/components/common';

interface PlatformCardProps {
  platform: Platform;
  onSelect?: (platform: Platform) => void;
  disabled?: boolean;
}

export const PlatformCard: React.FC<PlatformCardProps> = React.memo(({
  platform,
  onSelect,
  disabled = false,
}) => {
  const { t } = useTranslation();
  const [isHovered, setIsHovered] = useState(false);
  const status = usePlatformStatus(platform.metadata.uid);

  const handleClick = useCallback(() => {
    if (!disabled && onSelect) {
      onSelect(platform);
    }
  }, [disabled, onSelect, platform]);

  const statusColor = useMemo(() => {
    switch (status) {
      case 'ready':
        return 'success';
      case 'failed':
        return 'error';
      case 'installing':
      case 'upgrading':
        return 'warning';
      default:
        return 'default';
    }
  }, [status]);

  return (
    <Card
      onClick={handleClick}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      sx={{
        cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.6 : 1,
        transform: isHovered ? 'translateY(-2px)' : 'none',
        transition: 'all 0.2s',
      }}
    >
      <Box p={2}>
        <Typography variant="h6">{platform.metadata.name}</Typography>
        <StatusIndicator status={status} color={statusColor} />
      </Box>
    </Card>
  );
});

PlatformCard.displayName = 'PlatformCard';
```

### Hooks Usage

```typescript
// Good: Custom hooks with proper typing
import { useState, useEffect, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface UsePlatformOptions {
  namespace?: string;
  autoRefresh?: boolean;
  refetchInterval?: number;
}

export function usePlatform(
  name: string,
  options: UsePlatformOptions = {}
) {
  const {
    namespace = 'default',
    autoRefresh = true,
    refetchInterval = 30000,
  } = options;

  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['platform', namespace, name],
    queryFn: () => platformAPI.get(name, namespace),
    refetchInterval: autoRefresh ? refetchInterval : false,
    staleTime: 5000,
  });

  const updateMutation = useMutation({
    mutationFn: (updates: Partial<Platform>) =>
      platformAPI.update(name, namespace, updates),
    onSuccess: (data) => {
      queryClient.setQueryData(['platform', namespace, name], data);
      queryClient.invalidateQueries(['platforms', namespace]);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => platformAPI.delete(name, namespace),
    onSuccess: () => {
      queryClient.invalidateQueries(['platforms']);
      queryClient.removeQueries(['platform', namespace, name]);
    },
  });

  return {
    platform: query.data,
    loading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    update: updateMutation.mutate,
    delete: deleteMutation.mutate,
    isUpdating: updateMutation.isLoading,
    isDeleting: deleteMutation.isLoading,
  };
}
```

### Event Handlers

```typescript
// Good: Properly typed event handlers
interface FormData {
  name: string;
  namespace: string;
  components: ComponentConfig[];
}

export const PlatformForm: React.FC = () => {
  const [formData, setFormData] = useState<FormData>({
    name: '',
    namespace: 'default',
    components: [],
  });

  // Good: Generic handler for form fields
  const handleInputChange = useCallback(
    <K extends keyof FormData>(field: K) =>
      (event: React.ChangeEvent<HTMLInputElement>) => {
        setFormData(prev => ({
          ...prev,
          [field]: event.target.value,
        }));
      },
    []
  );

  // Good: Typed event handler
  const handleSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      
      try {
        await createPlatform(formData);
      } catch (error) {
        console.error('Failed to create platform:', error);
      }
    },
    [formData]
  );

  return (
    <form onSubmit={handleSubmit}>
      <input
        value={formData.name}
        onChange={handleInputChange('name')}
        required
      />
      {/* More form fields */}
    </form>
  );
};
```

## State Management

### Zustand Store

```typescript
// Good: Typed Zustand store
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
  };
  
  // Actions
  setPlatforms: (platforms: Platform[]) => void;
  addPlatform: (platform: Platform) => void;
  updatePlatform: (id: string, updates: Partial<Platform>) => void;
  deletePlatform: (id: string) => void;
  selectPlatform: (id: string | null) => void;
  setFilter: <K extends keyof PlatformState['filters']>(
    key: K,
    value: PlatformState['filters'][K]
  ) => void;
  
  // Computed
  getSelectedPlatform: () => Platform | undefined;
  getFilteredPlatforms: () => Platform[];
}

export const usePlatformStore = create<PlatformState>()(
  devtools(
    persist(
      immer((set, get) => ({
        // State
        platforms: [],
        selectedPlatformId: null,
        filters: {},
        
        // Actions
        setPlatforms: (platforms) =>
          set((state) => {
            state.platforms = platforms;
          }),
          
        addPlatform: (platform) =>
          set((state) => {
            state.platforms.push(platform);
          }),
          
        updatePlatform: (id, updates) =>
          set((state) => {
            const index = state.platforms.findIndex(p => p.id === id);
            if (index !== -1) {
              Object.assign(state.platforms[index], updates);
            }
          }),
          
        deletePlatform: (id) =>
          set((state) => {
            state.platforms = state.platforms.filter(p => p.id !== id);
            if (state.selectedPlatformId === id) {
              state.selectedPlatformId = null;
            }
          }),
          
        selectPlatform: (id) =>
          set((state) => {
            state.selectedPlatformId = id;
          }),
          
        setFilter: (key, value) =>
          set((state) => {
            if (value === undefined) {
              delete state.filters[key];
            } else {
              state.filters[key] = value;
            }
          }),
          
        // Computed
        getSelectedPlatform: () => {
          const state = get();
          return state.platforms.find(p => p.id === state.selectedPlatformId);
        },
        
        getFilteredPlatforms: () => {
          const state = get();
          return state.platforms.filter(platform => {
            if (state.filters.namespace && platform.namespace !== state.filters.namespace) {
              return false;
            }
            if (state.filters.status && platform.status !== state.filters.status) {
              return false;
            }
            return true;
          });
        },
      })),
      {
        name: 'platform-store',
        partialize: (state) => ({
          selectedPlatformId: state.selectedPlatformId,
          filters: state.filters,
        }),
      }
    )
  )
);
```

## Styling Guidelines

### CSS-in-JS with MUI

```typescript
// Good: Type-safe theme usage
import { styled } from '@mui/material/styles';
import { Box, BoxProps } from '@mui/material';

interface StatusBoxProps extends BoxProps {
  status: 'success' | 'warning' | 'error';
}

export const StatusBox = styled(Box)<StatusBoxProps>(({ theme, status }) => ({
  padding: theme.spacing(2),
  borderRadius: theme.shape.borderRadius,
  backgroundColor: theme.palette[status].light,
  color: theme.palette[status].dark,
  border: `1px solid ${theme.palette[status].main}`,
  
  '&:hover': {
    backgroundColor: theme.palette[status].main,
    color: theme.palette[status].contrastText,
  },
}));

// Good: sx prop usage
<Card
  sx={{
    p: 2,
    mb: 2,
    cursor: 'pointer',
    transition: 'all 0.2s',
    '&:hover': {
      transform: 'translateY(-2px)',
      boxShadow: 4,
    },
  }}
>
  {/* Content */}
</Card>
```

## Testing Standards

### Component Testing

```typescript
// Good: Comprehensive component tests
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, describe, it, expect, beforeEach } from 'vitest';

import { PlatformCard } from './platform-card';
import { Platform } from '@/types';

// Test utilities
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
};

describe('PlatformCard', () => {
  const mockPlatform: Platform = {
    metadata: {
      uid: '123',
      name: 'test-platform',
      namespace: 'default',
    },
    spec: {
      components: {
        prometheus: { enabled: true },
      },
    },
    status: {
      phase: 'ready',
    },
  };

  const mockOnSelect = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render platform information', () => {
    render(
      <PlatformCard platform={mockPlatform} />,
      { wrapper: createWrapper() }
    );

    expect(screen.getByText('test-platform')).toBeInTheDocument();
    expect(screen.getByTestId('status-indicator')).toHaveAttribute(
      'data-status',
      'ready'
    );
  });

  it('should call onSelect when clicked', async () => {
    const user = userEvent.setup();

    render(
      <PlatformCard platform={mockPlatform} onSelect={mockOnSelect} />,
      { wrapper: createWrapper() }
    );

    await user.click(screen.getByRole('article'));

    expect(mockOnSelect).toHaveBeenCalledWith(mockPlatform);
  });

  it('should not call onSelect when disabled', async () => {
    const user = userEvent.setup();

    render(
      <PlatformCard
        platform={mockPlatform}
        onSelect={mockOnSelect}
        disabled
      />,
      { wrapper: createWrapper() }
    );

    await user.click(screen.getByRole('article'));

    expect(mockOnSelect).not.toHaveBeenCalled();
  });

  it('should show hover effects', async () => {
    const user = userEvent.setup();

    render(
      <PlatformCard platform={mockPlatform} />,
      { wrapper: createWrapper() }
    );

    const card = screen.getByRole('article');

    await user.hover(card);
    expect(card).toHaveStyle({ transform: 'translateY(-2px)' });

    await user.unhover(card);
    expect(card).toHaveStyle({ transform: 'none' });
  });
});
```

### Hook Testing

```typescript
// Good: Hook testing with renderHook
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { usePlatform } from './use-platform';
import * as platformAPI from '@/api/platform';

vi.mock('@/api/platform');

describe('usePlatform', () => {
  const createWrapper = () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    return ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };

  it('should fetch platform data', async () => {
    const mockPlatform = { name: 'test-platform' };
    vi.mocked(platformAPI.get).mockResolvedValue(mockPlatform);

    const { result } = renderHook(
      () => usePlatform('test-platform'),
      { wrapper: createWrapper() }
    );

    expect(result.current.loading).toBe(true);

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.platform).toEqual(mockPlatform);
    expect(platformAPI.get).toHaveBeenCalledWith('test-platform', 'default');
  });

  it('should update platform', async () => {
    const mockPlatform = { name: 'test-platform' };
    vi.mocked(platformAPI.get).mockResolvedValue(mockPlatform);
    vi.mocked(platformAPI.update).mockResolvedValue({
      ...mockPlatform,
      status: 'updated',
    });

    const { result } = renderHook(
      () => usePlatform('test-platform'),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    act(() => {
      result.current.update({ status: 'updated' });
    });

    await waitFor(() => {
      expect(result.current.isUpdating).toBe(false);
    });

    expect(platformAPI.update).toHaveBeenCalledWith(
      'test-platform',
      'default',
      { status: 'updated' }
    );
  });
});
```

## Performance Guidelines

### Memoization

```typescript
// Good: Proper memoization
export const ExpensiveComponent: React.FC<{ data: DataItem[] }> = ({ data }) => {
  // Memoize expensive computations
  const processedData = useMemo(() => {
    return data
      .filter(item => item.active)
      .sort((a, b) => b.priority - a.priority)
      .map(item => ({
        ...item,
        displayName: `${item.name} (${item.type})`,
      }));
  }, [data]);

  // Memoize callbacks
  const handleItemClick = useCallback((id: string) => {
    console.log('Item clicked:', id);
  }, []);

  return (
    <List>
      {processedData.map(item => (
        <ListItem key={item.id} onClick={() => handleItemClick(item.id)}>
          {item.displayName}
        </ListItem>
      ))}
    </List>
  );
};
```

### Code Splitting

```typescript
// Good: Lazy loading routes
import { lazy, Suspense } from 'react';
import { Routes, Route } from 'react-router-dom';

const PlatformList = lazy(() => import('./pages/platform-list'));
const PlatformDetail = lazy(() => import('./pages/platform-detail'));
const Settings = lazy(() => import('./pages/settings'));

export const AppRoutes: React.FC = () => {
  return (
    <Suspense fallback={<LoadingScreen />}>
      <Routes>
        <Route path="/platforms" element={<PlatformList />} />
        <Route path="/platforms/:id" element={<PlatformDetail />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Suspense>
  );
};
```

## Accessibility Standards

### ARIA and Semantic HTML

```typescript
// Good: Accessible form
export const PlatformForm: React.FC = () => {
  const [errors, setErrors] = useState<Record<string, string>>({});

  return (
    <form aria-label="Create platform form">
      <FormControl error={!!errors.name}>
        <InputLabel htmlFor="platform-name">
          Platform Name
          <span aria-label="required">*</span>
        </InputLabel>
        <Input
          id="platform-name"
          name="name"
          aria-describedby={errors.name ? 'name-error' : 'name-helper'}
          aria-invalid={!!errors.name}
          required
        />
        {errors.name ? (
          <FormHelperText id="name-error" role="alert">
            {errors.name}
          </FormHelperText>
        ) : (
          <FormHelperText id="name-helper">
            Enter a unique platform name
          </FormHelperText>
        )}
      </FormControl>
    </form>
  );
};
```

### Keyboard Navigation

```typescript
// Good: Keyboard accessible component
export const NavigationMenu: React.FC = () => {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const items = ['Dashboard', 'Platforms', 'Settings'];

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      switch (event.key) {
        case 'ArrowUp':
          event.preventDefault();
          setSelectedIndex(prev => 
            prev > 0 ? prev - 1 : items.length - 1
          );
          break;
        case 'ArrowDown':
          event.preventDefault();
          setSelectedIndex(prev => 
            prev < items.length - 1 ? prev + 1 : 0
          );
          break;
        case 'Enter':
        case ' ':
          event.preventDefault();
          // Handle selection
          break;
      }
    },
    [items.length]
  );

  return (
    <nav role="navigation" aria-label="Main navigation">
      <ul role="menu" onKeyDown={handleKeyDown}>
        {items.map((item, index) => (
          <li
            key={item}
            role="menuitem"
            tabIndex={index === selectedIndex ? 0 : -1}
            aria-selected={index === selectedIndex}
          >
            {item}
          </li>
        ))}
      </ul>
    </nav>
  );
};
```

## Code Review Checklist

Before submitting code for review:

- [ ] TypeScript strict mode passes
- [ ] No `any` types used
- [ ] All props are properly typed
- [ ] Components have display names
- [ ] Event handlers are properly typed
- [ ] Memoization used appropriately
- [ ] No unnecessary re-renders
- [ ] Accessibility requirements met
- [ ] Tests cover main scenarios
- [ ] Error boundaries in place
- [ ] Loading and error states handled

## Tools and Commands

```bash
# Development
npm run dev          # Start dev server
npm run build        # Build for production
npm run preview      # Preview production build

# Code quality
npm run lint         # Run ESLint
npm run lint:fix     # Fix ESLint issues
npm run format       # Format with Prettier
npm run type-check   # Run TypeScript compiler

# Testing
npm run test         # Run tests
npm run test:watch   # Run tests in watch mode
npm run test:coverage # Generate coverage report
npm run test:ui      # Open Vitest UI

# Analysis
npm run analyze      # Analyze bundle size
npm run lighthouse   # Run Lighthouse audit
```

## Resources

- [TypeScript Documentation](https://www.typescriptlang.org/docs/)
- [React Documentation](https://react.dev/)
- [React TypeScript Cheatsheet](https://react-typescript-cheatsheet.netlify.app/)
- [MUI Documentation](https://mui.com/)
- [React Query Documentation](https://tanstack.com/query/latest)
- [Zustand Documentation](https://github.com/pmndrs/zustand)

---

By following these standards, we ensure our TypeScript and React code is type-safe, performant, accessible, and maintainable.
