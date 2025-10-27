/**
 * Get localized path helper
 * Note: We're using cookie-based locale detection, so paths remain unchanged
 * This is kept for future reference if we switch to URL-based locale routing
 */
export function useLocalizedPath(path: string): string {
  // With cookie-based approach, return path as-is
  return path;
}
