'use client';

import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { SessionData } from '../types';

async function fetchSession(): Promise<SessionData | null> {
  try {
    const response = await fetch('/api/auth/session', {
      credentials: 'include',
      cache: 'no-store',
    });

    if (!response.ok) {
      if (response.status === 401) {
        return null;
      }
      throw new Error('Failed to fetch session');
    }

    const data = await response.json();
    return data;
  } catch (error) {
    console.error('[useSession] Error fetching session:', error);
    return null;
  }
}

export const SESSION_QUERY_KEY = ['auth', 'session'] as const;

export function useSession() {
  const queryClient = useQueryClient();

  const {
    data: session,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: SESSION_QUERY_KEY,
    queryFn: fetchSession,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: true,
  });

  const invalidateSession = () => {
    queryClient.invalidateQueries({ queryKey: SESSION_QUERY_KEY });
  };

  const clearSession = () => {
    queryClient.setQueryData(SESSION_QUERY_KEY, null);
  };

  return {
    session,
    user: session?.user || null,
    isAuthenticated: session?.isAuthenticated || false,
    isLoading,
    error,
    refetch,
    invalidateSession,
    clearSession,
  };
}
