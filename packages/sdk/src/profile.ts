/**
 * Profile SDK Module
 * 
 * Provides profile management functions that call the Next.js BFF routes.
 * All profile operations go through the BFF for authentication.
 */

import { ApiClient } from './client';
import { ENDPOINTS } from './config';
import type {
  UserProfileModel,
  UpdateProfileRequest,
  ProfileQueryFilter,
  ProfilesResponse,
} from './types';

/**
 * Profile API interface
 */
export interface IProfileApi {
  /**
   * Get current user's profile
   */
  getMyProfile(): Promise<UserProfileModel>;

  /**
   * Get profile by user ID
   */
  getProfileById(userId: string): Promise<UserProfileModel>;

  /**
   * Get profile by social name
   */
  getProfileBySocialName(socialName: string): Promise<UserProfileModel>;

  /**
   * Update current user's profile
   */
  updateProfile(data: UpdateProfileRequest): Promise<void>;

  /**
   * Get multiple profiles by IDs
   */
  getProfilesByIds(userIds: string[]): Promise<UserProfileModel[]>;

  /**
   * Query profiles with filters
   */
  queryProfiles(filter?: ProfileQueryFilter): Promise<ProfilesResponse>;

  /**
   * Search profiles for autocomplete
   */
  searchProfiles(query: string): Promise<UserProfileModel[]>;
}

/**
 * Create Profile API instance
 */
export const profileApi = (client: ApiClient): IProfileApi => ({
  getMyProfile: async (): Promise<UserProfileModel> => {
    return client.get<UserProfileModel>(ENDPOINTS.PROFILE.MY);
  },

  getProfileById: async (userId: string): Promise<UserProfileModel> => {
    return client.get<UserProfileModel>(ENDPOINTS.PROFILE.BY_ID(userId));
  },

  getProfileBySocialName: async (socialName: string): Promise<UserProfileModel> => {
    return client.get<UserProfileModel>(ENDPOINTS.PROFILE.BY_SOCIAL_NAME(socialName));
  },

  updateProfile: async (data: UpdateProfileRequest): Promise<void> => {
    await client.put(ENDPOINTS.PROFILE.UPDATE, data);
  },

  getProfilesByIds: async (userIds: string[]): Promise<UserProfileModel[]> => {
    return client.post<UserProfileModel[]>(ENDPOINTS.PROFILE.BY_IDS, { userIds });
  },

  queryProfiles: async (filter?: ProfileQueryFilter): Promise<ProfilesResponse> => {
    const params = new URLSearchParams();
    if (filter?.search) params.append('search', filter.search);
    if (filter?.limit) params.append('limit', filter.limit.toString());
    if (filter?.offset) params.append('offset', filter.offset.toString());
    
    const queryString = params.toString();
    const endpoint = queryString ? `${ENDPOINTS.PROFILE.QUERY}?${queryString}` : ENDPOINTS.PROFILE.QUERY;
    
    return client.get<ProfilesResponse>(endpoint);
  },

  searchProfiles: async (query: string): Promise<UserProfileModel[]> => {
    const params = new URLSearchParams();
    params.append('q', query);
    params.append('limit', '5');
    const endpoint = `${ENDPOINTS.PROFILE.SEARCH}?${params.toString()}`;
    return client.get<UserProfileModel[]>(endpoint);
  },
});



