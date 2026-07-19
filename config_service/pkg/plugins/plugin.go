package plugins

import (
	devicemodelregistry "OpenCNC_config_service/common/structures/devicemodelregistry"
	managementSessions "OpenCNC_config_service/config_service/pkg/managementSessions"

	"github.com/golang/protobuf/proto"
)

type Plugin interface {
	Name() string
	FeatureName() string
	SupportedByDevice(model *devicemodelregistry.DeviceModel) bool // returns true if the feature is supported by the device model: check is based on yang files names and revisions. it does not guarantee: leaf availability ,RPC support, full subtree support
	SupportedFields(msg proto.Message) []string                    // returns supported field names for the provided structure; empty means unsupported
	Map(msg proto.Message) (any, error)
	Push(mapped any, target managementSessions.DeviceTarget) error
	BuildFeatureXML(root any) (*FeatureXML, error)
}

type FeatureXML struct {
	Container string
	XML       []byte
}
