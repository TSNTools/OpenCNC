import { useState } from "react";
import { Sidebar, TopBar } from "./components";
import { navItems } from "./data";
import {
  DashboardScreen,
  DeviceModelsScreen,
  LinksScreen,
  LogsScreen,
  NodesScreen,
  SettingsScreen,
  StreamsScreen,
} from "./screens";
import type { ScreenKey } from "./types";

const screenTitles: Record<ScreenKey, string> = {
  dashboard: "Dashboard",
  "device-models": "Add Device Models",
  nodes: "Add / Remove Node",
  links: "Add / Remove Link",
  streams: "Add / Remove Streams",
  logs: "Internal Logs",
  settings: "Settings",
};

export default function App() {
  const [activeScreen, setActiveScreen] = useState<ScreenKey>("dashboard");

  return (
    <div className="app-shell">
      <Sidebar items={navItems} active={activeScreen} onSelect={setActiveScreen} />

      <main className="workspace">
        <TopBar title={screenTitles[activeScreen]} />
        {activeScreen === "dashboard" ? <DashboardScreen /> : null}
        {activeScreen === "device-models" ? <DeviceModelsScreen /> : null}
        {activeScreen === "nodes" ? <NodesScreen /> : null}
        {activeScreen === "links" ? <LinksScreen /> : null}
        {activeScreen === "streams" ? <StreamsScreen /> : null}
        {activeScreen === "logs" ? <LogsScreen /> : null}
        {activeScreen === "settings" ? <SettingsScreen /> : null}
      </main>
    </div>
  );
}