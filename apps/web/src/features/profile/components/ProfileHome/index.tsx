'use client';

import { Grid, Stack } from '@mui/material';
import type { UserProfileModel } from '@telar/sdk';
import { ProfileFollows } from './ProfileFollows';
import { ProfileAbout } from './ProfileAbout';
import { ProfileSocial } from './ProfileSocial';

interface ProfileHomeProps {
  profile: UserProfileModel;
}

export function ProfileHome({ profile }: ProfileHomeProps) {
  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 4 }}>
        <Stack spacing={3}>
          <ProfileFollows 
            followerCount={profile.followerCount || 0}
            followCount={profile.followCount || 0}
          />

          <ProfileAbout profile={profile} />

          <ProfileSocial profile={profile} />
        </Stack>
      </Grid>

      <Grid size={{ xs: 12, md: 8 }}>
        <Stack spacing={3}>
          {/* Post input and feed will be added when posts module is implemented */}
        </Stack>
      </Grid>
    </Grid>
  );
}


