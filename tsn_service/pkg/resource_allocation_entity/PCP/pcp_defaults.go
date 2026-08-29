package pcp

import (
	"fmt"

	"OpenCNC/common/structures/pcp"
	topologyconfig "OpenCNC/common/structures/topology_config"
)

/*
PCP defaults are provided as explicit IEEE 802.1Q PCP selection
configurations.

For example, given a bridge NodeConfig with ports "eth0", "eth1", and
"eth2":

	err := PopulateDefaultPcp8P0D(nodeConfig)

The function will populate the missing PCP configuration on each port
approximately as:

	Each port:
	    PCP configuration:
	        priority 0 -> traffic class 0
	        priority 1 -> traffic class 1
	        priority 2 -> traffic class 2
	        priority 3 -> traffic class 3
	        priority 4 -> traffic class 4
	        priority 5 -> traffic class 5
	        priority 6 -> traffic class 6
	        priority 7 -> traffic class 7

	    8P0D encoding:
	        (priority 0, DEI 0) -> PCP 0
	        (priority 1, DEI 0) -> PCP 1
	        ...
	        (priority 7, DEI 0) -> PCP 7

	    8P0D decoding:
	        PCP 0 -> priority 0, DEI 0
	        PCP 1 -> priority 1, DEI 0
	        ...
	        PCP 7 -> priority 7, DEI 0

	    Priority regeneration:
	        priority 0 -> 0
	        priority 1 -> 1
	        ...
	        priority 7 -> 7

The corresponding functions are:

	PopulateDefaultPcp8P0D(nodeConfig)
	PopulateDefaultPcp7P1D(nodeConfig)
	PopulateDefaultPcp6P2D(nodeConfig)
	PopulateDefaultPcp5P3D(nodeConfig)

Only the selected PCP mapping type is installed. The other PCP mapping
types are not automatically added.

N.b.
The functions preserve existing PCP configuration. If a port already has
pcp_config, it is not replaced or altered.

The functions also preserve all other node and port configuration.

The default traffic-class mapping uses eight traffic classes (0..7),
providing the direct priority-to-traffic-class mapping:

	priority 0 -> traffic class 0
	priority 1 -> traffic class 1
	...
	priority 7 -> traffic class 7

PCP mapping enablement is deliberately not changed by these functions.
Whether PCP mapping is enabled is controlled separately through:

	PortConfig.pcp_mapping_enabled

After calling one of the PopulateDefaultPcp* functions, nodeConfig itself
has been populated. There is no returned configuration object to apply.

==============
Example usage:
==============

nodeConfig := &topologyconfig.NodeConfig{
	NodeId: "bridge-1",
	Bridge: &topologyconfig.BridgeConfig{},
	PortConfigs: []*topologyconfig.PortConfig{
		{
			PortId: "eth0",
		},
		{
			PortId: "eth1",
		},
		{
			PortId: "eth2",
		},
	},
}

if err := pcp.PopulateDefaultPcp8P0D(nodeConfig); err != nil {
	return fmt.Errorf("populate PCP 8P0D defaults: %w", err)
}

The user can subsequently change individual values using the constructors
and lookup helpers from pcp_helpers.go.
*/

// -----------------------------------------------------------------------------
// Default PCP Selection Types
// -----------------------------------------------------------------------------

// DefaultPcpTypes returns all PCP selection formats represented by this model.
func DefaultPcpTypes() []pcp.PcpType {
	return []pcp.PcpType{
		pcp.PcpType_PCP_TYPE_8P0D,
		pcp.PcpType_PCP_TYPE_7P1D,
		pcp.PcpType_PCP_TYPE_6P2D,
		pcp.PcpType_PCP_TYPE_5P3D,
	}
}

// -----------------------------------------------------------------------------
// Standard PCP Selection Definitions
// -----------------------------------------------------------------------------

type defaultPcpDefinition struct {
	pcpType pcp.PcpType

	// PCP produced for each internal priority when DEI is false.
	normal [8]uint32

	// PCP produced for each internal priority when DEI is true.
	dei [8]uint32

	// Internal priority decoded from each PCP value.
	decodedPriority [8]uint32

	// DEI decoded from each PCP value.
	decodedDei [8]bool
}

func defaultPcpDefinitionFor(
	pcpType pcp.PcpType,
) (defaultPcpDefinition, error) {
	switch pcpType {
	case pcp.PcpType_PCP_TYPE_8P0D:
		return defaultPcpDefinition{
			pcpType:         pcpType,
			normal:          [8]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			dei:             [8]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			decodedPriority: [8]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			decodedDei: [8]bool{
				false, false, false, false,
				false, false, false, false,
			},
		}, nil

	case pcp.PcpType_PCP_TYPE_7P1D:
		return defaultPcpDefinition{
			pcpType:         pcpType,
			normal:          [8]uint32{0, 1, 2, 3, 5, 5, 6, 7},
			dei:             [8]uint32{0, 1, 2, 3, 4, 4, 6, 7},
			decodedPriority: [8]uint32{0, 1, 2, 3, 4, 4, 6, 7},
			decodedDei: [8]bool{
				false, false, false, false,
				true, false, false, false,
			},
		}, nil

	case pcp.PcpType_PCP_TYPE_6P2D:
		return defaultPcpDefinition{
			pcpType:         pcpType,
			normal:          [8]uint32{0, 1, 3, 3, 5, 5, 6, 7},
			dei:             [8]uint32{0, 1, 2, 2, 4, 4, 6, 7},
			decodedPriority: [8]uint32{0, 1, 2, 2, 4, 4, 6, 7},
			decodedDei: [8]bool{
				false, false, true, false,
				true, false, false, false,
			},
		}, nil

	case pcp.PcpType_PCP_TYPE_5P3D:
		return defaultPcpDefinition{
			pcpType:         pcpType,
			normal:          [8]uint32{0, 0, 3, 3, 5, 5, 6, 7},
			dei:             [8]uint32{1, 1, 2, 2, 4, 4, 6, 7},
			decodedPriority: [8]uint32{0, 0, 2, 2, 4, 4, 6, 7},
			decodedDei: [8]bool{
				false, true, true, false,
				true, false, false, false,
			},
		}, nil

	default:
		return defaultPcpDefinition{}, fmt.Errorf(
			"unsupported PCP type %v",
			pcpType,
		)
	}
}

// -----------------------------------------------------------------------------
// Default Encoding Table
// -----------------------------------------------------------------------------

// NewDefaultPcpEncodingTable creates the standard encoding table for the
// requested PCP selection type.
//
// The table maps:
//
//	(priority, DEI) -> PCP
func NewDefaultPcpEncodingTable(
	pcpType pcp.PcpType,
) (pcp.PcpEncodingTable, error) {
	definition, err := defaultPcpDefinitionFor(pcpType)
	if err != nil {
		return pcp.PcpEncodingTable{}, err
	}

	entries := make([]*pcp.PcpEncodingEntry, 0, 16)

	for priority := uint32(0); priority <= 7; priority++ {
		entry, err := NewPcpEncodingEntry(
			priority,
			false,
			definition.normal[priority],
		)
		if err != nil {
			return pcp.PcpEncodingTable{}, fmt.Errorf(
				"create default %v encoding entry for priority %d, DEI false: %w",
				pcpType,
				priority,
				err,
			)
		}

		entries = append(entries, &entry)

		entry, err = NewPcpEncodingEntry(
			priority,
			true,
			definition.dei[priority],
		)
		if err != nil {
			return pcp.PcpEncodingTable{}, fmt.Errorf(
				"create default %v encoding entry for priority %d, DEI true: %w",
				pcpType,
				priority,
				err,
			)
		}

		entries = append(entries, &entry)
	}

	return NewPcpEncodingTable(pcpType, entries)
}

// -----------------------------------------------------------------------------
// Default Decoding Table
// -----------------------------------------------------------------------------

// NewDefaultPcpDecodingTable creates the standard decoding table for the
// requested PCP selection type.
//
// The table maps:
//
//	PCP -> (priority, DEI)
func NewDefaultPcpDecodingTable(
	pcpType pcp.PcpType,
) (pcp.PcpDecodingTable, error) {
	definition, err := defaultPcpDefinitionFor(pcpType)
	if err != nil {
		return pcp.PcpDecodingTable{}, err
	}

	entries := make([]*pcp.PcpDecodingEntry, 0, 8)

	for priorityCodePoint := uint32(0); priorityCodePoint <= 7; priorityCodePoint++ {
		entry, err := NewPcpDecodingEntry(
			priorityCodePoint,
			definition.decodedPriority[priorityCodePoint],
			definition.decodedDei[priorityCodePoint],
		)
		if err != nil {
			return pcp.PcpDecodingTable{}, fmt.Errorf(
				"create default %v decoding entry for PCP %d: %w",
				pcpType,
				priorityCodePoint,
				err,
			)
		}

		entries = append(entries, &entry)
	}

	return NewPcpDecodingTable(pcpType, entries)
}

// -----------------------------------------------------------------------------
// Default Priority-to-Traffic-Class Mapping
// -----------------------------------------------------------------------------

// NewDefaultTrafficClassTable creates the default priority-to-traffic-class
// mapping for eight traffic classes.
//
// With eight traffic classes, every priority has its own traffic class:
//
//	priority -> traffic class
//
//	0 -> 0
//	1 -> 1
//	...
//	7 -> 7
func NewDefaultTrafficClassTable() (
	[]*pcp.TrafficClassMappingEntry,
	error,
) {
	entries := make([]*pcp.TrafficClassMappingEntry, 0, 8)

	for priority := uint32(0); priority <= 7; priority++ {
		entry, err := NewPriorityTrafficClassMappingEntry(
			priority,
			priority,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create default traffic-class mapping for priority %d: %w",
				priority,
				err,
			)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

// -----------------------------------------------------------------------------
// Default Priority Regeneration
// -----------------------------------------------------------------------------

// NewDefaultPriorityRegenerationTable creates the default identity priority
// regeneration table.
//
//	priority -> regenerated priority
//
//	0 -> 0
//	1 -> 1
//	...
//	7 -> 7
func NewDefaultPriorityRegenerationTable() (
	pcp.PriorityRegenerationTable,
	error,
) {
	entries := make([]*pcp.PriorityRegenerationEntry, 0, 8)

	for priority := uint32(0); priority <= 7; priority++ {
		entry, err := NewPriorityRegenerationEntry(
			priority,
			priority,
		)
		if err != nil {
			return pcp.PriorityRegenerationTable{}, fmt.Errorf(
				"create default priority regeneration entry for priority %d: %w",
				priority,
				err,
			)
		}

		entries = append(entries, &entry)
	}

	return NewPriorityRegenerationTable(entries)
}

// -----------------------------------------------------------------------------
// Default PCP Port Configuration
// -----------------------------------------------------------------------------

// NewDefaultPcpPortConfig creates the complete PCP configuration for one
// port using the requested IEEE 802.1Q PCP selection type.
//
// The resulting configuration contains:
//
//   - priority-to-traffic-class mapping
//   - one encoding table for the selected PCP type
//   - one decoding table for the selected PCP type
//   - identity priority regeneration
func NewDefaultPcpPortConfig(
	pcpType pcp.PcpType,
) (pcp.PcpPortConfig, error) {
	trafficClassTable, err := NewDefaultTrafficClassTable()
	if err != nil {
		return pcp.PcpPortConfig{}, fmt.Errorf(
			"create default traffic-class table: %w",
			err,
		)
	}

	encodingTable, err := NewDefaultPcpEncodingTable(pcpType)
	if err != nil {
		return pcp.PcpPortConfig{}, fmt.Errorf(
			"create default %v encoding table: %w",
			pcpType,
			err,
		)
	}

	decodingTable, err := NewDefaultPcpDecodingTable(pcpType)
	if err != nil {
		return pcp.PcpPortConfig{}, fmt.Errorf(
			"create default %v decoding table: %w",
			pcpType,
			err,
		)
	}

	priorityRegeneration, err := NewDefaultPriorityRegenerationTable()
	if err != nil {
		return pcp.PcpPortConfig{}, fmt.Errorf(
			"create default priority regeneration table: %w",
			err,
		)
	}

	return pcp.PcpPortConfig{
		TrafficClassTable:    trafficClassTable,
		EncodingTables:       []*pcp.PcpEncodingTable{&encodingTable},
		DecodingTables:       []*pcp.PcpDecodingTable{&decodingTable},
		PriorityRegeneration: &priorityRegeneration,
	}, nil
}

// -----------------------------------------------------------------------------
// Populate NodeConfig
// -----------------------------------------------------------------------------

// populateDefaultPcp populates missing PCP configuration on every port of
// a bridge NodeConfig.
//
// Existing pcp_config is preserved completely. Other node and port
// configuration is also left untouched.
func populateDefaultPcp(
	nodeConfig *topologyconfig.NodeConfig,
	pcpType pcp.PcpType,
) error {
	if nodeConfig == nil {
		return fmt.Errorf("node config is nil")
	}

	if nodeConfig.Bridge == nil {
		return fmt.Errorf(
			"node %q is not a bridge",
			nodeConfig.NodeId,
		)
	}

	defaultConfig, err := NewDefaultPcpPortConfig(pcpType)
	if err != nil {
		return fmt.Errorf(
			"create default %v PCP configuration: %w",
			pcpType,
			err,
		)
	}

	for _, portConfig := range nodeConfig.PortConfigs {
		if portConfig == nil {
			continue
		}

		// Never replace an existing PCP configuration.
		if portConfig.PcpConfig != nil {
			continue
		}

		// Make a separate configuration object for each port.
		config := defaultConfig
		portConfig.PcpConfig = &config
	}

	return nil
}

// -----------------------------------------------------------------------------
// Public Standard Default Functions
// -----------------------------------------------------------------------------

// PopulateDefaultPcp8P0D populates the IEEE 802.1Q 8P0D PCP configuration.
//
// 8P0D provides eight priority levels and zero drop-eligible priority levels.
func PopulateDefaultPcp8P0D(
	nodeConfig *topologyconfig.NodeConfig,
) error {
	return populateDefaultPcp(
		nodeConfig,
		pcp.PcpType_PCP_TYPE_8P0D,
	)
}

// PopulateDefaultPcp7P1D populates the IEEE 802.1Q 7P1D PCP configuration.
//
// 7P1D provides seven priority levels and one drop-eligible level.
func PopulateDefaultPcp7P1D(
	nodeConfig *topologyconfig.NodeConfig,
) error {
	return populateDefaultPcp(
		nodeConfig,
		pcp.PcpType_PCP_TYPE_7P1D,
	)
}

// PopulateDefaultPcp6P2D populates the IEEE 802.1Q 6P2D PCP configuration.
//
// 6P2D provides six priority levels and two drop-eligible levels.
func PopulateDefaultPcp6P2D(
	nodeConfig *topologyconfig.NodeConfig,
) error {
	return populateDefaultPcp(
		nodeConfig,
		pcp.PcpType_PCP_TYPE_6P2D,
	)
}

// PopulateDefaultPcp5P3D populates the IEEE 802.1Q 5P3D PCP configuration.
//
// 5P3D provides five priority levels and three drop-eligible levels.
func PopulateDefaultPcp5P3D(
	nodeConfig *topologyconfig.NodeConfig,
) error {
	return populateDefaultPcp(
		nodeConfig,
		pcp.PcpType_PCP_TYPE_5P3D,
	)
}
