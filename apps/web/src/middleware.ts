/**
 * Protects routes requiring authentication
 * Runs on Edge Runtime for maximum performance
 */

import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { getSessionToken } from '@/lib/auth/cookies';
import { verifyToken, isTokenExpired } from '@/lib/auth/jwt';

const PROTECTED_ROUTES = [
  '/dashboard',
  '/profile',
  '/settings',
  '/posts',
  '/messages',
] as const;

function isProtectedRoute(pathname: string): boolean {
  return PROTECTED_ROUTES.some(route => pathname.startsWith(route));
}

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Skip middleware for API routes, static files, etc.
  if (
    pathname.startsWith('/api/') ||
    pathname.startsWith('/_next/') ||
    pathname.startsWith('/static/') ||
    pathname.match(/\.(ico|png|jpg|jpeg|svg|css|js|woff|woff2|ttf|eot)$/)
  ) {
    return NextResponse.next();
  }

  const needsAuth = isProtectedRoute(pathname);
  
  if (!needsAuth) {
    // Allow access to public routes and unprotected pages
    return NextResponse.next();
  }

  const cookieHeader = request.headers.get('cookie');
  const token = getSessionToken(cookieHeader);

  // No token - redirect to login
  if (!token) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('from', pathname);
    return NextResponse.redirect(loginUrl);
  }

  try {
    const claim = await verifyToken(token);

    if (!claim) {
      // Invalid token - redirect to login
      const loginUrl = new URL('/login', request.url);
      loginUrl.searchParams.set('from', pathname);
      loginUrl.searchParams.set('error', 'invalid_token');
      return NextResponse.redirect(loginUrl);
    }

    if (isTokenExpired(claim)) {
      // Expired token - redirect to login
      const loginUrl = new URL('/login', request.url);
      loginUrl.searchParams.set('from', pathname);
      loginUrl.searchParams.set('error', 'expired_token');
      return NextResponse.redirect(loginUrl);
    }

    // Valid token - allow access
    return NextResponse.next();

  } catch (error) {
    console.error('[Middleware] Token verification error:', error);
    
    // Error verifying - redirect to login
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('from', pathname);
    loginUrl.searchParams.set('error', 'verification_failed');
    return NextResponse.redirect(loginUrl);
  }
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - api routes (handled separately)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    '/((?!api|_next/static|_next/image|favicon.ico).*)',
  ],
};
