import { Metadata } from 'next';
import { ProfileView } from '@/features/profile/components/ProfileView';

export const metadata: Metadata = {
  title: 'User Profile | Telar Social',
  description: 'View user profile on Telar Social',
};

interface UserProfilePageProps {
  params: Promise<{ userId: string }>;
}

export default async function UserProfilePage({ params }: UserProfilePageProps) {
  const { userId } = await params;
  return <ProfileView userId={userId} />;
}


