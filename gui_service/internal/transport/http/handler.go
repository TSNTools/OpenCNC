package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"OpenCNC_config_service/gui_service/internal/app"
	"OpenCNC_config_service/gui_service/internal/domain"
)

type Handler struct {
	service *app.Service
}

type simpleInput struct {
	Query       string `json:"query"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Bandwidth   string `json:"bandwidth"`
	Severity    string `json:"severity"`
	OrderBy     string `json:"orderBy"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	State       string `json:"state"`
	Ports       string `json:"ports"`
	Links       string `json:"links"`
	Version     string `json:"version"`
	Vendor      string `json:"vendor"`
	Yang        string `json:"yang"`
}

func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	data, err := h.service.GetDashboard(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) RefreshDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	result, err := h.service.RefreshData(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) DeviceModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	models, err := h.service.GetDeviceModels(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, models)
}

func (h *Handler) MonitoringCounters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	counters, err := h.service.GetMonitoringCounters(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, counters)
}

func (h *Handler) MonitoringMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	metrics, err := h.service.GetMonitoringMetrics(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *Handler) MonitoringTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	targets, err := h.service.GetMonitoringTargets(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, targets)
}

func (h *Handler) MonitoringData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	rawMetrics := strings.TrimSpace(r.URL.Query().Get("metrics"))
	metricIDs := []string{}
	if rawMetrics != "" {
		for _, entry := range strings.Split(rawMetrics, ",") {
			trimmed := strings.TrimSpace(entry)
			if trimmed != "" {
				metricIDs = append(metricIDs, trimmed)
			}
		}
	}
	fmt.Println("Query:", r.URL.Query())
	query := domain.MonitoringDataQuery{
		Node:      strings.TrimSpace(r.URL.Query().Get("node")),
		TargetID:  strings.TrimSpace(r.URL.Query().Get("targetId")),
		MetricIDs: metricIDs,
	}

	data, err := h.service.GetMonitoringData(r.Context(), query)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) UploadModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	input := decodeInput(r)
	result, err := h.service.UploadModel(r.Context(), input.Query)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ModelByID(w http.ResponseWriter, r *http.Request) {
	modelID := strings.TrimPrefix(r.URL.Path, "/api/v1/device-models/")
	if modelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing model id"})
		return
	}

	switch r.Method {
	case http.MethodPatch:
		input := decodeInput(r)
		result, err := h.service.EditModel(r.Context(), modelID, input.Name, input.Version, input.Vendor, input.Yang)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodDelete:
		result, err := h.service.DeleteModel(r.Context(), modelID)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) UploadTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	input := decodeInput(r)

	result, err := h.service.UploadTopology(r.Context(), input.Query)
	if err != nil {
		internalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
func (h *Handler) Nodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		nodes, err := h.service.GetNodes(r.Context())
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, nodes)
	case http.MethodPost:
		input := decodeInput(r)
		result, err := h.service.AddNode(r.Context(), input.Query)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) NodeByID(w http.ResponseWriter, r *http.Request) {
	nodeID := strings.TrimPrefix(r.URL.Path, "/api/v1/nodes/")
	if nodeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing node id"})
		return
	}

	switch r.Method {
	case http.MethodPatch:
		input := decodeInput(r)
		result, err := h.service.EditNode(r.Context(), nodeID, input.Name, input.Type, input.State, input.Ports, input.Links)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodDelete:
		result, err := h.service.DeleteNode(r.Context(), nodeID)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) Links(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		links, err := h.service.GetLinks(r.Context())
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, links)
	case http.MethodPost:
		input := decodeInput(r)
		result, err := h.service.AddLink(r.Context(), input.Source, input.Destination, input.Bandwidth)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) LinkByID(w http.ResponseWriter, r *http.Request) {
	linkID := strings.TrimPrefix(r.URL.Path, "/api/v1/links/")
	if linkID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing link id"})
		return
	}

	switch r.Method {
	case http.MethodPatch:
		result, err := h.service.UpdateLink(r.Context(), linkID)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodDelete:
		result, err := h.service.DeleteLink(r.Context(), linkID)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) Streams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		streams, err := h.service.GetStreams(r.Context())
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, streams)
	case http.MethodPost:
		input := decodeInput(r)
		result, err := h.service.AddStream(r.Context(), input.Query)
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) StreamByID(w http.ResponseWriter, r *http.Request) {
	streamID := strings.TrimPrefix(r.URL.Path, "/api/v1/streams/")
	if streamID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing stream id"})
		return
	}

	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}

	result, err := h.service.RemoveStream(r.Context(), streamID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Logs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	query := domain.LogsQuery{
		Severity: r.URL.Query().Get("severity"),
		OrderBy:  r.URL.Query().Get("orderBy"),
	}

	logs, err := h.service.GetLogs(r.Context(), query)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (h *Handler) FilterLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	input := decodeInput(r)
	result, err := h.service.FilterLogs(r.Context(), input.Severity)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) OrderLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	input := decodeInput(r)
	result, err := h.service.OrderLogs(r.Context(), input.OrderBy)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RecentEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	events, err := h.service.GetRecentEvents(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func decodeInput(r *http.Request) simpleInput {
	input := simpleInput{}
	if r.Body == nil {
		return input
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	return input
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func internalError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}
