package nni

import (
	"main-service/pkg/structures/configuration"
	monitor "main-service/pkg/structures/temp-monitor-conf"
	"main-service/pkg/structures/topology"
	"strings"

	"encoding/json"

	"github.com/gogo/protobuf/jsonpb"
	"google.golang.org/protobuf/encoding/protojson"
)

func convertNodesToJson(nodes []*topology.Node) (string, error) {
	marshaller := protojson.MarshalOptions{
		Multiline:     true,
		Indent:        "  ",
		UseProtoNames: false, // to match json_name in .proto
	}

	var builder strings.Builder
	builder.WriteString("[\n")

	for i, node := range nodes {
		if node == nil {
			continue
		}
		jsonData, err := marshaller.Marshal(node)
		if err != nil {
			return "", err
		}
		builder.WriteString("  ")
		builder.WriteString(string(jsonData))
		if i < len(nodes)-1 {
			builder.WriteString(",\n")
		}
	}

	builder.WriteString("\n]")
	return builder.String(), nil
}

// /////
func convertSwitchesToJson(switches []monitor.MonitorConfig) (string, error) {

	jsonBytes, err := json.Marshal(switches)
	if err != nil {
		//log.Errorf("Failed converting %v to json: %v", switches, err)
		return "", err
	}
	jsonString := string(jsonBytes)

	return jsonString, nil
}

func convertStreamsToJson(streams *configuration.ConfigResponse) (string, error) {
	jsonMarshaler := jsonpb.Marshaler{}

	jsonString, err := jsonMarshaler.MarshalToString(streams)
	if err != nil {
		//log.Errorf("Failed converting streams to json: %v", err)
		return "", err
	}

	return jsonString, nil
}
