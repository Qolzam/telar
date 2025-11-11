'use client';

import { Avatar, Card, CardHeader, CardContent, CardActions, IconButton, Typography } from '@mui/material';
import { Favorite, ChatBubbleOutline, Share } from '@mui/icons-material';
import type { Post } from '@telar/sdk';
import { formatDistanceToNow } from 'date-fns';

interface PostCardProps {
  post: Post;
}

export function PostCard({ post }: PostCardProps) {
  // createdDate is in milliseconds (from UTCNowUnix() which returns UnixNano / 1,000,000)
  const formattedDate = formatDistanceToNow(new Date(post.createdDate), { addSuffix: true });

  return (
    <Card sx={{ mb: 2 }}>
      <CardHeader
        avatar={<Avatar src={post.ownerAvatar} alt={post.ownerDisplayName}>{post.ownerDisplayName?.[0]?.toUpperCase()}</Avatar>}
        title={post.ownerDisplayName}
        subheader={formattedDate}
      />
      <CardContent>
        <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap' }}>
          {post.body}
        </Typography>
      </CardContent>
      <CardActions disableSpacing>
        <IconButton aria-label="like">
          <Favorite />
        </IconButton>
        <Typography variant="caption" sx={{ mr: 2 }}>
          {post.score}
        </Typography>
        
        <IconButton aria-label="comment">
          <ChatBubbleOutline />
        </IconButton>
        <Typography variant="caption" sx={{ mr: 2 }}>
          {post.commentCounter}
        </Typography>
        
        <IconButton aria-label="share">
          <Share />
        </IconButton>
      </CardActions>
    </Card>
  );
}

