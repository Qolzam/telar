'use client';

import { useState } from 'react';
import {
  Box,
  TextField,
  Button,
  Avatar,
  Paper,
  Stack,
  IconButton,
  LinearProgress,
  Snackbar,
  Alert,
  Typography,
  CircularProgress,
} from '@mui/material';
import ImageIcon from '@mui/icons-material/Image';
import CloseIcon from '@mui/icons-material/Close';
import { useCreatePostMutation } from '../../client';
import { useSession } from '@/features/auth/client';
import { useImageUpload } from '@/features/storage/hooks/useImageUpload';

export function PostForm() {
  const [body, setBody] = useState('');
  const { user } = useSession();
  const createPost = useCreatePostMutation();

  const {
    imageUrl,
    previewUrl,
    uploadProgress,
    uploadError,
    isCompressing,
    isUploading,
    isDragActive,
    getRootProps,
    getInputProps,
    fileInputRef,
    handleImageClick,
    handleImageChange,
    handleRemoveImage,
  } = useImageUpload({
    disabled: createPost.isPending,
  });

  const handleSubmit = async () => {
    if (!body.trim() && !imageUrl) return;
    
    try {
      await createPost.mutateAsync({
        postTypeId: 1,
        body: body.trim(),
        permission: 'Public',
        ...(imageUrl ? { imageFullPath: imageUrl } : {}),
      });
      setBody(''); // Clear on success
      handleRemoveImage(); // Clear image
    } catch (error) {
      console.error('Failed to create post:', error);
    }
  };

  return (
    <Paper sx={{ p: 2, mb: 2 }} {...getRootProps()}>
      <Stack direction="row" spacing={2}>
        <Avatar src={user?.avatar} alt={user?.displayName}>
          {user?.displayName?.[0]?.toUpperCase()}
        </Avatar>
        <Box sx={{ flex: 1, position: 'relative' }}>
          {isDragActive && (
            <Box
              sx={{
                position: 'absolute',
                top: 0,
                left: 0,
                right: 0,
                bottom: 0,
                bgcolor: 'action.hover',
                border: '2px dashed',
                borderColor: 'primary.main',
                borderRadius: 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex: 10,
              }}
            >
              <Typography variant="h6" color="primary">
                Drop to upload image
              </Typography>
            </Box>
          )}
          <TextField
            fullWidth
            multiline
            rows={3}
            placeholder="What's on your mind?"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            disabled={createPost.isPending || isUploading}
            variant="outlined"
          />
          {(previewUrl || imageUrl) && (
            <Box sx={{ mt: 1, position: 'relative' }}>
              <Box
                component="img"
                src={previewUrl || imageUrl || ''}
                alt="Uploaded"
                sx={{
                  width: '100%',
                  maxHeight: 300,
                  objectFit: 'contain',
                  borderRadius: 1,
                  border: '1px solid',
                  borderColor: 'divider',
                  filter: isCompressing ? 'blur(2px)' : 'none',
                  transition: 'filter 0.2s',
                }}
              />
              {isCompressing && (
                <Box
                  sx={{
                    position: 'absolute',
                    top: '50%',
                    left: '50%',
                    transform: 'translate(-50%, -50%)',
                    bgcolor: 'rgba(0, 0, 0, 0.7)',
                    color: 'white',
                    px: 2,
                    py: 1,
                    borderRadius: 1,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1,
                  }}
                >
                  <CircularProgress size={16} sx={{ color: 'white' }} />
                  <Typography variant="body2">✨ Optimizing for Web...</Typography>
                </Box>
              )}
              <IconButton
                size="small"
                onClick={handleRemoveImage}
                disabled={isUploading}
                sx={{
                  position: 'absolute',
                  top: 8,
                  right: 8,
                  bgcolor: 'background.paper',
                  '&:hover': { bgcolor: 'action.hover' },
                }}
              >
                <CloseIcon fontSize="small" />
              </IconButton>
            </Box>
          )}
          {uploadProgress !== null && uploadProgress !== undefined && !isCompressing && (
            <Box sx={{ mt: 1 }}>
              <LinearProgress variant="determinate" value={uploadProgress} />
              <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
                Uploading image... {uploadProgress}%
              </Typography>
            </Box>
          )}
          <Box sx={{ mt: 1, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <IconButton
              onClick={handleImageClick}
              disabled={createPost.isPending || isUploading}
              color="primary"
            >
              <ImageIcon />
            </IconButton>
            <input
              {...getInputProps()}
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              style={{ display: 'none' }}
              onChange={handleImageChange}
            />
            <Button
              variant="contained"
              onClick={handleSubmit}
              disabled={(!body.trim() && !imageUrl) || createPost.isPending || isUploading}
            >
              {createPost.isPending ? 'Posting...' : 'Post'}
            </Button>
          </Box>
        </Box>
      </Stack>
      <Snackbar
        open={!!uploadError}
        autoHideDuration={6000}
        onClose={() => {
          // Error is managed by hook, but we can't directly clear it
          // The hook will clear it on next upload attempt
        }}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity="error">
          {uploadError}
        </Alert>
      </Snackbar>
    </Paper>
  );
}


