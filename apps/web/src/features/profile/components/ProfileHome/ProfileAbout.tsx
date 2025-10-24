'use client';

import { Box, Card, CardHeader, Link, Stack } from '@mui/material';
import EmailIcon from '@mui/icons-material/Email';
import BusinessIcon from '@mui/icons-material/Business';
import LanguageIcon from '@mui/icons-material/Language';
import type { UserProfileModel } from '@telar/sdk';

interface ProfileAboutProps {
  profile: UserProfileModel;
}

export function ProfileAbout({ profile }: ProfileAboutProps) {
  return (
    <Card>
      <CardHeader title="About" />

      <Stack spacing={2} sx={{ p: 3 }}>
        {profile.tagLine && (
          <Box sx={{ typography: 'body2', color: 'text.secondary' }}>
            {profile.tagLine}
          </Box>
        )}

        {profile.email && (
          <Stack direction="row" spacing={2} sx={{ typography: 'body2' }}>
            <EmailIcon sx={{ color: 'text.secondary' }} />
            <Box>{profile.email}</Box>
          </Stack>
        )}

        {profile.companyName && (
          <Stack direction="row" spacing={2}>
            <BusinessIcon sx={{ color: 'text.secondary' }} />
            <Box sx={{ typography: 'body2' }}>
              Works at{' '}
              <Link variant="subtitle2" color="inherit">
                {profile.companyName}
              </Link>
            </Box>
          </Stack>
        )}

        {profile.webUrl && (
          <Stack direction="row" spacing={2}>
            <LanguageIcon sx={{ color: 'text.secondary' }} />
            <Link 
              href={profile.webUrl} 
              target="_blank" 
              rel="noopener noreferrer"
              variant="body2"
              color="primary"
            >
              {profile.webUrl}
            </Link>
          </Stack>
        )}
      </Stack>
    </Card>
  );
}


