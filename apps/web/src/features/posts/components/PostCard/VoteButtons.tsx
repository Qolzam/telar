'use client';

import React from 'react';
import { Box, IconButton, Typography } from '@mui/material';
import { ArrowUpward, ArrowDownward } from '@mui/icons-material';
import type { Post } from '@telar/sdk';
import { useVoteMutation } from '@/features/votes/client';

interface VoteButtonsProps {
  post: Post;
}

/**
 * VoteButtons component for Up/Down voting on posts
 * 
 * State machine:
 * - voteType === 0: None (no vote)
 * - voteType === 1: Up (highlighted)
 * - voteType === 2: Down (highlighted)
 * 
 * Interaction logic:
 * - Clicking Up when state is Up -> Toggle Off (voteType: 0)
 * - Clicking Up when state is Down -> Switch to Up (voteType: 1)
 * - Clicking Up when state is None -> New Up Vote (voteType: 1)
 * - Clicking Down when state is Down -> Toggle Off (voteType: 0)
 * - Clicking Down when state is Up -> Switch to Down (voteType: 2)
 * - Clicking Down when state is None -> New Down Vote (voteType: 2)
 */
export function VoteButtons({ post }: VoteButtonsProps) {
  const voteMutation = useVoteMutation();
  const currentType = post.voteType ?? 0;
  const currentScore = post.score ?? 0;

  const handleUpVote = () => {
    // Send the action (1 = Up), not the resulting state
    // Backend will handle toggle logic: if current === clicked, toggle off
    voteMutation.mutate({ postId: post.objectId, voteType: 1 });
  };

  const handleDownVote = () => {
    // Send the action (2 = Down), not the resulting state
    // Backend will handle toggle logic: if current === clicked, toggle off
    voteMutation.mutate({ postId: post.objectId, voteType: 2 });
  };

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
      {/* Up Vote Button */}
      <IconButton
        onClick={handleUpVote}
        disabled={voteMutation.isPending}
        size="small"
        sx={{
          color: currentType === 1 ? '#4F46E5' : '#94A3B8',
          padding: '4px',
          '&:hover': {
            color: currentType === 1 ? '#4338CA' : '#1E293B',
            backgroundColor: 'transparent',
          },
          '&:disabled': {
            color: currentType === 1 ? '#4F46E5' : '#94A3B8',
            opacity: 0.6,
          },
        }}
        aria-label="upvote"
      >
        <ArrowUpward sx={{ fontSize: '20px' }} />
      </IconButton>

      {/* Score Display */}
      <Typography
        sx={{
          fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
          fontSize: '14px',
          fontWeight: 500,
          lineHeight: '20px',
          letterSpacing: '-0.084px',
          color: currentScore > 0 ? '#4F46E5' : currentScore < 0 ? '#EF4444' : '#1E293B',
          minWidth: '24px',
          textAlign: 'center',
        }}
      >
        {currentScore}
      </Typography>

      {/* Down Vote Button */}
      <IconButton
        onClick={handleDownVote}
        disabled={voteMutation.isPending}
        size="small"
        sx={{
          color: currentType === 2 ? '#EF4444' : '#94A3B8',
          padding: '4px',
          '&:hover': {
            color: currentType === 2 ? '#DC2626' : '#1E293B',
            backgroundColor: 'transparent',
          },
          '&:disabled': {
            color: currentType === 2 ? '#EF4444' : '#94A3B8',
            opacity: 0.6,
          },
        }}
        aria-label="downvote"
      >
        <ArrowDownward sx={{ fontSize: '20px' }} />
      </IconButton>
    </Box>
  );
}




