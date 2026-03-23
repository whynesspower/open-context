import { cookies } from 'next/headers';
import { NextResponse } from 'next/server';

import { getAdminCookieName } from '@/lib/admin-auth';

export async function POST() {
  const jar = await cookies();
  jar.delete(getAdminCookieName());
  return NextResponse.json({ ok: true });
}
