import { ApiError } from '@/lib/api/client';

/**
 * Authentication Error Mapper
 * 
 * This module provides centralized error handling for authentication flows.
 * 
 * TODO:
 * 1. Implement i18n support using i18nKey fields
 * 2. Add more context-specific error messages
 */

export type AuthErrorCode =
  | 'USER_NOT_FOUND'
  | 'AUTHENTICATION_FAILED'
  | 'INVALID_CREDENTIALS'
  | 'USER_ALREADY_EXISTS'
  | 'VALIDATION_FAILED'
  | 'MISSING_REQUIRED_FIELD'
  | 'INVALID_FIELD_VALUE'
  | 'VERIFICATION_FAILED'
  | 'TOKEN_EXPIRED'
  | 'TOKEN_INVALID'
  | 'RATE_LIMIT_EXCEEDED'
  | 'DATABASE_ERROR'
  | 'SYSTEM_ERROR'
  | 'PERMISSION_DENIED'
  | 'TIMEOUT'
  | 'NETWORK_ERROR';

interface ErrorCodeMapping {
  code: AuthErrorCode;
  userMessage: string;
  contextSpecific?: Record<string, string>;
  // future i18n support
  i18nKey?: string;
  contextSpecificI18nKeys?: Record<string, string>;
}

const ERROR_CODE_MAPPINGS: ErrorCodeMapping[] = [
  {
    code: 'USER_NOT_FOUND',
    userMessage: 'No account found with this email.',
    contextSpecific: {
      login: 'No account found with this email. Please check your email or sign up.',
      'forgot-password': 'No account found with this email. Please check your email address.',
    },
  },
  {
    code: 'AUTHENTICATION_FAILED',
    userMessage: 'Authentication failed.',
    contextSpecific: {
      login: 'Incorrect password. Please try again or use "Forgot password" to reset it.',
      'change-password': 'Your current password is incorrect. Please try again.',
    },
  },
  {
    code: 'INVALID_CREDENTIALS',
    userMessage: 'Invalid email or password. Please check your credentials and try again.',
  },
  {
    code: 'USER_ALREADY_EXISTS',
    userMessage: 'An account with this email already exists. Please sign in instead.',
  },
  {
    code: 'VALIDATION_FAILED',
    userMessage: 'Please check your input and try again.',
    contextSpecific: {
      login: 'Please verify your email address before signing in. Check your inbox for the verification code.',
      signup: 'Your password is too weak. Please use at least 8 characters with a mix of letters, numbers, and symbols.',
      'reset-password': 'Passwords do not match. Please make sure both passwords are identical.',
    },
  },
  {
    code: 'MISSING_REQUIRED_FIELD',
    userMessage: 'Please fill in all required fields.',
  },
  {
    code: 'INVALID_FIELD_VALUE',
    userMessage: 'One or more fields contain invalid values. Please check your input.',
  },
  {
    code: 'VERIFICATION_FAILED',
    userMessage: 'Verification failed. Please check the code and try again.',
    contextSpecific: {
      verify: 'Invalid verification code. Please check the code in your email and try again.',
    },
  },
  {
    code: 'TOKEN_EXPIRED',
    userMessage: 'This link has expired. Please request a new one.',
    contextSpecific: {
      'reset-password': 'This reset link has expired. Please request a new password reset.',
    },
  },
  {
    code: 'TOKEN_INVALID',
    userMessage: 'This link is invalid. Please request a new one.',
    contextSpecific: {
      'reset-password': 'This reset link is invalid. Please request a new password reset.',
    },
  },
  {
    code: 'RATE_LIMIT_EXCEEDED',
    userMessage: 'Too many attempts. Please wait a few minutes and try again.',
  },
  {
    code: 'DATABASE_ERROR',
    userMessage: 'Service temporarily unavailable. Please try again in a moment.',
  },
  {
    code: 'SYSTEM_ERROR',
    userMessage: 'An unexpected server error occurred. Please try again later.',
  },
  {
    code: 'PERMISSION_DENIED',
    userMessage: 'You do not have permission to perform this action.',
  },
  {
    code: 'TIMEOUT',
    userMessage: 'The request took too long. Please try again.',
  },
  {
    code: 'NETWORK_ERROR',
    userMessage: 'Unable to connect. Please check your internet connection and try again.',
  },
];


export function mapAuthError(
  error: unknown,
  context: 'login' | 'signup' | 'forgot-password' | 'reset-password' | 'change-password' | 'verify' = 'login'
): string {
  if (!(error instanceof Error)) {
    return 'An unexpected error occurred. Please try again.';
  }

  let errorCode: string | undefined;

  if (error instanceof ApiError && error.code) {
    errorCode = error.code;
  }

  if (errorCode) {
    const mapping = ERROR_CODE_MAPPINGS.find(m => m.code === errorCode);
    if (mapping) {
      if (mapping.contextSpecific && mapping.contextSpecific[context]) {
        return mapping.contextSpecific[context];
      }
      return mapping.userMessage;
    }
  }

  return error.message;
}
