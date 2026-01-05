import { jwtVerify, importSPKI } from 'jose';
import type { TokenClaim } from '@telar/sdk';
import { COOKIE_CONFIG } from './cookies';

// Cache the public key to avoid re-importing on every request
let cachedPublicKey: Awaited<ReturnType<typeof importSPKI>> | null = null;

/**
 * Get the public key for JWT verification
 * Uses static key from environment variable (zero network latency)
 */
async function getPublicKey() {
  if (cachedPublicKey) {
    return cachedPublicKey;
  }

  const pem = process.env.AUTH_PUBLIC_KEY || process.env.NEXT_PUBLIC_AUTH_PUBLIC_KEY;
  if (!pem) {
    throw new Error('AUTH_PUBLIC_KEY or NEXT_PUBLIC_AUTH_PUBLIC_KEY environment variable is required');
  }

  // Import the PEM-encoded ECDSA public key (ES256 algorithm)
  cachedPublicKey = await importSPKI(pem, 'ES256');
  return cachedPublicKey;
}

/**
 * Verify JWT token and extract claims
 * Uses static public key (no network calls, Edge Runtime compatible)
 * 
 * @param token - JWT token string
 * @returns TokenClaim if valid, null if invalid
 */
export async function verifyToken(token: string): Promise<TokenClaim | null> {
  try {
    const publicKey = await getPublicKey();
    const { payload } = await jwtVerify(token, publicKey, {
      issuer: 'telar-social@telar', // Matches backend: "telar-social@" + providerName where providerName="telar"
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
