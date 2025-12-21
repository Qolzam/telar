/**
 * Comments SDK Module
 *
 * Provides comment functions that call the Go API directly.
 */

import { ApiClient } from './client';
import { ENDPOINTS } from './config';
import type {
  Comment,
  CreateCommentRequest,
  UpdateCommentRequest,
  CommentQueryFilter,
  CommentsListResponse,
} from './types';

export interface ICommentsApi {
  createComment(data: CreateCommentRequest): Promise<Comment>;
  updateComment(data: UpdateCommentRequest): Promise<Comment>;
  getCommentsByPost(
    postId: string,
    cursor?: string,
    limit?: number,
  ): Promise<CommentsListResponse>;
  getComment(commentId: string): Promise<Comment>;
  deleteComment(commentId: string, postId: string): Promise<void>;
  toggleLike(commentId: string): Promise<Comment>;
  getCommentReplies(
    parentCommentId: string,
    cursor?: string,
    limit?: number,
  ): Promise<CommentsListResponse>;
}

export const commentsApi = (client: ApiClient): ICommentsApi => ({
  async createComment(data: CreateCommentRequest): Promise<Comment> {
    return client.post<Comment>(ENDPOINTS.COMMENTS.CREATE, data);
  },

  async updateComment(data: UpdateCommentRequest): Promise<Comment> {
    return client.put<Comment>(ENDPOINTS.COMMENTS.UPDATE, data);
  },

  async getCommentsByPost(
    postId: string,
    cursor?: string,
    limit?: number,
  ): Promise<CommentsListResponse> {
    const params = new URLSearchParams();
    params.append('postId', postId);
    
    // Cursor-based pagination only (required for 1M+ users performance)
    if (cursor) {
      params.append('cursor', cursor);
    }
    
    if (limit) {
      params.append('limit', limit.toString());
    }

    const url =
      ENDPOINTS.COMMENTS.GET_BY_POST +
      (params.toString() ? `?${params.toString()}` : '');
    
    return client.get<CommentsListResponse>(url);
  },

  async getComment(commentId: string): Promise<Comment> {
    const url = ENDPOINTS.COMMENTS.GET_BY_ID(commentId);
    return client.get<Comment>(url);
  },

  async deleteComment(commentId: string, postId: string): Promise<void> {
    const url = ENDPOINTS.COMMENTS.DELETE(commentId, postId);
    await client.delete<void>(url);
  },

  async toggleLike(commentId: string): Promise<Comment> {
    const url = ENDPOINTS.COMMENTS.TOGGLE_LIKE(commentId);
    return client.post<Comment>(url, {});
  },

  async getCommentReplies(
    parentCommentId: string,
    cursor?: string,
    limit?: number,
  ): Promise<CommentsListResponse> {
    const params = new URLSearchParams();
    // Cursor-based pagination only (required for 1M+ users performance)
    if (cursor) {
      params.append('cursor', cursor);
    }
    if (limit) {
      params.append('limit', limit.toString());
    }
    const url = `${ENDPOINTS.COMMENTS.GET_REPLIES(parentCommentId)}${
      params.toString() ? `?${params.toString()}` : ''
    }`;
    return client.get<CommentsListResponse>(url);
  },
});


