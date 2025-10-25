import { Metadata } from 'next';
import { ProfileView } from '@/features/profile/components/ProfileView';

export const metadata: Metadata = {
  title: 'Profile | Telar Social',
  description: 'View your profile on Telar Social',
};

export default function ProfilePage() {
  return <ProfileView />;
}


