import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';
import type { User, LoginCredentials } from '@/types';
import { authAPI } from '@/api/auth';
import { useWebSocketStore } from '../realtime/websocketStore';
import { usePlatformStore } from '../platform/platformStore';
import { useMetricsStore } from '../monitoring/metricsStore';

interface AuthState {
  // State
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  accessToken: string | null;
  refreshToken: string | null;
  tokenExpiresAt: number | null;
  
  // Actions
  login: (credentials: LoginCredentials) => Promise<void>;
  loginWithOIDC: (provider: string) => void;
  logout: () => void;
  refreshTokens: () => Promise<void>;
  updateUser: (updates: Partial<User>) => void;
  checkPermission: (permission: string) => boolean;
  hasRole: (role: string) => boolean;
  clearError: () => void;
  
  // Internal actions
  _setTokens: (accessToken: string, refreshToken: string, expiresIn: number) => void;
  _clearAuth: () => void;
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
        tokenExpiresAt: null,

        // Login with credentials
        login: async (credentials) => {
          set((state) => {
            state.isLoading = true;
            state.error = null;
          });

          try {
            const response = await authAPI.login(credentials);
            
            set((state) => {
              state.user = response.user;
              state.isAuthenticated = true;
              state.isLoading = false;
            });

            // Set tokens
            get()._setTokens(
              response.accessToken,
              response.refreshToken,
              response.expiresIn
            );

            // Initialize WebSocket connection
            useWebSocketStore.getState().connect(response.accessToken);
            
            // Fetch initial data
            await Promise.all([
              usePlatformStore.getState().fetchPlatforms(),
              // Add other initial data fetches here
            ]);
          } catch (error: any) {
            set((state) => {
              state.error = error.message || 'Login failed';
              state.isLoading = false;
            });
            throw error;
          }
        },

        // Login with OIDC provider
        loginWithOIDC: (provider) => {
          // Redirect to OIDC provider
          window.location.href = `/api/auth/oidc/${provider}`;
        },

        // Logout
        logout: () => {
          const { refreshToken } = get();
          
          // Call logout API if we have a refresh token
          if (refreshToken) {
            authAPI.logout(refreshToken).catch(() => {
              // Ignore logout errors
            });
          }

          // Clear local auth state
          get()._clearAuth();

          // Disconnect WebSocket
          useWebSocketStore.getState().disconnect();
          
          // Clear all stores
          usePlatformStore.getState().reset();
          useMetricsStore.getState().reset();
          
          // Redirect to login
          window.location.href = '/login';
        },

        // Refresh tokens
        refreshTokens: async () => {
          const { refreshToken } = get();
          
          if (!refreshToken) {
            get()._clearAuth();
            throw new Error('No refresh token available');
          }

          try {
            const response = await authAPI.refreshTokens(refreshToken);
            
            get()._setTokens(
              response.accessToken,
              response.refreshToken,
              response.expiresIn
            );
            
            // Reconnect WebSocket with new token
            useWebSocketStore.getState().connect(response.accessToken);
          } catch (error) {
            get()._clearAuth();
            throw error;
          }
        },

        // Update user profile
        updateUser: (updates) => {
          set((state) => {
            if (state.user) {
              Object.assign(state.user, updates);
            }
          });
        },

        // Check permission
        checkPermission: (permission) => {
          const user = get().user;
          if (!user) return false;
          
          // Check direct permissions
          if (user.permissions.includes(permission)) return true;
          
          // Check wildcard permissions
          const permissionParts = permission.split(':');
          for (let i = permissionParts.length; i > 0; i--) {
            const wildcardPermission = permissionParts.slice(0, i).join(':') + ':*';
            if (user.permissions.includes(wildcardPermission)) return true;
          }
          
          // Check admin permission
          return user.permissions.includes('*');
        },

        // Check role
        hasRole: (role) => {
          const user = get().user;
          return user?.roles.includes(role) ?? false;
        },

        // Clear error
        clearError: () => {
          set((state) => {
            state.error = null;
          });
        },

        // Internal: Set tokens
        _setTokens: (accessToken, refreshToken, expiresIn) => {
          set((state) => {
            state.accessToken = accessToken;
            state.refreshToken = refreshToken;
            state.tokenExpiresAt = Date.now() + expiresIn * 1000;
          });
          
          // Set up token refresh timer
          const refreshTime = expiresIn * 1000 * 0.9; // Refresh at 90% of expiry
          setTimeout(() => {
            get().refreshTokens().catch(() => {
              get().logout();
            });
          }, refreshTime);
        },

        // Internal: Clear auth state
        _clearAuth: () => {
          set((state) => {
            state.user = null;
            state.isAuthenticated = false;
            state.accessToken = null;
            state.refreshToken = null;
            state.tokenExpiresAt = null;
            state.error = null;
          });
        },
      })),
      {
        name: 'gunj-auth-store',
        partialize: (state) => ({
          accessToken: state.accessToken,
          refreshToken: state.refreshToken,
          tokenExpiresAt: state.tokenExpiresAt,
        }),
        onRehydrateStorage: () => (state) => {
          // Check if tokens are still valid after rehydration
          if (state && state.tokenExpiresAt && state.tokenExpiresAt < Date.now()) {
            state._clearAuth();
          } else if (state && state.accessToken) {
            // Fetch user data if we have a valid token
            authAPI.getCurrentUser().then((user) => {
              state.user = user;
              state.isAuthenticated = true;
              
              // Reconnect WebSocket
              useWebSocketStore.getState().connect(state.accessToken!);
            }).catch(() => {
              state._clearAuth();
            });
          }
        },
      }
    ),
    {
      name: 'auth-store',
    }
  )
);

// Axios interceptor for adding auth token
if (typeof window !== 'undefined') {
  import('@/lib/axios').then(({ axios }) => {
    axios.interceptors.request.use(
      (config) => {
        const token = useAuthStore.getState().accessToken;
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    axios.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;
        
        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;
          
          try {
            await useAuthStore.getState().refreshTokens();
            return axios(originalRequest);
          } catch (refreshError) {
            useAuthStore.getState().logout();
            return Promise.reject(refreshError);
          }
        }
        
        return Promise.reject(error);
      }
    );
  });
}