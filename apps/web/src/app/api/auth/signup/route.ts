import { NextRequest, NextResponse } from 'next/server';
import { apiRequest, ApiError } from '@/lib/api';

interface SignupRequest {
  fullName: string;
  email: string;
  newPassword: string;
  responseType?: string;
  verifyType?: string;
}

interface SignupResponse {
  verificationId: string;
  message: string;
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json() as SignupRequest;

    if (!body.fullName || !body.email || !body.newPassword) {
      return NextResponse.json(
        { error: 'Full name, email, and password are required' },
        { status: 400 }
      );
    }

    const signupResponse = await apiRequest<SignupResponse>('/auth/signup', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: new URLSearchParams({
        fullName: body.fullName,
        email: body.email,
        newPassword: body.newPassword,
        responseType: body.responseType || 'spa',
        verifyType: body.verifyType || 'email',
      }).toString(),
    });

    return NextResponse.json(signupResponse, { status: 200 });

  } catch (error) {
    if (error instanceof ApiError) {
      return NextResponse.json(
        { error: error.message },
        { status: error.statusCode }
      );
    }

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

