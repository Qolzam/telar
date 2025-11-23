'use client';

import { ReactNode } from 'react';
import { Container, Typography } from '@mui/material';
import { useSession } from '@/features/auth/client';
import { useRouter } from 'next/navigation';

export default function AdminLayout({ children }: { children: ReactNode }) {
  const { user, isAuthenticated } = useSession();
  const router = useRouter();

  if (!isAuthenticated) {
    // Redirect to login preserving path
    if (typeof window !== 'undefined') {
      const from = encodeURIComponent(window.location.pathname);
      router.push(`/login?from=${from}`);
    }
    return null;
  }

  if (user?.role !== 'admin') {
    return (
      <Container maxWidth="md" sx={{ py: 6 }}>
        <Typography variant="h6">Access denied</Typography>
        <Typography variant="body2" color="text.secondary">
          You don&apos;t have permission to access the admin dashboard.
        </Typography>
      </Container>
    );
  }

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <Typography variant="h4" sx={{ mb: 3 }}>
        Admin Dashboard
      </Typography>
      {children}
    </Container>
  );
}






