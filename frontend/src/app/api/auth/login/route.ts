import { cookies } from 'next/headers';
import { NextResponse } from 'next/server';

import {
  clearFailedLogin,
  createAdminSession,
  getAdminAuthConfigError,
  getAdminCookieName,
  getAdminSessionTtlSeconds,
  getLoginClientKey,
  getRemainingLoginCooldownMs,
  recordFailedLogin,
} from '@/lib/admin-auth';

export async function POST(req: Request) {
  const authConfigError = getAdminAuthConfigError();
  if (authConfigError) {
    return NextResponse.json({ message: authConfigError }, { status: 500 });
  }

  const clientKey = getLoginClientKey(req);
  const cooldownMs = getRemainingLoginCooldownMs(clientKey);
  if (cooldownMs > 0) {
    return NextResponse.json(
      { message: 'too many login attempts, please wait before retrying' },
      {
        status: 429,
        headers: {
          'Retry-After': String(Math.ceil(cooldownMs / 1000)),
        },
      },
    );
  }

  const body = (await req.json()) as { username?: string; password?: string };
  const username = process.env.OPEN_CONTEXT_ADMIN_USERNAME ?? 'admin';
  const password = process.env.OPEN_CONTEXT_ADMIN_PASSWORD ?? 'admin';
  if (body.username !== username || body.password !== password) {
    recordFailedLogin(clientKey);
    return NextResponse.json({ message: 'unauthorized' }, { status: 401 });
  }

  clearFailedLogin(clientKey);

  const jar = await cookies();
  jar.set(getAdminCookieName(), await createAdminSession(username), {
    httpOnly: true,
    sameSite: 'strict',
    secure: process.env.NODE_ENV === 'production',
    path: '/',
    maxAge: getAdminSessionTtlSeconds(),
  });
  return NextResponse.json({ ok: true });
}
