'use client';

import React from 'react';
import { Box, Button } from '@mui/material';
import type { Comment } from '@telar/sdk';
import { useCommentRepliesQuery } from '../../client';
import { ReplyComment } from '../ReplyComment/ReplyComment';
import { CreateCommentForm } from '../CreateCommentForm';

interface ReplyListProps {
  rootId: string;
  postId: string;
  currentUserId?: string;
  onDelete?: (comment: Comment) => void;
  activeReplyId?: string | null;
  onReplyClick?: (targetId: string | null) => void;
}

/**
 * ReplyList: Renders a flat list of replies for a root comment.
 */
export function ReplyList({ rootId, postId, currentUserId, onDelete, activeReplyId, onReplyClick }: ReplyListProps) {

  const { data: replyPages, fetchNextPage, hasNextPage, isFetching, refetch } =
    useCommentRepliesQuery(rootId, 10);
  
  // Track previous reply count to detect new replies
  const prevReplyCountRef = React.useRef<number>(0);
  
  // Auto-fetch when component mounts (replies are expanded)
  React.useEffect(() => {
    if (replyPages === undefined && !isFetching) {
      refetch();
    }
  }, [replyPages, isFetching, refetch]);
  
  const replies = (replyPages?.pages ?? []).flatMap((page) => page.comments || []);

  // Filter out duplicates by objectId to prevent React key errors
  const uniqueReplies = replies.filter((r, index, self) => 
    index === self.findIndex((c) => c.objectId === r.objectId)
  );
  
  // Scroll to new reply when it's added (detect by count increase)
  React.useEffect(() => {
    if (uniqueReplies.length > prevReplyCountRef.current && prevReplyCountRef.current > 0) {
      const lastReply = uniqueReplies[uniqueReplies.length - 1];
      if (lastReply) {
        setTimeout(() => {
          const element = document.querySelector(`[data-reply-id="${lastReply.objectId}"]`);
          if (element) {
            element.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
          }
        }, 100);
      }
    }
    prevReplyCountRef.current = uniqueReplies.length;
  }, [uniqueReplies.length, uniqueReplies]);

  if (uniqueReplies.length === 0) {
    return null;
  }

  return (
    <Box sx={{ mt: '16px', pl: '20px', borderLeft: '2px solid', borderColor: 'grey.200' }}>
      {uniqueReplies.map((reply) => (
        <Box key={reply.objectId} data-reply-id={reply.objectId} sx={{ mb: '24px' }}>
          <ReplyComment
            comment={reply}
            currentUserId={currentUserId}
            onDelete={onDelete}
            activeReplyId={activeReplyId}
            onReplyClick={onReplyClick}
          />
          {activeReplyId === reply.objectId && (
            <Box sx={{ pl: '56px', mt: '12px', mb: '16px' }}>
              <CreateCommentForm
                postId={postId}
                parentCommentId={reply.objectId}
                replyToDisplayName={reply.ownerDisplayName}
                onSuccess={() => {
                  onReplyClick?.(null);
                }}
                autoFocus
              />
            </Box>
          )}
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
  );
}

