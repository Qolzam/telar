import { useTranslation } from 'react-i18next';
import { z } from 'zod';

/**
 * Hook that provides reusable, translated Zod validation schemas
 * Use this hook in form components to get consistent, translated validation messages
 */
export function useValidationSchema() {
  const { t } = useTranslation('validation');
  
  return {
    /**
     * Email validation schema
     */
    email: z.string()
      .min(1, t('email.required'))
      .email(t('email.invalid')),
    
    /**
     * Password validation schema with optional min length
     * @param min - Minimum password length (default: 8)
     */
    password: (min = 8) => z.string()
      .min(1, t('password.required'))
      .min(min, t('password.minLength', { min })),
    
    /**
     * Name validation schema
     */
    name: z.string()
      .min(1, t('name.required')),
    
    /**
     * First/Last name validation with min/max
     * @param min - Minimum length (default: 2)
     * @param max - Maximum length (default: 50)
     */
    nameWithLength: (min = 2, max = 50) => z.string()
      .min(min, t('name.minLength', { min }))
      .max(max, t('name.maxLength', { max })),
    
    /**
     * URL validation schema
     */
    url: z.string()
      .url(t('url.invalid'))
      .optional()
      .or(z.literal('')),
    
    /**
     * Confirm password validation schema
     */
    confirmPassword: z.string()
      .min(1, t('password.required')),
    
    /**
     * Helper for password match validation
     */
    passwordMatchError: t('password.mismatch'),
  };
}
