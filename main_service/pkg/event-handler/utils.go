package eventhandler

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"OpenCNC/common/configuration"
	"OpenCNC/common/observability"
	configservice "OpenCNC/config_service/grpc_server"
	"OpenCNC/tsn_service/pkg/notificationServer"

	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	defaultTsnServiceAddress = configuration.GetEnv("TSN_SERVICE_HOST", "localhost") +
		":" + configuration.GetEnv("TSN_SERVICE_PORT", "5152")
	defaultConfigServiceAddress = configuration.GetEnv("CONFIG_SERVICE_HOST", "localhost") +
		":" + configuration.GetEnv("CONFIG_SERVICE_PORT", "5150")
)

// Notifies the TSN service through gRPC that it should start calculating
// a new configuration.
func notifyTsnService(
	ctx context.Context,
	obs *observability.Client,
	event *notificationServer.Event,
) (string, error) {
	// TODO: consider keeping a persistent connection to the TSN service.
	conn, err := grpc.Dial(
		defaultTsnServiceAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed dialing TSN service: %v",
				err,
			))
		}
		return "", err
	}
	defer conn.Close()

	if obs != nil {
		_ = obs.Info(ctx, "[Main-service] Notified TSN service successfully.")
	}

	client := notificationServer.NewNotificationClient(conn)

	resp, err := client.Notify(ctx, event)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed sending notification to TSN service: %v",
				err,
			))
		}
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
func notifyConfigService(
	ctx context.Context,
	obs *observability.Client,
	id string,
) error {
	client, conn, err := ConnectToConfigService(
		ctx,
		obs,
		defaultConfigServiceAddress,
	)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed connecting to gNMI service: %v",
				err,
			))
		}

		return fmt.Errorf("failed to notify config-service: %w", err)
	}
	defer conn.Close()

	if obs != nil {
		_ = obs.Info(
			ctx,
			"[Main-service] Successfully sent configuration to config-service!",
		)
	}

	req := &configservice.ConfigurationRequest{
		Id: &id,
	}

	_, err = client.ApplyConfigurationById(ctx, req)

	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Target returned RPC error for Set: %v",
				err,
			))
		}

		return fmt.Errorf("failed to notify config-service: %w", err)
	}

	return nil
}

// Takes in addr such as "config-service:5150" and returns a gNMI-client
func ConnectToGnmiService(
	ctx context.Context,
	obs *observability.Client,
	addr string,
) (client.Impl, error) {
	if obs != nil {
		_ = obs.Info(ctx, "Loading TLS certificates...")
	}

	// Load cert and key files from mounted volume
	cert, err := tls.LoadX509KeyPair("/certs/tls.crt", "/certs/tls.key")
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to load TLS cert/key: %v",
				err,
			))
		}
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	if obs != nil {
		_ = obs.Info(
			ctx,
			"Successfully loaded TLS certificates: client.crt and client.key",
		)
	}

	// Optionally load CA cert if you have a custom CA (recommended)
	caCertPEM, err := os.ReadFile("/certs/ca.crt")
	if err != nil {
		// If you don't have a CA cert, you can skip this or handle error differently
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Warning: failed to load CA cert: %v",
				err,
			))
		}
	}

	if obs != nil {
		_ = obs.Info(ctx, "Successfully loaded the CA certificate.")
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertPEM); !ok {
		if obs != nil {
			_ = obs.Error(ctx, "Warning: failed to append CA cert to pool")
		}
	}

	if obs != nil {
		_ = obs.Info(ctx, "Successfully appended the CA certificate.")
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	}

	if obs != nil {
		_ = obs.Info(ctx, "Prepared the TLS config.")
	}

	client, err := gclient.New(ctx, client.Destination{
		Addrs:       []string{addr},
		Target:      strings.Split(addr, ":")[0],
		Timeout:     20 * time.Second,
		Credentials: nil,
		TLS:         tlsConfig,
	})

	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed creating gNMI client to %s: %v",
				addr,
				err,
			))
		}
		return nil, err
	}

	return client, nil
}

func ConnectToConfigService(
	ctx context.Context,
	obs *observability.Client,
	address string,
) (configservice.ConfigServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed creating gRPC connection to config-service: %v",
				err,
			))
		}
		return nil, nil, err
	}

	if obs != nil {
		_ = obs.Info(ctx, "Successfully connected to config-service")
	}

	client := configservice.NewConfigServiceClient(conn)

	return client, conn, nil
}
