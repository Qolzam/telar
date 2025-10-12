/**
 * Telar SDK Type Definitions
 * 
 * This file contains all TypeScript types for the Telar API.
 * Types are organized by domain and match Go backend structs.
 */

// ============================================================================
// Common / Foundation Types
// ============================================================================

/**
 * User permission levels for content visibility
 * @see Go: UserPermissionType enum
 */
export enum UserPermissionType {
  OnlyMe = 'OnlyMe',
  Public = 'Public',
  Circles = 'Circles',
  Custom = 'Custom',
}

/**
 * Base domain interface for entities with common fields
 */
export interface BaseDomain {
  objectId?: string;
  created_date?: number;
  last_updated?: number;
}

// ============================================================================
// Authentication Types
// ============================================================================

/**
 * Token claim extracted from JWT
 * @see Go: apps/api/auth/models/auth.go - TokenClaim struct
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
  iss?: string;
  sub?: string;
  aud?: string;
  exp?: number;
  nbf?: number;
  iat?: number;
  jti?: string;
}

/**
 * User profile data returned from API
 * @see Go: UserProfile struct
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
 * Complete user profile model with all fields
 * @see Go: UserProfileModel struct
 */
export interface UserProfileModel {
  objectId: string;
  fullName: string;
  socialName: string;
  avatar: string;
  banner: string;
  tagLine: string;
  created_date: number;
  last_updated: number;
  email: string;
  birthday: number;
  webUrl: string;
  companyName: string;
  voteCount: number;
  shareCount: number;
  followCount: number;
  followerCount: number;
  postCount: number;
  facebookId: string;
  instagramId: string;
  twitterId: string;
  accessUserList: string[];
  permission: UserPermissionType;
}

/**
 * Session data returned to the frontend
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
 * @see Go: apps/api/auth/login/model.go - LoginRequest
 */
export interface LoginRequest {
  username: string;
  password: string;
}

/**
 * Login response from Go API
 * @see Go: apps/api/auth/login/handler.go - LoginResponse
 */
export interface GoApiLoginResponse {
  user: UserProfile;
  accessToken: string;
  tokenType: string;
  expires_in: string;
}

/**
 * Signup request payload
 * @see Go: apps/api/auth/signup/model.go
 */
export interface SignupRequest {
  fullName: string;
  email: string;
  newPassword: string;
}

/**
 * Signup response
 */
export interface SignupResponse {
  verificationId: string;
  message: string;
}

/**
 * Forgot password request
 */
export interface ForgotPasswordRequest {
  email: string;
}

/**
 * Reset password request
 */
export interface ResetPasswordRequest {
  token: string;
  newPassword: string;
}

/**
 * Change password request
 */
export interface ChangePasswordRequest {
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
}

/**
 * Email verification request
 */
export interface VerifyEmailRequest {
  code: string;
  verificationId: string;
}

/**
 * JWKS key structure for ES256 (ECDSA)
 * @see Go: apps/api/auth/jwks/handler.go
 */
export interface JWK {
  kty: string;
  kid: string;
  use: string;
  alg: string;
  crv: string;
  x: string;
  y: string;
}

/**
 * JWKS response
 */
export interface JWKS {
  keys: JWK[];
}

// ============================================================================
// User Types
// ============================================================================

/**
 * User entity
 * @see Go: User struct
 */
export interface User {
  objectId?: string;
  avatar?: string;
  fullName?: string;
  socialName?: string;
  banner?: string;
  tagLine?: string;
  creationDate?: number;
  email?: string;
  birthday?: number;
  webUrl?: string;
  companyName?: string;
  country?: string;
  school?: string;
  address?: string;
  location?: string;
  phone?: number;
  voteCount?: number;
  shareCount?: number;
  followCount?: number;
  followerCount?: number;
  postCount?: number;
  userId?: string;
  twitterId?: string;
  facebookId?: string;
  instagramId?: string;
  linkedInId?: string;
  accessUserList?: string[];
  permission?: UserPermissionType;
}

// ============================================================================
// Post Types
// ============================================================================

/**
 * Post type enumeration
 */
export enum PostType {
  Text = 0,
  Photo = 1,
  Video = 2,
  PhotoGallery = 3,
  Album = 4,
}

/**
 * Post album for media posts
 */
export interface PostAlbum {
  count: number;
  cover: string;
  coverId: string;
  photos: string[];
  title: string;
}

/**
 * Post model
 * @see Go: Post struct
 */
export interface Post {
  objectId: string;
  ownerUserId: string;
  ownerDisplayName: string;
  ownerAvatar: string;
  body: string;
  image?: string;
  imageFullPath?: string;
  video?: string;
  thumbnail?: string;
  album?: PostAlbum;
  score: number;
  commentCounter: number;
  viewCount: number;
  votes: { [userId: string]: boolean };
  tags: string[];
  postTypeId: PostType;
  permission: UserPermissionType;
  accessUserList: string[];
  disableComments: boolean;
  disableSharing: boolean;
  deleted: boolean;
  deletedDate?: number;
  urlKey?: string;
  version?: string;
  created_date: number;
  last_updated: number;
}

// ============================================================================
// File Types
// ============================================================================

/**
 * File upload/download result
 */
export interface FileResult {
  url?: string;
  fileName?: string;
  fileSize?: number;
}

// ============================================================================
// API Error Types
// ============================================================================

/**
 * API error response structure
 */
export interface ApiErrorResponse {
  error: string;
  message: string;
  statusCode: number;
}

