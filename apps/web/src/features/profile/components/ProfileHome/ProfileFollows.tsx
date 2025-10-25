'use client';

import { Box, Card, Divider, Stack } from '@mui/material';

interface ProfileFollowsProps {
  followerCount: number;
  followCount: number;
}

export function ProfileFollows({ followerCount, followCount }: ProfileFollowsProps) {
  return (
    <Card sx={{ py: 3, textAlign: 'center', typography: 'h4' }}>
      <Stack
        direction="row"
        divider={<Divider orientation="vertical" flexItem sx={{ borderStyle: 'dashed' }} />}
      >
        <Stack width={1}>
          {followerCount.toLocaleString()}
          <Box component="span" sx={{ color: 'text.secondary', typography: 'body2' }}>
            Follower{followerCount !== 1 ? 's' : ''}
          </Box>
        </Stack>

        <Stack width={1}>
          {followCount.toLocaleString()}
          <Box component="span" sx={{ color: 'text.secondary', typography: 'body2' }}>
            Following
          </Box>
        </Stack>
      </Stack>
    </Card>
  );
}


