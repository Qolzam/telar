import { NextRequest, NextResponse } from 'next/server';
import { apiRequest, ApiError } from '@/lib/api';
import { createSessionCookie } from '@/lib/auth/cookies';
import type { LoginRequest, GoApiLoginResponse } from '@telar/sdk';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json() as LoginRequest;

    if (!body.username || !body.password) {
      return NextResponse.json(
        { error: 'Username and password are required' },
        { status: 400 }
      );
    }

    if (typeof body.username !== 'string' || typeof body.password !== 'string') {
      return NextResponse.json(
        { error: 'Invalid input format' },
        { status: 400 }
      );
    }

    console.log('[Login] Authenticating user:', body.username);
    
    const loginResponse = await apiRequest<GoApiLoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({
        username: body.username,
        password: body.password,
      }),
    });

    if (!loginResponse.accessToken) {
      console.error('[Login] No accessToken in response');
      return NextResponse.json(
        { error: 'Authentication failed' },
        { status: 500 }
      );
    }

    const response = NextResponse.json(
      { 
        success: true,
        user: {
          id: loginResponse.user.objectId,
          displayName: loginResponse.user.fullName,
          socialName: loginResponse.user.socialName,
          email: loginResponse.user.email,
        }
      },
      { status: 200 }
    );

    response.headers.set('Set-Cookie', createSessionCookie(loginResponse.accessToken));

    console.log('[Login] ✅ Login successful for:', body.username);
    return response;

  } catch (error) {
    if (error instanceof ApiError) {
      console.error('[Login] API error:', error.message, error.statusCode);
      return NextResponse.json(
        { error: error.message },
        { status: error.statusCode }
      );
    }

    console.error('[Login] Unexpected error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
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
