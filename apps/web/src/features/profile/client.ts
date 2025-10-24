'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { UpdateProfileRequest, UserProfileModel } from '@telar/sdk';

export function useProfile(userId?: string) {
  return useQuery({
    queryKey: ['profile', userId],
    queryFn: () => (userId ? sdk.profile.getProfileById(userId) : sdk.profile.getMyProfile()),
    enabled: !!userId || userId === undefined,
  });
}

export function useProfileBySocialName(socialName: string) {
  return useQuery({
    queryKey: ['profile', 'social', socialName],
    queryFn: () => sdk.profile.getProfileBySocialName(socialName),
    enabled: !!socialName,
  });
}

export function useUpdateProfileMutation() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: UpdateProfileRequest) => sdk.profile.updateProfile(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] });
    },
  });
}

export function useProfilesByIds(userIds: string[]) {
  return useQuery({
    queryKey: ['profiles', 'bulk', userIds],
    queryFn: () => sdk.profile.getProfilesByIds(userIds),
    enabled: userIds.length > 0,
  });
}


