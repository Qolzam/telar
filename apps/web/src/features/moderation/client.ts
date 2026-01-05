'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { ModerationListResponse, ModerationDetails, FlaggedPost } from '@telar/sdk';

// Re-export types for backward compatibility
export type { ModerationDetails, FlaggedPost, ModerationListResponse };

async function fetchModerationQueue(): Promise<ModerationListResponse> {
  return sdk.admin.getModerationQueue();
}

async function approvePost(postId: string): Promise<{ message: string }> {
  return sdk.admin.approvePost(postId);
}

async function rejectPost(postId: string): Promise<{ message: string }> {
  return sdk.admin.rejectPost(postId);
}

export const moderationKeys = {
  all: ['moderation'] as const,
  queue: () => [...moderationKeys.all, 'queue'] as const,
};

export function useModerationQueue() {
  return useQuery({
    queryKey: moderationKeys.queue(),
    queryFn: fetchModerationQueue,
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

export function useApprovePost() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: approvePost,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: moderationKeys.queue() });
    },
  });
}

export function useRejectPost() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: rejectPost,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: moderationKeys.queue() });
    },
  });
}




