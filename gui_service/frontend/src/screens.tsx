import { Panel, PanelHeader, TopologyCanvas, MetricCard } from "./components";
import { activities, metrics, topologyLinks, topologyNodes } from "./data";

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

export function NodesScreen() {
  return (
    <div className="two-column-layout">
      <Panel>
        <PanelHeader title="Current nodes" />
        <div className="table-card">
          <div className="table-row nodes-row table-head"><span>Node</span><span>Role</span><span>Status</span></div>
          <div className="table-row nodes-row"><span>core-1</span><span>Switch</span><span className="tag success">Up</span></div>
          <div className="table-row nodes-row"><span>edge-2</span><span>Switch</span><span className="tag success">Up</span></div>
          <div className="table-row nodes-row"><span>cam-4</span><span>Endpoint</span><span className="tag pending">Idle</span></div>
        </div>
      </Panel>
      <Panel>
        <PanelHeader title="Add or remove node" />
        <div className="form-grid">
          <label><span>Name</span><input type="text" placeholder="node-5" /></label>
          <label><span>Type</span><select><option>Switch</option><option>Endpoint</option></select></label>
          <label><span>Ports</span><input type="number" placeholder="8" /></label>
          <label><span>Management IP</span><input type="text" placeholder="192.168.1.10" /></label>
        </div>
        <div className="action-row">
          <button className="danger-button" type="button">Remove</button>
          <button className="primary-button" type="button">Add Node</button>
        </div>
      </Panel>
    </div>
  );
}

export function LinksScreen() {
  return (
    <div className="two-column-layout">
      <Panel>
        <PanelHeader title="Current links" />
        <div className="table-card">
          <div className="table-row links-row table-head"><span>From</span><span>To</span><span>State</span></div>
          <div className="table-row links-row"><span>core-1</span><span>edge-2</span><span className="tag success">Up</span></div>
          <div className="table-row links-row"><span>core-1</span><span>edge-3</span><span className="tag warning">Warn</span></div>
          <div className="table-row links-row"><span>edge-2</span><span>cam-4</span><span className="tag success">Up</span></div>
        </div>
      </Panel>
      <Panel>
        <PanelHeader title="Create or remove link" />
        <div className="form-grid">
          <label><span>Source node</span><select><option>core-1</option><option>edge-2</option></select></label>
          <label><span>Destination node</span><select><option>edge-3</option><option>cam-4</option></select></label>
          <label><span>Bandwidth</span><input type="text" placeholder="1 Gbps" /></label>
          <label><span>Constraint</span><input type="text" placeholder="critical uplink" /></label>
        </div>
        <div className="action-row">
          <button className="danger-button" type="button">Remove Link</button>
          <button className="primary-button" type="button">Create Link</button>
        </div>
      </Panel>
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