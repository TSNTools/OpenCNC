export type ScreenKey =
  | "dashboard"
  | "device-models"
  | "topology"
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

export type NodeOption = {
  id: string;
  label: string;
  subtitle: string;
};

export type TrafficTypeValue =
  | "TRAFFIC_TYPE_ISOCHRONOUS"
  | "TRAFFIC_TYPE_SYNCHRONOUS"
  | "TRAFFIC_TYPE_ASYNCHRONOUS"
  | "TRAFFIC_TYPE_MANAGEMENT"
  | "TRAFFIC_TYPE_ALARM"
  | "TRAFFIC_TYPE_BEST_EFFORT_HIGH"
  | "TRAFFIC_TYPE_BEST_EFFORT_LOW";

export type StreamRankValue =
  | "RANK_UNSPECIFIED"
  | "RANK_A"
  | "RANK_B";

export type StreamFormState = {
  name: string;
  talkerNodeId: string;
  listenerNodeIds: string[];
  trafficType: TrafficTypeValue;
  rank: StreamRankValue;
  destinationMac: string;
  sourceMac: string;
  vlanId: number;
  intervalNs: number;
  maxFrameSize: number;
  maxFramesPerInterval: number;
  maxLatencyNs: number;
  maxJitterNs: number;
  minTransmitOffsetNs: number;
  maxTransmitOffsetNs: number;
  numSeamlessTrees: number;
};

export type StreamRecord = StreamFormState & {
  id: string;
  source: string;
  listeners: string;
  characteristics: string;
};