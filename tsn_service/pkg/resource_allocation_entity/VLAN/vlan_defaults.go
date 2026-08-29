package vlan

import (
	"fmt"

	topologyconfig "OpenCNC/common/structures/topology_config"
	"OpenCNC/common/structures/vlan"
)

/*
Given a bridge NodeConfig with ports "eth0", "eth1", and "eth2":

    err := PopulateDefaultVlanConfig(nodeConfig)

The function will populate the missing VLAN configuration approximately as:

    Bridge:
        VLAN 1 -> label "default"
        VLAN 1 -> FID 1
        VLAN 1 -> static registration
            eth0 -> untagged
            eth1 -> untagged
            eth2 -> untagged

    Each port:
        default_vlan_id = 1
        VLAN membership:
            VLAN 1 -> untagged
        VLAN advanced:
            admit all frames
            ingress filtering disabled
            restricted VLAN registration disabled

N.b.
the function preserves existing VLAN configuration. Only missing defaults are added. E.g. if you have vlan 100 in the nodeConfig, it will not be removed or altered.
After calling PopulateDefaultVlanConfig(), nodeConfig itself has been populated. You don't need to use the returned value because the function modifies the existing protobuf object.

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

if err := vlan.PopulateDefaultVlanConfig(nodeConfig); err != nil {
	return fmt.Errorf("populate VLAN defaults: %w", err)
}

*/

const (
	// IEEE 802.1Q defines VID 1 as the conventional default PVID.
	defaultVlanID uint32 = 1

	// The baseline VLAN is associated with the first/default Filtering
	// Database. More elaborate FID assignments are deployment-specific.
	defaultFID uint32 = 1

	defaultVlanLabel = "default"
)

// -----------------------------------------------------------------------------
// Port VLAN defaults
// -----------------------------------------------------------------------------

// NewDefaultPortVlanAdvancedConfig returns the baseline 802.1Q bridge-port
// VLAN configuration.
//
// These values follow the conventional 802.1Q defaults:
//
//   - PVID is handled separately by PortConfig.DefaultVlanId.
//   - Admit all frame types.
//   - Ingress filtering disabled.
//   - Restricted VLAN registration disabled.
//
// These are protocol defaults, not TSN-specific policy. A TSN deployment
// should explicitly configure VLAN admission and filtering when deterministic
// VLAN isolation is required.
func NewDefaultPortVlanAdvancedConfig() vlan.PortVlanAdvancedConfig {
	acceptableFrame :=
		vlan.AcceptableFrameType_ACCEPTABLE_FRAME_TYPE_ADMIT_ALL_FRAMES

	ingressFiltering := false
	restrictedVlanRegistration := false

	config, err := NewPortVlanAdvancedConfig(
		acceptableFrame,
		ingressFiltering,
		restrictedVlanRegistration,
	)
	if err != nil {
		// All values above are known-valid enum/constants, so reaching this
		// point indicates an internal programming error rather than invalid
		// user input.
		panic(err)
	}

	return config
}

// -----------------------------------------------------------------------------
// Bridge VLAN defaults
// -----------------------------------------------------------------------------

// NewDefaultBridgeVlanConfig creates the baseline bridge-level VLAN
// configuration.
//
// The configuration contains the conventional default VLAN (VID 1), its
// human-readable label, and its VID-to-FID mapping.
//
// Port-specific configuration is intentionally not created here because the
// bridge-level constructor does not know which ports belong to the bridge.
// PopulateDefaultVlanConfig should be used when a NodeConfig is available.
func NewDefaultBridgeVlanConfig() vlan.BridgeVlanConfig {
	return vlan.BridgeVlanConfig{
		VlanLabels:              make([]*vlan.VlanLabel, 0, 1),
		VlanRegistrationEntries: make([]*vlan.VlanRegistrationEntry, 0, 1),
		VidToFidMappings:        make([]*vlan.VidToFidMapping, 0, 1),
		ProtocolGroupDatabase:   make([]*vlan.ProtocolGroupDatabaseEntry, 0),
	}
}

// -----------------------------------------------------------------------------
// NodeConfig population
// -----------------------------------------------------------------------------

// PopulateDefaultVlanConfig populates the missing VLAN configuration of a
// bridge NodeConfig with the conventional 802.1Q baseline.
//
// The function uses the ports already present in nodeConfig. It does not
// invent ports and does not modify an explicitly configured VLAN value.
//
// Bridge-level defaults:
//   - VID 1 exists and is labelled "default".
//   - VID 1 maps to FID 1.
//   - VID 1 has a static VLAN registration entry covering the configured
//     bridge ports.
//   - The static registration uses Registration Fixed (New ignored).
//   - VID 1 is transmitted untagged on those ports.
//
// Port-level defaults:
//   - PVID/default VLAN is VID 1.
//   - VID 1 is an untagged VLAN membership.
//   - Advanced VLAN configuration admits all frames.
//   - Ingress filtering is disabled.
//   - Restricted VLAN registration is disabled.
//
// Existing configuration is preserved. In particular, this function never
// replaces an explicitly configured VLAN ID, membership, advanced VLAN
// configuration, label, or VID-to-FID mapping.
//
// This function intentionally does not create TSN-specific VLANs. TSN VLAN
// identifiers and their port membership are topology/application policy and
// should be explicitly supplied by the caller.
func PopulateDefaultVlanConfig(nodeConfig *topologyconfig.NodeConfig) error {
	if nodeConfig == nil {
		return fmt.Errorf("node config must not be nil")
	}

	// VLAN bridge configuration is only applicable to bridge nodes.
	if nodeConfig.Bridge == nil {
		return nil
	}

	// Create the bridge-level VLAN configuration when it is absent.
	if nodeConfig.Bridge.VlanConfig == nil {
		config := NewDefaultBridgeVlanConfig()
		nodeConfig.Bridge.VlanConfig = &config
	}

	vlanConfig := nodeConfig.Bridge.VlanConfig

	// -------------------------------------------------------------------------
	// Bridge-level default VLAN
	// -------------------------------------------------------------------------

	if !hasVlanLabel(vlanConfig, defaultVlanID) {
		label, err := NewVlanLabel(
			defaultVlanID,
			defaultVlanLabel,
		)
		if err != nil {
			return fmt.Errorf("create default VLAN label: %w", err)
		}

		vlanConfig.VlanLabels = append(
			vlanConfig.VlanLabels,
			&label,
		)
	}

	if !hasVidToFidMapping(vlanConfig, defaultVlanID) {
		mapping, err := NewVidToFidMapping(
			defaultVlanID,
			defaultFID,
		)
		if err != nil {
			return fmt.Errorf("create default VID-to-FID mapping: %w", err)
		}

		vlanConfig.VidToFidMappings = append(
			vlanConfig.VidToFidMappings,
			&mapping,
		)
	}

	// -------------------------------------------------------------------------
	// Port-level defaults
	// -------------------------------------------------------------------------

	for _, portConfig := range nodeConfig.PortConfigs {
		if portConfig == nil {
			continue
		}

		if portConfig.PortId == "" {
			return fmt.Errorf("port configuration contains empty port ID")
		}

		// PVID 1 is the conventional default PVID. Do not overwrite an
		// explicitly configured PVID.
		if portConfig.DefaultVlanId == nil {
			vlanID := defaultVlanID
			portConfig.DefaultVlanId = &vlanID
		}

		// A port using the default PVID must have membership in that VLAN.
		// The default VLAN is transmitted untagged on an access-style
		// baseline configuration.
		if !hasVlanMembership(portConfig, defaultVlanID) {
			membership, err := NewVlanMembership(
				defaultVlanID,
				false,
			)
			if err != nil {
				return fmt.Errorf(
					"create default VLAN membership for port %q: %w",
					portConfig.PortId,
					err,
				)
			}

			portConfig.VlanMemberships = append(
				portConfig.VlanMemberships,
				&membership,
			)
		}

		// Populate the advanced VLAN controls only when the caller has not
		// provided them explicitly.
		if portConfig.VlanAdvanced == nil {
			advancedConfig := NewDefaultPortVlanAdvancedConfig()
			portConfig.VlanAdvanced = &advancedConfig
		}
	}

	// -------------------------------------------------------------------------
	// Default VLAN registration
	// -------------------------------------------------------------------------

	// A single static VLAN registration entry can describe the default VLAN
	// and contain a port map for every bridge port. This is preferable to
	// creating one registration entry per port.
	portMaps, err := buildDefaultVlanPortMaps(nodeConfig)
	if err != nil {
		return err
	}

	if len(portMaps) > 0 &&
		!hasStaticVlanRegistrationEntry(vlanConfig, defaultVlanID) {

		registration, err := NewVlanRegistrationEntry(
			[]uint32{defaultVlanID},
			defaultFID,
			vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_STATIC,
			portMaps,
		)
		if err != nil {
			return fmt.Errorf(
				"create default VLAN registration entry: %w",
				err,
			)
		}

		vlanConfig.VlanRegistrationEntries = append(
			vlanConfig.VlanRegistrationEntries,
			&registration,
		)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Default VLAN registration construction
// -----------------------------------------------------------------------------

// buildDefaultVlanPortMaps creates the static registration port map for the
// default VLAN.
//
// Registration is Fixed (New ignored) because this is a statically established
// VLAN membership. This prevents dynamic MVRP registration changes from
// turning the statically configured default membership into normal dynamic
// registration behaviour.
//
// The VLAN is transmitted untagged because VID 1 is the PVID/access VLAN in
// the baseline configuration.
func buildDefaultVlanPortMaps(nodeConfig *topologyconfig.NodeConfig) ([]*vlan.VlanPortMap, error) {
	portMaps := make(
		[]*vlan.VlanPortMap,
		0,
		len(nodeConfig.PortConfigs),
	)

	for _, portConfig := range nodeConfig.PortConfigs {
		if portConfig == nil {
			continue
		}

		if portConfig.PortId == "" {
			return nil, fmt.Errorf("port configuration contains empty port ID")
		}

		portMap, err := NewVlanPortMap(
			portConfig.PortId,
			vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_FIXED_NEW_IGNORED,
			vlan.VlanTransmitted_VLAN_TRANSMITTED_UNTAGGED,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create default VLAN port map for port %q: %w",
				portConfig.PortId,
				err,
			)
		}

		portMaps = append(
			portMaps,
			&portMap,
		)
	}

	return portMaps, nil
}

// -----------------------------------------------------------------------------
// Lookup helpers
// -----------------------------------------------------------------------------

// hasVlanLabel reports whether the bridge already contains a label for the
// specified VLAN.
//
// Existing labels are preserved because default population must not replace
// user-provided naming.
func hasVlanLabel(config *vlan.BridgeVlanConfig, vlanID uint32) bool {
	if config == nil {
		return false
	}

	for _, label := range config.VlanLabels {
		if label != nil && label.VlanId == vlanID {
			return true
		}
	}

	return false
}

// hasVidToFidMapping reports whether a VID already has a FID mapping.
//
// The existing mapping is preserved even if it points to a different FID.
// FID assignment can be an intentional deployment-specific decision.
func hasVidToFidMapping(config *vlan.BridgeVlanConfig, vid uint32) bool {
	if config == nil {
		return false
	}

	for _, mapping := range config.VidToFidMappings {
		if mapping != nil && mapping.Vid == vid {
			return true
		}
	}

	return false
}

// hasVlanMembership reports whether a port already has membership in the
// specified VLAN.
func hasVlanMembership(portConfig *topologyconfig.PortConfig, vlanID uint32) bool {
	if portConfig == nil {
		return false
	}

	for _, membership := range portConfig.VlanMemberships {
		if membership != nil && membership.VlanId == vlanID {
			return true
		}
	}

	return false
}

// hasStaticVlanRegistrationEntry reports whether the bridge already contains
// a static registration entry for the specified VLAN.
//
// The function intentionally checks only for the existence of the VLAN in a
// static registration entry. It does not inspect or modify the existing port
// map because an existing entry may represent intentional deployment policy.
func hasStaticVlanRegistrationEntry(config *vlan.BridgeVlanConfig, vlanID uint32) bool {
	if config == nil {
		return false
	}

	for _, registration := range config.VlanRegistrationEntries {
		if registration == nil {
			continue
		}

		if registration.EntryType !=
			vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_STATIC {
			continue
		}

		if containsVlanID(registration.VlanIds, vlanID) {
			return true
		}
	}

	return false
}

// containsVlanID reports whether vlanID is present in vlanIDs.
func containsVlanID(vlanIDs []uint32, vlanID uint32) bool {
	for _, id := range vlanIDs {
		if id == vlanID {
			return true
		}
	}

	return false
}
