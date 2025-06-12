/**
 * ConnectionManager.ts
 * Manages WebSocket and SSE connections for real-time updates
 */

import { EventEmitter } from 'events';
import { AuthStore } from '../../stores/auth';

export interface ConnectionConfig {
  wsUrl: string;
  sseUrl: string;
  reconnectInterval: number;
  maxReconnectAttempts: number;
  heartbeatInterval: number;
  authToken?: string;
}

export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error';

export interface ConnectionStatus {
  websocket: ConnectionState;
  sse: Map<string, ConnectionState>;
  overall: ConnectionState;
}

export class ConnectionManager extends EventEmitter {
  private wsConnection: WebSocket | null = null;
  private sseConnections: Map<string, EventSource> = new Map();
  private reconnectTimers: Map<string, NodeJS.Timeout> = new Map();
  private config: ConnectionConfig;
  private connectionState: ConnectionState = 'disconnected';
  private reconnectAttempts: number = 0;
  private heartbeatTimer?: NodeJS.Timeout;
  private authStore: AuthStore;

  constructor(config: ConnectionConfig, authStore: AuthStore) {
    super();
    this.config = config;
    this.authStore = authStore;
    this.setupConnectionMonitoring();
  }

  /**
   * Setup connection monitoring and auto-reconnection
   */
  private setupConnectionMonitoring(): void {
    // Monitor online/offline status
    window.addEventListener('online', () => this.handleOnline());
    window.addEventListener('offline', () => this.handleOffline());
  }

  /**
   * Connect WebSocket with authentication
   */
  async connectWebSocket(): Promise<void> {
    if (this.wsConnection?.readyState === WebSocket.OPEN) {
      return;
    }

    this.connectionState = 'connecting';
    this.emit('connectionStateChanged', this.connectionState);

    try {
      const token = await this.authStore.getAccessToken();
      const wsUrl = `${this.config.wsUrl}?token=${encodeURIComponent(token)}`;

      this.wsConnection = new WebSocket(wsUrl);
      this.setupWebSocketHandlers();
    } catch (error) {
      console.error('WebSocket connection error:', error);
      this.connectionState = 'error';
      this.emit('connectionStateChanged', this.connectionState);
      this.scheduleReconnect('websocket');
    }
  }

  /**
   * Setup WebSocket event handlers
   */
  private setupWebSocketHandlers(): void {
    if (!this.wsConnection) return;

    this.wsConnection.onopen = () => {
      console.log('WebSocket connected');
      this.connectionState = 'connected';
      this.reconnectAttempts = 0;
      this.emit('connectionStateChanged', this.connectionState);
      this.startHeartbeat();
    };

    this.wsConnection.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        this.emit('message', { type: 'websocket', data: message });
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };

    this.wsConnection.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.connectionState = 'error';
      this.emit('connectionStateChanged', this.connectionState);
    };

    this.wsConnection.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason);
      this.connectionState = 'disconnected';
      this.emit('connectionStateChanged', this.connectionState);
      this.stopHeartbeat();
      
      if (!event.wasClean) {
        this.scheduleReconnect('websocket');
      }
    };
  }

  /**
   * Connect to Server-Sent Events stream
   */
  connectSSE(streamType: string, endpoint: string): EventSource {
    // Close existing connection if any
    const existing = this.sseConnections.get(streamType);
    if (existing) {
      existing.close();
    }

    const token = this.authStore.getAccessTokenSync();
    const url = `${this.config.sseUrl}${endpoint}`;
    
    const eventSource = new EventSource(url, {
      withCredentials: true,
    });

    // Add auth token via post-connection message if needed
    eventSource.onopen = () => {
      console.log(`SSE connected: ${streamType}`);
      this.emit('sseConnectionStateChanged', { streamType, state: 'connected' });
    };

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.emit('message', { type: 'sse', streamType, data });
      } catch (error) {
        console.error(`Error parsing SSE message for ${streamType}:`, error);
      }
    };

    eventSource.onerror = (error) => {
      console.error(`SSE error for ${streamType}:`, error);
      this.emit('sseConnectionStateChanged', { streamType, state: 'error' });
      this.scheduleReconnect(`sse-${streamType}`);
    };

    this.sseConnections.set(streamType, eventSource);
    return eventSource;
  }

  /**
   * Send message via WebSocket
   */
  sendMessage(message: any): void {
    if (this.wsConnection?.readyState === WebSocket.OPEN) {
      this.wsConnection.send(JSON.stringify(message));
    } else {
      throw new Error('WebSocket is not connected');
    }
  }

  /**
   * Schedule reconnection attempt
   */
  private scheduleReconnect(connectionType: string): void {
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      console.error(`Max reconnection attempts reached for ${connectionType}`);
      return;
    }

    const delay = Math.min(
      this.config.reconnectInterval * Math.pow(2, this.reconnectAttempts),
      30000 // Max 30 seconds
    );

    console.log(`Scheduling reconnect for ${connectionType} in ${delay}ms`);

    const timer = setTimeout(() => {
      this.reconnectAttempts++;
      
      if (connectionType === 'websocket') {
        this.connectWebSocket();
      } else if (connectionType.startsWith('sse-')) {
        const streamType = connectionType.substring(4);
        const endpoint = this.getSSEEndpoint(streamType);
        if (endpoint) {
          this.connectSSE(streamType, endpoint);
        }
      }
    }, delay);

    this.reconnectTimers.set(connectionType, timer);
  }

  /**
   * Start heartbeat to keep connection alive
   */
  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      if (this.wsConnection?.readyState === WebSocket.OPEN) {
        this.sendMessage({ type: 'ping' });
      }
    }, this.config.heartbeatInterval);
  }

  /**
   * Stop heartbeat
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = undefined;
    }
  }

  /**
   * Handle online event
   */
  private handleOnline(): void {
    console.log('Connection online, attempting to reconnect...');
    this.connectWebSocket();
    
    // Reconnect all SSE streams
    this.sseConnections.forEach((_, streamType) => {
      const endpoint = this.getSSEEndpoint(streamType);
      if (endpoint) {
        this.connectSSE(streamType, endpoint);
      }
    });
  }

  /**
   * Handle offline event
   */
  private handleOffline(): void {
    console.log('Connection offline');
    this.connectionState = 'disconnected';
    this.emit('connectionStateChanged', this.connectionState);
  }

  /**
   * Get SSE endpoint for stream type
   */
  private getSSEEndpoint(streamType: string): string | null {
    const endpoints: Record<string, string> = {
      logs: '/logs',
      metrics: '/metrics',
      events: '/events',
      alerts: '/alerts',
    };
    
    return endpoints[streamType] || null;
  }

  /**
   * Get current connection status
   */
  getConnectionStatus(): ConnectionStatus {
    const sseStates = new Map<string, ConnectionState>();
    
    this.sseConnections.forEach((eventSource, streamType) => {
      const state = eventSource.readyState === EventSource.OPEN
        ? 'connected'
        : eventSource.readyState === EventSource.CONNECTING
        ? 'connecting'
        : 'disconnected';
      
      sseStates.set(streamType, state);
    });

    return {
      websocket: this.connectionState,
      sse: sseStates,
      overall: this.determineOverallState(this.connectionState, sseStates),
    };
  }

  /**
   * Determine overall connection state
   */
  private determineOverallState(
    wsState: ConnectionState,
    sseStates: Map<string, ConnectionState>
  ): ConnectionState {
    if (wsState === 'error' || Array.from(sseStates.values()).includes('error')) {
      return 'error';
    }
    
    if (wsState === 'connected' || Array.from(sseStates.values()).includes('connected')) {
      return 'connected';
    }
    
    if (wsState === 'connecting' || Array.from(sseStates.values()).includes('connecting')) {
      return 'connecting';
    }
    
    return 'disconnected';
  }

  /**
   * Disconnect all connections
   */
  disconnect(): void {
    // Close WebSocket
    if (this.wsConnection) {
      this.wsConnection.close();
      this.wsConnection = null;
    }

    // Close all SSE connections
    this.sseConnections.forEach((eventSource) => {
      eventSource.close();
    });
    this.sseConnections.clear();

    // Clear all timers
    this.reconnectTimers.forEach((timer) => {
      clearTimeout(timer);
    });
    this.reconnectTimers.clear();

    this.stopHeartbeat();
    this.connectionState = 'disconnected';
    this.emit('connectionStateChanged', this.connectionState);
  }
}
