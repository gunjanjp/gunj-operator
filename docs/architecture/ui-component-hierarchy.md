# Gunj Operator UI Component Hierarchy Design

**Project**: Gunj Operator v2.0  
**Phase**: 1 - Foundation & Architecture Design  
**Task**: 1.1.3 - UI Architecture Design  
**Micro-task**: Design Component Hierarchy  
**Date**: June 12, 2025  

## Component Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        App (Root)                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   AppProviders                       │   │
│  │  (Theme, Auth, Router, Query, i18n, WebSocket)     │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   AppRouter                          │   │
│  │  ┌─────────────────┬────────────────────────────┐  │   │
│  │  │   AuthLayout     │      MainLayout           │  │   │
│  │  │   (Public)       │      (Protected)          │  │   │
│  │  └─────────────────┴────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 1. Core Layout Components

### 1.1 App (Root Component)
```typescript
interface AppProps {
  config?: AppConfig;
}

// Responsibilities:
// - Initialize app-wide providers
// - Handle global error boundaries
// - Setup service workers
// - Configure monitoring
```

### 1.2 AppProviders
```typescript
interface AppProvidersProps {
  children: ReactNode;
  config: AppConfig;
}

// Nested Providers:
// - ThemeProvider (MUI/custom theme)
// - AuthProvider (authentication context)
// - RouterProvider (React Router)
// - QueryClientProvider (React Query)
// - I18nProvider (internationalization)
// - WebSocketProvider (real-time updates)
// - NotificationProvider (toast/alerts)
// - ErrorBoundary (global error handling)
```

### 1.3 Layout Components

#### MainLayout
```typescript
interface MainLayoutProps {
  children: ReactNode;
}

// Structure:
MainLayout/
├── Header/
│   ├── Logo
│   ├── Navigation
│   ├── SearchBar
│   ├── NotificationBell
│   ├── UserMenu
│   └── ThemeToggle
├── Sidebar/
│   ├── NavigationMenu
│   ├── QuickActions
│   └── ContextualHelp
├── Content/
│   ├── Breadcrumbs
│   ├── PageHeader
│   └── {children}
└── Footer/
    ├── StatusBar
    └── VersionInfo
```

#### AuthLayout
```typescript
interface AuthLayoutProps {
  children: ReactNode;
}

// Structure:
AuthLayout/
├── AuthHeader
├── AuthContent
└── AuthFooter
```

## 2. Page Components

### 2.1 Dashboard Pages
```
pages/dashboard/
├── DashboardPage/
│   ├── DashboardHeader
│   ├── MetricsSummary
│   ├── PlatformGrid
│   └── ActivityFeed
├── PlatformOverviewPage/
│   ├── PlatformHeader
│   ├── ComponentStatusGrid
│   ├── ResourceUsageCharts
│   └── RecentEvents
└── SystemHealthPage/
    ├── HealthMatrix
    ├── AlertsSummary
    └── PerformanceMetrics
```

### 2.2 Platform Management Pages
```
pages/platforms/
├── PlatformListPage/
│   ├── PlatformFilters
│   ├── PlatformTable
│   └── PlatformCards
├── PlatformCreatePage/
│   ├── CreateWizard/
│   │   ├── BasicInfoStep
│   │   ├── ComponentSelectionStep
│   │   ├── ConfigurationStep
│   │   ├── ResourceAllocationStep
│   │   └── ReviewStep
│   └── CreateActions
├── PlatformDetailPage/
│   ├── PlatformTabs/
│   │   ├── OverviewTab
│   │   ├── ComponentsTab
│   │   ├── ConfigurationTab
│   │   ├── MetricsTab
│   │   ├── LogsTab
│   │   └── EventsTab
│   └── PlatformActions
└── PlatformEditPage/
    ├── EditForm
    └── EditActions
```

### 2.3 Component Management Pages
```
pages/components/
├── PrometheusPage/
│   ├── PrometheusConfig
│   ├── PrometheusTargets
│   └── PrometheusRules
├── GrafanaPage/
│   ├── GrafanaDashboards
│   ├── GrafanaDataSources
│   └── GrafanaUsers
├── LokiPage/
│   ├── LokiConfig
│   ├── LokiIngestion
│   └── LokiRetention
└── TempoPage/
    ├── TempoConfig
    ├── TempoTraces
    └── TempoServiceMap
```

### 2.4 Operations Pages
```
pages/operations/
├── BackupRestorePage/
│   ├── BackupList
│   ├── BackupScheduler
│   └── RestoreWizard
├── MonitoringPage/
│   ├── MetricsDashboard
│   ├── LogViewer
│   └── TraceExplorer
├── AlertingPage/
│   ├── AlertRules
│   ├── AlertHistory
│   └── NotificationChannels
└── CostOptimizationPage/
    ├── CostAnalysis
    ├── ResourceRecommendations
    └── OptimizationHistory
```

### 2.5 Settings Pages
```
pages/settings/
├── GeneralSettingsPage/
├── SecuritySettingsPage/
├── IntegrationsPage/
├── TeamManagementPage/
└── AuditLogPage/
```

## 3. Reusable UI Components

### 3.1 Common Components
```
components/common/
├── Button/
│   ├── Button.tsx
│   ├── IconButton.tsx
│   └── ButtonGroup.tsx
├── Form/
│   ├── TextField/
│   ├── Select/
│   ├── Checkbox/
│   ├── RadioGroup/
│   ├── Switch/
│   ├── DatePicker/
│   └── FormValidator/
├── DataDisplay/
│   ├── Table/
│   │   ├── DataTable.tsx
│   │   ├── TablePagination.tsx
│   │   └── TableFilters.tsx
│   ├── Card/
│   ├── List/
│   ├── Badge/
│   └── Tooltip/
├── Feedback/
│   ├── Alert/
│   ├── Dialog/
│   ├── Snackbar/
│   ├── Progress/
│   └── Skeleton/
├── Navigation/
│   ├── Breadcrumbs/
│   ├── Tabs/
│   ├── Stepper/
│   └── Pagination/
└── Layout/
    ├── Grid/
    ├── Stack/
    ├── Divider/
    └── Spacer/
```

### 3.2 Domain-Specific Components
```
components/domain/
├── Platform/
│   ├── PlatformCard/
│   ├── PlatformStatus/
│   ├── PlatformMetrics/
│   └── PlatformActions/
├── Component/
│   ├── ComponentCard/
│   ├── ComponentConfig/
│   ├── ComponentHealth/
│   └── ComponentVersion/
├── Monitoring/
│   ├── MetricsChart/
│   ├── LogStream/
│   ├── TraceViewer/
│   └── AlertPanel/
├── Resource/
│   ├── ResourceGauge/
│   ├── ResourceChart/
│   └── ResourceOptimizer/
└── Configuration/
    ├── YamlEditor/
    ├── ConfigValidator/
    └── ConfigDiff/
```

### 3.3 Composite Components
```
components/composite/
├── PlatformWizard/
│   ├── WizardProvider
│   ├── WizardSteps
│   └── WizardNavigation
├── ResourceDashboard/
│   ├── ResourceOverview
│   ├── ResourceCharts
│   └── ResourceAlerts
├── ComponentManager/
│   ├── ComponentList
│   ├── ComponentEditor
│   └── ComponentActions
└── MonitoringDashboard/
    ├── MetricsPanel
    ├── LogsPanel
    └── TracesPanel
```

## 4. Hooks and Utilities

### 4.1 Custom Hooks
```
hooks/
├── auth/
│   ├── useAuth.ts
│   ├── usePermissions.ts
│   └── useUser.ts
├── api/
│   ├── usePlatforms.ts
│   ├── useComponents.ts
│   ├── useMetrics.ts
│   └── useLogs.ts
├── ui/
│   ├── useTheme.ts
│   ├── useMediaQuery.ts
│   ├── useDebounce.ts
│   └── useLocalStorage.ts
└── realtime/
    ├── useWebSocket.ts
    ├── useEventStream.ts
    └── useNotifications.ts
```

### 4.2 Context Providers
```
contexts/
├── AuthContext.tsx
├── ThemeContext.tsx
├── NotificationContext.tsx
├── WebSocketContext.tsx
└── ConfigContext.tsx
```

## 5. Folder Structure

```
ui/src/
├── components/
│   ├── common/        # Reusable UI components
│   ├── domain/        # Domain-specific components
│   ├── composite/     # Complex composite components
│   └── layouts/       # Layout components
├── pages/            # Page components
│   ├── auth/
│   ├── dashboard/
│   ├── platforms/
│   ├── components/
│   ├── operations/
│   └── settings/
├── hooks/            # Custom React hooks
├── contexts/         # React contexts
├── services/         # API and external services
├── stores/           # State management (Zustand)
├── utils/            # Utility functions
├── types/            # TypeScript types
├── styles/           # Global styles and themes
├── locales/          # i18n translations
└── __tests__/        # Test files
```

## 6. Component Communication Pattern

### 6.1 Props Flow
```
App
 └─> AppProviders (config)
      └─> MainLayout
           └─> Page Components (route params)
                └─> Domain Components (data, callbacks)
                     └─> Common Components (props)
```

### 6.2 State Management Flow
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Components    │────>│  Actions/Hooks  │────>│     Stores      │
│                 │     │                 │     │   (Zustand)     │
│                 │<────│                 │<────│                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        ↓                        ↓                        ↓
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Local State   │     │   API Calls     │     │  Subscriptions  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### 6.3 Data Flow Pattern
```
1. Page Component
   - Fetches data using hooks
   - Manages page-level state
   - Handles routing

2. Domain Components
   - Receive data as props
   - Handle business logic
   - Emit events upward

3. Common Components
   - Pure presentation
   - No business logic
   - Highly reusable
```

## 7. Key Design Decisions

### 7.1 Component Principles
- **Single Responsibility**: Each component has one clear purpose
- **Composition over Inheritance**: Use React composition patterns
- **Props Interface**: Clear TypeScript interfaces for all components
- **Memoization**: Use React.memo for expensive components
- **Error Boundaries**: Wrap feature areas with error boundaries

### 7.2 State Management Strategy
- **Local State**: Component-specific UI state
- **Zustand Stores**: Global application state
- **React Query**: Server state and caching
- **Context**: Cross-cutting concerns (theme, auth)
- **URL State**: Navigation and shareable state

### 7.3 Performance Optimizations
- **Code Splitting**: Lazy load routes and heavy components
- **Virtual Lists**: For large data sets
- **Debouncing**: For search and filter inputs
- **Memoization**: For expensive computations
- **Suspense**: For async component loading

## 8. Component Examples

### 8.1 PlatformCard Component
```typescript
interface PlatformCardProps {
  platform: Platform;
  onSelect?: (platform: Platform) => void;
  onEdit?: (platform: Platform) => void;
  onDelete?: (platform: Platform) => void;
  isSelected?: boolean;
  showActions?: boolean;
}

// Features:
// - Displays platform status and health
// - Shows resource usage
// - Quick actions menu
// - Real-time status updates
// - Responsive design
```

### 8.2 MetricsChart Component
```typescript
interface MetricsChartProps {
  data: MetricData[];
  type: 'line' | 'bar' | 'area' | 'gauge';
  timeRange: TimeRange;
  refreshInterval?: number;
  onDataPointClick?: (point: DataPoint) => void;
}

// Features:
// - Multiple chart types
// - Real-time updates
// - Interactive tooltips
// - Zoom and pan
// - Export functionality
```

## 9. Testing Strategy

### 9.1 Component Testing
- Unit tests for all components
- Integration tests for complex flows
- Visual regression tests
- Accessibility tests
- Performance tests

### 9.2 Test Structure
```
__tests__/
├── components/
│   ├── common/
│   ├── domain/
│   └── pages/
├── hooks/
├── utils/
└── integration/
```

## 10. Next Steps

1. **State Management Design** (Next micro-task)
2. **Routing Structure** 
3. **Real-time Update Mechanism**
4. **Theme and Design System**
5. **Wireframes and Mockups**

---

**Status**: Component Hierarchy Design Complete  
**Next**: State Management Architecture (Zustand)
