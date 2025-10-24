'use client';

import { Card, CardContent, Typography, Box } from '@mui/material';
import PhotoLibraryIcon from '@mui/icons-material/PhotoLibrary';

export function ProfileGallery() {
  return (
    <Card>
      <CardContent>
        <Box sx={{ textAlign: 'center', py: 4 }}>
          <PhotoLibraryIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            Gallery Feature Coming Soon
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            This will display photos and media shared by this user
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
}


