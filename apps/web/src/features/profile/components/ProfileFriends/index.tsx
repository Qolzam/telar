'use client';

import { Card, CardContent, Typography, Box } from '@mui/material';
import GroupIcon from '@mui/icons-material/Group';

export function ProfileFriends() {
  return (
    <Card>
      <CardContent>
        <Box sx={{ textAlign: 'center', py: 4 }}>
          <GroupIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            Friends Feature Coming Soon
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            This will display mutual connections and friends
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
}


