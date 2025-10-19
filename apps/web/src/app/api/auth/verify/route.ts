import { NextRequest, NextResponse } from 'next/server';
import { apiRequest, ApiError } from '@/lib/api';
import { createSessionCookie } from '@/lib/auth/cookies';

interface VerifyRequest {
  verificationId: string;
  code: string;
  responseType?: string;
}

interface VerifyResponse {
  accessToken: string;
  tokenType: string;
  expires_in: string;
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json() as VerifyRequest;

    if (!body.verificationId || !body.code) {
      return NextResponse.json(
        { error: 'Verification ID and code are required' },
        { status: 400 }
      );
    }

    console.log('[Verify] Verifying email with code');
    console.log('[Verify] VerificationId:', body.verificationId);
    console.log('[Verify] Code:', body.code);
    
    const formData = new URLSearchParams({
      verificationId: body.verificationId,
      code: body.code,
      responseType: body.responseType || 'spa',
    }).toString();
    
    console.log('[Verify] Request body:', formData);
    
    const verifyResponse = await apiRequest<VerifyResponse>('/auth/signup/verify', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: formData,
    });

    const response = NextResponse.json(
      { success: true, message: 'Email verified successfully' },
      { status: 200 }
    );

    if (verifyResponse.accessToken) {
      response.headers.set('Set-Cookie', createSessionCookie(verifyResponse.accessToken));
    }

    console.log('[Verify] ✅ Email verified');
    return response;

  } catch (error) {
    if (error instanceof ApiError) {
      console.error('[Verify] API error:', error.message);
      return NextResponse.json(
        { error: error.message },
        { status: error.statusCode }
      );
    }

    console.error('[Verify] Unexpected error:', error);
    return NextResponse.json(
      { error: 'Verification failed' },
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
