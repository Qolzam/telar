'use client';

import { Tab, Tabs } from '@mui/material';

interface ProfileTabsProps {
  value: string;
  onChange: (event: React.SyntheticEvent, newValue: string) => void;
}

const TABS = [
  { value: 'profile', label: 'Profile' },
  { value: 'followers', label: 'Followers' },
  { value: 'friends', label: 'Friends' },
  { value: 'gallery', label: 'Gallery' },
];

export function ProfileTabs({ value, onChange }: ProfileTabsProps) {
  return (
    <Tabs value={value} onChange={onChange}>
      {TABS.map((tab) => (
        <Tab key={tab.value} value={tab.value} label={tab.label} />
      ))}
    </Tabs>
  );
}


