package admissioncontrol

import (
	"context"
	"fmt"
	"time"

	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
)

// Starting subcription
// func StartSubcription(typeOfQuery client.Type, queryPath []client.Path) {
// log.Info("Start Subscription")
// go sub(typeOfQuery, queryPath)
// }

// Creating a client, a request and start subscription to a target.
func Sub(typeOfQuery client.Type, queryPath []client.Path) {
	ctx := context.Background()

	address := []string{"monitor-service:11161"}

	c, err := gclient.New(ctx, client.Destination{
		Addrs:       address,
		Target:      "monitor-service",
		Timeout:     time.Second * 5,
		Credentials: nil,
		TLS:         nil,
	})

	if err != nil {
		//log.Errorf("Could not create a gNMI client: %v", err)
		fmt.Printf("Could not create a gNMI client: %v", err)

		return
	}

	query := client.Query{
		Addrs:               address,
		Target:              "",
		UpdatesOnly:         true,
		Queries:             queryPath,
		Type:                typeOfQuery, // 1 - Once, 2 - Poll, 3 - Stream
		NotificationHandler: callback,
	}

	//	log.Infof("Establishing connection to %v", query.Addrs)
	fmt.Printf("Establishing connection to %v", query.Addrs)

	err = c.(*gclient.Client).Subscribe(ctx, query)

	if err != nil {
		//log.Errorf("Target returned RPC error for Subscribe: %v", err)
		return
	}

	//log.Info("Client connected successfully")
	fmt.Printf("Client connected successfully")

	for {
		c.(*gclient.Client).Recv()
	}

}

// Subcription data is recived to this function
func callback(msg client.Notification) error {
	// log.Infof("protoCallback msg: %v ", msg)
	fmt.Printf("protoCallback msg: %v ", msg)

	return nil
}
