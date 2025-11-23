'use client';

import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  keepPreviousData,
} from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { Comment, Post, PostsResponse } from '@telar/sdk';
import { postsKeys } from '@/features/posts/client';
import type { InfiniteData } from '@tanstack/react-query';
import { useSession } from '@/features/auth/client';

export const commentsKeys = {
  all: ['comments'] as const,
  lists: () => [...commentsKeys.all, 'list'] as const,
  byPost: (postId: string, page?: number, limit?: number) =>
    [...commentsKeys.lists(), { postId, page, limit }] as const,
  detail: (commentId: string) => [...commentsKeys.all, 'detail', commentId] as const,
};

export function useCommentsQuery(postId: string, page = 1, limit = 10) {
  return useQuery({
    queryKey: commentsKeys.byPost(postId, page, limit),
    queryFn: async () => {
      return sdk.comments.getCommentsByPost(postId, { page, limit });
    },
    placeholderData: keepPreviousData,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

// Lazy replies for a specific parent comment (infinite)
export function useCommentRepliesQuery(parentCommentId: string, limit = 10) {
  return useInfiniteQuery({
    queryKey: [...commentsKeys.detail(parentCommentId), 'replies', { limit }],
    queryFn: async ({ pageParam = 1 }) => {
      return sdk.comments.getCommentReplies(parentCommentId, {
        page: pageParam as number,
        limit,
      });
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) =>
      Array.isArray(lastPage) && lastPage.length === limit ? allPages.length + 1 : undefined,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: false, // opt-in loading on expand
  });
}

/**
 * Infinite comments query for pagination with "Show more comments"
 * Returns pages of Comment[]; hasNextPage is true if the last page length === limit.
 */
export function useInfiniteCommentsQuery(postId: string, limit = 10, enabled = true) {
  return useInfiniteQuery({
    queryKey: commentsKeys.byPost(postId, undefined, limit),
    queryFn: async ({ pageParam = 1 }) => {
      return sdk.comments.getCommentsByPost(postId, { page: pageParam as number, limit });
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) => {
      if (!Array.isArray(lastPage)) return undefined;
      return lastPage.length === limit ? allPages.length + 1 : undefined;
    },
    enabled,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useCommentQuery(commentId: string) {
  return useQuery({
    queryKey: commentsKeys.detail(commentId),
    queryFn: () => sdk.comments.getComment(commentId),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useCreateCommentMutation(postId: string) {
  const queryClient = useQueryClient();
  const { user } = useSession();

  return useMutation({
    mutationFn: (data: { text: string; parentCommentId?: string }) =>
      sdk.comments.createComment({ postId, text: data.text, parentCommentId: data.parentCommentId }),
    onMutate: async (variables) => {
      // Cancel any outgoing queries for this post to avoid race conditions
      await queryClient.cancelQueries({ queryKey: postsKeys.lists() });
      await queryClient.cancelQueries({ queryKey: postsKeys.detail(postId) });
      await queryClient.cancelQueries({ queryKey: commentsKeys.byPost(postId) });
      
      // Only update post.commentCounter for root comments (not replies)
      // Root comments have no parentCommentId, replies have a parentCommentId
      const isRootComment = !variables.parentCommentId;
      
      // Create optimistic comment object
      const optimisticComment: Comment = {
        objectId: `temp-${Date.now()}`,
        postId,
        parentCommentId: variables.parentCommentId,
        text: variables.text,
        ownerUserId: user?.id || '',
        ownerDisplayName: user?.displayName || user?.socialName || 'You',
        ownerAvatar: user?.avatar || '',
        score: 0,
        replyCount: 0,
        createdDate: Date.now(),
        lastUpdated: Date.now(),
        deleted: false,
        deletedDate: 0,
      };
      
      // Optimistically add comment to all comment queries for this post
      // Use predicate function to ensure proper matching regardless of page/limit parameters
      const updateCommentsCache = (old: InfiniteData<Comment[]> | undefined) => {
        if (!old) {
          // If no data exists yet, create initial page with the new comment
          return {
            pages: [[optimisticComment]],
            pageParams: [1],
          };
        }
        
        // Add the new comment to the first page (root comments appear first)
        const newPages = old.pages.map((page, pageIndex) => {
          if (pageIndex === 0 && isRootComment) {
            // Add root comment to the beginning of the first page
            return [optimisticComment, ...(Array.isArray(page) ? page : [])];
          } else if (!isRootComment && Array.isArray(page)) {
            // For replies, find the parent and add reply to it
            // This is a simplified optimistic update - actual structure depends on API response
            return page;
          }
          return page;
        });
        
        return {
          ...old,
          pages: newPages,
        };
      };
      
      // Get all matching queries and update them individually to ensure proper matching
      const matchingQueries = queryClient.getQueriesData<InfiniteData<Comment[]>>({
        queryKey: commentsKeys.lists(),
        predicate: (query) => {
          const queryKey = query.queryKey;
          // Match any query key that contains this postId in the object parameter
          if (queryKey.length >= 3 && typeof queryKey[2] === 'object' && queryKey[2] !== null) {
            const params = queryKey[2] as { postId?: string };
            return params.postId === postId;
          }
          return false;
        },
      });
      
      // Update all matching queries
      matchingQueries.forEach(([queryKey]) => {
        queryClient.setQueryData<InfiniteData<Comment[]>>(queryKey, updateCommentsCache);
      });
      
      // If no queries exist yet, set the default query key structure used by useInfiniteCommentsQuery
      // This ensures the optimistic update is visible even if the query hasn't been initialized
      if (matchingQueries.length === 0) {
        // Use the default limit from useInfiniteCommentsQuery (10)
        const defaultQueryKey = commentsKeys.byPost(postId, undefined, 10);
        queryClient.setQueryData<InfiniteData<Comment[]>>(defaultQueryKey, updateCommentsCache);
      }
      
      if (isRootComment) {
        // Optimistically update post comment count in cache
        queryClient.setQueriesData<InfiniteData<PostsResponse>>(
          { queryKey: postsKeys.lists() },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                posts: page.posts.map((p) =>
                  p.objectId === postId
                    ? { ...p, commentCounter: (p.commentCounter || 0) + 1 }
                    : p
                ),
              })),
            };
          }
        );
        
        queryClient.setQueryData<Post>(postsKeys.detail(postId), (old) => {
          if (!old) return old;
          return { ...old, commentCounter: (old.commentCounter || 0) + 1 };
        });
      }
      
      // Return context for rollback
      return { optimisticComment, isRootComment };
    },
    onSuccess: (data, _variables, context) => {
      // Replace optimistic comment with real comment from server response
      // API now returns the full Comment object, so we can use it directly without refetching
      const updateCommentsCache = (old: InfiniteData<Comment[]> | undefined) => {
        if (!old) {
          // If no data exists, create initial page with the real comment
          return {
            pages: [[data]],
            pageParams: [1],
          };
        }
        
        const newPages = old.pages.map((page, pageIndex) => {
          if (!Array.isArray(page)) return page;
          
          // Replace optimistic comment with real comment from API
          const hasOptimisticComment = page.some(
            (comment) => comment.objectId === context?.optimisticComment.objectId
          );
          
          if (hasOptimisticComment) {
            // Replace optimistic comment with real comment from API
            return page.map((comment) =>
              comment.objectId === context?.optimisticComment.objectId
                ? data
                : comment
            );
          } else if (pageIndex === 0 && context?.isRootComment && !context?.optimisticComment.parentCommentId) {
            // If optimistic comment wasn't found (edge case), add real comment to first page
            return [data, ...page];
          }
          
          return page;
        });
        
        return {
          ...old,
          pages: newPages,
        };
      };
      
      // Get all matching queries and update them individually to ensure proper matching
      const matchingQueries = queryClient.getQueriesData<InfiniteData<Comment[]>>({
        queryKey: commentsKeys.lists(),
        predicate: (query) => {
          const queryKey = query.queryKey;
          // Match any query key that contains this postId in the object parameter
          if (queryKey.length >= 3 && typeof queryKey[2] === 'object' && queryKey[2] !== null) {
            const params = queryKey[2] as { postId?: string };
            return params.postId === postId;
          }
          return false;
        },
      });
      
      // Update all matching queries
      matchingQueries.forEach(([queryKey]) => {
        queryClient.setQueryData<InfiniteData<Comment[]>>(queryKey, updateCommentsCache);
      });
      
      // If no queries exist yet, set the default query key structure used by useInfiniteCommentsQuery
      if (matchingQueries.length === 0) {
        const defaultQueryKey = commentsKeys.byPost(postId, undefined, 10);
        queryClient.setQueryData<InfiniteData<Comment[]>>(defaultQueryKey, updateCommentsCache);
      }
      
      // NO refetch needed - we already have the complete data from the API response
      // Only update post counter if it's a root comment (backend handles this, but we sync UI optimistically)
      if (context?.isRootComment) {
        // Update post counter optimistically - backend already incremented it
        // Just ensure our cache is in sync, but don't refetch
        queryClient.setQueryData<Post>(postsKeys.detail(postId), (old) => {
          if (!old) return old;
          // Ensure counter matches what we optimistically set
          return { ...old, commentCounter: (old.commentCounter || 0) };
        });
      }
    },
    onError: (_error, _variables, context) => {
      // On error, remove optimistic comment
      if (context?.optimisticComment) {
        const removeOptimisticComment = (old: InfiniteData<Comment[]> | undefined) => {
          if (!old) return old;
          
          const newPages = old.pages.map((page) => {
            if (!Array.isArray(page)) return page;
            return page.filter(
              (comment) => comment.objectId !== context.optimisticComment.objectId
            );
          });
          
          return {
            ...old,
            pages: newPages,
          };
        };
        
        // Get all matching queries and update them individually to ensure proper matching
        const matchingQueries = queryClient.getQueriesData<InfiniteData<Comment[]>>({
          queryKey: commentsKeys.lists(),
          predicate: (query) => {
            const queryKey = query.queryKey;
            // Match any query key that contains this postId in the object parameter
            if (queryKey.length >= 3 && typeof queryKey[2] === 'object' && queryKey[2] !== null) {
              const params = queryKey[2] as { postId?: string };
              return params.postId === postId;
            }
            return false;
          },
        });
        
        // Update all matching queries
        matchingQueries.forEach(([queryKey]) => {
          queryClient.setQueryData<InfiniteData<Comment[]>>(queryKey, removeOptimisticComment);
        });
      }
      
      // Revert optimistic post counter updates
      if (context?.isRootComment) {
        queryClient.invalidateQueries({ queryKey: postsKeys.lists() });
        queryClient.invalidateQueries({ queryKey: postsKeys.detail(postId) });
      }
    },
  });
}

export function useUpdateCommentMutation(postId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: { objectId: string; text: string }) =>
      sdk.comments.updateComment({ objectId: data.objectId, text: data.text }),
    onSuccess: (_data, variables) => {
      // Only invalidate this specific post's comments and the specific comment
      queryClient.invalidateQueries({ 
        queryKey: commentsKeys.byPost(postId),
        exact: false, // Invalidate all pages/limits for this post
      });
      queryClient.invalidateQueries({ queryKey: commentsKeys.detail(variables.objectId) });
    },
  });
}

export function useDeleteCommentMutation(postId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (commentId: string) => sdk.comments.deleteComment(commentId, postId),
    onMutate: async (commentId) => {
      // Cancel any outgoing queries to avoid race conditions
      await queryClient.cancelQueries({ queryKey: postsKeys.lists() });
      await queryClient.cancelQueries({ queryKey: postsKeys.detail(postId) });
      await queryClient.cancelQueries({ queryKey: commentsKeys.byPost(postId) });
      
      // Check if this is a root comment by looking it up in the comments cache
      // Only root comments affect post.commentCounter
      let isRootComment = false;
      const commentsData = queryClient.getQueryData<InfiniteData<any[]>>(commentsKeys.byPost(postId));
      if (commentsData?.pages) {
        const allComments = commentsData.pages.flat();
        const comment = allComments.find((c: any) => c.objectId === commentId);
        isRootComment = !comment?.parentCommentId;
      }
      
      // Only optimistically update post.commentCounter for root comments
      if (isRootComment) {
        queryClient.setQueriesData<InfiniteData<PostsResponse>>(
          { queryKey: postsKeys.lists() },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                posts: page.posts.map((p) =>
                  p.objectId === postId
                    ? { ...p, commentCounter: Math.max(0, (p.commentCounter || 0) - 1) }
                    : p
                ),
              })),
            };
          }
        );
        
        queryClient.setQueryData<Post>(postsKeys.detail(postId), (old) => {
          if (!old) return old;
          return { ...old, commentCounter: Math.max(0, (old.commentCounter || 0) - 1) };
        });
      }
    },
    onSuccess: () => {
      // Only invalidate this specific post's comments, not all posts
      queryClient.invalidateQueries({ 
        queryKey: commentsKeys.byPost(postId),
        exact: false, // Invalidate all pages/limits for this post
      });
      
      // Invalidate post query to get updated commentCounter from backend
      // This ensures the count is accurate after the backend updates it
      queryClient.invalidateQueries({ queryKey: postsKeys.detail(postId) });
      queryClient.invalidateQueries({ queryKey: postsKeys.lists() });
    },
    onError: () => {
      // On error, revert optimistic updates
      queryClient.invalidateQueries({ queryKey: postsKeys.lists() });
      queryClient.invalidateQueries({ queryKey: postsKeys.detail(postId) });
    },
  });
}

export function useLikeCommentMutation(postId: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ commentId, delta }: { commentId: string; delta: number }) =>
      sdk.comments.likeComment(commentId, delta),
    onMutate: async ({ commentId, delta }) => {
      // Cancel any outgoing queries to avoid race conditions
      await queryClient.cancelQueries({ queryKey: commentsKeys.byPost(postId) });
      
      // Optimistically update the comment's score in cache
      // Update all comment queries for this post (different pages/limits)
      queryClient.setQueriesData<InfiniteData<Comment[]>>(
        { queryKey: commentsKeys.byPost(postId), exact: false },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) =>
              Array.isArray(page)
                ? page.map((comment) =>
                    comment.objectId === commentId
                      ? { ...comment, score: (comment.score || 0) + delta }
                      : comment
                  )
                : page
            ),
          };
        }
      );
      
      // Also update the individual comment query if it exists
      queryClient.setQueryData<Comment>(commentsKeys.detail(commentId), (old) => {
        if (!old) return old;
        return { ...old, score: (old.score || 0) + delta };
      });
    },
    onSuccess: () => {
      // Only invalidate this specific post's comments to refetch latest from server
      // This ensures data consistency while minimizing unnecessary refetches
      queryClient.invalidateQueries({ 
        queryKey: commentsKeys.byPost(postId),
        exact: false,
        refetchType: 'active', // Only refetch active queries (currently visible)
      });
    },
    onError: () => {
      // On error, revert optimistic updates by invalidating
      queryClient.invalidateQueries({ 
        queryKey: commentsKeys.byPost(postId),
        exact: false,
      });
    },
  });
}


