'use client';

import { useTranslation } from 'react-i18next';
import { Card, CardContent, Typography, Box } from '@mui/material';
import GroupIcon from '@mui/icons-material/Group';

export function ProfileFriends() {
  const { t } = useTranslation('profile');
  
  return (
    <Card>
      <CardContent>
        <Box sx={{ textAlign: 'center', py: 4 }}>
          <GroupIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            {t('friends.comingSoon')}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            {t('friends.description')}
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
}


