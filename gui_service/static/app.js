const navItems = Array.from(document.querySelectorAll(".nav-item"));
const screenNodes = Array.from(document.querySelectorAll(".screen"));
const screenTitle = document.getElementById("screen-title");
const toast = document.getElementById("toast");

initializeNavigation();
initializeActionButtons();
initializeListSelection();
initializeSelectActions();

function initializeNavigation() {
  navItems.forEach((item) => {
    item.addEventListener("click", () => {
      const screen = item.dataset.screen;
      navItems.forEach((entry) => entry.classList.toggle("is-active", entry === item));
      screenNodes.forEach((entry) => entry.classList.toggle("is-visible", entry.id === `screen-${screen}`));
      screenTitle.textContent = item.textContent.trim();
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

      const payload = gatherPayload(action);
      const response = await callAction(action, payload);
      showToast(response.message || `${action} completed`);
    });
  });
}

function initializeSelectActions() {
  const logsFilter = document.getElementById("logs-filter");
  const logsOrder = document.getElementById("logs-order");

  if (logsFilter) {
    logsFilter.addEventListener("change", async () => {
      const response = await callAction("filterLogs", { value: logsFilter.value });
      showToast(response.message || "Logs filtered");
    });
  }

  if (logsOrder) {
    logsOrder.addEventListener("change", async () => {
      const response = await callAction("orderLogs", { value: logsOrder.value });
      showToast(response.message || "Log ordering changed");
    });
  }
}

function initializeListSelection() {
  const selectableLists = Array.from(document.querySelectorAll(".selectable-list"));
  selectableLists.forEach((list) => {
    list.addEventListener("click", (event) => {
      const clicked = event.target.closest("li");
      if (!clicked) {
        return;
      }
      Array.from(list.querySelectorAll("li")).forEach((item) => item.classList.remove("selected"));
      clicked.classList.add("selected");
    });
  });
}

function gatherPayload(action) {
  switch (action) {
    case "addNode":
      return { query: getValue("node-search") };
    case "addLink":
      return {
        source: getValue("link-source"),
        destination: getValue("link-destination"),
        bandwidth: getValue("link-bandwidth"),
      };
    case "addStream":
      return { query: getValue("stream-search") };
    case "uploadModel":
      return { query: getValue("device-model-search") };
    default:
      return {};
  }
}

async function callAction(action, payload) {
  try {
    const response = await fetch(`/api/actions/${action}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload || {}),
    });

    if (!response.ok) {
      return { message: `Action ${action} failed with status ${response.status}` };
    }

    return await response.json();
  } catch (error) {
    return { message: `Action ${action} failed: ${error.message}` };
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