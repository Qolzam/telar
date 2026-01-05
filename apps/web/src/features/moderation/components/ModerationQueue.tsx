'use client';

import { Box, Typography, CircularProgress, Alert, Button, Stack } from '@mui/material';
import { Refresh } from '@mui/icons-material';
import { useModerationQueue, useApprovePost, useRejectPost, FlaggedPost } from '../client';
import { ModerationCard } from './ModerationCard';
import { useState } from 'react';

export function ModerationQueue() {
  const { data, isLoading, isError, error, refetch } = useModerationQueue();
  const approveMutation = useApprovePost();
  const rejectMutation = useRejectPost();
  const [processingIds, setProcessingIds] = useState<Set<string>>(new Set());

  const handleApprove = async (postId: string) => {
    setProcessingIds(prev => new Set(prev).add(postId));
    try {
      await approveMutation.mutateAsync(postId);
    } finally {
      setProcessingIds(prev => {
        const next = new Set(prev);
        next.delete(postId);
        return next;
      });
    }
  };

  const handleReject = async (postId: string) => {
    setProcessingIds(prev => new Set(prev).add(postId));
    try {
      await rejectMutation.mutateAsync(postId);
    } finally {
      setProcessingIds(prev => {
        const next = new Set(prev);
        next.delete(postId);
        return next;
      });
    }
  };

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (isError) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        Failed to load moderation queue: {error instanceof Error ? error.message : 'Unknown error'}
        <Button
          size="small"
          onClick={() => refetch()}
          sx={{ ml: 2 }}
        >
          Retry
        </Button>
      </Alert>
    );
  }

  const posts = data?.items || [];
  const count = data?.count || 0;

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" component="h1">
          Moderation Queue
        </Typography>
        <Button
          variant="outlined"
          startIcon={<Refresh />}
          onClick={() => refetch()}
          disabled={isLoading}
        >
          Refresh
        </Button>
      </Stack>

      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        {count} {count === 1 ? 'post' : 'posts'} waiting for review
      </Typography>

      {posts.length === 0 ? (
        <Alert severity="info">
          No posts in the moderation queue. All clear! 🎉
        </Alert>
      ) : (
        <Box>
          {posts.map((post: FlaggedPost) => (
            <ModerationCard
              key={post.id}
              post={post}
              onApprove={handleApprove}
              onReject={handleReject}
              isProcessing={processingIds.has(post.id)}
            />
          ))}
        </Box>
      )}
    </Box>
  );
}





