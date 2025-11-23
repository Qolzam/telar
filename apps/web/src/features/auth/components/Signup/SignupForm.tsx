'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useTranslation } from 'react-i18next';
import { useValidationSchema } from '@/lib/i18n/useValidationSchema';
import {
  TextField,
  Typography,
  Box,
  Divider,
  Stack,
  Alert,
  useTheme,
  Button,
} from '@mui/material';
import PasswordStrengthBar from 'react-password-strength-bar';
import { useSignup } from '@/features/auth/client';
import SocialLoginButtons from '@/features/auth/components/SocialLoginButtons';
import { mapAuthError } from '@/features/auth/utils/errorMapper';

interface SignupFormProps {
  onSuccess?: (verificationId: string, email: string) => void;
}

export default function SignupForm({ onSuccess }: SignupFormProps) {
  const { t } = useTranslation(['auth', 'validation']);
  const theme = useTheme();
  const { signup } = useSignup();
  const [verificationId, setVerificationId] = useState<string | null>(null);

  const validationSchemas = useValidationSchema();

  const signupSchema = z.object({
    firstName: validationSchemas.nameWithLength(2, 50),
    lastName: validationSchemas.nameWithLength(2, 50),
    email: validationSchemas.email,
    password: validationSchemas.password(8),
  });

  type SignupFormData = z.infer<typeof signupSchema>;

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
    clearErrors,
    watch,
  } = useForm<SignupFormData>({
    resolver: zodResolver(signupSchema),
    defaultValues: {
      firstName: '',
      lastName: '',
      email: '',
      password: '',
    },
  });

  const passwordValue = watch('password');

  const onSubmit = async (data: SignupFormData) => {
    try {
      clearErrors('root');
      
      const fullName = `${data.firstName} ${data.lastName}`;
      const result = await signup({ 
        fullName, 
        email: data.email, 
        newPassword: data.password 
      });
      
      if (result?.verificationId) {
        setVerificationId(result.verificationId);
        
        if (onSuccess) {
          onSuccess(result.verificationId, data.email);
        }
      }
    } catch (error: unknown) {
      const errorMessage = mapAuthError(error, 'signup');
      setError('root', { message: errorMessage });
    }
  };

  if (verificationId && onSuccess) {
    return null;
  }

  return (
    <Box 
      component="form" 
      sx={{ width: '100%' }} 
      autoComplete="off" 
      noValidate 
      onSubmit={handleSubmit(onSubmit)}
    >
      <Box sx={{ mb: 4, textAlign: 'center' }}>
        <Typography variant="h4" component="h1" gutterBottom>
          {t('signup.title')}
        </Typography>
        <Typography variant="body1" color="text.secondary">
          {t('signup.subtitle')}
        </Typography>
      </Box>

      {errors.root && (
        <Alert 
          severity="error" 
          sx={{ mb: 3 }} 
          onClose={() => clearErrors('root')}
        >
          {errors.root.message}
        </Alert>
      )}

      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 2 }}>
        <TextField
          fullWidth
          label={t('signup.fields.firstName')}
          {...register('firstName')}
          error={!!errors.firstName}
          helperText={errors.firstName?.message}
          disabled={isSubmitting}
          variant="outlined"
          margin="normal"
          onFocus={() => clearErrors('root')}
        />
        <TextField
          fullWidth
          label={t('signup.fields.lastName')}
          {...register('lastName')}
          error={!!errors.lastName}
          helperText={errors.lastName?.message}
          disabled={isSubmitting}
          variant="outlined"
          margin="normal"
          onFocus={() => clearErrors('root')}
        />
      </Stack>

      <TextField
        fullWidth
        autoComplete="email"
        type="email"
        label={t('signup.fields.email')}
        {...register('email')}
        error={!!errors.email}
        helperText={errors.email?.message}
        disabled={isSubmitting}
        variant="outlined"
        margin="normal"
        onFocus={() => clearErrors('root')}
      />

      <TextField
        fullWidth
        autoComplete="new-password"
        type="password"
        label={t('signup.fields.password')}
        {...register('password')}
        error={!!errors.password}
        helperText={errors.password?.message}
        disabled={isSubmitting}
        variant="outlined"
        margin="normal"
        onFocus={() => clearErrors('root')}
      />

      {passwordValue && (
        <Box sx={{ mt: 1, mb: 2 }}>
          <PasswordStrengthBar 
            password={passwordValue}
            minLength={8}
          />
        </Box>
      )}

        <Button
          fullWidth
          size="large"
          type="submit"
          variant="contained"
          disabled={isSubmitting}
          sx={{ mt: 2, mb: 3, py: 1.5 }}
        >
          {isSubmitting ? t('signup.actions.submitting') : t('signup.actions.submit')}
        </Button>

        <Divider sx={{ my: 3 }}>
          <Typography variant="body2" color="text.secondary">
            {t('signup.divider')}
          </Typography>
        </Divider>

        <SocialLoginButtons disabled={isSubmitting} />

        <Box sx={{ mt: 4, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            {t('signup.footer.hasAccount')}{' '}
            <Link 
              href="/login"
              style={{
                color: theme.palette.primary.main,
                textDecoration: 'none',
                fontWeight: 500,
              }}
            >
              {t('signup.footer.signIn')}
            </Link>
          </Typography>
        </Box>

        <Box sx={{ mt: 2, textAlign: 'center' }}>
          <Typography variant="caption" color="text.secondary">
            {t('signup.terms')}
          </Typography>
        </Box>
      </Box>
  );
}
