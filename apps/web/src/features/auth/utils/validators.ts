import { VALIDATION_RULES, VALIDATION_MESSAGES } from '../constants/validation';

export function validateEmail(email: string): boolean {
  if (!email) return false;
  return VALIDATION_RULES.EMAIL_REGEX.test(email);
}

export function validatePassword(password: string): {
  valid: boolean;
  error?: string;
} {
  if (!password) {
    return {
      valid: false,
      error: VALIDATION_MESSAGES.PASSWORD_REQUIRED,
    };
  }

  if (password.length < VALIDATION_RULES.PASSWORD_MIN_LENGTH) {
    return {
      valid: false,
      error: VALIDATION_MESSAGES.PASSWORD_TOO_SHORT,
    };
  }

  if (password.length > VALIDATION_RULES.PASSWORD_MAX_LENGTH) {
    return {
      valid: false,
      error: VALIDATION_MESSAGES.PASSWORD_TOO_LONG,
    };
  }

  return { valid: true };
}

export function validatePasswordMatch(
  password: string,
  confirmPassword: string
): boolean {
  return password === confirmPassword && password.length > 0;
}

export function validateVerificationCode(code: string): boolean {
  if (!code) return false;
  return (
    code.length === VALIDATION_RULES.VERIFICATION_CODE_LENGTH &&
    /^\d+$/.test(code)
  );
}

export function validateFullName(name: string): boolean {
  if (!name) return false;
  return name.trim().length >= 2;
}

export function validateResetToken(token: string): boolean {
  if (!token) return false;
  return token.length >= 32;
}

export function getEmailError(email: string): string | null {
  if (!email) {
    return VALIDATION_MESSAGES.EMAIL_REQUIRED;
  }
  if (!validateEmail(email)) {
    return VALIDATION_MESSAGES.EMAIL_INVALID;
  }
  return null;
}

export function getPasswordError(password: string): string | null {
  const result = validatePassword(password);
  return result.valid ? null : result.error || null;
}

export function getPasswordMatchError(
  password: string,
  confirmPassword: string
): string | null {
  if (!confirmPassword) {
    return VALIDATION_MESSAGES.PASSWORD_REQUIRED;
  }
  if (!validatePasswordMatch(password, confirmPassword)) {
    return VALIDATION_MESSAGES.PASSWORDS_DONT_MATCH;
  }
  return null;
}

