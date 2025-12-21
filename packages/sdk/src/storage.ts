/**
 * Storage API Client
 * 
 * Handles file uploads with client-side compression to reduce file sizes
 * before upload, saving server CPU and R2 storage costs.
 */

import { ApiClient } from './client';
import { ENDPOINTS } from './config';

/**
 * Upload request payload
 */
export interface UploadRequest {
  name: string;
  contentType: string;
  size: number; // Size in bytes (after compression)
}

/**
 * Upload response with presigned URL
 */
export interface UploadResponse {
  uploadUrl: string;
  fileId: string;
  key: string;
}

/**
 * Confirm upload request
 */
export interface ConfirmUploadRequest {
  fileId: string;
}

/**
 * File URL response
 */
export interface FileURLResponse {
  url: string;
}

/**
 * Storage API interface
 */
export interface IStorageApi {
  /**
   * Initialize an upload (returns presigned URL)
   * Client should compress the file BEFORE calling this
   */
  initializeUpload(request: UploadRequest): Promise<UploadResponse>;

  /**
   * Confirm upload completion (marks file as uploaded)
   */
  confirmUpload(request: ConfirmUploadRequest): Promise<{ message: string }>;

  /**
   * Get file URL (CDN URL if configured, otherwise presigned URL)
   */
  getFileURL(fileId: string): Promise<FileURLResponse>;

  /**
   * Delete a file
   */
  deleteFile(fileId: string): Promise<void>;
}

/**
 * Storage API implementation
 */
export const storageApi = (client: ApiClient): IStorageApi => ({
  initializeUpload: async (request: UploadRequest): Promise<UploadResponse> => {
    return client.post<UploadResponse>(ENDPOINTS.STORAGE.INIT, request);
  },

  confirmUpload: async (request: ConfirmUploadRequest): Promise<{ message: string }> => {
    return client.post<{ message: string }>(ENDPOINTS.STORAGE.CONFIRM, request);
  },

  getFileURL: async (fileId: string): Promise<FileURLResponse> => {
    return client.get<FileURLResponse>(ENDPOINTS.STORAGE.GET_URL(fileId));
  },

  deleteFile: async (fileId: string): Promise<void> => {
    await client.delete(ENDPOINTS.STORAGE.DELETE(fileId));
  },
});



