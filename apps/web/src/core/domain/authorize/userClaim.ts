/**
 * User claim interface
 */
export interface UserClaim {
  /** User ID */
  uid: string;
  /** User email */
  email: string;
  /** User display name */
  displayName: string;
  /** Whether the user's email is verified */
  emailVerified: boolean;
}
