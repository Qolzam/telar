import { UserClaim } from '@/core/domain/authorize/userClaim';
import { UserRegisterModel } from '@/models/users/userRegisterModel';
import { LoginUser } from '@/core/domain/authorize/loginUser';
import { OAuthType } from '@/core/domain/authorize/oauthType';
import { RegisterUserResult } from '@/core/domain/authorize/registerUserResult';
import { TelarLoginResponse } from '@/core/domain/authorize/authUserProfileModel';

/**
 * Authentication service interface
 */
export interface IAuthorizeService {
    /**
     * Login the user
     */
    login: (email: string, password: string) => Promise<TelarLoginResponse>;

    /**
     * Whether user is loged in or not
     */
    isLoggedin: () => boolean;

    /**
     * Get user auth
     */
    getUserAuth: () => any;

    /**
     * Get access token
     */
    getAccessToken: () => string | null;

    /**
     * Login by token
     */
    loginByToken: (token: string) => Promise<LoginUser | null>;

    /**
     * Logs out the user
     */
    logout: () => void;

    /**
     * Whether user is login or not
     */
    isUserUserVerified: () => boolean;

    /**
     * Get user claim
     */
    getUserClaim: () => Promise<UserClaim>;

    /**
     * Update user password
     */
    updatePassword: (newPassword: string, confirmPassword: string) => Promise<void>;

    /**
     * Get register user token
     */
    getUserRegisterToken: (user: UserRegisterModel) => Promise<any>;

    /**
     * Verify user register code
     * 
     * Updated signature for EmailVerification component test
     */
    verifyUserRegisterCode: (code: string, registerToken: string) => Promise<string | void>;

    /**
     * Change password
     */
    changePassword: (currentPassword: string, newPassword: string, confirmPassword: string) => Promise<void>;

    /**
     * Register new user
     */
    registerUser: (user: UserRegisterModel) => Promise<RegisterUserResult>;

    /**
     * Reset user password
     */
    resetPassword: (email: string) => Promise<void>;

    /**
     * Send email verification
     */
    sendEmailVerification: (value: any) => Promise<string>;

    /**
     * Login user by OAuth authentication
     */
    loginWithOAuth: (type: OAuthType) => Promise<LoginUser>;

    /**
     * Send sms verfication
     */
    sendSmsVerification: (phoneNumber: string, value: any) => Promise<string>;

    /**
     * Send sms verfication
     */
    sendResetPasswordVerification: (email: string, value: any) => Promise<string>;

    /**
     * Confirm verfication phone
     */
    confirmVerificationPhone: (code: string, verifyId: string, phoneNumber: string) => Promise<any>;

    /**
     * Confirm verfication email
     */
    confirmVerificationEmail: (code: string, verifyId: string) => Promise<any>;

    /**
     * Confirm verfication code
     */
    confirmResetPassword: (code: string, verifyId: string, email: string) => Promise<any>;
    
    /**
     * Verify reset code
     * @param email User email
     * @param code Verification code
     */
    verifyResetCode?: (email: string, code: string) => Promise<string>;
    
    /**
     * Update password with token (for password reset flow)
     * @param email User email
     * @param token Reset token
     * @param password New password
     */
    updatePasswordWithToken?: (email: string, token: string, password: string) => Promise<void>;
    
    /**
     * Resend verification email
     * @param email User email address
     */
    resendVerificationEmail?: (email: string) => Promise<void>;
}
