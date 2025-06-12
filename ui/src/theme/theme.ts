// Theme configuration for Gunj Operator UI
import { createTheme, ThemeOptions, alpha } from '@mui/material/styles';
import { PaletteMode } from '@mui/material';

// Brand colors
export const brandColors = {
  kubernetes: '#326CE5',
  prometheus: '#E6522C',
  grafana: '#F46800',
  loki: '#F5A623',
  tempo: '#7B61FF',
  gunj: '#1976D2',
} as const;

// Semantic colors for platform states
export const semanticColors = {
  healthy: '#4CAF50',
  degraded: '#FF9800',
  critical: '#F44336',
  unknown: '#9E9E9E',
  installing: '#2196F3',
  upgrading: '#9C27B0',
} as const;

// Metrics colors
export const metricsColors = {
  cpu: '#2196F3',
  memory: '#9C27B0',
  storage: '#FF5722',
  network: '#00BCD4',
  latency: '#795548',
  throughput: '#607D8B',
} as const;

// Get palette based on mode
const getPalette = (mode: PaletteMode) => ({
  mode,
  ...(mode === 'light'
    ? {
        // Light palette
        primary: {
          main: '#1976D2',
          light: '#42A5F5',
          dark: '#1565C0',
          contrastText: '#FFFFFF',
        },
        secondary: {
          main: '#00ACC1',
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
        background: {
          default: '#FAFAFA',
          paper: '#FFFFFF',
        },
        text: {
          primary: 'rgba(0, 0, 0, 0.87)',
          secondary: 'rgba(0, 0, 0, 0.6)',
          disabled: 'rgba(0, 0, 0, 0.38)',
        },
        divider: 'rgba(0, 0, 0, 0.12)',
      }
    : {
        // Dark palette
        primary: {
          main: '#42A5F5',
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
        background: {
          default: '#121212',
          paper: '#1E1E1E',
        },
        text: {
          primary: '#FFFFFF',
          secondary: 'rgba(255, 255, 255, 0.7)',
          disabled: 'rgba(255, 255, 255, 0.5)',
        },
        divider: 'rgba(255, 255, 255, 0.12)',
      }),
});

// Typography configuration
const typography = {
  fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
  h1: {
    fontSize: '3rem',
    fontWeight: 300,
    lineHeight: 1.2,
    letterSpacing: '-0.01562em',
  },
  h2: {
    fontSize: '2.125rem',
    fontWeight: 300,
    lineHeight: 1.2,
    letterSpacing: '-0.00833em',
  },
  h3: {
    fontSize: '1.5rem',
    fontWeight: 400,
    lineHeight: 1.167,
    letterSpacing: '0em',
  },
  h4: {
    fontSize: '1.25rem',
    fontWeight: 400,
    lineHeight: 1.235,
    letterSpacing: '0.00735em',
  },
  h5: {
    fontSize: '1.125rem',
    fontWeight: 400,
    lineHeight: 1.334,
    letterSpacing: '0em',
  },
  h6: {
    fontSize: '1rem',
    fontWeight: 500,
    lineHeight: 1.6,
    letterSpacing: '0.0075em',
  },
  body1: {
    fontSize: '0.875rem',
    lineHeight: 1.5,
    letterSpacing: '0.00938em',
  },
  body2: {
    fontSize: '0.75rem',
    lineHeight: 1.43,
    letterSpacing: '0.01071em',
  },
  button: {
    fontSize: '0.875rem',
    fontWeight: 500,
    lineHeight: 1.75,
    letterSpacing: '0.02857em',
    textTransform: 'none' as const,
  },
  caption: {
    fontSize: '0.75rem',
    lineHeight: 1.66,
    letterSpacing: '0.03333em',
  },
  overline: {
    fontSize: '0.625rem',
    fontWeight: 400,
    lineHeight: 2.66,
    letterSpacing: '0.08333em',
    textTransform: 'uppercase' as const,
  },
};

// Shape configuration
const shape = {
  borderRadius: 8,
};

// Shadows for light and dark modes
const getShadows = (mode: PaletteMode) => {
  const shadowKeyUmbraOpacity = mode === 'light' ? 0.08 : 0.4;
  const shadowKeyPenumbraOpacity = mode === 'light' ? 0.12 : 0.5;
  const shadowAmbientShadowOpacity = mode === 'light' ? 0.16 : 0.6;

  const createShadow = (...px: number[]) => {
    return [
      `${px[0]}px ${px[1]}px ${px[2]}px ${px[3]}px rgba(0,0,0,${shadowKeyUmbraOpacity})`,
      `${px[4]}px ${px[5]}px ${px[6]}px ${px[7]}px rgba(0,0,0,${shadowKeyPenumbraOpacity})`,
      `${px[8]}px ${px[9]}px ${px[10]}px ${px[11]}px rgba(0,0,0,${shadowAmbientShadowOpacity})`,
    ].join(',');
  };

  return [
    'none',
    createShadow(0, 2, 1, -1, 0, 1, 1, 0, 0, 1, 3, 0),
    createShadow(0, 3, 1, -2, 0, 2, 2, 0, 0, 1, 5, 0),
    createShadow(0, 3, 3, -2, 0, 3, 4, 0, 0, 1, 8, 0),
    createShadow(0, 2, 4, -1, 0, 4, 5, 0, 0, 1, 10, 0),
    createShadow(0, 3, 5, -1, 0, 5, 8, 0, 0, 1, 14, 0),
    createShadow(0, 3, 5, -1, 0, 6, 10, 0, 0, 1, 18, 0),
    createShadow(0, 4, 5, -2, 0, 7, 10, 1, 0, 2, 16, 1),
    createShadow(0, 5, 5, -3, 0, 8, 10, 1, 0, 3, 14, 2),
    createShadow(0, 5, 6, -3, 0, 9, 12, 1, 0, 3, 16, 2),
    createShadow(0, 6, 6, -3, 0, 10, 14, 1, 0, 4, 18, 3),
    createShadow(0, 6, 7, -4, 0, 11, 15, 1, 0, 4, 20, 3),
    createShadow(0, 7, 8, -4, 0, 12, 17, 2, 0, 5, 22, 4),
    createShadow(0, 7, 8, -4, 0, 13, 19, 2, 0, 5, 24, 4),
    createShadow(0, 7, 9, -4, 0, 14, 21, 2, 0, 5, 26, 4),
    createShadow(0, 8, 9, -5, 0, 15, 22, 2, 0, 6, 28, 5),
    createShadow(0, 8, 10, -5, 0, 16, 24, 2, 0, 6, 30, 5),
    createShadow(0, 8, 11, -5, 0, 17, 26, 2, 0, 6, 32, 5),
    createShadow(0, 9, 11, -5, 0, 18, 28, 2, 0, 7, 34, 6),
    createShadow(0, 9, 12, -6, 0, 19, 29, 2, 0, 7, 36, 6),
    createShadow(0, 10, 13, -6, 0, 20, 31, 3, 0, 8, 38, 7),
    createShadow(0, 10, 13, -6, 0, 21, 33, 3, 0, 8, 40, 7),
    createShadow(0, 10, 14, -6, 0, 22, 35, 3, 0, 8, 42, 7),
    createShadow(0, 11, 14, -7, 0, 23, 36, 3, 0, 9, 44, 8),
    createShadow(0, 11, 15, -7, 0, 24, 38, 3, 0, 9, 46, 8),
  ] as any;
};

// Component overrides
const getComponents = (mode: PaletteMode): ThemeOptions['components'] => ({
  MuiCssBaseline: {
    styleOverrides: {
      body: {
        scrollbarColor: mode === 'light' ? '#9E9E9E #F5F5F5' : '#6E6E6E #2E2E2E',
        '&::-webkit-scrollbar, & *::-webkit-scrollbar': {
          width: 8,
          height: 8,
        },
        '&::-webkit-scrollbar-track, & *::-webkit-scrollbar-track': {
          background: mode === 'light' ? '#F5F5F5' : '#2E2E2E',
        },
        '&::-webkit-scrollbar-thumb, & *::-webkit-scrollbar-thumb': {
          borderRadius: 4,
          background: mode === 'light' ? '#9E9E9E' : '#6E6E6E',
          '&:hover': {
            background: mode === 'light' ? '#757575' : '#8E8E8E',
          },
        },
      },
    },
  },
  MuiButton: {
    styleOverrides: {
      root: {
        borderRadius: 8,
        textTransform: 'none',
        fontWeight: 500,
      },
      containedPrimary: {
        boxShadow: 'none',
        '&:hover': {
          boxShadow: '0px 2px 4px rgba(0, 0, 0, 0.08)',
        },
      },
    },
  },
  MuiCard: {
    styleOverrides: {
      root: {
        borderRadius: 12,
        backgroundImage: 'none',
        transition: 'box-shadow 200ms ease-in-out',
      },
    },
  },
  MuiPaper: {
    styleOverrides: {
      root: {
        backgroundImage: 'none',
      },
    },
  },
  MuiTextField: {
    defaultProps: {
      variant: 'outlined',
      size: 'small',
    },
  },
  MuiTooltip: {
    styleOverrides: {
      tooltip: {
        borderRadius: 4,
        fontSize: '0.75rem',
      },
    },
  },
  MuiChip: {
    styleOverrides: {
      root: {
        borderRadius: 6,
      },
    },
  },
  MuiTableCell: {
    styleOverrides: {
      root: {
        borderBottom: mode === 'light' 
          ? '1px solid rgba(0, 0, 0, 0.08)'
          : '1px solid rgba(255, 255, 255, 0.08)',
      },
    },
  },
});

// Create theme function
export const createGunjTheme = (mode: PaletteMode): ThemeOptions => ({
  palette: getPalette(mode),
  typography,
  shape,
  spacing: 8,
  shadows: getShadows(mode),
  components: getComponents(mode),
  transitions: {
    duration: {
      shortest: 150,
      shorter: 200,
      short: 250,
      standard: 300,
      complex: 375,
      enteringScreen: 225,
      leavingScreen: 195,
    },
    easing: {
      easeInOut: 'cubic-bezier(0.4, 0, 0.2, 1)',
      easeOut: 'cubic-bezier(0.0, 0, 0.2, 1)',
      easeIn: 'cubic-bezier(0.4, 0, 1, 1)',
      sharp: 'cubic-bezier(0.4, 0, 0.6, 1)',
    },
  },
});

// Export pre-configured themes
export const lightTheme = createTheme(createGunjTheme('light'));
export const darkTheme = createTheme(createGunjTheme('dark'));

// Theme context type
export type ThemeMode = 'light' | 'dark' | 'system';

// Custom theme extensions
declare module '@mui/material/styles' {
  interface Theme {
    custom?: {
      semantic: typeof semanticColors;
      metrics: typeof metricsColors;
      brand: typeof brandColors;
    };
  }
  interface ThemeOptions {
    custom?: {
      semantic?: typeof semanticColors;
      metrics?: typeof metricsColors;
      brand?: typeof brandColors;
    };
  }
}

// Add custom properties to themes
[lightTheme, darkTheme].forEach((theme) => {
  theme.custom = {
    semantic: semanticColors,
    metrics: metricsColors,
    brand: brandColors,
  };
});
