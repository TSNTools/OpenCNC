package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	rtMonitoringState := stub.NewMonitoringState()

	backend := stub.NewBackend(rtMonitoringState)

	service := app.NewService(backend, backend)

	router := httptransport.NewRouter(cfg, service)

	metricsConsumer := stub.NewMetricsConsumer(rtMonitoringState)
	defer metricsConsumer.Close()

	go func() {
		if err := metricsConsumer.Run(ctx); err != nil {
			log.Printf("metrics consumer stopped: %v", err)
		}
	}()

	log.Printf(
		"GUI service listening on :%s and serving %s",
		cfg.Port,
		cfg.StaticDir,
	)

	if err := router.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
