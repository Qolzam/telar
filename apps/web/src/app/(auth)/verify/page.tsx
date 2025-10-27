'use client';

import { useEffect, useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTranslation } from 'react-i18next';
import { Box, CircularProgress, Alert, Typography, Container } from '@mui/material';
import { Button } from '@mui/material';
import AuthLayout from '@/components/auth/AuthLayout';
import { useVerifyEmail } from '@/features/auth/client';
import { mapAuthError } from '@/features/auth/utils/errorMapper';

function VerifyContent() {
  const { t } = useTranslation('auth');
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
        setErrorMessage(t('verify.errors.invalidLink'));
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
  }, [verificationId, code, verifyAsync, router, t]);

  const renderVerifying = () => (
    <Box sx={{ textAlign: 'center', py: 8 }}>
      <CircularProgress size={60} sx={{ mb: 3 }} />
      <Typography variant="h5" gutterBottom>
        {t('verify.verifying')}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {t('verify.pleaseWait')}
      </Typography>
    </Box>
  );

  const renderSuccess = () => (
    <Box sx={{ textAlign: 'center', py: 8 }}>
      <Alert severity="success" sx={{ mb: 3 }}>
        {t('verify.success')}
      </Alert>
      <Typography variant="h5" gutterBottom>
        {t('verify.welcome')}
      </Typography>
      <Typography variant="body1" sx={{ mb: 3 }}>
        {t('verify.redirecting')}
      </Typography>
      <CircularProgress size={32} />
    </Box>
  );

  const renderError = () => (
    <Box sx={{ textAlign: 'center', py: 8 }}>
      <Alert severity="error" sx={{ mb: 3 }}>
        {errorMessage || t('verify.errors.generic')}
      </Alert>
      <Typography variant="h5" gutterBottom>
        {t('verify.failed')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        {t('verify.tryManually')}
      </Typography>
      <Button
        variant="contained"
        onClick={() => router.push('/signup')}
        sx={{ mt: 2, px: 4, py: 1.5 }}
      >
        {t('verify.backToSignup')}
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

function VerifyLinkPageContent() {
  const { t } = useTranslation('auth');
  
  return (
    <AuthLayout
      title={t('verify.title')}
      subtitle={t('verify.subtitle')}
    >
      <Suspense fallback={<CircularProgress />}>
        <VerifyContent />
      </Suspense>
    </AuthLayout>
  );
}

export default function VerifyLinkPage() {
  return <VerifyLinkPageContent />;
}

