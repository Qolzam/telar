'use client';

import { useEffect, useState } from 'react';
import { initI18nClient } from '@/lib/i18n/client';
import { fallbackLng } from '@/lib/i18n/settings';

interface I18nProviderProps {
  children: React.ReactNode;
}

export function I18nProvider({ children }: I18nProviderProps) {
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    const initI18n = async () => {
      // Get locale from cookie (same as server-side)
      const getCookieValue = (name: string): string | null => {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) {
          return parts.pop()?.split(';').shift() || null;
        }
        return null;
      };

      const locale = getCookieValue('i18next') || fallbackLng;

      // Initialize i18n with the locale
      await initI18nClient(locale, ['common', 'auth', 'settings', 'validation', 'dashboard', 'profile', 'posts']);
      setIsReady(true);
    };

    initI18n();
  }, []);

  // Show nothing while initializing to avoid hydration mismatch
  if (!isReady) {
    return null;
  }

  return <>{children}</>;
}
