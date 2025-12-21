'use client';

import { Box, Typography, CircularProgress, Alert } from '@mui/material';
import { useInView } from 'react-intersection-observer';
import { useEffect } from 'react';
import { useInfiniteBookmarksQuery } from '@/features/bookmarks';
import { PostCard, PostCardSkeleton } from '@/features/posts/components/PostCard';
import type { PostsResponse } from '@telar/sdk';

export default function BookmarksPage() {
  const { data, isLoading, isError, error, fetchNextPage, hasNextPage, isFetchingNextPage } = 
    useInfiniteBookmarksQuery({ limit: 10 });
  
  const { ref, inView } = useInView();

  // Auto-fetch next page when bottom sentinel is visible
  useEffect(() => {
    if (inView && hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [inView, hasNextPage, isFetchingNextPage, fetchNextPage]);

  if (isLoading) {
    return (
      <Box>
        {[...Array(3)].map((_, i) => (
          <PostCardSkeleton key={i} />
        ))}
      </Box>
    );
  }

  if (isError) {
    return (
      <Alert severity="error">
        Failed to load bookmarks: {error?.message || 'Unknown error'}
      </Alert>
    );
  }

  // Flatten all pages and deduplicate by objectId
  const allPosts = data?.pages.flatMap((page: PostsResponse) => page.posts) || [];
  
  const postsMap = new Map<string, (typeof allPosts)[0]>();
  allPosts.forEach((post) => {
    if (post.objectId && !postsMap.has(post.objectId)) {
      postsMap.set(post.objectId, post);
    }
  });
  
  const posts = Array.from(postsMap.values());

  if (posts.length === 0) {
    return (
      <Box sx={{ textAlign: 'center', py: 8 }}>
        <Typography variant="h6" color="text.secondary" sx={{ mb: 2 }}>
          You haven&apos;t saved any posts yet.
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Bookmark posts you want to read later by clicking the bookmark icon on any post.
        </Typography>
      </Box>
    );
  }

  return (
    <Box>
      {posts.map((post) => (
        <PostCard key={post.objectId} post={post} />
      ))}
      
      {/* Infinite scroll sentinel */}
      <Box ref={ref} sx={{ py: 2, textAlign: 'center' }}>
        {isFetchingNextPage && <CircularProgress size={24} />}
        {!hasNextPage && posts.length > 0 && (
          <Typography variant="caption" color="text.secondary">
            You&apos;ve reached the end
          </Typography>
        )}
      </Box>
    </Box>
  );
}

