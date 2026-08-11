package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	staticDir, err := findStaticDir()
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("GUI_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/actions/", handleAction)
	mux.HandleFunc("/api/health", handleHealth)
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fileServer)

	log.Printf("GUI service listening on :%s and serving %s", port, staticDir)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

type actionResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Action  string            `json:"action"`
	Data    map[string]string `json:"data"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	action := strings.TrimPrefix(r.URL.Path, "/api/actions/")
	if action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing action"})
		return
	}

	var reqPayload map[string]interface{}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&reqPayload)
	}

	log.Printf("[GUI] function triggered: %s payload=%v", action, reqPayload)

	resp, ok := stubAction(action)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown action"})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func stubAction(action string) (actionResponse, bool) {
	responses := map[string]actionResponse{
		"refreshData": {
			Success: true,
			Action:  "refreshData",
			Message: "refreshData executed (stub)",
			Data:    map[string]string{"result": "dashboard data refreshed"},
		},
		"uploadModel": {
			Success: true,
			Action:  "uploadModel",
			Message: "uploadModel executed (stub)",
			Data:    map[string]string{"modelId": "model-stub-001"},
		},
		"addNode": {
			Success: true,
			Action:  "addNode",
			Message: "addNode executed (stub)",
			Data:    map[string]string{"nodeId": "node-stub-001"},
		},
		"editNode": {
			Success: true,
			Action:  "editNode",
			Message: "editNode executed (stub)",
			Data:    map[string]string{"nodeId": "node-stub-001"},
		},
		"deleteNode": {
			Success: true,
			Action:  "deleteNode",
			Message: "deleteNode executed (stub)",
			Data:    map[string]string{"nodeId": "node-stub-001"},
		},
		"addLink": {
			Success: true,
			Action:  "addLink",
			Message: "addLink executed (stub)",
			Data:    map[string]string{"linkId": "link-stub-001"},
		},
		"updateLink": {
			Success: true,
			Action:  "updateLink",
			Message: "updateLink executed (stub)",
			Data:    map[string]string{"linkId": "link-stub-001"},
		},
		"deleteLink": {
			Success: true,
			Action:  "deleteLink",
			Message: "deleteLink executed (stub)",
			Data:    map[string]string{"linkId": "link-stub-001"},
		},
		"addStream": {
			Success: true,
			Action:  "addStream",
			Message: "addStream executed (stub)",
			Data:    map[string]string{"streamId": "stream-stub-001"},
		},
		"removeStream": {
			Success: true,
			Action:  "removeStream",
			Message: "removeStream executed (stub)",
			Data:    map[string]string{"streamId": "stream-stub-001"},
		},
		"filterLogs": {
			Success: true,
			Action:  "filterLogs",
			Message: "filterLogs executed (stub)",
			Data:    map[string]string{"result": "filtered logs"},
		},
		"orderLogs": {
			Success: true,
			Action:  "orderLogs",
			Message: "orderLogs executed (stub)",
			Data:    map[string]string{"result": "ordered logs"},
		},
	}

	resp, ok := responses[action]
	return resp, ok
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func findStaticDir() (string, error) {
	candidates := []string{
		"gui_service/frontend/dist",
		"frontend/dist",
		filepath.Join("..", "frontend", "dist"),
		"gui_service/static",
		"static",
		filepath.Join("..", "static"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", os.ErrNotExist
}
