'use client';

import { ThemeProvider } from '@/lib/theme/theme-provider';
import EmotionCacheProvider from './EmotionCache';

export default function ThemeRegistry({ children }: { children: React.ReactNode }) {
  return (
    <EmotionCacheProvider>
      <ThemeProvider>
        {children}
      </ThemeProvider>
    </EmotionCacheProvider>
  );
}
