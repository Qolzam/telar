/**
 * Auth Feature Client
 * 
 * React Query hooks for authentication using the Telar SDK.
 * All hooks use the SDK for API communication.
 */

'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { sdk } from '@/lib/sdk';
import type {
  LoginRequest,
  SignupRequest,
  SignupResponse,
  ForgotPasswordRequest,
  ResetPasswordRequest,
  ChangePasswordRequest,
  VerifyEmailRequest,
  SessionData,
} from '@telar/sdk';

// ============================================================================
// Session Management
// ============================================================================

export const SESSION_QUERY_KEY = ['auth', 'session'] as const;

/**
 * Fetch session data
 */
async function fetchSession(): Promise<SessionData | null> {
  try {
    const data = await sdk.auth.getSession();
    return data;
  } catch (error) {
    console.error('[useSession] Error fetching session:', error);
    return null;
  }
}

/**
 * Hook to get current session and user data
 */
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

// ============================================================================
// Login
// ============================================================================

/**
 * Hook to login user
 */
export function useLogin() {
  const router = useRouter();
  const { invalidateSession } = useSession();

  const mutation = useMutation({
    mutationFn: (credentials: LoginRequest) => sdk.auth.login(credentials),
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

// ============================================================================
// Signup
// ============================================================================

/**
 * Hook to register new user
 */
export function useSignup() {
  const [verificationId, setVerificationId] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: (data: SignupRequest) => sdk.auth.signup(data),
    onSuccess: (data: SignupResponse) => {
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

// ============================================================================
// Logout
// ============================================================================

/**
 * Hook to logout user
 */
export function useLogout() {
  const router = useRouter();
  const { clearSession } = useSession();

  const mutation = useMutation({
    mutationFn: () => sdk.auth.logout(),
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

// ============================================================================
// Forgot Password
// ============================================================================

/**
 * Hook to request password reset
 */
export function useForgotPassword() {
  const [emailSent, setEmailSent] = useState(false);

  const mutation = useMutation({
    mutationFn: (data: ForgotPasswordRequest) => sdk.auth.forgotPassword(data),
    onSuccess: () => {
      setEmailSent(true);
      console.log('[ForgotPassword] ✅ Reset email sent');
    },
    onError: (error: Error) => {
      console.error('[ForgotPassword] Failed:', error.message);
    },
  });

  return {
    requestReset: mutation.mutate,
    requestResetAsync: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error?.message || null,
    isError: mutation.isError,
    isSuccess: mutation.isSuccess,
    emailSent,
    reset: mutation.reset,
  };
}

// ============================================================================
// Reset Password
// ============================================================================

/**
 * Hook to reset password with token
 */
export function useResetPassword() {
  const router = useRouter();

  const mutation = useMutation({
    mutationFn: (data: ResetPasswordRequest) => sdk.auth.resetPassword(data),
    onSuccess: () => {
      console.log('[ResetPassword] ✅ Password reset successfully');
      router.push('/login?message=password_reset_success');
    },
    onError: (error: Error) => {
      console.error('[ResetPassword] Failed:', error.message);
    },
  });

  return {
    resetPassword: mutation.mutate,
    resetPasswordAsync: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error?.message || null,
    isError: mutation.isError,
    isSuccess: mutation.isSuccess,
    reset: mutation.reset,
  };
}

// ============================================================================
// Change Password
// ============================================================================

/**
 * Hook to change password for authenticated user
 */
export function useChangePassword() {
  const mutation = useMutation({
    mutationFn: (data: ChangePasswordRequest) => sdk.auth.changePassword(data),
    onSuccess: () => {
      console.log('[ChangePassword] ✅ Password changed successfully');
    },
    onError: (error: Error) => {
      console.error('[ChangePassword] Failed:', error.message);
    },
  });

  return {
    changePassword: mutation.mutate,
    changePasswordAsync: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error?.message || null,
    isError: mutation.isError,
    isSuccess: mutation.isSuccess,
    reset: mutation.reset,
  };
}

// ============================================================================
// Email Verification
// ============================================================================

/**
 * Hook to verify email with code
 */
export function useVerifyEmail() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (data: VerifyEmailRequest) => sdk.auth.verifyEmail(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SESSION_QUERY_KEY });
      
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

