package nni

import (
	"OpenCNC/common/structures/topology"
	"strings"

	"OpenCNC/common/structures/uni"

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

func convertStreamsToJson(streams *uni.ConfigResponse) (string, error) {
	jsonMarshaler := jsonpb.Marshaler{}

	jsonString, err := jsonMarshaler.MarshalToString(streams)
	if err != nil {
		//log.Errorf("Failed converting streams to json: %v", err)
		return "", err
	}

	return jsonString, nil
}
