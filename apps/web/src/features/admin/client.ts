'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { AdminMember, MembersListResponse } from '@telar/sdk';
import { useState } from 'react';

export const adminKeys = {
  all: ['admin'] as const,
  members: () => [...adminKeys.all, 'members'] as const,
  membersList: (limit: number, offset: number) => [...adminKeys.members(), { limit, offset }] as const,
  member: (userId: string) => [...adminKeys.members(), 'detail', userId] as const,
};

export function useMembersQuery(args: { limit: number; offset: number; search?: string; sortBy?: string; sortOrder?: 'asc' | 'desc' }) {
  const { limit, offset, search, sortBy, sortOrder } = args;
  return useQuery({
    queryKey: [...adminKeys.membersList(limit, offset), { search, sortBy, sortOrder }],
    queryFn: async (): Promise<MembersListResponse> => {
      return sdk.admin.listMembers({ limit, offset, search, sortBy, sortOrder });
    },
    staleTime: 60_000,
  });
}

export function useUpdateMemberRoleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (args: { userId: string; role: string }) => {
      await sdk.admin.updateMemberRole(args.userId, args.role);
    },
    onSuccess: () => {
      // feedback is handled in component via mutation state
      queryClient.invalidateQueries({ queryKey: adminKeys.members() });
    },
  });
}

export function useBanUserMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (userId: string) => {
      await sdk.admin.banMember(userId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminKeys.members() });
    },
  });
}


