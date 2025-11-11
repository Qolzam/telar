/**
 * Dashboard Home Page
 * 
 * Main authenticated landing page - shows the feed
 */

'use client';

import { Container } from '@mui/material';
import { PostForm, PostList } from '@/features/posts/components';

export default function DashboardPage() {
  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <PostForm />
      <PostList />
    </Container>
  );
}
