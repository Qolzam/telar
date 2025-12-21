
export const COOKIE_CONFIG = {
  SESSION_NAME: 'access_token',
  MAX_AGE: 7 * 24 * 60 * 60, // 7 days in seconds
  PATH: '/',
  SAME_SITE: 'lax' as const,
  HTTP_ONLY: true,
  SECURE: process.env.NODE_ENV === 'production',
} as const;

/**
 * Create session cookie string
 * 
 * @param token - JWT token to store
 * @returns Cookie string for Set-Cookie header
 */
export function createSessionCookie(token: string): string {
  const parts = [
    `${COOKIE_CONFIG.SESSION_NAME}=${token}`,
    `Max-Age=${COOKIE_CONFIG.MAX_AGE}`,
    `Path=${COOKIE_CONFIG.PATH}`,
    `SameSite=${COOKIE_CONFIG.SAME_SITE}`,
  ];

  if (COOKIE_CONFIG.HTTP_ONLY) {
    parts.push('HttpOnly');
  }

  if (COOKIE_CONFIG.SECURE) {
    parts.push('Secure');
  }

  return parts.join('; ');
}

/**
 * Create cookie deletion string
 * 
 * @returns Cookie string that deletes the session
 */
export function deleteSessionCookie(): string {
  return `${COOKIE_CONFIG.SESSION_NAME}=; Max-Age=0; Path=${COOKIE_CONFIG.PATH}`;
}

/**
 * Parse cookies from header string
 * 
 * @param cookieHeader - Cookie header value
 * @returns Map of cookie name to value
 */
export function parseCookies(cookieHeader: string | null): Map<string, string> {
  const cookies = new Map<string, string>();
  
  if (!cookieHeader) return cookies;

  cookieHeader.split(';').forEach(cookie => {
    const [name, ...rest] = cookie.trim().split('=');
    if (name && rest.length > 0) {
      cookies.set(name, rest.join('='));
    }
  });

  return cookies;
}

/**
 * Get session token from cookies
 * 
 * @param cookieHeader - Cookie header value
 * @returns Session token or null
 */
export function getSessionToken(cookieHeader: string | null): string | null {
  const cookies = parseCookies(cookieHeader);
  return cookies.get(COOKIE_CONFIG.SESSION_NAME) || null;
}
