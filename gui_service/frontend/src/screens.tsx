import { Panel, PanelHeader, TopologyCanvas, MetricCard } from "./components";
import { metrics, topologyLinks, topologyNodes } from "./data";

function topologyStateTag(status: string) {
  return status === "Warn" ? "warning" : "success";
}

function TopologySection({
  title,
  detail,
  actionLabel,
  searchPlaceholder,
  emptyTitle,
  emptyDetail,
  entries,
  listTitle,
  detailsTitle,
}: {
  title: string;
  detail: string;
  actionLabel: string;
  searchPlaceholder: string;
  emptyTitle: string;
  emptyDetail: string;
  entries: Array<{ label: string; meta: string; state: string }>;
  listTitle: string;
  detailsTitle: string;
}) {
  return (
    <Panel className="topology-section">
      <div className="topology-section-head">
        <div>
          <h3>{title}</h3>
          <p>{detail}</p>
        </div>
        <button className="topology-action-button" type="button">
          <span aria-hidden="true">+</span>
          {actionLabel}
        </button>
      </div>

      <div className="topology-section-grid">
        <Panel className="topology-card topology-list-card">
          <PanelHeader title={listTitle} />
          <label className="topology-search-box search-box">
            <span>Search</span>
            <input type="text" placeholder={searchPlaceholder} />
          </label>

          {entries.length > 0 ? (
            <div className="topology-entry-list">
              {entries.map((entry) => (
                <article className="topology-entry" key={entry.label}>
                  <div>
                    <strong>{entry.label}</strong>
                    <p>{entry.meta}</p>
                  </div>
                  <span className={`tag ${topologyStateTag(entry.state)}`}>{entry.state}</span>
                </article>
              ))}
            </div>
          ) : (
            <div className="topology-empty-state">
              <div className="topology-empty-icon" aria-hidden="true" />
              <strong>{emptyTitle}</strong>
              <p>{emptyDetail}</p>
            </div>
          )}
        </Panel>

        <Panel className="topology-card topology-details-card">
          <PanelHeader title={detailsTitle} />
          <div className="topology-details-empty">
            <p>Select an item to view details</p>
          </div>
        </Panel>
      </div>
    </Panel>
  );
}

export function DashboardScreen() {
  return (
    <div className="dashboard-grid">
      <Panel className="topology-panel">
        <PanelHeader
          eyebrow="Current topology"
          title="Network map"
          suffix={<span className="status-pill">Store-backed view</span>}
        />
        <TopologyCanvas nodes={topologyNodes} links={topologyLinks} />
      </Panel>

      <Panel className="metrics-panel">
        <PanelHeader eyebrow="Traffic dashboard" title="Current summary" />
        <div className="metric-cards">
          {metrics.map((metric) => (
            <MetricCard key={metric.label} {...metric} />
          ))}
        </div>

        <div className="mini-table">
          <div className="mini-table-header">
            <span>Busiest links</span>
            <span>Utilization</span>
          </div>
          <div className="mini-table-row"><span>sw0p1 - sw0p3</span><span>78%</span></div>
          <div className="mini-table-row"><span>sw0p2 - sw0p4</span><span>65%</span></div>
          <div className="mini-table-row"><span>uplink-a - core-1</span><span>61%</span></div>
        </div>
      </Panel>

      <Panel className="activity-panel">
        <PanelHeader eyebrow="Recent events" title="Operator activity" />
        <div className="activity-list">
          {activities.map((activity) => (
            <article key={activity.title}>
              <span className={`activity-dot ${activity.tone}`} />
              <div>
                <strong>{activity.title}</strong>
                <p>{activity.detail}</p>
              </div>
            </article>
          ))}
        </div>
      </Panel>
    </div>
  );
}

export function DeviceModelsScreen() {
  return (
    <div className="three-column-layout">
      <Panel>
        <PanelHeader title="Available models" />
        <div className="list-stack">
          <button className="list-item is-selected" type="button">TTTech EVB</button>
          <button className="list-item" type="button">Rely TSN Core</button>
          <button className="list-item" type="button">Demo Switch Model</button>
        </div>
      </Panel>
      <Panel>
        <PanelHeader title="Upload queue" />
        <div className="table-card">
          <div className="table-row table-head models-row"><span>Name</span><span>Status</span></div>
          <div className="table-row models-row"><span>tttech_EVB_device_model.json</span><span className="tag success">Validated</span></div>
          <div className="table-row models-row"><span>rely-tsn-core.json</span><span className="tag pending">Pending</span></div>
        </div>
        <div className="action-row">
          <button className="ghost-button" type="button">Validate</button>
          <button className="primary-button" type="button">Upload Model</button>
        </div>
      </Panel>
      <Panel>
        <PanelHeader title="Model details" />
        <dl className="detail-grid">
          <div><dt>Vendor</dt><dd>TTTech</dd></div>
          <div><dt>Version</dt><dd>1.0.0</dd></div>
          <div><dt>Capabilities</dt><dd>NETCONF, QBV, PSFP</dd></div>
          <div><dt>Activation</dt><dd>Ready</dd></div>
        </dl>
      </Panel>
    </div>
  );
}

export function TopologyScreen() {
  return (
    <div className="topology-page">
      <TopologySection
        title="Nodes"
        detail="Manage and view all network nodes."
        actionLabel="Add Node"
        searchPlaceholder="Search nodes..."
        emptyTitle="No nodes available"
        emptyDetail="Add a node to get started."
        entries={topologyNodes.map((node) => ({
          label: node.id,
          meta: node.subtitle,
          state: node.status === "warn" ? "Warn" : "Up",
        }))}
        listTitle="Available Nodes"
        detailsTitle="Node Details"
      />

      <TopologySection
        title="Links"
        detail="Manage and view all network links."
        actionLabel="Add Link"
        searchPlaceholder="Search links..."
        emptyTitle="No links available"
        emptyDetail="Add a link to get started."
        entries={topologyLinks.map((link) => ({
          label: `${link.from} → ${link.to}`,
          meta: link.label,
          state: link.label.toLowerCase().includes("warn") ? "Warn" : "Up",
        }))}
        listTitle="Available Links"
        detailsTitle="Link Details"
      />
    </div>
  );
}

export function StreamsScreen() {
  return (
    <div className="two-column-layout">
      <Panel>
        <PanelHeader title="Configured streams" />
        <div className="table-card">
          <div className="table-row streams-row table-head"><span>Name</span><span>Path</span><span>Priority</span></div>
          <div className="table-row streams-row"><span>Video-A</span><span>cam-4 to core-1</span><span>High</span></div>
          <div className="table-row streams-row"><span>Control-B</span><span>edge-2 to edge-3</span><span>Critical</span></div>
          <div className="table-row streams-row"><span>Sync-C</span><span>core-1 to all</span><span>Medium</span></div>
        </div>
      </Panel>
      <Panel>
        <PanelHeader title="Add or remove stream" />
        <div className="form-grid">
          <label><span>Name</span><input type="text" placeholder="stream-12" /></label>
          <label><span>Source</span><select><option>cam-4</option><option>edge-2</option></select></label>
          <label><span>Destination</span><select><option>core-1</option><option>edge-3</option></select></label>
          <label><span>Priority</span><select><option>Critical</option><option>High</option><option>Medium</option></select></label>
        </div>
        <div className="action-row">
          <button className="ghost-button" type="button">Validate</button>
          <button className="danger-button" type="button">Remove Stream</button>
          <button className="primary-button" type="button">Add Stream</button>
        </div>
      </Panel>
    </div>
  );
}

export function LogsScreen() {
  return (
    <Panel>
      <PanelHeader eyebrow="Kafka-backed logs" title="Internal CNC logs" />
      <div className="filter-row">
        <select><option>All severities</option></select>
        <select><option>Last 15 minutes</option></select>
        <input type="text" placeholder="Search message or correlation ID" />
      </div>
      <div className="table-card log-table">
        <div className="table-row logs-row table-head"><span>Time</span><span>Severity</span><span>Subsystem</span><span>Message</span></div>
        <div className="table-row logs-row"><span>10:42:01</span><span className="tag success">INFO</span><span>config-service</span><span>Configuration applied successfully</span></div>
        <div className="table-row logs-row"><span>10:40:17</span><span className="tag warning">WARN</span><span>monitor-service</span><span>High utilization on sw0p4</span></div>
        <div className="table-row logs-row"><span>10:38:11</span><span className="tag danger">ERROR</span><span>engine</span><span>Validation failed for stream path</span></div>
      </div>
    </Panel>
  );
}

export function SettingsScreen() {
  return (
    <Panel className="empty-panel">
      <div>
        <p className="eyebrow">Reserved</p>
        <h3>Settings</h3>
        <p className="muted">This screen is intentionally empty for now.</p>
      </div>
    </Panel>
  );
}