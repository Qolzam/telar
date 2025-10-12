import { NextRequest, NextResponse } from 'next/server';
import { verifyToken, isTokenExpired } from '@/lib/auth/jwt';
import { getSessionToken } from '@/lib/auth/cookies';
import type { SessionData } from '@/lib/auth/types';

export async function GET(request: NextRequest) {
  try {
    const cookieHeader = request.headers.get('cookie');
    const token = getSessionToken(cookieHeader);

    if (!token) {
      return NextResponse.json(
        { error: 'Not authenticated', isAuthenticated: false },
        { status: 401 }
      );
    }

    const claim = await verifyToken(token);

    if (!claim) {
      return NextResponse.json(
        { error: 'Invalid token', isAuthenticated: false },
        { status: 401 }
      );
    }

    if (isTokenExpired(claim)) {
      return NextResponse.json(
        { error: 'Token expired', isAuthenticated: false },
        { status: 401 }
      );
    }

    const sessionData: SessionData = {
      user: {
        id: claim.uid,
        displayName: claim.displayName,
        socialName: claim.socialName,
        email: claim.email,
        role: claim.role,
        createdDate: claim.createdDate,
        avatar: claim.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(claim.displayName)}&background=random`,
        banner: claim.banner,
        tagLine: claim.tagLine,
      },
      isAuthenticated: true,
    };

    return NextResponse.json(sessionData, { status: 200 });

  } catch (error) {
    console.error('[Session] Error verifying session:', error);
    return NextResponse.json(
      { error: 'Session verification failed', isAuthenticated: false },
      { status: 500 }
    );
  }
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 204,
    headers: {
      'Access-Control-Allow-Methods': 'GET, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  });
}
