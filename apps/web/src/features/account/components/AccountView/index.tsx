'use client';

import { useState } from 'react';
import { Box, CircularProgress, Stack, Tab, Tabs, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { useProfile } from '@/features/profile/client';
import { AccountGeneral } from '../AccountGeneral';
import { AccountNotifications } from '../AccountNotifications';
import { AccountSocial } from '../AccountSocial';
import { AccountSecurity } from '../AccountSecurity';
import { AccountLanguage } from '../AccountLanguage';
import { AccountTheme } from '../AccountTheme';

export function AccountView() {
  const { t } = useTranslation('settings');
  const [currentTab, setCurrentTab] = useState('general');
  const { data: profile, isLoading, error } = useProfile();

  const TABS = [
    { value: 'general', label: t('tabs.general') },
    { value: 'notifications', label: t('tabs.notifications') },
    { value: 'social', label: t('tabs.social') },
    { value: 'security', label: t('tabs.security') },
    { value: 'language', label: t('tabs.language') },
    { value: 'theme', label: t('tabs.theme') },
  ];

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
      {currentTab === 'language' && <AccountLanguage />}
      {currentTab === 'theme' && <AccountTheme />}
    </Stack>
  );
}


