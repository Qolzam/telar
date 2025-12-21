'use client';

import { IconButton, Tooltip } from '@mui/material';
import { Bookmark, BookmarkBorder } from '@mui/icons-material';
import { useTheme, alpha } from '@mui/material/styles';
import { useMemo } from 'react';
import type { Post } from '@telar/sdk';
import { useBookmarkMutation } from '@/features/bookmarks';

interface BookmarkButtonProps {
  post: Post;
}

export function BookmarkButton({ post }: BookmarkButtonProps) {
  const theme = useTheme();
  const bookmarkMutation = useBookmarkMutation();
  
  const isBookmarked = post.isBookmarked ?? false;
  
  const isDarkMode = useMemo(() => {
    if (theme.palette.mode === 'dark') return true;
    if (typeof document !== 'undefined') {
      const scheme = document.documentElement.getAttribute('data-mui-color-scheme') 
        || document.documentElement.getAttribute('data-color-scheme');
      return scheme === 'dark';
    }
    return false;
  }, [theme.palette.mode]);
  
  const iconColor = isDarkMode ? '#94a3b8' : theme.palette.text.secondary;
  const iconHoverColor = theme.palette.primary.main;
  const hoverBg = alpha(iconHoverColor, 0.08);

  const handleClick = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await bookmarkMutation.mutateAsync({ postId: post.objectId });
    } catch (error) {
      console.error('Failed to toggle bookmark:', error);
    }
  };

  return (
    <Tooltip title={isBookmarked ? 'Remove bookmark' : 'Save post'}>
      <IconButton
        aria-label={isBookmarked ? 'Remove bookmark' : 'Save post'}
        onClick={handleClick}
        disabled={bookmarkMutation.isPending}
        sx={{
          color: isBookmarked ? iconHoverColor : iconColor,
          padding: 0,
          width: '20px',
          height: '20px',
          minWidth: 'auto',
          '&:hover': {
            color: iconHoverColor,
            backgroundColor: hoverBg,
          },
          transition: 'color 0.2s ease, background-color 0.2s ease',
        }}
      >
        {isBookmarked ? (
          <Bookmark sx={{ fontSize: '18px' }} />
        ) : (
          <BookmarkBorder sx={{ fontSize: '18px' }} />
        )}
      </IconButton>
    </Tooltip>
  );
}

