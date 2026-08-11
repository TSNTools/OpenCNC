export type ScreenKey =
  | "dashboard"
  | "device-models"
  | "nodes"
  | "links"
  | "streams"
  | "logs"
  | "settings";

export type NavItem = {
  key: ScreenKey;
  label: string;
};

export type TopologyNode = {
  id: string;
  x: number;
  y: number;
  subtitle: string;
  status: "ok" | "warn";
};

export type TopologyLink = {
  from: string;
  to: string;
  label: string;
};

export type Metric = {
  label: string;
  value: string;
  detail: string;
  tone: "blue" | "gold" | "green" | "red";
};

export type ActivityItem = {
  title: string;
  detail: string;
  tone: "ok" | "warn";
};