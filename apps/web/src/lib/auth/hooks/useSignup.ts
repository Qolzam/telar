'use client';

import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';

interface SignupRequest {
  fullName: string;
  email: string;
  newPassword: string;
}

interface SignupResponse {
  verificationId: string;
  message: string;
}

async function signupMutation(data: SignupRequest): Promise<SignupResponse> {
  const response = await fetch('/api/auth/signup', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Signup failed');
  }

  return response.json();
}

export function useSignup() {
  const [verificationId, setVerificationId] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: signupMutation,
    onSuccess: (data) => {
      setVerificationId(data.verificationId);
      console.log('[Signup] ✅ Registration successful, verification ID:', data.verificationId);
    },
    onError: (error: Error) => {
      console.error('[Signup] Signup failed:', error.message);
    },
  });

  const signupWithReturn = async (data: SignupRequest): Promise<SignupResponse> => {
    return mutation.mutateAsync(data);
  };

  return {
    signup: signupWithReturn,
    signupAsync: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error?.message || null,
    isError: mutation.isError,
    isSuccess: mutation.isSuccess,
    verificationId,
    reset: mutation.reset,
  };
}

