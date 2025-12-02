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
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import { formatDistanceToNow } from 'date-fns';
import type { Comment as CommentModel } from '@telar/sdk';
import { useToggleLikeCommentMutation, useUpdateCommentMutation, useCommentRepliesQuery } from '../../client';
import { useState } from 'react';
import { CreateCommentForm } from '../CreateCommentForm';
import { ConfirmDeleteDialog } from '../ConfirmDeleteDialog';
import { useSession } from '@/features/auth/client';

interface CommentProps {
  comment: CommentModel;
  currentUserId?: string;
  onEdit?: (comment: CommentModel) => void;
  onDelete?: (comment: CommentModel) => void;
  replies?: CommentModel[];
}

export function Comment({ comment, currentUserId, onEdit, onDelete, replies = [] }: CommentProps) {
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
  const [replyOpen, setReplyOpen] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(comment.text);
  const [menuEl, setMenuEl] = useState<null | HTMLElement>(null);
  const menuOpen = Boolean(menuEl);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [repliesExpanded, setRepliesExpanded] = useState(false);
  const { data: replyPages, fetchNextPage, hasNextPage, refetch, isFetching } =
    useCommentRepliesQuery(comment.objectId, 10);
  // Extract replies from CommentsListResponse pages
  const lazyReplies = (replyPages?.pages ?? []).flatMap((page) => page.comments || []);

  // Use isLiked from API response (enriched by backend)
  const isLiked = comment.isLiked || false;

  const handleSave = async () => {
    await updateMutation.mutateAsync({ objectId: comment.objectId, text: draft });
    setIsEditing(false);
  };

  const handleLikeToggle = async () => {
    try {
      // The mutation handles optimistic updates automatically
      await toggleLikeMutation.mutateAsync(comment.objectId);
    } catch (error) {
      // Error handling is done in the mutation's onError callback
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
                color: '#1E293B',
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
                color: '#475569',
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
                onClick={(e) => setMenuEl(e.currentTarget)}
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
                  onClick={() => {
                    setMenuEl(null);
                    setIsEditing(true);
                    setDraft(comment.text);
                  }}
                >
                  Edit
                </MenuItem>
                <MenuItem
                  onClick={() => {
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
              color: '#475569',
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
              color: isLiked ? '#EF4444' : '#475569',
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
                color: isLiked ? '#DC2626' : '#1E293B',
              },
            }}
          >
            {isLiked ? (
              <FavoriteIcon sx={{ fontSize: 18, color: '#EF4444' }} />
            ) : (
              <FavoriteBorderIcon sx={{ fontSize: 18, color: '#475569' }} />
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
                  color: '#1E293B',
                }}
              >
                {comment.score}
              </Typography>
            )}
          </Button>

          <Button
            onClick={() => setReplyOpen((o) => !o)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              color: '#1E293B',
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
                color: '#1E293B',
              },
            }}
          >
            <MessageIcon sx={{ fontSize: 18, color: '#475569' }} />
            <Typography component="span" sx={{ fontSize: '13px', fontWeight: 600 }}>
              Reply
            </Typography>
          </Button>
        </Box>
        {!!(replies.length || comment.replyCount || lazyReplies.length) && (
            <Button
              onClick={async () => {
                setRepliesExpanded((v) => !v);
                if (!repliesExpanded) {
                  // First expand: if no replies passed down, load first page
                  if (!replies.length && lazyReplies.length === 0) {
                    await refetch();
                  }
                }
              }}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '4px',
              color: '#4F46E5',
              textTransform: 'none',
              px: 0,
              mt: '12px',
              fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 700,
              lineHeight: '20px',
              letterSpacing: '-0.084px',
              '&:hover': {
                backgroundColor: 'transparent',
                color: '#4338CA',
              },
            }}
            aria-label={repliesExpanded ? 'Hide replies' : `See ${replies.length || comment.replyCount || 0} Replies`}
            >
            <KeyboardArrowDownIcon
              sx={{
                fontSize: 16,
                color: '#4F46E5',
                transition: 'transform 0.2s',
                transform: repliesExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
              }}
            />
            {repliesExpanded ? 'Hide Replies' : `See ${replies.length || comment.replyCount || 0} Replies`}
            </Button>
        )}
        {repliesExpanded && (replies.length > 0 || lazyReplies.length > 0) && (
          <Box sx={{ mt: '16px', pl: '56px' }}>
            {(replies.length > 0 ? replies : lazyReplies).map((r) => (
              <Box key={r.objectId} sx={{ mb: '24px' }}>
                <Comment
                  comment={r}
                  currentUserId={currentUserId}
                  onDelete={onDelete}
                  // Do not pass further replies to keep single-level nesting
                  replies={[]}
                />
              </Box>
            ))}
            {hasNextPage && (
              <Button
                size="small"
                variant="text"
                onClick={() => fetchNextPage()}
                disabled={isFetching}
                sx={{ color: (t) => t.palette.primary.main, textTransform: 'none', px: 0, mt: 0.5 }}
              >
                {isFetching ? 'Loading…' : 'Load more replies'}
              </Button>
            )}
          </Box>
        )}
        {replyOpen && (
          <Box sx={{ mt: '12px' }}>
            <CreateCommentForm
              postId={comment.postId}
              parentCommentId={comment.objectId}
              onSuccess={() => setReplyOpen(false)}
            />
          </Box>
        )}
        <ConfirmDeleteDialog
          open={confirmOpen}
          onCancel={() => setConfirmOpen(false)}
          onConfirm={() => {
            setConfirmOpen(false);
            onDelete?.(comment);
          }}
        />
      </Box>
    </Box>
  );
}


