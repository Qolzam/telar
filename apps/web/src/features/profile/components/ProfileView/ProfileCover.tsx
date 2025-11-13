'use client';

import { Avatar, Box, Stack, ListItemText, useTheme } from '@mui/material';

interface ProfileCoverProps {
  name: string;
  avatarUrl?: string;
  role?: string;
  coverUrl?: string;
}

export function ProfileCover({ name, avatarUrl, role, coverUrl }: ProfileCoverProps) {
  const theme = useTheme();

  return (
    <Box
      sx={{
        height: '100%',
        color: 'common.white',
        backgroundImage: coverUrl ? `url(${coverUrl})` : 'none',
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        bgcolor: theme.palette.primary.dark,
        position: 'relative',
        '&::before': {
          content: '""',
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          bgcolor: 'rgba(0, 0, 0, 0.4)',
        },
      }}
    >
      <Stack
        direction={{ xs: 'column', md: 'row' }}
        sx={{
          position: { md: 'absolute' },
          bottom: { md: 24 },
          left: { md: 24 },
          zIndex: 10,
          pt: { xs: 6, md: 0 },
        }}
      >
        <Avatar
          alt={name}
          src={avatarUrl}
          sx={{
            mx: 'auto',
            width: { xs: 64, md: 128 },
            height: { xs: 64, md: 128 },
            border: `solid 2px ${theme.palette.common.white}`,
          }}
        >
          {name?.charAt(0).toUpperCase()}
        </Avatar>

        <ListItemText
          sx={{ mt: 3, ml: { md: 3 }, textAlign: { xs: 'center', md: 'unset' } }}
          primary={name}
          secondary={role}
          slotProps={{
            primary: { typography: 'h4' },
            secondary: {
              mt: 0.5,
              color: 'inherit',
              component: 'span',
              typography: 'body2',
              sx: { opacity: 0.7 },
            },
          }}
        />
      </Stack>
    </Box>
  );
}


