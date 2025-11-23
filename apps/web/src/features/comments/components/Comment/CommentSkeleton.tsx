'use client';

import { Box, Skeleton } from '@mui/material';

export function CommentSkeleton() {
  return (
    <Box sx={{ display: 'flex', py: 2, borderBottom: (theme) => `1px solid ${theme.palette.divider}` }}>
      <Skeleton variant="circular" width={40} height={40} sx={{ mr: 2 }} />
      <Box sx={{ flexGrow: 1 }}>
        <Skeleton variant="text" width="30%" />
        <Skeleton variant="text" width="20%" />
        <Skeleton variant="text" width="90%" />
        <Skeleton variant="text" width="80%" />
      </Box>
    </Box>
  );
}






