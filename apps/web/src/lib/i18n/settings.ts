/**
 * Default fallback language when locale detection fails
 */
export const fallbackLng = 'en';

/**
 * List of all supported languages
 */
export const languages = ['en', 'fa', 'zh', 'ar', 'es', 'fr'] as const;

/**
 * Default namespace for translations
 */
export const defaultNS = 'common';

/**
 * Cookie name used for locale persistence
 */
export const cookieName = 'i18next';

/**
 * Human-readable names for each language
 */
export const languageNames: Record<string, string> = {
  en: 'English',
  fa: 'فارسی', // Persian
  zh: '中文', // Chinese
  ar: 'العربية', // Arabic
  es: 'Español', // Spanish
  fr: 'Français', // French
};

/**
 * Right-to-Left (RTL) languages that require layout mirroring
 */
export const rtlLanguages = ['ar', 'fa'];
