'use client';

import { useMutation } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useSession } from './useSession';

async function logoutMutation(): Promise<void> {
  const response = await fetch('/api/auth/logout', {
    method: 'POST',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error('Logout failed');
  }
}

export function useLogout() {
  const router = useRouter();
  const { clearSession } = useSession();

  const mutation = useMutation({
    mutationFn: logoutMutation,
    onSuccess: () => {
      clearSession();
      router.push('/login');
    },
    onError: (error: Error) => {
      console.error('[Logout] Logout failed:', error.message);
    },
  });

  return {
    logout: mutation.mutate,
    logoutAsync: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error?.message || null,
    isError: mutation.isError,
  };
}
