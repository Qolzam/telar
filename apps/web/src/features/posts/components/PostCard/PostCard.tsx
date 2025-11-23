'use client';

import { useState } from 'react';
import React from 'react';
import { Avatar, Card, CardHeader, CardContent, CardActions, IconButton, Typography, Box, Button } from '@mui/material';
import { ThumbUp, ChatBubbleOutlineTwoTone, Share, BookmarkBorder, MoreVert } from '@mui/icons-material';
import type { Post } from '@telar/sdk';
import { formatDistanceToNow } from 'date-fns';
import { CommentList } from '@/features/comments';
import { useSession } from '@/features/auth/client';
import { useQueryClient } from '@tanstack/react-query';
import { postsKeys } from '../../client';

interface PostCardProps {
  post: Post;
}

export function PostCard({ post }: PostCardProps) {
  // createdDate is in milliseconds (from UTCNowUnix() which returns UnixNano / 1,000,000)
  const formattedDate = formatDistanceToNow(new Date(post.createdDate), { addSuffix: true });
  const [isExpanded, setIsExpanded] = useState(false);
  const queryClient = useQueryClient();
  const { user } = useSession();
  
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
        border: '1px solid #E2E8F0',
        boxShadow: '0 2px 4px -2px rgba(23, 23, 23, 0.06), 0 4px 8px -2px rgba(23, 23, 23, 0.10)',
        overflow: 'hidden'
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
              color: '#1E293B'
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
              color: '#475569',
              mt: 0.5
            }}
          >
            {formattedDate}
          </Typography>
        }
        action={
          <IconButton 
            aria-label="more options"
            sx={{ 
              color: '#CBD5E1',
              '&:hover': { color: '#94A3B8' }
            }}
          >
            <MoreVert />
          </IconButton>
        }
        sx={{
          px: '20px',
          py: '20px',
          borderBottom: '1px solid #E2E8F0',
          '& .MuiCardHeader-avatar': {
            mr: '12px'
          }
        }}
      />
      <CardContent sx={{ px: '20px', py: '16px' }}>
        <Typography
          sx={{
            fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
            fontSize: '14px',
            fontWeight: 400,
            lineHeight: '22.4px',
            color: '#1E293B',
            whiteSpace: 'pre-wrap'
          }}
        >
          {contentParts.map((part) =>
            part.isHashtag ? (
              <span key={part.key} style={{ color: '#4F46E5' }}>
                {part.text}
              </span>
            ) : (
              <span key={part.key}>{part.text}</span>
            )
          )}
        </Typography>
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
          borderTop: '1px solid #E2E8F0',
          gap: 0,
          alignItems: 'center',
          display: 'flex',
          justifyContent: 'space-between'
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
          {/* Like Section */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <IconButton 
              aria-label="like"
              sx={{ 
                color: '#94A3B8',
                padding: 0,
                width: '20px',
                height: '20px',
                minWidth: 'auto',
                '&:hover': { color: '#1E293B', backgroundColor: 'transparent' }
              }}
            >
              <ThumbUp sx={{ fontSize: '18px' }} />
            </IconButton>
            {post.score > 0 && (
              <Typography
                sx={{
                  fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                  fontSize: '14px',
                  fontWeight: 500,
                  lineHeight: '20px',
                  letterSpacing: '-0.084px',
                  color: '#1E293B'
                }}
              >
                {post.score} {post.score === 1 ? 'Like' : 'Likes'}
              </Typography>
            )}
          </Box>
          
          {/* Comment Section */}
          <Box 
            onClick={() => setIsExpanded((v) => !v)}
            sx={{ 
              display: 'flex', 
              alignItems: 'center', 
              gap: '8px',
              cursor: 'pointer',
              '&:hover': {
                '& .comment-icon': { color: '#1E293B' },
                '& .comment-text': { color: '#1E293B' }
              }
            }}
          >
            <IconButton 
              aria-label="comment" 
              sx={{ 
                color: '#94A3B8',
                padding: 0,
                width: '20px',
                height: '20px',
                minWidth: 'auto',
                pointerEvents: 'none',
                '&:hover': { color: '#1E293B', backgroundColor: 'transparent' }
              }}
              className="comment-icon"
            >
              <ChatBubbleOutlineTwoTone sx={{ fontSize: '18px' }} />
            </IconButton>
            <Typography
              className="comment-text"
              sx={{
                fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 500,
                lineHeight: '20px',
                letterSpacing: '-0.084px',
                color: '#1E293B',
                userSelect: 'none',
                pointerEvents: 'none'
              }}
            >
              {displayCount} {displayCount === 1 ? 'Comment' : 'Comments'}
            </Typography>
          </Box>
          
          {/* Share Section */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <IconButton 
              aria-label="share"
              sx={{ 
                color: '#94A3B8',
                padding: 0,
                width: '20px',
                height: '20px',
                minWidth: 'auto',
                '&:hover': { color: '#1E293B', backgroundColor: 'transparent' }
              }}
            >
              <Share sx={{ fontSize: '18px' }} />
            </IconButton>
            {/* Share count would be shown here if available - currently not tracked */}
          </Box>
        </Box>
        
        <Box sx={{ flexGrow: 1 }} />
        
        {/* Bookmark Section */}
        <IconButton 
          aria-label="bookmark"
          sx={{ 
            color: '#94A3B8',
            padding: 0,
            width: '20px',
            height: '20px',
            minWidth: 'auto',
            '&:hover': { color: '#1E293B', backgroundColor: 'transparent' }
          }}
        >
          <BookmarkBorder sx={{ fontSize: '18px' }} />
        </IconButton>
      </CardActions>
      {isExpanded && (
        <Box sx={{ px: '20px', pb: '20px', borderTop: '1px solid #E2E8F0' }}>
          <CommentList 
            postId={post.objectId} 
            currentUserId={user?.id}
          />
        </Box>
      )}
    </Card>
  );
}

