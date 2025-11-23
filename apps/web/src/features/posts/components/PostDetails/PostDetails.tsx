'use client';

import { Avatar, Box, Card, CardContent, CardHeader, Typography } from '@mui/material';
import { formatDistanceToNow } from 'date-fns';
import { usePostQuery } from '../../client';

interface PostDetailsProps {
  postId: string;
}

export function PostDetails({ postId }: PostDetailsProps) {
  const { data: post, isLoading, isError } = usePostQuery(postId);

  if (isLoading) {
    return (
      <Card sx={{ mb: 3 }}>
        <CardHeader
          avatar={<Box sx={{ width: 40, height: 40, bgcolor: 'action.hover', borderRadius: '50%' }} />}
          title={<Box sx={{ width: '30%', height: 16, bgcolor: 'action.hover', borderRadius: 1 }} />}
          subheader={<Box sx={{ width: '20%', height: 14, bgcolor: 'action.hover', borderRadius: 1, mt: 1 }} />}
        />
        <CardContent>
          <Box sx={{ width: '100%', height: 18, bgcolor: 'action.hover', borderRadius: 1, mb: 1 }} />
          <Box sx={{ width: '90%', height: 18, bgcolor: 'action.hover', borderRadius: 1 }} />
        </CardContent>
      </Card>
    );
  }

  if (isError || !post) {
    return (
      <Typography variant="body2" color="error">
        Failed to load post.
      </Typography>
    );
  }

  const createdAt = new Date(post.createdDate);

  return (
    <Card sx={{ mb: 3 }}>
      <CardHeader
        avatar={<Avatar src={post.ownerAvatar} alt={post.ownerDisplayName}>{post.ownerDisplayName?.[0]?.toUpperCase()}</Avatar>}
        title={post.ownerDisplayName}
        subheader={formatDistanceToNow(createdAt, { addSuffix: true })}
      />
      <CardContent>
        <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap' }}>
          {post.body}
        </Typography>
        {post.imageFullPath && (
          <Box component="img" src={post.imageFullPath} alt="" sx={{ mt: 2, width: '100%', borderRadius: 1 }} />
        )}
      </CardContent>
    </Card>
  );
}






