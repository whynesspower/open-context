"use client";

import { useEffect, useRef, useState } from "react";
import * as d3 from "d3";

type Triplet = {
  sourceNode: { uuid: string; name: string };
  edge: { uuid: string; name: string; fact: string };
  targetNode: { uuid: string; name: string };
};

export default function GraphPage() {
  const [mode, setMode] = useState<"user" | "group">("user");
  const [id, setId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const ref = useRef<SVGSVGElement | null>(null);

  async function load() {
    setError(null);
    if (!id) return;
    const res = await fetch(`/api/graph/${mode}/${encodeURIComponent(id)}/triplets`);
    if (!res.ok) {
      setError("failed to load graph");
      return;
    }
    const data = (await res.json()) as { triplets: Triplet[] };
    render(ref.current, data.triplets);
  }

  useEffect(() => {
    return () => {
      d3.select(ref.current).selectAll("*").remove();
    };
  }, []);

  return (
    <main style={{ padding: 24 }}>
      <h1 style={{ marginTop: 0 }}>graph</h1>
      <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
        <label>
          <input type="radio" checked={mode === "user"} onChange={() => setMode("user")} /> user
        </label>
        <label>
          <input type="radio" checked={mode === "group"} onChange={() => setMode("group")} /> graph
        </label>
        <input
          placeholder="id"
          value={id}
          onChange={(e) => setId(e.target.value)}
          style={{ padding: 8, minWidth: 260, borderRadius: 8, border: "1px solid #30363d", background: "#0d1117", color: "#e6edf3" }}
        />
        <button type="button" onClick={load} style={{ padding: "8px 12px", borderRadius: 8, border: 0, background: "#238636", color: "white" }}>
          load
        </button>
      </div>
      {error ? <p style={{ color: "#f85149" }}>{error}</p> : null}
      <svg ref={ref} width="100%" height="720" style={{ marginTop: 16, border: "1px solid #30363d", borderRadius: 12, background: "#0d1117" }} />
    </main>
  );
}

function render(svgEl: SVGSVGElement | null, triplets: Triplet[]) {
  if (!svgEl) return;
  const svg = d3.select(svgEl);
  svg.selectAll("*").remove();
  const w = svgEl.clientWidth || 900;
  const h = svgEl.clientHeight || 720;

  const nodesMap = new Map<string, { id: string; name: string }>();
  const links: { source: string; target: string; label: string }[] = [];
  for (const t of triplets) {
    nodesMap.set(t.sourceNode.uuid, { id: t.sourceNode.uuid, name: t.sourceNode.name });
    nodesMap.set(t.targetNode.uuid, { id: t.targetNode.uuid, name: t.targetNode.name });
    links.push({ source: t.sourceNode.uuid, target: t.targetNode.uuid, label: t.edge.name || t.edge.fact });
  }
  const nodes = Array.from(nodesMap.values());

  const sim = d3
    .forceSimulation(nodes as any)
    .force(
      "link",
      d3
        .forceLink(links)
        .id((d: any) => d.id)
        .distance(120),
    )
    .force("charge", d3.forceManyBody().strength(-200))
    .force("center", d3.forceCenter(w / 2, h / 2));

  const g = svg.append("g");
  const zoom = d3.zoom<SVGSVGElement, unknown>().on("zoom", (ev) => {
    g.attr("transform", ev.transform.toString());
  });
  svg.call(zoom as any);

  const link = g
    .append("g")
    .attr("stroke", "#8b949e")
    .selectAll("line")
    .data(links)
    .join("line")
    .attr("stroke-width", 1.5);

  const node = g
    .append("g")
    .selectAll("circle")
    .data(nodes)
    .join("circle")
    .attr("r", 10)
    .attr("fill", "#58a6ff")
    .call(
      d3
        .drag<SVGCircleElement, any>()
        .on("start", (ev, d: any) => {
          if (!ev.active) sim.alphaTarget(0.3).restart();
          d.fx = d.x;
          d.fy = d.y;
        })
        .on("drag", (ev, d: any) => {
          d.fx = ev.x;
          d.fy = ev.y;
        })
        .on("end", (ev, d: any) => {
          if (!ev.active) sim.alphaTarget(0);
          d.fx = null;
          d.fy = null;
        }) as any,
    );

  const label = g
    .append("g")
    .selectAll("text")
    .data(nodes)
    .join("text")
    .text((d: any) => d.name)
    .attr("font-size", 12)
    .attr("fill", "#e6edf3")
    .attr("dx", 14)
    .attr("dy", 4);

  sim.on("tick", () => {
    link
      .attr("x1", (d: any) => d.source.x)
      .attr("y1", (d: any) => d.source.y)
      .attr("x2", (d: any) => d.target.x)
      .attr("y2", (d: any) => d.target.y);
    node.attr("cx", (d: any) => d.x).attr("cy", (d: any) => d.y);
    label.attr("x", (d: any) => d.x).attr("y", (d: any) => d.y);
  });
}
