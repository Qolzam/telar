'use client';

import { Card, CardContent, Typography, Box } from '@mui/material';
import NotificationsIcon from '@mui/icons-material/Notifications';

export function AccountNotifications() {
  return (
    <Card>
      <CardContent>
        <Box sx={{ textAlign: 'center', py: 4 }}>
          <NotificationsIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            Notification Settings Coming Soon
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            Manage your email and push notification preferences
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
}


