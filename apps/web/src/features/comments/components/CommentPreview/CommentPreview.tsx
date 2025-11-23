'use client';

import { Avatar, Box, Typography } from '@mui/material';
import type { Comment } from '@telar/sdk';

interface CommentPreviewProps {
  comments?: Comment[];
  count: number;
}

export function CommentPreview({ comments, count }: CommentPreviewProps) {
  if (!count || count <= 0) {
    return (
      <Typography variant="caption" color="text.secondary">
        Be the first to comment
      </Typography>
    );
  }

  const latest = comments && comments.length > 0 ? comments[0] : null;

  if (!latest) {
    return (
      <Typography variant="caption" color="text.secondary">
        {count} {count === 1 ? 'Comment' : 'Comments'}
      </Typography>
    );
  }

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <Avatar
        src={latest.ownerAvatar}
        alt={latest.ownerDisplayName}
        sx={{ width: 20, height: 20 }}
      >
        {latest.ownerDisplayName?.[0]?.toUpperCase()}
      </Avatar>
      <Typography variant="caption">
        <Box component="span" sx={{ fontWeight: 600 }}>
          {latest.ownerDisplayName}:
        </Box>{' '}
        <Box component="span" sx={{ color: 'text.secondary' }}>
          {latest.text}
        </Box>
        {count > 1 && (
          <Box component="span" sx={{ color: 'text.secondary', ml: 0.5 }}>
            • {count - 1} more
          </Box>
        )}
      </Typography>
    </Box>
  );
}






