'use client';

import { AccountGeneralForm } from './AccountGeneralForm';
import type { UserProfileModel } from '@telar/sdk';

interface AccountGeneralProps {
  profile: UserProfileModel;
}

export function AccountGeneral({ profile }: AccountGeneralProps) {
  return <AccountGeneralForm profile={profile} />;
}


