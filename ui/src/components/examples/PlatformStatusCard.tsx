/**
 * PlatformStatusCard.tsx
 * Example component using real-time updates
 */

import React, { useState } from 'react';
import { useRealtimePlatform, useRealtime } from '../../hooks/useRealtime';
import './PlatformStatusCard.css';

interface PlatformStatusCardProps {
  platformId: string;
  platformName: string;
  namespace: string;
}

export const PlatformStatusCard: React.FC<PlatformStatusCardProps> = ({
  platformId,
  platformName,
  namespace,
}) => {
  const [showLogs, setShowLogs] = useState(false);
  
  // Connect to real-time updates
  const { isConnected, connectionStatus } = useRealtime();
  
  // Subscribe to platform updates
  const { platform, componentHealth, logs, isReady } = useRealtimePlatform({
    platformId,
    subscribeLogs: showLogs,
    subscribeMetrics: true,
    metricsToTrack: ['cpu_usage', 'memory_usage'],
  });

  // Render connection indicator
  const renderConnectionStatus = () => {
    const statusColors = {
      connected: '#4caf50',
      connecting: '#ff9800',
      disconnected: '#f44336',
      error: '#f44336',
    };

    return (
      <div className="connection-indicator">
        <span 
          className="connection-dot"
          style={{ backgroundColor: statusColors[connectionStatus] }}
        />
        <span className="connection-text">{connectionStatus}</span>
      </div>
    );
  };

  // Render platform phase badge
  const renderPhaseBadge = () => {
    if (!platform) return null;

    const phaseColors = {
      Ready: '#4caf50',
      Installing: '#2196f3',
      Upgrading: '#ff9800',
      Failed: '#f44336',
      Pending: '#9e9e9e',
    };

    return (
      <span 
        className="phase-badge"
        style={{ backgroundColor: phaseColors[platform.phase] }}
      >
        {platform.phase}
      </span>
    );
  };

  // Render component health indicators
  const renderComponentHealth = () => {
    const components = ['prometheus', 'grafana', 'loki', 'tempo'];
    
    return (
      <div className="component-health">
        <h4>Component Health</h4>
        <div className="health-grid">
          {components.map(component => {
            const health = componentHealth.find(h => h.component === component);
            const statusColor = health?.status === 'healthy' ? '#4caf50' : 
                              health?.status === 'degraded' ? '#ff9800' : '#f44336';
            
            return (
              <div key={component} className="health-item">
                <span className="component-name">{component}</span>
                <span 
                  className="health-indicator"
                  style={{ backgroundColor: statusColor }}
                  title={health?.status || 'unknown'}
                />
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  // Render real-time logs
  const renderLogs = () => {
    if (!showLogs) return null;

    return (
      <div className="logs-section">
        <h4>Real-time Logs</h4>
        <div className="logs-container">
          {logs.slice(-10).map((log, index) => (
            <div key={index} className={`log-entry log-${log.level}`}>
              <span className="log-time">
                {log.timestamp.toLocaleTimeString()}
              </span>
              <span className="log-level">[{log.level}]</span>
              <span className="log-message">{log.message}</span>
            </div>
          ))}
        </div>
      </div>
    );
  };

  // Loading state
  if (!isReady) {
    return (
      <div className="platform-card loading">
        <div className="card-header">
          <h3>{platformName}</h3>
          <span className="namespace">{namespace}</span>
        </div>
        <div className="loading-spinner">Connecting...</div>
      </div>
    );
  }

  return (
    <div className="platform-card">
      <div className="card-header">
        <div className="header-left">
          <h3>{platformName}</h3>
          <span className="namespace">{namespace}</span>
        </div>
        <div className="header-right">
          {renderPhaseBadge()}
          {renderConnectionStatus()}
        </div>
      </div>

      <div className="card-content">
        {platform?.message && (
          <div className="status-message">
            <p>{platform.message}</p>
            <span className="last-updated">
              Last updated: {platform.lastUpdated.toLocaleTimeString()}
            </span>
          </div>
        )}

        {renderComponentHealth()}

        <div className="card-actions">
          <button 
            className="toggle-logs-btn"
            onClick={() => setShowLogs(!showLogs)}
          >
            {showLogs ? 'Hide Logs' : 'Show Logs'}
          </button>
        </div>

        {renderLogs()}
      </div>
    </div>
  );
};
