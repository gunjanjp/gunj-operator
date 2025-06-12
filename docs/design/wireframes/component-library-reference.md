# Gunj Operator UI - Component Library Reference

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Design Phase  

## 📚 Component Library Overview

This document provides a detailed reference for all UI components used in the Gunj Operator interface. Each component is designed following our theme and design system guidelines.

---

## 🎨 Base Components

### Buttons

#### Primary Button
```tsx
<Button
  variant="contained"
  color="primary"
  size="medium"
  startIcon={<AddIcon />}
  onClick={handleClick}
>
  Create Platform
</Button>
```

**Variants:**
- `contained` - Filled background (primary actions)
- `outlined` - Border only (secondary actions)
- `text` - No border/background (tertiary actions)

**Sizes:**
- `small` - 32px height
- `medium` - 40px height (default)
- `large` - 48px height

**States:**
- Default
- Hover (elevation + brightness)
- Active (pressed)
- Disabled (50% opacity)
- Loading (with spinner)

#### Icon Button
```tsx
<IconButton
  color="primary"
  size="medium"
  onClick={handleRefresh}
  aria-label="refresh"
>
  <RefreshIcon />
</IconButton>
```

**Common Icons:**
- Settings: `<SettingsIcon />`
- Refresh: `<RefreshIcon />`
- Delete: `<DeleteIcon />`
- Edit: `<EditIcon />`
- More: `<MoreVertIcon />`

### Status Components

#### Status Chip
```tsx
<Chip
  label="Healthy"
  color="success"
  size="small"
  icon={<CheckCircleIcon />}
  variant="filled"
/>
```

**Status Colors:**
- `success` - Healthy/Ready (Green)
- `warning` - Degraded/Warning (Orange)
- `error` - Critical/Failed (Red)
- `info` - Installing/Updating (Blue)
- `default` - Unknown/Disabled (Gray)

#### Status Indicator
```tsx
<StatusIndicator
  status="healthy"
  size="small"
  showLabel={true}
  pulse={true}
/>
```

**Features:**
- Animated pulse for active states
- Tooltip on hover
- Accessible color + icon combinations
- Real-time updates via WebSocket

### Form Components

#### Text Field
```tsx
<TextField
  label="Platform Name"
  value={platformName}
  onChange={handleChange}
  required
  fullWidth
  helperText="Use lowercase letters, numbers, and hyphens"
  error={!!errors.platformName}
  InputProps={{
    startAdornment: <InputAdornment position="start">platform-</InputAdornment>,
  }}
/>
```

**Variants:**
- `outlined` - Default with border
- `filled` - Gray background
- `standard` - Underline only

#### Select
```tsx
<FormControl fullWidth>
  <InputLabel>Prometheus Version</InputLabel>
  <Select
    value={version}
    onChange={handleVersionChange}
    label="Prometheus Version"
  >
    <MenuItem value="v2.48.0">v2.48.0 (Latest)</MenuItem>
    <MenuItem value="v2.47.0">v2.47.0</MenuItem>
    <MenuItem value="v2.46.0">v2.46.0</MenuItem>
  </Select>
</FormControl>
```

#### Switch
```tsx
<FormControlLabel
  control={
    <Switch
      checked={enableHA}
      onChange={handleHAToggle}
      color="primary"
    />
  }
  label="Enable High Availability"
/>
```

### Data Display

#### Data Table
```tsx
<DataTable
  columns={[
    { field: 'name', headerName: 'Platform', width: 200 },
    { field: 'status', headerName: 'Status', width: 120, renderCell: StatusCell },
    { field: 'version', headerName: 'Version', width: 100 },
    { field: 'created', headerName: 'Created', width: 150, type: 'dateTime' },
  ]}
  rows={platforms}
  pageSize={10}
  checkboxSelection
  onRowClick={handleRowClick}
  loading={isLoading}
/>
```

**Features:**
- Sortable columns
- Filterable data
- Pagination
- Row selection
- Responsive scroll
- Export functionality

#### Metric Card
```tsx
<MetricCard
  title="CPU Usage"
  value={68}
  unit="%"
  trend="up"
  trendValue={2.3}
  sparklineData={cpuHistory}
  color="primary"
  icon={<CpuIcon />}
/>
```

**Displays:**
- Current value with unit
- Trend indicator (up/down/stable)
- Mini sparkline chart
- Status color coding
- Optional icon

### Navigation

#### Sidebar Navigation
```tsx
<Sidebar
  width={280}
  collapsible
  defaultCollapsed={false}
>
  <SidebarHeader>
    <Logo />
    <Title>Gunj Operator</Title>
  </SidebarHeader>
  
  <SidebarNav>
    <NavItem icon={<DashboardIcon />} label="Dashboard" to="/" />
    <NavItem icon={<ListIcon />} label="Platforms" to="/platforms" />
    <NavItem icon={<MonitoringIcon />} label="Monitoring" to="/monitoring" />
    <NavItem icon={<SettingsIcon />} label="Settings" to="/settings" />
  </SidebarNav>
  
  <SidebarFooter>
    <UserMenu />
  </SidebarFooter>
</Sidebar>
```

#### Breadcrumbs
```tsx
<Breadcrumbs separator="›" aria-label="breadcrumb">
  <Link color="inherit" href="/">
    Dashboard
  </Link>
  <Link color="inherit" href="/platforms">
    Platforms
  </Link>
  <Typography color="text.primary">production</Typography>
</Breadcrumbs>
```

### Feedback Components

#### Alert
```tsx
<Alert
  severity="warning"
  variant="filled"
  onClose={handleClose}
  action={
    <Button color="inherit" size="small" onClick={handleAction}>
      View Details
    </Button>
  }
>
  <AlertTitle>High Resource Usage</AlertTitle>
  CPU usage has exceeded 90% threshold on prometheus-2
</Alert>
```

#### Toast/Snackbar
```tsx
<Snackbar
  open={open}
  autoHideDuration={6000}
  onClose={handleClose}
  anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
>
  <Alert severity="success" variant="filled">
    Platform created successfully!
  </Alert>
</Snackbar>
```

#### Progress
```tsx
// Linear Progress
<LinearProgress
  variant="determinate"
  value={progress}
  color="primary"
  sx={{ height: 8, borderRadius: 4 }}
/>

// Circular Progress
<CircularProgress
  variant="determinate"
  value={progress}
  size={60}
  thickness={4}
/>
```

### Charts & Visualizations

#### Line Chart
```tsx
<LineChart
  data={metricsData}
  lines={[
    { dataKey: 'cpu', stroke: '#1976D2', name: 'CPU Usage' },
    { dataKey: 'memory', stroke: '#9C27B0', name: 'Memory Usage' },
  ]}
  xAxis={{ dataKey: 'timestamp', type: 'time' }}
  yAxis={{ unit: '%', domain: [0, 100] }}
  height={300}
  showGrid
  showTooltip
  showLegend
/>
```

#### Resource Gauge
```tsx
<ResourceGauge
  label="Memory Usage"
  value={52}
  max={100}
  unit="GB"
  thresholds={[
    { value: 0, color: 'success' },
    { value: 70, color: 'warning' },
    { value: 90, color: 'error' },
  ]}
  size="medium"
  animated
/>
```

---

## 🎯 Composite Components

### Platform Card
```tsx
<PlatformCard
  platform={{
    name: 'production',
    namespace: 'monitoring',
    status: 'healthy',
    version: 'v2.48.0',
    components: ['prometheus', 'grafana', 'loki'],
    resources: { cpu: 8, memory: 32, storage: 100 },
  }}
  onView={handleView}
  onEdit={handleEdit}
  onDelete={handleDelete}
  elevation={1}
  interactive
/>
```

**Features:**
- Status indicator with color coding
- Component icons
- Resource usage summary
- Action buttons
- Hover effects
- Click to view details

### Component Configuration Panel
```tsx
<ComponentConfigPanel
  component="prometheus"
  config={prometheusConfig}
  onChange={handleConfigChange}
  onSave={handleSave}
  onCancel={handleCancel}
  showAdvanced={false}
  validationErrors={errors}
/>
```

**Sections:**
- Basic settings (version, replicas)
- Resource allocation
- Storage configuration
- Advanced settings (collapsed by default)
- Validation messages
- Action buttons

### Wizard Stepper
```tsx
<WizardStepper
  steps={[
    { label: 'Basic Info', component: <BasicInfoStep /> },
    { label: 'Components', component: <ComponentsStep /> },
    { label: 'Resources', component: <ResourcesStep /> },
    { label: 'Review', component: <ReviewStep /> },
  ]}
  activeStep={activeStep}
  onNext={handleNext}
  onBack={handleBack}
  onFinish={handleFinish}
  orientation="horizontal"
/>
```

**Features:**
- Step validation
- Progress indication
- Skip optional steps
- Review before submit
- Error recovery

### Metric Dashboard Panel
```tsx
<MetricPanel
  title="System Resources"
  timeRange="1h"
  refreshInterval={30}
  metrics={[
    { type: 'cpu', query: 'rate(cpu_usage[5m])' },
    { type: 'memory', query: 'memory_usage_bytes' },
  ]}
  layout="grid"
  downloadable
  shareable
/>
```

**Capabilities:**
- Real-time data updates
- Time range selector
- Auto-refresh toggle
- Export (PNG/CSV)
- Share link generation
- Full-screen mode

---

## 🎨 Theme Integration

### Using Theme Variables
```tsx
// In styled components
const StyledCard = styled(Card)(({ theme }) => ({
  backgroundColor: theme.palette.background.paper,
  borderRadius: theme.shape.borderRadius * 1.5,
  padding: theme.spacing(3),
  transition: theme.transitions.create(['box-shadow', 'transform']),
  
  '&:hover': {
    boxShadow: theme.shadows[4],
    transform: 'translateY(-2px)',
  },
}));

// Using sx prop
<Box
  sx={{
    bgcolor: 'background.paper',
    p: 3,
    borderRadius: 2,
    boxShadow: 1,
    
    '&:hover': {
      boxShadow: 3,
    },
  }}
>
  Content
</Box>
```

### Dark Mode Support
```tsx
// Automatic theme switching
<Paper
  sx={{
    bgcolor: 'background.paper',
    color: 'text.primary',
    border: 1,
    borderColor: 'divider',
  }}
>
  This adapts to light/dark mode automatically
</Paper>

// Conditional styling
<Box
  sx={{
    bgcolor: (theme) => 
      theme.palette.mode === 'dark' ? 'grey.900' : 'grey.100',
  }}
>
  Custom dark mode handling
</Box>
```

---

## ♿ Accessibility Guidelines

### Keyboard Navigation
- All interactive elements accessible via Tab
- Escape key closes modals/dropdowns
- Enter/Space activates buttons
- Arrow keys navigate menus/lists

### Screen Reader Support
```tsx
// Proper labeling
<IconButton aria-label="delete platform">
  <DeleteIcon />
</IconButton>

// Live regions for updates
<div aria-live="polite" aria-atomic="true">
  {statusMessage}
</div>

// Describing complex interactions
<Tooltip title="CPU usage over the last hour">
  <div aria-describedby="cpu-tooltip">
    <LineChart {...props} />
  </div>
</Tooltip>
```

### Focus Management
```tsx
// Trap focus in modals
<Dialog
  open={open}
  onClose={handleClose}
  aria-labelledby="dialog-title"
  disableRestoreFocus
>
  <DialogTitle id="dialog-title">
    Confirm Delete
  </DialogTitle>
  {/* Auto-focus first interactive element */}
  <DialogContent>
    <DialogContentText>
      Are you sure you want to delete this platform?
    </DialogContentText>
  </DialogContent>
  <DialogActions>
    <Button onClick={handleClose} autoFocus>
      Cancel
    </Button>
    <Button onClick={handleDelete} color="error">
      Delete
    </Button>
  </DialogActions>
</Dialog>
```

---

## 📱 Responsive Behavior

### Breakpoint Utilities
```tsx
// Hide on mobile
<Box sx={{ display: { xs: 'none', sm: 'block' } }}>
  Desktop only content
</Box>

// Stack on mobile
<Stack
  direction={{ xs: 'column', sm: 'row' }}
  spacing={2}
>
  <Item>First</Item>
  <Item>Second</Item>
</Stack>

// Responsive typography
<Typography
  variant="h4"
  sx={{
    fontSize: {
      xs: '1.5rem',
      sm: '2rem',
      md: '2.5rem',
    },
  }}
>
  Responsive Heading
</Typography>
```

### Mobile Adaptations
1. **Navigation**: Drawer replaces sidebar
2. **Tables**: Horizontal scroll or card view
3. **Forms**: Full-width inputs, stacked layout
4. **Charts**: Simplified, touch-friendly
5. **Modals**: Full-screen on mobile

---

## 🚀 Performance Considerations

### Component Optimization
```tsx
// Memoize expensive components
const ExpensiveChart = React.memo(({ data }) => {
  return <ComplexVisualization data={data} />;
}, (prevProps, nextProps) => {
  return prevProps.data === nextProps.data;
});

// Lazy load heavy components
const MonitoringDashboard = React.lazy(() => 
  import('./MonitoringDashboard')
);

// Virtualize long lists
<VirtualizedList
  items={platforms}
  itemHeight={72}
  renderItem={renderPlatformRow}
  overscan={5}
/>
```

### Bundle Optimization
- Tree-shake MUI imports
- Use dynamic imports for routes
- Optimize image assets
- Minify CSS-in-JS

---

This component library reference provides the foundation for building consistent, accessible, and performant UI components for the Gunj Operator.
