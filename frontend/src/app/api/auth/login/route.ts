import { cookies } from "next/headers";
import { NextResponse } from "next/server";

export async function POST(req: Request) {
  const body = (await req.json()) as { username?: string; password?: string };
  const u = process.env.OPEN_CONTEXT_ADMIN_USERNAME ?? "admin";
  const p = process.env.OPEN_CONTEXT_ADMIN_PASSWORD ?? "admin";
  if (body.username !== u || body.password !== p) {
    return NextResponse.json({ message: "unauthorized" }, { status: 401 });
  }
  const jar = await cookies();
  jar.set("oc_admin", "1", { httpOnly: true, sameSite: "lax", path: "/" });
  return NextResponse.json({ ok: true });
}
