'use client';

import { useColorScheme } from '@mui/material/styles';

export type ColorScheme = 'light' | 'dark' | 'system';

export function useTheme() {
  const { mode, setMode } = useColorScheme();
  
  const toggleTheme = () => {
    if (mode === 'light') {
      setMode('dark');
    } else if (mode === 'dark') {
      setMode('system');
    } else {
      setMode('light');
    }
  };

  return {
    colorScheme: mode,
    setColorScheme: setMode,
    resolvedMode: mode === 'system' ? 'light' : mode, // For system mode, we'll let MUI handle it
    toggleTheme,
  };
}

/**
 * Hook to get the current resolved theme mode
 * 
 * @returns The actual theme mode being used (light or dark)
 * 
 * @example
 * ```tsx
 * function ThemeIndicator() {
 *   const mode = useThemeMode();
 *   
 *   return (
 *     <span>Current mode: {mode}</span>
 *   );
 * }
 * ```
 */
export function useThemeMode(): 'light' | 'dark' {
  const { resolvedMode } = useTheme();
  return resolvedMode ?? 'light';
}

/**
 * Hook to check if the current theme is dark
 * 
 * @returns True if the current theme is dark
 * 
 * @example
 * ```tsx
 * function DarkModeIndicator() {
 *   const isDark = useIsDarkMode();
 *   
 *   return (
 *     <span>{isDark ? 'Dark' : 'Light'} mode</span>
 *   );
 * }
 * ```
 */
export function useIsDarkMode(): boolean {
  const mode = useThemeMode();
  return mode === 'dark';
}

/**
 * Hook to get theme-aware color values
 * 
 * @returns Object with theme-aware color values
 * 
 * @example
 * ```tsx
 * function ThemedComponent() {
 *   const colors = useThemeColors();
 *   
 *   return (
 *     <div style={{ 
 *       backgroundColor: colors.background,
 *       color: colors.text 
 *     }}>
 *       Themed content
 *     </div>
 *   );
 * }
 * ```
 */
export function useThemeColors() {
  const isDark = useIsDarkMode();
  
  return {
    background: isDark ? '#141A21' : '#FFFFFF',
    paper: isDark ? '#1C252E' : '#FFFFFF',
    text: isDark ? '#FFFFFF' : '#1C252E',
    textSecondary: isDark ? '#919EAB' : '#637381',
    primary: '#00A76F',
    secondary: '#8E33FF',
  };
}