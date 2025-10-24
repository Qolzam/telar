/**
 * Auth SDK Module
 * 
 * Provides authentication functions that call the Next.js BFF routes.
 * All auth operations go through the BFF for httpOnly cookie management.
 */

import { ApiClient } from './client';
import { ENDPOINTS } from './config';
import type {
  LoginRequest,
  SignupRequest,
  SignupResponse,
  ForgotPasswordRequest,
  ResetPasswordRequest,
  ChangePasswordRequest,
  VerifyEmailRequest,
  ResendVerificationRequest,
  SessionData,
} from './types';

/**
 * Auth API interface
 */
export interface IAuthApi {
  /**
   * Login with username and password
   * Sets httpOnly session cookie on success
   */
  login(credentials: LoginRequest): Promise<void>;

  /**
   * Logout current user
   * Clears httpOnly session cookie
   */
  logout(): Promise<void>;

  /**
   * Register a new user account
   * Returns verification ID for email verification
   */
  signup(data: SignupRequest): Promise<SignupResponse>;

  /**
   * Request password reset email
   */
  forgotPassword(data: ForgotPasswordRequest): Promise<void>;

  /**
   * Reset password with token from email
   */
  resetPassword(data: ResetPasswordRequest): Promise<void>;

  /**
   * Change password for authenticated user
   */
  changePassword(data: ChangePasswordRequest): Promise<void>;

  /**
   * Verify email with code
   */
  verifyEmail(data: VerifyEmailRequest): Promise<void>;

  /**
   * Resend verification email with new code
   */
  resendVerification(data: ResendVerificationRequest): Promise<void>;

  /**
   * Get current session data
   * Returns user info if authenticated
   */
  getSession(): Promise<SessionData>;
}

/**
 * Create Auth API instance
 */
export const authApi = (client: ApiClient): IAuthApi => ({
  login: async (credentials: LoginRequest): Promise<void> => {
    await client.post(ENDPOINTS.AUTH.LOGIN, credentials);
  },

  logout: async (): Promise<void> => {
    await client.post(ENDPOINTS.AUTH.LOGOUT);
  },

  signup: async (data: SignupRequest): Promise<SignupResponse> => {
    return client.post<SignupResponse>(ENDPOINTS.AUTH.SIGNUP, data);
  },

  forgotPassword: async (data: ForgotPasswordRequest): Promise<void> => {
    await client.post(ENDPOINTS.AUTH.FORGOT_PASSWORD, data);
  },

  resetPassword: async (data: ResetPasswordRequest): Promise<void> => {
    await client.post(ENDPOINTS.AUTH.RESET_PASSWORD, data);
  },

  changePassword: async (data: ChangePasswordRequest): Promise<void> => {
    await client.post(ENDPOINTS.AUTH.CHANGE_PASSWORD, data);
  },

  verifyEmail: async (data: VerifyEmailRequest): Promise<void> => {
    await client.post(ENDPOINTS.AUTH.VERIFY_EMAIL, data);
  },

  resendVerification: async (data: ResendVerificationRequest): Promise<void> => {
    await client.post(ENDPOINTS.AUTH.RESEND_VERIFICATION, data);
  },

  getSession: async (): Promise<SessionData> => {
    return client.get<SessionData>(ENDPOINTS.AUTH.SESSION);
  },
});

