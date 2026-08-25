// -------------------- Upload JSON --------------------
async function upload(endpoint) {
    const fileInput = document.getElementById("fileInput");
    const responseBox = document.getElementById("responseUpload");
    const message = document.getElementById("messageUpload");
    message.textContent = '';
    responseBox.value = '';

    if (!fileInput.files.length) { alert("Please select a JSON file first."); return; }
    const file = fileInput.files[0];
    let text;
    try { text = await file.text(); } 
    catch (err) { message.textContent = "Error reading file: " + err; message.className = "error"; return; }

    try {
        const r = await fetch(endpoint, { method: "POST", headers: { "Content-Type": "application/json" }, body: text });
        const bodyText = await r.text();
        let jsonResponse;
        try { jsonResponse = JSON.parse(bodyText); } catch { jsonResponse = { status: r.status, text: bodyText }; }
        responseBox.value = JSON.stringify(jsonResponse, null, 2);
        message.textContent = r.ok ? "Upload successful!" : "Upload failed!";
        message.className = r.ok ? "success" : "error";
    } catch (err) {
        responseBox.value = "Error connecting to server:\n" + err;
        message.textContent = "Failed to upload to server.";
        message.className = "error";
    }
}

async function uploadWithWarning(endpoint) {
    const confirmed = confirm(
        "⚠️ WARNING: You are about to upload an extended topology.\n" +
        "Existing nodes/links may be overwritten.\nProceed?"
    );
    if (!confirmed) return;
    await upload(endpoint);
}

// -------------------- Fetch and Render Data --------------------
async function getData(endpoint, tableId) {
    const tablesContainer = document.getElementById("tablesContainer");
    const message = document.getElementById("messageGet");
    message.textContent = '';

    // Clear all previous tables
    tablesContainer.innerHTML = '';

    try {
        const r = await fetch(endpoint);
        const bodyText = await r.text();
        let data;
        try { data = JSON.parse(bodyText); } catch { data = bodyText; }

        if (!r.ok) { 
            message.textContent = "Failed to fetch data"; 
            message.className = "error"; 
            return; 
        }

        message.textContent = "Data fetched successfully!"; 
        message.className = "success";

        if (tableId === "topologyTable") {
            renderTopologyTable(data, tablesContainer);
        } else if (tableId === "deviceModelsTable") {
            renderDeviceModelsTable(data, tablesContainer);
        } else if (Array.isArray(data)) {
            renderTable(data, tableId, tablesContainer);
        } else if (typeof data === 'object') {
            renderTable([data], tableId, tablesContainer);
        } else {
            tablesContainer.textContent = data;
        }

    } catch (err) {
        tablesContainer.textContent = "Error connecting to server: " + err;
        message.textContent = "Failed to fetch data.";
        message.className = "error";
    }
}

// -------------------- Render Table --------------------
function renderTable(dataArray, tableId, container) {
    const table = document.createElement("table");
    table.id = tableId;
    const thead = document.createElement("thead");
    const tbody = document.createElement("tbody");

    const columns = new Set();
    dataArray.forEach(item => Object.keys(item).forEach(k => columns.add(k)));
    const cols = Array.from(columns);

    const trHead = document.createElement("tr");
    cols.forEach(c => {
        const th = document.createElement("th"); th.textContent = c; trHead.appendChild(th);
    });
    thead.appendChild(trHead);

    dataArray.forEach(item => {
        const tr = document.createElement("tr");
        cols.forEach(c => {
            const td = document.createElement("td");
            let val = item[c];
            if (typeof val === 'object') val = JSON.stringify(val, null, 2);
            td.textContent = val;
            tr.appendChild(td);
        });
        tbody.appendChild(tr);
    });

    table.appendChild(thead);
    table.appendChild(tbody);
    container.appendChild(table);
}

// -------------------- Render Topology --------------------
function renderTopologyTable(topo, container) {
    let html = "<table border='1' style='width:100%; border-collapse:collapse;'><tr><th>Name</th><th>Type</th><th>Ports</th></tr>";
    topo.nodes.forEach(node => {
        const nodeType = node.type;
        const ports = node.ports.map(p => `${p.name} (${p.number_of_queues} queues, ${p.capabilities.port_speed} Mbps)`).join("<br>");
        html += `<tr><td>${node.name}</td><td>${nodeType}</td><td>${ports}</td></tr>`;
    });
    html += "</table>";
    container.appendChild(createDivWithHtml(html));
}

// -------------------- Render Device Models --------------------
function renderDeviceModelsTable(modelsData, container) {
    const models = modelsData.models || modelsData; // handle fallback
    if (!models.length) {
        container.appendChild(createDivWithHtml("No device models found."));
        return;
    }

    let html = "<table border='1' style='width:100%; border-collapse:collapse;'><tr><th>Model Name</th><th>Yang Files</th></tr>";
    models.forEach(model => {
        const modelName = model["model-name"] || model["Name"] || "Unknown";
        const yangFiles = (model["yang-files"] || model.YangFiles || []).map(f => {
            const fileName = f["file-name"] || f.Name || "Unknown";
            const revision = f["file-revision"] || f.Revision || "";
            return `${fileName} (${revision})`;
        }).join("<br>");
        html += `<tr><td>${modelName}</td><td>${yangFiles}</td></tr>`;
    });
    html += "</table>";
    container.appendChild(createDivWithHtml(html));
}

// Helper to wrap HTML in a div
function createDivWithHtml(innerHtml) {
    const div = document.createElement("div");
    div.innerHTML = innerHtml;
    return div;
}
