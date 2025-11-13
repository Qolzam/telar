'use client';

import { useTranslation } from 'react-i18next';
import { Card, CardContent, Typography, Box } from '@mui/material';
import NotificationsIcon from '@mui/icons-material/Notifications';

export function AccountNotifications() {
  const { t } = useTranslation('settings');
  
  return (
    <Card>
      <CardContent>
        <Box sx={{ textAlign: 'center', py: 4 }}>
          <NotificationsIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            {t('notifications.title')}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            {t('notifications.description')}
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
}


