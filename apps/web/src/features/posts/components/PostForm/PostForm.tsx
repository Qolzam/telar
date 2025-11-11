'use client';

import { useState } from 'react';
import { Box, TextField, Button, Avatar, Paper, Stack } from '@mui/material';
import { useCreatePostMutation } from '../../client';
import { useSession } from '@/features/auth/client';

export function PostForm() {
  const [body, setBody] = useState('');
  const { user } = useSession();
  const createPost = useCreatePostMutation();

  const handleSubmit = async () => {
    if (!body.trim()) return;
    
    try {
      await createPost.mutateAsync({
        postTypeId: 1,
        body: body.trim(),
        permission: 'Public',
      });
      setBody(''); // Clear on success
    } catch (error) {
      console.error('Failed to create post:', error);
    }
  };

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Stack direction="row" spacing={2}>
        <Avatar src={user?.avatar} alt={user?.displayName}>
          {user?.displayName?.[0]?.toUpperCase()}
        </Avatar>
        <Box sx={{ flex: 1 }}>
          <TextField
            fullWidth
            multiline
            rows={3}
            placeholder="What's on your mind?"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            disabled={createPost.isPending}
            variant="outlined"
          />
          <Box sx={{ mt: 1, display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              variant="contained"
              onClick={handleSubmit}
              disabled={!body.trim() || createPost.isPending}
            >
              {createPost.isPending ? 'Posting...' : 'Post'}
            </Button>
          </Box>
        </Box>
      </Stack>
    </Paper>
  );
}


