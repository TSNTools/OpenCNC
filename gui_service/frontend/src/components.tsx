import { useMemo } from "react";
import {
  Background,
  Controls,
  MarkerType,
  ReactFlow,
  type Edge as FlowEdge,
  type Node as FlowNode,
} from "@xyflow/react";
import dagre from "dagre";
import "@xyflow/react/dist/style.css";
import type { PropsWithChildren, ReactNode } from "react";
import type { NavItem, ScreenKey, TopologyLink, TopologyNode } from "./types";

const nodeWidth = 170;
const nodeHeight = 76;

function buildLayoutedFlow(nodes: TopologyNode[], links: TopologyLink[]) {
  const graph = new dagre.graphlib.Graph();
  graph.setDefaultEdgeLabel(() => ({}));
  graph.setGraph({ rankdir: "LR", nodesep: 90, ranksep: 100 });

  nodes.forEach((node) => {
    graph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
  });

  links.forEach((link) => {
    graph.setEdge(link.from, link.to);
  });

  dagre.layout(graph);

  const flowNodes: FlowNode[] = nodes.map((node) => {
    const position = graph.node(node.id);
    return {
      id: node.id,
      type: "default",
      data: {
        label: (
          <div style={{ display: "flex", flexDirection: "column", gap: 2, fontSize: 12 }}>
            <strong>{node.id}</strong>
            <span>{node.subtitle}</span>
          </div>
        ),
      },
      position: {
        x: position.x - nodeWidth / 2,
        y: position.y - nodeHeight / 2,
      },
      style: {
        width: nodeWidth,
        borderRadius: 12,
        border: node.status === "warn" ? "1px solid #f9a826" : "1px solid #7aa2f7",
        background: node.status === "warn" ? "#fff3d6" : "#edf4ff",
        color: "#111827",
      },
    };
  });

  const flowEdges: FlowEdge[] = links.map((link) => ({
    id: `${link.from}-${link.to}`,
    source: link.from,
    target: link.to,
    label: link.label,
    type: "smoothstep",
    animated: false,
    markerEnd: { type: MarkerType.ArrowClosed },
    style: { stroke: "#7aa2f7", strokeWidth: 2 },
  }));

  return { nodes: flowNodes, edges: flowEdges };
}

export function Sidebar({
  items,
  active,
  onSelect,
}: {
  items: NavItem[];
  active: ScreenKey;
  onSelect: (key: ScreenKey) => void;
}) {
  return (
    <aside className="sidebar">
      <div>
        <div className="brand-block">
          <p className="eyebrow">OpenCNC</p>
          <h1>Control Center</h1>
          <p className="muted sidebar-muted">Simple network operations for non-developers.</p>
        </div>

        <nav className="nav-list">
          {items.map((item) => (
            <button
              key={item.key}
              className={item.key === active ? "nav-item is-active" : "nav-item"}
              onClick={() => onSelect(item.key)}
              type="button"
            >
              {item.label}
            </button>
          ))}
        </nav>
      </div>

      <div className="toggle-card">
        <div>
          <p className="toggle-title">Live Data</p>
          <p className="muted sidebar-muted">Disabled for now</p>
        </div>
        <label className="switch">
          <input type="checkbox" disabled />
          <span className="slider" />
        </label>
      </div>
    </aside>
  );
}

export function TopBar({ title, actions }: { title: string; actions?: ReactNode }) {
  return (
    <header className="topbar">
      <div>
        <p className="eyebrow">Global View</p>
        <h2>{title}</h2>
      </div>
      <div className="topbar-actions">
        <label className="search-box">
          <span>Search</span>
          <input type="text" placeholder="Find node, stream, model" />
        </label>
        {actions ?? (
          <>
            <button className="ghost-button" type="button">
              Lab
            </button>
            <button className="primary-button" type="button">
              Refresh
            </button>
          </>
        )}
      </div>
    </header>
  );
}

export function Panel({ children, className = "" }: PropsWithChildren<{ className?: string }>) {
  return <section className={`panel ${className}`.trim()}>{children}</section>;
}

export function PanelHeader({ eyebrow, title, suffix }: { eyebrow?: string; title: string; suffix?: ReactNode }) {
  return (
    <div className="panel-header">
      <div>
        {eyebrow ? <p className="eyebrow">{eyebrow}</p> : null}
        <h3>{title}</h3>
      </div>
      {suffix}
    </div>
  );
}

export function TopologyCanvas({ nodes, links }: { nodes: TopologyNode[]; links: TopologyLink[] }) {
  if (nodes.length === 0) {
    return (
      <div className="topology-canvas topology-empty-wrap">
        <div className="topology-empty">
          <div>
            <strong>No topology in store</strong>
            <p>Use the sidebar actions to add nodes, links, streams, and device models.</p>
          </div>
        </div>
      </div>
    );
  }

  const layouted = useMemo(() => buildLayoutedFlow(nodes, links), [nodes, links]);

  return (
    <div className="topology-canvas">
      <ReactFlow
        nodes={layouted.nodes}
        edges={layouted.edges}
        fitView
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        panOnDrag={false}
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#c7d2fe" gap={16} />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}

export function MetricCard({ label, value, detail, tone }: { label: string; value: string; detail: string; tone: string }) {
  return (
    <article className={`metric-card accent-${tone}`}>
      <p>{label}</p>
      <strong>{value}</strong>
      <span>{detail}</span>
    </article>
  );
}