'use client';

import { Box, Button, Chip, Stack, Typography } from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import PendingIcon from '@mui/icons-material/Pending';
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked';
import Image from 'next/image';
import { AI_ACCENT_GRADIENT } from '@/lib/theme/theme';

type FeatureStatus = 'live' | 'in-progress' | 'coming-soon';

type FeaturePreviewProps = {
  title: string;
  subtitle: string;
  description?: string;
  bullets: Array<{
    label: string;
    status?: FeatureStatus;
  }>;
  ctaLabel?: string;
  ctaHref?: string;
  badgeLabel?: string;
  backgroundImage?: string;
};

const STATUS_ICON: Record<FeatureStatus, JSX.Element> = {
  live: <CheckCircleIcon color="success" fontSize="small" />,
  'in-progress': <PendingIcon color="warning" fontSize="small" />,
  'coming-soon': <RadioButtonUncheckedIcon color="disabled" fontSize="small" />,
};

export function FeaturePreview({
  title,
  subtitle,
  description,
  bullets,
  ctaLabel,
  ctaHref,
  badgeLabel,
  backgroundImage,
}: FeaturePreviewProps) {
  return (
    <Box
      sx={{
        position: 'relative',
        overflow: 'hidden',
        borderRadius: 3,
        bgcolor: 'background.paper',
        border: '1px solid',
        borderColor: 'divider',
        boxShadow: '0 10px 30px rgba(15, 23, 42, 0.15)',
        p: { xs: 3, md: 4 },
        display: 'grid',
        gap: 3,
        gridTemplateColumns: { xs: '1fr', md: '1.3fr 1fr' },
      }}
    >
      {backgroundImage ? (
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            opacity: 0.25,
            filter: 'blur(12px)',
            transform: 'scale(1.05)',
          }}
        >
          <Image
            src={backgroundImage}
            alt=""
            fill
            sizes="100vw"
            priority
            style={{ objectFit: 'cover' }}
          />
        </Box>
      ) : null}

      <Box
        sx={{
          position: 'absolute',
          inset: 0,
          background:
            'radial-gradient(circle at 20% 20%, rgba(99, 102, 241, 0.12), transparent 35%), radial-gradient(circle at 80% 0%, rgba(168, 85, 247, 0.12), transparent 30%)',
          opacity: 0.9,
        }}
      />

      <Box sx={{ position: 'relative', zIndex: 1, display: 'grid', gap: 1.5 }}>
        {badgeLabel ? (
          <Chip
            label={badgeLabel}
            sx={{
              alignSelf: 'flex-start',
              backgroundImage: AI_ACCENT_GRADIENT,
              color: 'common.white',
              fontWeight: 700,
            }}
          />
        ) : null}
        <Typography variant="h4" sx={{ fontWeight: 800, letterSpacing: '-0.02em' }}>
          {title}
        </Typography>
        <Typography variant="subtitle1" color="text.secondary" sx={{ fontSize: 16 }}>
          {subtitle}
        </Typography>
        {description ? (
          <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 640 }}>
            {description}
          </Typography>
        ) : null}

        <Stack spacing={1.2} sx={{ mt: 1 }}>
          {bullets.map((item) => (
            <Stack key={item.label} direction="row" spacing={1} alignItems="center">
              {item.status ? STATUS_ICON[item.status] : <RadioButtonUncheckedIcon color="disabled" fontSize="small" />}
              <Typography variant="body1" sx={{ fontWeight: 600 }}>
                {item.label}
              </Typography>
            </Stack>
          ))}
        </Stack>

        {ctaLabel && ctaHref ? (
          <Button
            variant="contained"
            href={ctaHref}
            sx={{
              mt: 2,
              alignSelf: 'flex-start',
              backgroundImage: AI_ACCENT_GRADIENT,
              color: 'common.white',
              fontWeight: 700,
              px: 3,
              py: 1,
              borderRadius: 2,
              '&:hover': {
                backgroundImage: AI_ACCENT_GRADIENT,
                filter: 'brightness(0.95)',
              },
            }}
          >
            {ctaLabel}
          </Button>
        ) : null}
      </Box>

      <Box
        sx={{
          position: 'relative',
          zIndex: 1,
          borderRadius: 3,
          background: AI_ACCENT_GRADIENT,
          p: 3,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          minHeight: 240,
          color: 'common.white',
          overflow: 'hidden',
        }}
      >
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            background:
              'radial-gradient(circle at 30% 30%, rgba(255,255,255,0.2), transparent 40%), radial-gradient(circle at 70% 20%, rgba(255,255,255,0.12), transparent 35%)',
            opacity: 0.8,
          }}
        />
        <Box
          sx={{
            position: 'relative',
            backdropFilter: 'blur(6px)',
            border: '1px solid rgba(255,255,255,0.3)',
            borderRadius: 2,
            p: 2.5,
            width: '100%',
            maxWidth: 360,
            backgroundColor: 'rgba(15, 23, 42, 0.25)',
          }}
        >
          <Typography variant="body2" sx={{ opacity: 0.9, mb: 1, fontWeight: 700 }}>
            Sneak peek
          </Typography>
          <Typography variant="subtitle1" sx={{ fontWeight: 800, lineHeight: 1.4 }}>
            Visual preview of the experience
          </Typography>
          <Typography variant="body2" sx={{ opacity: 0.9, mt: 1.5 }}>
            A blurred mock of the Expertise Graph so users feel the product is already taking shape.
          </Typography>
        </Box>
      </Box>
    </Box>
  );
}
