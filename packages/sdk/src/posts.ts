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
  UpdatePostRequest,
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
  /**
   * Get a single post by id
   */
  getById(postId: string): Promise<Post>;
  /**
   * Get a post by its URL key (for shareable links)
   */
  getByUrlKey(urlKey: string): Promise<Post>;
  /**
   * Update an existing post
   */
  updatePost(data: UpdatePostRequest): Promise<void>;
  /**
   * Delete a post (soft delete)
   */
  deletePost(postId: string): Promise<void>;

  /**
   * Search posts for autocomplete
   */
  searchPosts(query: string): Promise<Post[]>;

  /**
   * Generate a shareable URL key for a post
   */
  generateUrlKey(postId: string): Promise<{ urlKey: string }>;
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
    if (params?.owner) queryParams.append('owner', params.owner);
    
    const url = `/posts/queries/cursor${queryParams.toString() ? `?${queryParams}` : ''}`;
    return client.get<PostsResponse>(url);
  },

  getById: async (postId: string): Promise<Post> => {
    return client.get<Post>(`/posts/${postId}`);
  },

  getByUrlKey: async (urlKey: string): Promise<Post> => {
    return client.get<Post>(`/posts/urlkey/${urlKey}`);
  },

  updatePost: async (data: UpdatePostRequest): Promise<void> => {
    await client.put<void>('/posts', data);
  },

  deletePost: async (postId: string): Promise<void> => {
    await client.delete<void>(`/posts/${postId}`);
  },

  searchPosts: async (query: string): Promise<Post[]> => {
    const params = new URLSearchParams();
    params.append('q', query);
    params.append('limit', '5');
    const endpoint = `/posts/search?${params.toString()}`;
    return client.get<Post[]>(endpoint);
  },

  generateUrlKey: async (postId: string): Promise<{ urlKey: string }> => {
    return client.put<{ urlKey: string }>(`/posts/urlkey/${postId}`);
  },
});

