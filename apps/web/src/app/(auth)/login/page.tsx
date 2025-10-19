import { Metadata } from 'next';
import AuthLayout from '@/components/auth/AuthLayout';
import LoginForm from '@/features/auth/components/Login/LoginForm';

export const metadata: Metadata = {
  title: 'Sign In | Telar',
  description: 'Sign in to your Telar account',
};

export default function LoginPage() {
  return (
    <AuthLayout
      title="Welcome Back to Telar"
      subtitle="Connect with friends, share your moments, and discover new experiences in our vibrant community."
    >
      <LoginForm />
    </AuthLayout>
  );
}
