'use client';

import { useState } from 'react';
import SignupForm from './SignupForm';
import VerificationCodeInput from '../EmailVerification/VerificationCodeInput';

export default function SignupContainer() {
  const [verificationId, setVerificationId] = useState<string | null>(null);
  const [email, setEmail] = useState<string>('');

  const handleSignupSuccess = (vid: string, userEmail: string) => {
    setVerificationId(vid);
    setEmail(userEmail);
  };

  if (verificationId) {
    return <VerificationCodeInput verificationId={verificationId} email={email} />;
  }

  return <SignupForm onSuccess={handleSignupSuccess} />;
}

