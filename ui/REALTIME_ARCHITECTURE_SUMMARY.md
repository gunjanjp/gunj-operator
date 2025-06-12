# Real-time Update Architecture - Implementation Summary

## Overview

The real-time update mechanism for the Gunj Operator UI has been designed and implemented using a hybrid approach combining WebSocket and Server-Sent Events (SSE) for optimal performance and resource utilization.

## Architecture Decisions

### 1. Hybrid Approach: WebSocket + SSE

**WebSocket** (Primary):
- Bidirectional communication
- Platform status updates
- Component health monitoring
- Alert notifications
- Command execution feedback

**Server-Sent Events** (Secondary):
- Unidirectional data streams
- Log streaming
- Metrics time-series data
- Progress tracking
- Event feeds

### 2. Key Components Implemented

#### ConnectionManager (`ConnectionManager.ts`)
- Manages both WebSocket and SSE connections
- Handles authentication with token refresh
- Automatic reconnection with exponential backoff
- Heartbeat mechanism for connection health
- Online/offline transition handling

#### EventRouter (`EventRouter.ts`)
- Pattern-based event subscription system
- Supports wildcards: `*` (single segment), `**` (multiple segments)
- Priority-based event handling
- Event buffering with ring buffer (1000 events)
- Filter and transform capabilities

#### Zustand Store (`useRealtimeStore.ts`)
- Centralized real-time state management
- Platform updates and component health tracking
- Log and metric stream management
- Automatic data cleanup (5-minute retention)
- Action methods for subscription management

#### BackpressureManager (`BackpressureManager.ts`)
- Adaptive rate limiting (50-200 msgs/sec)
- Priority queue for message ordering
- Buffer overflow handling (10K message buffer)
- Performance monitoring and adjustment
- CPU/memory/frame rate based throttling

### 3. React Integration

#### Hooks Provided:
- `useRealtime()` - Connection management
- `useRealtimePlatform()` - Platform-specific updates
- `useRealtimeAlerts()` - Alert monitoring
- `useRealtimeLogs()` - Log streaming with filters
- `useRealtimeMetrics()` - Metric aggregation

#### Example Component:
- `PlatformStatusCard.tsx` demonstrates real-world usage
- Shows connection status, platform phase, component health
- Optional real-time log streaming
- Responsive to connection state changes

### 4. Configuration

The system is configured via `realtime.config.ts` with:
- WebSocket and SSE endpoints
- Reconnection parameters
- Performance settings
- Feature flags
- Security options
- Development/production environment support

## File Structure Created

```
D:\claude\gunj-operator\ui\src\
├── services\
│   └── realtime\
│       ├── ConnectionManager.ts
│       ├── EventRouter.ts
│       └── BackpressureManager.ts
├── stores\
│   └── useRealtimeStore.ts
├── hooks\
│   └── useRealtime.ts
├── config\
│   └── realtime.config.ts
└── components\
    └── examples\
        └── PlatformStatusCard.tsx
```

## Next Steps

For the next micro-task, you should proceed with:
- **Task 1.1.3, Micro-task 5**: Plan theme and design system
- This will involve creating a comprehensive theming solution for the UI
- Consider Material-UI, Ant Design, or custom CSS-in-JS solution
- Define color palettes, typography, spacing, and component variants
