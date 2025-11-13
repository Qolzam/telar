'use client';

import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation('settings');
  const { colorScheme, setColorScheme, resolvedMode } = useTheme();

  const handleThemeChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const scheme = event.target.value as 'light' | 'dark' | 'system';
    setColorScheme(scheme);
  }, [setColorScheme]);

  const getThemeDescription = useCallback((scheme: 'light' | 'dark' | 'system') => {
    switch (scheme) {
      case 'light':
        return t('theme.modes.light.description');
      case 'dark':
        return t('theme.modes.dark.description');
      case 'system':
        return t('theme.modes.system.description', { mode: resolvedMode });
      default:
        return '';
    }
  }, [t, resolvedMode]);

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
            {t('theme.title')}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {t('theme.description')}
          </Typography>
        </Box>

        <Divider />

        <FormControl component="fieldset" fullWidth>
          <FormLabel component="legend" sx={{ mb: 2, fontWeight: 600 }}>
            {t('theme.colorScheme')}
          </FormLabel>
          <RadioGroup
            value={colorScheme}
            onChange={handleThemeChange}
            sx={{ gap: 1 }}
            aria-label={t('theme.selection')}
          >
            <FormControlLabel
              value="light"
              control={<Radio />}
              label={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                  {getThemeIcon('light')}
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="body2" fontWeight={500}>
                      {t('theme.modes.light.label')}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {getThemeDescription('light')}
                    </Typography>
                  </Box>
                  {colorScheme === 'light' && (
                    <Chip 
                      label={t('theme.active')} 
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
                      {t('theme.modes.dark.label')}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {getThemeDescription('dark')}
                    </Typography>
                  </Box>
                  {colorScheme === 'dark' && (
                    <Chip 
                      label={t('theme.active')} 
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
                      {t('theme.modes.system.label')}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {getThemeDescription('system')}
                    </Typography>
                  </Box>
                  {colorScheme === 'system' && (
                    <Chip 
                      label={t('theme.active')} 
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
          <AlertTitle>{t('theme.info.title')}</AlertTitle>
          <Typography variant="body2">
            {t('theme.info.currentTheme', { theme: colorScheme === 'system' ? `System (${resolvedMode})` : colorScheme })}
          </Typography>
          <Typography variant="body2" sx={{ mt: 1 }}>
            {t('theme.info.persistence')}
          </Typography>
        </Alert>
      </Stack>
    </Card>
  );
}