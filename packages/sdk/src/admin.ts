import { ApiClient } from './client';

export interface AdminMember {
  objectId: string;
  displayName: string;
  email: string;
  role: string;
  createdDate: number;
  avatar?: string;
}

export interface MembersListResponse {
  members: AdminMember[];
  limit: number;
  offset: number;
}

// Match the Go struct `ModerationScores`
export interface ModerationScores {
  toxic?: number;
  severe_toxic?: number;
  obscene?: number;
  threat?: number;
  insult?: number;
  identity_hate?: number;
  spam?: number;
  misinformation?: number;
  // Backward compatibility: support old keys
  toxicity?: number;
  sexual?: number;
  violence?: number;
}

// Match the Go struct `ModerationDetails`
export interface ModerationDetails {
  is_flagged: boolean;
  flag_reason: string;
  scores: ModerationScores;
  confidence: number;
  suggested_action: "approve" | "review_needed";
  model_used: string; // Critical for the "Forensic Badge"
  timestamp: string;
  error?: string; // For analysis_failed posts
  analysis_status?: string;
  analysis_time_ms?: number; // Backward compatibility
}

// Match the Go struct `FlaggedPost`
export interface FlaggedPost {
  id: string;
  content: string;
  authorId: string;
  authorName: string;
  authorAvatar?: string;
  createdAt: number; // Unix timestamp
  status: "needs_moderation" | "published" | "rejected" | "analysis_failed";
  moderationDetails: ModerationDetails | null;
}

export interface ModerationListResponse {
  items: FlaggedPost[];
  count: number;
}

export interface IAdminApi {
  listMembers(args?: { limit?: number; offset?: number; search?: string; sortBy?: string; sortOrder?: 'asc' | 'desc' }): Promise<MembersListResponse>;
  getMember(userId: string): Promise<AdminMember>;
  updateMemberRole(userId: string, role: string): Promise<void>;
  banMember(userId: string): Promise<void>;
  getModerationQueue(): Promise<ModerationListResponse>;
  approvePost(postId: string): Promise<{ message: string }>;
  rejectPost(postId: string): Promise<{ message: string }>;
}

export const adminApi = (client: ApiClient): IAdminApi => ({
  async listMembers(args?: { limit?: number; offset?: number; search?: string; sortBy?: string; sortOrder?: 'asc' | 'desc' }): Promise<MembersListResponse> {
    const params = new URLSearchParams();
    if (args?.limit != null) params.append('limit', String(args.limit));
    if (args?.offset != null) params.append('offset', String(args.offset));
    if (args?.search) params.append('search', args.search);
    if (args?.sortBy) params.append('sortBy', args.sortBy);
    if (args?.sortOrder) params.append('sortOrder', args.sortOrder);
    const qs = params.toString();
    const url = `/admin/members${qs ? `?${qs}` : ''}`;
    return client.get<MembersListResponse>(url);
  },

  async getMember(userId: string): Promise<AdminMember> {
    return client.get<AdminMember>(`/admin/members/${userId}`);
  },

  async updateMemberRole(userId: string, role: string): Promise<void> {
    await client.put(`/admin/members/${userId}/role`, { role });
  },

  async banMember(userId: string): Promise<void> {
    await client.post(`/admin/members/${userId}/ban`);
  },

  async getModerationQueue(): Promise<ModerationListResponse> {
    return client.get<ModerationListResponse>('/admin/moderation/queue');
  },

  async approvePost(postId: string): Promise<{ message: string }> {
    return client.post<{ message: string }>(`/admin/moderation/${postId}/approve`, {});
  },

  async rejectPost(postId: string): Promise<{ message: string }> {
    return client.post<{ message: string }>(`/admin/moderation/${postId}/reject`, {});
  },
});


