/**
 * Authentication Types
 * 
 * Type definitions for authentication flows, matching Go API responses
 * 
 * Reference: apps/api/auth/models/auth.go - TokenClaim struct
 * Reference: apps/api/auth/login/handler.go - LoginResponse format
 */

/**
 * Token claim extracted from JWT
 * Matches Go API's TokenClaim struct (models/auth.go line 94-103)
 */
export interface TokenClaim {
  displayName: string;
  socialName: string;
  email: string;
  uid: string;
  role: string;
  createdDate: number;
  avatar?: string;
  banner?: string;
  tagLine?: string;
  custom?: Record<string, unknown>;
  // Standard JWT claims
  iss?: string;  // Issuer
  sub?: string;  // Subject
  aud?: string;  // Audience
  exp?: number;  // Expiration time
  nbf?: number;  // Not before
  iat?: number;  // Issued at
  jti?: string;  // JWT ID
}

/**
 * User profile data
 * Matches Go API's UserProfile struct
 */
export interface UserProfile {
  objectId: string;
  fullName: string;
  socialName: string;
  email: string;
  avatar: string;
  banner: string;
  tagLine: string;
  createdDate: number;
}

/**
 * Session data returned to the frontend
 * Sanitized version of TokenClaim
 */
export interface SessionData {
  user: {
    id: string;
    displayName: string;
    socialName: string;
    email: string;
    role: string;
    avatar?: string;
    banner?: string;
    tagLine?: string;
    createdDate: number;
  };
  isAuthenticated: boolean;
}

/**
 * Login request payload
 * Matches Go API's LoginRequest struct (login/model.go)
 */
export interface LoginRequest {
  username: string;  // Email address
  password: string;
}

/**
 * Login response from Go API
 * Matches Go API's response format (login/handler.go line 115-120)
 */
export interface GoApiLoginResponse {
  user: UserProfile;     // User profile data
  accessToken: string;   // JWT token
  tokenType: string;     // "Bearer"
  expires_in: string;    // Expiration as string
}

/**
 * API error response structure
 */
export interface ApiErrorResponse {
  error: string;
  message: string;
  statusCode: number;
}

/**
 * JWKS key structure for ES256 (ECDSA)
 * Reference: apps/api/auth/jwks/handler.go - EC key format
 */
export interface JWK {
  kty: string;  // "EC" for ECDSA keys
  kid: string;  // Key ID
  use: string;  // "sig" for signature
  alg: string;  // "ES256" - ECDSA with SHA-256
  crv: string;  // "P-256" - Curve name
  x: string;    // X coordinate (base64url)
  y: string;    // Y coordinate (base64url)
}

/**
 * JWKS response from Go API
 */
export interface JWKS {
  keys: JWK[];
}
