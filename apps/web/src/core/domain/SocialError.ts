/**
 * Standardized error class for Telar Social Engine
 */
export class SocialError extends Error {
    /**
     * Error code identifying the specific error type
     */
    code: string;
    context?: string;

    /**
     * Creates a new SocialError
     *
     * @param message Human-readable error message
     * @param code Error code identifying the specific error type
     */
    constructor(message: string, code: string) {
        super(message);
        this.name = 'SocialError';
        this.code = code;
        
        // This is necessary for proper instanceof checks in TypeScript
        Object.setPrototypeOf(this, SocialError.prototype);
    }

    /**
     * Add context to the error
     */
    withContext(context: string): SocialError {
        this.context = context;
        return this;
    }

    /**
     * Creates a standardized authentication error
     *
     * @param message More specific error message
     * @returns SocialError with auth error code
     */
    static auth(message: string): SocialError {
        return new SocialError(message, 'AUTH_ERROR');
    }

    /**
     * Creates a standardized network error
     *
     * @param message More specific error message
     * @returns SocialError with network error code
     */
    static network(message: string): SocialError {
        return new SocialError(message, 'NETWORK_ERROR');
    }

    /**
     * Creates a standardized API error
     *
     * @param message More specific error message
     * @returns SocialError with api error code
     */
    static api(message: string): SocialError {
        return new SocialError(message, 'API_ERROR');
    }

    /**
     * Creates a standardized validation error
     *
     * @param message More specific error message
     * @returns SocialError with validation error code
     */
    static validation(message: string): SocialError {
        return new SocialError(message, 'VALIDATION_ERROR');
    }

    /**
     * Create an unknown error
     */
    static unknown(message = 'An unknown error occurred'): SocialError {
        return new SocialError(message, 'UNKNOWN_ERROR');
    }

    /**
     * Create a SocialError from any error type
     */
    static from(error: unknown): SocialError {
        if (error instanceof SocialError) {
            return error;
        }

        if (error instanceof Error) {
            return new SocialError(error.message, 'UNKNOWN_ERROR');
        }

        return SocialError.unknown(String(error));
    }

    /**
     * Helper method to extract the most useful error message from various error types
     *
     * @param error The error to extract a message from
     * @returns A user-friendly error message
     */
    static getErrorMessage(error: unknown): string {
        if (error instanceof SocialError) {
            return error.message;
        }

        if (error instanceof Error) {
            return error.message;
        }

        if (typeof error === 'string') {
            return error;
        }

        if (typeof error === 'object' && error !== null) {
            const errorObj = error as any;
            if (errorObj.response?.data?.error) {
                return errorObj.response.data.error;
            }
            if (errorObj.message) {
                return errorObj.message;
            }
        }

        return 'An unknown error occurred';
    }
}
