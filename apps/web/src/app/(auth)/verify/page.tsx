'use client';

import { useEffect, useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Box, CircularProgress, Alert, Typography, Container } from '@mui/material';
import { Button } from '@mui/material';
import AuthLayout from '@/components/auth/AuthLayout';
import { useVerifyEmail } from '@/features/auth/client';
import { mapAuthError } from '@/features/auth/utils/errorMapper';

function VerifyContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { verifyAsync } = useVerifyEmail();
  
  const [status, setStatus] = useState<'verifying' | 'success' | 'error'>('verifying');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  
  const verificationId = searchParams.get('verificationId');
  const code = searchParams.get('code');

  useEffect(() => {
    const autoVerify = async () => {
      if (!verificationId || !code) {
        setStatus('error');
        setErrorMessage('Invalid verification link. Please check your email or try entering the code manually.');
        return;
      }

      try {
        console.log('[VerifyLink] Auto-verifying with link parameters');
        await verifyAsync({ verificationId, code });
        
        setStatus('success');
        setTimeout(() => {
          router.push('/dashboard');
        }, 2000);
      } catch (error: unknown) {
        console.error('[VerifyLink] Verification failed:', error);
        setStatus('error');
        const userMessage = mapAuthError(error, 'verify');
        setErrorMessage(userMessage);
      }
    };

    autoVerify();
  }, [verificationId, code, verifyAsync, router]);

  const renderVerifying = () => (
    <Box sx={{ textAlign: 'center', py: 8 }}>
      <CircularProgress size={60} sx={{ mb: 3 }} />
      <Typography variant="h5" gutterBottom>
        Verifying Your Email...
      </Typography>
      <Typography variant="body2" color="text.secondary">
        Please wait while we confirm your email address
      </Typography>
    </Box>
  );

  const renderSuccess = () => (
    <Box sx={{ textAlign: 'center', py: 8 }}>
      <Alert severity="success" sx={{ mb: 3 }}>
        Email verified successfully!
      </Alert>
      <Typography variant="h5" gutterBottom>
        Welcome to Telar!
      </Typography>
      <Typography variant="body1" sx={{ mb: 3 }}>
        Redirecting you to the dashboard...
      </Typography>
      <CircularProgress size={32} />
    </Box>
  );

  const renderError = () => (
    <Box sx={{ textAlign: 'center', py: 8 }}>
      <Alert severity="error" sx={{ mb: 3 }}>
        {errorMessage || 'Verification failed. Please try again.'}
      </Alert>
      <Typography variant="h5" gutterBottom>
        Verification Failed
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        You can try entering the code manually or request a new verification email.
      </Typography>
      <Button
        variant="contained"
        onClick={() => router.push('/signup')}
        sx={{ mt: 2, px: 4, py: 1.5 }}
      >
        Back to Signup
      </Button>
    </Box>
  );

  return (
    <Container maxWidth="sm">
      {status === 'verifying' && renderVerifying()}
      {status === 'success' && renderSuccess()}
      {status === 'error' && renderError()}
    </Container>
  );
}

export default function VerifyLinkPage() {
  return (
    <AuthLayout
      title="Email Verification"
      subtitle="Confirming your email address"
    >
      <Suspense fallback={<CircularProgress />}>
        <VerifyContent />
      </Suspense>
    </AuthLayout>
  );
}

