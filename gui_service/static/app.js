const navItems = Array.from(document.querySelectorAll(".nav-item"));
const screenNodes = Array.from(document.querySelectorAll(".screen"));
const screenTitle = document.getElementById("screen-title");
const toast = document.getElementById("toast");

const state = {
  selectedModelID: "",
  selectedNodeID: "",
  selectedLinkID: "",
  selectedStreamID: "",
};

initializeNavigation();
initializeActionButtons();
initializeListSelection();
initializeSelectActions();
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

function initializeSelectActions() {
  const logsFilter = document.getElementById("logs-filter");
  const logsOrder = document.getElementById("logs-order");

  if (logsFilter) {
    logsFilter.addEventListener("change", async () => {
      const response = await callAction("filterLogs", { severity: logsFilter.value });
      showToast(response.message || "Logs filtered");
      await loadLogs();
    });
  }

  if (logsOrder) {
    logsOrder.addEventListener("change", async () => {
      const response = await callAction("orderLogs", { orderBy: logsOrder.value });
      showToast(response.message || "Log ordering changed");
      await loadLogs();
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
    loadLogs(),
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

    case "logs":
      await loadLogs();
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

async function loadLogs() {
  const severity = getValue("logs-filter");
  const orderBy = getValue("logs-order");
  const params = new URLSearchParams();
  if (severity && severity !== "all") {
    params.set("severity", severity);
  }
  if (orderBy && orderBy !== "time") {
    params.set("orderBy", orderBy);
  }

  const suffix = params.toString() ? `?${params.toString()}` : "";
  const logs = await fetchJSON(`/api/v1/logs${suffix}`, []);
  const tbody = document.getElementById("internal-logs-body");
  tbody.innerHTML = "";

  if (!logs.length) {
    tbody.innerHTML = '<tr><td colspan="4" class="empty-cell">No internal logs</td></tr>';
    clearLogDetails();
    return;
  }

  logs.forEach((entry) => {
    const row = document.createElement("tr");
    row.innerHTML = `<td>${entry.time || "-"}</td><td>${entry.type || "-"}</td><td>${entry.severity || "-"}</td><td>${entry.message || "-"}</td>`;
    row.addEventListener("click", () => {
      setText("log-time", entry.time || "-");
      setText("log-severity", entry.severity || "-");
      setText("log-topic", entry.topic || "-");
      setText("log-correlation-id", entry.correlationId || "-");
      setText("log-message", entry.message || "-");
    });
    tbody.appendChild(row);
  });

  clearLogDetails();
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

function clearLogDetails() {
  setText("log-time", "-");
  setText("log-severity", "-");
  setText("log-topic", "-");
  setText("log-correlation-id", "-");
  setText("log-message", "-");
}

async function gatherPayload(action) {
  switch (action) {
    case "addNode":
      return { query: getValue("node-search") };
    case "editNode":
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
    case "filterLogs":
      return { severity: getValue("logs-filter") };
    case "orderLogs":
      return { orderBy: getValue("logs-order") };
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
    filterLogs: { method: "POST", path: "/api/v1/logs/filter" },
    orderLogs: { method: "POST", path: "/api/v1/logs/order" },
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