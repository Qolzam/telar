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

  const comments = (data?.pages ?? []).flatMap((page) => page.comments || []);
  
  const totalCommentCount = React.useMemo(() => {
    if (comments.length === 0) return 0;
    
    const rootComments = comments.filter((c: Comment) => !c.parentCommentId);
    
    let totalReplies = 0;
    rootComments.forEach((comment: Comment) => {
      totalReplies += comment.replyCount || 0;
    });
    
    return rootComments.length + totalReplies;
  }, [comments]);

  React.useEffect(() => {
    if (onCommentCountChange && !isLoading && comments.length > 0) {
      onCommentCountChange(totalCommentCount);
    }
  }, [totalCommentCount, isLoading, onCommentCountChange, comments.length]);

  const rootComments = React.useMemo(
    () => comments.filter((c: Comment) => !c.parentCommentId),
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
              onDelete={(commentToDelete: Comment) =>
                deleteMutation.mutate(commentToDelete.objectId, {
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


