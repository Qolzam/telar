'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Card,
  Grid,
  Stack,
  TextField,
  Typography,
  InputAdornment,
  Alert,
} from '@mui/material';
import { LoadingButton } from '@mui/lab';
import FacebookIcon from '@mui/icons-material/Facebook';
import TwitterIcon from '@mui/icons-material/Twitter';
import InstagramIcon from '@mui/icons-material/Instagram';
import { useState } from 'react';
import { useUpdateProfileMutation } from '@/features/profile/client';
import type { UserProfileModel } from '@telar/sdk';

const socialSchema = z.object({
  facebookId: z.string().optional(),
  twitterId: z.string().optional(),
  instagramId: z.string().optional(),
});

type SocialFormData = z.infer<typeof socialSchema>;

interface SocialLinksFormProps {
  profile: UserProfileModel;
}

export function SocialLinksForm({ profile }: SocialLinksFormProps) {
  const updateMutation = useUpdateProfileMutation();
  const [successMessage, setSuccessMessage] = useState('');

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<SocialFormData>({
    resolver: zodResolver(socialSchema),
    defaultValues: {
      facebookId: profile.facebookId || '',
      twitterId: profile.twitterId || '',
      instagramId: profile.instagramId || '',
    },
  });

  const onSubmit = async (data: SocialFormData) => {
    try {
      setSuccessMessage('');
      await updateMutation.mutateAsync(data);
      setSuccessMessage('Social links updated successfully!');
    } catch (error) {
      console.error('Failed to update social links:', error);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <Card sx={{ p: 3 }}>
        <Stack spacing={3}>
          <Typography variant="h6">Social Links</Typography>

          {successMessage && <Alert severity="success">{successMessage}</Alert>}

          <Grid container spacing={2}>
            <Grid size={{ xs: 12 }}>
              <TextField
                {...register('facebookId')}
                fullWidth
                label="Facebook"
                placeholder="username"
                error={!!errors.facebookId}
                helperText={errors.facebookId?.message}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <FacebookIcon sx={{ color: '#1877F2' }} />
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>

            <Grid size={{ xs: 12 }}>
              <TextField
                {...register('twitterId')}
                fullWidth
                label="Twitter"
                placeholder="username"
                error={!!errors.twitterId}
                helperText={errors.twitterId?.message}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <TwitterIcon sx={{ color: '#1DA1F2' }} />
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>

            <Grid size={{ xs: 12 }}>
              <TextField
                {...register('instagramId')}
                fullWidth
                label="Instagram"
                placeholder="username"
                error={!!errors.instagramId}
                helperText={errors.instagramId?.message}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <InstagramIcon sx={{ color: '#E4405F' }} />
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>
          </Grid>

          <LoadingButton
            type="submit"
            variant="contained"
            loading={isSubmitting}
            sx={{ alignSelf: 'flex-start' }}
          >
            Save Changes
          </LoadingButton>
        </Stack>
      </Card>
    </form>
  );
}


