package eventhandler

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	configservice "OpenCNC/config_service/grpc_server"

	// "OpenCNC/main_service/pkg/structures/configuration"
	"OpenCNC/tsn_service/pkg/notificationServer"
	"time"

	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultTsnServiceAddress    = "localhost:5152"
	defaultConfigServiceAddress = "localhost:5150"
)

// Notifies the TSN service through gRPC that it should start calculating
// a new configuration.

func notifyTsnService(event *notificationServer.Event) (string, error) {
	// TODO: consider keeping a persistent connection to the TSN service.
	conn, err := grpc.Dial(
		defaultTsnServiceAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		fmt.Printf("Failed dialing TSN service: %v\n", err)
		return "", err
	}
	defer conn.Close()

	fmt.Println("[Main-service] Notified TSN service successfully.")

	client := notificationServer.NewNotificationClient(conn)

	resp, err := client.Notify(context.Background(), event)
	if err != nil {
		fmt.Printf("Failed sending notification to TSN service: %v\n", err)
		return "", err
	}

	if !resp.GetAccepted() {
		return "", fmt.Errorf(
			"TSN service rejected notification: %s",
			resp.GetMessage(),
		)
	}

	configID := resp.GetConfigId()

	return configID, nil
}

// Applies configuration (sends network change to config-service)
// MTODO:
func notifyConfigService(id string) error {
	client, conn, err := ConnectToConfigService(defaultConfigServiceAddress)
	if err != nil {
		//log.Errorf("Failed connecting to gNMI service: %v", err)
		fmt.Printf("Failed connecting to gNMI service: %v", err)

		return fmt.Errorf("failed to notify config-service: %w", err)
	}
	defer conn.Close()

	fmt.Println("[Main-service] Successfully sent configuration to config-service!")

	//log.Info("Connected to config-service!")

	req := &configservice.ConfigurationRequest{
		Id: &id,
	}

	_, err = client.ApplyConfigurationById(context.Background(), req)

	if err != nil {
		//log.Errorf("Target returned RPC error for Set: %v", err)
		fmt.Printf("Target returned RPC error for Set: %v", err)

		return fmt.Errorf("failed to notify config-service: %w", err)
	}

	return nil
}

// Takes in addr such as "config-service:5150" and returns a gNMI-client
func ConnectToGnmiService(addr string) (client.Impl, error) {
	fmt.Println("Loading TLS certificates...")

	// Load cert and key files from mounted volume
	cert, err := tls.LoadX509KeyPair("/certs/tls.crt", "/certs/tls.key")
	if err != nil {
		fmt.Printf("Failed to load TLS cert/key: %v\n", err)
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	fmt.Printf("Successfully loaded TLS certificates: client.crt and client.key\n")

	// Optionally load CA cert if you have a custom CA (recommended)
	caCertPEM, err := os.ReadFile("/certs/ca.crt")
	if err != nil {
		// If you don't have a CA cert, you can skip this or handle error differently
		fmt.Printf("Warning: failed to load CA cert: %v\n", err)
	}

	fmt.Printf("Successfully loaded the CA certificate.\n")

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertPEM); !ok {
		fmt.Println("Warning: failed to append CA cert to pool")
	}

	fmt.Printf("Successfully appended the CA certificate.\n")

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		// Set InsecureSkipVerify false if using valid CA certs
		InsecureSkipVerify: false,
	}
	fmt.Printf("prepared the tls.config\n")

	client, err := gclient.New(context.Background(), client.Destination{
		Addrs:       []string{addr},
		Target:      strings.Split(addr, ":")[0],
		Timeout:     20 * time.Second,
		Credentials: nil,
		TLS:         tlsConfig,
	})

	if err != nil {
		fmt.Printf("Failed creating gNMI client to %s: %v\n", addr, err)
		return nil, err
	}

	return client, nil
}

func ConnectToConfigService(address string) (configservice.ConfigServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	client := configservice.NewConfigServiceClient(conn)

	return client, conn, nil
}
