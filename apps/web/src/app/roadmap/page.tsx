'use client';

import { Box, Chip, Container, Divider, Stack, Typography } from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import PendingIcon from '@mui/icons-material/Pending';
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked';
import { FeaturePreview } from '@/components/FeaturePreview';
import { AI_ACCENT_GRADIENT } from '@/lib/theme/theme';

const STATUS_META = {
  ready: { label: 'Ready', icon: <CheckCircleIcon color="success" fontSize="small" /> },
  'in-integration': { label: 'In Integration', icon: <PendingIcon color="warning" fontSize="small" /> },
  planned: { label: 'Planned', icon: <RadioButtonUncheckedIcon color="disabled" fontSize="small" /> },
} as const;

type StackStatusKey = keyof typeof STATUS_META;

type RoadmapItem = {
  title: string;
  description: string;
  id?: string;
  stackStatuses: Array<{
    label: string;
    status: StackStatusKey;
  }>;
};

const ROADMAP: RoadmapItem[] = [
  {
    title: 'AI Engine Core',
    description: 'Standalone microservice powering moderation, bios, and ignition.',
    stackStatuses: [{ label: 'AI Engine Core', status: 'ready' }],
  },
  {
    title: 'AI Co-Moderator (The Guardian)',
    description: 'Moderation guardrails backed by the AI engine; wiring UI is ongoing.',
    stackStatuses: [{ label: 'The Guardian', status: 'in-integration' }],
  },
  {
    title: 'AI Bio (The Muse)',
    description: 'Profile storytelling powered by AI; UI hooks are being connected.',
    stackStatuses: [{ label: 'The Muse', status: 'in-integration' }],
  },
  {
    title: 'Community Ignition (The Spark)',
    description: 'AI prompts and conversation starters to keep the community vibrant.',
    stackStatuses: [{ label: 'The Spark', status: 'in-integration' }],
  },
  {
    title: 'Expertise Engine (Verified Answers)',
    description: 'Topic-based expert matching and verified answers with signal-first feeds.',
    id: 'expertise-engine',
    stackStatuses: [{ label: 'Expertise Engine', status: 'planned' }],
  },
];

export default function RoadmapPage() {
  return (
    <Container maxWidth="md" sx={{ py: 4, display: 'grid', gap: 3 }}>
      <Box>
        <Typography variant="h3" sx={{ fontWeight: 800, letterSpacing: '-0.02em' }}>
          Telar AI Roadmap
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ mt: 1, maxWidth: 720 }}>
          A focused look at what is live, what is in-flight, and what is next. No external boards—everything you need is here.
        </Typography>
      </Box>

      <FeaturePreview
        title="The Trifecta"
        subtitle="Guardian · Muse · Spark"
        description="Core AI pillars that keep the platform safe, expressive, and always-on."
        bullets={[
          { label: 'Guardian — AI Co-Moderator (Live)', status: 'live' },
          { label: 'Muse — AI Bio (Live)', status: 'live' },
          { label: 'Spark — Community Ignition (In Progress)', status: 'in-progress' },
        ]}
        badgeLabel="AI Preview"
        ctaLabel="View Expertise Engine Preview"
        ctaHref="/connections"
      />

      <Divider />

      <Stack spacing={2}>
        {ROADMAP.map((item) => (
          <Box
            key={item.title}
            id={item.id}
            sx={{
              border: '1px solid',
              borderColor: 'divider',
              borderRadius: 2,
              p: 2.5,
              display: 'grid',
              gap: 0.75,
              backgroundColor: 'background.paper',
              boxShadow: '0 4px 12px rgba(15, 23, 42, 0.06)',
            }}
          >
            <Stack spacing={1.25}>
              <Stack direction="row" spacing={1.5} alignItems="center">
                <Typography variant="h6" sx={{ fontWeight: 800 }}>
                  {item.title}
                </Typography>
              </Stack>

              <Typography variant="body2" color="text.secondary">
                {item.description}
              </Typography>

              <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                {item.stackStatuses.map((stack) => (
                  <Chip
                    key={`${item.title}-${stack.label}`}
                    label={`${stack.label}: ${STATUS_META[stack.status].label}`}
                    icon={STATUS_META[stack.status].icon}
                    size="small"
                    sx={{
                      backgroundColor: 'background.default',
                      border: '1px solid',
                      borderColor: 'divider',
                      fontWeight: 700,
                    }}
                  />
                ))}
              </Stack>

              {item.id === 'expertise-engine' ? (
                <Box
                  sx={{
                    mt: 1,
                    p: 1.5,
                    borderRadius: 2,
                    backgroundImage: AI_ACCENT_GRADIENT,
                    color: 'common.white',
                    fontWeight: 700,
                  }}
                >
                  Verified Answers + Interest Graph matching are queued for Phase 3.
                </Box>
              ) : null}
            </Stack>
          </Box>
        ))}
      </Stack>
    </Container>
  );
}
