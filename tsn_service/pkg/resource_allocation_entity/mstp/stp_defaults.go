package mstp

import (
	"fmt"

	stp "OpenCNC/common/structures/stp"
	topologyconfig "OpenCNC/common/structures/topology_config"
)

/*
Given a bridge NodeConfig with ports "eth0", "eth1", and "eth2":

	err := PopulateDefaultSTPConfig(nodeConfig)

The function populates the missing STP configuration approximately as:

	Bridge:
	    STP enabled = false
	    protocol = RSTP
	    bridge priority = 32768
	    hello time = 2 seconds
	    max age = 20 seconds
	    forward delay = 15 seconds
	    max hops = 20
	    restricted role = false
	    restricted TCN = false

	Each port:
	    port priority = 128
	    path cost = 0 (automatic)
	    edge port = false
	    MAC enabled = true
	    restricted role = false
	    restricted TCN = false
	    protocol migration = true
	    BPDU RX enabled = true
	    BPDU TX enabled = true
	    L2 gateway port = false
	    point-to-point = auto
	    BPDU guard = false
	    BPDU filter = false

The function modifies nodeConfig directly. No returned STP configuration
is required.

Existing STP configuration is preserved.

In particular, this function does not overwrite an existing
StpConfiguration, StpBridgeConfiguration, or StpPortConfiguration.

If an STP submessage is absent, the appropriate default submessage is
created and populated.

The function intentionally does not create MSTIs, VLAN-to-MSTI mappings,
FID-to-MSTI mappings, or an MST configuration unless the caller explicitly
creates/configures MST operation.

Those are configuration policy rather than universal bridge defaults.
*/

// ============================================================================
// IEEE / STP DEFAULT VALUES
// ============================================================================
//
// This section contains DEFAULT VALUES only.
//
// Validation limits and validation helpers belong in stp_helpers.go.
//
// The values are based on the IEEE 802.1 STP/RSTP/MSTP model. Where IEEE
// specifies a default or conventional initial value, that value is kept here
// so the defaults layer has a single authoritative location.
//
// A value appearing here does NOT automatically mean that the same value is
// a validation limit.
//
// For example:
//
//	32768
//
// is the normal default bridge priority, while the valid bridge-priority
// range is handled separately by the validation helpers.
//
// ============================================================================

// -----------------------------------------------------------------------------
// Bridge STP defaults
// -----------------------------------------------------------------------------

const (
	// Default STP administrative enable state.
	//
	// STP/RSTP/MSTP is not administratively enabled merely because a bridge
	// supports the protocol.
	DefaultSTPEnabled = false

	// Default protocol operation.
	//
	// RSTP is the baseline OpenCNC spanning-tree operating mode.
	DefaultSTPProtocolMode = stp.StpProtocolMode_STP_PROTOCOL_RSTP

	// IEEE default bridge priority.
	//
	// Bridge priority is represented in increments of 4096.
	DefaultSTPBridgePriority uint32 = 32768

	// IEEE default STP/RSTP hello time.
	DefaultSTPHelloTimeSeconds uint32 = 2

	// IEEE default maximum age.
	DefaultSTPMaxAgeSeconds uint32 = 20

	// IEEE default forward delay.
	DefaultSTPForwardDelaySeconds uint32 = 15

	// IEEE MSTP default CIST maximum hop count.
	DefaultSTPMaxHops uint32 = 20

	// A normal bridge is not restricted from becoming root by default.
	DefaultSTPRestrictedRole = false

	// Topology-change propagation is not restricted by default.
	DefaultSTPRestrictedTCN = false
)

// -----------------------------------------------------------------------------
// Port STP defaults
// -----------------------------------------------------------------------------

const (
	// IEEE-compatible default port priority.
	//
	// Port priority is represented in increments of 16.
	DefaultSTPPortPriority uint32 = 128

	// A path cost of zero is the OpenCNC representation for automatic
	// path-cost calculation.
	//
	// IEEE does not define "zero" as a universal path-cost default. The
	// implementation calculates the operational cost from the port/link
	// characteristics when this value is zero.
	DefaultSTPPathCost uint32 = 0

	// Ports are not edge ports by default.
	DefaultSTPEdgePort = false

	// STP MAC operation is enabled for a participating port.
	DefaultSTPMACEnabled = true

	// A port is not restricted from becoming a root port by default.
	DefaultSTPPortRestrictedRole = false

	// Topology-change propagation is not restricted on a port by default.
	DefaultSTPPortRestrictedTCN = false

	// Protocol migration is permitted by default.
	DefaultSTPProtocolMigration = true

	// BPDUs are received during normal STP operation.
	DefaultSTPBPDURXEnabled = true

	// BPDUs are transmitted during normal STP operation.
	DefaultSTPBPDUtxEnabled = true

	// L2 gateway behavior is disabled by default.
	//
	// This field is retained for compatibility with the existing OpenCNC
	// implementation and is not an IEEE universal STP default.
	DefaultSTPL2GatewayPort = false

	// Point-to-point status is determined automatically by default.
	DefaultSTPPointToPoint = stp.PointToPointMode_POINT_TO_POINT_AUTO

	// BPDU guard is disabled by default.
	DefaultSTPBPDUGuard = false

	// BPDU filtering is disabled by default.
	DefaultSTPBPDUFilter = false
)

// -----------------------------------------------------------------------------
// MST defaults
// -----------------------------------------------------------------------------

const (
	// IEEE default MST configuration revision level.
	DefaultMSTRevisionLevel uint32 = 0

	// IEEE MST configuration format selector.
	DefaultMSTFormatSelector uint32 = 0

	// IEEE-compatible default bridge priority for an MSTI.
	DefaultMSTIBridgePriority uint32 = 32768

	// IEEE-compatible default MSTI port priority.
	DefaultMSTIPortPriority uint32 = 128

	// Automatic MSTI path-cost calculation.
	//
	// As with the common port path cost, zero is an OpenCNC representation
	// for automatic calculation rather than an IEEE numeric cost.
	DefaultMSTIPathCost uint32 = 0
)

// ============================================================================
// Default constructors
// ============================================================================

// NewDefaultSTPBridgeConfiguration returns a new bridge-level STP
// configuration populated with the OpenCNC/IEEE baseline defaults.
//
// This constructor does not create MST configuration or spanning-tree
// instances. Those structures represent explicit MST deployment policy.
func NewDefaultSTPBridgeConfiguration() stp.StpBridgeConfiguration {
	return stp.StpBridgeConfiguration{
		BridgePriority:      DefaultSTPBridgePriority,
		HelloTimeSeconds:    DefaultSTPHelloTimeSeconds,
		MaxAgeSeconds:       DefaultSTPMaxAgeSeconds,
		ForwardDelaySeconds: DefaultSTPForwardDelaySeconds,
		MaxHops:             DefaultSTPMaxHops,
		RestrictedRole:      DefaultSTPRestrictedRole,
		RestrictedTcn:       DefaultSTPRestrictedTCN,
	}
}

// NewDefaultSTPConfiguration returns a complete bridge-level STP
// configuration populated with the baseline defaults.
//
// The STP enabled field is optional in the protobuf and is therefore set
// explicitly so that the administrative default is represented as a
// present value rather than being confused with protobuf absence.
//
// MST configuration is intentionally not created here.
func NewDefaultSTPConfiguration() stp.StpConfiguration {
	enabled := DefaultSTPEnabled
	bridge := NewDefaultSTPBridgeConfiguration()

	return stp.StpConfiguration{
		StpEnabled:   &enabled,
		ProtocolMode: DefaultSTPProtocolMode,
		Bridge:       &bridge,
	}
}

// NewDefaultSTPPortConfiguration returns a new port-level STP
// configuration populated with the baseline defaults.
//
// The port ID is intentionally not populated because it is topology-specific
// and comes from the containing PortConfig.
func NewDefaultSTPPortConfiguration() stp.StpPortConfiguration {
	return stp.StpPortConfiguration{
		PortPriority:      DefaultSTPPortPriority,
		PathCost:          DefaultSTPPathCost,
		EdgePort:          DefaultSTPEdgePort,
		MacEnabled:        DefaultSTPMACEnabled,
		RestrictedRole:    DefaultSTPPortRestrictedRole,
		RestrictedTcn:     DefaultSTPPortRestrictedTCN,
		ProtocolMigration: DefaultSTPProtocolMigration,
		BpduRxEnabled:     DefaultSTPBPDURXEnabled,
		BpduTxEnabled:     DefaultSTPBPDUtxEnabled,
		L2GatewayPort:     DefaultSTPL2GatewayPort,
		PointToPoint:      DefaultSTPPointToPoint,
		BpduGuard:         DefaultSTPBPDUGuard,
		BpduFilter:        DefaultSTPBPDUFilter,
	}
}

// NewDefaultMSTIConfiguration returns a new MSTI bridge configuration.
//
// The MSTI identifier must be supplied by the caller because the identifier
// is deployment-specific.
func NewDefaultMSTIConfiguration(
	mstiID uint32,
) stp.SpanningTreeInstance {
	return stp.SpanningTreeInstance{
		MstiId:         mstiID,
		BridgePriority: DefaultMSTIBridgePriority,
	}
}

// NewDefaultMSTIPortConfiguration returns a new MSTI-specific port
// configuration.
//
// The MSTI identifier must be supplied by the caller.
//
// RestrictedRole is intentionally left unset because the protobuf models
// this field as optional. This preserves protobuf presence semantics.
func NewDefaultMSTIPortConfiguration(
	mstiID uint32,
) stp.MstiPortConfiguration {
	return stp.MstiPortConfiguration{
		MstiId:       mstiID,
		PortPriority: DefaultMSTIPortPriority,
		PathCost:     DefaultMSTIPathCost,
	}
}

// NewDefaultMSTConfigurationIdentifier returns a new MST configuration
// identifier populated with IEEE baseline values.
//
// The configuration name is intentionally empty because a meaningful MST
// region name is deployment-specific.
func NewDefaultMSTConfigurationIdentifier() stp.MstConfigurationIdentifier {
	return stp.MstConfigurationIdentifier{
		RevisionLevel:  DefaultMSTRevisionLevel,
		FormatSelector: DefaultMSTFormatSelector,
	}
}

// ============================================================================
// NodeConfig population
// ============================================================================

// PopulateDefaultSTPConfig populates the missing STP configuration of a
// bridge NodeConfig.
//
// The NodeConfig is modified in place.
//
// Behavior:
//
//   - A non-bridge NodeConfig is ignored.
//   - A missing bridge STP configuration is created completely.
//   - A missing port STP configuration is created completely.
//   - Existing STP configuration is preserved.
//   - Existing port STP configuration is preserved.
//   - Port IDs are copied into newly-created StpPortConfiguration messages.
//   - No MSTIs are invented.
//   - No MST VLAN/FID mappings are invented.
//   - No MST configuration name is invented.
//   - No bridge MAC address is invented.
//   - No pseudo-root ID is invented.
//
// This separation is intentional:
//
//	stp_defaults.go
//	    -> establishes the baseline configuration
//
//	stp_helpers.go
//	    -> validates and manipulates configuration
//
//	stp.proto
//	    -> remains the STP configuration data model
func PopulateDefaultSTPConfig(
	nodeConfig *topologyconfig.NodeConfig,
) error {
	if nodeConfig == nil {
		return fmt.Errorf("node config must not be nil")
	}

	// STP configuration is meaningful for bridge nodes.
	if nodeConfig.Bridge == nil {
		return nil
	}

	// -------------------------------------------------------------------------
	// Bridge-level STP configuration
	// -------------------------------------------------------------------------

	if nodeConfig.Bridge.StpConfiguration == nil {
		configuration := NewDefaultSTPConfiguration()

		nodeConfig.Bridge.StpConfiguration = &configuration
	}

	// -------------------------------------------------------------------------
	// Port-level STP configuration
	// -------------------------------------------------------------------------

	for _, portConfig := range nodeConfig.PortConfigs {
		if portConfig == nil {
			continue
		}

		if portConfig.PortId == "" {
			return fmt.Errorf(
				"port configuration contains empty port ID",
			)
		}

		if portConfig.Stp == nil {
			configuration := NewDefaultSTPPortConfiguration()

			// Port identity belongs to topology_config.PortConfig, so the
			// defaults layer copies it into the standalone STP port model.
			configuration.PortId = portConfig.PortId

			portConfig.Stp = &configuration
		}
	}

	return nil
}

// ============================================================================
// MST configuration population
// ============================================================================

// PopulateDefaultMSTConfiguration ensures that the bridge-level MST
// configuration and its configuration identifier exist.
//
// This function should normally be called only when the caller has selected
// MSTP operation.
//
// It does not create MSTIs or VLAN/FID mappings because those are explicit
// network configuration.
func PopulateDefaultMSTConfiguration(
	configuration *stp.StpConfiguration,
) error {
	if configuration == nil {
		return fmt.Errorf("STP configuration must not be nil")
	}

	if configuration.ProtocolMode !=
		stp.StpProtocolMode_STP_PROTOCOL_MSTP {
		return fmt.Errorf(
			"MST configuration requires MSTP protocol mode, got %s",
			configuration.ProtocolMode.String(),
		)
	}

	if configuration.MstConfig == nil {
		configuration.MstConfig = &stp.BridgeMstConfig{}
	}

	if configuration.MstConfig.ConfigurationIdentifier == nil {
		identifier := NewDefaultMSTConfigurationIdentifier()

		configuration.MstConfig.ConfigurationIdentifier = &identifier
	}

	return nil
}

// ============================================================================
// MSTI default population
// ============================================================================

// PopulateDefaultMSTI adds a new MSTI using the IEEE-compatible baseline
// bridge priority.
//
// Existing MSTI configuration is preserved.
//
// This function intentionally does not create an MSTI merely because the
// bridge operates in MSTP mode. MSTI creation and VLAN-to-MSTI assignment
// are explicit network policy.
func PopulateDefaultMSTI(
	configuration *stp.StpConfiguration,
	mstiID uint32,
) error {
	if configuration == nil {
		return fmt.Errorf("STP configuration must not be nil")
	}

	if configuration.ProtocolMode !=
		stp.StpProtocolMode_STP_PROTOCOL_MSTP {
		return fmt.Errorf(
			"MSTI configuration requires MSTP protocol mode, got %s",
			configuration.ProtocolMode.String(),
		)
	}

	// MSTI IDs are 1..4094.
	// MSTI 0 is the CIST and is intentionally rejected here.
	if err := validateMSTIID(mstiID); err != nil {
		return err
	}

	for _, instance := range configuration.Instances {
		if instance != nil && instance.GetMstiId() == mstiID {
			return nil
		}
	}

	instance := NewDefaultMSTIConfiguration(mstiID)

	configuration.Instances = append(
		configuration.Instances,
		&instance,
	)

	return nil
}

// ============================================================================
// MSTI port default population
// ============================================================================

// PopulateDefaultMSTIPortConfiguration creates the baseline configuration
// for an MSTI on a port.
//
// Existing MSTI port configuration is preserved.
//
// MSTI 0 is rejected because it represents the CIST.
func PopulateDefaultMSTIPortConfiguration(
	portConfiguration *stp.StpPortConfiguration,
	mstiID uint32,
) error {
	if portConfiguration == nil {
		return fmt.Errorf("STP port configuration must not be nil")
	}

	// MSTI-specific configuration accepts only MSTI IDs 1..4094.
	// CIST (0) is handled by the common port configuration.
	if err := validateMSTIID(mstiID); err != nil {
		return err
	}

	for _, configuration := range portConfiguration.MstiConfigurations {
		if configuration != nil && configuration.GetMstiId() == mstiID {
			return nil
		}
	}

	configuration := NewDefaultMSTIPortConfiguration(mstiID)

	portConfiguration.MstiConfigurations = append(
		portConfiguration.MstiConfigurations,
		&configuration,
	)

	return nil
}
