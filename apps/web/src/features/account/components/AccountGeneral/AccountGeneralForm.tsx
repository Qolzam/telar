'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
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
} from '@mui/material';
import PhotoCameraIcon from '@mui/icons-material/PhotoCamera';
import { useUpdateProfileMutation } from '@/features/profile/client';
import type { UserProfileModel } from '@telar/sdk';

const profileSchema = z.object({
  fullName: z.string().min(1, 'Full name is required'),
  socialName: z.string().min(1, 'Social name is required'),
  tagLine: z.string().optional(),
  webUrl: z.string().url('Must be a valid URL').optional().or(z.literal('')),
  companyName: z.string().optional(),
  facebookId: z.string().optional(),
  instagramId: z.string().optional(),
  twitterId: z.string().optional(),
});

type ProfileFormData = z.infer<typeof profileSchema>;

interface AccountGeneralFormProps {
  profile: UserProfileModel;
}

export function AccountGeneralForm({ profile }: AccountGeneralFormProps) {
  const updateMutation = useUpdateProfileMutation();

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

  const onSubmit = async (data: ProfileFormData) => {
    try {
      await updateMutation.mutateAsync(data);
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
              <Avatar
                src={profile.avatar}
                alt={profile.fullName}
                sx={{ width: 128, height: 128, mx: 'auto' }}
              >
                {profile.fullName.charAt(0).toUpperCase()}
              </Avatar>
              <IconButton
                sx={{
                  position: 'absolute',
                  bottom: 0,
                  right: 0,
                  bgcolor: 'background.paper',
                  '&:hover': { bgcolor: 'action.hover' },
                }}
                size="small"
              >
                <PhotoCameraIcon fontSize="small" />
              </IconButton>
            </Box>
            <Typography variant="caption" color="text.secondary">
              Allowed *.jpeg, *.jpg, *.png, *.gif
              <br />
              Max size of 3.1 MB
            </Typography>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, md: 8 }}>
          <Card sx={{ p: 3 }}>
            <Stack spacing={3}>
              <Typography variant="h6">General Information</Typography>

              <Grid container spacing={2}>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    {...register('fullName')}
                    fullWidth
                    label="Full Name"
                    error={!!errors.fullName}
                    helperText={errors.fullName?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    {...register('socialName')}
                    fullWidth
                    label="Social Name"
                    error={!!errors.socialName}
                    helperText={errors.socialName?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12 }}>
                  <TextField
                    {...register('tagLine')}
                    fullWidth
                    label="Tag Line"
                    placeholder="A short description about you"
                    error={!!errors.tagLine}
                    helperText={errors.tagLine?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    {...register('companyName')}
                    fullWidth
                    label="Company"
                    error={!!errors.companyName}
                    helperText={errors.companyName?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    {...register('webUrl')}
                    fullWidth
                    label="Website"
                    placeholder="https://example.com"
                    error={!!errors.webUrl}
                    helperText={errors.webUrl?.message}
                  />
                </Grid>
              </Grid>

              <Stack direction="row" justifyContent="flex-end" spacing={2}>
                <Button variant="outlined" type="button">
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="contained"
                  disabled={isSubmitting}
                  startIcon={isSubmitting && <CircularProgress size={16} />}
                >
                  {isSubmitting ? 'Saving...' : 'Save Changes'}
                </Button>
              </Stack>
            </Stack>
          </Card>
        </Grid>
      </Grid>
    </form>
  );
}


