package main

import (
	"log"

	"OpenCNC_config_service/gui_service/internal/adapters/stub"
	"OpenCNC_config_service/gui_service/internal/app"
	"OpenCNC_config_service/gui_service/internal/config"
	httptransport "OpenCNC_config_service/gui_service/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	backend := stub.NewBackend()
	service := app.NewService(backend, backend)
	router := httptransport.NewRouter(cfg, service)

	log.Printf("GUI service listening on :%s and serving %s", cfg.Port, cfg.StaticDir)
	if err := router.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
