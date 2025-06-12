import { Theme } from '@mui/material/styles';
import { semanticColors } from './theme';

export type PlatformStatus = 'healthy' | 'degraded' | 'critical' | 'unknown' | 'installing' | 'upgrading';
export type ComponentStatus = 'running' | 'pending' | 'failed' | 'stopped' | 'unknown';

// Get color for platform status
export const getPlatformStatusColor = (status: PlatformStatus): string => {
  return semanticColors[status] || semanticColors.unknown;
};

// Get color for component status
export const getComponentStatusColor = (status: ComponentStatus): string => {
  const statusColorMap: Record<ComponentStatus, string> = {
    running: semanticColors.healthy,
    pending: semanticColors.installing,
    failed: semanticColors.critical,
    stopped: semanticColors.degraded,
    unknown: semanticColors.unknown,
  };
  return statusColorMap[status] || semanticColors.unknown;
};

// Get styles for status chips
export const getStatusChipStyles = (status: PlatformStatus | ComponentStatus, theme: Theme) => {
  const color = status in semanticColors 
    ? getPlatformStatusColor(status as PlatformStatus)
    : getComponentStatusColor(status as ComponentStatus);

  return {
    backgroundColor: theme.palette.mode === 'light' 
      ? `${color}20` // 20% opacity for light mode
      : `${color}30`, // 30% opacity for dark mode
    color: color,
    borderColor: color,
    '& .MuiChip-icon': {
      color: color,
    },
  };
};

// Animation for different statuses
export const getStatusAnimation = (status: PlatformStatus | ComponentStatus) => {
  switch (status) {
    case 'installing':
    case 'upgrading':
    case 'pending':
      return {
        animation: 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        '@keyframes pulse': {
          '0%, 100%': { opacity: 1 },
          '50%': { opacity: 0.5 },
        },
      };
    case 'critical':
    case 'failed':
      return {
        animation: 'shake 0.5s cubic-bezier(0.36, 0.07, 0.19, 0.97) both',
        '@keyframes shake': {
          '10%, 90%': { transform: 'translate3d(-1px, 0, 0)' },
          '20%, 80%': { transform: 'translate3d(2px, 0, 0)' },
          '30%, 50%, 70%': { transform: 'translate3d(-2px, 0, 0)' },
          '40%, 60%': { transform: 'translate3d(2px, 0, 0)' },
        },
      };
    default:
      return {};
  }
};

// Get icon for status
export const getStatusIcon = (status: PlatformStatus | ComponentStatus) => {
  const iconMap: Record<string, string> = {
    healthy: 'CheckCircle',
    running: 'CheckCircle',
    degraded: 'Warning',
    critical: 'Error',
    failed: 'Error',
    unknown: 'Help',
    installing: 'CloudDownload',
    upgrading: 'SystemUpdate',
    pending: 'Schedule',
    stopped: 'StopCircle',
  };
  return iconMap[status] || 'Help';
};

// Platform health score calculation
export const calculateHealthScore = (components: Record<string, ComponentStatus>): number => {
  const statuses = Object.values(components);
  const weights = {
    running: 100,
    pending: 50,
    stopped: 25,
    failed: 0,
    unknown: 0,
  };

  if (statuses.length === 0) return 0;

  const totalScore = statuses.reduce((sum, status) => sum + (weights[status] || 0), 0);
  return Math.round(totalScore / statuses.length);
};

// Get health score color
export const getHealthScoreColor = (score: number): string => {
  if (score >= 90) return semanticColors.healthy;
  if (score >= 70) return semanticColors.degraded;
  if (score >= 50) return semanticColors.degraded;
  return semanticColors.critical;
};
