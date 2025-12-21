'use client';

import { Alert, Box, CircularProgress, Container, Typography } from '@mui/material';
import { PostList } from '@/features/posts/components';
import { useSession } from '@/features/auth/client';

export default function MyPostsPage() {
  const { user, isAuthenticated, isLoading } = useSession();

  if (isLoading) {
    return (
      <Container maxWidth="md" sx={{ py: 4, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress />
      </Container>
    );
  }

  if (!isAuthenticated || !user?.id) {
    return (
      <Container maxWidth="md" sx={{ py: 4 }}>
        <Alert severity="info">Please sign in to view your posts.</Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4">My Posts</Typography>
      </Box>
      <PostList params={{ owner: user.id, limit: 10 }} />
    </Container>
  );
}

