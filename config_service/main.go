package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"os"
	"time"

	"OpenCNC_config_service/common/observability"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	service "OpenCNC_config_service/common/structures/service"
	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/config_service/pkg/engine" // Your wrapper implementing GNMIService
	"OpenCNC_config_service/config_service/pkg/plugins"
	"OpenCNC_config_service/config_service/pkg/protocolbackends"
	// Official gNMI package
)

func main() {
	logger := log.New(os.Stdout, "[Config-Service] ", log.LstdFlags)

	obsClient, err := observability.NewFromEnv("config-service")
	if err != nil {
		logger.Fatalf("Observability init failed: %v", err)
	}
	if obsClient != nil {
		defer func() {
			if err := obsClient.Close(); err != nil {
				logger.Printf("Observability producer close error: %v", err)
			}
		}()

		startupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := obsClient.EmitHealthStarted(startupCtx, "config-service-startup", "config-service started"); err != nil {
			logger.Printf("Observability startup publish failed: %v", err)
		}
	}

	// --- Load server certificate and key ---
	serverCert, err := tls.LoadX509KeyPair("/certs/tls.crt", "/certs/tls.key")
	if err != nil {
		logger.Fatalf("Failed to load server TLS cert/key: %v", err)
	}

	// --- Load CA certificate to verify clients ---
	caCertPEM, err := os.ReadFile("/certs/ca.crt")
	if err != nil {
		logger.Fatalf("Failed to load CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCertPEM) {
		logger.Fatalf("Failed to append CA certificate to pool")
	}

	// --- Configure mutual TLS ---
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert}, // server identity
		ClientCAs:    caCertPool,                    // trust clients signed by this CA
		ClientAuth:   tls.NoClientCert,              // only server authenticates
		// //tls.RequireAndVerifyClientCert, // 🔒 require valid client cert
		MinVersion: tls.VersionTLS13, // enforce modern TLS
	}

	creds := credentials.NewTLS(tlsConfig)

	// --- Create TCP listener ---
	listener, err := net.Listen("tcp", ":5150")
	if err != nil {
		logger.Fatalf("Failed to listen on :5150: %v", err)
	}

	// --- Create gRPC server with TLS credentials ---
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	//grpcServer := grpc.NewServer()
	//logger.Println("Starting gRPC server without TLS (for testing)...")

	// --- Create the configuration engine and register backends ---
	engine := engine.NewMappingEngine(logger)
	// register the Netconf backend
	netconfPlugins := plugins.ForProtocol(topology.ManagementProtocol_NETCONF, logger)
	netconf_backend := protocolbackends.NewNetconfBackend("netconf", netconfPlugins...)
	engine.RegisterBackend(netconf_backend)

	// --- Register ConfigService and gNMI service ---
	svc := service.NewConfigServiceServerImpl(logger, engine)
	service.RegisterConfigServiceServer(grpcServer, svc)

	//gnmi.RegisterGNMIServer(grpcServer, gnmiImpl.NewGNMIService(logger))

	// --- Optional: reflection ---
	reflection.Register(grpcServer)

	logger.Println("🔐 gRPC server with TLS started on port 5150")

	// --- Start serving ---
	if err := grpcServer.Serve(listener); err != nil {
		if obsClient != nil {
			errCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if emitErr := obsClient.EmitHealthError(errCtx, "config-service-serve", err.Error(), ""); emitErr != nil {
				logger.Printf("Observability serve-failure publish failed: %v", emitErr)
			}
		}
		logger.Fatalf("gRPC server failed: %v", err)
	}

}
