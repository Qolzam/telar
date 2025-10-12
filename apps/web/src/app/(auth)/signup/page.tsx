import { Metadata } from 'next';
import { Box } from '@mui/material';
import AuthLayout from '@/components/auth/AuthLayout';
import SignupContainer from '@/features/auth/components/Signup/SignupContainer';

export const metadata: Metadata = {
  title: 'Sign Up | Telar',
  description: 'Create your Telar account',
};

export default function SignupPage() {
  const signupIllustration = (
    <Box
      sx={{
        width: 300,
        height: 200,
        mx: 'auto',
        backgroundColor: 'rgba(255, 255, 255, 0.1)',
        borderRadius: 3,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        border: '2px dashed rgba(255, 255, 255, 0.3)',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      {/* Simple illustration elements for "Join Community" */}
      <Box
        sx={{
          position: 'absolute',
          top: '15%',
          left: '15%',
          width: 35,
          height: 35,
          borderRadius: '50%',
          backgroundColor: 'rgba(255, 255, 255, 0.4)',
        }}
      />
      <Box
        sx={{
          position: 'absolute',
          top: '25%',
          right: '20%',
          width: 25,
          height: 25,
          borderRadius: '50%',
          backgroundColor: 'rgba(255, 255, 255, 0.3)',
        }}
      />
      <Box
        sx={{
          position: 'absolute',
          bottom: '20%',
          left: '25%',
          width: 30,
          height: 30,
          borderRadius: '50%',
          backgroundColor: 'rgba(255, 255, 255, 0.35)',
        }}
      />
      {/* Central connection element */}
      <Box
        sx={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: 80,
          height: 80,
          borderRadius: '50%',
          border: '4px solid rgba(255, 255, 255, 0.5)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: 'rgba(255, 255, 255, 0.1)',
        }}
      >
        <Box
          sx={{
            fontSize: '2rem',
          }}
        >
          🚀
        </Box>
      </Box>
      {/* Connection lines */}
      <Box
        sx={{
          position: 'absolute',
          top: '30%',
          left: '30%',
          width: '40%',
          height: '2px',
          backgroundColor: 'rgba(255, 255, 255, 0.2)',
          transform: 'rotate(45deg)',
        }}
      />
    </Box>
  );

  return (
    <AuthLayout
      title="Welcome to Telar Social"
      subtitle="Connect with friends, share your moments, and discover new experiences in our vibrant community."
      illustration={signupIllustration}
    >
      <SignupContainer />
    </AuthLayout>
  );
}
