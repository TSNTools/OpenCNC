package managemententity

import (
	"context"
	"fmt"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
)

//var log = logger.GetLogger()

func StartMonitor(ipAddress string, index string) {
	//log.Info("Start monitoring")
	setReq("Start", ipAddress, index)
	//setReq("Stop", "192.168.0.2")

}

func setReq(action string, target string, confIndex ...string) {
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
		// fmt.Errorf("could not create a gNMI client: %v", err)
		//log.Errorf("Could not create a gNMI client: %v", err)
	}

	setRequest := pb.SetRequest{
		Update: []*pb.Update{
			{
				Path: &pb.Path{
					Target: target,
					Elem: []*pb.PathElem{
						{
							Name: "Action",
							Key: map[string]string{
								"Action": action,
							},
						},
					},
				},
			},
		},
	}

	if confIndex != nil {
		setRequest.Update[0].Path.Elem = append(setRequest.Update[0].Path.Elem, &pb.PathElem{
			Name: "ConfigIndex",
			Key: map[string]string{
				"ConfigIndex": confIndex[0],
			},
		})
	}

	response, err := c.(*gclient.Client).Set(ctx, &setRequest)

	//log.Infof("Response from device-monitor is: %v", response

	if err != nil {
		//log.Errorf("Target returned RPC error for Set: %v", err)
	} else {
		for _, resp := range response.Response {
			//log.Infof("device-monitor started successfully for: %v", resp.Path.Target)
			fmt.Printf("device-monitor started successfully for: %v", resp.Path.Target)

		}
	}
}
