/**
 * useRealtime.ts
 * React hook for real-time updates in components
 */

import { useEffect, useRef, useCallback } from 'react';
import { useRealtimeStore } from '../stores/useRealtimeStore';
import { shallow } from 'zustand/shallow';

export interface UseRealtimeOptions {
  autoConnect?: boolean;
  reconnectOnMount?: boolean;
}

export interface UseRealtimePlatformOptions {
  platformId: string;
  subscribeLogs?: boolean;
  subscribeMetrics?: boolean;
  metricsToTrack?: string[];
}

/**
 * Hook for managing real-time connection
 */
export function useRealtime(options: UseRealtimeOptions = {}) {
  const { autoConnect = true, reconnectOnMount = true } = options;

  const {
    connectionStatus,
    connect,
    disconnect,
    lastHeartbeat,
    reconnectAttempts,
  } = useRealtimeStore(
    (state) => ({
      connectionStatus: state.connectionStatus,
      connect: state.connect,
      disconnect: state.disconnect,
      lastHeartbeat: state.lastHeartbeat,
      reconnectAttempts: state.reconnectAttempts,
    }),
    shallow
  );

  // Auto-connect on mount
  useEffect(() => {
    if (autoConnect && connectionStatus === 'disconnected') {
      connect();
    }

    return () => {
      if (reconnectOnMount) {
        // Keep connection alive when component unmounts
      }
    };
  }, [autoConnect, reconnectOnMount, connectionStatus, connect]);

  // Calculate connection health
  const isHealthy = useCallback(() => {
    if (connectionStatus !== 'connected') return false;
    if (!lastHeartbeat) return false;
    
    const heartbeatAge = Date.now() - lastHeartbeat.getTime();
    return heartbeatAge < 60000; // Less than 1 minute old
  }, [connectionStatus, lastHeartbeat]);

  return {
    connectionStatus,
    isConnected: connectionStatus === 'connected',
    isHealthy: isHealthy(),
    reconnectAttempts,
    lastHeartbeat,
    connect,
    disconnect,
  };
}

/**
 * Hook for subscribing to platform updates
 */
export function useRealtimePlatform(options: UseRealtimePlatformOptions) {
  const { platformId, subscribeLogs = false, subscribeMetrics = false, metricsToTrack = [] } = options;
  
  const isMountedRef = useRef(true);

  const {
    platformUpdate,
    componentHealth,
    logs,
    metrics,
    subscribeToUpdates,
    unsubscribeFromUpdates,
    subscribeToLogs,
    unsubscribeFromLogs,
    subscribeToMetrics,
    unsubscribeFromMetrics,
  } = useRealtimeStore(
    (state) => ({
      platformUpdate: state.platformUpdates.get(platformId),
      componentHealth: Array.from(state.componentHealth.entries())
        .filter(([key]) => key.startsWith(platformId))
        .map(([, health]) => health),
      logs: state.logStreams.get(platformId) || [],
      metrics: state.metricStreams.get(platformId) || [],
      subscribeToUpdates: state.subscribeToUpdates,
      unsubscribeFromUpdates: state.unsubscribeFromUpdates,
      subscribeToLogs: state.subscribeToLogs,
      unsubscribeFromLogs: state.unsubscribeFromLogs,
      subscribeToMetrics: state.subscribeToMetrics,
      unsubscribeFromMetrics: state.unsubscribeFromMetrics,
    }),
    shallow
  );

  // Subscribe to platform updates
  useEffect(() => {
    if (!platformId) return;

    subscribeToUpdates(platformId);

    return () => {
      if (isMountedRef.current) {
        unsubscribeFromUpdates(platformId);
      }
    };
  }, [platformId, subscribeToUpdates, unsubscribeFromUpdates]);

  // Subscribe to logs if requested
  useEffect(() => {
    if (!platformId || !subscribeLogs) return;

    subscribeToLogs(platformId);

    return () => {
      if (isMountedRef.current) {
        unsubscribeFromLogs(platformId);
      }
    };
  }, [platformId, subscribeLogs, subscribeToLogs, unsubscribeFromLogs]);

  // Subscribe to metrics if requested
  useEffect(() => {
    if (!platformId || !subscribeMetrics || metricsToTrack.length === 0) return;

    subscribeToMetrics(platformId, metricsToTrack);

    return () => {
      if (isMountedRef.current) {
        unsubscribeFromMetrics(platformId);
      }
    };
  }, [platformId, subscribeMetrics, metricsToTrack, subscribeToMetrics, unsubscribeFromMetrics]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  return {
    platform: platformUpdate,
    componentHealth,
    logs,
    metrics,
    isReady: !!platformUpdate,
  };
}

/**
 * Hook for subscribing to alerts
 */
export function useRealtimeAlerts() {
  const activeAlerts = useRealtimeStore((state) => 
    Array.from(state.activeAlerts.values())
  );

  const criticalAlerts = activeAlerts.filter(alert => alert.severity === 'critical');
  const warningAlerts = activeAlerts.filter(alert => alert.severity === 'warning');

  return {
    activeAlerts,
    criticalAlerts,
    warningAlerts,
    hasAlerts: activeAlerts.length > 0,
    hasCritical: criticalAlerts.length > 0,
  };
}

/**
 * Hook for log streaming with filters
 */
export function useRealtimeLogs(platformId: string, filters?: {
  level?: string[];
  component?: string[];
  search?: string;
}) {
  const logs = useRealtimeStore(
    (state) => state.logStreams.get(platformId) || []
  );

  // Apply filters
  const filteredLogs = logs.filter(log => {
    if (filters?.level && !filters.level.includes(log.level)) {
      return false;
    }
    
    if (filters?.component && log.component && !filters.component.includes(log.component)) {
      return false;
    }
    
    if (filters?.search && !log.message.toLowerCase().includes(filters.search.toLowerCase())) {
      return false;
    }
    
    return true;
  });

  // Subscribe to logs
  useEffect(() => {
    const { subscribeToLogs, unsubscribeFromLogs } = useRealtimeStore.getState();
    
    subscribeToLogs(platformId);
    
    return () => {
      unsubscribeFromLogs(platformId);
    };
  }, [platformId]);

  return {
    logs: filteredLogs,
    totalLogs: logs.length,
    filteredCount: filteredLogs.length,
  };
}

/**
 * Hook for metric streaming with aggregation
 */
export function useRealtimeMetrics(
  platformId: string,
  metricName: string,
  options?: {
    aggregation?: 'avg' | 'sum' | 'min' | 'max' | 'last';
    window?: number; // seconds
  }
) {
  const { aggregation = 'last', window = 60 } = options || {};
  
  const metrics = useRealtimeStore(
    (state) => state.metricStreams.get(platformId) || []
  );

  // Filter metrics by name and time window
  const now = Date.now();
  const windowMs = window * 1000;
  
  const relevantMetrics = metrics.filter(
    metric => 
      metric.metric === metricName &&
      now - metric.timestamp.getTime() <= windowMs
  );

  // Calculate aggregated value
  const aggregatedValue = useCallback(() => {
    if (relevantMetrics.length === 0) return null;

    const values = relevantMetrics.map(m => m.value);

    switch (aggregation) {
      case 'avg':
        return values.reduce((sum, val) => sum + val, 0) / values.length;
      case 'sum':
        return values.reduce((sum, val) => sum + val, 0);
      case 'min':
        return Math.min(...values);
      case 'max':
        return Math.max(...values);
      case 'last':
      default:
        return values[values.length - 1];
    }
  }, [relevantMetrics, aggregation]);

  // Subscribe to metrics
  useEffect(() => {
    const { subscribeToMetrics, unsubscribeFromMetrics } = useRealtimeStore.getState();
    
    subscribeToMetrics(platformId, [metricName]);
    
    return () => {
      unsubscribeFromMetrics(platformId);
    };
  }, [platformId, metricName]);

  return {
    value: aggregatedValue(),
    dataPoints: relevantMetrics,
    lastUpdated: relevantMetrics.length > 0 
      ? relevantMetrics[relevantMetrics.length - 1].timestamp 
      : null,
  };
}
