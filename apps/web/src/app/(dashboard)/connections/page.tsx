'use client';

import { Container, Stack, Typography } from '@mui/material';
import { FeaturePreview } from '@/components/FeaturePreview';

const GRAPH_BG =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="800" height="400" viewBox="0 0 800 400"><defs><linearGradient id="g" x1="0" x2="1" y1="0" y2="1"><stop stop-color="%236366f1" stop-opacity="0.9"/><stop offset="1" stop-color="%23a855f7" stop-opacity="0.8"/></linearGradient></defs><rect width="800" height="400" fill="%230b1021"/><g stroke="url(%23g)" stroke-width="1.2" stroke-opacity="0.35" fill="none"><circle cx="120" cy="120" r="80"/><circle cx="300" cy="180" r="120"/><circle cx="520" cy="140" r="110"/><circle cx="640" cy="240" r="90"/></g><g fill="url(%23g)" fill-opacity="0.9"><circle cx="120" cy="120" r="8"/><circle cx="180" cy="180" r="6"/><circle cx="260" cy="120" r="7"/><circle cx="320" cy="220" r="9"/><circle cx="440" cy="140" r="8"/><circle cx="520" cy="200" r="7"/><circle cx="600" cy="120" r="9"/><circle cx="660" cy="240" r="7"/></g></svg>';

export default function ConnectionsPage() {
  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <Stack spacing={2} sx={{ mb: 2 }}>
        <Typography variant="h3" sx={{ fontWeight: 800, letterSpacing: '-0.02em' }}>
          Connections
        </Typography>
        <Typography variant="body1" color="text.secondary">
          Social Graphs are evolving into Interest Graphs. Welcome to the Expertise Engine preview.
        </Typography>
      </Stack>

      <FeaturePreview
        title="The Expertise Engine"
        subtitle="We are building an AI-powered connection system that automatically pairs you with verified experts and peers by topic."
        description="Instead of noisy follow lists, you will get signal-first matches across your interests, skills, and goals."
        bullets={[
          { label: 'AI-Verified Expert Matching', status: 'coming-soon' },
          { label: 'Topic-based Clustering', status: 'coming-soon' },
          { label: 'Noise-free Signal Feed', status: 'coming-soon' },
        ]}
        ctaLabel="Notify me when the Expertise Engine launches"
        ctaHref="/roadmap#expertise-engine"
        badgeLabel="Velvet Rope Preview"
        backgroundImage={GRAPH_BG}
      />
    </Container>
  );
}
