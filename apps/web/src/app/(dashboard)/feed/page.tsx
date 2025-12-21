'use client';

import { Chip, Container, Stack, Typography } from '@mui/material';
import { PostList } from '@/features/posts/components';
import { AI_ACCENT_GRADIENT } from '@/lib/theme/theme';

export default function FeedPage() {
  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h4">Discover</Typography>
        <Chip
          label="Curated by Telar AI (Alpha)"
          size="small"
          sx={{
            backgroundImage: AI_ACCENT_GRADIENT,
            color: 'common.white',
            fontWeight: 700,
            cursor: 'default',
            userSelect: 'none',
          }}
        />
      </Stack>
      <PostList />
    </Container>
  );
}

