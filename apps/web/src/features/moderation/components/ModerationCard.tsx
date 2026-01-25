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
  Divider,
} from '@mui/material';
import { FlaggedPost } from '@telar/sdk';
import { formatDistanceToNow } from 'date-fns';
import { ForensicBadge } from '@/features/admin/components/ForensicBadge';

interface ModerationCardProps {
  post: FlaggedPost;
  onApprove: (postId: string) => void;
  onReject: (postId: string) => void;
  isProcessing?: boolean;
}

export function ModerationCard({ post, onApprove, onReject, isProcessing }: ModerationCardProps) {
  const handleApprove = () => {
    onApprove(post.id);
  };

  const handleReject = () => {
    onReject(post.id);
  };

  const details = post.moderationDetails;
  const scores = details?.scores;

  // Get the highest score for display
  const getHighestScore = () => {
    if (!scores) return null;
    let maxKey = '';
    let maxValue = 0;
    Object.entries(scores).forEach(([key, value]) => {
      if (typeof value === 'number' && value > maxValue) {
        maxValue = value;
        maxKey = key;
      }
    });
    return maxKey ? { key: maxKey, value: maxValue } : null;
  };

  const highestScore = getHighestScore();

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        {/* Author Info */}
        <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
          <Avatar 
            src={post.authorAvatar} 
            sx={{ bgcolor: 'primary.main' }}
          >
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

        {/* Post Content */}
        <Typography variant="body1" sx={{ mb: 2, whiteSpace: 'pre-wrap' }}>
          {post.content}
        </Typography>

        <Divider sx={{ my: 2 }} />

        {/* Forensic Badge and Flag Reason */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap', mb: 2 }}>
          <ForensicBadge details={details} />
          {details?.flag_reason && (
            <Typography variant="body2" color="text.secondary">
              {details.flag_reason}
            </Typography>
          )}
        </Box>

        {/* Moderation Details */}
        {details && (
          <Box sx={{ mb: 2 }}>
            <Stack direction="row" spacing={2} sx={{ mb: 1 }}>
              {details.confidence !== undefined && (
                <Typography variant="caption" color="text.secondary">
                  Confidence: <strong>{(details.confidence * 100).toFixed(0)}%</strong>
                </Typography>
              )}
              {highestScore && (
                <Typography variant="caption" color="text.secondary">
                  Highest: <strong>{highestScore.key} ({(highestScore.value * 100).toFixed(0)}%)</strong>
                </Typography>
              )}
              {details.analysis_time_ms !== undefined && (
                <Typography variant="caption" color="text.secondary">
                  Analysis: <strong>{details.analysis_time_ms}ms</strong>
                </Typography>
              )}
            </Stack>

            {/* Scores Display */}
            {scores && (
              <Box sx={{ mt: 1 }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                  Detailed Scores:
                </Typography>
                <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                  {Object.entries(scores)
                    .filter(([_, value]) => typeof value === 'number' && value > 0)
                    .sort(([_, a], [__, b]) => (b as number) - (a as number))
                    .slice(0, 6) // Show top 6 scores
                    .map(([key, value]) => (
                      <Chip
                        key={key}
                        label={`${key}: ${((value as number) * 100).toFixed(0)}%`}
                        size="small"
                        variant="outlined"
                        color={value > 0.7 ? 'error' : value > 0.5 ? 'warning' : 'default'}
                      />
                    ))}
                </Box>
              </Box>
            )}
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

