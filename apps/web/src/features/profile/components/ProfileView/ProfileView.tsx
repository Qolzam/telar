'use client';

import { useState } from 'react';
import { Box, Card, CircularProgress, Stack, Typography } from '@mui/material';
import { useProfile } from '../../client';
import { ProfileCover } from './ProfileCover';
import { ProfileTabs } from './ProfileTabs';
import { ProfileHome } from '../ProfileHome';
import { ProfileFollowers } from '../ProfileFollowers';
import { ProfileFriends } from '../ProfileFriends';
import { ProfileGallery } from '../ProfileGallery';

interface ProfileViewProps {
  userId?: string;
}

export function ProfileView({ userId }: ProfileViewProps) {
  const [currentTab, setCurrentTab] = useState('profile');
  const { data: profile, isLoading, error } = useProfile(userId);

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !profile) {
    return (
      <Card>
        <Box sx={{ textAlign: 'center', py: 8 }}>
          <Typography variant="h6" color="text.secondary">
            Profile not found
          </Typography>
        </Box>
      </Card>
    );
  }

  return (
    <Stack spacing={3}>
      <Card sx={{ height: 290, position: 'relative' }}>
        <ProfileCover
          name={profile.fullName}
          avatarUrl={profile.avatar}
          role={profile.tagLine}
          coverUrl={profile.banner}
        />
        <Box
          sx={{
            position: 'absolute',
            bottom: 0,
            width: '100%',
            bgcolor: 'background.paper',
            zIndex: 9,
            display: 'flex',
            justifyContent: { xs: 'center', md: 'flex-end' },
            px: { md: 3 },
          }}
        >
          <ProfileTabs
            value={currentTab}
            onChange={(_, newValue) => setCurrentTab(newValue)}
          />
        </Box>
      </Card>

      {currentTab === 'profile' && <ProfileHome profile={profile} />}
      {currentTab === 'followers' && <ProfileFollowers />}
      {currentTab === 'friends' && <ProfileFriends />}
      {currentTab === 'gallery' && <ProfileGallery />}
    </Stack>
  );
}


