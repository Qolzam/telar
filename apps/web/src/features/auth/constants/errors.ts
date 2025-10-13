export const AUTH_ERRORS = {
  INVALID_CREDENTIALS: 'Invalid email or password',
  USER_NOT_FOUND: 'User not found',
  EMAIL_ALREADY_EXISTS: 'Email already registered',
  WEAK_PASSWORD: 'Password must be at least 8 characters',
  PASSWORDS_DONT_MATCH: 'Passwords do not match',
  INVALID_EMAIL: 'Invalid email address',
  VERIFICATION_FAILED: 'Verification code is invalid or expired',
  SESSION_EXPIRED: 'Your session has expired. Please log in again.',
  UNAUTHORIZED: 'You must be logged in to access this page',
  NETWORK_ERROR: 'Network error. Please check your connection.',
  SERVER_ERROR: 'Server error. Please try again later.',
} as const;

export type AuthError = typeof AUTH_ERRORS[keyof typeof AUTH_ERRORS];

