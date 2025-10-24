'use client';

import { useState, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import * as Yup from 'yup';
import { useFormik, Form, FormikProvider } from 'formik';
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
} from '@mui/material';
import { LoadingButton } from '@mui/lab';
import { Visibility, VisibilityOff } from '@mui/icons-material';
import { useLogin } from '@/features/auth/client';
import SocialLoginButtons from '@/features/auth/components/SocialLoginButtons';
import { mapAuthError } from '@/features/auth/utils/errorMapper';

function LoginFormContent() {
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
      .email('Email must be a valid email address')
      .required('Email is required'),
    password: Yup.string()
      .required('Password is required'),
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
            Welcome Back
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Sign in to continue to Telar
          </Typography>
        </Box>

        {enabledOAuthLogin && (
          <>
            <SocialLoginButtons disabled={isSubmitting} />
            <Divider sx={{ mb: 3, mt: 3 }}>
              <Typography variant="body2" color="text.secondary">
                or continue with email
              </Typography>
            </Divider>
          </>
        )}

        {urlError && (
          <Alert severity="warning" sx={{ mb: 3 }}>
            {urlError === 'invalid_token' && 'Your session is invalid. Please log in again.'}
            {urlError === 'expired_token' && 'Your session has expired. Please log in again.'}
            {urlError === 'verification_failed' && 'Session verification failed. Please log in again.'}
          </Alert>
        )}
        
        {urlMessage === 'password_reset_success' && (
          <Alert severity="success" sx={{ mb: 3 }}>
            Password reset successfully! You can now log in with your new password.
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
          label="Email Address"
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
          label="Password"
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
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
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
            label="Remember me"
          />
          <Link
            href="/forgot-password"
            style={{
              fontSize: '0.875rem',
              color: theme.palette.primary.main,
              textDecoration: 'none',
            }}
          >
            Forgot password?
          </Link>
        </Box>

        <LoadingButton
          fullWidth
          size="large"
          type="submit"
          variant="contained"
          loading={isSubmitting}
          loadingIndicator="Signing in..."
          sx={{ mt: 2, mb: 3, py: 1.5 }}
        >
          Sign In
        </LoadingButton>

        <Box sx={{ mt: 4, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            Don&apos;t have an account?{' '}
            <Link 
              href="/signup"
              style={{
                color: theme.palette.primary.main,
                textDecoration: 'none',
                fontWeight: 500,
              }}
            >
              Sign up
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
