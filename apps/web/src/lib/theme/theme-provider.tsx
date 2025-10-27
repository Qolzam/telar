'use client';

import React from 'react';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider as MuiThemeProvider } from '@mui/material/styles';
import { createAppTheme } from './theme';

// ----------------------------------------------------------------------

export const schemeConfig = {
  modeStorageKey: 'theme-mode',
  defaultMode: 'light' as const,
};

export type ThemeProviderProps = {
  children: React.ReactNode;
  direction?: 'ltr' | 'rtl';
};

export function ThemeProvider({ children, direction = 'ltr' }: ThemeProviderProps): React.ReactElement {
  const theme = createAppTheme(direction);
  
  return (
    <MuiThemeProvider theme={theme} defaultMode={schemeConfig.defaultMode}>
      <CssBaseline enableColorScheme />
      {children}
    </MuiThemeProvider>
  );
}