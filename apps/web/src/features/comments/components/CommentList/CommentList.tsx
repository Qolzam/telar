'use client';

import React from 'react';
import { Box, Button, CircularProgress, Snackbar, Typography } from '@mui/material';
import { Comment as CommentItem } from '../Comment';
import { CommentSkeleton } from '../Comment';
import { CreateCommentForm } from '../CreateCommentForm';
import { useInfiniteCommentsQuery, useDeleteCommentMutation } from '../../client';
import type { Comment } from '@telar/sdk';

interface CommentListProps {
  postId: string;
  currentUserId?: string;
  onCommentCountChange?: (count: number) => void;
}

export function CommentList({ postId, currentUserId, onCommentCountChange }: CommentListProps) {
  const {
    data,
    isLoading,
    isError,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useInfiniteCommentsQuery(postId, 10, true); // enabled=true: always fetch when CommentList is rendered
  const deleteMutation = useDeleteCommentMutation(postId);
  const [snackbarOpen, setSnackbarOpen] = React.useState(false);
  const [snackbarMsg, setSnackbarMsg] = React.useState('');

  const comments = (data?.pages ?? []).flat();
  
  // Calculate total comment count (root comments + all replies using replyCount from API)
  // This matches the calculation in PostCard to ensure consistency
  const totalCommentCount = React.useMemo(() => {
    if (comments.length === 0) return 0;
    
    // Count root comments
    const rootComments = comments.filter((c: any) => !c.parentCommentId);
    
    // Calculate total replies using replyCount from API (more accurate than counting loaded replies)
    let totalReplies = 0;
    rootComments.forEach((comment: any) => {
      totalReplies += comment.replyCount || 0;
    });
    
    // Total = root comments + all replies (from replyCount field)
    return rootComments.length + totalReplies;
  }, [comments]);

  // Notify parent of comment count changes (only when comments are actually loaded)
  // This should match the count calculation in PostCard
  React.useEffect(() => {
    if (onCommentCountChange && !isLoading && comments.length > 0) {
      onCommentCountChange(totalCommentCount);
    }
  }, [totalCommentCount, isLoading, onCommentCountChange, comments.length]);

  // Build replies map (one-level nesting supported)
  const repliesByParent = React.useMemo(() => {
    const map = new Map<string, Comment[]>();
    for (const c of comments) {
      const parentId = (c as any).parentCommentId as string | undefined;
      if (parentId) {
        const arr = map.get(parentId) ?? [];
        arr.push(c);
        map.set(parentId, arr);
      }
    }
    return map;
  }, [comments]);

  const rootComments = React.useMemo(
    () => comments.filter((c: any) => !c.parentCommentId),
    [comments],
  );

  return (
    <Box>
      <CreateCommentForm postId={postId} />

      {isLoading && (
        <>
          <CommentSkeleton />
          <CommentSkeleton />
        </>
      )}

      {isError && (
        <Typography variant="body2" color="error" sx={{ mt: 1 }}>
          Failed to load comments.
        </Typography>
      )}

      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0, mt: '20px' }}>
      {rootComments.map((comment) => (
          <Box key={comment.objectId} sx={{ mb: '24px' }}>
        <CommentItem
          comment={comment}
          currentUserId={currentUserId}
          replies={repliesByParent.get(comment.objectId) ?? []}
          onDelete={() =>
            deleteMutation.mutate(comment.objectId, {
              onSuccess: () => {
                setSnackbarMsg('Comment deleted');
                setSnackbarOpen(true);
              },
              onError: () => {
                setSnackbarMsg('Failed to delete comment');
                setSnackbarOpen(true);
              },
            })
          }
        />
          </Box>
      ))}
      </Box>

      {hasNextPage && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
          <Button
            variant="outlined"
            onClick={() => fetchNextPage()}
            disabled={isFetchingNextPage}
            sx={{ borderRadius: 999 }}
            aria-label="Show more comments"
          >
            {isFetchingNextPage ? (
              <CircularProgress size={18} sx={{ mr: 1 }} />
            ) : null}
            {isFetchingNextPage ? 'Loading...' : 'Show more comments'}
          </Button>
        </Box>
      )}
      <Snackbar
        open={snackbarOpen}
        autoHideDuration={3000}
        message={snackbarMsg}
        onClose={() => setSnackbarOpen(false)}
      />
    </Box>
  );
}


