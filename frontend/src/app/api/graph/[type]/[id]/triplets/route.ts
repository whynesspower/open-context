import { cookies } from 'next/headers';
import { NextResponse } from 'next/server';

import { getAdminCookieName, verifyAdminSession } from '@/lib/admin-auth';

type Node = { uuid: string; name: string; summary?: string; labels?: string[] };
type Edge = {
  uuid: string;
  name: string;
  fact: string;
  source_node_uuid: string;
  target_node_uuid: string;
};

function baseUrl() {
  return (
    process.env.OPEN_CONTEXT_INTERNAL_API_URL ||
    process.env.OPEN_CONTEXT_API_URL ||
    "http://localhost:8000"
  );
}

function apiKey() {
  return process.env.OPEN_CONTEXT_API_KEY || "";
}

async function zepFetch(path: string, init?: RequestInit) {
  const url = `${baseUrl()}/api/v2/${path}`;
  const headers = new Headers(init?.headers);
  headers.set("Authorization", `Api-Key ${apiKey()}`);
  if (!headers.has("Content-Type") && init?.method && init.method !== "GET") {
    headers.set("Content-Type", "application/json");
  }
  return fetch(url, { ...init, headers });
}

export async function GET(_: Request, ctx: { params: Promise<{ type: string; id: string }> }) {
  const jar = await cookies();
  const session = await verifyAdminSession(jar.get(getAdminCookieName())?.value);
  if (!session) {
    return NextResponse.json({ message: 'unauthorized' }, { status: 401 });
  }
  const { type, id } = await ctx.params;
  const group = encodeURIComponent(id);
  let nodes: Node[] = [];
  let edges: Edge[] = [];
  if (type === "user") {
    const nr = await zepFetch(`graph/node/user/${group}`, { method: "POST", body: JSON.stringify({ limit: 200 }) });
    const er = await zepFetch(`graph/edge/user/${group}`, { method: "POST", body: JSON.stringify({ limit: 200 }) });
    nodes = (await nr.json()) as Node[];
    edges = (await er.json()) as Edge[];
  } else {
    const nr = await zepFetch(`graph/node/graph/${group}`, { method: "POST", body: JSON.stringify({ limit: 200 }) });
    const er = await zepFetch(`graph/edge/graph/${group}`, { method: "POST", body: JSON.stringify({ limit: 200 }) });
    nodes = (await nr.json()) as Node[];
    edges = (await er.json()) as Edge[];
  }
  const byId = new Map(nodes.map((n) => [n.uuid, n]));
  const triplets = edges
    .map((e) => ({
      sourceNode: byId.get(e.source_node_uuid) ?? { uuid: e.source_node_uuid, name: e.source_node_uuid },
      edge: e,
      targetNode: byId.get(e.target_node_uuid) ?? { uuid: e.target_node_uuid, name: e.target_node_uuid },
    }))
    .filter((t) => t.sourceNode && t.edge && t.targetNode);
  return NextResponse.json({ triplets });
}
