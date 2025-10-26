import { createTheme } from '@mui/material/styles';
import { Roboto } from 'next/font/google';

// Load Roboto font (MUI's default)
export const roboto = Roboto({
  weight: ['300', '400', '500', '700'],
  subsets: ['latin'],
  display: 'swap',
});

// Colors from template
const COLORS = {
  primary: {
    lighter: '#C8FAD6',
    light: '#5BE49B',
    main: '#00A76F',
    dark: '#007867',
    darker: '#004B50',
    contrastText: '#FFFFFF'
  },
  secondary: {
    lighter: '#EFD6FF',
    light: '#C684FF',
    main: '#8E33FF',
    dark: '#5119B7',
    darker: '#27097A',
    contrastText: '#FFFFFF'
  },
  info: {
    lighter: '#CAFDF5',
    light: '#61F3F3',
    main: '#00B8D9',
    dark: '#006C9C',
    darker: '#003768',
    contrastText: '#FFFFFF'
  },
  success: {
    lighter: '#D3FCD2',
    light: '#77ED8B',
    main: '#22C55E',
    dark: '#118D57',
    darker: '#065E49',
    contrastText: '#ffffff'
  },
  warning: {
    lighter: '#FFF5CC',
    light: '#FFD666',
    main: '#FFAB00',
    dark: '#B76E00',
    darker: '#7A4100',
    contrastText: '#1C252E'
  },
  error: {
    lighter: '#FFE9D5',
    light: '#FFAC82',
    main: '#FF5630',
    dark: '#B71D18',
    darker: '#7A0916',
    contrastText: '#FFFFFF'
  },
  grey: {
    50: '#FCFDFD',
    100: '#F9FAFB',
    200: '#F4F6F8',
    300: '#DFE3E8',
    400: '#C4CDD5',
    500: '#919EAB',
    600: '#637381',
    700: '#454F5B',
    800: '#1C252E',
    900: '#141A21'
  },
  common: {
    black: '#000000',
    white: '#FFFFFF'
  }
};

// MUI v7 Theme with CSS Variables - Updated for proper dark mode support
export const theme = createTheme({
  cssVariables: {
    colorSchemeSelector: 'data-mui-color-scheme',
  },
  colorSchemes: {
    light: {
      palette: {
        primary: COLORS.primary,
        secondary: COLORS.secondary,
        info: COLORS.info,
        success: COLORS.success,
        warning: COLORS.warning,
        error: COLORS.error,
        grey: COLORS.grey,
        common: COLORS.common,
        background: {
          default: '#FAFAFA',
          paper: '#FFFFFF',
        },
        text: {
          primary: '#1C252E',
          secondary: '#637381',
          disabled: '#919EAB',
        },
        divider: 'rgba(145, 158, 171, 0.2)',
        action: {
          hover: 'rgba(145, 158, 171, 0.08)',
          selected: 'rgba(145, 158, 171, 0.16)',
          focus: 'rgba(145, 158, 171, 0.24)',
          disabled: 'rgba(145, 158, 171, 0.8)',
          disabledBackground: 'rgba(145, 158, 171, 0.24)',
          active: '#637381',
        },
      },
    },
    dark: {
      palette: {
        primary: COLORS.primary,
        secondary: COLORS.secondary,
        info: COLORS.info,
        success: COLORS.success,
        warning: COLORS.warning,
        error: COLORS.error,
        grey: COLORS.grey,
        common: COLORS.common,
        background: {
          default: '#141A21',
          paper: '#1C252E',
        },
        text: {
          primary: '#FFFFFF',
          secondary: '#919EAB',
          disabled: '#637381',
        },
        divider: 'rgba(145, 158, 171, 0.2)',
        action: {
          hover: 'rgba(145, 158, 171, 0.08)',
          selected: 'rgba(145, 158, 171, 0.16)',
          focus: 'rgba(145, 158, 171, 0.24)',
          disabled: 'rgba(145, 158, 171, 0.8)',
          disabledBackground: 'rgba(145, 158, 171, 0.24)',
          active: '#919EAB',
        },
      },
    },
  },
  typography: {
    fontFamily: roboto.style.fontFamily,
    h1: {
      fontSize: '2.125rem',
      fontWeight: 400,
      lineHeight: 1.235,
    },
    h2: {
      fontSize: '1.5rem',
      fontWeight: 400,
      lineHeight: 1.334,
    },
    h3: {
      fontSize: '1.25rem',
      fontWeight: 400,
      lineHeight: 1.6,
    },
    h4: {
      fontSize: '1.125rem',
      fontWeight: 400,
      lineHeight: 1.6,
    },
    h5: {
      fontSize: '1rem',
      fontWeight: 400,
      lineHeight: 1.6,
    },
    h6: {
      fontSize: '0.875rem',
      fontWeight: 500,
      lineHeight: 1.6,
    },
    body1: {
      fontSize: '1rem',
      fontWeight: 400,
      lineHeight: 1.5,
    },
    body2: {
      fontSize: '0.875rem',
      fontWeight: 400,
      lineHeight: 1.43,
    },
    caption: {
      fontSize: '0.75rem',
      fontWeight: 400,
      lineHeight: 1.66,
    },
  },
  shape: { borderRadius: 8 },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          backgroundColor: 'var(--mui-palette-background-default)',
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: 'var(--mui-palette-background-paper) !important',
          borderRight: '1px solid',
          borderColor: 'var(--mui-palette-divider) !important',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          borderRadius: 8,
          fontWeight: 500,
        },
        contained: {
          boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
          '&:hover': {
            boxShadow: '0 4px 8px rgba(0,0,0,0.15)',
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 12,
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
          '&:hover': {
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
          },
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          borderRadius: 8,
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 16,
        },
      },
    },
  },
});