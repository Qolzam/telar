'use client';

import { useTranslation } from 'react-i18next';
import { Tab, Tabs } from '@mui/material';

interface ProfileTabsProps {
  value: string;
  onChange: (event: React.SyntheticEvent, newValue: string) => void;
}

export function ProfileTabs({ value, onChange }: ProfileTabsProps) {
  const { t } = useTranslation('profile');
  
  const TABS = [
    { value: 'profile', label: t('tabs.home') },
    { value: 'followers', label: t('tabs.followers') },
    { value: 'friends', label: t('tabs.friends') },
    { value: 'gallery', label: t('tabs.gallery') },
  ];

  return (
    <Tabs value={value} onChange={onChange}>
      {TABS.map((tab) => (
        <Tab key={tab.value} value={tab.value} label={tab.label} />
      ))}
    </Tabs>
  );
}


