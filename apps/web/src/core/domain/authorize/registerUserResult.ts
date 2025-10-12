/**
 * Result of user registration
 */
export interface RegisterUserResult {
  /** Whether the registration was successful */
  success: boolean;
  /** Message about the registration result */
  message: string;
  /** User ID if registration was successful */
  userId?: string;
}
