/**
 * Login user interface
 */
export interface LoginUser {
  /** User ID */
  uid: string;
  /** User email */
  email: string;
  /** User display name */
  displayName: string;
  /** User avatar URL */
  avatar?: string;
  /** Whether the user's email is verified */
  emailVerified: boolean;
}
