'use client';

import { useState, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import * as Yup from 'yup';
import { useFormik, Form, FormikProvider } from 'formik';
import { useTranslation } from 'react-i18next';
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

  const LoginSchema = Yup.object().shape({
    email: Yup.string()
      .email(t('validation:email.invalid'))
      .required(t('validation:email.required')),
    password: Yup.string()
      .required(t('validation:password.required')),
  });

  const formik = useFormik({
    initialValues: {
      email: '',
      password: '',
      rememberMe: false,
    },
    validationSchema: LoginSchema,
    onSubmit: async (values, { setStatus, setSubmitting }) => {
      try {
        setStatus(null);
        
        await loginAsync({ 
          username: values.email, 
          password: values.password 
        });
        
        if (values.rememberMe) {
          localStorage.setItem('rememberMe', 'true');
        } else {
          localStorage.removeItem('rememberMe');
        }
        
        if (setSubmitting) {
          setSubmitting(false);
        }
      } catch (error: unknown) {
        console.error('[Login] Login failed:', error);
        
        if (setSubmitting) {
          setSubmitting(false);
        }
        
        const errorMessage = mapAuthError(error, 'login');
        setStatus({ error: errorMessage });
      }
    },
  });

  const { status, errors, touched, isSubmitting, handleSubmit, getFieldProps, setStatus } = formik;

  const enabledOAuthLogin = false; // TODO: Move to config

  return (
    <FormikProvider value={formik}>
      <Box component={Form} sx={{ width: '100%' }} autoComplete="off" noValidate onSubmit={handleSubmit}>
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

        {status && status.error && (
          <Alert severity="error" sx={{ mb: 3 }}>
            {status.error}
          </Alert>
        )}

        <TextField
          fullWidth
          autoComplete="email"
          type="email"
          label={t('login.fields.email')}
          {...getFieldProps('email')}
          error={Boolean(touched.email && errors.email)}
          helperText={touched.email && errors.email}
          disabled={isSubmitting}
          variant="outlined"
          margin="normal"
          onFocus={() => setStatus(null)}
        />

        <TextField
          fullWidth
          autoComplete="current-password"
          type={showPassword ? 'text' : 'password'}
          label={t('login.fields.password')}
          {...getFieldProps('password')}
          error={Boolean(touched.password && errors.password)}
          helperText={touched.password && errors.password}
          disabled={isSubmitting}
          variant="outlined"
          margin="normal"
          onFocus={() => setStatus(null)}
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
                {...getFieldProps('rememberMe')}
                checked={formik.values.rememberMe}
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
    </FormikProvider>
  );
}

export default function LoginForm() {
  return (
    <Suspense fallback={<CircularProgress />}>
      <LoginFormContent />
    </Suspense>
  );
}
