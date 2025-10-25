import { NextRequest, NextResponse } from 'next/server';
import { apiRequest, ApiError } from '@/lib/api';

interface ResendRequest {
  verificationId: string;
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json() as ResendRequest;

    if (!body.verificationId) {
      return NextResponse.json(
        { error: 'Verification ID is required' },
        { status: 400 }
      );
    }

    console.log('[Resend] Resending verification email for:', body.verificationId);
    
    await apiRequest('/auth/signup/resend', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: new URLSearchParams({
        verificationId: body.verificationId,
      }).toString(),
    });

    console.log('[Resend] ✅ Verification email resent successfully');
    return NextResponse.json(
      { success: true, message: 'Verification email resent' },
      { status: 200 }
    );

  } catch (error) {
    if (error instanceof ApiError) {
      console.error('[Resend] API error:', error.message);
      return NextResponse.json(
        { error: error.message, code: error.code },
        { status: error.statusCode }
      );
    }

    console.error('[Resend] Unexpected error:', error);
    return NextResponse.json(
      { error: 'Failed to resend verification email' },
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









