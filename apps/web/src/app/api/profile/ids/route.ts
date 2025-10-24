import { NextRequest, NextResponse } from 'next/server';
import { getAuthHeaders } from '@/lib/auth-helper';

const GO_API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://127.0.0.1:8080';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    const response = await fetch(`${GO_API_URL}/profile/ids`, {
      method: 'POST',
      headers: getAuthHeaders(request),
      credentials: 'include',
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const error = await response.text();
      return NextResponse.json(
        { error: error || 'Failed to fetch profiles' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data, { status: 200 });
  } catch (error) {
    console.error('Error fetching profiles by IDs:', error);
    return NextResponse.json(
      { error: 'Failed to fetch profiles' },
      { status: 500 }
    );
  }
}


