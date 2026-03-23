import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';

import { getAdminCookieName, verifyAdminSession } from '@/lib/admin-auth';

export async function middleware(req: NextRequest) {
  if (req.nextUrl.pathname === '/' || req.nextUrl.pathname.startsWith('/api/auth')) {
    return NextResponse.next();
  }
  if (req.nextUrl.pathname.startsWith('/api/')) {
    return NextResponse.next();
  }

  const token = req.cookies.get(getAdminCookieName())?.value;
  const session = await verifyAdminSession(token);
  if (!session) {
    const loginUrl = new URL('/', req.url);
    loginUrl.searchParams.set('next', req.nextUrl.pathname);
    const response = NextResponse.redirect(loginUrl);
    response.cookies.delete(getAdminCookieName());
    return response;
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/dashboard/:path*', '/graph/:path*'],
};
