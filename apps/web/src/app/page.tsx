'use client';

import { useTranslation } from 'react-i18next';
import { Button, Typography, Box, Stack } from '@mui/material';
import { Home as HomeIcon } from '@mui/icons-material';

export default function HomePage() {
  const { t } = useTranslation('common');
  
  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: 'background.default',
      }}
    >
      <Stack spacing={3} alignItems="center">
        <HomeIcon color="primary" sx={{ fontSize: 60 }} />
        <Typography variant="h3" component="h1" gutterBottom>
          {t('homepage.title')}
        </Typography>
        <Typography variant="body1" color="text.secondary" textAlign="center" maxWidth={600}>
          {t('homepage.subtitle')}
        </Typography>
        <Stack direction="row" spacing={2}>
          <Button variant="contained" size="large">
            {t('homepage.getStarted')}
          </Button>
          <Button variant="outlined" size="large">
            {t('homepage.learnMore')}
          </Button>
        </Stack>
      </Stack>
    </Box>
  );
}
