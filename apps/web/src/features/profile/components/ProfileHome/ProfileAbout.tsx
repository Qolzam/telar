'use client';

import { useTranslation } from 'react-i18next';
import { Box, Card, CardHeader, Link, Stack } from '@mui/material';
import BusinessIcon from '@mui/icons-material/Business';
import LanguageIcon from '@mui/icons-material/Language';
import type { UserProfileModel } from '@telar/sdk';

interface ProfileAboutProps {
  profile: UserProfileModel;
}

export function ProfileAbout({ profile }: ProfileAboutProps) {
  const { t } = useTranslation('profile');
  
  return (
    <Card>
      <CardHeader title={t('about.title')} />

      <Stack spacing={2} sx={{ p: 3 }}>
        <Box sx={{ typography: 'body2', color: 'text.secondary' }}>
          {profile.tagLine || t('about.defaultTagline')}
        </Box>

        {profile.companyName && (
          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            spacing={2}
            alignItems={{ xs: 'flex-start', sm: 'center' }}
          >
            <BusinessIcon sx={{ color: 'text.secondary' }} />
            <Box sx={{ typography: 'body2', overflowWrap: 'anywhere' }}>
              {t('about.worksAt')}{' '}
              <Link variant="subtitle2" color="inherit">
                {profile.companyName}
              </Link>
            </Box>
          </Stack>
        )}

        {profile.webUrl && (
          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            spacing={2}
            alignItems={{ xs: 'flex-start', sm: 'center' }}
          >
            <LanguageIcon sx={{ color: 'text.secondary' }} />
            <Link
              href={profile.webUrl} 
              target="_blank" 
              rel="noopener noreferrer"
              variant="body2"
              color="primary"
              sx={{ overflowWrap: 'anywhere' }}
            >
              {profile.webUrl}
            </Link>
          </Stack>
        )}
      </Stack>
    </Card>
  );
}


