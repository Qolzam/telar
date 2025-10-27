'use client';

import { useTranslation } from 'react-i18next';
import { Card, CardContent, Typography, Box } from '@mui/material';
import PhotoLibraryIcon from '@mui/icons-material/PhotoLibrary';

export function ProfileGallery() {
  const { t } = useTranslation('profile');
  
  return (
    <Card>
      <CardContent>
        <Box sx={{ textAlign: 'center', py: 4 }}>
          <PhotoLibraryIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            {t('gallery.comingSoon')}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            {t('gallery.description')}
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
}


