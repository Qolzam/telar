'use client';

import React from 'react';
import { Box, IconButton, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
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
  const theme = useTheme();
  const primary = `var(--mui-palette-primary-main, ${theme.palette.primary.main})`;
  const error = `var(--mui-palette-error-main, ${theme.palette.error.main})`;
  const textPrimary = `var(--mui-palette-text-primary, ${theme.palette.text.primary})`;
  const textSecondary = `var(--mui-palette-text-secondary, ${theme.palette.text.secondary})`;
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
          color: currentType === 1 ? primary : textSecondary,
          padding: '4px',
          '&:hover': {
            color: currentType === 1 ? primary : textPrimary,
            backgroundColor: 'transparent',
          },
          '&:disabled': {
            color: currentType === 1 ? primary : textSecondary,
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
          color: currentScore > 0 ? primary : currentScore < 0 ? error : textPrimary,
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
          color: currentType === 2 ? error : textSecondary,
          padding: '4px',
          '&:hover': {
            color: currentType === 2 ? error : textPrimary,
            backgroundColor: 'transparent',
          },
          '&:disabled': {
            color: currentType === 2 ? error : textSecondary,
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




