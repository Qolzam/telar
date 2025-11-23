/**
 * Telar SDK
 * 
 * Main entry point for the Telar TypeScript SDK.
 * Provides a clean, type-safe interface for all Telar API operations.
 * 
 * @example
 * ```typescript
 * import { createTelarSDK } from '@telar/sdk';
 * 
 * const sdk = createTelarSDK();
 * 
 * // Login
 * await sdk.auth.login({ username: 'user@example.com', password: 'password' });
 * 
 * // Get session
 * const session = await sdk.auth.getSession();
 * ```
 */

export * from './types';
export * from './config';
export { ApiClient, ApiError } from './client';
export type { RequestOptions, ApiClientConfig } from './client';
export { authApi } from './auth';
export type { IAuthApi } from './auth';
export { profileApi } from './profile';
export type { IProfileApi } from './profile';
export { postsApi } from './posts';
export type { IPostsApi } from './posts';
export { commentsApi } from './comments';
export type { ICommentsApi } from './comments';
export { adminApi } from './admin';
export type { IAdminApi } from './admin';
export type { AdminMember, MembersListResponse } from './admin';

import { ApiClient } from './client';
import { SDK_CONFIG } from './config';
import { authApi, IAuthApi } from './auth';
import { profileApi, IProfileApi } from './profile';
import { postsApi, IPostsApi } from './posts';
import { commentsApi, ICommentsApi } from './comments';
import { adminApi, IAdminApi } from './admin';

/**
 * Telar SDK interface
 */
export interface ITelarSDK {
  /**
   * Authentication API
   */
  auth: IAuthApi;

  /**
   * Profile API
   */
  profile: IProfileApi;

  /**
   * Posts API
   */
  posts: IPostsApi;

  /**
   * Comments API
   */
  comments: ICommentsApi;

  /**
   * Admin API
   */
  admin: IAdminApi;
}

/**
 * Create a new instance of the Telar SDK
 * 
 * @param config - Optional configuration overrides
 * @returns Initialized SDK instance
 */
export const createTelarSDK = (): ITelarSDK => {
  // BFF Client for authentication operations (same-origin)
  const bffClient = new ApiClient({
    apiBaseUrl: SDK_CONFIG.GO_API_BASE_URL,
    bffBaseUrl: SDK_CONFIG.BFF_BASE_URL,  // Empty string = same-origin
    timeout: SDK_CONFIG.TIMEOUT,
  });

  // Direct API Client for data operations (Go API)
  const apiClient = new ApiClient({
    apiBaseUrl: SDK_CONFIG.GO_API_BASE_URL,  // Read from NEXT_PUBLIC_API_URL env var
    bffBaseUrl: SDK_CONFIG.BFF_BASE_URL,
    timeout: SDK_CONFIG.TIMEOUT,
  });

  return {
    auth: authApi(bffClient),       // Auth uses BFF (cookie management)
    profile: profileApi(apiClient),
    posts: postsApi(apiClient),
    comments: commentsApi(apiClient),
    admin: adminApi(apiClient),
  };
};

