'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
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
  Language as LanguageIcon,
  Info as InfoIcon,
} from '@mui/icons-material';
import { languageNames, rtlLanguages, languages } from '@/lib/i18n/settings';

interface AccountLanguageProps {
  className?: string;
}

export function AccountLanguage({ className }: AccountLanguageProps) {
  const { i18n, t } = useTranslation('settings');
  const router = useRouter();
  const [currentLocale, setCurrentLocale] = useState(i18n.language);

  // Update current locale when i18n language changes
  useEffect(() => {
    const updateLocale = () => setCurrentLocale(i18n.language);
    i18n.on('languageChanged', updateLocale);
    return () => {
      i18n.off('languageChanged', updateLocale);
    };
  }, [i18n]);

  const handleLanguageChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const newLocale = event.target.value;
      
      // Set cookie with 1 year expiry
      document.cookie = `i18next=${newLocale};path=/;max-age=31536000;SameSite=Lax`;
      
      // Change language in i18next
      i18n.changeLanguage(newLocale).then(() => {
        // Trigger a soft refresh to re-render with new locale
        router.refresh();
      });
    },
    [i18n, router]
  );

  const getLanguageDescription = useCallback(
    (locale: string) => {
      const isRTL = rtlLanguages.includes(locale);
      return isRTL
        ? t('language.descriptions.rtl', { language: languageNames[locale] })
        : t('language.descriptions.ltr', { language: languageNames[locale] });
    },
    [t]
  );

  return (
    <Card className={className} sx={{ p: 3 }}>
      <Stack spacing={3}>
        <Box>
          <Typography variant="h6" gutterBottom>
            {t('language.title')}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {t('language.description')}
          </Typography>
        </Box>

        <Divider />

        <FormControl component="fieldset" fullWidth>
          <FormLabel component="legend" sx={{ mb: 2, fontWeight: 600 }}>
            {t('language.selectLanguage')}
          </FormLabel>
          <RadioGroup
            value={currentLocale}
            onChange={handleLanguageChange}
            sx={{ gap: 1 }}
            aria-label={t('language.selectLanguage')}
          >
            {languages.map((locale) => (
              <FormControlLabel
                key={locale}
                value={locale}
                control={<Radio />}
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                    <LanguageIcon fontSize="small" />
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="body2" fontWeight={500}>
                        {languageNames[locale]}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {getLanguageDescription(locale)}
                      </Typography>
                    </Box>
                    {currentLocale === locale && (
                      <Chip
                        label={t('language.active')}
                        size="small"
                        color="primary"
                        variant="outlined"
                      />
                    )}
                    {rtlLanguages.includes(locale) && (
                      <Chip
                        label={t('language.rtl')}
                        size="small"
                        variant="outlined"
                      />
                    )}
                  </Box>
                }
                sx={{
                  p: 2,
                  borderRadius: 1,
                  border: '1px solid',
                  borderColor: currentLocale === locale ? 'primary.main' : 'divider',
                  bgcolor: currentLocale === locale ? 'action.selected' : 'transparent',
                  '&:hover': {
                    bgcolor: 'action.hover',
                  },
                }}
              />
            ))}
          </RadioGroup>
        </FormControl>

        <Alert severity="info" icon={<InfoIcon />}>
          <AlertTitle>{t('language.info.title')}</AlertTitle>
          <Typography variant="body2">
            {t('language.info.currentLanguage', { language: languageNames[currentLocale] })}
          </Typography>
          <Typography variant="body2" sx={{ mt: 1 }}>
            {t('language.info.persistence')}
          </Typography>
        </Alert>
      </Stack>
    </Card>
  );
}
