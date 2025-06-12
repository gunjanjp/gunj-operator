/**
 * realtime.config.ts
 * Configuration for real-time update system
 */

export const realtimeConfig = {
  // WebSocket configuration
  websocket: {
    url: import.meta.env.VITE_WS_URL || 'wss://api.gunj-operator.com/ws',
    reconnectInterval: 5000,
    maxReconnectAttempts: 10,
    heartbeatInterval: 30000,
    messageTimeout: 10000,
    protocols: ['v1.gunj.operator'],
  },
  
  // Server-Sent Events configuration
  sse: {
    baseUrl: import.meta.env.VITE_SSE_URL || 'https://api.gunj-operator.com/sse',
    reconnectDelay: 3000,
    maxEventListeners: 100,
    withCredentials: true,
  },
  
  // Performance configuration
  performance: {
    maxMessagesPerSecond: 100,
    bufferSize: 10000,
    dropPolicy: 'oldest' as const,
    enableBackpressure: true,
    adaptiveRateLimit: true,
    performanceCheckInterval: 5000,
  },
  
  // Feature flags
  features: {
    enableOfflineQueue: true,
    enableCompression: true,
    enableBatching: true,
    batchInterval: 100, // ms
    enableMetrics: true,
    enableDebugLogs: import.meta.env.DEV,
  },
  
  // Data retention
  retention: {
    logs: 5 * 60 * 1000, // 5 minutes
    metrics: 10 * 60 * 1000, // 10 minutes
    events: 60 * 60 * 1000, // 1 hour
    alerts: 24 * 60 * 60 * 1000, // 24 hours
  },
  
  // Buffer sizes
  buffers: {
    logs: 1000,
    metrics: 500,
    events: 200,
    alerts: 100,
  },
  
  // Subscription limits
  limits: {
    maxPlatformSubscriptions: 10,
    maxLogStreams: 5,
    maxMetricStreams: 20,
    maxConcurrentConnections: 3,
  },
  
  // Fallback configuration
  fallback: {
    enablePolling: true,
    pollingInterval: 30000, // 30 seconds
    maxPollingRetries: 3,
  },
  
  // Security
  security: {
    enableEncryption: true,
    validateMessages: true,
    maxMessageSize: 1024 * 1024, // 1MB
    allowedOrigins: [
      'https://gunj-operator.com',
      'https://api.gunj-operator.com',
    ],
  },
};

// Development overrides
if (import.meta.env.DEV) {
  realtimeConfig.websocket.url = 'ws://localhost:8080/ws';
  realtimeConfig.sse.baseUrl = 'http://localhost:8080/sse';
  realtimeConfig.security.allowedOrigins.push('http://localhost:3000');
}

// Export typed config
export type RealtimeConfig = typeof realtimeConfig;
