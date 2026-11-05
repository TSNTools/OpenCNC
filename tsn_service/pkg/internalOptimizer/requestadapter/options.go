package requestadapter

// MechanismSelection identifies the TSN mechanism requested by the caller.
// Routing and placement are optimization decisions, not mechanism names.
type MechanismSelection struct {
	Name        string         `json:"name"`
	Constraints map[string]any `json:"constraints,omitempty"`
}

// StreamOptions carries OaaS-only decisions not present in OpenCNC StreamModel.
// The map key is StreamKey(macAddress, uniqueID), e.g. "02:00:00:00:00:01/1".
type StreamOptions struct {
	Routing                string   // fixed | optimize | preserve
	Scheduling             string   // fixed | optimize | preserve
	Placement              string   // fixed | optimize | preserve
	Change                 string   // none | add | modify | remove (reconfiguration only)
	TalkerApplicationID    string   // required when placement is optimized
	ListenerApplicationIDs []string // required when placement is optimized

	// Optional OaaS Ethernet occupancy values, not present in OpenCNC protobuf.
	// Pointers distinguish an omitted field from an explicit zero.
	SerializationOverheadBytes *int64 `json:"serializationOverheadBytes,omitempty"`
	InterFrameGapBytes         *int64 `json:"interFrameGapBytes,omitempty"`
}

// OaaS-only inter-stream dependency.
// Targets use the optimizer's stable stream IDs.
type DependencyGroup struct {
	Type       string         `json:"type"`
	Target     [][]string     `json:"target"`
	Parameters map[string]any `json:"parameters"`
}

// to contain info that is not available from optimizationTask.proto or forwardingPlaneModel.proto
type RequestOptions struct {
	Mechanisms []MechanismSelection

	// Optional OaaS-only scope restriction. OpenCNC Scope has no trafficClasses.
	TrafficClasses []string

	// Per-stream OaaS optimization instructions, keyed by derived StreamKey.
	Streams map[string]StreamOptions

	// Optional:  Extra hardware information, such as maximum GCL entries ||| external capability inventory in v1.4 capabilities format.
	// Merged by device ID and port ID; explicitly provided values take precedence.
	//
	AdditionalCapabilities map[string]any // a way to provide hardware information to this OpenCNC request adapte from an external source.
	// Example
	// {
	//   "AdditionalCapabilities": {
	//     "devices": [
	//       {
	//         "nodeId": "SW1",
	//         "mechanisms": {
	//           "TAS": {
	//             "maxGclEntries": 64
	//           }
	//         }
	//       }
	//     ]
	//   }
	// }

	// OaaS placement and FRER policies, supplied by an explicit source.
	Applications []map[string]any
	Reliability  map[string]any
	Metadata     map[string]any

	// Explicit inter-stream dependencies supplied by the caller.
	// Not part of OpenCNC ForwardingPlaneModel.
	DependencyGroups []DependencyGroup `json:"dependencyGroups,omitempty"`
}
