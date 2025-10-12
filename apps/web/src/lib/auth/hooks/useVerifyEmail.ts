'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';

interface VerifyRequest {
  verificationId: string;
  code: string;
}

async function verifyEmailMutation(data: VerifyRequest): Promise<void> {
  const response = await fetch('/api/auth/verify', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Verification failed');
  }
}

export function useVerifyEmail() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: verifyEmailMutation,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['session'] });
      
      console.log('[Verify] ✅ Email verified, redirecting to dashboard');
      
      router.push('/dashboard');
    },
    onError: (error: Error) => {
      console.error('[Verify] Verification failed:', error.message);
    },
  });

  return {
    verify: mutation.mutate,
    verifyAsync: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error?.message || null,
    isError: mutation.isError,
    isSuccess: mutation.isSuccess,
    reset: mutation.reset,
  };
}

