'use client';

import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { CreatePostRequest, UpdatePostRequest, CursorQueryParams, PostsResponse, Post } from '@telar/sdk';

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

/**
 * Mutation hook for updating a post with optimistic updates
 */
export function useUpdatePostMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdatePostRequest) => sdk.posts.updatePost(data),
    onMutate: async (variables) => {
      // Cancel any outgoing queries to avoid race conditions
      await queryClient.cancelQueries({ queryKey: postsKeys.lists() });
      await queryClient.cancelQueries({ queryKey: postsKeys.detail(variables.objectId) });

      // Get the previous post value for rollback
      const previousPost = queryClient.getQueryData<Post>(postsKeys.detail(variables.objectId));

      // Optimistically update the post in infinite query cache
      queryClient.setQueriesData<InfiniteData<PostsResponse>>(
        { queryKey: postsKeys.lists() },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              posts: page.posts.map((p) =>
                p.objectId === variables.objectId
                  ? { ...p, ...variables, body: variables.body ?? p.body }
                  : p
              ),
            })),
          };
        }
      );

      // Optimistically update the single post query
      queryClient.setQueryData<Post>(postsKeys.detail(variables.objectId), (old) => {
        if (!old) return old;
        return { ...old, ...variables, body: variables.body ?? old.body };
      });

      return { previousPost };
    },
    onSuccess: (_data, variables) => {
      // Invalidate to ensure we have the latest data from server
      queryClient.invalidateQueries({ queryKey: postsKeys.detail(variables.objectId) });
      queryClient.invalidateQueries({ queryKey: postsKeys.lists() });
    },
    onError: (_error, variables, context) => {
      // Rollback optimistic updates on error
      if (context?.previousPost) {
        queryClient.setQueryData<Post>(postsKeys.detail(variables.objectId), context.previousPost);
      }
      queryClient.invalidateQueries({ queryKey: postsKeys.lists() });
    },
  });
}

/**
 * Mutation hook for deleting a post with optimistic removal
 */
export function useDeletePostMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (postId: string) => sdk.posts.deletePost(postId),
    onMutate: async (postId) => {
      // Cancel any outgoing queries to avoid race conditions
      await queryClient.cancelQueries({ queryKey: postsKeys.lists() });
      await queryClient.cancelQueries({ queryKey: postsKeys.detail(postId) });

      // Get the previous post value for rollback
      const previousPost = queryClient.getQueryData<Post>(postsKeys.detail(postId));

      // Get all pages to find and remove the post
      const allQueries = queryClient.getQueriesData<InfiniteData<PostsResponse>>({
        queryKey: postsKeys.lists(),
      });

      // Store previous state for rollback
      const previousQueries = allQueries.map(([key, data]) => [key, data] as const);

      // Optimistically remove the post from infinite query cache
      queryClient.setQueriesData<InfiniteData<PostsResponse>>(
        { queryKey: postsKeys.lists() },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              posts: page.posts.filter((p) => p.objectId !== postId),
            })),
          };
        }
      );

      // Remove the single post query
      queryClient.removeQueries({ queryKey: postsKeys.detail(postId) });

      return { previousPost, previousQueries };
    },
    onSuccess: () => {
      // Invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: postsKeys.lists() });
    },
    onError: (_error, postId, context) => {
      // Rollback optimistic updates on error
      if (context?.previousQueries) {
        context.previousQueries.forEach(([key, data]) => {
          queryClient.setQueryData(key, data);
        });
      }
      if (context?.previousPost) {
        queryClient.setQueryData<Post>(postsKeys.detail(postId), context.previousPost);
      }
      queryClient.invalidateQueries({ queryKey: postsKeys.lists() });
    },
  });
}
