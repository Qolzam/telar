'use client';

import { useState } from 'react';
import Link from 'next/link';
import * as Yup from 'yup';
import { useFormik, Form, FormikProvider } from 'formik';
import { useTranslation } from 'react-i18next';
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

  const RegisterSchema = Yup.object().shape({
    firstName: Yup.string()
      .min(2, t('validation:name.minLength', { min: 2 }))
      .max(50, t('validation:name.maxLength', { max: 50 }))
      .required(t('validation:name.required')),
    lastName: Yup.string()
      .min(2, t('validation:name.minLength', { min: 2 }))
      .max(50, t('validation:name.maxLength', { max: 50 }))
      .required(t('validation:name.required')),
    email: Yup.string()
      .email(t('validation:email.invalid'))
      .required(t('validation:email.required')),
    password: Yup.string()
      .required(t('validation:password.required'))
      .min(8, t('validation:password.minLength', { min: 8 })),
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
        
        const errorMessage = mapAuthError(error, 'signup');
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
            {t('signup.title')}
          </Typography>
          <Typography variant="body1" color="text.secondary">
            {t('signup.subtitle')}
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
            label={t('signup.fields.firstName')}
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
            label={t('signup.fields.lastName')}
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
          label={t('signup.fields.email')}
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
          label={t('signup.fields.password')}
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
    </FormikProvider>
  );
}
