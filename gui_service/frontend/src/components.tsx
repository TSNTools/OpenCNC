import type { PropsWithChildren, ReactNode } from "react";
import type { NavItem, ScreenKey, TopologyLink, TopologyNode } from "./types";

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

export function TopBar({ title }: { title: string }) {
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
        <button className="ghost-button" type="button">
          Lab
        </button>
        <button className="primary-button" type="button">
          Refresh
        </button>
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

  return (
    <div className="topology-canvas">
      <svg width="100%" height="100%" viewBox="0 0 760 460" preserveAspectRatio="none">
        {links.map((link) => {
          const source = nodes.find((node) => node.id === link.from);
          const destination = nodes.find((node) => node.id === link.to);
          if (!source || !destination) {
            return null;
          }

          const x1 = source.x + 62;
          const y1 = source.y + 32;
          const x2 = destination.x + 62;
          const y2 = destination.y + 32;
          const labelX = (x1 + x2) / 2 - 26;
          const labelY = (y1 + y2) / 2 - 16;

          return (
            <g key={`${link.from}-${link.to}`}>
              <line
                x1={x1}
                y1={y1}
                x2={x2}
                y2={y2}
                stroke="#7aa2f7"
                strokeWidth="4"
                strokeLinecap="round"
              />
              <foreignObject x={labelX} y={labelY} width="72" height="32">
                <div xmlns="http://www.w3.org/1999/xhtml" className="topology-link-label">
                  {link.label}
                </div>
              </foreignObject>
            </g>
          );
        })}
      </svg>

      {nodes.map((node) => (
        <div
          key={node.id}
          className={`topology-node ${node.status}`}
          style={{ left: node.x, top: node.y }}
        >
          <strong>{node.id}</strong>
          <span>{node.subtitle}</span>
        </div>
      ))}
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