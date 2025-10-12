'use client';

import { useState } from 'react';
import Link from 'next/link';
import * as Yup from 'yup';
import { useFormik, Form, FormikProvider } from 'formik';
import {
  TextField,
  Button,
  Typography,
  Box,
  Divider,
  Stack,
  Alert,
  CircularProgress,
  useTheme,
} from '@mui/material';
import PasswordStrengthBar from 'react-password-strength-bar';
import { useSignup } from '@/features/auth/client';
import SocialLoginButtons from '@/features/auth/components/SocialLoginButtons';

interface SignupFormProps {
  onSuccess?: (verificationId: string, email: string) => void;
}

export default function SignupForm({ onSuccess }: SignupFormProps) {
  const theme = useTheme();
  const { signup } = useSignup();
  const [verificationId, setVerificationId] = useState<string | null>(null);

  const RegisterSchema = Yup.object().shape({
    firstName: Yup.string()
      .min(2, 'First name is too short')
      .max(50, 'First name is too long')
      .required('First name is required'),
    lastName: Yup.string()
      .min(2, 'Last name is too short')
      .max(50, 'Last name is too long')
      .required('Last name is required'),
    email: Yup.string()
      .email('Email must be a valid email address')
      .required('Email is required'),
    password: Yup.string()
      .required('Password is required')
      .min(8, 'Password must be at least 8 characters'),
  });

  const formik = useFormik({
    initialValues: {
      firstName: '',
      lastName: '',
      email: '',
      password: '',
    },
    validationSchema: RegisterSchema,
    onSubmit: async (values, { setStatus, setSubmitting }) => {
      try {
        console.log('[Signup] Starting registration process...', { email: values.email });
        setStatus(null);
        
        const fullName = `${values.firstName} ${values.lastName}`;
        const result = await signup({ 
          fullName, 
          email: values.email, 
          newPassword: values.password 
        });
        
        console.log('[Signup] Registration successful!');
        
        if (result?.verificationId) {
          setVerificationId(result.verificationId);
          
          if (onSuccess) {
            onSuccess(result.verificationId, values.email);
          }
        }
        
        setSubmitting(false);
      } catch (error: unknown) {
        console.error('[Signup] Registration failed:', error);
        const errorMessage = error instanceof Error ? error.message : 'Registration failed';
        setStatus({ error: errorMessage });
        setSubmitting(false);
      }
    },
  });

  const { status, errors, touched, isSubmitting, handleSubmit, getFieldProps, setStatus } = formik;

  if (verificationId && onSuccess) {
    return null;
  }

  return (
    <FormikProvider value={formik}>
      <Box component={Form} sx={{ width: '100%' }} autoComplete="off" noValidate onSubmit={handleSubmit}>
        <Box sx={{ mb: 4, textAlign: 'center' }}>
          <Typography variant="h4" component="h1" gutterBottom>
            Create Account
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Join Telar and connect with friends
          </Typography>
        </Box>

        {status && status.error && (
          <Alert severity="error" sx={{ mb: 3 }} onClose={() => setStatus(null)}>
            {status.error}
          </Alert>
        )}

        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 2 }}>
          <TextField
            fullWidth
            label="First Name"
            {...getFieldProps('firstName')}
            error={Boolean(touched.firstName && errors.firstName)}
            helperText={touched.firstName && errors.firstName}
            disabled={isSubmitting}
            variant="outlined"
            margin="normal"
            onFocus={() => setStatus(null)}
          />
          <TextField
            fullWidth
            label="Last Name"
            {...getFieldProps('lastName')}
            error={Boolean(touched.lastName && errors.lastName)}
            helperText={touched.lastName && errors.lastName}
            disabled={isSubmitting}
            variant="outlined"
            margin="normal"
            onFocus={() => setStatus(null)}
          />
        </Stack>

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
          autoComplete="new-password"
          type="password"
          label="Password"
          {...getFieldProps('password')}
          error={Boolean(touched.password && errors.password)}
          helperText={touched.password && errors.password}
          disabled={isSubmitting}
          variant="outlined"
          margin="normal"
          onFocus={() => setStatus(null)}
        />

        {formik.values.password && (
          <Box sx={{ mt: 1, mb: 2 }}>
            <PasswordStrengthBar 
              password={formik.values.password}
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
          {isSubmitting ? (
            <>
              <CircularProgress size={20} sx={{ mr: 1, color: 'inherit' }} />
              Creating account...
            </>
          ) : (
            'Sign Up'
          )}
        </Button>

        <Divider sx={{ my: 3 }}>
          <Typography variant="body2" color="text.secondary">
            or sign up with
          </Typography>
        </Divider>

        <SocialLoginButtons disabled={isSubmitting} />

        <Box sx={{ mt: 4, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            Already have an account?{' '}
            <Link 
              href="/login"
              style={{
                color: theme.palette.primary.main,
                textDecoration: 'none',
                fontWeight: 500,
              }}
            >
              Sign in
            </Link>
          </Typography>
        </Box>

        <Box sx={{ mt: 2, textAlign: 'center' }}>
          <Typography variant="caption" color="text.secondary">
            By signing up, you agree to our{' '}
            <Link 
              href="/terms"
              style={{
                color: theme.palette.primary.main,
                textDecoration: 'none',
              }}
            >
              Terms & Conditions
            </Link>
          </Typography>
        </Box>
      </Box>
    </FormikProvider>
  );
}
