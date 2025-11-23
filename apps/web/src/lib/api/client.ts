
const getInternalApiUrl = () => {
  const url = process.env.INTERNAL_API_URL || 'http://localhost:8080';
  return url.replace('localhost', '127.0.0.1');
};

export const API_CONFIG = {
  INTERNAL_API_URL: getInternalApiUrl(),
  TIMEOUT: 10000, // 10 seconds
} as const;

/**
 * API Error class
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
 * Make HTTP request to Go API
 * 
 * @param endpoint - API endpoint (e.g., '/auth/login')
 * @param options - Fetch options
 * @returns Response data
 */
export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_CONFIG.INTERNAL_API_URL}${endpoint}`;

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), API_CONFIG.TIMEOUT);

  try {
    const response = await fetch(url, {
      ...options,
      signal: controller.signal,
      headers: {
        // Respect caller-provided Content-Type; only default if none is given
        ...(options.headers && (options.headers as Record<string, string>)['Content-Type']
          ? {}
          : { 'Content-Type': 'application/json' }),
        ...options.headers,
      },
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      const errorText = await response.text();
      let errorMessage = 'API request failed';
      let errorCode: string | undefined;
      
      try {
        const errorJson = JSON.parse(errorText);
        errorMessage = errorJson.message || errorJson.error || errorMessage;
        errorCode = errorJson.code;
      } catch {
        errorMessage = errorText || errorMessage;
      }

      throw new ApiError(errorMessage, response.status, errorCode);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      const data = await response.json();
      return data as T;
    } else {
      return {} as T;
    }
  } catch (error) {
    clearTimeout(timeoutId);

    if (error instanceof ApiError) {
      throw error;
    }

    if (error instanceof Error && error.name === 'AbortError') {
      throw new ApiError('Request timeout', 408, 'TIMEOUT', error);
    }

    throw new ApiError(
      'Network error',
      500,
      'NETWORK_ERROR',
      error
    );
  }
}
