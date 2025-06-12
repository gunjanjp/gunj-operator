# Gunj Operator UI State Management Architecture

**Version**: 1.0  
**Date**: June 12, 2025  
**Status**: Draft  
**Decision**: Zustand for State Management  

---

## 📋 Executive Summary

This document outlines the state management architecture for the Gunj Operator UI using Zustand. The architecture is designed to handle complex state requirements including real-time updates, multi-cluster management, and seamless integration with WebSocket/SSE streams.

### 🎯 Key Design Principles

1. **Domain Separation**: Separate stores for different business domains
2. **Performance First**: Optimized for minimal re-renders
3. **Type Safety**: Full TypeScript support
4. **Real-time Ready**: Built-in support for WebSocket/SSE updates
5. **Developer Experience**: Simple API with powerful capabilities
6. **Testability**: Easy to test with minimal mocking

---

## 🏗️ Architecture Overview

### Store Structure

```
stores/
├── auth/                    # Authentication & Authorization
│   ├── authStore.ts        # User auth state
│   └── permissionStore.ts  # RBAC permissions
├── platform/               # Platform Management
│   ├── platformStore.ts    # Platform CRUD
│   ├── componentStore.ts   # Component states
│   └── statusStore.ts      # Real-time status
├── monitoring/             # Observability Data
│   ├── metricsStore.ts    # Metrics data
│   ├── logsStore.ts       # Log streaming
│   ├── tracesStore.ts     # Distributed traces
│   └── alertsStore.ts     # Alert management
├── ui/                     # UI State
│   ├── themeStore.ts      # Theme preferences
│   ├── layoutStore.ts     # Layout state
│   └── notificationStore.ts # Notifications
├── realtime/              # Real-time Updates
│   ├── websocketStore.ts  # WebSocket management
│   └── eventStore.ts      # Event streaming
└── index.ts               # Store exports
```

---

## 📦 Domain Store Designs

### 1. Authentication Store

```typescript
// stores/auth/authStore.ts
import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';

interface User {
  id: string;
  email: string;
  name: string;
  roles: string[];
  permissions: string[];
  avatarUrl?: string;
}

interface AuthState {
  // State
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  accessToken: string | null;
  refreshToken: string | null;
  
  // Actions
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => void;
  refreshTokens: () => Promise<void>;
  updateUser: (updates: Partial<User>) => void;
  checkPermission: (permission: string) => boolean;
  hasRole: (role: string) => boolean;
}

export const useAuthStore = create<AuthState>()(
  devtools(
    persist(
      immer((set, get) => ({
        // Initial state
        user: null,
        isAuthenticated: false,
        isLoading: false,
        error: null,
        accessToken: null,
        refreshToken: null,

        // Actions
        login: async (credentials) => {
          set((state) => {
            state.isLoading = true;
            state.error = null;
          });

          try {
            const response = await authAPI.login(credentials);
            
            set((state) => {
              state.user = response.user;
              state.accessToken = response.accessToken;
              state.refreshToken = response.refreshToken;
              state.isAuthenticated = true;
              state.isLoading = false;
            });

            // Initialize WebSocket connection
            useWebSocketStore.getState().connect(response.accessToken);
          } catch (error) {
            set((state) => {
              state.error = error.message;
              state.isLoading = false;
            });
          }
        },

        logout: () => {
          set((state) => {
            state.user = null;
            state.accessToken = null;
            state.refreshToken = null;
            state.isAuthenticated = false;
          });

          // Disconnect WebSocket
          useWebSocketStore.getState().disconnect();
          
          // Clear all stores
          usePlatformStore.getState().reset();
          useMetricsStore.getState().reset();
        },

        checkPermission: (permission) => {
          const user = get().user;
          return user?.permissions.includes(permission) ?? false;
        },

        hasRole: (role) => {
          const user = get().user;
          return user?.roles.includes(role) ?? false;
        },
      })),
      {
        name: 'auth-store',
        partialize: (state) => ({
          accessToken: state.accessToken,
          refreshToken: state.refreshToken,
        }),
      }
    )
  )
);
```

### 2. Platform Store

```typescript
// stores/platform/platformStore.ts
import { create } from 'zustand';
import { devtools, subscribeWithSelector } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';

interface Platform {
  id: string;
  name: string;
  namespace: string;
  spec: PlatformSpec;
  status: PlatformStatus;
  metadata: {
    createdAt: string;
    updatedAt: string;
    version: string;
  };
}

interface PlatformState {
  // State
  platforms: Record<string, Platform>;
  selectedPlatformId: string | null;
  isLoading: boolean;
  error: string | null;
  filters: PlatformFilters;
  sortBy: SortCriteria;
  
  // Computed
  selectedPlatform: Platform | null;
  filteredPlatforms: Platform[];
  
  // Actions
  fetchPlatforms: (namespace?: string) => Promise<void>;
  fetchPlatform: (id: string) => Promise<void>;
  createPlatform: (data: CreatePlatformData) => Promise<Platform>;
  updatePlatform: (id: string, updates: Partial<PlatformSpec>) => Promise<void>;
  deletePlatform: (id: string) => Promise<void>;
  selectPlatform: (id: string | null) => void;
  setFilters: (filters: Partial<PlatformFilters>) => void;
  setSortBy: (sortBy: SortCriteria) => void;
  
  // Real-time updates
  handlePlatformUpdate: (update: PlatformUpdate) => void;
  handleStatusUpdate: (id: string, status: PlatformStatus) => void;
  
  // Utilities
  reset: () => void;
}

export const usePlatformStore = create<PlatformState>()(
  subscribeWithSelector(
    devtools(
      immer((set, get) => ({
        // Initial state
        platforms: {},
        selectedPlatformId: null,
        isLoading: false,
        error: null,
        filters: {
          namespace: null,
          status: null,
          search: '',
        },
        sortBy: {
          field: 'name',
          direction: 'asc',
        },

        // Computed values
        get selectedPlatform() {
          const id = get().selectedPlatformId;
          return id ? get().platforms[id] : null;
        },

        get filteredPlatforms() {
          const { platforms, filters, sortBy } = get();
          let filtered = Object.values(platforms);

          // Apply filters
          if (filters.namespace) {
            filtered = filtered.filter(p => p.namespace === filters.namespace);
          }
          if (filters.status) {
            filtered = filtered.filter(p => p.status.phase === filters.status);
          }
          if (filters.search) {
            const search = filters.search.toLowerCase();
            filtered = filtered.filter(p => 
              p.name.toLowerCase().includes(search) ||
              p.namespace.toLowerCase().includes(search)
            );
          }

          // Apply sorting
          filtered.sort((a, b) => {
            const aVal = a[sortBy.field];
            const bVal = b[sortBy.field];
            const direction = sortBy.direction === 'asc' ? 1 : -1;
            return aVal > bVal ? direction : -direction;
          });

          return filtered;
        },

        // Actions
        fetchPlatforms: async (namespace) => {
          set((state) => {
            state.isLoading = true;
            state.error = null;
          });

          try {
            const platforms = await platformAPI.list(namespace);
            
            set((state) => {
              state.platforms = platforms.reduce((acc, platform) => {
                acc[platform.id] = platform;
                return acc;
              }, {} as Record<string, Platform>);
              state.isLoading = false;
            });
          } catch (error) {
            set((state) => {
              state.error = error.message;
              state.isLoading = false;
            });
          }
        },

        handlePlatformUpdate: (update) => {
          set((state) => {
            if (update.type === 'created' || update.type === 'updated') {
              state.platforms[update.platform.id] = update.platform;
            } else if (update.type === 'deleted') {
              delete state.platforms[update.platformId];
              if (state.selectedPlatformId === update.platformId) {
                state.selectedPlatformId = null;
              }
            }
          });
        },

        handleStatusUpdate: (id, status) => {
          set((state) => {
            if (state.platforms[id]) {
              state.platforms[id].status = status;
            }
          });
        },

        reset: () => {
          set((state) => {
            state.platforms = {};
            state.selectedPlatformId = null;
            state.isLoading = false;
            state.error = null;
          });
        },
      }))
    )
  )
);

// Subscribe to real-time updates
usePlatformStore.subscribe(
  (state) => state.selectedPlatformId,
  (platformId) => {
    if (platformId) {
      // Subscribe to platform-specific updates
      useWebSocketStore.getState().subscribe(`platform:${platformId}`, (data) => {
        usePlatformStore.getState().handleStatusUpdate(platformId, data.status);
      });
    }
  }
);
```

### 3. Real-time Store

```typescript
// stores/realtime/websocketStore.ts
import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

interface WebSocketState {
  // Connection state
  ws: WebSocket | null;
  isConnected: boolean;
  reconnectAttempts: number;
  maxReconnectAttempts: number;
  reconnectDelay: number;
  
  // Subscriptions
  subscriptions: Map<string, Set<(data: any) => void>>;
  
  // Actions
  connect: (token: string) => void;
  disconnect: () => void;
  subscribe: (channel: string, callback: (data: any) => void) => () => void;
  unsubscribe: (channel: string, callback: (data: any) => void) => void;
  send: (message: WebSocketMessage) => void;
  
  // Private actions
  handleOpen: () => void;
  handleMessage: (event: MessageEvent) => void;
  handleError: (event: Event) => void;
  handleClose: (event: CloseEvent) => void;
  reconnect: () => void;
}

export const useWebSocketStore = create<WebSocketState>()(
  devtools((set, get) => ({
    // Initial state
    ws: null,
    isConnected: false,
    reconnectAttempts: 0,
    maxReconnectAttempts: 5,
    reconnectDelay: 1000,
    subscriptions: new Map(),

    // Connect to WebSocket
    connect: (token) => {
      const { ws } = get();
      if (ws && ws.readyState === WebSocket.OPEN) {
        return;
      }

      const wsUrl = `${WS_BASE_URL}?token=${token}`;
      const newWs = new WebSocket(wsUrl);

      newWs.onopen = () => get().handleOpen();
      newWs.onmessage = (event) => get().handleMessage(event);
      newWs.onerror = (event) => get().handleError(event);
      newWs.onclose = (event) => get().handleClose(event);

      set({ ws: newWs });
    },

    // Disconnect WebSocket
    disconnect: () => {
      const { ws } = get();
      if (ws) {
        ws.close();
        set({ ws: null, isConnected: false });
      }
    },

    // Subscribe to channel
    subscribe: (channel, callback) => {
      const { subscriptions, ws, isConnected } = get();
      
      if (!subscriptions.has(channel)) {
        subscriptions.set(channel, new Set());
      }
      
      subscriptions.get(channel)!.add(callback);

      // Send subscription message if connected
      if (isConnected && ws) {
        ws.send(JSON.stringify({
          type: 'subscribe',
          channel,
        }));
      }

      // Return unsubscribe function
      return () => get().unsubscribe(channel, callback);
    },

    // Handle WebSocket open
    handleOpen: () => {
      set({ isConnected: true, reconnectAttempts: 0 });
      
      // Re-subscribe to all channels
      const { subscriptions, ws } = get();
      subscriptions.forEach((_, channel) => {
        ws?.send(JSON.stringify({
          type: 'subscribe',
          channel,
        }));
      });

      // Notify UI
      useNotificationStore.getState().addNotification({
        type: 'success',
        message: 'Connected to real-time updates',
      });
    },

    // Handle incoming messages
    handleMessage: (event) => {
      try {
        const message = JSON.parse(event.data);
        const { subscriptions } = get();
        
        if (message.channel && subscriptions.has(message.channel)) {
          subscriptions.get(message.channel)!.forEach(callback => {
            callback(message.data);
          });
        }

        // Handle global events
        switch (message.type) {
          case 'platform_update':
            usePlatformStore.getState().handlePlatformUpdate(message.data);
            break;
          case 'metrics_update':
            useMetricsStore.getState().handleMetricsUpdate(message.data);
            break;
          case 'alert':
            useAlertsStore.getState().handleNewAlert(message.data);
            break;
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    },

    // Handle connection close
    handleClose: (event) => {
      set({ isConnected: false });
      
      // Attempt reconnection if not a normal close
      if (event.code !== 1000) {
        get().reconnect();
      }
    },

    // Reconnection logic
    reconnect: () => {
      const { reconnectAttempts, maxReconnectAttempts, reconnectDelay } = get();
      
      if (reconnectAttempts < maxReconnectAttempts) {
        setTimeout(() => {
          set(state => ({ reconnectAttempts: state.reconnectAttempts + 1 }));
          const token = useAuthStore.getState().accessToken;
          if (token) {
            get().connect(token);
          }
        }, reconnectDelay * Math.pow(2, reconnectAttempts));
      } else {
        useNotificationStore.getState().addNotification({
          type: 'error',
          message: 'Failed to connect to real-time updates',
          action: {
            label: 'Retry',
            onClick: () => {
              set({ reconnectAttempts: 0 });
              get().reconnect();
            },
          },
        });
      }
    },
  }))
);
```

### 4. UI State Store

```typescript
// stores/ui/notificationStore.ts
import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { v4 as uuidv4 } from 'uuid';

interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
  description?: string;
  duration?: number;
  action?: {
    label: string;
    onClick: () => void;
  };
  timestamp: number;
}

interface NotificationState {
  notifications: Notification[];
  maxNotifications: number;
  
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => void;
  removeNotification: (id: string) => void;
  clearAll: () => void;
}

export const useNotificationStore = create<NotificationState>()(
  devtools((set, get) => ({
    notifications: [],
    maxNotifications: 5,

    addNotification: (notification) => {
      const newNotification: Notification = {
        ...notification,
        id: uuidv4(),
        timestamp: Date.now(),
        duration: notification.duration ?? 5000,
      };

      set((state) => {
        const notifications = [newNotification, ...state.notifications];
        
        // Limit number of notifications
        if (notifications.length > state.maxNotifications) {
          notifications.pop();
        }
        
        return { notifications };
      });

      // Auto-remove after duration
      if (newNotification.duration && newNotification.duration > 0) {
        setTimeout(() => {
          get().removeNotification(newNotification.id);
        }, newNotification.duration);
      }
    },

    removeNotification: (id) => {
      set((state) => ({
        notifications: state.notifications.filter(n => n.id !== id),
      }));
    },

    clearAll: () => {
      set({ notifications: [] });
    },
  }))
);
```

---

## 🔄 Real-time Integration

### WebSocket Event Handlers

```typescript
// stores/realtime/eventHandlers.ts

export const platformEventHandler = (event: PlatformEvent) => {
  switch (event.type) {
    case 'platform.created':
    case 'platform.updated':
      usePlatformStore.getState().handlePlatformUpdate({
        type: event.type === 'platform.created' ? 'created' : 'updated',
        platform: event.data,
      });
      break;
      
    case 'platform.deleted':
      usePlatformStore.getState().handlePlatformUpdate({
        type: 'deleted',
        platformId: event.data.id,
      });
      break;
      
    case 'platform.status.changed':
      usePlatformStore.getState().handleStatusUpdate(
        event.data.id,
        event.data.status
      );
      break;
  }
};

export const metricsEventHandler = (event: MetricsEvent) => {
  useMetricsStore.getState().appendMetrics({
    platformId: event.platformId,
    metrics: event.data,
  });
};

export const alertEventHandler = (event: AlertEvent) => {
  const alert = event.data;
  
  // Update alerts store
  useAlertsStore.getState().addAlert(alert);
  
  // Show notification for critical alerts
  if (alert.severity === 'critical') {
    useNotificationStore.getState().addNotification({
      type: 'error',
      message: `Critical Alert: ${alert.name}`,
      description: alert.description,
      action: {
        label: 'View',
        onClick: () => {
          // Navigate to alert details
          router.push(`/alerts/${alert.id}`);
        },
      },
    });
  }
};
```

### SSE Integration for Logs

```typescript
// stores/monitoring/logsStore.ts
export const useLogsStore = create<LogsState>()(
  devtools((set, get) => ({
    logs: {},
    activeStreams: new Map(),
    
    startLogStream: (platformId: string, component: string) => {
      const key = `${platformId}:${component}`;
      const { activeStreams } = get();
      
      // Don't create duplicate streams
      if (activeStreams.has(key)) {
        return;
      }
      
      const eventSource = new EventSource(
        `/api/v1/platforms/${platformId}/components/${component}/logs/stream`
      );
      
      eventSource.onmessage = (event) => {
        const logEntry = JSON.parse(event.data);
        
        set((state) => {
          if (!state.logs[key]) {
            state.logs[key] = [];
          }
          
          state.logs[key].push(logEntry);
          
          // Keep only last 1000 logs
          if (state.logs[key].length > 1000) {
            state.logs[key] = state.logs[key].slice(-1000);
          }
        });
      };
      
      eventSource.onerror = () => {
        eventSource.close();
        activeStreams.delete(key);
        
        useNotificationStore.getState().addNotification({
          type: 'error',
          message: `Log stream disconnected for ${component}`,
        });
      };
      
      activeStreams.set(key, eventSource);
    },
    
    stopLogStream: (platformId: string, component: string) => {
      const key = `${platformId}:${component}`;
      const { activeStreams } = get();
      
      const eventSource = activeStreams.get(key);
      if (eventSource) {
        eventSource.close();
        activeStreams.delete(key);
      }
    },
  }))
);
```

---

## 💾 State Persistence Strategy

### Persistence Configuration

```typescript
// stores/persistence.ts
import { StateStorage } from 'zustand/middleware';

// Custom storage with encryption
const secureStorage: StateStorage = {
  getItem: async (name: string) => {
    const value = localStorage.getItem(name);
    if (!value) return null;
    
    try {
      // Decrypt if sensitive data
      if (name.includes('auth')) {
        return await decrypt(value);
      }
      return value;
    } catch {
      return null;
    }
  },
  
  setItem: async (name: string, value: string) => {
    // Encrypt sensitive data
    if (name.includes('auth')) {
      value = await encrypt(value);
    }
    localStorage.setItem(name, value);
  },
  
  removeItem: async (name: string) => {
    localStorage.removeItem(name);
  },
};

// Persistence configuration for different stores
export const persistConfigs = {
  auth: {
    name: 'gunj-auth-store',
    storage: secureStorage,
    partialize: (state: AuthState) => ({
      accessToken: state.accessToken,
      refreshToken: state.refreshToken,
    }),
  },
  
  ui: {
    name: 'gunj-ui-store',
    partialize: (state: UIState) => ({
      theme: state.theme,
      language: state.language,
      sidebarCollapsed: state.sidebarCollapsed,
    }),
  },
  
  platform: {
    name: 'gunj-platform-store',
    partialize: (state: PlatformState) => ({
      filters: state.filters,
      sortBy: state.sortBy,
    }),
  },
};
```

### Migration Strategy

```typescript
// stores/migrations.ts
export const migrations = {
  0: (state: any) => {
    // Initial version
    return state;
  },
  
  1: (state: any) => {
    // Migrate from v0 to v1
    return {
      ...state,
      // Add new fields with defaults
      preferences: state.preferences || {},
    };
  },
  
  2: (state: any) => {
    // Migrate from v1 to v2
    const { oldField, ...rest } = state;
    return {
      ...rest,
      newField: oldField || 'default',
    };
  },
};

// Apply migrations
export const migrate = (persistedState: any, version: number) => {
  let migratedState = persistedState;
  
  for (let i = version; i < Object.keys(migrations).length; i++) {
    migratedState = migrations[i](migratedState);
  }
  
  return migratedState;
};
```

---

## 🧪 Testing Strategy

### Store Testing

```typescript
// stores/__tests__/platformStore.test.ts
import { renderHook, act } from '@testing-library/react';
import { usePlatformStore } from '../platform/platformStore';

describe('PlatformStore', () => {
  beforeEach(() => {
    // Reset store before each test
    usePlatformStore.getState().reset();
  });
  
  it('should fetch platforms', async () => {
    const { result } = renderHook(() => usePlatformStore());
    
    // Mock API response
    jest.spyOn(platformAPI, 'list').mockResolvedValue(mockPlatforms);
    
    await act(async () => {
      await result.current.fetchPlatforms();
    });
    
    expect(result.current.platforms).toHaveLength(2);
    expect(result.current.isLoading).toBe(false);
  });
  
  it('should handle real-time updates', () => {
    const { result } = renderHook(() => usePlatformStore());
    
    act(() => {
      result.current.handlePlatformUpdate({
        type: 'created',
        platform: mockPlatform,
      });
    });
    
    expect(result.current.platforms[mockPlatform.id]).toEqual(mockPlatform);
  });
  
  it('should filter platforms correctly', () => {
    const { result } = renderHook(() => usePlatformStore());
    
    // Add test platforms
    act(() => {
      result.current.platforms = {
        '1': { ...mockPlatform, namespace: 'prod' },
        '2': { ...mockPlatform, id: '2', namespace: 'dev' },
      };
      
      result.current.setFilters({ namespace: 'prod' });
    });
    
    expect(result.current.filteredPlatforms).toHaveLength(1);
    expect(result.current.filteredPlatforms[0].namespace).toBe('prod');
  });
});
```

### Integration Testing

```typescript
// stores/__tests__/integration.test.ts
describe('Store Integration', () => {
  it('should clear all stores on logout', () => {
    const { result: authResult } = renderHook(() => useAuthStore());
    const { result: platformResult } = renderHook(() => usePlatformStore());
    
    // Setup initial state
    act(() => {
      platformResult.current.platforms = { '1': mockPlatform };
    });
    
    // Logout
    act(() => {
      authResult.current.logout();
    });
    
    // Verify all stores are cleared
    expect(authResult.current.user).toBeNull();
    expect(platformResult.current.platforms).toEqual({});
  });
});
```

---

## 🎯 Performance Optimization

### Selective Subscriptions

```typescript
// components/PlatformDetail.tsx
export const PlatformDetail: React.FC<{ platformId: string }> = ({ platformId }) => {
  // Subscribe only to specific platform updates
  const platform = usePlatformStore(
    useCallback(
      (state) => state.platforms[platformId],
      [platformId]
    )
  );
  
  // Subscribe to specific platform status
  const status = usePlatformStore(
    useCallback(
      (state) => state.platforms[platformId]?.status,
      [platformId]
    )
  );
  
  return (
    <div>
      {/* Component content */}
    </div>
  );
};
```

### Computed Values Memoization

```typescript
// stores/platform/selectors.ts
export const platformSelectors = {
  getFilteredPlatforms: (state: PlatformState) => {
    return createSelector(
      [
        (state) => state.platforms,
        (state) => state.filters,
        (state) => state.sortBy,
      ],
      (platforms, filters, sortBy) => {
        // Expensive filtering and sorting logic
        return filteredAndSortedPlatforms;
      }
    );
  },
};
```

---

## 📚 Best Practices

1. **Store Granularity**: Keep stores focused on single domains
2. **Action Naming**: Use descriptive verb-based action names
3. **State Shape**: Keep state normalized and flat
4. **Computed Values**: Use getters for derived state
5. **Subscriptions**: Subscribe to minimal state slices
6. **Error Handling**: Consistent error handling across stores
7. **Type Safety**: Full TypeScript coverage
8. **Testing**: Test stores in isolation and integration

---

## 🔄 Migration from Existing State

For users migrating from Redux or other state management:

1. **Gradual Migration**: Migrate one domain at a time
2. **Compatibility Layer**: Create adapters for existing components
3. **Data Migration**: Use persistence migrations
4. **Testing**: Comprehensive testing during migration

---

**This architecture provides a solid foundation for the Gunj Operator UI state management with excellent developer experience and performance.**