'use client';

import { AppThemeProvider } from '@/lib/theme/theme-provider';
import EmotionCacheProvider from './EmotionCache';

export type ThemeRegistryProps = {
  children: React.ReactNode;
  direction?: 'ltr' | 'rtl';
};

export default function ThemeRegistry({ children, direction = 'ltr' }: ThemeRegistryProps) {
  return (
    <EmotionCacheProvider direction={direction}>
      <AppThemeProvider direction={direction}>
        {children}
      </AppThemeProvider>
    </EmotionCacheProvider>
  );
}
