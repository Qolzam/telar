'use client';

import { SocialLinksForm } from './SocialLinksForm';
import type { UserProfileModel } from '@telar/sdk';

interface AccountSocialProps {
  profile: UserProfileModel;
}

export function AccountSocial({ profile }: AccountSocialProps) {
  return <SocialLinksForm profile={profile} />;
}


