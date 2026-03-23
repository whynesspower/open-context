'use client';

import * as d3 from 'd3';
import {
  CheckCircle2,
  Layers3,
  LoaderCircle,
  Orbit,
  Radar,
  Sparkles,
} from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';

import { AppShell } from '@/components/app-shell';
import { Button } from '@/components/button';
import { Card } from '@/components/card';
import { TextField } from '@/components/text-field';

type Triplet = {
  sourceNode: { uuid: string; name: string };
  edge: { uuid: string; name: string; fact: string };
  targetNode: { uuid: string; name: string };
};

type GraphNode = {
  id: string;
  name: string;
  x?: number;
  y?: number;
  fx?: number | null;
  fy?: number | null;
};

type GraphLink = d3.SimulationLinkDatum<GraphNode> & {
  source: string | GraphNode;
  target: string | GraphNode;
  label: string;
};

const modeOptions = [
  { value: 'user', label: 'User graph' },
  { value: 'group', label: 'Group graph' },
] as const;

export default function GraphPage() {
  const [mode, setMode] = useState<'user' | 'group'>('user');
  const [id, setId] = useState('');
  const [triplets, setTriplets] = useState<Triplet[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [lastLoadedId, setLastLoadedId] = useState<string | null>(null);
  const ref = useRef<SVGSVGElement | null>(null);

  async function load() {
    if (!id.trim()) {
      setError('Enter a valid identifier before loading the graph.');
      return;
    }

    setError(null);
    setIsLoading(true);
    setHasLoaded(true);

    try {
      const response = await fetch(`/api/graph/${mode}/${encodeURIComponent(id.trim())}/triplets`);
      const payload = (await response.json().catch(() => null)) as
        | { message?: string; triplets?: Triplet[] }
        | null;

      if (!response.ok) {
        setTriplets([]);
        setError(
          payload?.message === 'unauthorized'
            ? 'Your session has expired. Sign in again to continue.'
            : payload?.message ?? 'Failed to load graph data.',
        );
        return;
      }

      setTriplets(payload?.triplets ?? []);
      setLastLoadedId(id.trim());
    } finally {
      setIsLoading(false);
    }
  }

  useEffect(() => {
    const cleanup = render(ref.current, triplets);
    return cleanup;
  }, [triplets]);

  const summary = useMemo(() => {
    const nodeIds = new Set<string>();
    const edgeLabels = new Set<string>();

    for (const triplet of triplets) {
      nodeIds.add(triplet.sourceNode.uuid);
      nodeIds.add(triplet.targetNode.uuid);
      edgeLabels.add(triplet.edge.name || triplet.edge.fact);
    }

    return {
      nodeCount: nodeIds.size,
      edgeCount: triplets.length,
      labels: Array.from(edgeLabels).filter(Boolean).slice(0, 4),
    };
  }, [triplets]);

  return (
    <AppShell
      title="Graph explorer"
      description="Load a user or group graph, inspect the relationship network visually, and verify the returned triplets in one place."
      actions={
        <div className="status-chip">
          <CheckCircle2 size={16} />
          Protected graph queries
        </div>
      }
    >
      <section className="graph-layout">
        <div className="stack">
          <Card
            description="Choose a graph type, enter a valid identifier, and load the latest graph snapshot from the backend."
            icon={<Radar size={18} />}
            title="Load graph data"
          >
            <div className="graph-toolbar">
              <div className="toggle-group" aria-label="Graph type" role="radiogroup">
                {modeOptions.map((option) => (
                  <button
                    aria-checked={mode === option.value}
                    className={`mode-toggle ${mode === option.value ? 'mode-toggle-active' : ''}`}
                    key={option.value}
                    onClick={() => setMode(option.value)}
                    role="radio"
                    type="button"
                  >
                    {option.label}
                  </button>
                ))}
              </div>

              <TextField
                hint="Paste a user UUID or graph UUID that exists in your backend."
                label={mode === 'user' ? 'User UUID' : 'Graph UUID'}
                onChange={(event) => setId(event.target.value)}
                placeholder="00000000-0000-4000-8000-000000000001"
                value={id}
              />

              <Button disabled={isLoading} onClick={load}>
                {isLoading ? (
                  <>
                    <LoaderCircle className="spin" size={16} />
                    Loading
                  </>
                ) : (
                  'Load graph'
                )}
              </Button>
            </div>

            {error ? (
              <p className="field-error">{error}</p>
            ) : (
              <p className="muted">
                Requests are proxied through the authenticated frontend route before they reach the
                backend API.
              </p>
            )}
          </Card>

          <Card
            description="Drag nodes, zoom the canvas, and inspect edge labels inline without leaving the page."
            icon={<Orbit size={18} />}
            title="Relationship map"
          >
            <div className="link-row">
              <span className="legend-chip">Blue nodes represent entities</span>
              <span className="legend-chip">Labeled edges represent relationships</span>
              <span className="legend-chip">Drag to reposition, scroll to zoom</span>
            </div>

            <div className="graph-canvas">
              {triplets.length > 0 ? (
                <svg className="graph-svg" ref={ref} />
              ) : (
                <div className="graph-empty">
                  <Layers3 size={28} />
                  <div>
                    <strong>
                      {hasLoaded
                        ? 'No relationships were returned for this identifier.'
                        : 'Load a graph to visualize the relationship network.'}
                    </strong>
                    <p className="muted">
                      {hasLoaded
                        ? 'Try a different id or switch between user and group graph modes.'
                        : 'Once loaded, the graph will render here with labeled edges and draggable nodes.'}
                    </p>
                  </div>
                </div>
              )}
            </div>
          </Card>

          <Card
            description={
              triplets.length
                ? 'The first relationships returned from the latest graph request.'
                : 'Triplets from the backend response will appear here after a successful load.'
            }
            icon={<Sparkles size={18} />}
            title="Returned triplets"
          >
            {triplets.length > 0 ? (
              <div className="triplets-list">
                {triplets.slice(0, 8).map((triplet) => (
                  <div className="triplet-item" key={triplet.edge.uuid}>
                    <div>
                      <span className="kicker">Source node</span>
                      <strong>{triplet.sourceNode.name}</strong>
                    </div>
                    <div className="triplet-arrow mono">
                      {truncateLabel(triplet.edge.name || triplet.edge.fact, 28)}
                    </div>
                    <div>
                      <span className="kicker">Target node</span>
                      <strong>{triplet.targetNode.name}</strong>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="muted">Load a graph to inspect the returned triplets.</p>
            )}
          </Card>
        </div>

        <aside className="stack">
          <Card
            description="A compact snapshot of the currently loaded graph."
            icon={<Layers3 size={18} />}
            title="Current snapshot"
          >
            <div className="stack">
              <div className="dashboard-callout metric-card">
                <span className="metric-label">Loaded identifier</span>
                <span className="mono">{lastLoadedId ?? 'Not loaded yet'}</span>
              </div>
              <div className="dashboard-callout metric-card">
                <span className="metric-label">Nodes</span>
                <span className="metric-value">{summary.nodeCount}</span>
              </div>
              <div className="dashboard-callout metric-card">
                <span className="metric-label">Relationships</span>
                <span className="metric-value">{summary.edgeCount}</span>
              </div>
              <div className="dashboard-callout metric-card">
                <span className="metric-label">Mode</span>
                <span>{mode === 'user' ? 'User graph' : 'Group graph'}</span>
              </div>
            </div>
          </Card>

          <Card
            description="Common labels from the current graph response."
            icon={<Sparkles size={18} />}
            title="Relationship labels"
          >
            {summary.labels.length > 0 ? (
              <div className="link-row">
                {summary.labels.map((label) => (
                  <span className="legend-chip" key={label}>
                    {truncateLabel(label, 32)}
                  </span>
                ))}
              </div>
            ) : (
              <p className="muted">Labels will appear here after the first successful load.</p>
            )}
          </Card>

          <Card
            description="How to interpret the graph once it has rendered."
            icon={<Radar size={18} />}
            title="Reading guide"
          >
            <div className="stack">
              <div className="dashboard-callout">
                <h3>Entity nodes</h3>
                <p className="muted">Each node is a source or target entity returned by the graph API.</p>
              </div>
              <div className="dashboard-callout">
                <h3>Relationship edges</h3>
                <p className="muted">
                  Edge labels are rendered inline so you can read the graph without inspecting raw
                  JSON first.
                </p>
              </div>
              <div className="dashboard-callout">
                <h3>Triplet verification</h3>
                <p className="muted">
                  Use the triplet list below the canvas when you need a cleaner textual summary of
                  the same response.
                </p>
              </div>
            </div>
          </Card>
        </aside>
      </section>
    </AppShell>
  );
}

function truncateLabel(label: string, maxLength: number) {
  if (label.length <= maxLength) {
    return label;
  }

  return `${label.slice(0, maxLength - 1)}…`;
}

function render(svgEl: SVGSVGElement | null, triplets: Triplet[]) {
  if (!svgEl) {
    return;
  }

  const svg = d3.select(svgEl);
  svg.selectAll('*').remove();

  if (!triplets.length) {
    return;
  }

  const width = svgEl.clientWidth || 960;
  const height = svgEl.clientHeight || 688;

  const nodesMap = new Map<string, GraphNode>();
  const links: GraphLink[] = [];

  for (const triplet of triplets) {
    nodesMap.set(triplet.sourceNode.uuid, {
      id: triplet.sourceNode.uuid,
      name: triplet.sourceNode.name,
    });
    nodesMap.set(triplet.targetNode.uuid, {
      id: triplet.targetNode.uuid,
      name: triplet.targetNode.name,
    });
    links.push({
      source: triplet.sourceNode.uuid,
      target: triplet.targetNode.uuid,
      label: truncateLabel(triplet.edge.name || triplet.edge.fact, 28),
    });
  }

  const nodes = Array.from(nodesMap.values());
  const simulation = d3
    .forceSimulation(nodes)
    .force(
      'link',
      d3
        .forceLink<GraphNode, GraphLink>(links)
        .id((node) => node.id)
        .distance(150)
        .strength(0.7),
    )
    .force('charge', d3.forceManyBody().strength(-520))
    .force('collision', d3.forceCollide<GraphNode>().radius(34))
    .force('center', d3.forceCenter(width / 2, height / 2));

  const defs = svg.append('defs');
  defs
    .append('marker')
    .attr('id', 'graph-arrow')
    .attr('viewBox', '0 -5 10 10')
    .attr('refX', 26)
    .attr('refY', 0)
    .attr('markerWidth', 8)
    .attr('markerHeight', 8)
    .attr('orient', 'auto')
    .append('path')
    .attr('fill', 'rgba(157, 181, 218, 0.72)')
    .attr('d', 'M0,-5L10,0L0,5');

  const root = svg.append('g');
  const zoom = d3
    .zoom<SVGSVGElement, unknown>()
    .scaleExtent([0.5, 2.4])
    .on('zoom', (event) => {
      root.attr('transform', event.transform.toString());
    });

  svg.call(zoom as never);
  svg.call(zoom.transform as never, d3.zoomIdentity.translate(width * 0.04, height * 0.04).scale(0.94));

  const link = root
    .append('g')
    .selectAll('line')
    .data(links)
    .join('line')
    .attr('stroke', 'rgba(157, 181, 218, 0.56)')
    .attr('stroke-width', 1.6)
    .attr('marker-end', 'url(#graph-arrow)');

  const edgeLabelHalo = root
    .append('g')
    .selectAll('text')
    .data(links)
    .join('text')
    .text((item) => item.label)
    .attr('fill', 'transparent')
    .attr('font-size', 11)
    .attr('font-family', 'var(--font-mono), monospace')
    .attr('stroke', 'rgba(7, 17, 29, 0.96)')
    .attr('stroke-linejoin', 'round')
    .attr('stroke-width', 6);

  const edgeLabel = root
    .append('g')
    .selectAll('text')
    .data(links)
    .join('text')
    .text((item) => item.label)
    .attr('fill', '#dfe9ff')
    .attr('font-size', 11)
    .attr('font-family', 'var(--font-mono), monospace')
    .attr('text-anchor', 'middle');

  const node = root
    .append('g')
    .selectAll('circle')
    .data(nodes)
    .join('circle')
    .attr('r', 13)
    .attr('fill', '#6fa3ff')
    .attr('stroke', '#d8e4ff')
    .attr('stroke-width', 1.4)
    .call(
      d3
        .drag<SVGCircleElement, GraphNode>()
        .on('start', (event, datum) => {
          if (!event.active) {
            simulation.alphaTarget(0.25).restart();
          }
          datum.fx = datum.x;
          datum.fy = datum.y;
        })
        .on('drag', (event, datum) => {
          datum.fx = event.x;
          datum.fy = event.y;
        })
        .on('end', (event, datum) => {
          if (!event.active) {
            simulation.alphaTarget(0);
          }
          datum.fx = null;
          datum.fy = null;
        }) as never,
    );

  node.append('title').text((datum) => datum.name);

  const nodeLabelHalo = root
    .append('g')
    .selectAll('text')
    .data(nodes)
    .join('text')
    .text((datum) => truncateLabel(datum.name, 28))
    .attr('fill', 'transparent')
    .attr('font-size', 12)
    .attr('stroke', 'rgba(7, 17, 29, 0.98)')
    .attr('stroke-width', 7)
    .attr('stroke-linejoin', 'round');

  const nodeLabel = root
    .append('g')
    .selectAll('text')
    .data(nodes)
    .join('text')
    .text((datum) => truncateLabel(datum.name, 28))
    .attr('fill', '#f5f7ff')
    .attr('font-size', 12)
    .attr('font-weight', 700);

  simulation.on('tick', () => {
    link
      .attr('x1', (datum) => (datum.source as GraphNode).x ?? 0)
      .attr('y1', (datum) => (datum.source as GraphNode).y ?? 0)
      .attr('x2', (datum) => (datum.target as GraphNode).x ?? 0)
      .attr('y2', (datum) => (datum.target as GraphNode).y ?? 0);

    edgeLabelHalo
      .attr('x', (datum) => (((datum.source as GraphNode).x ?? 0) + ((datum.target as GraphNode).x ?? 0)) / 2)
      .attr('y', (datum) => (((datum.source as GraphNode).y ?? 0) + ((datum.target as GraphNode).y ?? 0)) / 2 - 10);

    edgeLabel
      .attr('x', (datum) => (((datum.source as GraphNode).x ?? 0) + ((datum.target as GraphNode).x ?? 0)) / 2)
      .attr('y', (datum) => (((datum.source as GraphNode).y ?? 0) + ((datum.target as GraphNode).y ?? 0)) / 2 - 10);

    node.attr('cx', (datum) => datum.x ?? 0).attr('cy', (datum) => datum.y ?? 0);

    nodeLabelHalo.attr('x', (datum) => (datum.x ?? 0) + 18).attr('y', (datum) => (datum.y ?? 0) + 4);
    nodeLabel.attr('x', (datum) => (datum.x ?? 0) + 18).attr('y', (datum) => (datum.y ?? 0) + 4);
  });

  return () => {
    simulation.stop();
    svg.selectAll('*').remove();
  };
}
