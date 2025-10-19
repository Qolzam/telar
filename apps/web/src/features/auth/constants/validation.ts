export const VALIDATION_RULES = {
  PASSWORD_MIN_LENGTH: 8,
  PASSWORD_MAX_LENGTH: 128,
  VERIFICATION_CODE_LENGTH: 6,
  EMAIL_REGEX: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
  RESET_TOKEN_LENGTH: 64, // High-entropy token from Go API
} as const;

export const VALIDATION_MESSAGES = {
  EMAIL_REQUIRED: 'Email is required',
  EMAIL_INVALID: 'Please enter a valid email address',
  PASSWORD_REQUIRED: 'Password is required',
  PASSWORD_TOO_SHORT: `Password must be at least ${VALIDATION_RULES.PASSWORD_MIN_LENGTH} characters`,
  PASSWORD_TOO_LONG: `Password must be less than ${VALIDATION_RULES.PASSWORD_MAX_LENGTH} characters`,
  PASSWORDS_DONT_MATCH: 'Passwords do not match',
  FULL_NAME_REQUIRED: 'Full name is required',
  VERIFICATION_CODE_INVALID: `Verification code must be ${VALIDATION_RULES.VERIFICATION_CODE_LENGTH} digits`,
} as const;

