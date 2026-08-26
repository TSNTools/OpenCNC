package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"OpenCNC/gui_service/internal/adapters/stub"
	"OpenCNC/gui_service/internal/app"
	"OpenCNC/gui_service/internal/config"
	httptransport "OpenCNC/gui_service/internal/transport/http"
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

	go func() {
		if err := router.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server failed: %v", err)
			stop()
		}
	}()

	<-ctx.Done()

	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := router.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown failed: %v", err)
	}

	log.Println("GUI service stopped")
}
