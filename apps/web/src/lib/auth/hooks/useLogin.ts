'use client';

import { useMutation } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useSession } from './useSession';
import type { LoginRequest } from '../types';

interface LoginError {
  error: string;
}

async function loginMutation(credentials: LoginRequest): Promise<void> {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(credentials),
    credentials: 'include',
  });

  if (!response.ok) {
    const error: LoginError = await response.json();
    throw new Error(error.error || 'Login failed');
  }

  return;
}

export function useLogin() {
  const router = useRouter();
  const { invalidateSession } = useSession();

  const mutation = useMutation({
    mutationFn: loginMutation,
    onSuccess: () => {
      invalidateSession();
      
      const searchParams = new URLSearchParams(window.location.search);
      const from = searchParams.get('from') || '/dashboard';
      router.push(from);
    },
    onError: (error: Error) => {
      console.error('[Login] Login failed:', error.message);
    },
  });

  return {
    login: mutation.mutate,
    loginAsync: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error?.message || null,
    isError: mutation.isError,
    isSuccess: mutation.isSuccess,
    reset: mutation.reset,
  };
}
