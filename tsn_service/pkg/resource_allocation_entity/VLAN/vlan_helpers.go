package vlan

import (
	"fmt"

	"OpenCNC/common/structures/vlan"
)

/*
Package vlan provides small constructors and validation helpers for building
VLAN configuration objects.

The VLAN package is responsible for creating valid VLAN domain objects from
the protobuf model.

It does not:
  - modify devices directly
  - build SchemaTree paths
  - create gNMI updates
  - assemble topology.PortConfig objects

Those responsibilities belong to higher-level packages such as topology.
*/

// -----------------------------------------------------------------------------
// Validation
// -----------------------------------------------------------------------------

// validateVlanID validates that a VLAN identifier is within the valid
// IEEE 802.1Q VLAN ID range.
func validateVlanID(vlanID uint32) error {
	if vlanID < 1 || vlanID > 4094 {
		return fmt.Errorf("invalid VLAN ID %d: must be between 1 and 4094", vlanID)
	}
	return nil
}

// validateVlanIDs validates a collection of VLAN identifiers.
func validateVlanIDs(vlanIDs []uint32) error {
	for _, vlanID := range vlanIDs {
		if err := validateVlanID(vlanID); err != nil {
			return err
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// VLAN Labels
// -----------------------------------------------------------------------------

// NewVlanLabel creates a human-readable label for a VLAN.
//
// vlanID must be in the valid VLAN ID range (1..4094).
//
// The label is informational and does not affect VLAN forwarding behaviour.
func NewVlanLabel(vlanID uint32, label string) (vlan.VlanLabel, error) {
	if err := validateVlanID(vlanID); err != nil {
		return vlan.VlanLabel{}, err
	}

	return vlan.VlanLabel{
		VlanId: vlanID,
		Label:  label,
	}, nil
}

// -----------------------------------------------------------------------------
// VLAN Port Maps
// -----------------------------------------------------------------------------

// NewVlanPortMap creates the per-port configuration associated with a VLAN
// registration entry.
//
// portID identifies the bridge port.
// registrarAdminControl controls the registrar behaviour for the VLAN.
// vlanTransmitted determines whether frames are transmitted tagged or
// untagged.
func NewVlanPortMap(
	portID string,
	registrarAdminControl vlan.RegistrarAdminControl,
	vlanTransmitted vlan.VlanTransmitted,
) (vlan.VlanPortMap, error) {
	if portID == "" {
		return vlan.VlanPortMap{}, fmt.Errorf("port ID cannot be empty")
	}

	return vlan.VlanPortMap{
		PortId:                portID,
		RegistrarAdminControl: registrarAdminControl,
		VlanTransmitted:       vlanTransmitted,
	}, nil
}

// -----------------------------------------------------------------------------
// VLAN Registration Entries
// -----------------------------------------------------------------------------

// NewVlanRegistrationEntry creates a VLAN registration entry.
//
// The entry can be static or dynamic, depending on entryType. The VLAN IDs
// and port mappings are validated before the configuration object is created.
func NewVlanRegistrationEntry(
	vlanIDs []uint32,
	databaseID uint32,
	entryType vlan.VlanRegistrationEntryType,
	portMaps []*vlan.VlanPortMap,
) (vlan.VlanRegistrationEntry, error) {
	if err := validateVlanIDs(vlanIDs); err != nil {
		return vlan.VlanRegistrationEntry{}, fmt.Errorf("invalid VLAN IDs: %w", err)
	}

	if entryType == vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_UNSPECIFIED {
		return vlan.VlanRegistrationEntry{}, fmt.Errorf("VLAN registration entry type cannot be unspecified")
	}

	return vlan.VlanRegistrationEntry{
		DatabaseId: databaseID,
		VlanIds:    vlanIDs,
		EntryType:  entryType,
		PortMaps:   portMaps,
	}, nil
}

// NewStaticVlanRegistrationEntry creates a static VLAN registration entry
// for a single port.
//
// vlanIDs specifies the VLAN identifiers associated with the entry.
// portID identifies the bridge port.
// databaseID identifies the Filtering Database.
// vlanTransmitted specifies whether VLAN frames are transmitted tagged or
// untagged.
// registrarAdminControl specifies the registrar behaviour for the VLAN.
func NewStaticVlanRegistrationEntry(
	vlanIDs []uint32,
	portID string,
	databaseID uint32,
	vlanTransmitted vlan.VlanTransmitted,
	registrarAdminControl vlan.RegistrarAdminControl,
) (vlan.VlanRegistrationEntry, error) {
	if err := validateVlanIDs(vlanIDs); err != nil {
		return vlan.VlanRegistrationEntry{}, fmt.Errorf("invalid VLAN IDs: %w", err)
	}

	portMap, err := NewVlanPortMap(
		portID,
		registrarAdminControl,
		vlanTransmitted,
	)
	if err != nil {
		return vlan.VlanRegistrationEntry{}, err
	}

	return vlan.VlanRegistrationEntry{
		DatabaseId: databaseID,
		VlanIds:    vlanIDs,
		EntryType:  vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_STATIC,
		PortMaps: []*vlan.VlanPortMap{
			&portMap,
		},
	}, nil
}

// -----------------------------------------------------------------------------
// VLAN / FID Mapping
// -----------------------------------------------------------------------------

// NewVidToFidMapping creates a mapping between a VLAN identifier (VID) and
// a Filtering Database identifier (FID).
func NewVidToFidMapping(
	vid uint32,
	fid uint32,
) (vlan.VidToFidMapping, error) {
	if err := validateVlanID(vid); err != nil {
		return vlan.VidToFidMapping{}, err
	}

	return vlan.VidToFidMapping{
		Vid: vid,
		Fid: fid,
	}, nil
}

// -----------------------------------------------------------------------------
// VLAN Membership
// -----------------------------------------------------------------------------

// NewVlanMembership creates a VLAN membership configuration.
//
// tagged determines whether frames associated with the VLAN are transmitted
// as VLAN-tagged frames on the corresponding port.
func NewVlanMembership(
	vlanID uint32,
	tagged bool,
) (vlan.VlanMembership, error) {
	if err := validateVlanID(vlanID); err != nil {
		return vlan.VlanMembership{}, err
	}

	return vlan.VlanMembership{
		VlanId: vlanID,
		Tagged: tagged,
	}, nil
}

// -----------------------------------------------------------------------------
// Port VLAN Advanced Configuration
// -----------------------------------------------------------------------------

// NewPortVlanAdvancedConfig creates the basic advanced VLAN configuration
// for a bridge port.
//
// acceptableFrame specifies which frame types are accepted on ingress.
// ingressFiltering enables or disables ingress VLAN membership filtering.
// restrictedVlanRegistration enables or disables restricted VLAN registration.
func NewPortVlanAdvancedConfig(
	acceptableFrame vlan.AcceptableFrameType,
	ingressFiltering bool,
	restrictedVlanRegistration bool,
) (vlan.PortVlanAdvancedConfig, error) {
	if acceptableFrame == vlan.AcceptableFrameType_ACCEPTABLE_FRAME_TYPE_UNSPECIFIED {
		return vlan.PortVlanAdvancedConfig{}, fmt.Errorf("acceptable frame type cannot be unspecified")
	}

	return vlan.PortVlanAdvancedConfig{
		AcceptableFrame:                   &acceptableFrame,
		IngressFilteringEnabled:           &ingressFiltering,
		RestrictedVlanRegistrationEnabled: &restrictedVlanRegistration,
	}, nil
}

// -----------------------------------------------------------------------------
// VID Translation
// -----------------------------------------------------------------------------

// NewVidTranslation creates a VLAN ID translation between a local VID and
// a relay VID.
func NewVidTranslation(
	localVID uint32,
	relayVID uint32,
) (vlan.VidTranslation, error) {
	if err := validateVlanID(localVID); err != nil {
		return vlan.VidTranslation{}, fmt.Errorf("invalid local VLAN ID: %w", err)
	}

	if err := validateVlanID(relayVID); err != nil {
		return vlan.VidTranslation{}, fmt.Errorf("invalid relay VLAN ID: %w", err)
	}

	return vlan.VidTranslation{
		LocalVid: localVID,
		RelayVid: relayVID,
	}, nil
}

// -----------------------------------------------------------------------------
// Protocol Group Database
// -----------------------------------------------------------------------------

// NewEthernetProtocolGroup creates a protocol group database entry for an
// Ethernet frame identified by an EtherType.
//
// The frame format is set to ETHERNET and the EtherType discriminator is set.
func NewEthernetProtocolGroup(
	dbIndex uint32,
	ethertype uint32,
	groupID uint32,
) (vlan.ProtocolGroupDatabaseEntry, error) {
	return vlan.ProtocolGroupDatabaseEntry{
		DbIndex:         dbIndex,
		FrameFormatType: vlan.ProtocolFrameFormatType_PROTOCOL_FRAME_FORMAT_ETHERNET,
		Ethertype:       &ethertype,
		GroupId:         groupID,
	}, nil
}

// NewRFC1042ProtocolGroup creates a protocol group database entry for an
// RFC 1042 frame identified by its protocol ID.
func NewRFC1042ProtocolGroup(
	dbIndex uint32,
	protocolID string,
	groupID uint32,
) (vlan.ProtocolGroupDatabaseEntry, error) {
	if protocolID == "" {
		return vlan.ProtocolGroupDatabaseEntry{}, fmt.Errorf("protocol ID cannot be empty")
	}

	return vlan.ProtocolGroupDatabaseEntry{
		DbIndex:         dbIndex,
		FrameFormatType: vlan.ProtocolFrameFormatType_PROTOCOL_FRAME_FORMAT_RFC1042,
		ProtocolId:      &protocolID,
		GroupId:         groupID,
	}, nil
}

// NewSNAP8021HProtocolGroup creates a protocol group database entry for
// SNAP 802.1H frames identified by their protocol ID.
func NewSNAP8021HProtocolGroup(
	dbIndex uint32,
	protocolID string,
	groupID uint32,
) (vlan.ProtocolGroupDatabaseEntry, error) {
	if protocolID == "" {
		return vlan.ProtocolGroupDatabaseEntry{}, fmt.Errorf("protocol ID cannot be empty")
	}

	return vlan.ProtocolGroupDatabaseEntry{
		DbIndex:         dbIndex,
		FrameFormatType: vlan.ProtocolFrameFormatType_PROTOCOL_FRAME_FORMAT_SNAP8021H,
		ProtocolId:      &protocolID,
		GroupId:         groupID,
	}, nil
}

// NewSNAPOtherProtocolGroup creates a protocol group database entry for
// other SNAP frames identified by their protocol ID.
func NewSNAPOtherProtocolGroup(
	dbIndex uint32,
	protocolID string,
	groupID uint32,
) (vlan.ProtocolGroupDatabaseEntry, error) {
	if protocolID == "" {
		return vlan.ProtocolGroupDatabaseEntry{}, fmt.Errorf("protocol ID cannot be empty")
	}

	return vlan.ProtocolGroupDatabaseEntry{
		DbIndex:         dbIndex,
		FrameFormatType: vlan.ProtocolFrameFormatType_PROTOCOL_FRAME_FORMAT_SNAP_OTHER,
		ProtocolId:      &protocolID,
		GroupId:         groupID,
	}, nil
}

// NewLLCOtherProtocolGroup creates a protocol group database entry for
// other LLC frames identified by a DSAP/SSAP address pair.
func NewLLCOtherProtocolGroup(
	dbIndex uint32,
	llcAddressPair string,
	groupID uint32,
) (vlan.ProtocolGroupDatabaseEntry, error) {
	if llcAddressPair == "" {
		return vlan.ProtocolGroupDatabaseEntry{}, fmt.Errorf("LLC address pair cannot be empty")
	}

	return vlan.ProtocolGroupDatabaseEntry{
		DbIndex:         dbIndex,
		FrameFormatType: vlan.ProtocolFrameFormatType_PROTOCOL_FRAME_FORMAT_LLC_OTHER,
		LlcAddressPair:  &llcAddressPair,
		GroupId:         groupID,
	}, nil
}

// -----------------------------------------------------------------------------
// Protocol Group VLAN Sets
// -----------------------------------------------------------------------------

// NewProtocolGroupVidSet creates a mapping between a protocol group and the
// VLAN identifiers assigned to that group.
func NewProtocolGroupVidSet(
	groupID uint32,
	vlanIDs []uint32,
) (vlan.ProtocolGroupVidSet, error) {
	if err := validateVlanIDs(vlanIDs); err != nil {
		return vlan.ProtocolGroupVidSet{}, fmt.Errorf("invalid VLAN IDs: %w", err)
	}

	return vlan.ProtocolGroupVidSet{
		GroupId: groupID,
		VlanIds: vlanIDs,
	}, nil
}
