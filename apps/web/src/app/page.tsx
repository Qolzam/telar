import { Button, Typography, Box, Stack } from '@mui/material';
import { Home as HomeIcon } from '@mui/icons-material';

export default function HomePage() {
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
          Welcome to Telar
        </Typography>
        <Typography variant="body1" color="text.secondary" textAlign="center" maxWidth={600}>
          Next.js 15 + MUI v7 + React Query v5 + TypeScript 5.6
        </Typography>
        <Stack direction="row" spacing={2}>
          <Button variant="contained" size="large">
            Get Started
          </Button>
          <Button variant="outlined" size="large">
            Learn More
          </Button>
        </Stack>
      </Stack>
    </Box>
  );
}
