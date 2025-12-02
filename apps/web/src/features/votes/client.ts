'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { Post } from '@telar/sdk';
import { postsKeys } from '@/features/posts/client';
import type { InfiniteData } from '@tanstack/react-query';

/**
 * Calculate new score based on vote transition
 * This matches the backend logic in vote_service.go
 */
const calculateNewScore = (currentScore: number, currentType: number, newType: number): number => {
  let delta = 0;

  // 1. Remove old vote influence
  if (currentType === 1) delta -= 1;      // Remove Up
  if (currentType === 2) delta += 1;      // Remove Down (double negative)

  // 2. Add new vote influence
  if (newType === 1) delta += 1;          // Add Up
  if (newType === 2) delta -= 1;          // Add Down

  return currentScore + delta;
};

/**
 * Hook for voting on posts with optimistic updates
 * 
 * Architecture: "Send Action, Not State"
 * - UI sends the clicked button type (1 or 2), not the resulting state (0, 1, or 2)
 * - Backend handles toggle logic: if current === clicked, toggle off
 * - Optimistic UI duplicates backend logic for instant feedback
 */
export function useVoteMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ postId, voteType }: { postId: string; voteType: 1 | 2 }) => {
      // Simply forward the action to the backend
      // No cache reading, no state calculation - just send what the user clicked
      await sdk.votes.vote(postId, voteType);
    },
    onMutate: async ({ postId, voteType: clickedType }) => {
      // Cancel any outgoing queries to avoid race conditions
      await queryClient.cancelQueries({ queryKey: postsKeys.detail(postId) });
      await queryClient.cancelQueries({ queryKey: postsKeys.lists() });

      // Get current post state (before optimistic update)
      const previousPost = queryClient.getQueryData<Post>(postsKeys.detail(postId));
      
      // Also check in infinite lists
      let postFromList: Post | undefined;
      queryClient.getQueriesData<InfiniteData<{ posts: Post[]; nextCursor?: string; hasNext?: boolean }>>({
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

      const currentPost = previousPost || postFromList;
      if (!currentPost) {
        return { previousPost: undefined };
      }

      const currentType = currentPost.voteType ?? 0;
      const currentScore = currentPost.score ?? 0;

      // Duplicate backend toggle logic for optimistic UI
      // Backend logic: if existing === clicked, toggle off (0); otherwise, set to clicked
      let newOptimisticType: 0 | 1 | 2;
      if (currentType === clickedType) {
        // Same type clicked again -> toggle off
        newOptimisticType = 0;
      } else {
        // Different type (or none) -> set to clicked type
        newOptimisticType = clickedType;
      }

      // Calculate score delta based on transition
      const newScore = calculateNewScore(currentScore, currentType, newOptimisticType);

      // Optimistically update the post
      const optimisticPost: Post = {
        ...currentPost,
        voteType: newOptimisticType,
        score: newScore,
      };

      // Update individual post query
      queryClient.setQueryData<Post>(postsKeys.detail(postId), optimisticPost);

      // Update all infinite list queries
      queryClient.setQueriesData<InfiniteData<{ posts: Post[]; nextCursor?: string; hasNext?: boolean }>>(
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

      return { previousPost: currentPost };
    },
    onSuccess: (_data, { postId }) => {
      // Backend currently returns success message, not Post
      // Optimistic update provides instant feedback
      // Invalidate queries to refetch authoritative data from server for final sync
      // Using invalidateQueries with refetchType: 'active' to only refetch active queries
      queryClient.invalidateQueries({ 
        queryKey: postsKeys.detail(postId),
        refetchType: 'active',
      });
      queryClient.invalidateQueries({ 
        queryKey: postsKeys.lists(),
        refetchType: 'active',
      });
    },
    onError: (_error, { postId }, context) => {
      // Rollback optimistic update on error
      if (context?.previousPost) {
        queryClient.setQueryData<Post>(postsKeys.detail(postId), context.previousPost);

        queryClient.setQueriesData<InfiniteData<{ posts: Post[]; nextCursor?: string; hasNext?: boolean }>>(
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
      }
    },
  });
}

