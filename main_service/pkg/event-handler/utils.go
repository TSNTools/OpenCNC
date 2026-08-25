package eventhandler

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	store "main-service/pkg/store-wrapper"
	// "main-service/pkg/structures/configuration"
	"main-service/pkg/structures/notification"
	"time"

	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"google.golang.org/grpc"
)

// Notifies the TSN service through gRPC that it should start calculating
// a new configuration.
func notifyTsnService(reqIds *notification.IdList) (*notification.UUID, error) {
	// Create gRPC client and connect to TSN service
	// TODO: consider having a constant connection to TSN service
	conn, err := grpc.Dial("tsn-service:5150", grpc.WithInsecure())
	if err != nil {
		//log.Fatalf("Failed dialing tsn-service: %v", err)
		fmt.Printf("Failed dialing tsn-service: %v", err)
		return nil, err
	}

	defer conn.Close()
	fmt.Println("Dialed TSN service succesfully.")
	client := notification.NewNotificationClient(conn)

	confId, err := client.CalcConfig(context.Background(), reqIds)
	if err != nil {
		//log.Errorf("Calculating configuration failed: %v", err)
		fmt.Printf("Calculating configuration failed: %v", err)

		return nil, err
	}

	return confId, nil
}

// Applies configuration (sends network change to config-service)
// MTODO:
func applyConfiguration(id *notification.UUID) error {
	client, err := ConnectToGnmiService("config-service:5150")
	if err != nil {
		//log.Errorf("Failed connecting to gNMI service: %v", err)
		fmt.Printf("Failed connecting to gNMI service: %v", err)

		return err
	}

	//log.Info("Connected to config-service!")
	fmt.Print("Connected to config-service!")

	// confReq := getSetRequestForConfig()
	confReq, err := store.GetConfigurationRequest(id.Value)
	if err != nil {
		//log.Errorf("Failed getting configuration request from store: %v", err)
		fmt.Printf("Failed getting configuration request from store: %v", err)

		return err
	}

	//log.Info("Successfully requested configuration request from k/v store!")

	fmt.Printf("Successfully requested configuration request from k/v store!")

	// log.Infof("Sending network change request looking like: %v", confReq)

	// response, err := client.(*gclient.Client).Set(context.Background(), confReq)
	_, err = client.(*gclient.Client).Set(context.Background(), confReq)
	if err != nil {
		//log.Errorf("Target returned RPC error for Set: %v", err)
		fmt.Printf("Target returned RPC error for Set: %v", err)

		return err
	}

	//log.Info("Successfully sent configuration to config-service!")
	fmt.Printf("Successfully sent configuration to config-service!")

	// Logs type of operation for each update, check for invalid operations should be added
	// for _, resp := range response.Response {
	// 	log.Infof("Response from config-service is: %v", resp.Op)
	// }

	return nil
}

/* Creates a set request for applying a new configuration.
// TODO: Make automatic, it currently builds a set request statically from predefined values.
func getSetRequestForConfig() *pb.SetRequest {
	// TODO: Generate all pb.Update objects for all the values that should be changed... (Let Hamza know
	// that the current implementation of config-service only takes in config and not a config ID).
	confSetRequest := pb.SetRequest{
		Update: []*pb.Update{
			{
				Path: &pb.Path{
					Target: "192.168.0.1",
					Elem: []*pb.PathElem{
						{
							Name: "interfaces",
							Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
						},
						{
							Name: "interface",
							Key:  map[string]string{"name": "sw0p1"},
						},
						{
							Name: "max-sdu-table",
							Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched", "traffic-class": "0"},
						},
						{
							Name: "queue-max-sdu",
						},
					}, // Path to an element that should be updated
				},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_DecimalVal{
						DecimalVal: &pb.Decimal64{
							Digits:    1500,
							Precision: 6,
						},
					},
				},
			},
			{
				Path: &pb.Path{
					Target: "192.168.0.1",
					Elem: []*pb.PathElem{
						{
							Name: "interfaces",
							Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
						},
						{
							Name: "interface",
							Key:  map[string]string{"name": "sw0p2"},
						},
						{
							Name: "max-sdu-table",
							Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched", "traffic-class": "0"},
						},
						{
							Name: "queue-max-sdu",
						},
					}, // Path to an element that should be updated
				},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_DecimalVal{
						DecimalVal: &pb.Decimal64{
							Digits:    1504,
							Precision: 6,
						},
					},
				},
			},
			{
				Path: &pb.Path{
					Target: "192.168.0.2",
					Elem: []*pb.PathElem{
						{
							Name: "interfaces",
							Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
						},
						{
							Name: "interface",
							Key:  map[string]string{"name": "sw0p1"},
						},
						{
							Name: "max-sdu-table",
							Key:  map[string]string{"namespace": "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched", "traffic-class": "0"},
						},
						{
							Name: "queue-max-sdu",
						},
					}, // Path to an element that should be updated
				},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_DecimalVal{
						DecimalVal: &pb.Decimal64{
							Digits:    1504,
							Precision: 6,
						},
					},
				},
			},
		},
		Extension: []*gnmi_ext.Extension{
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  gnmi_ext.ExtensionID(100),
						Msg: []byte("my_network_change"),
					},
				},
			},
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  gnmi_ext.ExtensionID(101),
						Msg: []byte("1.0.2"),
					},
				},
			},
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  gnmi_ext.ExtensionID(102),
						Msg: []byte("tsn-model"),
					},
				},
			},
		},
	}

	return &confSetRequest
}
*/

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
