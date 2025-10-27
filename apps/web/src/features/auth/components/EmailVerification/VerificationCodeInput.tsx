'use client';

import { useState, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslation } from 'react-i18next';
import { 
  Box, 
  Typography, 
  Container, 
  Stack, 
  TextField, 
  CircularProgress, 
  Alert,
  Button,
} from '@mui/material';
import { useVerifyEmail, useResendVerification } from '@/features/auth/client';
import { mapAuthError } from '@/features/auth/utils/errorMapper';

interface VerificationCodeInputProps {
  verificationId: string;
  email: string;
}

enum VerificationState {
  VerifyCode = 'VERIFY_CODE',
  Success = 'SUCCESS',
  Error = 'ERROR',
}

export default function VerificationCodeInput({ verificationId, email }: VerificationCodeInputProps) {
  const { t } = useTranslation('auth');
  const router = useRouter();
  const { verifyAsync, isLoading, error, isError } = useVerifyEmail();
  const { resendAsync, isLoading: resendLoading } = useResendVerification();
  
  const [state, setState] = useState<VerificationState>(VerificationState.VerifyCode);
  const [code, setCode] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [resendSuccess, setResendSuccess] = useState(false);
  const [resendCooldown, setResendCooldown] = useState(0);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);
    
    if (!code) {
      setFormError(t('verification.errors.required'));
      return;
    }
    
    if (code.length < 6) {
      setFormError(t('verification.errors.invalidLength'));
      return;
    }

    try {
      await verifyAsync({ verificationId, code });
      setState(VerificationState.Success);
      
      setTimeout(() => {
        router.push('/dashboard');
      }, 3000);
    } catch (error: unknown) {
      console.error('[Verify] Verification failed:', error);
      setState(VerificationState.Error);
      const userMessage = mapAuthError(error, 'verify');
      setFormError(userMessage);
    }
  };

  const handleResendEmail = async () => {
    if (resendCooldown > 0) {
      return;
    }
    
    setResendSuccess(false);
    setFormError(null);
    
    try {
      await resendAsync({ verificationId });
      setResendSuccess(true);
      setResendCooldown(60);
      
      const countdown = setInterval(() => {
        setResendCooldown(prev => {
          if (prev <= 1) {
            clearInterval(countdown);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
      
      setTimeout(() => setResendSuccess(false), 5000);
    } catch (error: unknown) {
      console.error('[Resend] Failed to resend:', error);
      const userMessage = mapAuthError(error, 'verify');
      setFormError(userMessage);
    }
  };

  const handleBackToLogin = () => {
    router.push('/login');
  };

  const handleTryAgain = () => {
    setState(VerificationState.VerifyCode);
    setCode('');
    setFormError(null);
  };

  const renderVerificationForm = () => (
    <Box component="form" onSubmit={handleSubmit} noValidate sx={{ mt: 3 }}>
      {isError && error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}
      
      {formError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {formError}
        </Alert>
      )}

      {resendSuccess && (
        <Alert severity="success" sx={{ mb: 3 }}>
          {t('verification.messages.resendSuccess')}
        </Alert>
      )}

      <TextField
        fullWidth
        label={t('verification.fields.code')}
        name="code"
        value={code}
        onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
        error={!!formError}
        helperText={formError || t('verification.helperText')}
        sx={{ mb: 3 }}
        inputProps={{ 
          maxLength: 6,
          pattern: '[0-9]{6}',
          inputMode: 'numeric',
        }}
        autoFocus
        disabled={isLoading}
      />

      <Button 
        fullWidth 
        type="submit" 
        variant="contained" 
        disabled={isLoading || code.length !== 6} 
        sx={{ mb: 2, py: 1.5 }}
      >
        {isLoading ? t('verification.actions.submitting') : t('verification.actions.submit')}
      </Button>

      <Stack direction="row" justifyContent="space-between" sx={{ mt: 3 }}>
        <Button 
          variant="text" 
          onClick={handleBackToLogin} 
          sx={{ color: 'text.secondary' }}
          disabled={isLoading}
        >
          {t('verification.actions.backToLogin')}
        </Button>

        <Button 
          variant="text" 
          onClick={handleResendEmail} 
          disabled={isLoading || resendLoading || resendCooldown > 0}
        >
          {resendLoading ? t('verification.actions.sending') : resendCooldown > 0 ? t('verification.actions.resendCooldown', { seconds: resendCooldown }) : t('verification.actions.resend')}
        </Button>
      </Stack>
    </Box>
  );

  const renderSuccess = () => (
    <Box sx={{ textAlign: 'center', my: 5 }}>
      <Alert severity="success" sx={{ mb: 3 }}>
        {t('verification.messages.success')}
      </Alert>

      <Typography variant="body1" sx={{ mb: 3 }}>
        {t('verification.messages.redirecting')}
      </Typography>

      <CircularProgress size={32} />
    </Box>
  );

  const renderError = () => (
    <Box sx={{ my: 5 }}>
      <Alert severity="error" sx={{ mb: 3 }}>
        {error || formError || t('verification.errors.failed')}
      </Alert>

      <Button
        fullWidth
        variant="contained"
        onClick={handleTryAgain}
        sx={{ mt: 3, py: 1.5 }}
      >
        {t('verification.actions.tryAgain')}
      </Button>
    </Box>
  );

  return (
    <Container maxWidth="sm">
      <Box sx={{ textAlign: 'center', mb: 5 }}>
        <Typography variant="h4" gutterBottom>
          {t('verification.title')}
        </Typography>

        <Typography variant="body1" color="text.secondary">
          {email 
            ? t('verification.subtitleWithEmail', { email }) 
            : t('verification.subtitle')}
        </Typography>
      </Box>

      {state === VerificationState.VerifyCode && renderVerificationForm()}
      {state === VerificationState.Success && renderSuccess()}
      {state === VerificationState.Error && renderError()}
    </Container>
  );
}
