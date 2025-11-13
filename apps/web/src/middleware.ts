/**
 * Protects routes requiring authentication
 * Runs on Edge Runtime for maximum performance
 */

import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { match as matchLocale } from '@formatjs/intl-localematcher';
import Negotiator from 'negotiator';
import { getSessionToken } from '@/lib/auth/cookies';
import { verifyToken, isTokenExpired } from '@/lib/auth/jwt';
import { languages, fallbackLng, cookieName } from '@/lib/i18n/settings';

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

/**
 * Get locale from cookie
 */
function getLocaleFromCookie(request: NextRequest): string | null {
  const cookieLocale = request.cookies.get(cookieName)?.value;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return cookieLocale && languages.includes(cookieLocale as any) ? cookieLocale : null;
}

/**
 * Get locale from Accept-Language header using negotiator
 */
function getLocaleFromHeader(request: NextRequest): string {
  const acceptLanguage = request.headers.get('accept-language') || '';
  const acceptedLanguages = { 'accept-language': acceptLanguage };
  
  try {
    const negotiator = new Negotiator({ headers: acceptedLanguages });
    const preferredLanguages = negotiator.languages([...languages]);
    return matchLocale(preferredLanguages, [...languages], fallbackLng);
  } catch {
    return fallbackLng;
  }
}

/**
 * Determine the locale for this request
 * Priority: Cookie > Accept-Language Header > Fallback
 */
function getLocale(request: NextRequest): string {
  return getLocaleFromCookie(request) || getLocaleFromHeader(request) || fallbackLng;
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

  // Step 1: Detect locale (will be used for setting cookie and later for rendering)
  const locale = getLocale(request);

  // Step 2: Create response (will be modified if locale cookie needs to be set)
  const response = NextResponse.next();

  // Step 3: Set locale cookie if not present or different
  const currentCookie = request.cookies.get(cookieName)?.value;
  if (currentCookie !== locale) {
    response.cookies.set(cookieName, locale, {
      path: '/',
      maxAge: 31536000, // 1 year
      sameSite: 'lax',
      httpOnly: false, // Allow client-side access for language switcher
    });
  }

  // Step 4: Check if route needs authentication
  const needsAuth = isProtectedRoute(pathname);
  
  if (!needsAuth) {
    // Allow access to public routes and unprotected pages
    // Response already has locale cookie set
    return response;
  }

  const cookieHeader = request.headers.get('cookie');
  const token = getSessionToken(cookieHeader);

  // No token - redirect to login
  if (!token) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('from', pathname);
    
    // Create redirect response with locale cookie
    const redirectResponse = NextResponse.redirect(loginUrl);
    
    // Ensure locale cookie is set in redirect response
    if (currentCookie !== locale) {
      redirectResponse.cookies.set(cookieName, locale, {
        path: '/',
        maxAge: 31536000,
        sameSite: 'lax',
        httpOnly: false,
      });
    }
    
    return redirectResponse;
  }

  try {
    const claim = await verifyToken(token);

    if (!claim) {
      // Invalid token - redirect to login
      const loginUrl = new URL('/login', request.url);
      loginUrl.searchParams.set('from', pathname);
      loginUrl.searchParams.set('error', 'invalid_token');
      
      const redirectResponse = NextResponse.redirect(loginUrl);
      if (currentCookie !== locale) {
        redirectResponse.cookies.set(cookieName, locale, {
          path: '/',
          maxAge: 31536000,
          sameSite: 'lax',
          httpOnly: false,
        });
      }
      
      return redirectResponse;
    }

    if (isTokenExpired(claim)) {
      // Expired token - redirect to login
      const loginUrl = new URL('/login', request.url);
      loginUrl.searchParams.set('from', pathname);
      loginUrl.searchParams.set('error', 'expired_token');
      
      const redirectResponse = NextResponse.redirect(loginUrl);
      if (currentCookie !== locale) {
        redirectResponse.cookies.set(cookieName, locale, {
          path: '/',
          maxAge: 31536000,
          sameSite: 'lax',
          httpOnly: false,
        });
      }
      
      return redirectResponse;
    }

    // Valid token - allow access
    // Response already has locale cookie set from earlier
    return response;

  } catch (error) {
    console.error('[Middleware] Token verification error:', error);
    
    // Error verifying - redirect to login
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('from', pathname);
    loginUrl.searchParams.set('error', 'verification_failed');
    
    const redirectResponse = NextResponse.redirect(loginUrl);
    if (currentCookie !== locale) {
      redirectResponse.cookies.set(cookieName, locale, {
        path: '/',
        maxAge: 31536000,
        sameSite: 'lax',
        httpOnly: false,
      });
    }
    
    return redirectResponse;
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
