'use client';

import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { CreatePostRequest, CursorQueryParams, PostsResponse, Post } from '@telar/sdk';

/**
 * Query keys for posts
 */
export const postsKeys = {
  all: ['posts'] as const,
  lists: () => [...postsKeys.all, 'list'] as const,
  infiniteList: (params?: CursorQueryParams) => [...postsKeys.lists(), 'infinite', { params }] as const,
  detail: (postId: string) => [...postsKeys.all, 'detail', postId] as const,
};

/**
 * Infinite scroll query hook for posts feed
 * 
 * @param params - Optional cursor query parameters (limit, etc.)
 * @returns React Query infinite query result
 */
export function useInfinitePostsQuery(params?: CursorQueryParams) {
  return useInfiniteQuery({
    queryKey: postsKeys.infiniteList(params),
    queryFn: async ({ pageParam }) => {
      const response = await sdk.posts.getPostsWithCursor({
        ...params,
        cursor: pageParam as string | undefined,
        limit: params?.limit || 10,
      });
      
      return response;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage: PostsResponse, allPages: PostsResponse[]) => {
      // Stop pagination if:
      // 1. hasNext is explicitly false (backend indicates no more pages)
      if (lastPage.hasNext === false) {
        return undefined;
      }
      
      // 2. No posts were returned AND no cursor (complete end)
      if (lastPage.posts.length === 0) {
        const nextCursor = lastPage.nextCursor;
        if (!nextCursor || nextCursor.trim() === '') {
          return undefined;
        }
        // If hasNext is true and there's a cursor, continue even with 0 posts
      }
      
      const nextCursor = lastPage.nextCursor;
      // 3. No cursor available (cannot continue)
      if (!nextCursor || nextCursor.trim() === '') {
        return undefined;
      }
      
      // CRITICAL: Always check for same cursor to prevent infinite loops
      // Even if backend says hasNext: true, if the cursor is the same as previous page,
      // we've reached the end. This is the ultimate defensive check.
      if (allPages.length > 1) {
        const previousPage = allPages[allPages.length - 2];
        
        // If previous page had a nextCursor and it matches current nextCursor, we're stuck
        if (previousPage.nextCursor && previousPage.nextCursor === nextCursor) {
          // Same cursor returned consecutively - backend may have incorrect hasNext flag
          // Stop pagination to prevent infinite loop
          return undefined;
        }
      }
      
      return nextCursor;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
  });
}

/**
 * Query hook for a single post
 */
export function usePostQuery(postId: string) {
  return useQuery({
    queryKey: postsKeys.detail(postId),
    queryFn: async (): Promise<Post> => {
      return sdk.posts.getById(postId);
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: Boolean(postId),
  });
}

/**
 * Mutation hook for creating a new post
 * 
 * @returns React Query mutation result
 */
export function useCreatePostMutation() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreatePostRequest) => sdk.posts.createPost(data),
    onSuccess: () => {
      // Invalidate all posts lists so newly created post appears in the feed
      queryClient.invalidateQueries({ queryKey: postsKeys.lists() });
    },
  });
}
