# GUI Service Prototype

This directory contains a separate GUI prototype service for OpenCNC.

Current scope:
- static multi-screen prototype for the CNC operator workflow
- dashboard-first layout with topology and traffic summary
- dedicated screens for device models, nodes, links, streams, logs, and settings
- simple Go server so the UI can run without Node.js
- React frontend scaffold ready for Node/Vite once frontend tooling is installed
- SVG screen sketches for each main screen under `gui_service/mockups`
- action buttons wired to Go stub handlers for backend integration points

Run from the repository root:

```bash
go run ./gui_service/cmd
```

Then open `http://localhost:8080`.

Notes:
- the current UI uses mocked frontend data
- the store connection is intentionally hidden from the GUI
- this is a good base to later replace with a React frontend once Node tooling is available
- all GUI action buttons call `/api/actions/<functionName>`
- each action prints its function name in the GUI service CLI and returns a hardcoded JSON response

React frontend:

```bash
cd gui_service/frontend
npm install
npm run dev
```

React build output:
- after `npm run build`, the Go server will automatically prefer `gui_service/frontend/dist`
- until then, it continues to serve the existing static prototype