import { Post, PostDTO } from '@/core/domain/posts/post';

export interface FetchPostsResponse {
    posts: PostDTO[];
    nextPage?: number;
    hasMore?: boolean;
    newLastPostId?: string;
    ids?: string[];
}

export interface PostResponse {
    post: PostDTO;
    postId: string;
}

export interface PostSearchParams {
    query?: string;
    filters?: string;
    lastPostId?: string;
    page?: number;
    limit?: number;
    searchKey?: string;
}

export interface PostStreamParams {
    userId: string;
    followingIds?: string[];
    lastPostId?: string;
    page?: number;
    limit?: number;
    searchKey?: string;
}

export interface UserPostsParams {
    userId: string;
    lastPostId?: string;
    page?: number;
    limit?: number;
    searchKey?: string;
}

/**
 * Post service interface - simplified to only include methods actually used
 */
export interface IPostService {
    addPost: (post: Post) => Promise<string>;
    updatePost: (post: Post) => Promise<void>;
    deletePost: (postId: string) => Promise<void>;
    
    getPostById: (postId: string) => Promise<Post>;
    getPostByURLKey: (urlKey: string) => Promise<Post>;
    
    searchPosts: (query: string, filters: string, pageParam?: number, limit?: number) => Promise<FetchPostsResponse>;
    
    getPostsByUserId: (userId: string, page?: number, limit?: number, searchKey?: string) => Promise<FetchPostsResponse>;
    
    getAlbumPosts: (pageParam?: number, limit?: number) => Promise<FetchPostsResponse>;
    
    getSearchKey: () => Promise<string>;
    
    disableComment: (postId: string, status: boolean) => Promise<void>;
    disableSharing: (postId: string, status: boolean) => Promise<void>;
}
