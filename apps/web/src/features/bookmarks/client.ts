'use client';

import { useInfiniteQuery, useMutation, useQueryClient, type InfiniteData } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { Post, PostsResponse, CursorQueryParams } from '@telar/sdk';
import { postsKeys } from '@/features/posts/client';

/**
 * Query keys for bookmarks
 */
export const bookmarkKeys = {
  all: ['bookmarks'] as const,
  list: () => [...bookmarkKeys.all, 'list'] as const,
  infiniteList: (params?: CursorQueryParams) => [...bookmarkKeys.list(), 'infinite', { params }] as const,
};


export function useBookmarkMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ postId }: { postId: string }) => {
      return sdk.bookmarks.toggleBookmark(postId);
    },
    onMutate: async ({ postId }) => {
      // Cancel any outgoing queries to avoid race conditions
      await queryClient.cancelQueries({ queryKey: postsKeys.lists() });
      await queryClient.cancelQueries({ queryKey: bookmarkKeys.list() });

      // Get current post state (before optimistic update)
      const previousPost = queryClient.getQueryData<Post>(postsKeys.detail(postId));
      
      // Also check in infinite lists (posts feed and bookmarks list)
      let postFromList: Post | undefined;
      queryClient.getQueriesData<InfiniteData<PostsResponse>>({
        queryKey: postsKeys.lists(),
      }).forEach(([, data]) => {
        if (data?.pages) {
          const allPosts = data.pages.flatMap((page) => page.posts || []);
          const found = allPosts.find((p) => p.objectId === postId);
          if (found) {
            postFromList = found;
          }
        }
      });

      // Also check in bookmarks list
      queryClient.getQueriesData<InfiniteData<PostsResponse>>({
        queryKey: bookmarkKeys.list(),
      }).forEach(([, data]) => {
        if (data?.pages) {
          const allPosts = data.pages.flatMap((page) => page.posts || []);
          const found = allPosts.find((p) => p.objectId === postId);
          if (found) {
            postFromList = found;
          }
        }
      });

      const currentPost = previousPost || postFromList;
      if (!currentPost) {
        return { previousPost: undefined, wasInBookmarksList: false };
      }

      const currentIsBookmarked = currentPost.isBookmarked ?? false;
      const newIsBookmarked = !currentIsBookmarked;

      // Optimistically update the post
      const optimisticPost: Post = {
        ...currentPost,
        isBookmarked: newIsBookmarked,
      };

      // Update individual post query
      queryClient.setQueryData<Post>(postsKeys.detail(postId), optimisticPost);

      // Update all posts feed queries
      queryClient.setQueriesData<InfiniteData<PostsResponse>>(
        { queryKey: postsKeys.lists(), exact: false },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              posts: page.posts.map((p) =>
                p.objectId === postId ? optimisticPost : p
              ),
            })),
          };
        }
      );

      // Update bookmarks list queries
      const wasInBookmarksList = currentIsBookmarked;
      queryClient.setQueriesData<InfiniteData<PostsResponse>>(
        { queryKey: bookmarkKeys.list(), exact: false },
        (old) => {
          if (!old) return old;
          
          // If unbookmarking, remove from list optimistically
          if (!newIsBookmarked) {
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                posts: page.posts.filter((p) => p.objectId !== postId),
              })),
            };
          }
          
          // If bookmarking, just update the flag (post will appear on next fetch)
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              posts: page.posts.map((p) =>
                p.objectId === postId ? optimisticPost : p
              ),
            })),
          };
        }
      );

      return { previousPost: currentPost, wasInBookmarksList };
    },
    onSuccess: (_data, { postId }) => {
      // Invalidate queries to refetch authoritative data from server
      queryClient.invalidateQueries({ 
        queryKey: postsKeys.detail(postId),
        refetchType: 'active',
      });
      queryClient.invalidateQueries({ 
        queryKey: postsKeys.lists(),
        refetchType: 'active',
      });
      queryClient.invalidateQueries({ 
        queryKey: bookmarkKeys.list(),
        refetchType: 'active',
      });
    },
    onError: (_error, { postId }, context) => {
      // Rollback optimistic update on error
      if (context?.previousPost) {
        queryClient.setQueryData<Post>(postsKeys.detail(postId), context.previousPost);
        
        queryClient.setQueriesData<InfiniteData<PostsResponse>>(
          { queryKey: postsKeys.lists(), exact: false },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                posts: page.posts.map((p) =>
                  p.objectId === postId ? context.previousPost! : p
                ),
              })),
            };
          }
        );

        // Rollback bookmarks list
        queryClient.setQueriesData<InfiniteData<PostsResponse>>(
          { queryKey: bookmarkKeys.list(), exact: false },
          (old) => {
            if (!old) return old;
            
            // If we removed it optimistically, add it back
            if (context.wasInBookmarksList && !context.previousPost.isBookmarked) {

              return {
                ...old,
                pages: old.pages.map((page) => {
                  const hasPost = page.posts.some((p) => p.objectId === postId);
                  if (hasPost) {
                    return {
                      ...page,
                      posts: page.posts.map((p) =>
                        p.objectId === postId ? context.previousPost! : p
                      ),
                    };
                  }
                  return page;
                }),
              };
            }
            
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                posts: page.posts.map((p) =>
                  p.objectId === postId ? context.previousPost! : p
                ),
              })),
            };
          }
        );
      }
    },
  });
}

/**
 * Infinite scroll query hook for bookmarked posts
 * 
 * @param params - Optional cursor query parameters (limit, etc.)
 * @returns React Query infinite query result
 */
export function useInfiniteBookmarksQuery(params?: CursorQueryParams) {
  const queryParams: CursorQueryParams = {
    limit: params?.limit ?? 10,
    cursor: params?.cursor,
  };

  return useInfiniteQuery({
    queryKey: bookmarkKeys.infiniteList(queryParams),
    queryFn: async ({ pageParam }) => {
      const response = await sdk.bookmarks.getBookmarks({
        ...queryParams,
        cursor: pageParam as string | undefined,
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
      }
      
      const nextCursor = lastPage.nextCursor;
      // 3. No cursor available (cannot continue)
      if (!nextCursor || nextCursor.trim() === '') {
        return undefined;
      }
      
      //  Always check for same cursor to prevent infinite loops
      if (allPages.length > 1) {
        const previousPage = allPages[allPages.length - 2];
        if (previousPage.nextCursor && previousPage.nextCursor === nextCursor) {
          return undefined;
        }
      }
      
      return nextCursor;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

