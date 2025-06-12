# Gunj Operator UI - Theme & Design System

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Planning Phase  

## 🎨 Design System Overview

### Design Philosophy
The Gunj Operator UI design system is built on these principles:
- **Professional & Enterprise-Ready**: Clean, modern aesthetic suitable for enterprise environments
- **Data-Centric**: Optimized for displaying complex observability data
- **Accessible**: WCAG 2.1 AA compliant
- **Kubernetes-Native**: Aligned with CNCF and Kubernetes ecosystem aesthetics
- **Performance-Focused**: Optimized for real-time data updates

### Technology Choice: Material-UI (MUI) v5

After careful evaluation, we've selected **Material-UI (MUI) v5** as our component library with extensive customization:

**Why MUI?**
- Comprehensive enterprise-ready components
- Excellent TypeScript support
- Robust theming system with sx prop
- Built-in accessibility features
- Strong data display components
- Large community and ecosystem
- Tree-shaking for optimal bundle size

## 🎨 Color System

### Brand Colors
```typescript
// Primary brand colors aligned with Kubernetes ecosystem
export const brandColors = {
  kubernetes: '#326CE5',
  prometheus: '#E6522C',
  grafana: '#F46800',
  loki: '#F5A623',
  tempo: '#7B61FF',
  gunj: '#1976D2', // Our primary
};
```

### Light Theme Palette
```typescript
export const lightPalette = {
  mode: 'light',
  primary: {
    main: '#1976D2',      // Kubernetes-inspired blue
    light: '#42A5F5',
    dark: '#1565C0',
    contrastText: '#FFFFFF',
  },
  secondary: {
    main: '#00ACC1',      // Teal for success/healthy states
    light: '#26C6DA',
    dark: '#00838F',
    contrastText: '#FFFFFF',
  },
  error: {
    main: '#D32F2F',
    light: '#EF5350',
    dark: '#C62828',
  },
  warning: {
    main: '#F57C00',
    light: '#FFB74D',
    dark: '#E65100',
  },
  info: {
    main: '#0288D1',
    light: '#29B6F6',
    dark: '#01579B',
  },
  success: {
    main: '#2E7D32',
    light: '#66BB6A',
    dark: '#1B5E20',
  },
  grey: {
    50: '#FAFAFA',
    100: '#F5F5F5',
    200: '#EEEEEE',
    300: '#E0E0E0',
    400: '#BDBDBD',
    500: '#9E9E9E',
    600: '#757575',
    700: '#616161',
    800: '#424242',
    900: '#212121',
  },
  background: {
    default: '#FAFAFA',
    paper: '#FFFFFF',
    elevated: '#FFFFFF',
  },
  text: {
    primary: 'rgba(0, 0, 0, 0.87)',
    secondary: 'rgba(0, 0, 0, 0.6)',
    disabled: 'rgba(0, 0, 0, 0.38)',
  },
  divider: 'rgba(0, 0, 0, 0.12)',
};
```

### Dark Theme Palette
```typescript
export const darkPalette = {
  mode: 'dark',
  primary: {
    main: '#42A5F5',      // Lighter blue for dark mode
    light: '#64B5F6',
    dark: '#1E88E5',
    contrastText: '#000000',
  },
  secondary: {
    main: '#26C6DA',
    light: '#4DD0E1',
    dark: '#00ACC1',
    contrastText: '#000000',
  },
  error: {
    main: '#EF5350',
    light: '#E57373',
    dark: '#E53935',
  },
  warning: {
    main: '#FFA726',
    light: '#FFB74D',
    dark: '#FB8C00',
  },
  info: {
    main: '#29B6F6',
    light: '#4FC3F7',
    dark: '#039BE5',
  },
  success: {
    main: '#66BB6A',
    light: '#81C784',
    dark: '#43A047',
  },
  grey: {
    50: '#121212',
    100: '#1E1E1E',
    200: '#2E2E2E',
    300: '#3E3E3E',
    400: '#4E4E4E',
    500: '#5E5E5E',
    600: '#6E6E6E',
    700: '#7E7E7E',
    800: '#8E8E8E',
    900: '#9E9E9E',
  },
  background: {
    default: '#121212',
    paper: '#1E1E1E',
    elevated: '#242424',
  },
  text: {
    primary: '#FFFFFF',
    secondary: 'rgba(255, 255, 255, 0.7)',
    disabled: 'rgba(255, 255, 255, 0.5)',
  },
  divider: 'rgba(255, 255, 255, 0.12)',
};
```

### Semantic Colors
```typescript
export const semanticColors = {
  // Component states
  healthy: '#4CAF50',
  degraded: '#FF9800',
  critical: '#F44336',
  unknown: '#9E9E9E',
  
  // Metrics
  cpu: '#2196F3',
  memory: '#9C27B0',
  storage: '#FF5722',
  network: '#00BCD4',
  
  // Observability components
  prometheus: '#E6522C',
  grafana: '#F46800',
  loki: '#F5A623',
  tempo: '#7B61FF',
  alertmanager: '#FF6B6B',
};
```

## 📝 Typography System

### Font Families
```typescript
export const typography = {
  fontFamily: {
    ui: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
    code: '"JetBrains Mono", "Fira Code", "Consolas", monospace',
    display: '"Inter", sans-serif',
  },
  
  // Type scale
  h1: {
    fontSize: '3rem',      // 48px
    fontWeight: 300,
    lineHeight: 1.2,
    letterSpacing: '-0.01562em',
  },
  h2: {
    fontSize: '2.125rem',  // 34px
    fontWeight: 300,
    lineHeight: 1.2,
    letterSpacing: '-0.00833em',
  },
  h3: {
    fontSize: '1.5rem',    // 24px
    fontWeight: 400,
    lineHeight: 1.167,
    letterSpacing: '0em',
  },
  h4: {
    fontSize: '1.25rem',   // 20px
    fontWeight: 400,
    lineHeight: 1.235,
    letterSpacing: '0.00735em',
  },
  h5: {
    fontSize: '1.125rem',  // 18px
    fontWeight: 400,
    lineHeight: 1.334,
    letterSpacing: '0em',
  },
  h6: {
    fontSize: '1rem',      // 16px
    fontWeight: 500,
    lineHeight: 1.6,
    letterSpacing: '0.0075em',
  },
  body1: {
    fontSize: '0.875rem',  // 14px
    lineHeight: 1.5,
    letterSpacing: '0.00938em',
  },
  body2: {
    fontSize: '0.75rem',   // 12px
    lineHeight: 1.43,
    letterSpacing: '0.01071em',
  },
  button: {
    fontSize: '0.875rem',  // 14px
    fontWeight: 500,
    lineHeight: 1.75,
    letterSpacing: '0.02857em',
    textTransform: 'none', // Override MUI default
  },
  caption: {
    fontSize: '0.75rem',   // 12px
    lineHeight: 1.66,
    letterSpacing: '0.03333em',
  },
  overline: {
    fontSize: '0.625rem',  // 10px
    fontWeight: 400,
    lineHeight: 2.66,
    letterSpacing: '0.08333em',
    textTransform: 'uppercase',
  },
  code: {
    fontFamily: '"JetBrains Mono", monospace',
    fontSize: '0.875rem',
  },
};
```

## 📐 Spacing & Layout System

### Spacing Scale
```typescript
export const spacing = {
  unit: 8, // Base unit in pixels
  
  // Spacing scale (multipliers of base unit)
  0: 0,      // 0px
  0.5: 4,    // 4px
  1: 8,      // 8px
  1.5: 12,   // 12px
  2: 16,     // 16px
  3: 24,     // 24px
  4: 32,     // 32px
  5: 40,     // 40px
  6: 48,     // 48px
  8: 64,     // 64px
  10: 80,    // 80px
  12: 96,    // 96px
  16: 128,   // 128px
};

// Layout constants
export const layout = {
  containerMaxWidth: 1440,
  sidebarWidth: 280,
  sidebarCollapsedWidth: 72,
  headerHeight: 64,
  contentPadding: 24,
  cardPadding: 16,
  
  // Breakpoints (aligned with MUI)
  breakpoints: {
    xs: 0,
    sm: 600,
    md: 960,
    lg: 1280,
    xl: 1920,
  },
  
  // Grid system
  grid: {
    columns: 12,
    gutter: 24,
  },
};
```

## 🎭 Component Variants & States

### Component States
```typescript
export const componentStates = {
  // Interactive states
  default: 'default',
  hover: 'hover',
  focus: 'focus',
  active: 'active',
  disabled: 'disabled',
  loading: 'loading',
  
  // Validation states
  valid: 'valid',
  invalid: 'invalid',
  warning: 'warning',
  
  // Platform-specific states
  healthy: 'healthy',
  degraded: 'degraded',
  critical: 'critical',
  unknown: 'unknown',
  installing: 'installing',
  upgrading: 'upgrading',
};
```

### Component Sizes
```typescript
export const componentSizes = {
  small: {
    height: 32,
    fontSize: 12,
    padding: '4px 12px',
    iconSize: 16,
  },
  medium: {
    height: 40,
    fontSize: 14,
    padding: '8px 16px',
    iconSize: 20,
  },
  large: {
    height: 48,
    fontSize: 16,
    padding: '12px 24px',
    iconSize: 24,
  },
};
```

### Elevation & Shadows
```typescript
export const elevation = {
  0: 'none',
  1: '0px 2px 4px rgba(0, 0, 0, 0.08)',
  2: '0px 4px 8px rgba(0, 0, 0, 0.12)',
  3: '0px 8px 16px rgba(0, 0, 0, 0.16)',
  4: '0px 12px 24px rgba(0, 0, 0, 0.20)',
  
  // Dark mode shadows (with glow effect)
  dark: {
    1: '0px 2px 4px rgba(0, 0, 0, 0.4)',
    2: '0px 4px 8px rgba(0, 0, 0, 0.5)',
    3: '0px 8px 16px rgba(0, 0, 0, 0.6)',
    4: '0px 12px 24px rgba(0, 0, 0, 0.7)',
  },
};
```

## 🎬 Animation & Transitions

### Transition Timing
```typescript
export const transitions = {
  // Duration
  duration: {
    shortest: 150,
    shorter: 200,
    short: 250,
    standard: 300,
    complex: 375,
    enteringScreen: 225,
    leavingScreen: 195,
  },
  
  // Easing functions
  easing: {
    easeInOut: 'cubic-bezier(0.4, 0, 0.2, 1)',
    easeOut: 'cubic-bezier(0.0, 0, 0.2, 1)',
    easeIn: 'cubic-bezier(0.4, 0, 1, 1)',
    sharp: 'cubic-bezier(0.4, 0, 0.6, 1)',
  },
  
  // Common transitions
  create: (props = ['all'], options = {}) => {
    const {
      duration = transitions.duration.standard,
      easing = transitions.easing.easeInOut,
      delay = 0,
    } = options;
    
    return props.map(prop => 
      `${prop} ${duration}ms ${easing} ${delay}ms`
    ).join(',');
  },
};
```

### Animation Patterns
```typescript
export const animations = {
  // Page transitions
  fadeIn: keyframes`
    from { opacity: 0; }
    to { opacity: 1; }
  `,
  
  slideInRight: keyframes`
    from { transform: translateX(24px); opacity: 0; }
    to { transform: translateX(0); opacity: 1; }
  `,
  
  // Real-time update indicators
  pulse: keyframes`
    0% { transform: scale(1); opacity: 1; }
    50% { transform: scale(1.05); opacity: 0.8; }
    100% { transform: scale(1); opacity: 1; }
  `,
  
  // Loading states
  shimmer: keyframes`
    0% { background-position: -200% 0; }
    100% { background-position: 200% 0; }
  `,
  
  // Status indicators
  statusGlow: keyframes`
    0% { box-shadow: 0 0 0 0 rgba(66, 165, 245, 0.7); }
    70% { box-shadow: 0 0 0 10px rgba(66, 165, 245, 0); }
    100% { box-shadow: 0 0 0 0 rgba(66, 165, 245, 0); }
  `,
};
```

## 🎯 Icon System

### Icon Libraries
```typescript
export const iconSystem = {
  primary: '@mui/icons-material',     // Material Icons
  secondary: 'lucide-react',          // Lucide for custom icons
  custom: '@gunj/icons',              // Custom icon set
  
  // Icon sizes
  sizes: {
    small: 16,
    medium: 20,
    large: 24,
    xlarge: 32,
  },
  
  // Common icons mapping
  icons: {
    // Navigation
    dashboard: 'Dashboard',
    platforms: 'ViewList',
    monitoring: 'Monitoring',
    settings: 'Settings',
    
    // Actions
    add: 'Add',
    edit: 'Edit',
    delete: 'Delete',
    refresh: 'Refresh',
    
    // Status
    success: 'CheckCircle',
    error: 'Error',
    warning: 'Warning',
    info: 'Info',
    
    // Components
    prometheus: 'CustomPrometheus',
    grafana: 'CustomGrafana',
    loki: 'CustomLoki',
    tempo: 'CustomTempo',
  },
};
```

## ♿ Accessibility Standards

### WCAG 2.1 AA Compliance
```typescript
export const accessibility = {
  // Color contrast ratios
  contrast: {
    AA: {
      normal: 4.5,    // Normal text
      large: 3.0,     // Large text (18pt+)
    },
    AAA: {
      normal: 7.0,
      large: 4.5,
    },
  },
  
  // Focus indicators
  focus: {
    outline: '2px solid #1976D2',
    outlineOffset: 2,
    borderRadius: 4,
  },
  
  // Interactive element sizes
  minTouchTarget: 44, // Minimum 44x44px
  
  // Keyboard navigation
  tabIndex: {
    focusable: 0,
    programmatic: -1,
    sequential: 1,
  },
  
  // Screen reader support
  aria: {
    live: {
      polite: 'polite',
      assertive: 'assertive',
      off: 'off',
    },
    required: [
      'label',
      'describedby',
      'role',
      'hidden',
    ],
  },
};
```

## 🧩 Component Examples

### Button Variants
```typescript
// Primary button
<Button
  variant="contained"
  color="primary"
  startIcon={<AddIcon />}
  sx={{
    boxShadow: elevation[1],
    '&:hover': {
      boxShadow: elevation[2],
    },
  }}
>
  Create Platform
</Button>

// Status indicator button
<Chip
  label="Healthy"
  color="success"
  size="small"
  icon={<CheckCircleIcon />}
  sx={{
    animation: `${animations.pulse} 2s infinite`,
  }}
/>
```

### Card Component
```typescript
<Card
  elevation={1}
  sx={{
    p: 2,
    transition: transitions.create(['box-shadow']),
    '&:hover': {
      boxShadow: elevation[3],
    },
  }}
>
  <CardContent>
    <Typography variant="h6" gutterBottom>
      Platform Overview
    </Typography>
    {/* Content */}
  </CardContent>
</Card>
```

## 📱 Responsive Design

### Breakpoint Usage
```typescript
export const responsive = {
  // Hide on mobile
  hideOnMobile: {
    display: { xs: 'none', sm: 'block' },
  },
  
  // Stack on mobile
  stackOnMobile: {
    flexDirection: { xs: 'column', sm: 'row' },
  },
  
  // Responsive spacing
  responsivePadding: {
    p: { xs: 2, sm: 3, md: 4 },
  },
  
  // Responsive typography
  responsiveText: {
    fontSize: { xs: '1rem', sm: '1.125rem', md: '1.25rem' },
  },
};
```

## 🎨 Theme Implementation

### Theme Configuration
```typescript
import { createTheme, ThemeOptions } from '@mui/material/styles';

const getTheme = (mode: 'light' | 'dark'): ThemeOptions => ({
  palette: mode === 'light' ? lightPalette : darkPalette,
  typography,
  spacing: spacing.unit,
  shape: {
    borderRadius: 8,
  },
  transitions,
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          borderRadius: 8,
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 12,
          backgroundImage: 'none',
        },
      },
    },
    // ... more component overrides
  },
});

export const lightTheme = createTheme(getTheme('light'));
export const darkTheme = createTheme(getTheme('dark'));
```

## 📋 Design Tokens Export

For cross-platform consistency, we'll export design tokens:

```json
{
  "color": {
    "primary": {
      "value": "#1976D2",
      "type": "color"
    },
    "secondary": {
      "value": "#00ACC1",
      "type": "color"
    }
  },
  "spacing": {
    "base": {
      "value": "8",
      "type": "spacing"
    }
  },
  "typography": {
    "heading1": {
      "value": {
        "fontFamily": "Inter",
        "fontSize": "48",
        "fontWeight": "300"
      },
      "type": "typography"
    }
  }
}
```

## 🚀 Next Steps

1. **Component Library Setup**: Configure Storybook for component development
2. **Design Token Pipeline**: Set up automated token generation
3. **Theme Provider**: Implement theme switching logic
4. **Component Templates**: Create reusable component templates
5. **Documentation Site**: Build design system documentation

---

This design system provides a solid foundation for building a professional, accessible, and maintainable UI for the Gunj Operator platform.