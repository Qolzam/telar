'use client';

import { useTranslation } from 'react-i18next';
import { Card, CardContent, Typography, Box } from '@mui/material';
import PeopleIcon from '@mui/icons-material/People';

export function ProfileFollowers() {
  const { t } = useTranslation('profile');
  
  return (
    <Card>
      <CardContent>
        <Box sx={{ textAlign: 'center', py: 4 }}>
          <PeopleIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            {t('followers.comingSoon')}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            {t('followers.description')}
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
}


