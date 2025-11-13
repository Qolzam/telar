'use client';

import { Box, CircularProgress, Typography, Alert } from '@mui/material';
import { useInView } from 'react-intersection-observer';
import { useEffect } from 'react';
import { useInfinitePostsQuery } from '../../client';
import { PostCard, PostCardSkeleton } from '../PostCard';
import type { PostsResponse } from '@telar/sdk';

export function PostList() {
  const { data, isLoading, isError, error, fetchNextPage, hasNextPage, isFetchingNextPage } = 
    useInfinitePostsQuery({ limit: 10 });
  
  const { ref, inView } = useInView();

  // Log React Query state changes
  useEffect(() => {
    console.log('[POST_LIST] 📊 React Query State:', {
      isLoading,
      isError,
      hasNextPage,
      isFetchingNextPage,
      totalPages: data?.pages.length || 0,
      totalPostsInCache: data?.pages.reduce((sum, page) => sum + page.posts.length, 0) || 0,
      timestamp: new Date().toISOString(),
    });
  }, [isLoading, isError, hasNextPage, isFetchingNextPage, data?.pages.length]);

  // Log when data pages change (new page added)
  useEffect(() => {
    if (data?.pages) {
      console.log('[POST_LIST] 📄 Data Pages Updated:', {
        pageCount: data.pages.length,
        pagesDetails: data.pages.map((page, idx) => ({
          pageIndex: idx,
          postsCount: page.posts.length,
          hasNext: page.hasNext,
          nextCursor: page.nextCursor ? page.nextCursor.substring(0, 20) + '...' : 'undefined',
          firstPostId: page.posts[0]?.objectId || 'none',
          lastPostId: page.posts[page.posts.length - 1]?.objectId || 'none',
        })),
        timestamp: new Date().toISOString(),
      });
    }
  }, [data?.pages]);

  // Auto-fetch next page when bottom sentinel is visible
  useEffect(() => {
    console.log('[POST_LIST] 👁️ Intersection Observer changed:', {
      inView,
      hasNextPage,
      isFetchingNextPage,
      shouldFetch: inView && hasNextPage && !isFetchingNextPage,
      timestamp: new Date().toISOString(),
    });
    
    if (inView && hasNextPage && !isFetchingNextPage) {
      console.log('[POST_LIST] 🚀 Triggering fetchNextPage()');
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
        Failed to load posts: {error?.message || 'Unknown error'}
      </Alert>
    );
  }

  // Flatten all pages and deduplicate by objectId to prevent duplicate keys
  const allPosts = data?.pages.flatMap((page: PostsResponse) => page.posts) || [];
  
  console.log('[POST_LIST] 📦 Data Processing:', {
    totalPages: data?.pages.length || 0,
    allPostsCount: allPosts.length,
    pagesBreakdown: data?.pages.map((page, idx) => ({
      pageIndex: idx,
      postsCount: page.posts.length,
      postIds: page.posts.map(p => p.objectId).slice(0, 3), // First 3 IDs per page
    })) || [],
    timestamp: new Date().toISOString(),
  });
  
  const postsMap = new Map<string, (typeof allPosts)[0]>();
  const duplicateIds: string[] = [];
  
  allPosts.forEach((post) => {
    if (post.objectId) {
      if (postsMap.has(post.objectId)) {
        duplicateIds.push(post.objectId);
      } else {
        postsMap.set(post.objectId, post);
      }
    }
  });
  
  const posts = Array.from(postsMap.values());
  
  console.log('[POST_LIST] 🔄 Deduplication Results:', {
    beforeDedup: allPosts.length,
    afterDedup: posts.length,
    duplicatesRemoved: duplicateIds.length,
    duplicateIds: duplicateIds.slice(0, 10), // First 10 duplicates
    finalPostIds: posts.map(p => p.objectId).slice(0, 10), // First 10 final IDs
    timestamp: new Date().toISOString(),
  });

  if (posts.length === 0) {
    return (
      <Box sx={{ textAlign: 'center', py: 8 }}>
        <Typography variant="h6" color="text.secondary">
          No posts yet. Be the first to post!
        </Typography>
      </Box>
    );
  }

  // Log rendering decision
  console.log('[POST_LIST] 🎨 Rendering:', {
    postsToRender: posts.length,
    hasNextPage,
    isFetchingNextPage,
    showingEndMessage: !hasNextPage && posts.length > 0,
    showingLoader: isFetchingNextPage,
    timestamp: new Date().toISOString(),
  });

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

