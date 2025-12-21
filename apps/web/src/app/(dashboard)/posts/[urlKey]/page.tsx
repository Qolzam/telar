'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { Container, CircularProgress, Alert, Box } from '@mui/material';
import { sdk } from '@/lib/sdk';
import type { Post } from '@telar/sdk';
import { PostCard } from '@/features/posts/components';

export default function SharedPostPage() {
  const params = useParams<{ urlKey: string }>();
  const urlKey = params?.urlKey;
  const [post, setPost] = useState<Post | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchPost = async () => {
      if (!urlKey) return;
      setLoading(true);
      setError(null);
      try {
        let fetchedPost: Post;
        // Try URL key first
        fetchedPost = await sdk.posts.getByUrlKey(urlKey);
        setPost(fetchedPost);
      } catch (err) {
        try {
          // Fallback: treat urlKey as objectId for legacy links
          const fallbackPost = await sdk.posts.getById(urlKey);
          setPost(fallbackPost);
        } catch (innerErr) {
          console.error('Failed to load post via share link:', innerErr);
          setError('Unable to load this post. It may be private or was removed.');
        }
      } finally {
        setLoading(false);
      }
    };

    fetchPost();
  }, [urlKey]);

  if (loading) {
    return (
      <Container maxWidth="md" sx={{ py: 6, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress />
      </Container>
    );
  }

  if (error || !post) {
    return (
      <Container maxWidth="md" sx={{ py: 6 }}>
        <Alert severity="error">{error || 'Post not found.'}</Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <Box>
        <PostCard post={post} />
      </Box>
    </Container>
  );
}

