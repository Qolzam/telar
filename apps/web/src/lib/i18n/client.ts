'use client';

import i18next from 'i18next';
import { initReactI18next, useTranslation as useRT } from 'react-i18next';
import resourcesToBackend from 'i18next-resources-to-backend';
import { languages, fallbackLng, defaultNS } from './settings';

let initialized = false;

/**
 * Import translation files for a specific language and namespace
 * Uses static imports for Turbopack compatibility
 * @param lng - Language code
 * @param ns - Namespace name
 * @returns Translation object
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const importTranslations = async (lng: string, ns: string): Promise<Record<string, any>> => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const translations: Record<string, any> = {};
  
  try {
    switch (lng) {
      case 'en':
        if (ns === 'common') translations.common = await import('../../../public/locales/en/common.json').then(m => m.default);
        if (ns === 'auth') translations.auth = await import('../../../public/locales/en/auth.json').then(m => m.default);
        if (ns === 'settings') translations.settings = await import('../../../public/locales/en/settings.json').then(m => m.default);
        if (ns === 'validation') translations.validation = await import('../../../public/locales/en/validation.json').then(m => m.default);
        if (ns === 'dashboard') translations.dashboard = await import('../../../public/locales/en/dashboard.json').then(m => m.default);
        if (ns === 'profile') translations.profile = await import('../../../public/locales/en/profile.json').then(m => m.default);
        break;
      case 'fa':
        if (ns === 'common') translations.common = await import('../../../public/locales/fa/common.json').then(m => m.default);
        if (ns === 'auth') translations.auth = await import('../../../public/locales/fa/auth.json').then(m => m.default);
        if (ns === 'settings') translations.settings = await import('../../../public/locales/fa/settings.json').then(m => m.default);
        if (ns === 'validation') translations.validation = await import('../../../public/locales/fa/validation.json').then(m => m.default);
        if (ns === 'dashboard') translations.dashboard = await import('../../../public/locales/fa/dashboard.json').then(m => m.default);
        if (ns === 'profile') translations.profile = await import('../../../public/locales/fa/profile.json').then(m => m.default);
        break;
      case 'zh':
        if (ns === 'common') translations.common = await import('../../../public/locales/zh/common.json').then(m => m.default);
        if (ns === 'auth') translations.auth = await import('../../../public/locales/zh/auth.json').then(m => m.default);
        if (ns === 'settings') translations.settings = await import('../../../public/locales/zh/settings.json').then(m => m.default);
        if (ns === 'validation') translations.validation = await import('../../../public/locales/zh/validation.json').then(m => m.default);
        if (ns === 'dashboard') translations.dashboard = await import('../../../public/locales/zh/dashboard.json').then(m => m.default);
        if (ns === 'profile') translations.profile = await import('../../../public/locales/zh/profile.json').then(m => m.default);
        break;
      case 'ar':
        if (ns === 'common') translations.common = await import('../../../public/locales/ar/common.json').then(m => m.default);
        if (ns === 'auth') translations.auth = await import('../../../public/locales/ar/auth.json').then(m => m.default);
        if (ns === 'settings') translations.settings = await import('../../../public/locales/ar/settings.json').then(m => m.default);
        if (ns === 'validation') translations.validation = await import('../../../public/locales/ar/validation.json').then(m => m.default);
        if (ns === 'dashboard') translations.dashboard = await import('../../../public/locales/ar/dashboard.json').then(m => m.default);
        if (ns === 'profile') translations.profile = await import('../../../public/locales/ar/profile.json').then(m => m.default);
        break;
      case 'es':
        if (ns === 'common') translations.common = await import('../../../public/locales/es/common.json').then(m => m.default);
        if (ns === 'auth') translations.auth = await import('../../../public/locales/es/auth.json').then(m => m.default);
        if (ns === 'settings') translations.settings = await import('../../../public/locales/es/settings.json').then(m => m.default);
        if (ns === 'validation') translations.validation = await import('../../../public/locales/es/validation.json').then(m => m.default);
        if (ns === 'dashboard') translations.dashboard = await import('../../../public/locales/es/dashboard.json').then(m => m.default);
        if (ns === 'profile') translations.profile = await import('../../../public/locales/es/profile.json').then(m => m.default);
        break;
      case 'fr':
        if (ns === 'common') translations.common = await import('../../../public/locales/fr/common.json').then(m => m.default);
        if (ns === 'auth') translations.auth = await import('../../../public/locales/fr/auth.json').then(m => m.default);
        if (ns === 'settings') translations.settings = await import('../../../public/locales/fr/settings.json').then(m => m.default);
        if (ns === 'validation') translations.validation = await import('../../../public/locales/fr/validation.json').then(m => m.default);
        if (ns === 'dashboard') translations.dashboard = await import('../../../public/locales/fr/dashboard.json').then(m => m.default);
        if (ns === 'profile') translations.profile = await import('../../../public/locales/fr/profile.json').then(m => m.default);
        break;
    }
  } catch (error) {
    console.warn(`Failed to load translation for ${lng}/${ns}:`, error);
  }
  
  return translations;
};

/**
 * Initialize i18next for client-side usage
 * Sets up react-i18next and loads translation resources
 * @param locale - The locale to initialize with
 * @param namespaces - List of translation namespaces to load
 */
export async function initI18nClient(locale: string, namespaces: string[] = [defaultNS]) {
  if (!languages.includes(locale as typeof languages[number])) {
    locale = fallbackLng;
  }

  if (!initialized) {
    i18next
      .use(initReactI18next)
      .use(
        resourcesToBackend(async (lng: string, ns: string) => {
          const translations = await importTranslations(lng, ns);
          return translations[ns] || {};
        })
      );

    await i18next.init({
      lng: locale,
      fallbackLng,
      ns: namespaces,
      defaultNS,
      interpolation: { 
        escapeValue: false 
      },
      supportedLngs: languages,
    });

    initialized = true;
  } else if (i18next.language !== locale) {
    await i18next.changeLanguage(locale);
  }
}

/**
 * Hook for using translations in client components
 * @param ns - Namespace to use (defaults to 'common')
 * @returns react-i18next useTranslation hook
 */
export function useClientTranslation(ns: string = defaultNS) {
  return useRT(ns);
}

export default i18next;
