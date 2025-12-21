'use client';

import {
  Avatar,
  Box,
  Button,
  IconButton,
  Menu,
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import FavoriteIcon from '@mui/icons-material/Favorite';
import FavoriteBorderIcon from '@mui/icons-material/FavoriteBorder';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import MessageIcon from '@mui/icons-material/Message';
import { formatDistanceToNow } from 'date-fns';
import type { Comment as CommentModel } from '@telar/sdk';
import { useToggleLikeCommentMutation, useUpdateCommentMutation } from '../../client';
import { useState } from 'react';
import { useTheme } from '@mui/material/styles';
import { ConfirmDeleteDialog } from '../ConfirmDeleteDialog';
import { useSession } from '@/features/auth/client';

interface ReplyCommentProps {
  comment: CommentModel;
  currentUserId?: string;
  onDelete?: (comment: CommentModel) => void;
  activeReplyId?: string | null;
  onReplyClick?: (targetId: string | null) => void;
}

export function ReplyComment({ comment, currentUserId, onDelete, activeReplyId, onReplyClick }: ReplyCommentProps) {
  const theme = useTheme();
  const textPrimary = `var(--mui-palette-text-primary, ${theme.palette.text.primary})`;
  const textSecondary = `var(--mui-palette-text-secondary, ${theme.palette.text.secondary})`;
  const primaryMain = `var(--mui-palette-primary-main, ${theme.palette.primary.main})`;
  const errorMain = `var(--mui-palette-error-main, ${theme.palette.error.main})`;
  const createdAt = new Date(comment.createdDate);
  const canModify = currentUserId && currentUserId === comment.ownerUserId;
  const { user } = useSession();
  const toggleLikeMutation = useToggleLikeCommentMutation(comment.postId);
  const updateMutation = useUpdateCommentMutation(comment.postId);
  
  // Use session user's avatar if this is the current user's comment, otherwise use comment.ownerAvatar
  const displayAvatar = currentUserId === comment.ownerUserId && user?.avatar 
    ? user.avatar 
    : comment.ownerAvatar;
  const displayName = currentUserId === comment.ownerUserId && user?.displayName
    ? user.displayName
    : comment.ownerDisplayName;
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(comment.text);
  const [menuEl, setMenuEl] = useState<null | HTMLElement>(null);
  const menuOpen = Boolean(menuEl);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const isLiked = comment.isLiked || false;

  const handleSave = async () => {
    await updateMutation.mutateAsync({ objectId: comment.objectId, text: draft });
    setIsEditing(false);
  };

  const handleLikeToggle = async () => {

    try {
      await toggleLikeMutation.mutateAsync(comment.objectId);
    } catch (error) {
      console.error('Failed to toggle like:', error);
    }
  };

  return (
    <Box sx={{ display: 'flex', gap: '16px', py: 0 }}>
      <Avatar
        src={displayAvatar}
        alt={displayName}
        sx={{ width: 40, height: 40, flexShrink: 0 }}
      >
        {displayName?.[0]?.toUpperCase()}
      </Avatar>

      <Box sx={{ flexGrow: 1, minWidth: 0 }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: '8px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Typography 
              component="span" 
              sx={{ 
                fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 700,
                lineHeight: '20px',
                letterSpacing: '-0.084px',
                color: textPrimary,
              }}
            >
              {displayName}
          </Typography>
            <Typography 
              component="span" 
              sx={{ 
                fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                fontSize: '12px',
                fontWeight: 500,
                lineHeight: '16px',
                letterSpacing: '-0.06px',
                color: textSecondary,
              }}
            >
            {formatDistanceToNow(createdAt, { addSuffix: true })}
          </Typography>
          </Box>
          {canModify && !isEditing && (
            <>
              <IconButton
                size="small"
                aria-label="More options"
                onClick={(e) => {
                  e.stopPropagation();
                  setMenuEl(e.currentTarget);
                }}
                sx={{ color: 'grey.400', '&:hover': { color: 'grey.600' } }}
              >
                <MoreVertIcon fontSize="small" />
              </IconButton>
              <Menu
                anchorEl={menuEl}
                open={menuOpen}
                onClose={() => setMenuEl(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
              >
                <MenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    setMenuEl(null);
                    setIsEditing(true);
                    setDraft(comment.text);
                  }}
                >
                  Edit
                </MenuItem>
                <MenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    setMenuEl(null);
                    setConfirmOpen(true);
                  }}
                >
                  Delete
                </MenuItem>
              </Menu>
            </>
          )}
        </Box>

        {/* Two-Tier Architecture: Show "Replying to @User" indicator before comment text */}
        {comment.replyToUserId && comment.replyToDisplayName && (
          <Box sx={{ mb: 0.5 }}>
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{
                fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                fontSize: '12px',
                fontWeight: 400,
                lineHeight: '16px',
                letterSpacing: '-0.06px',
              }}
            >
              Replying to{' '}
              <Typography
                component="span"
                variant="caption"
                sx={{
                  fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                  fontSize: '12px',
                  fontWeight: 600,
                  color: 'primary.main',
                  cursor: 'pointer',
                  '&:hover': {
                    textDecoration: 'underline',
                  },
                }}
              >
                @{comment.replyToDisplayName}
              </Typography>
            </Typography>
          </Box>
        )}
        {!isEditing ? (
          <Typography 
            sx={{ 
              whiteSpace: 'pre-wrap', 
              mb: 0, 
              mt: 0,
              fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 400,
              lineHeight: '22.4px',
              color: textSecondary,
            }}
          >
            {comment.text}
          </Typography>
        ) : (
          <Stack spacing={1.5} sx={{ mb: 1.5 }}>
            <TextField
              multiline
              minRows={2}
              fullWidth
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              aria-label="Edit comment"
            />
            <Stack direction="row" spacing={1}>
              <Button
                variant="contained"
                size="small"
                onClick={handleSave}
                disabled={updateMutation.isPending || draft.trim().length === 0}
              >
                {updateMutation.isPending ? 'Saving…' : 'Save'}
              </Button>
              <Button
                variant="text"
                size="small"
                onClick={() => {
                  setIsEditing(false);
                  setDraft(comment.text);
                }}
              >
                Cancel
              </Button>
            </Stack>
          </Stack>
        )}

        <Box sx={{ display: 'flex', alignItems: 'center', gap: '16px', mt: '8px' }}>
          <Button
            onClick={handleLikeToggle}
            disabled={toggleLikeMutation.isPending}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              color: isLiked ? errorMain : textSecondary,
              textTransform: 'none',
              minWidth: 'auto',
              padding: '0',
              fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 600,
              lineHeight: '20px',
              letterSpacing: '-0.084px',
              '&:hover': {
                backgroundColor: 'transparent',
                color: isLiked ? errorMain : textPrimary,
              },
            }}
          >
            {isLiked ? (
              <FavoriteIcon sx={{ fontSize: 18, color: errorMain }} />
            ) : (
              <FavoriteBorderIcon sx={{ fontSize: 18, color: textSecondary }} />
            )}
            {comment.score > 0 && (
              <Typography 
                component="span" 
                sx={{ 
                  fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                  fontSize: '14px',
                  fontWeight: 600,
                  lineHeight: '20px',
                  letterSpacing: '-0.084px',
                  color: textPrimary,
                }}
              >
                {comment.score}
              </Typography>
            )}
          </Button>

          <Button
            onClick={() => onReplyClick?.(comment.objectId)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              color: textPrimary,
              textTransform: 'none',
              minWidth: 'auto',
              padding: '0',
              fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontSize: '13px',
              fontWeight: 600,
              lineHeight: '20px',
              letterSpacing: '-0.084px',
              '&:hover': {
                backgroundColor: 'transparent',
                color: textPrimary,
              },
            }}
          >
            <MessageIcon sx={{ fontSize: 18, color: textSecondary }} />
            <Typography component="span" sx={{ fontSize: '13px', fontWeight: 600 }}>
              Reply
            </Typography>
          </Button>
        </Box>
        <ConfirmDeleteDialog
          open={confirmOpen}
          onCancel={() => setConfirmOpen(false)}
          onConfirm={() => {
            setConfirmOpen(false);
            onDelete?.(comment);
          }}
        />
        {/* CRITICAL: ReplyComment NEVER renders replies. This enforces Two-Tier architecture. */}
      </Box>
    </Box>
  );
}

