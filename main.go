package _main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"crypto/tls"
	"crypto/x509"

	gnmiImpl "OpenCNC_config_service/pkg/gnmi" // Your wrapper implementing GNMIService
	service "OpenCNC_config_service/pkg/structures/service"

	gnmi "github.com/openconfig/gnmi/proto/gnmi" // Official gNMI package
	"google.golang.org/grpc/credentials"
)

func main() {
	logger := log.New(os.Stdout, "[Config-Service] ", log.LstdFlags)

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

	// --- Register ConfigService and gNMI service ---
	svc := service.NewConfigServiceServerImpl(logger)
	service.RegisterConfigServiceServer(grpcServer, svc)
	gnmi.RegisterGNMIServer(grpcServer, gnmiImpl.NewGNMIService(logger))

	// --- Optional: reflection ---
	reflection.Register(grpcServer)

	logger.Println("🔐 gRPC server with TLS started on port 5150")

	// --- Start serving ---
	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatalf("gRPC server failed: %v", err)
	}
}
