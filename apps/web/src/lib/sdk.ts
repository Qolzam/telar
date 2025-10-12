/**
 * Telar SDK Instance
 * 
 * Singleton instance of the Telar SDK for use throughout the application.
 * 
 * @example
 * ```typescript
 * import { sdk } from '@/lib/sdk';
 * 
 * // Use in React Query hooks
 * const { data } = useQuery({
 *   queryKey: ['session'],
 *   queryFn: () => sdk.auth.getSession(),
 * });
 * ```
 */

import { createTelarSDK } from '@telar/sdk';

/**
 * Shared SDK instance
 */
export const sdk = createTelarSDK();

