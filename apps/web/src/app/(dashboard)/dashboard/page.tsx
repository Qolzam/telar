/**
 * Dashboard Home Page
 * 
 * Main authenticated landing page
 */

import { Metadata } from 'next';
import { Box, Typography, Paper } from '@mui/material';

export const metadata: Metadata = {
  title: 'Dashboard | Telar',
  description: 'Your personalized dashboard',
};

export default function DashboardPage() {
  return (
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        Welcome to Your Dashboard
      </Typography>
      
      <Paper sx={{ p: 3, mt: 3 }}>
        <Typography variant="body1">
          This is your personalized dashboard. More features will be added as we
          migrate additional plugins.
        </Typography>
      </Paper>
    </Box>
  );
}
