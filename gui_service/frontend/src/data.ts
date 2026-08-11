import type { ActivityItem, Metric, NavItem, TopologyLink, TopologyNode } from "./types";

export const navItems: NavItem[] = [
  { key: "dashboard", label: "Dashboard" },
  { key: "device-models", label: "Add Device Models" },
  { key: "nodes", label: "Add / Remove Node" },
  { key: "links", label: "Add / Remove Link" },
  { key: "streams", label: "Add / Remove Streams" },
  { key: "logs", label: "Internal Logs" },
  { key: "settings", label: "Settings" },
];

export const topologyNodes: TopologyNode[] = [
  { id: "core-1", x: 220, y: 90, subtitle: "Core switch", status: "ok" },
  { id: "edge-2", x: 56, y: 274, subtitle: "Edge switch", status: "ok" },
  { id: "edge-3", x: 388, y: 274, subtitle: "Edge switch", status: "warn" },
  { id: "cam-4", x: 558, y: 128, subtitle: "Video endpoint", status: "ok" },
];

export const topologyLinks: TopologyLink[] = [
  { from: "core-1", to: "edge-2", label: "1 Gbps" },
  { from: "core-1", to: "edge-3", label: "Warn" },
  { from: "edge-3", to: "cam-4", label: "TSN" },
];

export const metrics: Metric[] = [
  { label: "Packets forwarded", value: "1.28M", detail: "Last synced snapshot", tone: "blue" },
  { label: "Active streams", value: "24", detail: "6 critical", tone: "gold" },
  { label: "Healthy links", value: "18 / 20", detail: "2 warnings", tone: "green" },
  { label: "Packet drops", value: "143", detail: "Watchlisted", tone: "red" },
];

export const activities: ActivityItem[] = [
  { title: "Topology loaded from store", detail: "Last sync 2 minutes ago", tone: "ok" },
  { title: "Link warning on sw0p4", detail: "Utilization exceeded threshold", tone: "warn" },
  { title: "Device model validated", detail: "TTTech EVB model accepted", tone: "ok" },
];