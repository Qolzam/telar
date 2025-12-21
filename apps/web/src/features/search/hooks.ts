'use client';

import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { sdk } from '@/lib/sdk';
import type { Post, UserProfileModel } from '@telar/sdk';

const useDebouncedValue = (value: string, delay = 300) => {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debounced;
};

export const useGlobalSearch = (query: string) => {
  const debouncedQuery = useDebouncedValue(query, 300);
  const enabled = debouncedQuery.trim().length > 2;

  const profiles = useQuery({
    queryKey: ['search', 'profiles', debouncedQuery],
    queryFn: (): Promise<UserProfileModel[]> => sdk.profile.searchProfiles(debouncedQuery),
    enabled,
    staleTime: 60_000,
  });

  const posts = useQuery({
    queryKey: ['search', 'posts', debouncedQuery],
    queryFn: (): Promise<Post[]> => sdk.posts.searchPosts(debouncedQuery),
    enabled,
    staleTime: 60_000,
  });

  return {
    profiles: profiles.data ?? [],
    posts: posts.data ?? [],
    isLoading: profiles.isLoading || posts.isLoading,
  };
};






