import { useState } from "react";
import { Sidebar, TopBar } from "./components";
import { navItems } from "./data";
import {
  DashboardScreen,
  DeviceModelsScreen,
  LogsScreen,
  SettingsScreen,
  TopologyScreen,
} from "./screens";
import { StreamsScreen } from "./streamsScreen";
import type { ScreenKey } from "./types";

const screenTitles: Record<ScreenKey, string> = {
  dashboard: "Dashboard",
  "device-models": "Add Device Models",
  topology: "Topology",
  streams: "Add / Remove Streams",
  logs: "Internal Logs",
  settings: "Settings",
};

export default function App() {
  const [activeScreen, setActiveScreen] = useState<ScreenKey>("dashboard");
  const [streamsRefreshToken, setStreamsRefreshToken] = useState(0);
  const topBarActions =
    activeScreen === "streams" ? (
      <>
        <button className="ghost-button" type="button" onClick={() => setStreamsRefreshToken((token) => token + 1)}>
          Refresh
        </button>
        <button className="ghost-button" type="button">
          Admin ▼
        </button>
      </>
    ) : undefined;

  return (
    <div className="app-shell">
      <Sidebar items={navItems} active={activeScreen} onSelect={setActiveScreen} />

      <main className="workspace">
        <TopBar title={screenTitles[activeScreen]} actions={topBarActions} hideSearch={activeScreen === "streams"} />
        {activeScreen === "dashboard" ? <DashboardScreen /> : null}
        {activeScreen === "device-models" ? <DeviceModelsScreen /> : null}
        {activeScreen === "topology" ? <TopologyScreen /> : null}
        {activeScreen === "streams" ? <StreamsScreen refreshToken={streamsRefreshToken} /> : null}
        {activeScreen === "logs" ? <LogsScreen /> : null}
        {activeScreen === "settings" ? <SettingsScreen /> : null}
      </main>
    </div>
  );
}