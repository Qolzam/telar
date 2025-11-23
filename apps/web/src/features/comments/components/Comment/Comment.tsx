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
import ThumbUpOutlinedIcon from '@mui/icons-material/ThumbUpOutlined';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import ThumbDownOutlinedIcon from '@mui/icons-material/ThumbDownOutlined';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import MessageIcon from '@mui/icons-material/Message';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import { formatDistanceToNow } from 'date-fns';
import type { Comment as CommentModel } from '@telar/sdk';
import { useLikeCommentMutation, useUpdateCommentMutation, useCommentRepliesQuery } from '../../client';
import { useState, useEffect } from 'react';
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
  const likeMutation = useLikeCommentMutation(comment.postId);
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
  const [isLiked, setIsLiked] = useState(false);
  const [isDisliked, setIsDisliked] = useState(false);
  const [optimisticScore, setOptimisticScore] = useState(comment.score || 0);
  const { data: replyPages, fetchNextPage, hasNextPage, refetch, isFetching } =
    useCommentRepliesQuery(comment.objectId, 10);
  const lazyReplies = (replyPages?.pages ?? []).flat();

  // Update optimistic score when comment score changes
  useEffect(() => {
    setOptimisticScore(comment.score || 0);
  }, [comment.score]);

  const handleSave = async () => {
    await updateMutation.mutateAsync({ objectId: comment.objectId, text: draft });
    setIsEditing(false);
  };

  const handleLikeToggle = async () => {
    // Toggle like state optimistically
    const newLikedState = !isLiked;
    const newDislikedState = false; // Liking removes dislike
    
    setIsLiked(newLikedState);
    setIsDisliked(newDislikedState);
    
    // Calculate score change: if was disliked, need to add 2 (remove dislike + add like)
    // If was liked, remove 1. If was neither, add 1.
    const scoreDelta = isDisliked ? 2 : isLiked ? -1 : 1;
    setOptimisticScore((prev) => prev + scoreDelta);
    
    try {
      const delta = isDisliked ? 2 : isLiked ? -1 : 1;
      await likeMutation.mutateAsync({
        commentId: comment.objectId,
        delta,
      });
    } catch (error) {
      // Revert on error
      setIsLiked(!newLikedState);
      setIsDisliked(isDisliked);
      setOptimisticScore((prev) => prev - scoreDelta);
    }
  };

  const handleDislikeToggle = async () => {
    // Toggle dislike state optimistically
    const newDislikedState = !isDisliked;
    const newLikedState = false; // Disliking removes like
    
    setIsDisliked(newDislikedState);
    setIsLiked(newLikedState);
    
    // Calculate score change: if was liked, need to remove 2 (remove like + add dislike)
    // If was disliked, add 1. If was neither, remove 1.
    const scoreDelta = isLiked ? -2 : isDisliked ? 1 : -1;
    setOptimisticScore((prev) => prev + scoreDelta);
    
    try {
      const delta = isLiked ? -2 : isDisliked ? 1 : -1;
      await likeMutation.mutateAsync({
        commentId: comment.objectId,
        delta,
      });
    } catch (error) {
      // Revert on error
      setIsDisliked(!newDislikedState);
      setIsLiked(isLiked);
      setOptimisticScore((prev) => prev - scoreDelta);
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
            disabled={likeMutation.isPending}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              color: '#475569',
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
                color: '#1E293B',
              },
            }}
          >
            {isLiked ? (
              <ThumbUpIcon sx={{ fontSize: 18, color: '#475569' }} />
            ) : (
              <ThumbUpOutlinedIcon sx={{ fontSize: 18, color: '#475569' }} />
            )}
            {(optimisticScore > 0 || comment.score > 0) && (
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
                {optimisticScore > 0 ? optimisticScore : comment.score}
              </Typography>
            )}
          </Button>

          <Button
            onClick={handleDislikeToggle}
            disabled={likeMutation.isPending}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              color: '#475569',
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
                color: '#1E293B',
              },
            }}
          >
            {isDisliked ? (
              <ThumbDownIcon sx={{ fontSize: 18, color: '#475569' }} />
            ) : (
              <ThumbDownOutlinedIcon sx={{ fontSize: 18, color: '#475569' }} />
            )}
            {optimisticScore < 0 && (
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
                {Math.abs(optimisticScore)}
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


