/**
 * Bookmarks SDK Module
 * 
 * Provides bookmark functions that call the Go API directly.
 * Bookmark operations require JWT authentication via cookies.
 */

import { ApiClient } from './client';
import { ENDPOINTS } from './config';
import type { PostsResponse, CursorQueryParams } from './types';

/**
 * Toggle bookmark response
 */
export interface ToggleBookmarkResponse {
  isBookmarked: boolean;
}

/**
 * Bookmarks API interface
 */
export interface IBookmarksApi {
  /**
   * Toggle bookmark state for a post
   * @param postId - The post ID to bookmark/unbookmark
   * @returns Current bookmark state after toggle
   */
  toggleBookmark(postId: string): Promise<ToggleBookmarkResponse>;

  /**
   * Get paginated list of bookmarked posts
   * @param params - Optional cursor query parameters (cursor, limit)
   * @returns Paginated posts response
   */
  getBookmarks(params?: CursorQueryParams): Promise<PostsResponse>;
}

/**
 * Create Bookmarks API instance
 */
export const bookmarksApi = (client: ApiClient): IBookmarksApi => ({
  toggleBookmark: async (postId: string): Promise<ToggleBookmarkResponse> => {
    return client.post<ToggleBookmarkResponse>(ENDPOINTS.BOOKMARKS.TOGGLE(postId));
  },

  getBookmarks: async (params?: CursorQueryParams): Promise<PostsResponse> => {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.cursor) queryParams.append('cursor', params.cursor);
    
    const url = queryParams.toString() 
      ? `${ENDPOINTS.BOOKMARKS.LIST}?${queryParams}` 
      : ENDPOINTS.BOOKMARKS.LIST;
    return client.get<PostsResponse>(url);
  },
});

