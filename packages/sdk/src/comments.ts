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
} from './types';

export interface ICommentsApi {
  createComment(data: CreateCommentRequest): Promise<Comment>;
  updateComment(data: UpdateCommentRequest): Promise<Comment>;
  getCommentsByPost(
    postId: string,
    filter?: Partial<Pick<CommentQueryFilter, 'page' | 'limit'>>,
  ): Promise<Comment[]>;
  getComment(commentId: string): Promise<Comment>;
  deleteComment(commentId: string, postId: string): Promise<void>;
  likeComment(commentId: string, delta?: number): Promise<void>;
  getCommentReplies(
    parentCommentId: string,
    filter?: Partial<Pick<CommentQueryFilter, 'page' | 'limit'>>,
  ): Promise<Comment[]>;
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
    filter?: Partial<Pick<CommentQueryFilter, 'page' | 'limit'>>,
  ): Promise<Comment[]> {
    const params = new URLSearchParams();
    params.append('postId', postId);
    if (filter?.page) params.append('page', filter.page.toString());
    if (filter?.limit) params.append('limit', filter.limit.toString());

    const url =
      ENDPOINTS.COMMENTS.GET_BY_POST +
      (params.toString() ? `?${params.toString()}` : '');
    return client.get<Comment[]>(url);
  },

  async getComment(commentId: string): Promise<Comment> {
    const url = ENDPOINTS.COMMENTS.GET_BY_ID(commentId);
    return client.get<Comment>(url);
  },

  async deleteComment(commentId: string, postId: string): Promise<void> {
    const url = ENDPOINTS.COMMENTS.DELETE(commentId, postId);
    await client.delete<void>(url);
  },

  async likeComment(commentId: string, delta = 1): Promise<void> {
    await client.put<void>(ENDPOINTS.COMMENTS.SCORE, {
      commentId,
      delta,
    });
  },

  async getCommentReplies(
    parentCommentId: string,
    filter?: Partial<Pick<CommentQueryFilter, 'page' | 'limit'>>,
  ): Promise<Comment[]> {
    const params = new URLSearchParams();
    if (filter?.page) params.append('page', filter.page.toString());
    if (filter?.limit) params.append('limit', filter.limit.toString());
    const url = `${ENDPOINTS.COMMENTS.GET_BY_ID(parentCommentId)}/replies${
      params.toString() ? `?${params.toString()}` : ''
    }`;
    return client.get<Comment[]>(url);
  },
});


