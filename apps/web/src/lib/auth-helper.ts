import { NextRequest } from 'next/server';

/**
 * Extracts JWT token from session cookie and prepares headers for API calls
 * @param request - The NextRequest object containing cookies
 * @returns Headers object with Authorization header if session token exists
 */
export function getAuthHeaders(request: NextRequest): Record<string, string> {
  // Extract JWT token from access_token cookie
  const cookies = request.headers.get('cookie') || '';
  const sessionCookie = cookies
    .split(';')
    .find(cookie => cookie.trim().startsWith('access_token='))
    ?.split('=')[1];

  // Prepare headers
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  // Add Authorization header if session token exists
  if (sessionCookie) {
    headers['Authorization'] = `Bearer ${sessionCookie}`;
  }

  return headers;
}

