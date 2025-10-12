import { NextRequest, NextResponse } from 'next/server';
import { deleteSessionCookie } from '@/lib/auth/cookies';

export async function POST(request: NextRequest) {
  try {
    console.log('[Logout] User logging out');

    const response = NextResponse.json(
      { success: true, message: 'Logged out successfully' },
      { status: 200 }
    );

    response.headers.set('Set-Cookie', deleteSessionCookie());

    return response;

  } catch (error) {
    console.error('[Logout] Error during logout:', error);
    return NextResponse.json(
      { error: 'Logout failed' },
      { status: 500 }
    );
  }
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 204,
    headers: {
      'Access-Control-Allow-Methods': 'POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  });
}
