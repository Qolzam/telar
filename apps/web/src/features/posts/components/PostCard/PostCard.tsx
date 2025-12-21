'use client';

import { useMemo, useState } from 'react';
import React from 'react';
import { Avatar, Card, CardHeader, CardContent, CardActions, IconButton, Typography, Box, Button, Stack, Dialog, DialogTitle, DialogContent, DialogActions, TextField } from '@mui/material';
import { useTheme, alpha } from '@mui/material/styles';
import { ChatBubbleOutlineTwoTone, Share } from '@mui/icons-material';
import type { Post } from '@telar/sdk';
import { formatDistanceToNow } from 'date-fns';
import { CommentList } from '@/features/comments';
import { useSession } from '@/features/auth/client';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { postsKeys } from '../../client';
import { VoteButtons } from './VoteButtons';
import { PostMenu } from './PostMenu';
import { ShareDialog } from './ShareDialog';
import { BookmarkButton } from './BookmarkButton';
import { useUpdatePostMutation, useDeletePostMutation } from '../../client';

interface PostCardProps {
  post: Post;
}

export function PostCard({ post }: PostCardProps) {
  const { t } = useTranslation('posts');
  const theme = useTheme();
  const isDarkMode = useMemo(() => {
    if (theme.palette.mode === 'dark') return true;
    if (typeof document !== 'undefined') {
      const scheme = document.documentElement.getAttribute('data-mui-color-scheme') 
        || document.documentElement.getAttribute('data-color-scheme');
      return scheme === 'dark';
    }
    return false;
  }, [theme.palette.mode]);
  const darkCardBackground = '#0f172a';
  const darkBorder = '#1f2937';
  const darkTextPrimary = '#e2e8f0';
  const darkTextSecondary = '#94a3b8';
  const cardBorder = isDarkMode ? darkBorder : theme.palette.divider;
  const cardBackground = isDarkMode ? darkCardBackground : theme.palette.background.paper;
  const textPrimary = isDarkMode ? darkTextPrimary : theme.palette.text.primary;
  const textSecondary = isDarkMode ? darkTextSecondary : theme.palette.text.secondary;
  const iconColor = textSecondary;
  const iconHoverColor = textPrimary;
  const hoverBg = alpha(textPrimary, 0.08);

  // createdDate is in milliseconds (from UTCNowUnix() which returns UnixNano / 1,000,000)
  const formattedDate = formatDistanceToNow(new Date(post.createdDate), { addSuffix: true });
  const [isExpanded, setIsExpanded] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editBody, setEditBody] = useState(post.body);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [shareDialogOpen, setShareDialogOpen] = useState(false);
  const queryClient = useQueryClient();
  const { user } = useSession();
  const updatePost = useUpdatePostMutation();
  const deletePost = useDeletePostMutation();
  
  const isOwner = user?.id === post.ownerUserId;
  
  // Use session user's avatar if this is the current user's post, otherwise use post.ownerAvatar
  const displayAvatar = user?.id === post.ownerUserId && user?.avatar 
    ? user.avatar 
    : post.ownerAvatar;
  const displayName = user?.id === post.ownerUserId && user?.displayName
    ? user.displayName
    : post.ownerDisplayName;
  

  // Use post prop directly as it comes from API with commentCounter field
  const displayCount = post.commentCounter ?? 0;

  // Parse hashtags from post body
  const parseContent = (text: string) => {
    const parts = text.split(/(#[^\s#]+)/g);
    return parts.map((part, index) => {
      if (part.startsWith('#')) {
        return { text: part, isHashtag: true, key: `hashtag-${index}` };
      }
      return { text: part, isHashtag: false, key: `text-${index}` };
    });
  };

  const contentParts = parseContent(post.body || '');

  return (
    <Card 
      sx={{ 
        mb: 2,
        borderRadius: '24px',
        border: `1px solid var(--mui-palette-divider, ${cardBorder})`,
        backgroundColor: `var(--mui-palette-background-paper, ${cardBackground})`,
        boxShadow: isDarkMode 
          ? '0 12px 40px rgba(0, 0, 0, 0.45)'
          : '0 2px 4px -2px rgba(23, 23, 23, 0.06), 0 4px 8px -2px rgba(23, 23, 23, 0.10)',
        overflow: 'hidden',
      }}
    >
      <CardHeader
        avatar={
          <Avatar 
            src={displayAvatar} 
            alt={displayName}
            sx={{ width: 40, height: 40 }}
          >
            {displayName?.[0]?.toUpperCase()}
          </Avatar>
        }
        title={
          <Typography
            sx={{
              fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 700,
              lineHeight: '20px',
              letterSpacing: '-0.084px',
              color: `var(--mui-palette-text-primary, ${textPrimary})`
            }}
          >
            {displayName}
          </Typography>
        }
        subheader={
          <Typography
            sx={{
              fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 400,
              lineHeight: '22.4px',
              color: `var(--mui-palette-text-secondary, ${textSecondary})`,
              mt: 0.5
            }}
          >
            {formattedDate}
          </Typography>
        }
        action={
          isOwner ? (
            <PostMenu
              postId={post.objectId}
              onEdit={() => {
                setIsEditing(true);
                setEditBody(post.body);
              }}
              onDelete={() => setDeleteDialogOpen(true)}
            />
          ) : null
        }
        sx={{
          px: '20px',
          py: '20px',
          borderBottom: `1px solid var(--mui-palette-divider, ${cardBorder})`,
          '& .MuiCardHeader-avatar': {
            mr: '12px'
          }
        }}
      />
      <CardContent sx={{ px: '20px', py: '16px' }}>
        {isEditing ? (
          <Stack spacing={2}>
            <TextField
              multiline
              fullWidth
              rows={4}
              value={editBody}
              onChange={(e) => setEditBody(e.target.value)}
              disabled={updatePost.isPending}
            />
            <Stack direction="row" spacing={1} justifyContent="flex-end">
              <Button
                size="small"
                onClick={() => {
                  setIsEditing(false);
                  setEditBody(post.body);
                }}
                disabled={updatePost.isPending}
              >
                Cancel
              </Button>
              <Button
                size="small"
                variant="contained"
                onClick={async () => {
                  try {
                    await updatePost.mutateAsync({
                      objectId: post.objectId,
                      body: editBody.trim(),
                    });
                    setIsEditing(false);
                  } catch (error) {
                    console.error('Failed to update post:', error);
                  }
                }}
                disabled={!editBody.trim() || updatePost.isPending}
              >
                {updatePost.isPending ? 'Saving...' : 'Save'}
              </Button>
            </Stack>
          </Stack>
        ) : (
          <Typography
            sx={{
              fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 400,
              lineHeight: '22.4px',
              color: `var(--mui-palette-text-primary, ${textPrimary})`,
              whiteSpace: 'pre-wrap'
            }}
          >
            {contentParts.map((part) =>
              part.isHashtag ? (
                <span key={part.key} style={{ color: theme.palette.primary.main }}>
                  {part.text}
                </span>
              ) : (
                <span key={part.key}>{part.text}</span>
              )
            )}
          </Typography>
        )}
        {post.imageFullPath && (
          <Box
            component="img"
            src={post.imageFullPath}
            alt=""
            sx={{
              mt: '16px',
              width: '100%',
              borderRadius: '16px',
              objectFit: 'cover',
              maxHeight: '260px'
            }}
          />
        )}
      </CardContent>
      <CardActions 
        disableSpacing 
        sx={{ 
          px: '20px', 
          py: '20px',
          borderTop: `1px solid var(--mui-palette-divider, ${cardBorder})`,
          gap: 0,
          alignItems: 'center',
          display: 'flex',
          justifyContent: 'space-between'
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
          {/* Vote Section (Up/Down) */}
          <VoteButtons post={post} />
          
          {/* Comment Section */}
          <Box 
            onClick={() => setIsExpanded((v) => !v)}
            sx={{ 
              display: 'flex', 
              alignItems: 'center', 
              gap: '8px',
              cursor: 'pointer',
              '&:hover': {
                '& .comment-icon': { color: iconHoverColor },
                '& .comment-text': { color: iconHoverColor }
              }
            }}
          >
            <IconButton 
              aria-label="comment" 
              sx={{ 
                color: iconColor,
                padding: 0,
                width: '20px',
                height: '20px',
                minWidth: 'auto',
                pointerEvents: 'none',
                '&:hover': { color: iconHoverColor, backgroundColor: hoverBg }
              }}
              className="comment-icon"
            >
              <ChatBubbleOutlineTwoTone sx={{ fontSize: '18px' }} />
            </IconButton>
            {displayCount > 0 && (
              <Typography
                className="comment-text"
                sx={{
                  fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                  fontSize: '14px',
                  fontWeight: 500,
                  lineHeight: '20px',
                  letterSpacing: '-0.084px',
                  color: `var(--mui-palette-text-primary, ${textPrimary})`,
                  userSelect: 'none',
                  pointerEvents: 'none'
                }}
              >
                {displayCount}
              </Typography>
            )}
          </Box>
          
          {/* Share Section */}
          {!post.disableSharing && (
            <IconButton 
              aria-label="share"
              onClick={() => setShareDialogOpen(true)}
              sx={{ 
                color: iconColor,
                padding: 0,
                width: '20px',
                height: '20px',
                minWidth: 'auto',
                '&:hover': { color: iconHoverColor, backgroundColor: hoverBg }
              }}
            >
              <Share sx={{ fontSize: '18px' }} />
            </IconButton>
          )}
          
          {/* Bookmark Section */}
          <BookmarkButton post={post} />
        </Box>
      </CardActions>
      {isExpanded && (
        <Box sx={{ px: '20px', pb: '20px', borderTop: `1px solid var(--mui-palette-divider, ${cardBorder})` }}>
          <CommentList 
            postId={post.objectId} 
            currentUserId={user?.id}
          />
        </Box>
      )}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>Delete Post</DialogTitle>
        <DialogContent>
          Are you sure you want to delete this post? This action cannot be undone.
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button
            onClick={async () => {
              try {
                await deletePost.mutateAsync(post.objectId);
                setDeleteDialogOpen(false);
              } catch (error) {
                console.error('Failed to delete post:', error);
              }
            }}
            color="error"
            disabled={deletePost.isPending}
          >
            {deletePost.isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>

      <ShareDialog
        open={shareDialogOpen}
        onClose={() => setShareDialogOpen(false)}
        post={post}
      />
    </Card>
  );
}

