'use client';

import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { CreatePostRequest, CursorQueryParams, PostsResponse } from '@telar/sdk';

/**
 * Query keys for posts
 */
export const postsKeys = {
  all: ['posts'] as const,
  lists: () => [...postsKeys.all, 'list'] as const,
  infiniteList: (params?: CursorQueryParams) => [...postsKeys.lists(), 'infinite', { params }] as const,
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
      console.log('[POSTS_QUERY] 🔵 Fetching posts:', {
        pageParam: pageParam || 'undefined (first page)',
        limit: params?.limit || 10,
        timestamp: new Date().toISOString(),
      });
      
      const response = await sdk.posts.getPostsWithCursor({
        ...params,
        cursor: pageParam as string | undefined,
        limit: params?.limit || 10,
      });
      
      console.log('[POSTS_QUERY] ✅ Received response:', {
        postsCount: response.posts.length,
        hasNext: response.hasNext,
        nextCursor: response.nextCursor || 'undefined',
        postIds: response.posts.map(p => p.objectId).slice(0, 5), // First 5 IDs
        timestamp: new Date().toISOString(),
      });
      
      return response;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage: PostsResponse, allPages: PostsResponse[]) => {
      console.log('[POSTS_QUERY] 🔍 getNextPageParam called:', {
        currentPageIndex: allPages.length - 1,
        totalPages: allPages.length,
        lastPage: {
          postsCount: lastPage.posts.length,
          hasNext: lastPage.hasNext,
          nextCursor: lastPage.nextCursor || 'undefined',
        },
        allPagesSummary: allPages.map((page, idx) => ({
          pageIndex: idx,
          postsCount: page.posts.length,
          hasNext: page.hasNext,
          nextCursor: page.nextCursor || 'undefined',
        })),
        timestamp: new Date().toISOString(),
      });
      
      // Stop pagination if:
      // 1. hasNext is explicitly false (backend indicates no more pages)
      if (lastPage.hasNext === false) {
        console.log('[POSTS_QUERY] ⛔ Stopping: hasNext === false');
        return undefined;
      }
      
      // 2. No posts were returned AND no cursor (complete end)
      if (lastPage.posts.length === 0) {
        const nextCursor = lastPage.nextCursor;
        if (!nextCursor || nextCursor.trim() === '') {
          console.log('[POSTS_QUERY] ⛔ Stopping: No posts and no cursor');
          return undefined;
        }
        console.log('[POSTS_QUERY] ⚠️ No posts but cursor exists, continuing...');
        // If hasNext is true and there's a cursor, continue even with 0 posts
      }
      
      const nextCursor = lastPage.nextCursor;
      // 3. No cursor available (cannot continue)
      if (!nextCursor || nextCursor.trim() === '') {
        console.log('[POSTS_QUERY] ⛔ Stopping: No cursor available');
        return undefined;
      }
      
      // CRITICAL: Always check for same cursor to prevent infinite loops
      // Even if backend says hasNext: true, if the cursor is the same as previous page,
      // we've reached the end. This is the ultimate defensive check.
      if (allPages.length > 1) {
        const previousPage = allPages[allPages.length - 2];
        console.log('[POSTS_QUERY] 🔄 Checking for same cursor:', {
          previousPageIndex: allPages.length - 2,
          previousNextCursor: previousPage.nextCursor || 'undefined',
          currentNextCursor: nextCursor,
          cursorsMatch: previousPage.nextCursor === nextCursor,
        });
        
        // If previous page had a nextCursor and it matches current nextCursor, we're stuck
        if (previousPage.nextCursor && previousPage.nextCursor === nextCursor) {
          // Same cursor returned consecutively - backend may have incorrect hasNext flag
          // Stop pagination to prevent infinite loop
          console.log('[POSTS_QUERY] ⛔ Stopping: Same cursor detected (infinite loop prevention)');
          return undefined;
        }
      }
      
      console.log('[POSTS_QUERY] ✅ Returning nextCursor:', nextCursor);
      return nextCursor;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
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
