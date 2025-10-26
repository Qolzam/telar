'use client';

import React from 'react';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider as MuiThemeProvider } from '@mui/material/styles';
import { theme } from './theme';

// ----------------------------------------------------------------------

export const schemeConfig = {
  modeStorageKey: 'theme-mode',
  defaultMode: 'light' as const,
};

export function ThemeProvider({ children }: { children: React.ReactNode }): React.ReactElement {
  return (
    <MuiThemeProvider theme={theme} defaultMode={schemeConfig.defaultMode}>
      <CssBaseline enableColorScheme />
      {children}
    </MuiThemeProvider>
  );
}