package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
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
		target     = flag.String("target", "localhost:5153", "OpenCNC UNI gRPC address (host:port)")
		timeout    = flag.Duration("timeout", 60*time.Second, "Request timeout per iteration")
		iterations = flag.Int("iterations", 50, "Number of iterations to repeat AddStream")
		delay      = flag.Duration("delay", 0*time.Second, "Delay between iterations (e.g. 1s)")
		dumpJSON   = flag.Bool("dump-json", false, "Print JSON representation of request and exit")
	)
	flag.Parse()

	if envTarget := os.Getenv("CNC_GRPC_ENDPOINT"); envTarget != "" && *target == "localhost:5153" {
		*target = envTarget
	}

	if envIter := os.Getenv("ITERATIONS"); envIter != "" && *iterations == 50 {
		if val, err := strconv.Atoi(envIter); err == nil && val > 0 {
			*iterations = val
		}
	}

	if envDelay := os.Getenv("DELAY"); envDelay != "" && *delay == 0 {
		if val, err := time.ParseDuration(envDelay); err == nil {
			*delay = val
		}
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
	fmt.Println(" OpenCNC Add Stream Iterative Requests (gRPC)")
	fmt.Println("==================================================================")
	fmt.Printf(" Target gRPC Endpoint : %s\n", *target)
	fmt.Println(" Service             : uni_server.UniService/AddStream")
	fmt.Printf(" Total Iterations    : %d\n", *iterations)
	fmt.Printf(" Timeout per Request : %v\n", *timeout)
	if *delay > 0 {
		fmt.Printf(" Delay Between Runs  : %v\n", *delay)
	}
	fmt.Println(" Stream 1            : Talker tsn1 (enp1s0) -> Listener tsn2 (ens6)")
	fmt.Println(" Stream 2            : Talker tsn2 (ens6)   -> Listener tsn1 (enp1s0)")
	fmt.Println(" VLAN ID             : 100")
	fmt.Println(" PCP / Priority      : 7 (Gate 7)")
	fmt.Println(" Cycle Time          : 1s (1000ms)")
	fmt.Println(" Frames / Cycle      : 32")
	fmt.Println(" Frame Size          : 128 bytes")
	fmt.Println(" Bridge Config       : Required")
	fmt.Println("==================================================================")

	dialCtx, dialCancel := context.WithTimeout(context.Background(), *timeout)
	defer dialCancel()

	conn, err := grpc.DialContext(
		dialCtx,
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

	var (
		totalStart   = time.Now()
		successCount = 0
		failCount    = 0
		latencies    []time.Duration
		minLatency   time.Duration
		maxLatency   time.Duration
		sumLatency   time.Duration
	)

	for i := 1; i <= *iterations; i++ {
		fmt.Printf("[Iteration %d/%d] Sending AddStream request...\n", i, *iterations)

		iterCtx, iterCancel := context.WithTimeout(context.Background(), *timeout)
		start := time.Now()
		resp, err := client.AddStream(iterCtx, req)
		duration := time.Since(start)
		iterCancel()

		if err != nil {
			failCount++
			fmt.Fprintf(os.Stderr, "  [✗] Iteration %d/%d failed after %v: %v\n", i, *iterations, duration, err)
		} else {
			successCount++
			latencies = append(latencies, duration)
			sumLatency += duration
			if minLatency == 0 || duration < minLatency {
				minLatency = duration
			}
			if duration > maxLatency {
				maxLatency = duration
			}

			statusMsg := ""
			if resp != nil && len(resp.GetResponses()) > 0 {
				var ids []string
				for _, r := range resp.GetResponses() {
					if r.GetRequestId() != "" {
						ids = append(ids, r.GetRequestId())
					}
				}
				if len(ids) > 0 {
					statusMsg = fmt.Sprintf(" (Requests: %v)", ids)
				}
			}
			fmt.Printf("  [✓] Iteration %d/%d succeeded in %v%s\n", i, *iterations, duration, statusMsg)
		}

		if *delay > 0 && i < *iterations {
			time.Sleep(*delay)
		}
	}

	totalDuration := time.Since(totalStart)

	fmt.Println("\n==================================================================")
	fmt.Println(" Iterative Add Stream Summary")
	fmt.Println("==================================================================")
	fmt.Printf(" Total Iterations : %d\n", *iterations)
	fmt.Printf(" Successful       : %d\n", successCount)
	fmt.Printf(" Failed           : %d\n", failCount)
	fmt.Printf(" Total Wall Time  : %v\n", totalDuration)
	if successCount > 0 {
		avgLatency := sumLatency / time.Duration(successCount)
		fmt.Printf(" Min Latency      : %v\n", minLatency)
		fmt.Printf(" Max Latency      : %v\n", maxLatency)
		fmt.Printf(" Avg Latency      : %v\n", avgLatency)
	}
	fmt.Println("==================================================================")

	if failCount > 0 {
		os.Exit(1)
	}
}
