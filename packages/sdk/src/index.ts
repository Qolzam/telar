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

import { ApiClient } from './client';
import { SDK_CONFIG } from './config';
import { authApi, IAuthApi } from './auth';

/**
 * Telar SDK interface
 */
export interface ITelarSDK {
  /**
   * Authentication API
   */
  auth: IAuthApi;

  // Future APIs will be added here:
  // posts: IPostsApi;
  // profile: IProfileApi;
  // comments: ICommentsApi;
}

/**
 * Create a new instance of the Telar SDK
 * 
 * @param config - Optional configuration overrides
 * @returns Initialized SDK instance
 */
export const createTelarSDK = (): ITelarSDK => {
  const client = new ApiClient({
    baseUrl: SDK_CONFIG.BFF_BASE_URL,
    timeout: SDK_CONFIG.TIMEOUT,
  });

  return {
    auth: authApi(client),
  };
};

