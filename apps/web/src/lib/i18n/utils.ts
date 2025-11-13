import { rtlLanguages } from './settings';

/**
 * Check if a locale uses Right-to-Left (RTL) text direction
 * @param locale - The locale code (e.g., 'ar', 'fa')
 * @returns true if the locale is RTL
 */
export function isRTL(locale: string): boolean {
  return rtlLanguages.includes(locale);
}

/**
 * Get the text direction for a locale
 * @param locale - The locale code
 * @returns 'rtl' for RTL languages, 'ltr' for LTR languages
 */
export function getDirection(locale: string): 'rtl' | 'ltr' {
  return isRTL(locale) ? 'rtl' : 'ltr';
}
