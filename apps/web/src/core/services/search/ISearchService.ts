/**
 * Search service interface for unified search functionality
 */
export interface ISearchService {
    searchPosts(
        query: string,
        page?: number,
        limit?: number,
        filters?: PostSearchFilters,
        sortBy?: PostSearchSort
    ): Promise<PostSearchResponse>;

    searchPostsLegacy(
        query: string,
        followingIds?: string[],
        lastPostId?: string,
        page?: number,
        limit?: number,
        searchKey?: string
    ): Promise<{
        posts: any;
        ids: any;
        newLastPostId: string;
        hasMore: boolean;
    }>;

    searchUsers(
        query: string,
        page?: number,
        limit?: number,
        filters?: UserSearchFilters,
        sortBy?: UserSearchSort
    ): Promise<UserSearchResponse>;

    globalSearch(
        query: string,
        contentTypes?: SearchContentType[],
        page?: number,
        limit?: number,
        filters?: GlobalSearchFilters
    ): Promise<GlobalSearchResponse>;

    getSearchSuggestions(
        query: string,
        contentType?: SearchContentType,
        limit?: number
    ): Promise<SearchSuggestionsResponse>;

    getTrendingSearches(
        contentType?: SearchContentType,
        limit?: number
    ): Promise<TrendingSearchesResponse>;

    getSearchHistory(
        userId: string,
        limit?: number
    ): Promise<SearchHistoryResponse>;

    addToSearchHistory(
        userId: string,
        query: string,
        contentType: SearchContentType,
        resultCount: number
    ): Promise<void>;

    clearSearchHistory(userId: string): Promise<void>;

    saveSearch(
        userId: string,
        name: string,
        query: string,
        contentType: SearchContentType,
        filters?: any
    ): Promise<SavedSearch>;

    getSavedSearches(userId: string): Promise<SavedSearchResponse>;

    deleteSavedSearch(userId: string, searchId: string): Promise<void>;

    recordSearchInteraction(
        userId: string,
        query: string,
        contentType: SearchContentType,
        action: SearchInteractionType,
        itemId?: string
    ): Promise<void>;

    getSearchAnalytics(
        userId: string,
        period?: AnalyticsPeriod
    ): Promise<SearchAnalyticsResponse>;

    getSearchFilters(contentType: SearchContentType): Promise<SearchFilterConfig[]>;

    getSearchSortOptions(contentType: SearchContentType): Promise<SearchSortOption[]>;

    getSearchKey(): Promise<string>;

    buildSearchQuery(
        query: string,
        filters: any,
        contentType: SearchContentType
    ): Promise<string>;

    getPopularSearches(
        contentType?: SearchContentType,
        timeframe?: string,
        limit?: number
    ): Promise<string[]>;

    getRelatedSearches(
        query: string,
        contentType?: SearchContentType,
        limit?: number
    ): Promise<string[]>;

    getSearchCorrections(query: string): Promise<string[]>;

    indexContent(
        contentType: SearchContentType,
        contentId: string,
        content: any
    ): Promise<void>;

    removeFromIndex(
        contentType: SearchContentType,
        contentId: string
    ): Promise<void>;

    getSearchHealth(): Promise<SearchHealthResponse>;

    getSearchStats(): Promise<SearchStatsResponse>;
}


export interface PostSearchFilters {
    authorId?: string;
    dateRange?: DateRange;
    contentType?: 'text' | 'image' | 'video' | 'all';
    hasMedia?: boolean;
    minVotes?: number;
    maxVotes?: number;
    tags?: string[];
    circle?: string;
    isPublic?: boolean;
}

export interface UserSearchFilters {
    location?: string;
    country?: string;
    hasAvatar?: boolean;
    isPublic?: boolean;
    ageRange?: { min?: number; max?: number };
    interests?: string[];
    language?: string;
    lastActiveWithin?: number; // days
    followerCountRange?: { min?: number; max?: number };
    postCountRange?: { min?: number; max?: number };
}

export interface GlobalSearchFilters {
    post?: PostSearchFilters;
    user?: UserSearchFilters;
    dateRange?: DateRange;
    contentTypes?: SearchContentType[];
}

export interface PostSearchSort {
    field: 'relevance' | 'date' | 'votes' | 'comments' | 'views' | 'shares';
    direction: 'asc' | 'desc';
}

export interface UserSearchSort {
    field: 'relevance' | 'name' | 'joinDate' | 'lastActive' | 'followerCount' | 'postCount' | 'mutualFriends';
    direction: 'asc' | 'desc';
}

export interface DateRange {
    start?: string;
    end?: string;
}

export type SearchContentType = 'posts' | 'users' | 'comments' | 'media' | 'all';

export type SearchInteractionType = 'view' | 'click' | 'share' | 'save' | 'follow' | 'like';

export type AnalyticsPeriod = 'day' | 'week' | 'month' | 'year' | 'all';

// Response Types

export interface PostSearchResponse {
    posts: any[];
    hasMore: boolean;
    totalCount: number;
    nextPage?: number;
    lastPostId?: string;
    searchKey?: string;
    suggestions?: string[];
    facets?: SearchFacet[];
}

export interface UserSearchResponse {
    users: any[];
    hasMore: boolean;
    totalCount: number;
    nextPage?: number;
    lastUserId?: string;
    suggestions?: string[];
    facets?: SearchFacet[];
}

export interface GlobalSearchResponse {
    results: {
        posts?: PostSearchResponse;
        users?: UserSearchResponse;
        comments?: any[];
        media?: any[];
    };
    totalCount: number;
    hasMore: boolean;
    suggestions?: string[];
    facets?: SearchFacet[];
}

export interface SearchSuggestionsResponse {
    suggestions: string[];
    completions: SearchCompletion[];
    trending?: string[];
}

export interface TrendingSearchesResponse {
    trending: TrendingSearch[];
    period: string;
    lastUpdated: string;
}

export interface SearchHistoryResponse {
    history: SearchHistoryItem[];
    totalCount: number;
    hasMore: boolean;
}

export interface SearchAnalyticsResponse {
    topQueries: QueryAnalytics[];
    clickThroughRate: number;
    avgResultsPerQuery: number;
    popularFilters: FilterAnalytics[];
    searchVolume: VolumeAnalytics[];
    userSegments: SegmentAnalytics[];
}

export interface SavedSearchResponse {
    id: string;
    name: string;
    query: string;
    contentType: SearchContentType;
    filters?: any;
    createdAt: string;
    lastUsed?: string;
}

export interface SearchHealthResponse {
    status: 'healthy' | 'degraded' | 'down';
    responseTime: number;
    indexSize: number;
    lastIndexUpdate: string;
    errorRate: number;
}

export interface SearchStatsResponse {
    totalSearches: number;
    dailySearches: number;
    topQueries: string[];
    avgResponseTime: number;
    successRate: number;
}

// Supporting Types

export interface SearchFacet {
    field: string;
    values: Array<{
        value: string;
        count: number;
        selected?: boolean;
    }>;
}

export interface SearchCompletion {
    text: string;
    type: SearchContentType;
    weight: number;
    icon?: string;
}

export interface TrendingSearch {
    query: string;
    count: number;
    growth: number;
    contentType: SearchContentType;
}

export interface SearchHistoryItem {
    id: string;
    query: string;
    contentType: SearchContentType;
    timestamp: string;
    resultCount: number;
    filters?: any;
}

export interface SavedSearch {
    id: string;
    name: string;
    query: string;
    contentType: SearchContentType;
    filters?: any;
    createdAt: string;
    lastUsed?: string;
    useCount: number;
}

export interface SearchFilterConfig {
    name: string;
    field: string;
    type: 'text' | 'select' | 'range' | 'date' | 'boolean' | 'multiselect';
    options?: Array<{ label: string; value: any }>;
    min?: number;
    max?: number;
    placeholder?: string;
    description?: string;
}

export interface SearchSortOption {
    label: string;
    field: string;
    direction: 'asc' | 'desc';
    default?: boolean;
}

export interface QueryAnalytics {
    query: string;
    count: number;
    avgPosition: number;
    clickThroughRate: number;
    conversionRate: number;
}

export interface FilterAnalytics {
    filter: string;
    usage: number;
    effectiveness: number;
}

export interface VolumeAnalytics {
    date: string;
    searches: number;
    uniqueUsers: number;
    avgResultsPerSearch: number;
}

export interface SegmentAnalytics {
    segment: string;
    searches: number;
    topQueries: string[];
    avgSessionLength: number;
}

export default ISearchService;