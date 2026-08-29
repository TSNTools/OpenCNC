package pcp

import (
	"fmt"

	"OpenCNC/common/structures/pcp"
)

// -----------------------------------------------------------------------------
// IEEE 802.1Q value ranges
// -----------------------------------------------------------------------------

const (
	maxPCPValue       uint32 = 7
	maxPriorityValue  uint32 = 7
	maxTrafficClassID uint32 = 7
)

// validatePCP validates a Priority Code Point.
//
// PCP is a 3-bit field and therefore has the range 0..7.
func validatePCP(value uint32) error {
	if value > maxPCPValue {
		return fmt.Errorf(
			"invalid PCP value %d: must be in range 0..7",
			value,
		)
	}

	return nil
}

// validatePriority validates an IEEE 802.1Q priority.
//
// Priority values are represented using three bits and therefore have
// the range 0..7.
func validatePriority(value uint32) error {
	if value > maxPriorityValue {
		return fmt.Errorf(
			"invalid priority %d: must be in range 0..7",
			value,
		)
	}

	return nil
}

// validateTrafficClassID validates a traffic-class / egress-queue ID.
//
// IEEE 802.1Q supports up to eight traffic classes, represented by
// traffic-class IDs 0..7.
func validateTrafficClassID(value uint32) error {
	if value > maxTrafficClassID {
		return fmt.Errorf(
			"invalid traffic class ID %d: must be in range 0..7",
			value,
		)
	}

	return nil
}

// validatePcpType validates that a supported PCP selection type is specified.
func validatePcpType(pcpType pcp.PcpType) error {
	switch pcpType {
	case pcp.PcpType_PCP_TYPE_8P0D,
		pcp.PcpType_PCP_TYPE_7P1D,
		pcp.PcpType_PCP_TYPE_6P2D,
		pcp.PcpType_PCP_TYPE_5P3D:
		return nil

	default:
		return fmt.Errorf("unsupported PCP type %v", pcpType)
	}
}

// -----------------------------------------------------------------------------
// Priority to Traffic-Class Mapping
// -----------------------------------------------------------------------------

// NewPriorityTrafficClassMappingEntry creates a priority-to-traffic-class
// mapping entry.
//
// The mapping corresponds to the IEEE 802.1Q Traffic Class Table:
//
//	priority -> egress traffic class
func NewPriorityTrafficClassMappingEntry(
	priority uint32,
	egressQueueID uint32,
) (pcp.TrafficClassMappingEntry, error) {
	if err := validatePriority(priority); err != nil {
		return pcp.TrafficClassMappingEntry{}, err
	}

	if err := validateTrafficClassID(egressQueueID); err != nil {
		return pcp.TrafficClassMappingEntry{}, err
	}

	return pcp.TrafficClassMappingEntry{
		Priority:      priority,
		EgressQueueId: egressQueueID,
	}, nil
}

// -----------------------------------------------------------------------------
// PCP Encoding
// -----------------------------------------------------------------------------

// NewPcpEncodingEntry creates one PCP encoding entry.
//
// It maps:
//
//	(priority, DEI) -> PCP
func NewPcpEncodingEntry(
	priority uint32,
	dei bool,
	priorityCodePoint uint32,
) (pcp.PcpEncodingEntry, error) {
	if err := validatePriority(priority); err != nil {
		return pcp.PcpEncodingEntry{}, err
	}

	if err := validatePCP(priorityCodePoint); err != nil {
		return pcp.PcpEncodingEntry{}, fmt.Errorf(
			"invalid encoded PCP: %w",
			err,
		)
	}

	return pcp.PcpEncodingEntry{
		Priority:          priority,
		Dei:               dei,
		PriorityCodePoint: priorityCodePoint,
	}, nil
}

// NewPcpEncodingTable creates a PCP encoding table for a selection type.
//
// Partial tables are permitted. This allows callers to construct
// configuration incrementally.
func NewPcpEncodingTable(
	pcpType pcp.PcpType,
	entries []*pcp.PcpEncodingEntry,
) (pcp.PcpEncodingTable, error) {
	if err := validatePcpType(pcpType); err != nil {
		return pcp.PcpEncodingTable{}, err
	}

	if err := validateEncodingEntries(entries); err != nil {
		return pcp.PcpEncodingTable{}, err
	}

	return pcp.PcpEncodingTable{
		PcpType: pcpType,
		Entries: entries,
	}, nil
}

func validateEncodingEntries(
	entries []*pcp.PcpEncodingEntry,
) error {
	for index, entry := range entries {
		if entry == nil {
			return fmt.Errorf(
				"PCP encoding entry at index %d is nil",
				index,
			)
		}

		if err := validatePriority(entry.Priority); err != nil {
			return fmt.Errorf(
				"invalid priority in encoding entry %d: %w",
				index,
				err,
			)
		}

		if err := validatePCP(entry.PriorityCodePoint); err != nil {
			return fmt.Errorf(
				"invalid PCP in encoding entry %d: %w",
				index,
				err,
			)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// PCP Decoding
// -----------------------------------------------------------------------------

// NewPcpDecodingEntry creates one PCP decoding entry.
//
// It maps:
//
//	PCP -> (priority, DEI)
func NewPcpDecodingEntry(
	priorityCodePoint uint32,
	priority uint32,
	dei bool,
) (pcp.PcpDecodingEntry, error) {
	if err := validatePCP(priorityCodePoint); err != nil {
		return pcp.PcpDecodingEntry{}, err
	}

	if err := validatePriority(priority); err != nil {
		return pcp.PcpDecodingEntry{}, err
	}

	return pcp.PcpDecodingEntry{
		PriorityCodePoint: priorityCodePoint,
		Priority:          priority,
		Dei:               dei,
	}, nil
}

// NewPcpDecodingTable creates a PCP decoding table for a selection type.
func NewPcpDecodingTable(
	pcpType pcp.PcpType,
	entries []*pcp.PcpDecodingEntry,
) (pcp.PcpDecodingTable, error) {
	if err := validatePcpType(pcpType); err != nil {
		return pcp.PcpDecodingTable{}, err
	}

	if err := validateDecodingEntries(entries); err != nil {
		return pcp.PcpDecodingTable{}, err
	}

	return pcp.PcpDecodingTable{
		PcpType: pcpType,
		Entries: entries,
	}, nil
}

func validateDecodingEntries(
	entries []*pcp.PcpDecodingEntry,
) error {
	for index, entry := range entries {
		if entry == nil {
			return fmt.Errorf(
				"PCP decoding entry at index %d is nil",
				index,
			)
		}

		if err := validatePCP(entry.PriorityCodePoint); err != nil {
			return fmt.Errorf(
				"invalid PCP in decoding entry %d: %w",
				index,
				err,
			)
		}

		if err := validatePriority(entry.Priority); err != nil {
			return fmt.Errorf(
				"invalid priority in decoding entry %d: %w",
				index,
				err,
			)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Priority Regeneration
// -----------------------------------------------------------------------------

// NewPriorityRegenerationEntry creates a priority regeneration entry.
//
// It maps:
//
//	input priority -> regenerated priority
func NewPriorityRegenerationEntry(
	priority uint32,
	regeneratedPriority uint32,
) (pcp.PriorityRegenerationEntry, error) {
	if err := validatePriority(priority); err != nil {
		return pcp.PriorityRegenerationEntry{}, err
	}

	if err := validatePriority(regeneratedPriority); err != nil {
		return pcp.PriorityRegenerationEntry{}, fmt.Errorf(
			"invalid regenerated priority: %w",
			err,
		)
	}

	return pcp.PriorityRegenerationEntry{
		Priority:            priority,
		RegeneratedPriority: regeneratedPriority,
	}, nil
}

// NewPriorityRegenerationTable creates a priority regeneration table.
func NewPriorityRegenerationTable(
	entries []*pcp.PriorityRegenerationEntry,
) (pcp.PriorityRegenerationTable, error) {
	if err := validatePriorityRegenerationEntries(entries); err != nil {
		return pcp.PriorityRegenerationTable{}, err
	}

	return pcp.PriorityRegenerationTable{
		Entries: entries,
	}, nil
}

func validatePriorityRegenerationEntries(
	entries []*pcp.PriorityRegenerationEntry,
) error {
	for index, entry := range entries {
		if entry == nil {
			return fmt.Errorf(
				"priority regeneration entry at index %d is nil",
				index,
			)
		}

		if err := validatePriority(entry.Priority); err != nil {
			return fmt.Errorf(
				"invalid priority in regeneration entry %d: %w",
				index,
				err,
			)
		}

		if err := validatePriority(entry.RegeneratedPriority); err != nil {
			return fmt.Errorf(
				"invalid regenerated priority in entry %d: %w",
				index,
				err,
			)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Lookup Helpers
// -----------------------------------------------------------------------------

// FindTrafficClassMapping returns the traffic-class mapping for a priority.
func FindTrafficClassMapping(
	config *pcp.PcpPortConfig,
	priority uint32,
) (*pcp.TrafficClassMappingEntry, bool) {
	if config == nil {
		return nil, false
	}

	for _, entry := range config.TrafficClassTable {
		if entry != nil && entry.Priority == priority {
			return entry, true
		}
	}

	return nil, false
}

// ContainsTrafficClassMapping reports whether a mapping for a priority exists.
func ContainsTrafficClassMapping(
	config *pcp.PcpPortConfig,
	priority uint32,
) bool {
	_, found := FindTrafficClassMapping(config, priority)
	return found
}

// FindPcpEncodingTable returns the encoding table for a PCP selection type.
func FindPcpEncodingTable(
	config *pcp.PcpPortConfig,
	pcpType pcp.PcpType,
) (*pcp.PcpEncodingTable, bool) {
	if config == nil {
		return nil, false
	}

	for _, table := range config.EncodingTables {
		if table != nil && table.PcpType == pcpType {
			return table, true
		}
	}

	return nil, false
}

// ContainsPcpEncodingTable reports whether an encoding table exists.
func ContainsPcpEncodingTable(
	config *pcp.PcpPortConfig,
	pcpType pcp.PcpType,
) bool {
	_, found := FindPcpEncodingTable(config, pcpType)
	return found
}

// FindPcpDecodingTable returns the decoding table for a PCP selection type.
func FindPcpDecodingTable(
	config *pcp.PcpPortConfig,
	pcpType pcp.PcpType,
) (*pcp.PcpDecodingTable, bool) {
	if config == nil {
		return nil, false
	}

	for _, table := range config.DecodingTables {
		if table != nil && table.PcpType == pcpType {
			return table, true
		}
	}

	return nil, false
}

// ContainsPcpDecodingTable reports whether a decoding table exists.
func ContainsPcpDecodingTable(
	config *pcp.PcpPortConfig,
	pcpType pcp.PcpType,
) bool {
	_, found := FindPcpDecodingTable(config, pcpType)
	return found
}

// FindPriorityRegenerationEntry returns the regeneration entry for a priority.
func FindPriorityRegenerationEntry(
	table *pcp.PriorityRegenerationTable,
	priority uint32,
) (*pcp.PriorityRegenerationEntry, bool) {
	if table == nil {
		return nil, false
	}

	for _, entry := range table.Entries {
		if entry != nil && entry.Priority == priority {
			return entry, true
		}
	}

	return nil, false
}

// ContainsPriorityRegenerationEntry reports whether a regeneration entry
// exists for the specified priority.
func ContainsPriorityRegenerationEntry(
	table *pcp.PriorityRegenerationTable,
	priority uint32,
) bool {
	_, found := FindPriorityRegenerationEntry(table, priority)
	return found
}
