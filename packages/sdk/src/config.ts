/**
 * SDK Configuration
 * 
 * Centralized configuration for API endpoints and SDK settings.
 */

/**
 * SDK Configuration object
 */
export const SDK_CONFIG = {
  /**
   * Base URL for Next.js BFF routes (empty for same-origin)
   */
  BFF_BASE_URL: '',

  /**
   * Base URL for Go API (for future direct calls)
   */
  GO_API_BASE_URL: typeof process !== 'undefined' && process.env?.NEXT_PUBLIC_API_URL 
    ? process.env.NEXT_PUBLIC_API_URL 
    : 'http://localhost:9099',

  /**
   * Default request timeout in milliseconds
   */
  TIMEOUT: 10000,
} as const;

/**
 * API Endpoints
 * 
 * Centralized endpoint definitions for all API calls.
 * Auth endpoints call the BFF for security (cookie management).
 * Future endpoints (posts, profile, etc.) will call Go API directly.
 */
export const ENDPOINTS = {
  /**
   * Authentication endpoints (via BFF for httpOnly cookie security)
   */
  AUTH: {
    LOGIN: '/api/auth/login',
    LOGOUT: '/api/auth/logout',
    SIGNUP: '/api/auth/signup',
    SESSION: '/api/auth/session',
    FORGOT_PASSWORD: '/api/auth/forgot-password',
    RESET_PASSWORD: '/api/auth/reset-password',
    CHANGE_PASSWORD: '/api/auth/change-password',
    VERIFY_EMAIL: '/api/auth/verify',
    RESEND_VERIFICATION: '/api/auth/signup/resend',
  },

  /**
   * Profile endpoints (direct Go API calls)
   * These call the Go API directly via NEXT_PUBLIC_API_URL env var
   */
  PROFILE: {
    MY: '/profile/my',
    BY_ID: (userId: string) => `/profile/id/${userId}`,
    BY_SOCIAL_NAME: (socialName: string) => `/profile/social/${socialName}`,
    UPDATE: '/profile',
    BY_IDS: '/profile/ids',
    QUERY: '/profile',
    SEARCH: '/profile/search',
  },

  /**
   * Posts endpoints (direct Go API calls)
   */
  POSTS: {
    SEARCH: '/posts/search',
  },
  /**
   * Comments endpoints (direct Go API calls)
   * Mirrors Go API routes in apps/api/comments/routes.go
   */
  COMMENTS: {
    CREATE: '/comments/',
    UPDATE: '/comments/',
    GET_BY_POST: '/comments/',
    GET_BY_ID: (commentId: string) => `/comments/${commentId}`,
    GET_REPLIES: (commentId: string) => `/comments/${commentId}/replies`,
    DELETE: (commentId: string, postId: string) =>
      `/comments/id/${commentId}/post/${postId}`,
    TOGGLE_LIKE: (commentId: string) => `/comments/${commentId}/like`,
    SCORE: '/comments/score', // Legacy endpoint (deprecated)
  },

  /**
   * Votes endpoints (direct Go API calls)
   * Mirrors Go API routes in apps/api/votes/routes.go
   */
  VOTES: {
    VOTE: '/votes', // POST /votes with { postId, typeId }
  },

  /**
   * Bookmarks endpoints (direct Go API calls)
   * Mirrors Go API routes in apps/api/bookmarks/routes.go
   */
  BOOKMARKS: {
    TOGGLE: (postId: string) => `/bookmarks/${postId}/toggle`,
    LIST: '/bookmarks',
  },

  /**
   * Storage endpoints (direct Go API calls)
   * Mirrors Go API routes in apps/api/storage/routes.go
   */
  STORAGE: {
    INIT: '/storage/upload/init',
    CONFIRM: '/storage/upload/confirm',
    GET_URL: (fileId: string) => `/storage/files/${fileId}/url`,
    DELETE: (fileId: string) => `/storage/files/${fileId}`,
  },
} as const;

