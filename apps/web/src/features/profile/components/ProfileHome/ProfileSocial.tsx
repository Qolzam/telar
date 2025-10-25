'use client';

import { Card, CardHeader, Link, Stack } from '@mui/material';
import FacebookIcon from '@mui/icons-material/Facebook';
import TwitterIcon from '@mui/icons-material/Twitter';
import InstagramIcon from '@mui/icons-material/Instagram';
import type { UserProfileModel } from '@telar/sdk';

interface ProfileSocialProps {
  profile: UserProfileModel;
}

const socialLinks = [
  {
    key: 'facebookId',
    icon: FacebookIcon,
    label: 'Facebook',
    baseUrl: 'https://facebook.com/',
  },
  {
    key: 'twitterId',
    icon: TwitterIcon,
    label: 'Twitter',
    baseUrl: 'https://twitter.com/',
  },
  {
    key: 'instagramId',
    icon: InstagramIcon,
    label: 'Instagram',
    baseUrl: 'https://instagram.com/',
  },
] as const;

export function ProfileSocial({ profile }: ProfileSocialProps) {
  const hasAnySocial = profile.facebookId || profile.twitterId || profile.instagramId;

  if (!hasAnySocial) {
    return null;
  }

  return (
    <Card>
      <CardHeader title="Social" />

      <Stack spacing={2} sx={{ p: 3 }}>
        {socialLinks.map(({ key, icon: Icon, label, baseUrl }) => {
          const value = profile[key];
          if (!value) return null;

          return (
            <Stack
              key={key}
              spacing={2}
              direction="row"
              alignItems="center"
              sx={{ typography: 'body2' }}
            >
              <Icon sx={{ color: 'text.secondary' }} />
              <Link 
                href={`${baseUrl}${value}`}
                target="_blank"
                rel="noopener noreferrer"
                color="inherit"
                sx={{ wordBreak: 'break-all' }}
              >
                {value}
              </Link>
            </Stack>
          );
        })}
      </Stack>
    </Card>
  );
}


