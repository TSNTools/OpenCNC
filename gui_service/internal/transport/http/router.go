package http

import (
	"log"
	"net/http"
	"strings"

	"OpenCNC_config_service/gui_service/internal/app"
	"OpenCNC_config_service/gui_service/internal/config"
)

type Router struct {
	server *http.Server
}

func NewRouter(cfg config.Config, service *app.Service) *Router {
	h := NewHandler(service)
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", h.Health)
	mux.HandleFunc("/api/v1/dashboard", h.Dashboard)
	mux.HandleFunc("/api/v1/dashboard/refresh", h.RefreshDashboard)
	mux.HandleFunc("/api/v1/device-models", h.DeviceModels)
	mux.HandleFunc("/api/v1/device-models/upload", h.UploadModel)
	mux.HandleFunc("/api/v1/device-models/", h.ModelByID)
	mux.HandleFunc("/api/v1/nodes", h.Nodes)
	mux.HandleFunc("/api/v1/nodes/", h.NodeByID)
	mux.HandleFunc("/api/v1/links", h.Links)
	mux.HandleFunc("/api/v1/links/", h.LinkByID)
	mux.HandleFunc("/api/v1/streams", h.Streams)
	mux.HandleFunc("/api/v1/streams/", h.StreamByID)
	mux.HandleFunc("/api/v1/logs", h.Logs)
	mux.HandleFunc("/api/v1/logs/filter", h.FilterLogs)
	mux.HandleFunc("/api/v1/logs/order", h.OrderLogs)
	mux.HandleFunc("/api/v1/events/recent", h.RecentEvents)

	staticHandler := http.FileServer(http.Dir(cfg.StaticDir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		staticHandler.ServeHTTP(w, r)
	}))

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: loggingMiddleware(mux),
	}

	return &Router{server: server}
}

func (r *Router) ListenAndServe() error {
	return r.server.ListenAndServe()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[GUI] http %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
