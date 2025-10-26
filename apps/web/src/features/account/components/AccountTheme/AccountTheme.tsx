'use client';

import { useCallback } from 'react';
import {
  Box,
  Card,
  Stack,
  Typography,
  FormControl,
  FormLabel,
  RadioGroup,
  FormControlLabel,
  Radio,
  Divider,
  Chip,
  Alert,
  AlertTitle,
} from '@mui/material';
import {
  LightMode as LightModeIcon,
  DarkMode as DarkModeIcon,
  Settings as SystemIcon,
  Info as InfoIcon,
} from '@mui/icons-material';
import { useTheme } from '@/lib/theme/use-theme';

interface AccountThemeProps {
  className?: string;
}

export function AccountTheme({ className }: AccountThemeProps) {
  const { colorScheme, setColorScheme, resolvedMode } = useTheme();

  const handleThemeChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const scheme = event.target.value as 'light' | 'dark' | 'system';
    setColorScheme(scheme);
  }, [setColorScheme]);

  const getThemeDescription = useCallback((scheme: 'light' | 'dark' | 'system') => {
    switch (scheme) {
      case 'light':
        return 'Always use light theme regardless of system settings';
      case 'dark':
        return 'Always use dark theme regardless of system settings';
      case 'system':
        return `Follow your system settings (Currently: ${resolvedMode})`;
      default:
        return '';
    }
  }, [resolvedMode]);

  const getThemeIcon = useCallback((scheme: 'light' | 'dark' | 'system') => {
    switch (scheme) {
      case 'light':
        return <LightModeIcon fontSize="small" />;
      case 'dark':
        return <DarkModeIcon fontSize="small" />;
      case 'system':
        return <SystemIcon fontSize="small" />;
      default:
        return null;
    }
  }, []);

  return (
    <Card className={className} sx={{ p: 3 }}>
      <Stack spacing={3}>
        <Box>
          <Typography variant="h6" gutterBottom>
            Theme Preferences
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Customize the appearance of your interface. You can choose a specific theme or let it follow your system settings.
          </Typography>
        </Box>

        <Divider />

        <FormControl component="fieldset" fullWidth>
          <FormLabel component="legend" sx={{ mb: 2, fontWeight: 600 }}>
            Color Scheme
          </FormLabel>
          <RadioGroup
            value={colorScheme}
            onChange={handleThemeChange}
            sx={{ gap: 1 }}
            aria-label="Theme selection"
          >
            <FormControlLabel
              value="light"
              control={<Radio />}
              label={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                  {getThemeIcon('light')}
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="body2" fontWeight={500}>
                      Light
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {getThemeDescription('light')}
                    </Typography>
                  </Box>
                  {colorScheme === 'light' && (
                    <Chip 
                      label="Active" 
                      size="small" 
                      color="primary" 
                      variant="outlined"
                    />
                  )}
                </Box>
              }
              sx={{
                p: 2,
                borderRadius: 1,
                border: '1px solid',
                borderColor: colorScheme === 'light' ? 'primary.main' : 'divider',
                bgcolor: colorScheme === 'light' ? 'action.selected' : 'transparent',
                '&:hover': {
                  bgcolor: 'action.hover',
                },
                '&.Mui-focusVisible': {
                  bgcolor: 'action.focus',
                },
              }}
            />

            <FormControlLabel
              value="dark"
              control={<Radio />}
              label={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                  {getThemeIcon('dark')}
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="body2" fontWeight={500}>
                      Dark
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {getThemeDescription('dark')}
                    </Typography>
                  </Box>
                  {colorScheme === 'dark' && (
                    <Chip 
                      label="Active" 
                      size="small" 
                      color="primary" 
                      variant="outlined"
                    />
                  )}
                </Box>
              }
              sx={{
                p: 2,
                borderRadius: 1,
                border: '1px solid',
                borderColor: colorScheme === 'dark' ? 'primary.main' : 'divider',
                bgcolor: colorScheme === 'dark' ? 'action.selected' : 'transparent',
                '&:hover': {
                  bgcolor: 'action.hover',
                },
                '&.Mui-focusVisible': {
                  bgcolor: 'action.focus',
                },
              }}
            />

            <FormControlLabel
              value="system"
              control={<Radio />}
              label={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                  {getThemeIcon('system')}
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="body2" fontWeight={500}>
                      System
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {getThemeDescription('system')}
                    </Typography>
                  </Box>
                  {colorScheme === 'system' && (
                    <Chip 
                      label="Active" 
                      size="small" 
                      color="primary" 
                      variant="outlined"
                    />
                  )}
                </Box>
              }
              sx={{
                p: 2,
                borderRadius: 1,
                border: '1px solid',
                borderColor: colorScheme === 'system' ? 'primary.main' : 'divider',
                bgcolor: colorScheme === 'system' ? 'action.selected' : 'transparent',
                '&:hover': {
                  bgcolor: 'action.hover',
                },
                '&.Mui-focusVisible': {
                  bgcolor: 'action.focus',
                },
              }}
            />
          </RadioGroup>
        </FormControl>

        <Alert severity="info" icon={<InfoIcon />}>
          <AlertTitle>Theme Information</AlertTitle>
          <Typography variant="body2">
            <strong>Current theme:</strong> {colorScheme === 'system' ? `System (${resolvedMode})` : colorScheme}
          </Typography>
          <Typography variant="body2" sx={{ mt: 1 }}>
            Your theme preference is saved automatically and will be restored when you return to the application.
          </Typography>
        </Alert>
      </Stack>
    </Card>
  );
}