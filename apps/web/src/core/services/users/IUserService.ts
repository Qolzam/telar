import { User } from '@/core/domain/users/user';

export interface FetchUsersResponse {
    users: User[];
    nextPage?: number; // TODO: Need server implementation - Next page number if more pages exist
}

export interface UserSearchResponse {
    users: User[];
    hasMore: boolean;
    nextPage?: number;
}

export interface UserSuggestionsResponse {
    users: User[];
    hasMore: boolean;
    nextPage?: number;
}

export interface UserPostsResponse {
    postIds: string[];
    hasMore: boolean;
    lastPostId?: string;
}

export interface UserAlbumsResponse {
    albumIds: string[];
    hasMore: boolean;
    lastAlbumId?: string;
}

export interface UserStatistics {
    voteCount: number;
    shareCount: number;
    followCount: number;
    followerCount: number;
    postCount: number;
}

export interface UpdateUserProfileRequest {
    avatar?: string;
    fullName?: string;
    socialName?: string;
    banner?: string;
    tagLine?: string;
    email?: string;
    birthday?: number;
    webUrl?: string;
    companyName?: string;
    country?: string;
    school?: string;
    address?: string;
    location?: string;
    phone?: number;
    twitterId?: string;
    facebookId?: string;
    instagramId?: string;
    linkedInId?: string;
    accessUserList?: string[];
    permission?: string;
}

export interface UpdateUserSocialRequest {
    twitterId?: string;
    facebookId?: string;
    instagramId?: string;
    linkedInId?: string;
}

/**
 * User service interface
 *
 * @export
 * @interface IUserService
 */
export interface IUserService {
    // Profile Management
    getUserProfile: (userId: string) => Promise<User>;
    getProfileBySocialName: (socialName: string) => Promise<User>;
    getCurrentUserProfile: () => Promise<User>;
    updateUserProfile: (userId: string, profile: UpdateUserProfileRequest) => Promise<User>;
    updateUserSocialProfile: (userId: string, socialProfile: UpdateUserSocialRequest) => Promise<User>;
    
    // Profile Data
    getUserPosts: (userId: string, page?: number, limit?: number, lastPostId?: string) => Promise<UserPostsResponse>;
    getUserAlbums: (userId: string, page?: number, limit?: number, lastAlbumId?: string) => Promise<UserAlbumsResponse>;
    
    // User Discovery
    getUsersProfile: (
        userId: string,
        lastUserId?: string,
        page?: number,
        limit?: number,
    ) => Promise<{ users: { [userId: string]: User }[]; newLastUserId: string }>;
    
    // Search and Suggestions
    searchUser: (query: string, page: number, limit: number, nin: string[]) => Promise<UserSearchResponse>;
    getUserSuggestions: (page?: number, limit?: number, lastUserId?: string) => Promise<UserSuggestionsResponse>;
    findPeople: (page?: number, limit?: number, lastUserId?: string) => Promise<UserSuggestionsResponse>;
    
    // Bulk Operations
    fetchProfiles: (userIds: string[]) => Promise<User[]>;
    
    getUserStatistics: (userId: string) => Promise<UserStatistics>;
    updateUserStatistics: (userId: string, stats: Partial<UserStatistics>) => Promise<void>;
    
    // User Activity
    updateLastSeen: (userId: string) => Promise<void>;
    
    // Search Utilities
    getSearchKey: () => Promise<string>;
    
    // User Validation
    validateSocialName: (socialName: string) => Promise<boolean>;
    checkUserExists: (userId: string) => Promise<boolean>;
}
