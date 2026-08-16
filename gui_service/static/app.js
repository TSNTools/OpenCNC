const navItems = Array.from(document.querySelectorAll(".nav-item"));
const screenNodes = Array.from(document.querySelectorAll(".screen"));
const screenTitle = document.getElementById("screen-title");
const toast = document.getElementById("toast");

const state = {
  selectedModelID: "",
  selectedNodeID: "",
  selectedLinkID: "",
  selectedStreamID: "",
  monitoringCounters: [],
  monitoringMetrics: [],
  monitoringTargets: [],
  selectedMonitoringTargetID: "",
  activeMonitoringItemID: "",
  selectedMonitoringItemIDsByTarget: {},
  appliedMonitoringItemIDsByTarget: {},
};

initializeNavigation();
initializeActionButtons();
initializeListSelection();
initializeMonitoringUI();
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

      loadView(screen);
    });
  });
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

      const payload = {
        id: state.selectedNodeID,
        name: getValue("node-edit-name"),
        type: getValue("node-edit-type"),
        state: getValue("node-edit-state"),
        ports: getValue("node-edit-ports"),
        links: getValue("node-edit-links"),
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

  const addNodeUploadButton = document.getElementById("add-node-upload-btn");
  const addNodeUploadInput = document.getElementById("add-node-upload-input");

  if (addNodeUploadButton && addNodeUploadInput) {
    addNodeUploadButton.addEventListener("click", () => {
      addNodeUploadInput.value = "";
      addNodeUploadInput.click();
    });

    addNodeUploadInput.addEventListener("change", async () => {
      const nodePayload = await readUploadedFile("add-node-upload-input");
      if (!nodePayload) {
        showToast("Select a JSON node description file.");
        return;
      }

      const response = await callAction("addNode", { query: nodePayload });
      showToast(response.message || "addNode completed");

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
    setText("node-state", item.dataset.state || "-");
    setText("node-ports", item.dataset.ports || "-");
    setText("node-links", item.dataset.links || "-");
    setNodeEditFields();
  });

  bindSelectableList("link-list", (item) => {
    state.selectedLinkID = item.dataset.id || "";
    setText("link-detail-source", item.dataset.source || "-");
    setText("link-detail-destination", item.dataset.destination || "-");
    setText("link-detail-state", item.dataset.state || "-");
    setText("link-detail-bandwidth", item.dataset.bandwidth || "-");
  });

  bindSelectableList("stream-list", (item) => {
    state.selectedStreamID = item.dataset.id || "";
    setText("stream-name", item.dataset.name || "-");
    setText("stream-source", item.dataset.source || "-");
    setText("stream-listeners", item.dataset.listeners || "-");
    setText("stream-characteristics", item.dataset.characteristics || "-");
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

    case "topology-overview":
      await Promise.all([
        loadNodes(),
        loadLinks(),
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
  clearModelDetails();
}

async function loadNodes() {
  const nodes = await fetchJSON("/api/v1/nodes", []);
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
    return li;
  }, "No nodes available");
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
    li.dataset.state = link.state || "";
    li.dataset.bandwidth = link.bandwidth || "";
    return li;
  }, "No links available");
  clearLinkDetails();
}

async function loadStreams() {
  const streams = await fetchJSON("/api/v1/streams", []);
  const list = document.getElementById("stream-list");
  renderList(list, streams, (stream) => {
    const li = document.createElement("li");
    li.textContent = stream.name || stream.id;
    li.dataset.id = stream.id || "";
    li.dataset.name = stream.name || "";
    li.dataset.source = stream.source || "";
    li.dataset.listeners = stream.listeners || "";
    li.dataset.characteristics = stream.characteristics || "";
    return li;
  }, "No streams available");
  clearStreamDetails();
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
      if (!id || !node || !port) {
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
    li.textContent = item.label;
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
}

function clearMonitoringDetail() {
  setText("monitoring-detail-name", "-");
  setText("monitoring-detail-description", "-");
  setText("monitoring-detail-type", "-");
  setText("monitoring-detail-target", "-");
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
  setText("node-state", "-");
  setText("node-ports", "-");
  setText("node-links", "-");
  hideNodeEditForm();
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
  const stateValue = document.getElementById("node-state");
  const ports = document.getElementById("node-ports");
  const links = document.getElementById("node-links");

  const editName = document.getElementById("node-edit-name");
  const editType = document.getElementById("node-edit-type");
  const editState = document.getElementById("node-edit-state");
  const editPorts = document.getElementById("node-edit-ports");
  const editLinks = document.getElementById("node-edit-links");

  if (editName) {
    editName.value = name && name.textContent && name.textContent !== "-" ? name.textContent.trim() : "";
  }
  if (editType) {
    editType.value = type && type.textContent && type.textContent !== "-" ? type.textContent.trim() : "";
  }
  if (editState) {
    editState.value = stateValue && stateValue.textContent && stateValue.textContent !== "-" ? stateValue.textContent.trim() : "";
  }
  if (editPorts) {
    editPorts.value = ports && ports.textContent && ports.textContent !== "-" ? ports.textContent.trim() : "";
  }
  if (editLinks) {
    editLinks.value = links && links.textContent && links.textContent !== "-" ? links.textContent.trim() : "";
  }
}

function clearLinkDetails() {
  state.selectedLinkID = "";
  setText("link-detail-source", "-");
  setText("link-detail-destination", "-");
  setText("link-detail-state", "-");
  setText("link-detail-bandwidth", "-");
}

function clearStreamDetails() {
  state.selectedStreamID = "";
  setText("stream-name", "-");
  setText("stream-source", "-");
  setText("stream-listeners", "-");
  setText("stream-characteristics", "-");
}

async function gatherPayload(action) {
  switch (action) {
    case "addNode":
      return { query: getValue("node-search") };
    case "editNode":
      return {
        id: state.selectedNodeID,
        name: getValue("node-edit-name"),
        type: getValue("node-edit-type"),
        state: getValue("node-edit-state"),
        ports: getValue("node-edit-ports"),
        links: getValue("node-edit-links"),
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
    case "deleteLink":
      return { id: state.selectedLinkID };
    case "addStream":
      return { query: getValue("stream-search") };
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
    addNode: { method: "POST", path: "/api/v1/nodes" },
    editNode: { method: "PATCH", path: `/api/v1/nodes/${payload.id || "selected"}` },
    deleteNode: { method: "DELETE", path: `/api/v1/nodes/${payload.id || "selected"}` },
    addLink: { method: "POST", path: "/api/v1/links" },
    updateLink: { method: "PATCH", path: `/api/v1/links/${payload.id || "selected"}` },
    deleteLink: { method: "DELETE", path: `/api/v1/links/${payload.id || "selected"}` },
    addStream: { method: "POST", path: "/api/v1/streams" },
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