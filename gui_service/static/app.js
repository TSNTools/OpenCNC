const navItems = Array.from(document.querySelectorAll(".nav-item"));
const screenNodes = Array.from(document.querySelectorAll(".screen"));
const screenTitle = document.getElementById("screen-title");
const toast = document.getElementById("toast");

const state = {
  selectedModelID: "",
  selectedNodeID: "",
  selectedLinkID: "",
  selectedStreamID: "",
  nodes: [],
  streams: [],
  deviceModels: [],
  monitoringCounters: [],
  monitoringMetrics: [],
  monitoringTargets: [],
  selectedMonitoringTargetID: "",
  activeMonitoringItemID: "",
  selectedMonitoringItemIDsByTarget: {},
  appliedMonitoringItemIDsByTarget: {},
  monitoringPollIntervalsByTarget: {},
};

const DEFAULT_MONITORING_POLL_INTERVAL_S = 1;

initializeNavigation();
initializeActionButtons();
initializeListSelection();
initializeStreamMultiSelect();
initializeMonitoringUI();
updateTopologyTopbarControls("dashboard");
loadAllViews();

function initializeNavigation() {
  const topologyToggle = document.getElementById("topology-nav-toggle");
  const topologySubmenu = document.getElementById("topology-nav-submenu");
  const topologySubitems = Array.from(
    document.querySelectorAll(".nav-subitem")
  );

  // Normal top-level navigation items
  navItems.forEach((item) => {
    // Topology parent is handled separately below
    if (item === topologyToggle) {
      return;
    }

    item.addEventListener("click", () => {
      const screen = item.dataset.screen;

      // Update active state of main navigation
      navItems.forEach((entry) => {
        entry.classList.toggle("is-active", entry === item);
      });

      // Remove active state from topology submenu
      topologySubitems.forEach((entry) => {
        entry.classList.remove("is-active");
      });

      // Show selected screen
      screenNodes.forEach((entry) => {
        entry.classList.toggle(
          "is-visible",
          entry.id === `screen-${screen}`
        );
      });

      // Update title
      screenTitle.textContent = item.textContent.trim();
      updateTopologyTopbarControls(screen);

      loadView(screen);
    });
  });


  // Expand / collapse Topology submenu
  if (topologyToggle && topologySubmenu) {
    topologyToggle.addEventListener("click", () => {
      const collapsed = topologySubmenu.classList.toggle("is-collapsed");

      topologyToggle.classList.toggle("is-collapsed", collapsed);

      topologyToggle.setAttribute(
        "aria-expanded",
        String(!collapsed)
      );
    });
  }


  // Topology submenu items
  topologySubitems.forEach((item) => {
    item.addEventListener("click", () => {
      const screen = item.dataset.screen;

      // Keep Topology parent active
      navItems.forEach((entry) => {
        entry.classList.toggle(
          "is-active",
          entry === topologyToggle
        );
      });

      // Highlight selected submenu item
      topologySubitems.forEach((entry) => {
        entry.classList.toggle(
          "is-active",
          entry === item
        );
      });

      // Show selected screen
      screenNodes.forEach((entry) => {
        entry.classList.toggle(
          "is-visible",
          entry.id === `screen-${screen}`
        );
      });

      // Update title
      screenTitle.textContent =
        `Topology / ${item.textContent.trim()}`;
      updateTopologyTopbarControls(screen);

      loadView(screen);
    });
  });
}

function updateTopologyTopbarControls(screen) {
  const importButton = document.getElementById("import-topology-btn");
  if (!importButton) {
    return;
  }

  const show = screen === "topology-nodes" || screen === "topology-links";
  importButton.classList.toggle("hidden", !show);
}

function initializeActionButtons() {
  const buttons = Array.from(document.querySelectorAll("[data-action]"));
  buttons.forEach((button) => {
    button.addEventListener("click", async () => {
      const action = button.dataset.action;
      if (!action) {
        return;
      }

      const confirmText = button.dataset.confirm;
      if (confirmText && !window.confirm(confirmText)) {
        return;
      }

      const payload = await gatherPayload(action);

      if (payload === null) {
        return;
      }

      const response = await callAction(action, payload);
      showToast(response.message || `${action} completed`);

      if (response.success) {
        loadAllViews();
      }
    });
  });

  const toggleModelEdit = document.getElementById("toggle-model-edit");
  const saveModelEdit = document.getElementById("model-save-edit");
  const cancelModelEdit = document.getElementById("model-cancel-edit");
  const toggleNodeEdit = document.getElementById("toggle-node-edit");
  const saveNodeEdit = document.getElementById("node-save-edit");
  const cancelNodeEdit = document.getElementById("node-cancel-edit");
  const toggleLinkEditButton = document.getElementById("toggle-link-edit");
  const saveLinkEditButton = document.getElementById("link-save-edit");
  const cancelLinkEditButton = document.getElementById("link-cancel-edit");
  
  if (toggleModelEdit) {
    toggleModelEdit.addEventListener("click", () => {
      if (!state.selectedModelID) {
        showToast("Select a model first.");
        return;
      }
      toggleModelEditForm();
    });
  }

  if (saveModelEdit) {
    saveModelEdit.addEventListener("click", async () => {
      if (!state.selectedModelID) {
        showToast("Select a model first.");
        return;
      }

      const payload = {
        id: state.selectedModelID,
        name: getValue("model-edit-name"),
        version: getValue("model-edit-version"),
        vendor: getValue("model-edit-vendor"),
        yang: getValue("model-edit-yang"),
      };

      const result = await callAction("editModel", payload);
      showToast(result.message || "Model updated");
      if (result.success) {
        hideModelEditForm();
        await loadDeviceModels();
      }
    });
  }

  if (cancelModelEdit) {
    cancelModelEdit.addEventListener("click", () => {
      hideModelEditForm();
    });
  }

  if (toggleNodeEdit) {
    toggleNodeEdit.addEventListener("click", () => {
      if (!state.selectedNodeID) {
        showToast("Select a node first.");
        return;
      }
      toggleNodeEditForm();
    });
  }

  if (saveNodeEdit) {
    saveNodeEdit.addEventListener("click", async () => {
      if (!state.selectedNodeID) {
        showToast("Select a node first.");
        return;
      }

      const managementPortRaw = getValue("node-edit-management-port").trim();
      let managementPort;
      if (managementPortRaw) {
        managementPort = Number.parseInt(managementPortRaw, 10);

        if (
          Number.isNaN(managementPort) ||
          managementPort < 1 ||
          managementPort > 65535
        ) {
          showToast("Management port must be between 1 and 65535.");
          return;
        }
      }

      const payload = {
        id: state.selectedNodeID,
        name: getValue("node-edit-name"),
        type: getValue("node-edit-type"),
        ports: getValue("node-edit-ports"),
        username: getValue("node-edit-username"),
        password: getValue("node-edit-password"),
        deviceModel: getValue("node-edit-device-model"),
        managementIp: getValue("node-edit-management-ip"),
        managementProtocol: getValue("node-edit-management-protocol"),
        managementPort: managementPort,
      };

      const result = await callAction("editNode", payload);
      showToast(result.message || "Node updated");
      if (result.success) {
        hideNodeEditForm();
        await loadNodes();
      }
    });
  }

  if (cancelNodeEdit) {
    cancelNodeEdit.addEventListener("click", () => {
      hideNodeEditForm();
    });
  }


  if (toggleLinkEditButton) {
    toggleLinkEditButton.addEventListener("click", () => {
      if (!state.selectedLinkID) {
        showToast("Select a link first.");
        return;
      }

      toggleLinkEditForm();
    });
  }

  if (cancelLinkEditButton) {
    cancelLinkEditButton.addEventListener("click", () => {
      hideLinkEditForm();
    });
  }

  if (saveLinkEditButton) {
    saveLinkEditButton.addEventListener("click", async () => {
      if (!state.selectedLinkID) {
        showToast("Select a link first.");
        return;
      }

      const source = getValue("link-edit-source").trim();
      const destination = getValue("link-edit-destination").trim();
      const bandwidth = getValue("link-edit-bandwidth").trim();

      if (!source || !destination) {
        showToast("Select both source and destination.");
        return;
      }

      if (source === destination) {
        showToast("Source and destination must be different.");
        return;
      }

      if (!bandwidth) {
        showToast("Select a bandwidth.");
        return;
      }

      const response = await callAction("updateLink", {
        id: state.selectedLinkID,
        source,
        destination,
        bandwidth,
      });

      showToast(response.message || "Link updated");

      if (response.success) {
        hideLinkEditForm();
        await loadAllViews();
      }
    });
  }

  const addNodeButton = document.getElementById("add-node-btn");
  if (addNodeButton) {
    addNodeButton.addEventListener("click", async () => {
      const payload = buildNodeCreatePayload();
      if (!payload) {
        return;
      }

      const response = await callAction("addNode", { query: JSON.stringify(payload) });
      showToast(response.message || "Node created");

      if (response.success) {
        resetNodeCreateForm();
        await loadAllViews();
      }
    });
  }

  const importNodeButton = document.getElementById("import-node-btn");
  if (importNodeButton) {
    importNodeButton.addEventListener("click", async () => {
      const payloadText = await readUploadedFile("node-upload");
      if (!payloadText) {
        showToast("Select a node JSON file.");
        return;
      }

      await importNodePayload(payloadText);
    });
  }

  const addLinkButton = document.getElementById("add-link-btn");
  if (addLinkButton) {
    addLinkButton.addEventListener("click", async () => {
      const source = getValue("link-source").trim();
      const destination = getValue("link-destination").trim();
      const bandwidth = getValue("link-bandwidth").trim();

      if (!source || !destination) {
        showToast("Select both source and destination.");
        return;
      }

      if (source === destination) {
        showToast("Source and destination must be different.");
        return;
      }

      const response = await callAction("addLink", { source, destination, bandwidth });
      showToast(response.message || "Link created");

      if (response.success) {
        await loadAllViews();
      }
    });
  }

  const importButton = document.getElementById("import-topology-btn");
  const importInput = document.getElementById("import-topology-input");
  if (importButton && importInput) {
    importButton.addEventListener("click", () => {
      importInput.value = "";
      importInput.click();
    });

    importInput.addEventListener("change", async () => {
      const payloadText = await readUploadedFile("import-topology-input");
      if (!payloadText) {
        showToast("Select a topology JSON file.");
        return;
      }

      await importTopologyPayload(payloadText);
    });
  }

  const importStreamButton = document.getElementById("stream-import-btn");

  if (importStreamButton) {
    importStreamButton.addEventListener("click", async () => {
      const payloadText = await readUploadedFile("stream-import-input");

      if (!payloadText) {
        showToast("Select a stream JSON file.");
        return;
      }

      let importedPayload;

      try {
        importedPayload = JSON.parse(payloadText);
      } catch (error) {
        showToast(`Invalid stream JSON: ${error.message}`);
        return;
      }

      const payload = convertImportedStream(importedPayload);

      const response = await callAction("addStream", payload);

      showToast(response.message || "Stream imported");

      if (response.success) {
        await loadAllViews();
      }
    });
  }
}

function initializeMonitoringUI() {
  const nodeSelect = document.getElementById("monitoring-node-select");
  const portSelect = document.getElementById("monitoring-port-select");
  const searchInput = document.getElementById("monitoring-search");
  const availableList = document.getElementById("monitoring-available-list");
  const selectedList = document.getElementById("monitoring-selected-list");
  const addButton = document.getElementById("monitoring-add-selection");
  const removeButton = document.getElementById("monitoring-remove-selection");
  const applyButton = document.getElementById("monitoring-apply-selection");
  const pollIntervalInput = document.getElementById("monitoring-poll-interval");

  if (nodeSelect) {
    nodeSelect.addEventListener("change", () => {
      state.activeMonitoringItemID = "";
      populatePortSelector();
      syncMonitoringTargetFromSelectors();
      renderMonitoringAvailableList();
      renderMonitoringSelectedList();
      clearMonitoringDetail();
    });
  }

  if (portSelect) {
    portSelect.addEventListener("change", () => {
      state.activeMonitoringItemID = "";
      syncMonitoringTargetFromSelectors();
      renderMonitoringAvailableList();
      renderMonitoringSelectedList();
      clearMonitoringDetail();
    });
  }

  if (searchInput) {
    searchInput.addEventListener("input", () => {
      renderMonitoringAvailableList();
    });
  }

  if (availableList) {
    availableList.addEventListener("click", (event) => {
      const item = event.target.closest("li[data-id]");
      if (!item) {
        return;
      }

      state.activeMonitoringItemID = item.dataset.id || "";
      renderMonitoringAvailableList();
      renderMonitoringDetails(state.activeMonitoringItemID);
    });
  }

  if (selectedList) {
    selectedList.addEventListener("click", (event) => {
      const item = event.target.closest("li[data-id]");
      if (!item) {
        return;
      }

      state.activeMonitoringItemID = item.dataset.id || "";
      renderMonitoringSelectedList();
      renderMonitoringDetails(state.activeMonitoringItemID);
    });
  }

  if (pollIntervalInput) {
    pollIntervalInput.value = String(DEFAULT_MONITORING_POLL_INTERVAL_S);
    pollIntervalInput.addEventListener("input", () => {
      const targetID = getCurrentMonitoringTargetID();
      const itemID = state.activeMonitoringItemID;
      if (!targetID || !itemID) {
        return;
      }

      setMonitoringPollInterval(targetID, itemID, pollIntervalInput.value);
      renderMonitoringSelectedList();
    });
  }

  if (addButton) {
    addButton.addEventListener("click", () => {
      if (!state.activeMonitoringItemID) {
        showToast("Select a metric first.");
        return;
      }

      const targetID = getCurrentMonitoringTargetID();
      if (!targetID) {
        showToast("Select a target first.");
        return;
      }

      const selectedSet = getSelectedMonitoringSet(targetID);
      selectedSet.add(state.activeMonitoringItemID);
      renderMonitoringSelectedList();
    });
  }

  if (removeButton) {
    removeButton.addEventListener("click", () => {
      if (!state.activeMonitoringItemID) {
        showToast("Select a metric first.");
        return;
      }

      const targetID = getCurrentMonitoringTargetID();
      if (!targetID) {
        showToast("Select a target first.");
        return;
      }

      const selectedSet = getSelectedMonitoringSet(targetID);
      selectedSet.delete(state.activeMonitoringItemID);
      renderMonitoringSelectedList();
    });
  }

  if (applyButton) {
    applyButton.addEventListener("click", async () => {
      const targetID = getCurrentMonitoringTargetID();
      if (!targetID) {
        showToast("Select a target first.");
        return;
      }

      const selectedSet = getSelectedMonitoringSet(targetID);
      state.appliedMonitoringItemIDsByTarget[targetID] = new Set(selectedSet);
      await loadMonitoringDataPanel();
      showToast("Monitoring selection applied.");
    });
  }
}

function initializeListSelection() {
  bindSelectableList("device-model-list", (item) => {
    state.selectedModelID = item.dataset.id || "";
    setText("model-name", item.dataset.name || "-");
    setText("model-version", item.dataset.version || "-");
    setText("model-vendor", item.dataset.vendor || "-");
    setText("model-yang", item.dataset.yang || "-");
    setModelEditFields();
  });

  bindSelectableList("node-list", (item) => {
    state.selectedNodeID = item.dataset.id || "";
    setText("node-name", item.dataset.name || "-");
    setText("node-type", item.dataset.type || "-");
    setText("node-ports", item.dataset.ports || "-");
    setText("node-device-model", item.dataset.deviceModel || "-");
    setText("node-management-ip", item.dataset.managementIp || "-");
    setText("node-management-protocol", item.dataset.managementProtocol || "-");
    setText("node-management-port", item.dataset.managementPort || "-");
    setNodeEditFields();
  });

  bindSelectableList("link-list", (item) => {
    state.selectedLinkID = item.dataset.id || "";
    setText("link-detail-source", item.dataset.source || "-");
    setText("link-detail-destination", item.dataset.destination || "-");
    setText("link-detail-bandwidth", item.dataset.bandwidth || "-");
  });

  bindSelectableList("stream-list", (item) => {
    selectStreamItem(item);
  });
}

function initializeStreamMultiSelect() {
  const multiSelect = document.getElementById("stream-listeners-select");

  if (!multiSelect) {
    return;
  }

  const toggle = multiSelect.querySelector(".multi-select-toggle");

  if (!toggle) {
    return;
  }

  toggle.addEventListener("click", (event) => {
    event.stopPropagation();
    multiSelect.classList.toggle("open");
  });

  document.addEventListener("click", (event) => {
    if (!multiSelect.contains(event.target)) {
      multiSelect.classList.remove("open");
    }
  });
}

function bindSelectableList(listID, onSelect) {
  const list = document.getElementById(listID);
  if (!list) {
    return;
  }

  list.addEventListener("click", (event) => {
    const clicked = event.target.closest("li");
    if (!clicked || clicked.classList.contains("empty-item")) {
      return;
    }

    Array.from(list.querySelectorAll("li")).forEach((item) => item.classList.remove("selected"));
    clicked.classList.add("selected");
    onSelect(clicked);
  });
}

async function loadAllViews() {
  await Promise.all([
    loadDashboard(),
    loadDeviceModels(),
    loadNodes(),
    loadLinks(),
    loadStreams(),
    loadMonitoring(),
    loadRecentEvents(),
  ]);
}

async function loadView(screen) {
  switch (screen) {
    case "dashboard":
      await Promise.all([
        loadDashboard(),
        loadRecentEvents(),
      ]);
      break;

    case "device-models":
      await Promise.all([
        loadDeviceModels(),
        loadRecentEvents(),
      ]);
      break;

    case "topology-nodes":
      await loadNodes();
      break;

    case "topology-links":
      await loadLinks();
      break;

    case "streams":
      await Promise.all([
        loadStreams(),
        loadRecentEvents(),
      ]);
      break;

    case "monitoring":
      await loadMonitoring();
      break;

    case "settings":
      break;

    default:
      break;
  }
}

async function loadDashboard() {
  const data = await fetchJSON("/api/v1/dashboard", {});
  setText("packets-forwarded", data.packetsForwarded || "-");
  setText("packet-drops", data.packetDrops || "-");
  setText("active-streams", data.activeStreams || "-");
  renderTopology(data.networkModel || { nodes: [], links: [] });

  const topTalkers = document.getElementById("top-talkers-list");
  renderList(topTalkers, data.topTalkers || [], (item) => {
    const li = document.createElement("li");
    li.textContent = item;
    return li;
  }, "No data available");
}

function renderTopology(networkModel) {
  const topologyCanvas = document.getElementById("topology-canvas");
  if (!topologyCanvas) {
    return;
  }

  const nodes = Array.isArray(networkModel.nodes) ? networkModel.nodes : [];
  const links = Array.isArray(networkModel.links) ? networkModel.links : [];

  if (!nodes.length) {
    topologyCanvas.classList.add("empty");
    topologyCanvas.innerHTML = "<p>Topology will appear here when backend data is available.</p>";
    return;
  }

  topologyCanvas.classList.remove("empty");

  const width = 760;
  const height = 360;
  const centerX = width / 2;
  const centerY = height / 2;
  const radius = Math.min(width, height) * 0.33;

  const positionedNodes = nodes.map((node, index) => {
    const angle = (2 * Math.PI * index) / nodes.length;
    return {
      ...node,
      x: centerX + radius * Math.cos(angle),
      y: centerY + radius * Math.sin(angle),
    };
  });

  const nodeMap = new Map(positionedNodes.map((node) => [node.id, node]));

  const lineMarkup = links
    .map((link) => {
      const source = nodeMap.get(link.source);
      const target = nodeMap.get(link.destination);
      if (!source || !target) {
        return "";
      }
      return `<line x1="${source.x}" y1="${source.y}" x2="${target.x}" y2="${target.y}" class="topology-edge" />`;
    })
    .join("");

  const nodeMarkup = positionedNodes
    .map((node) => {
      const label = node.name || node.id;
      return `
        <g>
          <circle cx="${node.x}" cy="${node.y}" r="14" class="topology-node-dot" />
          <text x="${node.x}" y="${node.y + 34}" text-anchor="middle" class="topology-node-label">${escapeHtml(label)}</text>
        </g>
      `;
    })
    .join("");

  topologyCanvas.innerHTML = `
    <svg viewBox="0 0 ${width} ${height}" width="100%" height="360" class="topology-svg" preserveAspectRatio="xMidYMid meet" aria-label="Topology graph">
      ${lineMarkup}
      ${nodeMarkup}
    </svg>
  `;
}

async function loadDeviceModels() {
  const models = await fetchJSON("/api/v1/device-models", []);
  state.deviceModels = Array.isArray(models) ? models : [];
  const list = document.getElementById("device-model-list");
  renderList(list, models, (model) => {
    const li = document.createElement("li");
    li.textContent = model.name || model.id;
    li.dataset.id = model.id || "";
    li.dataset.name = model.name || "";
    li.dataset.version = model.version || "";
    li.dataset.vendor = model.vendor || "";
    li.dataset.yang = model.yang || "";
    return li;
  }, "No models available");
  populateNodeCreateEditDeviceModelOptions();
  clearModelDetails();
}

async function loadNodes() {
  const nodes = await fetchJSON("/api/v1/nodes", []);
  state.nodes = Array.isArray(nodes) ? nodes : [];
  const list = document.getElementById("node-list");
  renderList(list, nodes, (node) => {
    const li = document.createElement("li");
    li.textContent = node.name || node.id;
    li.dataset.id = node.id || "";
    li.dataset.name = node.name || "";
    li.dataset.type = node.type || "";
    li.dataset.state = node.state || "";
    li.dataset.ports = node.ports || "";
    li.dataset.links = node.links || "";
    li.dataset.deviceModel = String(node.deviceModel || "");
    li.dataset.managementIp = String(node.managementIp || "");
    li.dataset.managementProtocol = String(node.managementProtocol || "");
    li.dataset.managementPort = String(node.managementPort ?? "");
    return li;
  }, "No nodes available");
  populateLinkNodeOptions(nodes);
  populateStreamNodeOptions(nodes);
  clearNodeDetails();
}

async function loadLinks() {
  const links = await fetchJSON("/api/v1/links", []);
  const list = document.getElementById("link-list");
  renderList(list, links, (link) => {
    const li = document.createElement("li");
    li.textContent = `${link.source || "-"} -> ${link.destination || "-"}`;
    li.dataset.id = link.id || "";
    li.dataset.source = link.source || "";
    li.dataset.destination = link.destination || "";
    li.dataset.bandwidth = link.bandwidth || "";
    return li;
  }, "No links available");
  clearLinkDetails();
}

async function loadStreams() {

  const streams = await fetchJSON("/api/v1/streams", []);

  state.streams = Array.isArray(streams) ? streams : [];

  const list = document.getElementById("stream-list");

  renderList(
    list,
    streams,
    (stream) => {

      const li = document.createElement("li");

      li.textContent = stream.name || stream.id;

      li.dataset.id = stream.id || "";
      li.dataset.name = stream.name || "";
      li.dataset.source = stream.source || "";
      li.dataset.listeners = stream.listeners || "";
      li.dataset.characteristics = stream.characteristics || "";
      li.dataset.talkerNodeId = stream.talkerNodeId || "";

      li.dataset.listenerNodeIds =
        Array.isArray(stream.listenerNodeIds)
          ? stream.listenerNodeIds.join("|")
          : "";

      li.dataset.trafficType = stream.trafficType || "";
      li.dataset.rank = stream.rank || "";
      li.dataset.destinationMac = stream.destinationMac || "";
      li.dataset.sourceMac = stream.sourceMac || "";

      li.dataset.vlanId =
        stream.vlanId ?? "";

      li.dataset.intervalNs =
        stream.intervalNs ?? "";

      li.dataset.maxFrameSize =
        stream.maxFrameSize ?? "";

      li.dataset.maxFramesPerInterval =
        stream.maxFramesPerInterval ?? "";

      li.dataset.maxLatencyNs =
        stream.maxLatencyNs ?? "";

      li.dataset.maxJitterNs =
        stream.maxJitterNs ?? "";

      li.dataset.minTransmitOffsetNs =
        stream.minTransmitOffsetNs ?? "";

      li.dataset.maxTransmitOffsetNs =
        stream.maxTransmitOffsetNs ?? "";

      li.dataset.numSeamlessTrees =
        stream.numSeamlessTrees ?? "";

      return li;
    },
    "No streams available"
  );


  /*
   * Do not automatically select a stream.
   *
   * The Create Stream form must remain empty until
   * the user explicitly creates a new stream or uses
   * an Edit action.
   */

  state.selectedStreamID = "";

  clearStreamDetails();
}

function convertImportedStream(payload) {
  const listeners = [];

  if (Array.isArray(payload.lstnsrPssblPlcmnt)) {
    for (const group of payload.lstnsrPssblPlcmnt) {
      if (Array.isArray(group)) {
        for (const listener of group) {
          if (typeof listener === "string" && listener.trim()) {
            listeners.push(listener.trim());
          }
        }
      }
    }
  }

  const talkers = Array.isArray(payload.tlkrPssblPlcmnt)
    ? payload.tlkrPssblPlcmnt
    : [];

  return {
    name: payload.name || "",
    talkerNodeId: talkers.length > 0 ? talkers[0] : "",
    listenerNodeIds: listeners,

    trafficType: payload.type || "",

    intervalNs: payload.interval || 0,
    maxFramesPerInterval: payload.maxFramesPerInterval || 0,
    maxFrameSize: payload.maxFrameSize || 0,
    maxLatencyNs: payload.maxLatency || 0,
    maxJitterNs: payload.jitter || 0,

    minTransmitOffsetNs:
      payload["earliest-transmit-offset"] || 0,

    maxTransmitOffsetNs:
      payload["latest-transmit-offset"] || 0,
  };
}

async function loadMonitoring() {
  const [counters, metrics, targets] = await Promise.all([
    fetchMonitoringItems("/api/v1/monitoring/counters", "counter"),
    fetchMonitoringItems("/api/v1/monitoring/metrics", "metric"),
    fetchMonitoringTargets(),
  ]);

  state.monitoringCounters = counters;
  state.monitoringMetrics = metrics;
  state.monitoringTargets = targets;

  populateNodeSelector();
  populatePortSelector();
  syncMonitoringTargetFromSelectors();

  pruneMonitoringSelections();

  const availableIDs = new Set(getAvailableMonitoringItems().map((item) => item.id));

  if (state.activeMonitoringItemID && !availableIDs.has(state.activeMonitoringItemID)) {
    state.activeMonitoringItemID = "";
  }

  renderMonitoringAvailableList();
  renderMonitoringSelectedList();

  if (state.activeMonitoringItemID) {
    renderMonitoringDetails(state.activeMonitoringItemID);
  } else {
    clearMonitoringDetail();
  }

  await loadMonitoringDataPanel();
}

async function fetchMonitoringItems(path, type) {
  const items = await fetchJSON(path, []);
  if (!Array.isArray(items)) {
    return [];
  }

  return items
    .map((entry) => {
      if (typeof entry === "string") {
        return { id: entry, label: entry, description: "", type };
      }

      if (!entry || typeof entry !== "object") {
        return null;
      }

      const id = entry.id || entry.name || entry.key || "";
      if (!id) {
        return null;
      }

      return {
        id,
        label: entry.label || entry.displayName || entry.name || id,
        description: entry.description || "",
        type,
      };
    })
    .filter(Boolean);
}

async function fetchMonitoringTargets() {
  const targets = await fetchJSON("/api/v1/monitoring/targets", []);
  if (!Array.isArray(targets)) {
    return [];
  }

  return targets
    .map((entry) => {
      if (!entry || typeof entry !== "object") {
        return null;
      }

      const id = entry.id || "";
      const node = entry.node || "";
      const port = entry.port || "";
      if (!id || !node) {
        return null;
      }

      return {
        id,
        node,
        port,
        label: entry.label || `${node} - port ${port}`,
        counters: Array.isArray(entry.counters) ? entry.counters : [],
        metrics: Array.isArray(entry.metrics) ? entry.metrics : [],
      };
    })
    .filter(Boolean);
}

function populateNodeSelector() {
  const nodeSelect = document.getElementById("monitoring-node-select");
  if (!nodeSelect) {
    return;
  }

  const currentNode = nodeSelect.value;
  const nodes = Array.from(new Set(state.monitoringTargets.map((target) => target.node)));

  nodeSelect.innerHTML = '<option value="">Select Node</option>';
  nodes.forEach((node) => {
    const option = document.createElement("option");
    option.value = node;
    option.textContent = node;
    nodeSelect.appendChild(option);
  });

  if (currentNode && nodes.includes(currentNode)) {
    nodeSelect.value = currentNode;
    return;
  }

  if (nodes.length) {
    nodeSelect.value = nodes[0];
  }
}

function populatePortSelector() {
  const nodeSelect = document.getElementById("monitoring-node-select");
  const portSelect = document.getElementById("monitoring-port-select");
  if (!nodeSelect || !portSelect) {
    return;
  }

  const selectedNode = nodeSelect.value;
  const currentPort = portSelect.value;
  const ports = state.monitoringTargets
    .filter((target) => target.node === selectedNode)
    .map((target) => target.port);

  portSelect.innerHTML = '<option value="">Select Port</option>';
  ports.forEach((port) => {
    const option = document.createElement("option");
    option.value = port;
    option.textContent = port;
    portSelect.appendChild(option);
  });

  if (currentPort && ports.includes(currentPort)) {
    portSelect.value = currentPort;
    return;
  }

  if (ports.length) {
    portSelect.value = ports[0];
  }
}

function syncMonitoringTargetFromSelectors() {
  const nodeSelect = document.getElementById("monitoring-node-select");
  const portSelect = document.getElementById("monitoring-port-select");
  if (!nodeSelect || !portSelect) {
    state.selectedMonitoringTargetID = "";
    return;
  }

  const selectedNode = nodeSelect.value;
  const selectedPort = portSelect.value;
  const target = state.monitoringTargets.find(
    (entry) => entry.node === selectedNode && entry.port === selectedPort
  );

  state.selectedMonitoringTargetID = target ? target.id : "";
}

function getAvailableMonitoringItems() {
  const target = state.monitoringTargets.find((entry) => entry.id === state.selectedMonitoringTargetID);
  if (!target) {
    return [];
  }

  const allItems = [...state.monitoringCounters, ...state.monitoringMetrics];
  const allowedIDs = new Set([...target.counters, ...target.metrics]);
  const searchInput = document.getElementById("monitoring-search");
  const query = searchInput ? searchInput.value.trim().toLowerCase() : "";

  return allItems
    .filter((item) => allowedIDs.has(item.id))
    .filter((item) => {
      if (!query) {
        return true;
      }
      return item.label.toLowerCase().includes(query) || item.id.toLowerCase().includes(query);
    });
}

function renderMonitoringAvailableList() {
  const list = document.getElementById("monitoring-available-list");
  if (!list) {
    return;
  }

  const items = getAvailableMonitoringItems();
  list.innerHTML = "";

  if (!state.selectedMonitoringTargetID) {
    const li = document.createElement("li");
    li.className = "empty-item";
    li.textContent = "Select a target";
    list.appendChild(li);
    return;
  }

  if (!items.length) {
    const li = document.createElement("li");
    li.className = "empty-item";
    li.textContent = "No metrics available for target";
    list.appendChild(li);
    return;
  }

  items.forEach((item) => {
    const li = document.createElement("li");
    li.dataset.id = item.id;
    li.textContent = item.label;
    if (item.id === state.activeMonitoringItemID) {
      li.classList.add("selected");
    }
    list.appendChild(li);
  });
}

function renderMonitoringSelectedList() {
  const list = document.getElementById("monitoring-selected-list");
  if (!list) {
    return;
  }

  const targetID = getCurrentMonitoringTargetID();
  const selectedSet = getSelectedMonitoringSet(targetID, false);
  const selectedItems = Array.from(selectedSet)
    .map((id) => getMonitoringItemByID(id))
    .filter(Boolean);

  list.innerHTML = "";
  if (!selectedItems.length) {
    const li = document.createElement("li");
    li.className = "empty-item";
    li.textContent = "No metrics selected";
    list.appendChild(li);
    return;
  }

  selectedItems.forEach((item) => {
    const li = document.createElement("li");
    li.className = "monitoring-selected-item";
    li.dataset.id = item.id;
    li.textContent = `${item.label} (${getMonitoringPollInterval(targetID, item.id)} s)`;
    if (item.id === state.activeMonitoringItemID) {
      li.classList.add("selected");
    }
    list.appendChild(li);
  });
}

function getMonitoringItemByID(itemID) {
  return [...state.monitoringCounters, ...state.monitoringMetrics]
    .find((item) => item.id === itemID) || null;
}

function renderMonitoringDetails(itemID) {
  const item = getMonitoringItemByID(itemID);
  const target = state.monitoringTargets.find((entry) => entry.id === state.selectedMonitoringTargetID);

  if (!item || !target) {
    clearMonitoringDetail();
    return;
  }

  setText("monitoring-detail-name", item.label);
  setText("monitoring-detail-description", item.description || "-");
  setText("monitoring-detail-type", item.type === "counter" ? "Counter" : "Metric");
  setText("monitoring-detail-target", target.label);
  setMonitoringPollIntervalInput(getMonitoringPollInterval(target.id, item.id));
}

function clearMonitoringDetail() {
  setText("monitoring-detail-name", "-");
  setText("monitoring-detail-description", "-");
  setText("monitoring-detail-type", "-");
  setText("monitoring-detail-target", "-");
  setMonitoringPollIntervalInput(DEFAULT_MONITORING_POLL_INTERVAL_S);
}

async function loadMonitoringDataPanel() {
  const panel = document.getElementById("monitoring-data-panel");
  if (!panel) {
    return;
  }

  const appliedEntries = Object.entries(state.appliedMonitoringItemIDsByTarget)
    .filter(([targetID, idSet]) => targetID && idSet instanceof Set && idSet.size > 0);

  if (!appliedEntries.length) {
    panel.innerHTML = '<p class="empty-text">Apply selected metrics to view data.</p>';
    return;
  }

  const responses = await Promise.all(appliedEntries.map(async ([targetID, idSet]) => {
    const params = new URLSearchParams();
    params.set("targetId", targetID);
    params.set("metrics", Array.from(idSet).join(","));
    params.set("pollIntervals", JSON.stringify(buildMonitoringPollIntervals(targetID, idSet)));
    const data = await fetchJSON(`/api/v1/monitoring/data?${params.toString()}`, []);
    return Array.isArray(data) ? data : [];
  }));

  const data = responses.flat();
  if (!data.length) {
    panel.innerHTML = '<p class="empty-text">No monitoring data available.</p>';
    return;
  }

  panel.innerHTML = "";
  data.forEach((targetBlock) => {
    const section = document.createElement("section");
    section.className = "monitoring-data-target";

    const heading = document.createElement("h4");
    heading.textContent = targetBlock.label || targetBlock.targetLabel || "Target";
    section.appendChild(heading);

    const table = document.createElement("table");
    table.className = "monitoring-data-table";

    const tbody = document.createElement("tbody");
    const values = Array.isArray(targetBlock.values) ? targetBlock.values : [];
    values.forEach((valueEntry) => {
      const row = document.createElement("tr");
      const key = document.createElement("td");
      key.textContent = valueEntry.label || valueEntry.id || "-";

      const value = document.createElement("td");
      value.className = "monitoring-data-value";
      value.textContent = valueEntry.value || "-";

      row.appendChild(key);
      row.appendChild(value);
      tbody.appendChild(row);
    });

    if (!values.length) {
      const row = document.createElement("tr");
      const cell = document.createElement("td");
      cell.className = "empty-cell";
      cell.colSpan = 2;
      cell.textContent = "No values for selected metrics";
      row.appendChild(cell);
      tbody.appendChild(row);
    }

    table.appendChild(tbody);
    section.appendChild(table);
    panel.appendChild(section);
  });
}

function getSelectedMonitoringSet(targetID, createIfMissing = true) {
  if (!targetID) {
    return new Set();
  }

  let selectedSet = state.selectedMonitoringItemIDsByTarget[targetID];
  if (!selectedSet && createIfMissing) {
    selectedSet = new Set();
    state.selectedMonitoringItemIDsByTarget[targetID] = selectedSet;
  }
  return selectedSet || new Set();
}

function getMonitoringPollIntervalsForTarget(targetID, createIfMissing = true) {
  if (!targetID) {
    return {};
  }

  let intervals = state.monitoringPollIntervalsByTarget[targetID];
  if (!intervals && createIfMissing) {
    intervals = {};
    state.monitoringPollIntervalsByTarget[targetID] = intervals;
  }

  return intervals || {};
}

function normalizeMonitoringPollInterval(value) {
  const parsed = Number.parseInt(String(value), 10);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return DEFAULT_MONITORING_POLL_INTERVAL_S;
  }
  return parsed;
}

function getMonitoringPollInterval(targetID, itemID, createIfMissing = true) {
  const intervals = getMonitoringPollIntervalsForTarget(targetID, createIfMissing);
  if (!intervals || !itemID) {
    return DEFAULT_MONITORING_POLL_INTERVAL_S;
  }

  const interval = intervals[itemID];
  if (!interval || interval < 1) {
    if (createIfMissing) {
      intervals[itemID] = DEFAULT_MONITORING_POLL_INTERVAL_S;
    }
    return DEFAULT_MONITORING_POLL_INTERVAL_S;
  }

  return interval;
}

function setMonitoringPollInterval(targetID, itemID, value) {
  if (!targetID || !itemID) {
    return;
  }

  const intervals = getMonitoringPollIntervalsForTarget(targetID);
  intervals[itemID] = normalizeMonitoringPollInterval(value);
}

function setMonitoringPollIntervalInput(value) {
  const input = document.getElementById("monitoring-poll-interval");
  if (!input) {
    return;
  }

  input.value = String(normalizeMonitoringPollInterval(value));
}

function buildMonitoringPollIntervals(targetID, idSet) {
  const intervals = {};
  const targetIntervals = getMonitoringPollIntervalsForTarget(targetID, false);

  Array.from(idSet).forEach((metricID) => {
    intervals[metricID] = normalizeMonitoringPollInterval(
      targetIntervals[metricID] || DEFAULT_MONITORING_POLL_INTERVAL_S
    );
  });

  return intervals;
}

function getAppliedMonitoringSet(targetID, createIfMissing = true) {
  if (!targetID) {
    return new Set();
  }

  let appliedSet = state.appliedMonitoringItemIDsByTarget[targetID];
  if (!appliedSet && createIfMissing) {
    appliedSet = new Set();
    state.appliedMonitoringItemIDsByTarget[targetID] = appliedSet;
  }
  return appliedSet || new Set();
}

function getCurrentMonitoringTargetID() {
  syncMonitoringTargetFromSelectors();
  return state.selectedMonitoringTargetID;
}

function pruneMonitoringSelections() {
  const validTargetIDs = new Set(state.monitoringTargets.map((target) => target.id));

  Object.keys(state.selectedMonitoringItemIDsByTarget).forEach((targetID) => {
    if (!validTargetIDs.has(targetID)) {
      delete state.selectedMonitoringItemIDsByTarget[targetID];
      return;
    }

    const allowedIDs = getAllowedMonitoringIDsForTarget(targetID);
    const selectedSet = state.selectedMonitoringItemIDsByTarget[targetID];
    selectedSet.forEach((itemID) => {
      if (!allowedIDs.has(itemID)) {
        selectedSet.delete(itemID);
      }
    });
  });

  Object.keys(state.appliedMonitoringItemIDsByTarget).forEach((targetID) => {
    if (!validTargetIDs.has(targetID)) {
      delete state.appliedMonitoringItemIDsByTarget[targetID];
      return;
    }

    const allowedIDs = getAllowedMonitoringIDsForTarget(targetID);
    const appliedSet = state.appliedMonitoringItemIDsByTarget[targetID];
    appliedSet.forEach((itemID) => {
      if (!allowedIDs.has(itemID)) {
        appliedSet.delete(itemID);
      }
    });
  });

  Object.keys(state.monitoringPollIntervalsByTarget).forEach((targetID) => {
    if (!validTargetIDs.has(targetID)) {
      delete state.monitoringPollIntervalsByTarget[targetID];
      return;
    }

    const allowedIDs = getAllowedMonitoringIDsForTarget(targetID);
    const targetIntervals = state.monitoringPollIntervalsByTarget[targetID];

    Object.keys(targetIntervals).forEach((itemID) => {
      if (!allowedIDs.has(itemID)) {
        delete targetIntervals[itemID];
      }
    });
  });
}

function getAllowedMonitoringIDsForTarget(targetID) {
  const target = state.monitoringTargets.find((entry) => entry.id === targetID);
  if (!target) {
    return new Set();
  }

  return new Set([...target.counters, ...target.metrics]);
}

async function loadRecentEvents() {
  const events = await fetchJSON("/api/v1/events/recent", []);
  const bodies = Array.from(document.querySelectorAll(".recent-events-card tbody"));
  bodies.forEach((tbody) => {
    tbody.innerHTML = "";
    if (!events.length) {
      tbody.innerHTML = '<tr><td colspan="4" class="empty-cell">No recent events</td></tr>';
      return;
    }
    events.forEach((entry) => {
      const row = document.createElement("tr");
      row.innerHTML = `<td>${entry.time || "-"}</td><td>${entry.type || "-"}</td><td>${entry.severity || "-"}</td><td>${entry.message || "-"}</td>`;
      tbody.appendChild(row);
    });
  });
}

async function fetchJSON(path, fallback) {
  try {
    const response = await fetch(path);
    if (!response.ok) {
      return fallback;
    }
    return await response.json();
  } catch (_) {
    return fallback;
  }
}

function renderList(container, entries, buildItem, emptyText) {
  if (!container) {
    return;
  }

  container.innerHTML = "";
  if (!entries.length) {
    const li = document.createElement("li");
    li.className = "empty-item";
    li.textContent = emptyText;
    container.appendChild(li);
    return;
  }

  entries.forEach((entry) => {
    container.appendChild(buildItem(entry));
  });
}

function clearModelDetails() {
  state.selectedModelID = "";
  setText("model-name", "-");
  setText("model-version", "-");
  setText("model-vendor", "-");
  setText("model-yang", "-");
  hideModelEditForm();
}

function toggleModelEditForm() {
  const form = document.getElementById("model-edit-form");
  if (!form) {
    return;
  }
  form.classList.toggle("hidden");
  if (!form.classList.contains("hidden")) {
    setModelEditFields();
  }
}

function hideModelEditForm() {
  const form = document.getElementById("model-edit-form");
  if (form) {
    form.classList.add("hidden");
  }
}

function setModelEditFields() {
  const name = document.getElementById("model-name");
  const version = document.getElementById("model-version");
  const vendor = document.getElementById("model-vendor");
  const yang = document.getElementById("model-yang");

  const editName = document.getElementById("model-edit-name");
  const editVersion = document.getElementById("model-edit-version");
  const editVendor = document.getElementById("model-edit-vendor");
  const editYang = document.getElementById("model-edit-yang");

  if (editName) {
    editName.value = name && name.textContent && name.textContent !== "-" ? name.textContent.trim() : "";
  }
  if (editVersion) {
    editVersion.value = version && version.textContent && version.textContent !== "-" ? version.textContent.trim() : "";
  }
  if (editVendor) {
    editVendor.value = vendor && vendor.textContent && vendor.textContent !== "-" ? vendor.textContent.trim() : "";
  }
  if (editYang) {
    editYang.value = yang && yang.textContent && yang.textContent !== "-" ? yang.textContent.trim() : "";
  }
}

function clearNodeDetails() {
  state.selectedNodeID = "";
  setText("node-name", "-");
  setText("node-type", "-");
  setText("node-ports", "-");
  setText("node-device-model", "-");
  setText("node-management-ip", "-");
  setText("node-management-protocol", "-");
  setText("node-management-port", "-");
  hideNodeEditForm();
}

function populateNodeCreateEditDeviceModelOptions() {
  const selects = [
    document.getElementById("node-create-device-model"),
    document.getElementById("node-edit-device-model"),
  ];

  const models = Array.isArray(state.deviceModels)
    ? state.deviceModels
    : [];

  selects.forEach((select) => {
    if (!select) {
      return;
    }

    const current = select.value;

    select.innerHTML = '<option value="">None</option>';

    models.forEach((model) => {
      const label = model.name || model.id || "";

      if (!label) {
        return;
      }

      const option = document.createElement("option");
      option.value = label;
      option.textContent = label;
      select.appendChild(option);
    });

    if (current) {
      const hasCurrent = models.some(
        (model) => (model.name || model.id || "") === current
      );

      if (hasCurrent) {
        select.value = current;
      }
    }
  });
}

function populateLinkNodeOptions(nodes) {
  const sourceSelects = [
    document.getElementById("link-source"),
    document.getElementById("link-edit-source"),
  ].filter(Boolean);

  const destinationSelects = [
    document.getElementById("link-destination"),
    document.getElementById("link-edit-destination"),
  ].filter(Boolean);

  if (sourceSelects.length === 0 || destinationSelects.length === 0) {
    return;
  }

  const currentValues = {
    source: sourceSelects.map((select) => select.value),
    destination: destinationSelects.map((select) => select.value),
  };

  const normalizedNodes = Array.isArray(nodes) ? nodes : [];

  const nodeNames = normalizedNodes
    .map((node) => {
      if (!node) {
        return "";
      }

      return String(node.name || node.id || "").trim();
    })
    .filter((name) => name !== "");

  const uniqueNodeNames = Array.from(new Set(nodeNames));

  sourceSelects.forEach((select) => {
    select.innerHTML = '<option value="">None</option>';

    uniqueNodeNames.forEach((name) => {
      const option = document.createElement("option");
      option.value = name;
      option.textContent = name;
      select.appendChild(option);
    });
  });

  destinationSelects.forEach((select) => {
    select.innerHTML = '<option value="">None</option>';

    uniqueNodeNames.forEach((name) => {
      const option = document.createElement("option");
      option.value = name;
      option.textContent = name;
      select.appendChild(option);
    });
  });

  sourceSelects.forEach((select, index) => {
    const previousValue = currentValues.source[index];

    if (previousValue && uniqueNodeNames.includes(previousValue)) {
      select.value = previousValue;
    }
  });

  destinationSelects.forEach((select, index) => {
    const previousValue = currentValues.destination[index];

    if (previousValue && uniqueNodeNames.includes(previousValue)) {
      select.value = previousValue;
    }
  });
}

function buildNodeCreatePayload() {
  const name = getValue("node-create-name").trim();
  const type = getValue("node-create-type").trim();
  const deviceModel = getValue("node-create-device-model").trim();
  const ipAddress = getValue("node-create-ip").trim();
  const protocol = getValue("node-create-protocol").trim();
  const managementPortRaw = getValue("node-create-port").trim();
  const userName = getValue("node-create-username").trim();

  if (!name) {
    showToast("Node name is required.");
    return null;
  }

  if (!type) {
    showToast("Node type is required.");
    return null;
  }

  const allowedTypes = new Set(["END_STATION", "BRIDGE", "BRIDGED_END_STATION"]);
  if (!allowedTypes.has(type)) {
    showToast("Node type must be END_STATION, BRIDGE, or BRIDGED_END_STATION.");
    return null;
  }

  const payload = {
    name,
    type,
  };

  if (deviceModel) {
    payload.deviceInfo = {
      deviceModel,
    };
  }

  const hasManagementInfo = ipAddress || protocol || managementPortRaw || userName;
  if (hasManagementInfo) {
    payload.managementInfo = {};
    if (ipAddress) {
      payload.managementInfo.ipAddress = ipAddress;
    }
    if (protocol) {
      payload.managementInfo.protocol = protocol;
    }
    if (userName) {
      payload.managementInfo.userName = userName;
    }
    if (managementPortRaw) {
      const managementPort = Number.parseInt(managementPortRaw, 10);
      if (Number.isNaN(managementPort) || managementPort < 1 || managementPort > 65535) {
        showToast("Management port must be between 1 and 65535.");
        return null;
      }
      payload.managementInfo.managementPort = managementPort;
    }
  }

  return payload;
}

function resetNodeCreateForm() {
  setInputValue("node-create-name", "");
  setInputValue("node-create-type", "");
  setInputValue("node-create-device-model", "");
  setInputValue("node-create-ip", "");
  setInputValue("node-create-protocol", "");
  setInputValue("node-create-port", "");
  setInputValue("node-create-username", "");
  setInputValue("node-create-password", "");
}

async function importNodePayload(payloadText) {
  let parsed;
  try {
    parsed = JSON.parse(payloadText);
  } catch (error) {
    showToast(`Invalid JSON: ${error.message}`);
    return;
  }

  const rawNode = extractRawNodeFromImport(parsed);
  if (!rawNode) {
    showToast("No valid node object found in imported file.");
    return;
  }

  const nodePayload = normalizeImportedNode(rawNode);
  if (!nodePayload) {
    showToast("Imported node JSON is missing required fields.");
    return;
  }

  const response = await callAction("addNode", { query: JSON.stringify(nodePayload) });
  showToast(response.message || (response.success ? "Node imported." : "Failed to import node."));

  if (response.success) {
    setInputValue("node-upload", "");
    await loadAllViews();
  }
}

function extractRawNodeFromImport(parsed) {
  if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
    if (parsed.node && typeof parsed.node === "object") {
      return parsed.node;
    }

    if (Array.isArray(parsed.nodes) && parsed.nodes.length > 0) {
      return parsed.nodes[0];
    }

    if (parsed.topology && typeof parsed.topology === "object") {
      if (Array.isArray(parsed.topology.nodes) && parsed.topology.nodes.length > 0) {
        return parsed.topology.nodes[0];
      }
    }

    return parsed;
  }

  if (Array.isArray(parsed) && parsed.length > 0) {
    return parsed[0];
  }

  return null;
}

async function importTopologyPayload(payloadText) {
  console.log("Importing topology payload:", payloadText);
  if (!payloadText || !payloadText.trim()) {
    showToast("Select a topology JSON file.");
    return;
  }

  // Validate that the selected file contains valid JSON.
  try {
    JSON.parse(payloadText);
  } catch (error) {
    showToast(`Invalid JSON: ${error.message}`);
    return;
  }

  const response = await callAction("uploadTopology", {
    query: payloadText,
  });

  showToast(
    response.message ||
      (response.success
        ? "Topology imported successfully."
        : "Failed to import topology.")
  );

  if (response.success) {
    setInputValue("import-topology-input", "");
    await loadAllViews();
  }
}

function normalizeImportedNode(rawNode) {
  if (!rawNode || typeof rawNode !== "object") {
    return null;
  }

  const name = firstString(rawNode.name, rawNode.id);
  const type = firstString(rawNode.type, rawNode.nodeType, rawNode.role);

  if (!name || !type) {
    return null;
  }

  const normalizedType = String(type)
    .toUpperCase()
    .replaceAll("-", "_")
    .replaceAll(" ", "_");

  const allowedTypes = new Set([
    "END_STATION",
    "BRIDGE",
    "BRIDGED_END_STATION",
  ]);

  if (!allowedTypes.has(normalizedType)) {
    return null;
  }

  const payload = {
    name,
    type: normalizedType,
  };

  // Preserve ports from imported topology JSON.
  if (Array.isArray(rawNode.ports)) {
    payload.ports = rawNode.ports
      .filter((port) => port && typeof port === "object")
      .map((port) => {
        const normalizedPort = {};

        const id = firstString(port.id);
        const portName = firstString(port.name);

        if (id) {
          normalizedPort.id = id;
        }

        if (portName) {
          normalizedPort.name = portName;
        }

        // Map imported per-port speed to the protobuf capabilities object.
        if (
          port.port_speed !== undefined &&
          port.port_speed !== null &&
          port.port_speed !== ""
        ) {
          const portSpeed = Number(port.port_speed);

          if (Number.isFinite(portSpeed)) {
            normalizedPort.capabilities = {
              ...(normalizedPort.capabilities || {}),
              port_speed: portSpeed,
            };
          }
        }

        // Preserve number of queues.
        if (
          port.number_of_queues !== undefined &&
          port.number_of_queues !== null &&
          port.number_of_queues !== ""
        ) {
          const numberOfQueues = Number.parseInt(
            String(port.number_of_queues),
            10
          );

          if (!Number.isNaN(numberOfQueues)) {
            normalizedPort.number_of_queues = numberOfQueues;
          }
        }

        return normalizedPort;
      });
  }

  const deviceInfo =
    rawNode.deviceInfo ||
    rawNode.device_info ||
    {};

  const deviceModel = firstString(
    deviceInfo.deviceModel,
    deviceInfo.device_model,
    rawNode.deviceModel,
    rawNode.device_model
  );

  if (deviceModel) {
    payload.deviceInfo = {
      deviceModel,
    };
  }

  const managementInfo =
    rawNode.managementInfo ||
    rawNode.management_info ||
    {};

  const managementPayload = {};

  const ipAddress = firstString(
    managementInfo.ipAddress,
    managementInfo.ip_address
  );

  const protocol = firstString(
    managementInfo.protocol
  );

  const userName = firstString(
    managementInfo.userName,
    managementInfo.user_name
  );

  const managementPortRaw =
    managementInfo.managementPort ??
    managementInfo.management_port;

  if (ipAddress) {
    managementPayload.ipAddress = ipAddress;
  }

  if (protocol) {
    managementPayload.protocol =
      String(protocol).toUpperCase();
  }

  if (userName) {
    managementPayload.userName = userName;
  }

  if (
    managementPortRaw !== undefined &&
    managementPortRaw !== null &&
    managementPortRaw !== ""
  ) {
    const managementPort = Number.parseInt(
      String(managementPortRaw),
      10
    );

    if (
      !Number.isNaN(managementPort) &&
      managementPort > 0 &&
      managementPort <= 65535
    ) {
      managementPayload.managementPort = managementPort;
    }
  }

  if (Object.keys(managementPayload).length > 0) {
    payload.managementInfo = managementPayload;
  }

  return payload;
}

function normalizeImportedLink(rawLink) {
  if (!rawLink || typeof rawLink !== "object") {
    return null;
  }

  const source = firstString(rawLink.source, rawLink.sourceNode, rawLink.source_node);
  const destination = firstString(rawLink.destination, rawLink.target, rawLink.targetNode, rawLink.target_node);
  if (!source || !destination || source === destination) {
    return null;
  }

  const rawBandwidth = rawLink.bandwidth ?? rawLink.bandwidth_mbps;
  let bandwidth = "";
  if (typeof rawBandwidth === "number" && Number.isFinite(rawBandwidth)) {
    bandwidth = `${rawBandwidth}`;
  } else if (typeof rawBandwidth === "string") {
    bandwidth = rawBandwidth;
  }

  return {
    source,
    destination,
    bandwidth,
  };
}

function firstString(...values) {
  for (const value of values) {
    if (typeof value === "string" && value.trim() !== "") {
      return value.trim();
    }
  }

  return "";
}

function toggleNodeEditForm() {
  const form = document.getElementById("node-edit-form");
  if (!form) {
    return;
  }
  form.classList.toggle("hidden");
  if (!form.classList.contains("hidden")) {
    setNodeEditFields();
  }
}

function hideNodeEditForm() {
  const form = document.getElementById("node-edit-form");
  if (form) {
    form.classList.add("hidden");
  }
}

function setNodeEditFields() {
  const name = document.getElementById("node-name");
  const type = document.getElementById("node-type");
  const ports = document.getElementById("node-ports");
  const deviceModel = document.getElementById("node-device-model");
  const managementIp = document.getElementById("node-management-ip");
  const managementProtocol = document.getElementById("node-management-protocol");
  const managementPort = document.getElementById("node-management-port");

  const editName = document.getElementById("node-edit-name");
  const editType = document.getElementById("node-edit-type");
  const editPorts = document.getElementById("node-edit-ports");
  const editUsername = document.getElementById("node-edit-username");
  const editPassword = document.getElementById("node-edit-password");
  const editDeviceModel = document.getElementById("node-edit-device-model");
  const editManagementIp = document.getElementById("node-edit-management-ip");
  const editManagementProtocol = document.getElementById("node-edit-management-protocol");
  const editManagementPort = document.getElementById("node-edit-management-port");

  if (editName) {
    editName.value = name && name.textContent && name.textContent !== "-" ? name.textContent.trim() : "";
  }
  if (editType) {
    editType.value = type && type.textContent && type.textContent !== "-" ? type.textContent.trim() : "";
  }
  if (editPorts) {
    editPorts.value = ports && ports.textContent && ports.textContent !== "-" ? ports.textContent.trim() : "";
  }
  if (editUsername) {
  editUsername.value = "";
}
  if (editPassword) {
    editPassword.value = "";
  }
  if (editDeviceModel) {
    const currentDeviceModel =
      deviceModel &&
      deviceModel.textContent &&
      deviceModel.textContent !== "-"
        ? deviceModel.textContent.trim()
        : "";

    // Make sure the dropdown has all device model options
    populateNodeCreateEditDeviceModelOptions();

    // Select the node's current device model
    editDeviceModel.value = currentDeviceModel;
}
  if (editManagementIp) {
    editManagementIp.value = managementIp && managementIp.textContent && managementIp.textContent !== "-" ? managementIp.textContent.trim() : "";
  }
  if (editManagementProtocol) {
    editManagementProtocol.value = managementProtocol && managementProtocol.textContent && managementProtocol.textContent !== "-" ? managementProtocol.textContent.trim() : "";
  }
  if (editManagementPort) {
    editManagementPort.value = managementPort && managementPort.textContent && managementPort.textContent !== "-" ? managementPort.textContent.trim() : "";
  }
}

function toggleLinkEditForm() {
  const form = document.getElementById("link-edit-form");

  if (!form) {
    return;
  }

  form.classList.toggle("hidden");

  if (!form.classList.contains("hidden")) {
    setLinkEditFields();
  }
}

function hideLinkEditForm() {
  const form = document.getElementById("link-edit-form");

  if (form) {
    form.classList.add("hidden");
  }
}

function setLinkEditFields() {
  const source = document.getElementById("link-detail-source");
  const destination = document.getElementById("link-detail-destination");
  const bandwidth = document.getElementById("link-detail-bandwidth");

  const editSource = document.getElementById("link-edit-source");
  const editDestination = document.getElementById("link-edit-destination");
  const editBandwidth = document.getElementById("link-edit-bandwidth");

  if (editSource) {
    editSource.value =
      source && source.textContent && source.textContent !== "-"
        ? source.textContent.trim()
        : "";
  }

  if (editDestination) {
    editDestination.value =
      destination &&
      destination.textContent &&
      destination.textContent !== "-"
        ? destination.textContent.trim()
        : "";
  }

  if (editBandwidth) {
    editBandwidth.value =
      bandwidth &&
      bandwidth.textContent &&
      bandwidth.textContent !== "-"
        ? bandwidth.textContent.trim()
        : "";
  }
}

function clearLinkDetails() {
  state.selectedLinkID = "";
  setText("link-detail-source", "-");
  setText("link-detail-destination", "-");
  setText("link-detail-bandwidth", "-");
}

function clearStreamDetails() {
  state.selectedStreamID = "";
  setText("stream-name", "-");
  setText("stream-source", "-");
  setText("stream-listeners", "-");
  setText("stream-characteristics", "-");
  resetStreamForm();
}

function getSelectedStreamListeners() {

  const listenerMultiSelect =
    document.getElementById("stream-listeners-select");

  if (!listenerMultiSelect) {
    return [];
  }

  return Array.from(
    listenerMultiSelect.querySelectorAll(
      '.multi-select-option input[type="checkbox"]:checked'
    )
  )
    .map((checkbox) => checkbox.value.trim())
    .filter((value) => value !== "");
}

function buildStreamPayload(includeSelectedID = false) {
  const talkerSelect = document.getElementById("stream-talker-select");

  return {
    id: includeSelectedID ? state.selectedStreamID : "",
    name: getValue("stream-name-input").trim(),
    talkerNodeId: talkerSelect ? talkerSelect.value.trim() : "",
    listenerNodeIds: getSelectedStreamListeners(),
    trafficType: getValue("stream-traffic-type"),
    rank: getValue("stream-rank"),
    destinationMac: "",
    sourceMac: "",
    vlanId: Number(getValue("stream-vlan-id") || 0),
    intervalNs: Number(getValue("stream-interval-ns") || 0),
    maxFrameSize: Number(getValue("stream-max-frame-size") || 0),
    maxFramesPerInterval: Number(getValue("stream-max-frames-per-interval") || 0),
    maxLatencyNs: Number(getValue("stream-max-latency-ns") || 0),
    maxJitterNs: Number(getValue("stream-max-jitter-ns") || 0),
    minTransmitOffsetNs: Number(getValue("stream-min-transmit-offset-ns") || 0),
    maxTransmitOffsetNs: Number(getValue("stream-max-transmit-offset-ns") || 0),
    numSeamlessTrees: Number(getValue("stream-num-seamless-trees") || 0),
  };
}

function selectStreamItem(item) {

  if (!item) {
    clearStreamDetails();
    return;
  }

  const stream = {

    id: item.dataset.id || "",

    name: item.dataset.name || "",

    source: item.dataset.source || "",

    listeners: item.dataset.listeners || "",

    characteristics: item.dataset.characteristics || "",

    talkerNodeId: item.dataset.talkerNodeId || "",

    listenerNodeIds:
      (item.dataset.listenerNodeIds || "")
        .split("|")
        .filter(Boolean),

    trafficType: item.dataset.trafficType || "",

    rank: item.dataset.rank || "",

    destinationMac: item.dataset.destinationMac || "",

    sourceMac: item.dataset.sourceMac || "",

    vlanId: item.dataset.vlanId || "",

    intervalNs: item.dataset.intervalNs || "",

    maxFrameSize: item.dataset.maxFrameSize || "",

    maxFramesPerInterval:
      item.dataset.maxFramesPerInterval || "",

    maxLatencyNs:
      item.dataset.maxLatencyNs || "",

    maxJitterNs:
      item.dataset.maxJitterNs || "",

    minTransmitOffsetNs:
      item.dataset.minTransmitOffsetNs || "",

    maxTransmitOffsetNs:
      item.dataset.maxTransmitOffsetNs || "",

    numSeamlessTrees:
      item.dataset.numSeamlessTrees || "",

  };

  // Selecting a stream ONLY updates the Details card.
  // It must never populate the Create Stream form.
  setStreamDetailsFromStream(stream);
}

function markSelectedStream(streamID) {
  const list = document.getElementById("stream-list");
  if (!list) {
    return;
  }

  Array.from(list.querySelectorAll("li[data-id]")).forEach((item) => {
    item.classList.toggle("selected", item.dataset.id === streamID);
  });
}

function populateStreamNodeOptions(nodes) {
  const talkerSelect = document.getElementById("stream-talker-select");
  const listenerMultiSelect = document.getElementById("stream-listeners-select");

  if (!talkerSelect || !listenerMultiSelect) {
    return;
  }

  const currentTalker = talkerSelect.value;

  const currentListeners = Array.from(
    listenerMultiSelect.querySelectorAll(
      '.multi-select-option input[type="checkbox"]:checked'
    )
  ).map((checkbox) => checkbox.value);

  const nodeNames = Array.isArray(nodes)
    ? Array.from(
        new Set(
          nodes
            .map((node) =>
              String(node && (node.name || node.id || "")).trim()
            )
            .filter((name) => name !== "")
        )
      )
    : [];

  /* -----------------------------------------------------
     Talker
     ----------------------------------------------------- */

  talkerSelect.innerHTML = "";

  const talkerPlaceholder = document.createElement("option");
  talkerPlaceholder.value = "";
  talkerPlaceholder.textContent = "Select Talker";
  talkerPlaceholder.disabled = true;
  talkerPlaceholder.selected = true;

  talkerSelect.appendChild(talkerPlaceholder);

  nodeNames.forEach((name) => {
    const option = document.createElement("option");

    option.value = name;
    option.textContent = name;

    talkerSelect.appendChild(option);
  });

  if (currentTalker && nodeNames.includes(currentTalker)) {
    talkerSelect.value = currentTalker;
  }


  /* -----------------------------------------------------
     Listeners
     ----------------------------------------------------- */

  const optionsContainer =
    listenerMultiSelect.querySelector(".multi-select-options");

  const toggleText =
    listenerMultiSelect.querySelector(".multi-select-toggle span");

  if (!optionsContainer) {
    return;
  }

  optionsContainer.innerHTML = "";

  nodeNames.forEach((name) => {
    const label = document.createElement("label");

    label.className = "multi-select-option";

    const checkbox = document.createElement("input");

    checkbox.type = "checkbox";
    checkbox.value = name;

    if (currentListeners.includes(name)) {
      checkbox.checked = true;
    }

    const text = document.createElement("span");

    text.textContent = name;

    label.appendChild(checkbox);
    label.appendChild(text);

    optionsContainer.appendChild(label);

    checkbox.addEventListener("change", () => {
      updateStreamListenerLabel(listenerMultiSelect);
    });
  });

  updateStreamListenerLabel(listenerMultiSelect);
}

function updateStreamListenerLabel(multiSelect) {
  const toggleText =
    multiSelect.querySelector(".multi-select-toggle span");

  const selected = Array.from(
    multiSelect.querySelectorAll(
      '.multi-select-option input[type="checkbox"]:checked'
    )
  );

  if (!toggleText) {
    return;
  }

  if (selected.length === 0) {
    toggleText.textContent = "Select Listeners";
  } else if (selected.length === 1) {
    toggleText.textContent = selected[0].value;
  } else {
    toggleText.textContent = `${selected.length} listeners selected`;
  }
}

function resetStreamForm() {
  // Text / numeric inputs
  setInputValue("stream-name-input", "");
  setInputValue("stream-vlan-id", "");
  setInputValue("stream-interval-ns", "");
  setInputValue("stream-max-frame-size", "");
  setInputValue("stream-max-frames-per-interval", "");
  setInputValue("stream-max-latency-ns", "");
  setInputValue("stream-max-jitter-ns", "");
  setInputValue("stream-min-transmit-offset-ns", "");
  setInputValue("stream-max-transmit-offset-ns", "");
  setInputValue("stream-num-seamless-trees", "");

  // Selects
  const trafficType = document.getElementById("stream-traffic-type");
  if (trafficType) {
    trafficType.selectedIndex = -1;
  }

  const rank = document.getElementById("stream-rank");
  if (rank) {
    rank.selectedIndex = -1;
  }

  const talkerSelect =
    document.getElementById("stream-talker-select");

  if (talkerSelect) {
    talkerSelect.value = "";
  }

  // Listener checkboxes
  const listenerMultiSelect =
    document.getElementById("stream-listeners-select");

  if (listenerMultiSelect) {
    const checkboxes =
      listenerMultiSelect.querySelectorAll(
        '.multi-select-option input[type="checkbox"]'
      );

    checkboxes.forEach((checkbox) => {
      checkbox.checked = false;
    });

    updateStreamListenerLabel(listenerMultiSelect);

    listenerMultiSelect.classList.remove("open");
  }

  // Details panel
  setText("stream-name", "-");
  setText("stream-source", "-");
  setText("stream-listeners", "-");
  setText("stream-characteristics", "-");
}

function selectStreamItem(item) {

  if (!item) {
    clearStreamDetails();
    return;
  }

  const stream = {
    id: item.dataset.id || "",
    name: item.dataset.name || "",
    source: item.dataset.source || "",
    listeners: item.dataset.listeners || "",
    characteristics: item.dataset.characteristics || "",

    talkerNodeId:
      item.dataset.talkerNodeId || "",

    listenerNodeIds:
      (item.dataset.listenerNodeIds || "")
        .split("|")
        .filter(Boolean),

    trafficType:
      item.dataset.trafficType || "",

    rank:
      item.dataset.rank || "",

    destinationMac:
      item.dataset.destinationMac || "",

    sourceMac:
      item.dataset.sourceMac || "",

    vlanId:
      item.dataset.vlanId || "",

    intervalNs:
      item.dataset.intervalNs || "",

    maxFrameSize:
      item.dataset.maxFrameSize || "",

    maxFramesPerInterval:
      item.dataset.maxFramesPerInterval || "",

    maxLatencyNs:
      item.dataset.maxLatencyNs || "",

    maxJitterNs:
      item.dataset.maxJitterNs || "",

    minTransmitOffsetNs:
      item.dataset.minTransmitOffsetNs || "",

    maxTransmitOffsetNs:
      item.dataset.maxTransmitOffsetNs || "",

    numSeamlessTrees:
      item.dataset.numSeamlessTrees || "",
  };

  // Remember which stream is selected.
  state.selectedStreamID = stream.id;

  // Update ONLY the Details card.
  setText("stream-name", stream.name || "-");
  setText(
    "stream-source",
    stream.source || stream.talkerNodeId || "-"
  );
  setText(
    "stream-listeners",
    stream.listeners ||
      (stream.listenerNodeIds.length
        ? stream.listenerNodeIds.join(", ")
        : "-")
  );
  setText(
    "stream-characteristics",
    stream.characteristics || "-"
  );

  // Highlight the selected stream.
  markSelectedStream(stream.id);
}

function setStreamDetailsFromStream(stream) {
  if (!stream) {
    clearStreamDetails();
    return;
  }

  state.selectedStreamID = stream.id || "";
  setText("stream-name", stream.name || "-");
  setText("stream-source", stream.source || stream.talkerNodeId || "-");
  setText("stream-listeners", stream.listeners || "-");
  setText("stream-characteristics", stream.characteristics || "-");
  markSelectedStream(stream.id || "");
}

function selectStreamItem(item) {
  if (!item) {
    clearStreamDetails();
    return;
  }

  const stream = {
    id: item.dataset.id || "",
    name: item.dataset.name || "",
    source: item.dataset.source || "",
    listeners: item.dataset.listeners || "",
    characteristics: item.dataset.characteristics || "",

    talkerNodeId: item.dataset.talkerNodeId || "",

    listenerNodeIds: (item.dataset.listenerNodeIds || "")
      .split("|")
      .filter(Boolean),

    trafficType: item.dataset.trafficType || "",
    rank: item.dataset.rank || "",
    destinationMac: item.dataset.destinationMac || "",
    sourceMac: item.dataset.sourceMac || "",

    vlanId: item.dataset.vlanId || "",
    intervalNs: item.dataset.intervalNs || "",
    maxFrameSize: item.dataset.maxFrameSize || "",
    maxFramesPerInterval: item.dataset.maxFramesPerInterval || "",
    maxLatencyNs: item.dataset.maxLatencyNs || "",
    maxJitterNs: item.dataset.maxJitterNs || "",
    minTransmitOffsetNs: item.dataset.minTransmitOffsetNs || "",
    maxTransmitOffsetNs: item.dataset.maxTransmitOffsetNs || "",
    numSeamlessTrees: item.dataset.numSeamlessTrees || "",
  };

  // Selecting an existing stream ONLY updates the details card.
  // It must NOT populate the Create Stream form.
  setStreamDetailsFromStream(stream);
}

function markSelectedStream(streamID) {
  const list = document.getElementById("stream-list");
  if (!list) {
    return;
  }

  Array.from(list.querySelectorAll("li[data-id]")).forEach((item) => {
    item.classList.toggle("selected", item.dataset.id === streamID);
  });
}

async function gatherPayload(action) {
  switch (action) {
    case "addNode":
      return { query: "" };
    case "editNode":
      return {
        id: state.selectedNodeID,
        name: getValue("node-edit-name"),
        type: getValue("node-edit-type"),
        state: getValue("node-edit-state"),
        ports: getValue("node-edit-ports"),
        links: getValue("node-edit-username"),
        password: getValue("node-edit-password"),
      };
    case "deleteNode":
      return { id: state.selectedNodeID };
    case "addLink":
      return {
        source: getValue("link-source"),
        destination: getValue("link-destination"),
        bandwidth: getValue("link-bandwidth"),
      };
    case "updateLink":
      return {
        id: state.selectedLinkID,
        source: getValue("link-edit-source"),
        destination: getValue("link-edit-destination"),
        bandwidth: getValue("link-edit-bandwidth"),
      };
    case "deleteLink":
      return {
    id: state.selectedLinkID
  };
    case "addStream":
      return buildStreamPayload(false);
    case "updateStream":
        showToast("Edit Stream — To be implemented.");
      return null;
    case "removeStream":
      return { id: state.selectedStreamID };
    case "uploadModel":
      return { query: await readUploadedModelFile() };
    case "editModel":
    case "deleteModel":
      return { id: state.selectedModelID };
    default:
      return {};
  }
}

async function readUploadedFile(inputId) {
  const input = document.getElementById(inputId);
  if (!input || !input.files || input.files.length === 0) {
    return "";
  }

  try {
    return await input.files[0].text();
  } catch (error) {
    showToast(`Failed to read file: ${error.message}`);
    return "";
  }
}

async function readUploadedModelFile() {
  return readUploadedFile("device-model-upload");
}

async function callAction(action, payload) {
  const mapping = {
    refreshData: { method: "POST", path: "/api/v1/dashboard/refresh" },
    uploadModel: { method: "POST", path: "/api/v1/device-models/upload" },
    editModel: { method: "PATCH", path: `/api/v1/device-models/${payload.id || "selected"}` },
    deleteModel: { method: "DELETE", path: `/api/v1/device-models/${payload.id || "selected"}` },
    uploadTopology: { method: "POST", path: "/api/v1/topology/upload" },
    addNode: { method: "POST", path: "/api/v1/nodes" },
    editNode: { method: "PATCH", path: `/api/v1/nodes/${payload.id || "selected"}` },
    deleteNode: { method: "DELETE", path: `/api/v1/nodes/${payload.id || "selected"}` },
    addLink: { method: "POST", path: "/api/v1/links" },
    updateLink: { method: "PATCH", path: `/api/v1/links/${payload.id || "selected"}` },
    deleteLink: { method: "DELETE", path: `/api/v1/links/${payload.id || "selected"}` },
    addStream: { method: "POST", path: "/api/v1/streams" },
    updateStream: { method: "PATCH", path: `/api/v1/streams/${payload.id || "selected"}` },
    removeStream: { method: "DELETE", path: `/api/v1/streams/${payload.id || "selected"}` },
  };

  const target = mapping[action];
  if (!target) {
    return { success: false, message: `Unsupported action: ${action}` };
  }

  const missingSelection =
    ((action === "editModel" || action === "deleteModel") && !state.selectedModelID) ||
    ((action === "editNode" || action === "deleteNode") && !state.selectedNodeID) ||
    ((action === "updateLink" || action === "deleteLink") && !state.selectedLinkID) ||
    (action === "updateStream" && !state.selectedStreamID) ||
    (action === "removeStream" && !state.selectedStreamID);

  if (missingSelection) {
    return { success: false, message: "Select an item first." };
  }

  try {
    const requestInit = {
      method: target.method,
      headers: { "Content-Type": "application/json" },
    };

    if (target.method !== "DELETE") {
      requestInit.body = JSON.stringify(payload || {});
    }

    const response = await fetch(target.path, requestInit);
    if (!response.ok) {
      return { success: false, message: `Action ${action} failed with status ${response.status}` };
    }

    return await response.json();
  } catch (error) {
    return { success: false, message: `Action ${action} failed: ${error.message}` };
  }
}

function showToast(message) {
  if (!toast) {
    return;
  }

  toast.textContent = message;
  toast.classList.add("show");
  window.setTimeout(() => {
    toast.classList.remove("show");
  }, 2200);
}

function getValue(id) {
  const element = document.getElementById(id);
  if (!element) {
    return "";
  }
  return element.value || "";
}

function setInputValue(id, value) {
  const element = document.getElementById(id);
  if (element) {
    element.value = value;
  }
}

function setText(id, value) {
  const element = document.getElementById(id);
  if (element) {
    element.textContent = value;
  }
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

