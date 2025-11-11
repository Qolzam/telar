'use client';

import { Container, Typography } from '@mui/material';
import { PostForm, PostList } from '@/features/posts/components';

export default function DashboardHomePage() {
  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <Typography variant="h4" sx={{ mb: 3 }}>
        Feed
      </Typography>
      <PostForm />
      <PostList />
    </Container>
  );
}

