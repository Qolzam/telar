'use client';

import { useEffect, useState } from 'react';
import { Box, Button, IconButton, Typography } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { AI_ACCENT_GRADIENT } from '@/lib/theme/theme';

const STORAGE_KEY = 'telar-tech-preview-banner-dismissed';

export function TechPreviewBanner() {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const dismissed = localStorage.getItem(STORAGE_KEY);
    if (!dismissed) {
      setVisible(true);
    }
  }, []);

  const handleDismiss = () => {
    setVisible(false);
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, 'true');
    }
  };

  if (!visible) return null;

  return (
    <Box
      sx={{
        width: '100%',
        px: { xs: 2, md: 3 },
        py: 1.25,
        backgroundImage: AI_ACCENT_GRADIENT,
        color: 'common.white',
        display: 'flex',
        alignItems: 'center',
        gap: 2,
        position: 'sticky',
        top: 0,
        zIndex: 1100,
        boxShadow: '0 8px 24px rgba(0,0,0,0.2)',
      }}
    >
      <Typography sx={{ fontWeight: 800, letterSpacing: '-0.01em' }}>
        Telar AI Preview
      </Typography>
      <Typography sx={{ flex: 1, opacity: 0.9 }}>
        You are experiencing the alpha build of the new AI-Powered Architecture. The new AI architecture is live; modules are rolling out weekly.
      </Typography>
      <Button
        variant="outlined"
        href="/roadmap"
        sx={{
          borderColor: 'rgba(255,255,255,0.6)',
          color: 'common.white',
          fontWeight: 700,
          '&:hover': {
            borderColor: 'common.white',
            backgroundColor: 'rgba(255,255,255,0.1)',
          },
        }}
      >
        View Roadmap
      </Button>
      <IconButton
        aria-label="Dismiss banner"
        onClick={handleDismiss}
        sx={{ color: 'common.white' }}
        size="small"
      >
        <CloseIcon fontSize="small" />
      </IconButton>
    </Box>
  );
}
