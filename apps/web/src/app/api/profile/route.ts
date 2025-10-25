import { NextRequest, NextResponse } from 'next/server';
import { getAuthHeaders } from '@/lib/auth-helper';

const GO_API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://127.0.0.1:8080';

export async function GET(request: NextRequest) {
  try {
    const searchParams = request.nextUrl.searchParams;
    const queryString = searchParams.toString();
    const url = queryString 
      ? `${GO_API_URL}/profile?${queryString}` 
      : `${GO_API_URL}/profile`;

    const response = await fetch(url, {
      method: 'GET',
      headers: getAuthHeaders(request),
      credentials: 'include',
    });

    if (!response.ok) {
      const error = await response.text();
      return NextResponse.json(
        { error: error || 'Failed to query profiles' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data, { status: 200 });
  } catch (error) {
    console.error('Error querying profiles:', error);
    return NextResponse.json(
      { error: 'Failed to query profiles' },
      { status: 500 }
    );
  }
}

export async function PUT(request: NextRequest) {
  try {
    const body = await request.json();

    const response = await fetch(`${GO_API_URL}/profile`, {
      method: 'PUT',
      headers: getAuthHeaders(request),
      credentials: 'include',
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const error = await response.text();
      return NextResponse.json(
        { error: error || 'Failed to update profile' },
        { status: response.status }
      );
    }

    await response.text();
    
    return NextResponse.json({ success: true, message: 'Profile updated successfully' }, { status: 200 });
  } catch (error) {
    console.error('Error updating profile:', error);
    return NextResponse.json(
      { error: 'Failed to update profile' },
      { status: 500 }
    );
  }
}


