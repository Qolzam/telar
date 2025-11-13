'use client';

import { useState, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
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
  InputAdornment,
  IconButton,
  Checkbox,
  FormControlLabel,
  Alert,
  CircularProgress,
  useTheme,
  Button,
} from '@mui/material';
import { Visibility, VisibilityOff } from '@mui/icons-material';
import { useLogin } from '@/features/auth/client';
import SocialLoginButtons from '@/features/auth/components/SocialLoginButtons';
import { mapAuthError } from '@/features/auth/utils/errorMapper';

function LoginFormContent() {
  const { t } = useTranslation(['auth', 'validation']);
  const searchParams = useSearchParams();
  const theme = useTheme();
  const { loginAsync } = useLogin();
  const [showPassword, setShowPassword] = useState(false);
  const urlError = searchParams.get('error');
  const urlMessage = searchParams.get('message');

  const togglePasswordVisibility = () => {
    setShowPassword((prev) => !prev);
  };

  const validationSchemas = useValidationSchema();

  const loginSchema = z.object({
    email: validationSchemas.email,
    password: validationSchemas.password(),
    rememberMe: z.boolean().optional(),
  });

  type LoginFormData = z.infer<typeof loginSchema>;

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
    clearErrors,
    watch,
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: '',
      password: '',
      rememberMe: false,
    },
  });

  const rememberMeValue = watch('rememberMe');

  const onSubmit = async (data: LoginFormData) => {
    try {
      clearErrors('root');
      
      await loginAsync({ 
        username: data.email, 
        password: data.password 
      });
      
      if (data.rememberMe) {
        localStorage.setItem('rememberMe', 'true');
      } else {
        localStorage.removeItem('rememberMe');
      }
    } catch (error: unknown) {
      console.error('[Login] Login failed:', error);
      const errorMessage = mapAuthError(error, 'login');
      setError('root', { message: errorMessage });
    }
  };

  const enabledOAuthLogin = false; // TODO: Move to config

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
            {t('login.title')}
          </Typography>
          <Typography variant="body1" color="text.secondary">
            {t('login.subtitle')}
          </Typography>
        </Box>

        {enabledOAuthLogin && (
          <>
            <SocialLoginButtons disabled={isSubmitting} />
            <Divider sx={{ mb: 3, mt: 3 }}>
              <Typography variant="body2" color="text.secondary">
                {t('login.divider')}
              </Typography>
            </Divider>
          </>
        )}

        {urlError && (
          <Alert severity="warning" sx={{ mb: 3 }}>
            {urlError === 'invalid_token' && t('login.errors.invalidToken')}
            {urlError === 'expired_token' && t('login.errors.expiredToken')}
            {urlError === 'verification_failed' && t('login.errors.verificationFailed')}
          </Alert>
        )}
        
        {urlMessage === 'password_reset_success' && (
          <Alert severity="success" sx={{ mb: 3 }}>
            {t('login.messages.passwordResetSuccess')}
          </Alert>
        )}

        {errors.root && (
          <Alert 
            severity="error" 
            sx={{ mb: 3 }}
            onClose={() => clearErrors('root')}
          >
            {errors.root.message}
          </Alert>
        )}

        <TextField
          fullWidth
          autoComplete="email"
          type="email"
          label={t('login.fields.email')}
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
          autoComplete="current-password"
          type={showPassword ? 'text' : 'password'}
          label={t('login.fields.password')}
          {...register('password')}
          error={!!errors.password}
          helperText={errors.password?.message}
          disabled={isSubmitting}
          variant="outlined"
          margin="normal"
          onFocus={() => clearErrors('root')}
          InputProps={{
            endAdornment: (
              <InputAdornment position="end">
                <IconButton
                  onClick={togglePasswordVisibility}
                  edge="end"
                  aria-label={showPassword ? t('login.actions.hidePassword') : t('login.actions.showPassword')}
                >
                  {showPassword ? <VisibilityOff /> : <Visibility />}
                </IconButton>
              </InputAdornment>
            ),
          }}
        />

        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', my: 2 }}>
          <FormControlLabel
            control={
              <Checkbox
                {...register('rememberMe')}
                checked={rememberMeValue}
                color="primary"
                disabled={isSubmitting}
              />
            }
            label={t('login.fields.rememberMe')}
          />
          <Link
            href="/forgot-password"
            style={{
              fontSize: '0.875rem',
              color: theme.palette.primary.main,
              textDecoration: 'none',
            }}
          >
            {t('login.actions.forgotPassword')}
          </Link>
        </Box>

        <Button
          fullWidth
          size="large"
          type="submit"
          variant="contained"
          disabled={isSubmitting}
          sx={{ mt: 2, mb: 3, py: 1.5 }}
        >
          {isSubmitting ? t('login.actions.submitting') : t('login.actions.submit')}
        </Button>

        <Box sx={{ mt: 4, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            {t('login.footer.noAccount')}{' '}
            <Link 
              href="/signup"
              style={{
                color: theme.palette.primary.main,
                textDecoration: 'none',
                fontWeight: 500,
              }}
            >
              {t('login.footer.signUp')}
            </Link>
          </Typography>
        </Box>
      </Box>
  );
}

export default function LoginForm() {
  return (
    <Suspense fallback={<CircularProgress />}>
      <LoginFormContent />
    </Suspense>
  );
}
