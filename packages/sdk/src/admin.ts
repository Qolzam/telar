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

export interface IAdminApi {
  listMembers(args?: { limit?: number; offset?: number; search?: string; sortBy?: string; sortOrder?: 'asc' | 'desc' }): Promise<MembersListResponse>;
  getMember(userId: string): Promise<AdminMember>;
  updateMemberRole(userId: string, role: string): Promise<void>;
  banMember(userId: string): Promise<void>;
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
});


