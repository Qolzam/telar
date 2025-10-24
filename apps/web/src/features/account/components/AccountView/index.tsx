'use client';

import { useState } from 'react';
import { Box, CircularProgress, Stack, Tab, Tabs, Typography } from '@mui/material';
import { useProfile } from '@/features/profile/client';
import { AccountGeneral } from '../AccountGeneral';
import { AccountNotifications } from '../AccountNotifications';
import { AccountSocial } from '../AccountSocial';
import { AccountSecurity } from '../AccountSecurity';

const TABS = [
  { value: 'general', label: 'General' },
  { value: 'notifications', label: 'Notifications' },
  { value: 'social', label: 'Social Links' },
  { value: 'security', label: 'Security' },
];

export function AccountView() {
  const [currentTab, setCurrentTab] = useState('general');
  const { data: profile, isLoading, error } = useProfile();

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !profile) {
    return (
      <Box sx={{ textAlign: 'center', py: 8 }}>
        <Typography variant="h6" color="text.secondary">
          Failed to load profile
        </Typography>
      </Box>
    );
  }

  return (
    <Stack spacing={3}>
      <Tabs value={currentTab} onChange={(_, newValue) => setCurrentTab(newValue)}>
        {TABS.map((tab) => (
          <Tab key={tab.value} label={tab.label} value={tab.value} />
        ))}
      </Tabs>

      {currentTab === 'general' && <AccountGeneral profile={profile} />}
      {currentTab === 'notifications' && <AccountNotifications />}
      {currentTab === 'social' && <AccountSocial profile={profile} />}
      {currentTab === 'security' && <AccountSecurity />}
    </Stack>
  );
}


