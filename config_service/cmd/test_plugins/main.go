// go run .

package main

import (
	"OpenCNC/common/observability"
	"context"
	"log"
	"time"
)

func main() {
	obsClient, err := observability.NewFromEnv("config-service")
	if err != nil {
		log.Fatalf("Observability init failed: %v", err)
	}
	if obsClient != nil {
		defer func() {
			_ = obsClient.Close()
		}()

		startupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = obsClient.EmitHealthStarted(startupCtx, "config-service-startup", "config-service started")
	}
	//TestNetconfProtocol()
	// Run all plugin tests here
	TestVlanPlugin_tttech()
	//TestPriorityPlugin_tttech()
	//TestQbvPlugin()
	// TestQbvPlugin_tttech()

}
