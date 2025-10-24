/**
 * API Client
 * 
 * Base HTTP client for making requests to Telar APIs.
 * Handles errors, timeouts, and provides type-safe request methods.
 */

import { ApiErrorResponse } from './types';

/**
 * Custom API Error class
 */
export class ApiError extends Error {
  constructor(
    message: string,
    public statusCode: number,
    public code?: string,
    public originalError?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

/**
 * Request options
 */
export interface RequestOptions extends Omit<RequestInit, 'body'> {
  timeout?: number;
}

/**
 * API Client configuration
 */
export interface ApiClientConfig {
  baseUrl?: string;
  timeout?: number;
}

/**
 * Base API Client
 * 
 * Provides methods for making HTTP requests with automatic error handling,
 * timeout management, and type safety.
 */
export class ApiClient {
  private baseUrl: string;
  private timeout: number;

  constructor(config: ApiClientConfig = {}) {
    this.baseUrl = config.baseUrl || '';
    this.timeout = config.timeout || 10000;
  }

  /**
   * Make a GET request
   */
  async get<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'GET',
    });
  }

  /**
   * Make a POST request
   */
  async post<T>(
    endpoint: string,
    data?: unknown,
    options: RequestOptions = {}
  ): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  /**
   * Make a PUT request
   */
  async put<T>(
    endpoint: string,
    data?: unknown,
    options: RequestOptions = {}
  ): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  /**
   * Make a DELETE request
   */
  async delete<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'DELETE',
    });
  }

  /**
   * Core request method
   */
  private async request<T>(
    endpoint: string,
    options: RequestOptions & { method: string; body?: string }
  ): Promise<T> {
    const url = this.baseUrl + endpoint;
    const timeout = options.timeout || this.timeout;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
        credentials: 'include',
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        await this.handleErrorResponse(response);
      }

      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        const data = await response.json();
        return data as T;
      }

      return {} as T;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof ApiError) {
        throw error;
      }

      if (error instanceof Error && error.name === 'AbortError') {
        throw new ApiError('Request timeout', 408, 'TIMEOUT', error);
      }

      throw new ApiError('Network error', 500, 'NETWORK_ERROR', error);
    }
  }

  /**
   * Handle error responses from the API
   */
  private async handleErrorResponse(response: Response): Promise<never> {
    const errorText = await response.text();
    let errorMessage = 'API request failed';
    let errorCode: string | undefined;

    try {
      const errorJson: ApiErrorResponse = JSON.parse(errorText);
      errorMessage = errorJson.message || errorJson.error || errorMessage;
      errorCode = errorJson.code;
    } catch {
      errorMessage = errorText || errorMessage;
    }

    throw new ApiError(errorMessage, response.status, errorCode);
  }
}

