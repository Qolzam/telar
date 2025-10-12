'use client';

import { useState, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { 
  Box, 
  Button, 
  Typography, 
  Container, 
  Stack, 
  TextField, 
  CircularProgress, 
  Alert 
} from '@mui/material';
import { useVerifyEmail } from '@/lib/auth/hooks';

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
  const router = useRouter();
  const { verify, isLoading, error, isError } = useVerifyEmail();
  
  const [state, setState] = useState<VerificationState>(VerificationState.VerifyCode);
  const [code, setCode] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [resendLoading, setResendLoading] = useState(false);
  const [resendSuccess, setResendSuccess] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);
    
    if (!code) {
      setFormError('Verification code is required');
      return;
    }
    
    if (code.length < 6) {
      setFormError('Verification code must be 6 digits');
      return;
    }

    try {
      await verify({ verificationId, code });
      setState(VerificationState.Success);
      
      setTimeout(() => {
        router.push('/dashboard');
      }, 3000);
    } catch {
      setState(VerificationState.Error);
    }
  };

  const handleResendEmail = async () => {
    setResendLoading(true);
    setResendSuccess(false);
    
    try {
      // TODO: Implement resend verification API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      setResendSuccess(true);
      setTimeout(() => setResendSuccess(false), 5000);
    } catch {
      setFormError('Failed to resend verification email');
    } finally {
      setResendLoading(false);
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
          Verification email resent! Check your inbox.
        </Alert>
      )}

      <TextField
        fullWidth
        label="Verification Code"
        name="code"
        value={code}
        onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
        error={!!formError}
        helperText={formError || 'Enter the 6-digit code from your email'}
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
        {isLoading ? (
          <CircularProgress size={24} color="inherit" />
        ) : (
          'Verify Email'
        )}
      </Button>

      <Stack direction="row" justifyContent="space-between" sx={{ mt: 3 }}>
        <Button 
          variant="text" 
          onClick={handleBackToLogin} 
          sx={{ color: 'text.secondary' }}
          disabled={isLoading}
        >
          Back to Login
        </Button>

        <Button 
          variant="text" 
          onClick={handleResendEmail} 
          disabled={resendLoading || isLoading}
        >
          {resendLoading ? (
            <CircularProgress size={20} color="inherit" />
          ) : (
            'Resend Code'
          )}
        </Button>
      </Stack>
    </Box>
  );

  const renderSuccess = () => (
    <Box sx={{ textAlign: 'center', my: 5 }}>
      <Alert severity="success" sx={{ mb: 3 }}>
        Email verified successfully!
      </Alert>

      <Typography variant="body1" sx={{ mb: 3 }}>
        Redirecting you to the dashboard...
      </Typography>

      <CircularProgress size={32} />
    </Box>
  );

  const renderError = () => (
    <Box sx={{ my: 5 }}>
      <Alert severity="error" sx={{ mb: 3 }}>
        {error || 'Verification failed. Please try again.'}
      </Alert>

      <Button
        fullWidth
        variant="contained"
        onClick={handleTryAgain}
        sx={{ mt: 3, py: 1.5 }}
      >
        Try Again
      </Button>
    </Box>
  );

  return (
    <Container maxWidth="sm">
      <Box sx={{ textAlign: 'center', mb: 5 }}>
        <Typography variant="h4" gutterBottom>
          Verify Your Email
        </Typography>

        <Typography variant="body1" color="text.secondary">
          {email 
            ? `We've sent a verification code to ${email}` 
            : 'Enter the verification code sent to your email'}
        </Typography>
      </Box>

      {state === VerificationState.VerifyCode && renderVerificationForm()}
      {state === VerificationState.Success && renderSuccess()}
      {state === VerificationState.Error && renderError()}
    </Container>
  );
}
