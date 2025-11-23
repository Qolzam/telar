import { NextRequest, NextResponse } from 'next/server';
import { getAuthHeaders } from '@/lib/auth-helper';

const GO_API_URL = process.env.INTERNAL_API_URL || 'http://127.0.0.1:8080';

export async function GET(request: NextRequest) {
  try {
    const response = await fetch(`${GO_API_URL}/profile/my`, {
      method: 'GET',
      headers: getAuthHeaders(request),
      credentials: 'include',
    });

    if (!response.ok) {
      const error = await response.text();
      return NextResponse.json(
        { error: error || 'Failed to fetch profile' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data, { status: 200 });
  } catch (error) {
    console.error('Error fetching profile:', error);
    return NextResponse.json(
      { error: 'Failed to fetch profile' },
      { status: 500 }
    );
  }
}


