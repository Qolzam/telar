'use client';

import { useTranslation } from 'react-i18next';
import { Box, Card, Divider, Stack } from '@mui/material';

interface ProfileFollowsProps {
  followerCount: number;
  followCount: number;
}

export function ProfileFollows({ followerCount, followCount }: ProfileFollowsProps) {
  const { t } = useTranslation('profile');
  
  return (
    <Card sx={{ py: 3, textAlign: 'center', typography: 'h4' }}>
      <Stack
        direction="row"
        divider={<Divider orientation="vertical" flexItem sx={{ borderStyle: 'dashed' }} />}
      >
        <Stack width={1}>
          {followerCount.toLocaleString()}
          <Box component="span" sx={{ color: 'text.secondary', typography: 'body2' }}>
            {t('follows.follower', { count: followerCount })}
          </Box>
        </Stack>

        <Stack width={1}>
          {followCount.toLocaleString()}
          <Box component="span" sx={{ color: 'text.secondary', typography: 'body2' }}>
            {t('follows.following')}
          </Box>
        </Stack>
      </Stack>
    </Card>
  );
}


