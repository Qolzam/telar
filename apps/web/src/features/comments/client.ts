'use client';

import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  keepPreviousData,
} from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { Comment, Post, PostsResponse, CommentsListResponse } from '@telar/sdk';
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


// Lazy replies for a specific parent comment (infinite)
export function useCommentRepliesQuery(parentCommentId: string, limit = 10) {
  return useInfiniteQuery({
    queryKey: [...commentsKeys.detail(parentCommentId), 'replies', { limit }],
    queryFn: async ({ pageParam }) => {
      // pageParam is the cursor string (undefined for first page)
      const cursor = pageParam as string | undefined;
      return sdk.comments.getCommentReplies(parentCommentId, cursor, limit);
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      // Use nextCursor from API response for cursor-based pagination
      if (lastPage && 'nextCursor' in lastPage && lastPage.hasNext && lastPage.nextCursor) {
        return lastPage.nextCursor;
      }
      return undefined;
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: false, // opt-in loading on expand
  });
}

/**
 * Infinite comments query with cursor-based pagination
 * Returns pages of CommentsListResponse; hasNextPage uses nextCursor from API.
 */
export function useInfiniteCommentsQuery(postId: string, limit = 10, enabled = true) {
  return useInfiniteQuery({
    queryKey: commentsKeys.byPost(postId, undefined, limit),
    queryFn: async ({ pageParam }) => {
      // pageParam is the cursor string (undefined for first page)
      const cursor = pageParam as string | undefined;
      return sdk.comments.getCommentsByPost(postId, cursor, limit);
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      // Use nextCursor from API response for cursor-based pagination
      if (lastPage && 'nextCursor' in lastPage && lastPage.hasNext && lastPage.nextCursor) {
        return lastPage.nextCursor;
      }
      return undefined;
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
        isLiked: false, // New comment is not liked by default
        createdDate: Date.now(),
        lastUpdated: Date.now(),
        deleted: false,
        deletedDate: 0,
      };
      
      // Optimistically add comment to all comment queries for this post
      // Use predicate function to ensure proper matching regardless of cursor/limit parameters
      const updateCommentsCache = (old: InfiniteData<CommentsListResponse> | undefined) => {
        if (!old) {
          // If no data exists yet, create initial page with the new comment
          return {
            pages: [{
              comments: [optimisticComment],
              hasNext: false,
            }],
            pageParams: [undefined],
          };
        }
        
        // Add the new comment to the first page (root comments appear first)
        const newPages = old.pages.map((page, pageIndex) => {
          if (pageIndex === 0 && isRootComment) {
            // Add root comment to the beginning of the first page
            return {
              ...page,
              comments: [optimisticComment, ...(page.comments || [])],
            };
          } else if (!isRootComment) {
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
      const matchingQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
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
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, updateCommentsCache);
      });
      
      // If no queries exist yet, set the default query key structure used by useInfiniteCommentsQuery
      // This ensures the optimistic update is visible even if the query hasn't been initialized
      if (matchingQueries.length === 0) {
        // Use the default limit from useInfiniteCommentsQuery (10)
        const defaultQueryKey = commentsKeys.byPost(postId, undefined, 10);
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(defaultQueryKey, updateCommentsCache);
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
      } else if (variables.parentCommentId) {
        // For replies, optimistically increment parent comment's replyCount
        // Update parent comment in the main comments list
        queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
          { queryKey: commentsKeys.byPost(postId), exact: false },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                comments: page.comments.map((comment) =>
                  comment.objectId === variables.parentCommentId
                    ? { ...comment, replyCount: (comment.replyCount || 0) + 1 }
                    : comment
                ),
              })),
            };
          }
        );
        
        // Also optimistically add reply to the parent's replies query
        const repliesQueryKey = [...commentsKeys.detail(variables.parentCommentId), 'replies'];
        queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
          { queryKey: repliesQueryKey, exact: false },
          (old) => {
            if (!old) {
              return {
                pages: [{ comments: [optimisticComment], hasNext: false }],
                pageParams: [undefined],
              };
            }
            return {
              ...old,
              pages: old.pages.map((page, pageIndex) => {
                if (pageIndex === 0) {
                  // Add reply to the beginning of the first page
                  return {
                    ...page,
                    comments: [optimisticComment, ...(page.comments || [])],
                  };
                }
                return page;
              }),
            };
          }
        );
      }
      
      // Return context for rollback
      return { optimisticComment, isRootComment, parentCommentId: variables.parentCommentId };
    },
    onSuccess: (data, _variables, context) => {
      // Replace optimistic comment with real comment from server response
      // API now returns the full Comment object, so we can use it directly without refetching
      const updateCommentsCache = (old: InfiniteData<CommentsListResponse> | undefined) => {
        if (!old) {
          // If no data exists, create initial page with the real comment
          return {
            pages: [{
              comments: [data],
              hasNext: false,
            }],
            pageParams: [undefined],
          };
        }
        
        const newPages = old.pages.map((page, pageIndex) => {
          // Replace optimistic comment with real comment from API
          const hasOptimisticComment = page.comments?.some(
            (comment) => comment.objectId === context?.optimisticComment.objectId
          );
          
          if (hasOptimisticComment) {
            // Replace optimistic comment with real comment from API
            return {
              ...page,
              comments: page.comments.map((comment) =>
                comment.objectId === context?.optimisticComment.objectId
                  ? data
                  : comment
              ),
            };
          } else if (pageIndex === 0 && context?.isRootComment && !context?.optimisticComment.parentCommentId) {
            // If optimistic comment wasn't found (edge case), add real comment to first page
            return {
              ...page,
              comments: [data, ...(page.comments || [])],
            };
          }
          
          return page;
        });
        
        return {
          ...old,
          pages: newPages,
        };
      };
      
      // Get all matching queries and update them individually to ensure proper matching
      const matchingQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
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
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, updateCommentsCache);
      });
      
      // If no queries exist yet, set the default query key structure used by useInfiniteCommentsQuery
      if (matchingQueries.length === 0) {
        const defaultQueryKey = commentsKeys.byPost(postId, undefined, 10);
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(defaultQueryKey, updateCommentsCache);
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
      } else if (context?.parentCommentId) {
        // For replies, update the parent's replies query with the real comment
        const repliesQueryKey = [...commentsKeys.detail(context.parentCommentId), 'replies'];
        queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
          { queryKey: repliesQueryKey, exact: false },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                comments: page.comments.map((comment) =>
                  comment.objectId === context?.optimisticComment.objectId
                    ? data
                    : comment
                ),
              })),
            };
          }
        );
      }
    },
    onError: (_error, _variables, context) => {
      // On error, remove optimistic comment
      if (context?.optimisticComment) {
        const removeOptimisticComment = (old: InfiniteData<CommentsListResponse> | undefined) => {
          if (!old) return old;
          
          const newPages = old.pages.map((page) => {
            return {
              ...page,
              comments: page.comments.filter(
                (comment) => comment.objectId !== context.optimisticComment.objectId
              ),
            };
          });
          
          return {
            ...old,
            pages: newPages,
          };
        };
        
        // Get all matching queries and update them individually to ensure proper matching
        const matchingQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
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
          queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, removeOptimisticComment);
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
      
      // Find the comment to determine if it's a root comment or reply
      let comment: Comment | undefined;
      let isRootComment = false;
      let parentCommentId: string | undefined;
      
      const commentsData = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
        queryKey: commentsKeys.byPost(postId),
        exact: false,
      });
      
      for (const [, data] of commentsData) {
        if (data?.pages) {
          const allComments = data.pages.flatMap((page) => page.comments || []);
          const found = allComments.find((c) => c.objectId === commentId);
          if (found) {
            comment = found;
            isRootComment = !found.parentCommentId;
            parentCommentId = found.parentCommentId;
            break;
          }
        }
      }
      
      // If not found in main comments, check replies queries
      if (!comment) {
        const allRepliesQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
          predicate: (query) => {
            return query.queryKey.includes('replies');
          },
        });
        
        for (const [, data] of allRepliesQueries) {
          if (data?.pages) {
            const allReplies = data.pages.flatMap((page) => page.comments || []);
            const found = allReplies.find((c) => c.objectId === commentId);
            if (found) {
              comment = found;
              isRootComment = !found.parentCommentId;
              parentCommentId = found.parentCommentId;
              break;
            }
          }
        }
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
      } else if (parentCommentId) {
        // For replies, optimistically decrement parent comment's replyCount
        queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
          { queryKey: commentsKeys.byPost(postId), exact: false },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                comments: page.comments.map((c) =>
                  c.objectId === parentCommentId
                    ? { ...c, replyCount: Math.max(0, (c.replyCount || 0) - 1) }
                    : c
                ),
              })),
            };
          }
        );
        
        // Also remove from parent's replies query
        const repliesQueryKey = [...commentsKeys.detail(parentCommentId), 'replies'];
        queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
          { queryKey: repliesQueryKey, exact: false },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                comments: page.comments.filter((c) => c.objectId !== commentId),
              })),
            };
          }
        );
      }
      
      // Remove from main comments list
      queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
        { queryKey: commentsKeys.byPost(postId), exact: false },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              comments: page.comments.filter((c) => c.objectId !== commentId),
            })),
          };
        }
      );
      
      return { comment, isRootComment, parentCommentId };
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

export function useToggleLikeCommentMutation(postId: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (commentId: string) => sdk.comments.toggleLike(commentId),
    onMutate: async (commentId) => {
      // Cancel any outgoing queries to avoid race conditions
      await queryClient.cancelQueries({ queryKey: commentsKeys.byPost(postId) });
      await queryClient.cancelQueries({ queryKey: commentsKeys.detail(commentId) });
      
      // Get the current comment state to calculate optimistic update
      let previousComment: Comment | undefined;
      queryClient.getQueriesData<InfiniteData<CommentsListResponse>>(
        { queryKey: commentsKeys.byPost(postId), exact: false }
      ).forEach(([, data]) => {
        if (data?.pages) {
          const allComments = data.pages.flatMap((page) => page.comments || []);
          const comment = allComments.find((c) => c.objectId === commentId);
          if (comment) {
            previousComment = comment;
          }
        }
      });
      
      // If not found in list, try individual comment query
      if (!previousComment) {
        previousComment = queryClient.getQueryData<Comment>(commentsKeys.detail(commentId));
      }
      
      if (!previousComment) return { previousComment: undefined };
      
      // Calculate optimistic update: toggle isLiked and adjust score
      const previousIsLiked = previousComment.isLiked || false;
      const previousScore = previousComment.score || 0;
      const newIsLiked = !previousIsLiked;
      const newScore = newIsLiked ? previousScore + 1 : previousScore - 1;
      
      // Optimistically update the comment in cache
      const updateComment = (old: Comment | undefined): Comment | undefined => {
        if (!old) return old;
        return {
          ...old,
          isLiked: newIsLiked,
          score: newScore,
        };
      };
      
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
                    comment.objectId === commentId ? updateComment(comment)! : comment
                  )
                : page
            ),
          };
        }
      );
      
      // Also update the individual comment query if it exists
      queryClient.setQueryData<Comment>(commentsKeys.detail(commentId), updateComment);
      
      return { previousComment };
    },
    onSuccess: (data, commentId) => {
      // Update cache with server response (which includes correct score and isLiked)
      const updateComment = (old: Comment | undefined): Comment | undefined => {
        if (!old) return old;
        return {
          ...old,
          isLiked: data.isLiked,
          score: data.score,
        };
      };
      
      // Update all comment queries for this post
      queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
        { queryKey: commentsKeys.byPost(postId), exact: false },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              comments: page.comments.map((comment) =>
                comment.objectId === commentId ? updateComment(comment)! : comment
              ),
            })),
          };
        }
      );
      
      // Update individual comment query
      queryClient.setQueryData<Comment>(commentsKeys.detail(commentId), updateComment);
    },
    onError: (_error, commentId, context) => {
      // On error, revert optimistic updates
      if (context?.previousComment) {
        const revertComment = (old: Comment | undefined): Comment | undefined => {
          if (!old) return old;
          return {
            ...old,
            isLiked: context.previousComment!.isLiked,
            score: context.previousComment!.score,
          };
        };
        
        queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
          { queryKey: commentsKeys.byPost(postId), exact: false },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                comments: page.comments.map((comment) =>
                  comment.objectId === commentId ? revertComment(comment)! : comment
                ),
              })),
            };
          }
        );
        
        queryClient.setQueryData<Comment>(commentsKeys.detail(commentId), revertComment);
      }
    },
  });
}


