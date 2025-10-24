import { Metadata } from 'next';
import { AccountView } from '@/features/account/components/AccountView';

export const metadata: Metadata = {
  title: 'Settings | Telar Social',
  description: 'Manage your account settings and preferences',
};

export default function SettingsPage() {
  return <AccountView />;
}

