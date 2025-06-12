/**
 * useRealtimeStore.ts
 * Zustand store for real-time state management
 */

import { create } from 'zustand';
import { devtools, subscribeWithSelector } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';
import { ConnectionManager, ConnectionState } from '../services/realtime/ConnectionManager';
import { EventRouter, RealtimeEvent } from '../services/realtime/EventRouter';
import { AuthStore } from './auth';

export interface PlatformUpdate {
  id: string;
  name: string;
  namespace: string;
  phase: 'Pending' | 'Installing' | 'Ready' | 'Failed' | 'Upgrading';
  message?: string;
  lastUpdated: Date;
}

export interface ComponentHealth {
  platformId: string;
  component: 'prometheus' | 'grafana' | 'loki' | 'tempo';
  status: 'healthy' | 'degraded' | 'unhealthy';
  lastCheck: Date;
  details?: Record<string, any>;
}

export interface LogEntry {
  timestamp: Date;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  component?: string;
  metadata?: Record<string, any>;
}

export interface MetricData {
  timestamp: Date;
  metric: string;
  value: number;
  labels?: Record<string, string>;
}

export interface RealtimeState {
  // Connection state
  connectionStatus: ConnectionState;
  reconnectAttempts: number;
  lastHeartbeat: Date | null;
  
  // Platform updates
  platformUpdates: Map<string, PlatformUpdate>;
  componentHealth: Map<string, ComponentHealth>;
  
  // Streaming data
  logStreams: Map<string, LogEntry[]>;
  metricStreams: Map<string, MetricData[]>;
  activeAlerts: Map<string, any>;
  
  // Subscriptions
  subscriptions: Set<string>;
  
  // Actions
  connect: () => Promise<void>;
  disconnect: () => void;
  subscribeToUpdates: (platformId: string) => void;
  unsubscribeFromUpdates: (platformId: string) => void;
  subscribeToLogs: (platformId: string, component?: string) => void;
  unsubscribeFromLogs: (platformId: string) => void;
  subscribeToMetrics: (platformId: string, metrics: string[]) => void;
  unsubscribeFromMetrics: (platformId: string) => void;
  
  // Internal actions
  _updateConnectionStatus: (status: ConnectionState) => void;
  _handlePlatformUpdate: (update: PlatformUpdate) => void;
  _handleComponentHealth: (health: ComponentHealth) => void;
  _handleLogEntry: (platformId: string, entry: LogEntry) => void;
  _handleMetricData: (platformId: string, data: MetricData) => void;
  _handleAlert: (alert: any) => void;
  _clearOldData: () => void;
}

// Service instances (initialized separately)
let connectionManager: ConnectionManager;
let eventRouter: EventRouter;

// Configuration
const LOG_BUFFER_SIZE = 1000;
const METRIC_BUFFER_SIZE = 500;
const DATA_RETENTION_MS = 5 * 60 * 1000; // 5 minutes

export const useRealtimeStore = create<RealtimeState>()(
  devtools(
    subscribeWithSelector(
      immer((set, get) => ({
        // Initial state
        connectionStatus: 'disconnected',
        reconnectAttempts: 0,
        lastHeartbeat: null,
        platformUpdates: new Map(),
        componentHealth: new Map(),
        logStreams: new Map(),
        metricStreams: new Map(),
        activeAlerts: new Map(),
        subscriptions: new Set(),

        // Connect to real-time services
        connect: async () => {
          const state = get();
          if (state.connectionStatus === 'connected' || state.connectionStatus === 'connecting') {
            return;
          }

          set((state) => {
            state.connectionStatus = 'connecting';
          });

          try {
            // Initialize services if not already done
            if (!connectionManager || !eventRouter) {
              initializeServices();
            }

            // Setup event handlers
            setupEventHandlers(set, get);

            // Connect WebSocket
            await connectionManager.connectWebSocket();

            // Start data cleanup timer
            startDataCleanup(set);
          } catch (error) {
            console.error('Failed to connect:', error);
            set((state) => {
              state.connectionStatus = 'error';
            });
          }
        },

        // Disconnect from real-time services
        disconnect: () => {
          if (connectionManager) {
            connectionManager.disconnect();
          }

          set((state) => {
            state.connectionStatus = 'disconnected';
            state.subscriptions.clear();
          });
        },

        // Subscribe to platform updates
        subscribeToUpdates: (platformId: string) => {
          const unsubscribe = eventRouter.subscribe(
            `platform.${platformId}.**`,
            (data) => {
              const { _handlePlatformUpdate, _handleComponentHealth } = get();
              
              if (data.type === 'status') {
                _handlePlatformUpdate({
                  id: platformId,
                  ...data.payload,
                  lastUpdated: new Date(),
                });
              } else if (data.type === 'health') {
                _handleComponentHealth({
                  platformId,
                  ...data.payload,
                  lastCheck: new Date(),
                });
              }
            }
          );

          set((state) => {
            state.subscriptions.add(`platform.${platformId}`);
          });

          // Request current status
          connectionManager.sendMessage({
            type: 'subscribe',
            target: 'platform',
            id: platformId,
            events: ['status', 'health'],
          });
        },

        // Unsubscribe from platform updates
        unsubscribeFromUpdates: (platformId: string) => {
          connectionManager.sendMessage({
            type: 'unsubscribe',
            target: 'platform',
            id: platformId,
          });

          set((state) => {
            state.subscriptions.delete(`platform.${platformId}`);
            state.platformUpdates.delete(platformId);
            
            // Remove component health for this platform
            Array.from(state.componentHealth.keys()).forEach((key) => {
              if (key.startsWith(platformId)) {
                state.componentHealth.delete(key);
              }
            });
          });
        },

        // Subscribe to logs
        subscribeToLogs: (platformId: string, component?: string) => {
          const streamKey = component ? `${platformId}-${component}` : platformId;
          
          // Connect to SSE for log streaming
          const endpoint = `/platforms/${platformId}/logs${component ? `/${component}` : ''}`;
          connectionManager.connectSSE(`logs-${streamKey}`, endpoint);

          set((state) => {
            state.subscriptions.add(`logs.${streamKey}`);
            if (!state.logStreams.has(streamKey)) {
              state.logStreams.set(streamKey, []);
            }
          });
        },

        // Unsubscribe from logs
        unsubscribeFromLogs: (platformId: string) => {
          set((state) => {
            state.subscriptions.delete(`logs.${platformId}`);
            state.logStreams.delete(platformId);
          });
        },

        // Subscribe to metrics
        subscribeToMetrics: (platformId: string, metrics: string[]) => {
          const endpoint = `/platforms/${platformId}/metrics`;
          connectionManager.connectSSE(`metrics-${platformId}`, endpoint);

          connectionManager.sendMessage({
            type: 'subscribe',
            target: 'metrics',
            platformId,
            metrics,
          });

          set((state) => {
            state.subscriptions.add(`metrics.${platformId}`);
            if (!state.metricStreams.has(platformId)) {
              state.metricStreams.set(platformId, []);
            }
          });
        },

        // Unsubscribe from metrics
        unsubscribeFromMetrics: (platformId: string) => {
          connectionManager.sendMessage({
            type: 'unsubscribe',
            target: 'metrics',
            platformId,
          });

          set((state) => {
            state.subscriptions.delete(`metrics.${platformId}`);
            state.metricStreams.delete(platformId);
          });
        },

        // Internal actions
        _updateConnectionStatus: (status: ConnectionState) => {
          set((state) => {
            state.connectionStatus = status;
          });
        },

        _handlePlatformUpdate: (update: PlatformUpdate) => {
          set((state) => {
            state.platformUpdates.set(update.id, update);
          });
        },

        _handleComponentHealth: (health: ComponentHealth) => {
          const key = `${health.platformId}-${health.component}`;
          set((state) => {
            state.componentHealth.set(key, health);
          });
        },

        _handleLogEntry: (platformId: string, entry: LogEntry) => {
          set((state) => {
            const logs = state.logStreams.get(platformId) || [];
            logs.push(entry);
            
            // Maintain buffer size
            if (logs.length > LOG_BUFFER_SIZE) {
              logs.shift();
            }
            
            state.logStreams.set(platformId, logs);
          });
        },

        _handleMetricData: (platformId: string, data: MetricData) => {
          set((state) => {
            const metrics = state.metricStreams.get(platformId) || [];
            metrics.push(data);
            
            // Maintain buffer size
            if (metrics.length > METRIC_BUFFER_SIZE) {
              metrics.shift();
            }
            
            state.metricStreams.set(platformId, metrics);
          });
        },

        _handleAlert: (alert: any) => {
          set((state) => {
            if (alert.status === 'resolved') {
              state.activeAlerts.delete(alert.fingerprint);
            } else {
              state.activeAlerts.set(alert.fingerprint, alert);
            }
          });
        },

        _clearOldData: () => {
          const now = Date.now();
          
          set((state) => {
            // Clear old logs
            state.logStreams.forEach((logs, key) => {
              const filtered = logs.filter(
                (log) => now - log.timestamp.getTime() < DATA_RETENTION_MS
              );
              
              if (filtered.length !== logs.length) {
                state.logStreams.set(key, filtered);
              }
            });

            // Clear old metrics
            state.metricStreams.forEach((metrics, key) => {
              const filtered = metrics.filter(
                (metric) => now - metric.timestamp.getTime() < DATA_RETENTION_MS
              );
              
              if (filtered.length !== metrics.length) {
                state.metricStreams.set(key, filtered);
              }
            });
          });
        },
      }))
    ),
    {
      name: 'realtime-store',
    }
  )
);

// Initialize services
function initializeServices() {
  const authStore = AuthStore.getInstance();
  
  connectionManager = new ConnectionManager(
    {
      wsUrl: import.meta.env.VITE_WS_URL || 'wss://api.gunj-operator.com/ws',
      sseUrl: import.meta.env.VITE_SSE_URL || 'https://api.gunj-operator.com/sse',
      reconnectInterval: 5000,
      maxReconnectAttempts: 10,
      heartbeatInterval: 30000,
    },
    authStore
  );

  eventRouter = new EventRouter();
}

// Setup event handlers
function setupEventHandlers(set: any, get: any) {
  // Connection status updates
  connectionManager.on('connectionStateChanged', (state: ConnectionState) => {
    get()._updateConnectionStatus(state);
  });

  // Handle incoming messages
  connectionManager.on('message', ({ type, data, streamType }: any) => {
    if (type === 'websocket') {
      eventRouter.routeEvent({
        id: data.id || uuidv4(),
        type: data.type,
        data: data.payload,
        timestamp: new Date(data.timestamp || Date.now()),
        source: 'websocket',
        metadata: data.metadata,
      });
    } else if (type === 'sse') {
      handleSSEMessage(streamType, data, get);
    }
  });

  // Setup global event subscriptions
  eventRouter.subscribe('alert.**', (data) => {
    get()._handleAlert(data);
  });

  eventRouter.subscribe('heartbeat', () => {
    set((state: RealtimeState) => {
      state.lastHeartbeat = new Date();
    });
  });
}

// Handle SSE messages
function handleSSEMessage(streamType: string, data: any, get: any) {
  if (streamType.startsWith('logs-')) {
    const platformId = streamType.substring(5);
    get()._handleLogEntry(platformId, {
      timestamp: new Date(data.timestamp),
      level: data.level,
      message: data.message,
      component: data.component,
      metadata: data.metadata,
    });
  } else if (streamType.startsWith('metrics-')) {
    const platformId = streamType.substring(8);
    get()._handleMetricData(platformId, {
      timestamp: new Date(data.timestamp),
      metric: data.metric,
      value: data.value,
      labels: data.labels,
    });
  }
}

// Start periodic data cleanup
function startDataCleanup(set: any) {
  setInterval(() => {
    set((state: RealtimeState) => {
      state._clearOldData();
    });
  }, 60000); // Run every minute
}

// Helper to generate UUIDs
function uuidv4(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

// Export service getters for direct access if needed
export const getConnectionManager = () => connectionManager;
export const getEventRouter = () => eventRouter;
