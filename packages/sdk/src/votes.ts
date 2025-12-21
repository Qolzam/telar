/**
 * Votes SDK Module
 * 
 * Provides vote functions that call the Go API directly.
 * Votes operations require JWT authentication via headers.
 */

import { ApiClient } from './client';
import { ENDPOINTS } from './config';
import type { Post } from './types';

/**
 * Vote request payload
 */
export interface VoteRequest {
  postId: string;
  typeId: 1 | 2; // 1=Up, 2=Down
}

/**
 * Vote response (backend currently returns success message)
 */
export interface VoteResponse {
  message: string;
}

/**
 * Votes API interface
 */
export interface IVotesApi {
  /**
   * Vote on a post (Up or Down)
   * @param postId - The post ID to vote on
   * @param typeId - Vote type: 1=Up, 2=Down (sending same type toggles off)
   * @returns Success message (Post should be refetched via query invalidation)
   * @todo Backend should return updated Post object for better UX
   */
  vote(postId: string, typeId: 1 | 2): Promise<VoteResponse>;
}

/**
 * Create Votes API instance
 */
export const votesApi = (client: ApiClient): IVotesApi => ({
  vote: async (postId: string, typeId: 1 | 2): Promise<VoteResponse> => {
    return client.post<VoteResponse>(ENDPOINTS.VOTES.VOTE, {
      postId,
      typeId,
    });
  },
});

