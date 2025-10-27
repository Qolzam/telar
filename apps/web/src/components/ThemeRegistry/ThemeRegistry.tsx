'use client';

import { ThemeProvider } from '@/lib/theme/theme-provider';
import EmotionCacheProvider from './EmotionCache';

export type ThemeRegistryProps = {
  children: React.ReactNode;
  direction?: 'ltr' | 'rtl';
};

export default function ThemeRegistry({ children, direction = 'ltr' }: ThemeRegistryProps) {
  return (
    <EmotionCacheProvider direction={direction}>
      <ThemeProvider direction={direction}>
        {children}
      </ThemeProvider>
    </EmotionCacheProvider>
  );
}
