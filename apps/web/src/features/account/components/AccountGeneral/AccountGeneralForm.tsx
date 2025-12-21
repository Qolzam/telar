'use client';

import { useState, useRef, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Card,
  Grid,
  Stack,
  TextField,
  Typography,
  Button,
  Avatar,
  IconButton,
  CircularProgress,
  LinearProgress,
  Snackbar,
  Alert,
} from '@mui/material';
import PhotoCameraIcon from '@mui/icons-material/PhotoCamera';
import { useUpdateProfileMutation } from '@/features/profile/client';
import type { UserProfileModel } from '@telar/sdk';
import { sdk } from '@/lib/sdk';
import { uploadFileWithCompression } from '@telar/sdk';

type ProfileFormData = {
  fullName: string;
  socialName: string;
  tagLine?: string;
  webUrl?: string;
  companyName?: string;
  facebookId?: string;
  instagramId?: string;
  twitterId?: string;
};

interface AccountGeneralFormProps {
  profile: UserProfileModel;
}

export function AccountGeneralForm({ profile }: AccountGeneralFormProps) {
  const { t } = useTranslation(['settings', 'validation', 'common']);
  const updateMutation = useUpdateProfileMutation();
  const [avatarUrl, setAvatarUrl] = useState<string | undefined>(profile.avatar);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null); // Blob URL for optimistic preview
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const profileSchema = z.object({
    fullName: z.string().min(1, t('validation:profile.fullName')),
    socialName: z.string().min(1, t('validation:profile.socialName')),
    tagLine: z.string().optional(),
    webUrl: z.string().url(t('validation:url.invalid')).optional().or(z.literal('')),
    companyName: z.string().optional(),
    facebookId: z.string().optional(),
    instagramId: z.string().optional(),
    twitterId: z.string().optional(),
  });

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      fullName: profile.fullName || '',
      socialName: profile.socialName || '',
      tagLine: profile.tagLine || '',
      webUrl: profile.webUrl || '',
      companyName: profile.companyName || '',
      facebookId: profile.facebookId || '',
      instagramId: profile.instagramId || '',
      twitterId: profile.twitterId || '',
    },
  });

  // Cleanup blob URL on unmount
  useEffect(() => {
    return () => {
      if (previewUrl) {
        console.log('Revoking URL:', previewUrl);
        URL.revokeObjectURL(previewUrl);
      }
    };
  }, [previewUrl]);

  const handleAvatarClick = () => {
    console.log('[Avatar] Click handler fired');
    console.log('[Avatar] fileInputRef.current:', fileInputRef.current);
    if (fileInputRef.current) {
      fileInputRef.current.click();
      console.log('[Avatar] File input click triggered');
    } else {
      console.error('[Avatar] fileInputRef.current is NULL');
    }
  };

  const handleAvatarChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setUploadError(null);
    setUploadProgress(0);

    // 1. Generate optimistic preview (blob URL) - immediate feedback
    const blobUrl = URL.createObjectURL(file);
    setPreviewUrl(blobUrl);

    // Store previous avatar URL for error recovery
    const previousAvatarUrl = avatarUrl;

    try {
      const { url } = await uploadFileWithCompression(
        file,
        sdk.storage,
        (progress) => setUploadProgress(progress)
      );

      // Clean up blob URL
      if (previewUrl) {
        console.log('Revoking URL (success):', previewUrl);
        URL.revokeObjectURL(previewUrl);
      }

      setAvatarUrl(url);
      setPreviewUrl(null);
      setUploadProgress(null);

      // Update profile with new avatar URL
      await updateMutation.mutateAsync({
        avatar: url,
      });
    } catch (error) {
      console.error('Failed to upload avatar:', error);
      
      // Clean up blob URL on error
      if (previewUrl) {
        console.log('Revoking URL (error):', previewUrl);
        URL.revokeObjectURL(previewUrl);
      }
      
      // Revert to previous avatar
      setPreviewUrl(null);
      setAvatarUrl(previousAvatarUrl);
      setUploadError(error instanceof Error ? error.message : 'Failed to upload avatar');
      setUploadProgress(null);
    } finally {
      // Reset file input
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const onSubmit = async (data: ProfileFormData) => {
    try {
      await updateMutation.mutateAsync({
        ...data,
        ...(avatarUrl && avatarUrl !== profile.avatar ? { avatar: avatarUrl } : {}),
      });
    } catch (error) {
      console.error('Failed to update profile:', error);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ p: 3, textAlign: 'center' }}>
            <Box sx={{ mb: 3, position: 'relative', display: 'inline-block' }}>
              <Box sx={{ position: 'relative', display: 'inline-block' }}>
                <Avatar
                  src={previewUrl || avatarUrl || profile.avatar}
                  alt={profile.fullName}
                  onClick={uploadProgress === null ? handleAvatarClick : undefined}
                  sx={{
                    width: 128,
                    height: 128,
                    mx: 'auto',
                    filter: uploadProgress !== null && uploadProgress < 50 ? 'blur(2px)' : 'none',
                    transition: 'filter 0.2s',
                    cursor: uploadProgress === null ? 'pointer' : 'default',
                    '&:hover': uploadProgress === null ? {
                      opacity: 0.8,
                    } : {},
                  }}
                >
                  {profile.fullName.charAt(0).toUpperCase()}
                </Avatar>
                {/* Compression indicator overlay */}
                {uploadProgress !== null && uploadProgress < 50 && (
                  <Box
                    sx={{
                      position: 'absolute',
                      top: '50%',
                      left: '50%',
                      transform: 'translate(-50%, -50%)',
                      bgcolor: 'rgba(0, 0, 0, 0.7)',
                      color: 'white',
                      px: 2,
                      py: 1,
                      borderRadius: 1,
                      display: 'flex',
                      alignItems: 'center',
                      gap: 1,
                      zIndex: 1,
                      pointerEvents: 'none', // Allow clicks to pass through
                    }}
                  >
                    <CircularProgress size={16} sx={{ color: 'white' }} />
                    <Typography variant="caption">✨ Optimizing...</Typography>
                  </Box>
                )}
                {/* Upload progress spinner overlay */}
                {uploadProgress !== null && uploadProgress >= 50 && (
                  <Box
                    sx={{
                      position: 'absolute',
                      top: 0,
                      left: '50%',
                      transform: 'translateX(-50%)',
                      width: 128,
                      height: 128,
                      borderRadius: '50%',
                      bgcolor: 'rgba(0, 0, 0, 0.5)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      pointerEvents: 'none', // Allow clicks to pass through
                    }}
                  >
                    <CircularProgress
                      size={64}
                      variant="determinate"
                      value={uploadProgress}
                      sx={{ color: 'primary.contrastText' }}
                    />
                  </Box>
                )}
              </Box>
              <IconButton
                onClick={handleAvatarClick}
                disabled={uploadProgress !== null}
                sx={{
                  position: 'absolute',
                  bottom: 0,
                  right: 0,
                  bgcolor: 'background.paper',
                  '&:hover': { bgcolor: 'action.hover' },
                  zIndex: 10, // Ensure button is above overlays
                }}
                size="small"
              >
                <PhotoCameraIcon fontSize="small" />
              </IconButton>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                style={{ display: 'none' }}
                onChange={handleAvatarChange}
              />
            </Box>
            <Typography variant="caption" color="text.secondary">
              {t('settings:general.avatar.allowedFormats')}
              <br />
              {t('settings:general.avatar.maxSize')}
            </Typography>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, md: 8 }}>
          <Card sx={{ p: 3 }}>
            <Stack spacing={3}>
              <Typography variant="h6">{t('settings:general.title')}</Typography>

              <Grid container spacing={2}>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    {...register('fullName')}
                    fullWidth
                    label={t('settings:general.fields.fullName')}
                    error={!!errors.fullName}
                    helperText={errors.fullName?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    {...register('socialName')}
                    fullWidth
                    label={t('settings:general.fields.socialName')}
                    error={!!errors.socialName}
                    helperText={errors.socialName?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12 }}>
                  <TextField
                    {...register('tagLine')}
                    fullWidth
                    label={t('settings:general.fields.tagLine')}
                    placeholder={t('settings:general.fields.tagLinePlaceholder')}
                    error={!!errors.tagLine}
                    helperText={errors.tagLine?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    {...register('companyName')}
                    fullWidth
                    label={t('settings:general.fields.company')}
                    error={!!errors.companyName}
                    helperText={errors.companyName?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    {...register('webUrl')}
                    fullWidth
                    label={t('settings:general.fields.website')}
                    placeholder={t('settings:general.fields.websitePlaceholder')}
                    error={!!errors.webUrl}
                    helperText={errors.webUrl?.message}
                  />
                </Grid>
              </Grid>

              <Stack direction="row" justifyContent="flex-end" spacing={2}>
                <Button variant="outlined" type="button">
                  {t('common:buttons.cancel')}
                </Button>
                <Button
                  type="submit"
                  variant="contained"
                  disabled={isSubmitting}
                  startIcon={isSubmitting && <CircularProgress size={16} />}
                >
                  {isSubmitting ? t('common:states.saving') : t('common:buttons.save')}
                </Button>
              </Stack>
            </Stack>
          </Card>
        </Grid>
      </Grid>
      <Snackbar
        open={!!uploadError}
        autoHideDuration={6000}
        onClose={() => setUploadError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setUploadError(null)} severity="error">
          {uploadError}
        </Alert>
      </Snackbar>
    </form>
  );
}


