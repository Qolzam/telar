/**
 * Posts SDK Module
 * 
 * Provides posts functions that call the Go API directly.
 * Posts operations require JWT authentication via headers.
 */

import { ApiClient } from './client';
import type {
  Post,
  CreatePostRequest,
  CreatePostResponse,
  PostsResponse,
  CursorQueryParams,
} from './types';

/**
 * Posts API interface (MVP - minimal functions only)
 */
export interface IPostsApi {
  /**
   * Create a new post
   * Returns objectId - frontend should invalidate cache to refetch
   */
  createPost(data: CreatePostRequest): Promise<CreatePostResponse>;

  /**
   * Get posts with cursor-based pagination
   */
  getPostsWithCursor(params?: CursorQueryParams): Promise<PostsResponse>;
}

/**
 * Create Posts API instance
 */
export const postsApi = (client: ApiClient): IPostsApi => ({
  createPost: async (data: CreatePostRequest): Promise<CreatePostResponse> => {
    return client.post<CreatePostResponse>('/posts', data);
  },

  getPostsWithCursor: async (params?: CursorQueryParams): Promise<PostsResponse> => {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.cursor) queryParams.append('cursor', params.cursor);
    
    const url = `/posts/queries/cursor${queryParams.toString() ? `?${queryParams}` : ''}`;
    return client.get<PostsResponse>(url);
  },
});

