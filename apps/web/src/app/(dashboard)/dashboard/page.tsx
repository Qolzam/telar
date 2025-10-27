/**
 * Dashboard Home Page
 * 
 * Main authenticated landing page
 */

'use client';

import { useTranslation } from 'react-i18next';
import { Box, Typography, Paper } from '@mui/material';

export default function DashboardPage() {
  const { t } = useTranslation('dashboard');
  
  return (
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        {t('title')}
      </Typography>
      
      <Paper sx={{ p: 3, mt: 3 }}>
        <Typography variant="body1">
          {t('description')}
        </Typography>
      </Paper>
    </Box>
  );
}
