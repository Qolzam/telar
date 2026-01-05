'use client';

import {
  Card,
  CardContent,
  CardActions,
  Typography,
  Button,
  Box,
  Chip,
  Avatar,
  Stack,
} from '@mui/material';
import { FlaggedPost, ModerationDetails } from '../client';
import { formatDistanceToNow } from 'date-fns';

interface ModerationCardProps {
  post: FlaggedPost;
  onApprove: (postId: string) => void;
  onReject: (postId: string) => void;
  isProcessing?: boolean;
}

type ChipColor = 'default' | 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning';

function getBadgeInfo(details: ModerationDetails | null): {
  icon: string;
  color: ChipColor;
  label: string;
  subtext?: string;
} {
  if (!details) {
    return { icon: '❓', color: 'default', label: 'Unknown' };
  }

  const reason = details.flag_reason || '';
  const modelUsed = details.model_used || '';

  if (reason.includes('keyword') || modelUsed.includes('Keyword')) {
    return {
      icon: '⚡️',
      color: 'warning',
      label: 'Instant Filter',
      subtext: reason,
    };
  }

  if (modelUsed.includes('Cache')) {
    return {
      icon: '💾',
      color: 'info',
      label: 'Cached Result',
      subtext: reason,
    };
  }

  const toxicityScore = details.scores?.['toxicity'] || details.scores?.['confidence'] || 0;
  const isHighRisk = toxicityScore > 0.8;

  return {
    icon: '🤖',
    color: isHighRisk ? 'error' : 'warning',
    label: 'AI Analyzed',
    subtext: `${reason} (${(toxicityScore * 100).toFixed(0)}% confidence)`,
  };
}

export function ModerationCard({ post, onApprove, onReject, isProcessing }: ModerationCardProps) {
  const badgeInfo = getBadgeInfo(post.moderationDetails);

  const handleApprove = () => {
    onApprove(post.id);
  };

  const handleReject = () => {
    onReject(post.id);
  };

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
          <Avatar sx={{ bgcolor: 'primary.main' }}>
            {post.authorName.charAt(0).toUpperCase()}
          </Avatar>
          <Box sx={{ flexGrow: 1 }}>
            <Typography variant="subtitle1" fontWeight="bold">
              {post.authorName}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {formatDistanceToNow(
                new Date(post.createdAt > 1e12 ? post.createdAt : post.createdAt * 1000),
                { addSuffix: true }
              )}
            </Typography>
          </Box>
        </Stack>

        <Typography variant="body1" sx={{ mb: 2, whiteSpace: 'pre-wrap' }}>
          {post.content}
        </Typography>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
          <Chip
            icon={<span>{badgeInfo.icon}</span>}
            label={badgeInfo.label}
            color={badgeInfo.color}
            size="small"
            sx={{ fontWeight: 'medium' }}
          />
          {badgeInfo.subtext && (
            <Typography variant="caption" color="text.secondary">
              {badgeInfo.subtext}
            </Typography>
          )}
        </Box>

        {post.moderationDetails?.scores && (
          <Box sx={{ mt: 1 }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
              Scores:
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mt: 0.5 }}>
              {Object.entries(post.moderationDetails.scores).map(([key, value]) => (
                <Chip
                  key={key}
                  label={`${key}: ${(value * 100).toFixed(0)}%`}
                  size="small"
                  variant="outlined"
                />
              ))}
            </Box>
          </Box>
        )}
      </CardContent>

      <CardActions sx={{ justifyContent: 'flex-end', px: 2, pb: 2 }}>
        <Button
          variant="contained"
          color="success"
          size="small"
          onClick={handleApprove}
          disabled={isProcessing}
        >
          Approve
        </Button>
        <Button
          variant="contained"
          color="error"
          size="small"
          onClick={handleReject}
          disabled={isProcessing}
        >
          Reject
        </Button>
      </CardActions>
    </Card>
  );
}

