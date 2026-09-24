package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"OpenCNC/common/structures/stream"
	"OpenCNC/common/structures/uni"
	uni_server "OpenCNC/main_service/pkg/uni"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

func BuildAddStreamRequest() *uni.ConfigRequest {
	// Stream 1:
	// Talker:  tsn1, interface enp1s0, MAC 00-00-00-00-00-01
	// Listener: tsn2, interface ens6,   MAC 00-00-00-00-00-02
	stream1ID := &stream.StreamId{
		MacAddress: "00-00-00-00-00-01",
		UniqueId:   1,
	}

	stream1 := &uni.Request{
		Id: "req-stream-tsn1-to-tsn2",
		Talker: &uni.TalkerGroup{
			StrId: stream1ID,
			StrRank: &uni.StreamRank{
				Rank: stream.StreamRankValue_RANK_NON_EMERGENCY,
			},
			EndStationInterfaces: []*uni.Interface{
				{
					Index: 0,
					InterfaceId: &uni.InterfaceId{
						MacAddress:    "00-00-00-00-00-01",
						InterfaceName: "enp1s0",
					},
				},
			},
			DataFrameSpecification: []*uni.DataFrameSpecification{
				{
					Index: 0,
					MacAddr: &uni.IeeeMacAddress{
						DestinationMac: "00-00-00-00-00-02",
						SourceMac:      "00-00-00-00-00-01",
					},
					VlanTag: &uni.IeeeVlanTag{
						PriorityCodePoint: 7, // PCP 7 / Socket Priority 7 / Gate 7
						VlanId:            100,
					},
				},
			},
			TrafficSpecification: &uni.TrafficSpecification{
				Interval: &uni.Interval{
					Numerator:   1, // 1 second cycle
					Denominator: 1,
				},
				MaxFramesPerInterval:  32,  // 32 frames per cycle
				MaxFrameSize:          128, // 128 bytes
				TransmissionSelection: 0,
				TimeAware: &uni.TimeAware{
					EarliestTransmitOffset: 0,
					LatestTransmitOffset:   500000,
					Jitter:                 0,
				},
			},
			UserToNetReq: &uni.UserToNetworkRequirements{
				NumSeamlessTrees: 1,
				MaxLatency:       1000000000, // 1 second
			},
			InterfCap: &uni.InterfaceCapabilities{
				VlanTagCapable: true, // Bridge configuration required
			},
		},
		ListenerList: []*uni.ListenerGroup{
			{
				Index: 0,
				StrId: stream1ID,
				EndStationInterfaces: []*uni.Interface{
					{
						Index: 0,
						InterfaceId: &uni.InterfaceId{
							MacAddress:    "00-00-00-00-00-02",
							InterfaceName: "ens6",
						},
					},
				},
				UserToNetReq: &uni.UserToNetworkRequirements{
					NumSeamlessTrees: 1,
					MaxLatency:       1000000000,
				},
				InterfCap: &uni.InterfaceCapabilities{
					VlanTagCapable: true,
				},
			},
		},
	}

	// Stream 2:
	// Talker:  tsn2, interface ens6,   MAC 00-00-00-00-00-02
	// Listener: tsn1, interface enp1s0, MAC 00-00-00-00-00-01
	stream2ID := &stream.StreamId{
		MacAddress: "00-00-00-00-00-02",
		UniqueId:   2,
	}

	stream2 := &uni.Request{
		Id: "req-stream-tsn2-to-tsn1",
		Talker: &uni.TalkerGroup{
			StrId: stream2ID,
			StrRank: &uni.StreamRank{
				Rank: stream.StreamRankValue_RANK_NON_EMERGENCY,
			},
			EndStationInterfaces: []*uni.Interface{
				{
					Index: 0,
					InterfaceId: &uni.InterfaceId{
						MacAddress:    "00-00-00-00-00-02",
						InterfaceName: "ens6",
					},
				},
			},
			DataFrameSpecification: []*uni.DataFrameSpecification{
				{
					Index: 0,
					MacAddr: &uni.IeeeMacAddress{
						DestinationMac: "00-00-00-00-00-01",
						SourceMac:      "00-00-00-00-00-02",
					},
					VlanTag: &uni.IeeeVlanTag{
						PriorityCodePoint: 7, // PCP 7 / Socket Priority 7 / Gate 7
						VlanId:            100,
					},
				},
			},
			TrafficSpecification: &uni.TrafficSpecification{
				Interval: &uni.Interval{
					Numerator:   1, // 1 second cycle
					Denominator: 1,
				},
				MaxFramesPerInterval:  32,  // 32 frames per cycle
				MaxFrameSize:          128, // 128 bytes
				TransmissionSelection: 0,
				TimeAware: &uni.TimeAware{
					EarliestTransmitOffset: 0,
					LatestTransmitOffset:   500000,
					Jitter:                 0,
				},
			},
			UserToNetReq: &uni.UserToNetworkRequirements{
				NumSeamlessTrees: 1,
				MaxLatency:       1000000000,
			},
			InterfCap: &uni.InterfaceCapabilities{
				VlanTagCapable: true, // Bridge configuration required
			},
		},
		ListenerList: []*uni.ListenerGroup{
			{
				Index: 0,
				StrId: stream2ID,
				EndStationInterfaces: []*uni.Interface{
					{
						Index: 0,
						InterfaceId: &uni.InterfaceId{
							MacAddress:    "00-00-00-00-00-01",
							InterfaceName: "enp1s0",
						},
					},
				},
				UserToNetReq: &uni.UserToNetworkRequirements{
					NumSeamlessTrees: 1,
					MaxLatency:       1000000000,
				},
				InterfCap: &uni.InterfaceCapabilities{
					VlanTagCapable: true,
				},
			},
		},
	}

	return &uni.ConfigRequest{
		Version: 1.0,
		Requests: []*uni.Request{
			stream1,
			stream2,
		},
	}
}

func main() {
	var (
		target   = flag.String("target", "localhost:5153", "OpenCNC UNI gRPC address (host:port)")
		timeout  = flag.Duration("timeout", 60*time.Second, "Request timeout")
		dumpJSON = flag.Bool("dump-json", false, "Print JSON representation of request and exit")
	)
	flag.Parse()

	if envTarget := os.Getenv("CNC_GRPC_ENDPOINT"); envTarget != "" && *target == "localhost:5153" {
		*target = envTarget
	}

	req := BuildAddStreamRequest()

	if *dumpJSON {
		m := protojson.MarshalOptions{
			Indent:          "  ",
			EmitUnpopulated: false,
			UseProtoNames:   false,
		}
		jsonBytes, err := m.Marshal(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
		return
	}

	fmt.Println("==================================================================")
	fmt.Println(" OpenCNC Add Stream Request (gRPC)")
	fmt.Println("==================================================================")
	fmt.Printf(" Target gRPC Endpoint : %s\n", *target)
	fmt.Println(" Service             : uni_server.UniService/AddStream")
	fmt.Println(" Stream 1            : Talker tsn1 (enp1s0) -> Listener tsn2 (ens6)")
	fmt.Println(" Stream 2            : Talker tsn2 (ens6)   -> Listener tsn1 (enp1s0)")
	fmt.Println(" VLAN ID             : 100 enabled ")
	fmt.Println(" PCP / Priority      : 7 (Gate 7)")
	fmt.Println(" Cycle Time          : 1s (1000ms)")
	fmt.Println(" Frames / Cycle      : 32")
	fmt.Println(" Frame Size          : 128 bytes")
	fmt.Println(" Bridge Config       : Required")
	fmt.Println("==================================================================")

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		*target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[✗] Failed to connect to gRPC server at %s: %v\n", *target, err)
		os.Exit(1)
	}
	defer conn.Close()

	client := uni_server.NewUniServiceClient(conn)

	start := time.Now()
	resp, err := client.AddStream(ctx, req)
	duration := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[✗] AddStream gRPC call failed (%v): %v\n", duration, err)
		os.Exit(1)
	}

	fmt.Printf("\n[✓] AddStream gRPC call succeeded in %v!\n\n", duration)

	m := protojson.MarshalOptions{
		Indent:          "  ",
		EmitUnpopulated: false,
	}
	respJSON, err := m.Marshal(resp)
	if err == nil {
		fmt.Println("Response:")
		fmt.Println(string(respJSON))
	} else {
		fmt.Printf("Response: %+v\n", resp)
	}
}
