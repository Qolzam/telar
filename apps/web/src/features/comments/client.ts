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
  byPost: (postId: string, limit?: number) =>
    [...commentsKeys.lists(), { postId, limit }] as const,
  detail: (commentId: string) => [...commentsKeys.all, 'detail', commentId] as const,
};

/**
 * Key Factory for replies queries
 * Ensures consistent query key structure across all cache operations
 */
const getRepliesKey = (commentId: string, limit: number = 10) =>
  [...commentsKeys.detail(commentId), 'replies', { limit }] as const;

export function useCommentRepliesQuery(parentCommentId: string, limit = 10) {
  return useInfiniteQuery({
    queryKey: getRepliesKey(parentCommentId, limit),
    queryFn: async ({ pageParam }) => {
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
    enabled: false,
  });
}

/**
 * Infinite comments query with cursor-based pagination
 * Returns pages of CommentsListResponse; hasNextPage uses nextCursor from API.
 */
export function useInfiniteCommentsQuery(postId: string, limit = 10, enabled = true) {
  return useInfiniteQuery({
    queryKey: commentsKeys.byPost(postId, limit),
    queryFn: async ({ pageParam }) => {
      const cursor = pageParam as string | undefined;
      return sdk.comments.getCommentsByPost(postId, cursor, limit);
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
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
      
      let actualRootId: string | undefined;
      
      let replyToUserId: string | undefined;
      let replyToDisplayName: string | undefined;
      
      // This ensures "Replying to @User" appears instantly, not after server response
      if (!isRootComment && variables.parentCommentId) {
        let targetComment: Comment | undefined;
        actualRootId = variables.parentCommentId; // Initialize, will be updated if target found
        
        // Target the specific cache: First check root comments list for this post
        const postCommentsQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
          queryKey: commentsKeys.byPost(postId),
          exact: false,
        });
        
        for (const [, data] of postCommentsQueries) {
          if (data?.pages && !targetComment) {
            const allComments = data.pages.flatMap((page) => page.comments || []);
            const found = allComments.find((c) => c.objectId === variables.parentCommentId);
            if (found) {
              targetComment = found;
              break;
            }
          }
        }
        
        // If not found in root comments, check replies queries (target might be a reply)
        if (!targetComment) {
          // Search replies queries - we know the postId, so we can narrow the search
          const repliesQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
            predicate: (query) => {
              const key = query.queryKey;
              // Match replies query pattern: ['comments', 'detail', commentId, 'replies', { limit }]
              return (
                Array.isArray(key) &&
                key.length >= 4 &&
                key[0] === 'comments' &&
                key[1] === 'detail' &&
                key[3] === 'replies'
              );
            },
          });
          
          for (const [, data] of repliesQueries) {
            if (data?.pages && !targetComment) {
              const allComments = data.pages.flatMap((page) => page.comments || []);
              const found = allComments.find((c) => c.objectId === variables.parentCommentId);
              if (found) {
                targetComment = found;
                break; 
              }
            }
          }
        }
        
        if (targetComment) {
          // Smart Root Detection (Two-Tier Architecture):
          if (targetComment.parentCommentId) {
            // Target is a reply: Root is its parent
            actualRootId = targetComment.parentCommentId;
          } else {
            // Target is a root comment: Root is the target itself
            actualRootId = targetComment.objectId;
          }
          // Set replyToUserId and replyToDisplayName for "Replying to @User" display
          // Always reply to the target comment's owner (regardless of whether target is root or reply)
          replyToUserId = targetComment.ownerUserId;
          // Use ownerDisplayName with fallback to ensure it's never undefined
          replyToDisplayName = targetComment.ownerDisplayName || targetComment.ownerUserId || 'User';
          
       
        } 
      }
      
      // Create optimistic comment object WITH replyTo fields already populated
      const optimisticComment: Comment = {
        objectId: `temp-${Date.now()}`,
        postId,
        parentCommentId: isRootComment ? undefined : actualRootId, // For replies, use actualRootId (root)
        text: variables.text,
        ownerUserId: user?.id || '',
        ownerDisplayName: user?.displayName || user?.socialName || 'You',
        ownerAvatar: user?.avatar || '',
        score: 0,
        replyCount: 0,
        isLiked: false,
        createdDate: Date.now(),
        lastUpdated: Date.now(),
        deleted: false,
        deletedDate: 0,
        // This ensures "Replying to @User" appears instantly, matching persisted state
        replyToUserId: replyToUserId,
        replyToDisplayName: replyToDisplayName,
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
        const defaultQueryKey = commentsKeys.byPost(postId, 10);
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
        
        // For replies, optimistically increment root comment's replyCount
        queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
          { queryKey: commentsKeys.byPost(postId), exact: false },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                comments: page.comments.map((comment) =>
                  comment.objectId === actualRootId
                    ? { ...comment, replyCount: (comment.replyCount || 0) + 1 }
                    : comment
                ),
              })),
            };
          }
        );
        
        // Update the cache for the ROOT's reply list (not the immediate parent)
        // This ensures flattened structure in optimistic UI
        // Replies are sorted ASC (oldest first), so new replies appear at the END
        // Only update if we found the actualRootId (should always be set for replies)
        if (actualRootId) {
          // Use exact same key as ReplyList component (limit: 10)
          // ReplyList uses useCommentRepliesQuery(rootId, 10), so we must match that key
          const repliesKey = getRepliesKey(actualRootId, 10);
          
          queryClient.setQueryData<InfiniteData<CommentsListResponse>>(repliesKey, (old) => {
            if (!old) {
              // If cache doesn't exist, create it with the optimistic comment
              return {
                pages: [{ comments: [optimisticComment], hasNext: false }],
                pageParams: [undefined],
              };
            }
            
            // Immutable update - create new object/array references
            // React Query only triggers re-render if references change
            return {
              ...old,
              pages: old.pages.map((page, pageIndex) => {
                if (pageIndex === old.pages.length - 1) {
                  // Add reply to the END of the last page (ASC sort: newest at bottom)
                  // Create new array reference to trigger re-render
                  return {
                    ...page,
                    comments: [...(page.comments || []), optimisticComment],
                  };
                }
                return page;
              }),
            };
          });
        }
      }
      
      // Return context for rollback
      return { 
        optimisticComment, 
        isRootComment, 
        parentCommentId: variables.parentCommentId,
        actualRootId, // Only set for replies (undefined for root comments)
      };
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
        const defaultQueryKey = commentsKeys.byPost(postId, 10);
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
      } else if (context?.parentCommentId && context?.actualRootId) {
        // For replies, update the ROOT's replies query with the real comment (Two-Tier Architecture)
        // Use actualRootId (not parentCommentId) because replies always point to root
        // Use exact same key as ReplyList component (limit: 10)
        const repliesKey = getRepliesKey(context.actualRootId, 10);
        
        const updateRepliesCache = (old: InfiniteData<CommentsListResponse> | undefined) => {
          if (!old) {
            // If no cache exists, create it with the real comment
            return {
              pages: [{ comments: [data], hasNext: false }],
              pageParams: [undefined],
            };
          }
          
          // Replies are sorted ASC (oldest first), so new replies always go to the end.
          // The optimistic update already placed the item at the end, we just swap temp-ID with real-ID.
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              comments: page.comments.map((comment) => {
                // Match by Temp ID (Optimistic) OR Real ID (Idempotency)
                if (comment.objectId === context?.optimisticComment.objectId || comment.objectId === data.objectId) {
                  return data; // Replace temp with real
                }
                return comment;
              }),
            })),
          };
        };
        
        // Update the exact query key using Key Factory
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(repliesKey, updateRepliesCache);
        
        // NOTE: We don't invalidate here because we've already updated the cache with setQueryData
        // Invalidating would cause an unnecessary refetch and might clear the optimistic update
        // If we need to sync with server for different limits/sorting, we can invalidate selectively
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
    onMutate: async (variables) => {
      // Cancel any outgoing queries to avoid race conditions
      await queryClient.cancelQueries({ queryKey: commentsKeys.byPost(postId) });
      await queryClient.cancelQueries({ queryKey: commentsKeys.detail(variables.objectId) });
      
      // Optimistically update the comment text
      const updateCommentInCache = (old: InfiniteData<CommentsListResponse> | undefined) => {
        if (!old) return old;
        return {
          ...old,
          pages: old.pages.map((page) => ({
            ...page,
            comments: page.comments.map((comment) =>
              comment.objectId === variables.objectId
                ? { ...comment, text: variables.text }
                : comment
            ),
          })),
        };
      };
      
      // Find all matching queries using predicate pattern (same as delete mutation)
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
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, updateCommentInCache);
      });
      
      // Also update replies queries if this comment is a reply
      const allRepliesQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
        predicate: (query) => {
          return query.queryKey.includes('replies');
        },
      });
      
      allRepliesQueries.forEach(([queryKey]) => {
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, updateCommentInCache);
      });
      
      // Update individual comment query if it exists
      queryClient.setQueryData<Comment>(commentsKeys.detail(variables.objectId), (old) => {
        if (!old) return old;
        return { ...old, text: variables.text };
      });
    },
    onSuccess: (data, variables) => {
      // Replace optimistic update with real data from server
      const updateCommentWithServerData = (old: InfiniteData<CommentsListResponse> | undefined) => {
        if (!old) return old;
        return {
          ...old,
          pages: old.pages.map((page) => ({
            ...page,
            comments: page.comments.map((comment) =>
              comment.objectId === variables.objectId ? data : comment
            ),
          })),
        };
      };
      
      // Find all matching queries using predicate pattern
      const matchingQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
        queryKey: commentsKeys.lists(),
        predicate: (query) => {
          const queryKey = query.queryKey;
          if (queryKey.length >= 3 && typeof queryKey[2] === 'object' && queryKey[2] !== null) {
            const params = queryKey[2] as { postId?: string };
            return params.postId === postId;
          }
          return false;
        },
      });
      
      // Update all matching queries with server data
      matchingQueries.forEach(([queryKey]) => {
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, updateCommentWithServerData);
      });
      
      // Also update replies queries
      const allRepliesQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
        predicate: (query) => {
          return query.queryKey.includes('replies');
        },
      });
      
      allRepliesQueries.forEach(([queryKey]) => {
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, updateCommentWithServerData);
      });
      
      // Update individual comment query
      queryClient.setQueryData<Comment>(commentsKeys.detail(variables.objectId), data);
    },
    onError: (_error, variables) => {
      // On error, invalidate to refetch correct data
      const matchingQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
        queryKey: commentsKeys.lists(),
        predicate: (query) => {
          const queryKey = query.queryKey;
          if (queryKey.length >= 3 && typeof queryKey[2] === 'object' && queryKey[2] !== null) {
            const params = queryKey[2] as { postId?: string };
            return params.postId === postId;
          }
          return false;
        },
      });
      
      // Invalidate all matching queries to refetch
      matchingQueries.forEach(([queryKey]) => {
        queryClient.invalidateQueries({ queryKey });
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
      
      // Use predicate to match queries correctly (same pattern as create mutation)
      const commentsData = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
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
        const parentCommentQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
          queryKey: commentsKeys.lists(),
          predicate: (query) => {
            const queryKey = query.queryKey;
            if (queryKey.length >= 3 && typeof queryKey[2] === 'object' && queryKey[2] !== null) {
              const params = queryKey[2] as { postId?: string };
              return params.postId === postId;
            }
            return false;
          },
        });
        
        parentCommentQueries.forEach(([queryKey]) => {
          queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, (old) => {
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
          });
        });
        
        // Also remove from parent's replies query
        // Update all replies queries for this parent (different limits might exist)
        // Key structure: ['comments', 'detail', parentCommentId, 'replies', { limit }]
        const allRepliesQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
          predicate: (query) => {
            const key = query.queryKey;
            // Check if this is a replies query for the parent comment
            // Key should be: ['comments', 'detail', parentCommentId, 'replies', ...]
            if (key.length >= 4 && key[0] === 'comments' && key[1] === 'detail' && key[2] === parentCommentId && key[3] === 'replies') {
              return true;
            }
            return false;
          },
        });
        
        // Remove comment from all matching replies queries
        allRepliesQueries.forEach(([queryKey]) => {
          queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, (old) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                comments: page.comments.filter((c) => c.objectId !== commentId),
              })),
            };
          });
        });
      }
      
      // Remove from main comments list (root comments)
      // Use the same predicate pattern as create mutation to match queries correctly
      const allCommentQueries = queryClient.getQueriesData<InfiniteData<CommentsListResponse>>({
        queryKey: commentsKeys.lists(),
        predicate: (query) => {
          const queryKey = query.queryKey;
          // Match any query key that contains this postId in the object parameter
          // Query key structure: ['comments', 'list', { postId, page?, limit? }]
          if (queryKey.length >= 3 && typeof queryKey[2] === 'object' && queryKey[2] !== null) {
            const params = queryKey[2] as { postId?: string };
            return params.postId === postId;
          }
          return false;
        },
      });
      
      // Update all matching queries to remove the comment
      allCommentQueries.forEach(([queryKey]) => {
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(queryKey, (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              comments: page.comments.filter((c) => c.objectId !== commentId),
            })),
          };
        });
      });
      
      return { comment, isRootComment, parentCommentId };
    },
    onSuccess: () => {
      // DO NOT invalidate comments queries - we've already optimistically removed the comment
      // Invalidating would trigger a refetch that might bring back the comment before backend processes it
      // The optimistic update should persist until the next natural refetch
      
      // Only invalidate post queries to update commentCounter from backend
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
    mutationFn: (commentId: string) => {
      return sdk.comments.toggleLike(commentId);
    },
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
      
      // Also update replies queries (comments can be replies too)
      // Use Key Factory to ensure exact query key matching
      const parentCommentId = previousComment.parentCommentId;
      if (parentCommentId) {
        const repliesKey = getRepliesKey(parentCommentId);
        queryClient.setQueryData<InfiniteData<CommentsListResponse>>(repliesKey, (old) => {
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
        });
      }
      
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
      
      // Update ALL comment queries that might contain this comment
      // This includes post comments queries and replies queries
      queryClient.setQueriesData<InfiniteData<CommentsListResponse>>(
        { queryKey: ['comments'], exact: false },
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
      
      // Also specifically update post comments queries
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


