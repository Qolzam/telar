import { jwtVerify, createRemoteJWKSet } from 'jose';
import type { TokenClaim, JWKS } from '@telar/sdk';
import { COOKIE_CONFIG } from './cookies';

const getAuthApiUrl = () => {
  const url = process.env.INTERNAL_API_URL || 'http://localhost:8080';
  return url.replace('localhost', '127.0.0.1');
};

const AUTH_API_URL = getAuthApiUrl();
const JWKS_URL = `${AUTH_API_URL}/auth/.well-known/jwks.json`;

const JWKS = createRemoteJWKSet(new URL(JWKS_URL));

/**
 * Verify JWT token and extract claims
 * 
 * @param token - JWT token string
 * @returns TokenClaim if valid, null if invalid
 */
export async function verifyToken(token: string): Promise<TokenClaim | null> {
  try {
    const { payload } = await jwtVerify(token, JWKS, {
      issuer: 'telar-social@telar', 
      audience: '', 
    });

    const claimData = payload.claim as Record<string, unknown>;
    
    const claim: TokenClaim = {
      displayName: claimData.displayName as string,
      socialName: claimData.socialName as string,
      email: claimData.email as string,
      uid: claimData.uid as string,
      role: claimData.role as string,
      createdDate: claimData.createdDate as number,
      avatar: claimData.avatar as string | undefined,
      banner: claimData.banner as string | undefined,
      tagLine: claimData.tagLine as string | undefined,
      custom: claimData.custom as Record<string, unknown> | undefined,
      iss: payload.iss,
      sub: payload.sub,
      aud: payload.aud as string | undefined,
      exp: payload.exp,
      nbf: payload.nbf,
      iat: payload.iat,
      jti: payload.jti,
    };

    return claim;
  } catch (error) {
    console.error('[JWT] Token verification failed:', error);
    return null;
  }
}

/**
 * Check if token is expired
 * 
 * @param claim - Token claim
 * @returns true if expired
 */
export function isTokenExpired(claim: TokenClaim): boolean {
  if (!claim.exp) return false;
  return Date.now() >= claim.exp * 1000;
}

/**
 * Extract token from cookie header
 * 
 * @param cookieHeader - Cookie header string
 * @param cookieName - Name of the session cookie
 * @returns Token string or null
 */
export function extractTokenFromCookies(
  cookieHeader: string | null,
  cookieName: string = COOKIE_CONFIG.SESSION_NAME
): string | null {
  if (!cookieHeader) return null;

  const cookies = cookieHeader.split(';').map(c => c.trim());
  const sessionCookie = cookies.find(c => c.startsWith(`${cookieName}=`));
  
  if (!sessionCookie) return null;
  
  return sessionCookie.substring(cookieName.length + 1);
}
