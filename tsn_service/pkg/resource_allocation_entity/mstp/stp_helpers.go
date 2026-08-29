package mstp

import (
	"errors"
	"fmt"

	stp "OpenCNC/common/structures/stp"
)

const (
	// IEEE 802.1Q VLAN Identifier range.
	minVLANID = 1
	maxVLANID = 4094

	// IEEE 802.1Q MSTID range.
	//
	// MSTID 0 is the CIST and is handled separately by helpers that operate
	// on the complete spanning-tree instance space.
	minMSTIID = 1
	maxMSTIID = 4094

	// IEEE 802.1Q FID range used by the MST FID-to-MSTID management object.
	minFID = 1
	maxFID = 4094

	// IEEE 802.1Q Bridge Identifier priority component.
	bridgePriorityStep = 4096
	minBridgePriority  = 0
	maxBridgePriority  = 61440

	// IEEE 802.1Q Port Identifier priority component.
	portPriorityStep = 16
	minPortPriority  = 0
	maxPortPriority  = 240

	// IEEE 802.1Q bridge timer ranges, in whole seconds.
	minHelloTime    = 1
	maxHelloTime    = 10
	minMaxAge       = 6
	maxMaxAge       = 40
	minForwardDelay = 4
	maxForwardDelay = 30
	minMSTMaxHops   = 1
	maxMSTMaxHops   = 255

	// IEEE 802.1Q MST Configuration Identifier.
	maxMSTConfigurationNameLength  = 32
	mstConfigurationFormatSelector = 0
	maxMSTRevisionLevel            = 65535

	// IEEE 802.1Q Bridge Address is a 48-bit MAC address.
	bridgeAddressLength = 6

	// OpenCNC compatibility field retained by the protobuf.
	pseudoRootIDLength = 8
)

// -----------------------------------------------------------------------------
// Validation helpers
// -----------------------------------------------------------------------------

// validateBridgePriority validates the priority component of a Bridge
// Identifier.
//
// IEEE 802.1Q represents the configurable Bridge Identifier priority in
// increments of 4096:
//
//	0, 4096, 8192, ... 61440
func validateBridgePriority(priority uint32) error {
	if priority < minBridgePriority ||
		priority > maxBridgePriority ||
		priority%bridgePriorityStep != 0 {
		return fmt.Errorf(
			"invalid bridge priority %d: must be an IEEE value from %d to %d in increments of %d",
			priority,
			minBridgePriority,
			maxBridgePriority,
			bridgePriorityStep,
		)
	}

	return nil
}

// validatePortPriority validates the priority component of a Port Identifier.
//
// IEEE 802.1Q represents the configurable port priority in increments of 16:
//
//	0, 16, 32, ... 240
func validatePortPriority(priority uint32) error {
	if priority < minPortPriority ||
		priority > maxPortPriority ||
		priority%portPriorityStep != 0 {
		return fmt.Errorf(
			"invalid port priority %d: must be an IEEE value from %d to %d in increments of %d",
			priority,
			minPortPriority,
			maxPortPriority,
			portPriorityStep,
		)
	}

	return nil
}

// validateMSTIID validates a non-CIST MST instance identifier.
//
// IEEE 802.1Q defines MSTIDs 1..4094 for MSTIs. MSTID 0 is the CIST and is
// therefore intentionally rejected by this helper.
func validateMSTIID(mstiID uint32) error {
	if mstiID < minMSTIID || mstiID > maxMSTIID {
		return fmt.Errorf(
			"invalid MSTI ID %d: must be between %d and %d",
			mstiID,
			minMSTIID,
			maxMSTIID,
		)
	}

	return nil
}

// validateSpanningTreeInstanceID validates the complete instance identifier
// space used by StpConfiguration.
//
// The protobuf represents the CIST as MSTI ID 0 and MSTIs as 1..4094.
func validateSpanningTreeInstanceID(mstiID uint32) error {
	if mstiID > maxMSTIID {
		return fmt.Errorf(
			"invalid spanning-tree instance ID %d: must be between 0 and %d",
			mstiID,
			maxMSTIID,
		)
	}

	return nil
}

// validateVLANID validates an IEEE 802.1Q VLAN identifier.
//
// VID 0 and VID 4095 are reserved. Valid VLAN IDs are 1..4094.
func validateVLANID(vlanID uint32) error {
	if vlanID < minVLANID || vlanID > maxVLANID {
		return fmt.Errorf(
			"invalid VLAN ID %d: must be between %d and %d",
			vlanID,
			minVLANID,
			maxVLANID,
		)
	}

	return nil
}

// validateFID validates an IEEE Filtering Database Identifier.
//
// The IEEE MSTP management model represents the FID-to-MSTID table using
// FIDs in the range 1..4094.
func validateFID(fid uint32) error {
	if fid < minFID || fid > maxFID {
		return fmt.Errorf(
			"invalid FID %d: must be between %d and %d",
			fid,
			minFID,
			maxFID,
		)
	}

	return nil
}

// validateHelloTime validates the administrative Bridge Hello Time.
//
// IEEE 802.1Q management range: 1..10 seconds.
func validateHelloTime(seconds uint32) error {
	if seconds < minHelloTime || seconds > maxHelloTime {
		return fmt.Errorf(
			"invalid hello time %d: must be between %d and %d seconds",
			seconds,
			minHelloTime,
			maxHelloTime,
		)
	}

	return nil
}

// validateMaxAge validates the administrative Bridge Max Age.
//
// IEEE 802.1Q management range: 6..40 seconds.
func validateMaxAge(seconds uint32) error {
	if seconds < minMaxAge || seconds > maxMaxAge {
		return fmt.Errorf(
			"invalid max age %d: must be between %d and %d seconds",
			seconds,
			minMaxAge,
			maxMaxAge,
		)
	}

	return nil
}

// validateForwardDelay validates the administrative Bridge Forward Delay.
//
// IEEE 802.1Q management range: 4..30 seconds.
func validateForwardDelay(seconds uint32) error {
	if seconds < minForwardDelay || seconds > maxForwardDelay {
		return fmt.Errorf(
			"invalid forward delay %d: must be between %d and %d seconds",
			seconds,
			minForwardDelay,
			maxForwardDelay,
		)
	}

	return nil
}

// validateMSTMaxHops validates the administrative MST maximum hop count.
//
// The IEEE management model represents this as an unsigned value with a
// range of 1..255. A particular deployment may choose a smaller operational
// value; such policy belongs outside this low-level helper.
func validateMSTMaxHops(maxHops uint32) error {
	if maxHops < minMSTMaxHops || maxHops > maxMSTMaxHops {
		return fmt.Errorf(
			"invalid MST max hops %d: must be between %d and %d",
			maxHops,
			minMSTMaxHops,
			maxMSTMaxHops,
		)
	}

	return nil
}

// validateMSTConfigurationName validates the size of an MST Configuration
// Name.
//
// The IEEE MST Configuration Identifier allocates 32 octets for the name.
// No default or automatic name is generated here.
func validateMSTConfigurationName(name string) error {
	if len(name) > maxMSTConfigurationNameLength {
		return fmt.Errorf(
			"invalid MST configuration name: length %d exceeds maximum length %d bytes",
			len(name),
			maxMSTConfigurationNameLength,
		)
	}

	return nil
}

// validateMSTRevisionLevel validates the IEEE MST revision level.
func validateMSTRevisionLevel(revisionLevel uint32) error {
	if revisionLevel > maxMSTRevisionLevel {
		return fmt.Errorf(
			"invalid MST revision level %d: must be between 0 and %d",
			revisionLevel,
			maxMSTRevisionLevel,
		)
	}

	return nil
}

// validateMSTFormatSelector validates the IEEE MST Configuration Identifier
// Format Selector.
//
// IEEE 802.1Q defines value 0 for the format specified by the standard.
func validateMSTFormatSelector(formatSelector uint32) error {
	if formatSelector != mstConfigurationFormatSelector {
		return fmt.Errorf(
			"invalid MST configuration format selector %d: IEEE 802.1Q requires %d",
			formatSelector,
			mstConfigurationFormatSelector,
		)
	}

	return nil
}

// validateBridgeAddress validates an administrative bridge MAC address.
//
// An empty value is accepted because the protobuf explicitly defines an
// empty bridge_address as "derive/use the implementation bridge address".
// When supplied, the address must be exactly six octets.
func validateBridgeAddress(address []byte) error {
	if len(address) != 0 && len(address) != bridgeAddressLength {
		return fmt.Errorf(
			"invalid bridge address length %d: expected %d bytes",
			len(address),
			bridgeAddressLength,
		)
	}

	return nil
}

// validatePseudoRootID validates the OpenCNC compatibility pseudo-root ID.
//
// This field is not an IEEE primary administrative STP parameter. The
// protobuf retains it for compatibility with the existing implementation,
// which expects eight octets.
func validatePseudoRootID(pseudoRootID []byte) error {
	if len(pseudoRootID) != 0 && len(pseudoRootID) != pseudoRootIDLength {
		return fmt.Errorf(
			"invalid pseudo-root ID length %d: expected %d bytes",
			len(pseudoRootID),
			pseudoRootIDLength,
		)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Instance helpers
// -----------------------------------------------------------------------------

// FindSpanningTreeInstance returns the spanning-tree instance identified by
// mstiID.
//
// mstiID 0 represents the CIST.
// mstiID 1..4094 represent MSTIs.
func FindSpanningTreeInstance(
	configuration *stp.StpConfiguration,
	mstiID uint32,
) (*stp.SpanningTreeInstance, error) {
	if configuration == nil {
		return nil, errors.New("STP configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return nil, err
	}

	for _, instance := range configuration.GetInstances() {
		if instance != nil && instance.GetMstiId() == mstiID {
			return instance, nil
		}
	}

	return nil, fmt.Errorf(
		"spanning-tree instance %d not found",
		mstiID,
	)
}

// GetOrCreateSpanningTreeInstance returns an existing spanning-tree instance
// or creates an empty one.
//
// This helper never applies defaults.
func GetOrCreateSpanningTreeInstance(
	configuration *stp.StpConfiguration,
	mstiID uint32,
) (*stp.SpanningTreeInstance, error) {
	if configuration == nil {
		return nil, errors.New("STP configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return nil, err
	}

	if instance, err := FindSpanningTreeInstance(configuration, mstiID); err == nil {
		return instance, nil
	}

	instance := &stp.SpanningTreeInstance{
		MstiId: mstiID,
	}

	configuration.Instances = append(configuration.Instances, instance)

	return instance, nil
}

// RemoveSpanningTreeInstance removes a spanning-tree instance.
//
// The helper permits removing the CIST entry from the repeated field because
// the protobuf explicitly represents CIST as an instance with msti_id == 0.
// No default is restored after removal.
func RemoveSpanningTreeInstance(
	configuration *stp.StpConfiguration,
	mstiID uint32,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return err
	}

	instances := configuration.GetInstances()

	for i, instance := range instances {
		if instance != nil && instance.GetMstiId() == mstiID {
			configuration.Instances = append(instances[:i], instances[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf(
		"spanning-tree instance %d not found",
		mstiID,
	)
}

// -----------------------------------------------------------------------------
// MSTI port helpers
// -----------------------------------------------------------------------------

// FindMSTIPortConfiguration returns the MSTI-specific port configuration.
func FindMSTIPortConfiguration(
	portConfiguration *stp.StpPortConfiguration,
	mstiID uint32,
) (*stp.MstiPortConfiguration, error) {
	if portConfiguration == nil {
		return nil, errors.New("STP port configuration is nil")
	}

	if err := validateMSTIID(mstiID); err != nil {
		return nil, err
	}

	for _, configuration := range portConfiguration.GetMstiConfigurations() {
		if configuration != nil && configuration.GetMstiId() == mstiID {
			return configuration, nil
		}
	}

	return nil, fmt.Errorf(
		"MSTI %d port configuration not found",
		mstiID,
	)
}

// GetOrCreateMSTIPortConfiguration returns an existing MSTI-specific port
// configuration or creates an empty one.
//
// This helper never applies defaults.
func GetOrCreateMSTIPortConfiguration(
	portConfiguration *stp.StpPortConfiguration,
	mstiID uint32,
) (*stp.MstiPortConfiguration, error) {
	if portConfiguration == nil {
		return nil, errors.New("STP port configuration is nil")
	}

	if err := validateMSTIID(mstiID); err != nil {
		return nil, err
	}

	if configuration, err := FindMSTIPortConfiguration(portConfiguration, mstiID); err == nil {
		return configuration, nil
	}

	configuration := &stp.MstiPortConfiguration{
		MstiId: mstiID,
	}

	portConfiguration.MstiConfigurations =
		append(portConfiguration.MstiConfigurations, configuration)

	return configuration, nil
}

// RemoveMSTIPortConfiguration removes an MSTI-specific port configuration.
func RemoveMSTIPortConfiguration(
	portConfiguration *stp.StpPortConfiguration,
	mstiID uint32,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	if err := validateMSTIID(mstiID); err != nil {
		return err
	}

	configurations := portConfiguration.GetMstiConfigurations()

	for i, configuration := range configurations {
		if configuration != nil && configuration.GetMstiId() == mstiID {
			portConfiguration.MstiConfigurations =
				append(configurations[:i], configurations[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf(
		"MSTI %d port configuration not found",
		mstiID,
	)
}

// -----------------------------------------------------------------------------
// FID -> MSTI mapping helpers
// -----------------------------------------------------------------------------

// FindFIDToMSTIMapping returns the explicit FID-to-MSTI mapping for a FID.
func FindFIDToMSTIMapping(
	mstConfig *stp.BridgeMstConfig,
	fid uint32,
) (*stp.FidToMstidMapping, error) {
	if mstConfig == nil {
		return nil, errors.New("MST configuration is nil")
	}

	if err := validateFID(fid); err != nil {
		return nil, err
	}

	for _, mapping := range mstConfig.GetFidToMstidMappings() {
		if mapping != nil && mapping.GetFid() == fid {
			return mapping, nil
		}
	}

	return nil, fmt.Errorf(
		"FID %d mapping not found",
		fid,
	)
}

// GetOrCreateFIDToMSTIMapping returns an existing FID mapping or creates an
// empty mapping.
//
// No MSTI is assigned automatically.
func GetOrCreateFIDToMSTIMapping(
	mstConfig *stp.BridgeMstConfig,
	fid uint32,
) (*stp.FidToMstidMapping, error) {
	if mstConfig == nil {
		return nil, errors.New("MST configuration is nil")
	}

	if err := validateFID(fid); err != nil {
		return nil, err
	}

	if mapping, err := FindFIDToMSTIMapping(mstConfig, fid); err == nil {
		return mapping, nil
	}

	mapping := &stp.FidToMstidMapping{
		Fid: fid,
	}

	mstConfig.FidToMstidMappings =
		append(mstConfig.FidToMstidMappings, mapping)

	return mapping, nil
}

// RemoveFIDToMSTIMapping removes an explicit FID-to-MSTI mapping.
func RemoveFIDToMSTIMapping(
	mstConfig *stp.BridgeMstConfig,
	fid uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateFID(fid); err != nil {
		return err
	}

	mappings := mstConfig.GetFidToMstidMappings()

	for i, mapping := range mappings {
		if mapping != nil && mapping.GetFid() == fid {
			mstConfig.FidToMstidMappings =
				append(mappings[:i], mappings[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf(
		"FID %d mapping not found",
		fid,
	)
}

// -----------------------------------------------------------------------------
// VLAN -> MSTI mapping helpers
// -----------------------------------------------------------------------------

// FindVIDToMSTIMapping returns the VLAN-to-MSTI mapping for a VLAN.
func FindVIDToMSTIMapping(
	mstConfig *stp.BridgeMstConfig,
	vlanID uint32,
) (*stp.VidToMstidMapping, error) {
	if mstConfig == nil {
		return nil, errors.New("MST configuration is nil")
	}

	if err := validateVLANID(vlanID); err != nil {
		return nil, err
	}

	for _, mapping := range mstConfig.GetVidToMstidMappings() {
		if mapping != nil && mapping.GetVlanId() == vlanID {
			return mapping, nil
		}
	}

	return nil, fmt.Errorf(
		"VLAN %d MSTI mapping not found",
		vlanID,
	)
}

// GetOrCreateVIDToMSTIMapping returns an existing VLAN mapping or creates an
// empty mapping.
//
// No MSTI is assigned automatically. In particular, this helper does not
// silently assign the VLAN to the CIST.
func GetOrCreateVIDToMSTIMapping(
	mstConfig *stp.BridgeMstConfig,
	vlanID uint32,
) (*stp.VidToMstidMapping, error) {
	if mstConfig == nil {
		return nil, errors.New("MST configuration is nil")
	}

	if err := validateVLANID(vlanID); err != nil {
		return nil, err
	}

	if mapping, err := FindVIDToMSTIMapping(mstConfig, vlanID); err == nil {
		return mapping, nil
	}

	mapping := &stp.VidToMstidMapping{
		VlanId: vlanID,
	}

	mstConfig.VidToMstidMappings =
		append(mstConfig.VidToMstidMappings, mapping)

	return mapping, nil
}

// RemoveVIDToMSTIMapping removes a VLAN-to-MSTI mapping.
func RemoveVIDToMSTIMapping(
	mstConfig *stp.BridgeMstConfig,
	vlanID uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateVLANID(vlanID); err != nil {
		return err
	}

	mappings := mstConfig.GetVidToMstidMappings()

	for i, mapping := range mappings {
		if mapping != nil && mapping.GetVlanId() == vlanID {
			mstConfig.VidToMstidMappings =
				append(mappings[:i], mappings[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf(
		"VLAN %d MSTI mapping not found",
		vlanID,
	)
}

// -----------------------------------------------------------------------------
// Grouped FID allocation helpers
// -----------------------------------------------------------------------------

// FindFIDRangeAllocation returns the grouped FID allocation for an instance.
//
// The target mst_id may be 0 for the CIST or 1..4094 for an MSTI.
func FindFIDRangeAllocation(
	mstConfig *stp.BridgeMstConfig,
	mstiID uint32,
) (*stp.FidRangeToMstidAllocation, error) {
	if mstConfig == nil {
		return nil, errors.New("MST configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return nil, err
	}

	for _, allocation := range mstConfig.GetFidToMstidAllocations() {
		if allocation != nil && allocation.GetMstId() == mstiID {
			return allocation, nil
		}
	}

	return nil, fmt.Errorf(
		"FID allocation for MSTI %d not found",
		mstiID,
	)
}

// GetOrCreateFIDRangeAllocation returns an existing grouped FID allocation
// or creates an empty allocation.
//
// No FIDs are added automatically.
func GetOrCreateFIDRangeAllocation(
	mstConfig *stp.BridgeMstConfig,
	mstiID uint32,
) (*stp.FidRangeToMstidAllocation, error) {
	if mstConfig == nil {
		return nil, errors.New("MST configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return nil, err
	}

	if allocation, err := FindFIDRangeAllocation(mstConfig, mstiID); err == nil {
		return allocation, nil
	}

	allocation := &stp.FidRangeToMstidAllocation{
		MstId: mstiID,
	}

	mstConfig.FidToMstidAllocations =
		append(mstConfig.FidToMstidAllocations, allocation)

	return allocation, nil
}

// RemoveFIDRangeAllocation removes a grouped FID allocation for an instance.
func RemoveFIDRangeAllocation(
	mstConfig *stp.BridgeMstConfig,
	mstiID uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return err
	}

	allocations := mstConfig.GetFidToMstidAllocations()

	for i, allocation := range allocations {
		if allocation != nil && allocation.GetMstId() == mstiID {
			mstConfig.FidToMstidAllocations =
				append(allocations[:i], allocations[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf(
		"FID allocation for MSTI %d not found",
		mstiID,
	)
}

// -----------------------------------------------------------------------------
// MST configuration helpers
// -----------------------------------------------------------------------------

// EnsureMSTConfiguration returns the bridge-level MST configuration,
// creating the protobuf message when necessary.
//
// No defaults are applied.
func EnsureMSTConfiguration(
	configuration *stp.StpConfiguration,
) (*stp.BridgeMstConfig, error) {
	if configuration == nil {
		return nil, errors.New("STP configuration is nil")
	}

	if configuration.MstConfig == nil {
		configuration.MstConfig = &stp.BridgeMstConfig{}
	}

	return configuration.MstConfig, nil
}

// EnsureMSTConfigurationIdentifier returns the MST Configuration Identifier,
// creating the protobuf message when necessary.
//
// No defaults are applied.
func EnsureMSTConfigurationIdentifier(
	mstConfig *stp.BridgeMstConfig,
) (*stp.MstConfigurationIdentifier, error) {
	if mstConfig == nil {
		return nil, errors.New("MST configuration is nil")
	}

	if mstConfig.ConfigurationIdentifier == nil {
		mstConfig.ConfigurationIdentifier =
			&stp.MstConfigurationIdentifier{}
	}

	return mstConfig.ConfigurationIdentifier, nil
}

// EnsureBridgeConfiguration returns the bridge-level STP configuration
// message, creating it when necessary.
//
// No defaults are applied.
func EnsureBridgeConfiguration(
	configuration *stp.StpConfiguration,
) (*stp.StpBridgeConfiguration, error) {
	if configuration == nil {
		return nil, errors.New("STP configuration is nil")
	}

	if configuration.Bridge == nil {
		configuration.Bridge = &stp.StpBridgeConfiguration{}
	}

	return configuration.Bridge, nil
}

// -----------------------------------------------------------------------------
// Bridge-level configuration helpers
// -----------------------------------------------------------------------------

// SetSTPEnabled updates the administrative STP enable state.
func SetSTPEnabled(
	configuration *stp.StpConfiguration,
	enabled bool,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	configuration.StpEnabled = &enabled

	return nil
}

// SetProtocolMode updates the administrative STP protocol mode.
//
// The unspecified protobuf enum value is rejected because this helper is
// intended to write an explicit protocol operation.
func SetProtocolMode(
	configuration *stp.StpConfiguration,
	mode stp.StpProtocolMode,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	switch mode {
	case stp.StpProtocolMode_STP_PROTOCOL_STP,
		stp.StpProtocolMode_STP_PROTOCOL_RSTP,
		stp.StpProtocolMode_STP_PROTOCOL_MSTP:
		configuration.ProtocolMode = mode
		return nil

	default:
		return fmt.Errorf(
			"invalid STP protocol mode %d",
			mode,
		)
	}
}

// SetBridgePriority updates the common/CIST bridge priority.
func SetBridgePriority(
	configuration *stp.StpConfiguration,
	priority uint32,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateBridgePriority(priority); err != nil {
		return err
	}

	bridge, err := EnsureBridgeConfiguration(configuration)
	if err != nil {
		return err
	}

	bridge.BridgePriority = priority

	return nil
}

// SetBridgeAddress updates the administrative bridge MAC address.
//
// The byte slice is copied so callers may safely reuse their input buffer.
func SetBridgeAddress(
	configuration *stp.StpConfiguration,
	address []byte,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateBridgeAddress(address); err != nil {
		return err
	}

	bridge, err := EnsureBridgeConfiguration(configuration)
	if err != nil {
		return err
	}

	bridge.BridgeAddress = append([]byte(nil), address...)

	return nil
}

// SetHelloTime updates the administrative STP/RSTP Hello Time.
func SetHelloTime(
	configuration *stp.StpConfiguration,
	seconds uint32,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateHelloTime(seconds); err != nil {
		return err
	}

	bridge, err := EnsureBridgeConfiguration(configuration)
	if err != nil {
		return err
	}

	bridge.HelloTimeSeconds = seconds

	return nil
}

// SetMaxAge updates the administrative STP/RSTP Max Age.
func SetMaxAge(
	configuration *stp.StpConfiguration,
	seconds uint32,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateMaxAge(seconds); err != nil {
		return err
	}

	bridge, err := EnsureBridgeConfiguration(configuration)
	if err != nil {
		return err
	}

	bridge.MaxAgeSeconds = seconds

	return nil
}

// SetForwardDelay updates the administrative STP Forward Delay.
func SetForwardDelay(
	configuration *stp.StpConfiguration,
	seconds uint32,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateForwardDelay(seconds); err != nil {
		return err
	}

	bridge, err := EnsureBridgeConfiguration(configuration)
	if err != nil {
		return err
	}

	bridge.ForwardDelaySeconds = seconds

	return nil
}

// SetMSTMaxHops updates the administrative CIST maximum hop count.
func SetMSTMaxHops(
	configuration *stp.StpConfiguration,
	maxHops uint32,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateMSTMaxHops(maxHops); err != nil {
		return err
	}

	bridge, err := EnsureBridgeConfiguration(configuration)
	if err != nil {
		return err
	}

	bridge.MaxHops = maxHops

	return nil
}

// SetRestrictedRole updates the bridge-level restrictedRole parameter.
func SetRestrictedRole(
	configuration *stp.StpConfiguration,
	restricted bool,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	bridge, err := EnsureBridgeConfiguration(configuration)
	if err != nil {
		return err
	}

	bridge.RestrictedRole = restricted

	return nil
}

// SetRestrictedTCN updates the bridge-level restrictedTcn parameter.
func SetRestrictedTCN(
	configuration *stp.StpConfiguration,
	restricted bool,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	bridge, err := EnsureBridgeConfiguration(configuration)
	if err != nil {
		return err
	}

	bridge.RestrictedTcn = restricted

	return nil
}

// -----------------------------------------------------------------------------
// Common port-level configuration helpers
// -----------------------------------------------------------------------------

// SetPortID updates the OpenCNC topology port identifier.
//
// The protobuf does not define an IEEE range for this field, so the helper
// only rejects an empty identifier.
func SetPortID(
	portConfiguration *stp.StpPortConfiguration,
	portID string,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	if portID == "" {
		return errors.New("STP port ID cannot be empty")
	}

	portConfiguration.PortId = portID

	return nil
}

// SetPortPriority updates a port's administrative priority.
func SetPortPriority(
	portConfiguration *stp.StpPortConfiguration,
	priority uint32,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	if err := validatePortPriority(priority); err != nil {
		return err
	}

	portConfiguration.PortPriority = priority

	return nil
}

// SetPathCost updates the common/CIST administrative path cost.
//
// The protobuf intentionally uses uint32 and does not impose a narrower
// IEEE range on the administrative value. Therefore this helper does not
// invent a vendor-specific upper bound; uint32 itself is the validation
// boundary imposed by the protobuf.
func SetPathCost(
	portConfiguration *stp.StpPortConfiguration,
	pathCost uint32,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.PathCost = pathCost

	return nil
}

// SetEdgePort updates the administrative edge-port behavior.
func SetEdgePort(
	portConfiguration *stp.StpPortConfiguration,
	edgePort bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.EdgePort = edgePort

	return nil
}

// SetMACEnabled updates administrative MAC operation.
func SetMACEnabled(
	portConfiguration *stp.StpPortConfiguration,
	enabled bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.MacEnabled = enabled

	return nil
}

// SetPortRestrictedRole updates the port restrictedRole behavior.
func SetPortRestrictedRole(
	portConfiguration *stp.StpPortConfiguration,
	restricted bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.RestrictedRole = restricted

	return nil
}

// SetPortRestrictedTCN updates the port restrictedTcn behavior.
func SetPortRestrictedTCN(
	portConfiguration *stp.StpPortConfiguration,
	restricted bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.RestrictedTcn = restricted

	return nil
}

// SetProtocolMigration updates the administrative protocol migration
// behavior.
func SetProtocolMigration(
	portConfiguration *stp.StpPortConfiguration,
	enabled bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.ProtocolMigration = enabled

	return nil
}

// SetBPDURXEnabled updates administrative BPDU reception.
func SetBPDURXEnabled(
	portConfiguration *stp.StpPortConfiguration,
	enabled bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.BpduRxEnabled = enabled

	return nil
}

// SetBPDUTXEnabled updates administrative BPDU transmission.
func SetBPDUTXEnabled(
	portConfiguration *stp.StpPortConfiguration,
	enabled bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.BpduTxEnabled = enabled

	return nil
}

// SetL2GatewayPort updates the implementation-specific L2 gateway-port
// behavior.
func SetL2GatewayPort(
	portConfiguration *stp.StpPortConfiguration,
	enabled bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.L2GatewayPort = enabled

	return nil
}

// SetPseudoRootID updates the retained compatibility pseudo-root identifier.
//
// The byte slice is copied.
func SetPseudoRootID(
	portConfiguration *stp.StpPortConfiguration,
	pseudoRootID []byte,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	if err := validatePseudoRootID(pseudoRootID); err != nil {
		return err
	}

	portConfiguration.PseudoRootId =
		append([]byte(nil), pseudoRootID...)

	return nil
}

// SetPointToPointMode updates the administrative point-to-point mode.
func SetPointToPointMode(
	portConfiguration *stp.StpPortConfiguration,
	mode stp.PointToPointMode,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	switch mode {
	case stp.PointToPointMode_POINT_TO_POINT_AUTO,
		stp.PointToPointMode_POINT_TO_POINT_TRUE,
		stp.PointToPointMode_POINT_TO_POINT_FALSE:
		portConfiguration.PointToPoint = mode
		return nil

	default:
		return fmt.Errorf(
			"invalid point-to-point mode %d",
			mode,
		)
	}
}

// SetBPDUGuard updates BPDU guard behavior.
func SetBPDUGuard(
	portConfiguration *stp.StpPortConfiguration,
	enabled bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.BpduGuard = enabled

	return nil
}

// SetBPDUFilter updates BPDU filtering behavior.
func SetBPDUFilter(
	portConfiguration *stp.StpPortConfiguration,
	enabled bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	portConfiguration.BpduFilter = enabled

	return nil
}

// -----------------------------------------------------------------------------
// MSTI-specific configuration helpers
// -----------------------------------------------------------------------------

// SetMSTIBridgePriority updates the bridge priority of a specific MSTI.
func SetMSTIBridgePriority(
	configuration *stp.StpConfiguration,
	mstiID uint32,
	priority uint32,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateMSTIID(mstiID); err != nil {
		return err
	}

	if err := validateBridgePriority(priority); err != nil {
		return err
	}

	instance, err := GetOrCreateSpanningTreeInstance(configuration, mstiID)
	if err != nil {
		return err
	}

	instance.BridgePriority = priority

	return nil
}

// SetMSTIInstanceID updates the identity of an existing spanning-tree
// instance.
//
// Changing an instance ID is allowed only when the new ID is not already
// present in the configuration.
func SetMSTIInstanceID(
	configuration *stp.StpConfiguration,
	currentMSTIID uint32,
	newMSTIID uint32,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateMSTIID(currentMSTIID); err != nil {
		return err
	}

	if err := validateMSTIID(newMSTIID); err != nil {
		return err
	}

	if currentMSTIID == newMSTIID {
		return nil
	}

	instance, err := FindSpanningTreeInstance(
		configuration,
		currentMSTIID,
	)
	if err != nil {
		return err
	}

	if existing, err := FindSpanningTreeInstance(
		configuration,
		newMSTIID,
	); err == nil && existing != nil {
		return fmt.Errorf(
			"spanning-tree instance %d already exists",
			newMSTIID,
		)
	}

	instance.MstiId = newMSTIID

	return nil
}

// SetSpanningTreeInstanceName updates the optional human-readable instance
// identifier.
//
// The protobuf does not impose a standard length or character restriction
// for this compatibility/convenience field.
func SetSpanningTreeInstanceName(
	configuration *stp.StpConfiguration,
	mstiID uint32,
	instanceID string,
) error {
	if configuration == nil {
		return errors.New("STP configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return err
	}

	instance, err := FindSpanningTreeInstance(configuration, mstiID)
	if err != nil {
		return err
	}

	instance.InstanceId = instanceID

	return nil
}

// SetMSTIPortPriority updates the priority for a specific MSTI on a port.
func SetMSTIPortPriority(
	portConfiguration *stp.StpPortConfiguration,
	mstiID uint32,
	priority uint32,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	if err := validateMSTIID(mstiID); err != nil {
		return err
	}

	if err := validatePortPriority(priority); err != nil {
		return err
	}

	configuration, err := GetOrCreateMSTIPortConfiguration(
		portConfiguration,
		mstiID,
	)
	if err != nil {
		return err
	}

	configuration.PortPriority = priority

	return nil
}

// SetMSTIPathCost updates the path cost for a specific MSTI on a port.
//
// No vendor-specific upper bound is invented because the protobuf defines
// this field as uint32 without an IEEE management range in this model.
func SetMSTIPathCost(
	portConfiguration *stp.StpPortConfiguration,
	mstiID uint32,
	pathCost uint32,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	if err := validateMSTIID(mstiID); err != nil {
		return err
	}

	configuration, err := GetOrCreateMSTIPortConfiguration(
		portConfiguration,
		mstiID,
	)
	if err != nil {
		return err
	}

	configuration.PathCost = pathCost

	return nil
}

// SetMSTIPortRestrictedRole updates the restrictedRole override for an MSTI
// port configuration.
//
// The field is optional in the protobuf, so the helper preserves presence
// semantics rather than merely assigning a bool value.
func SetMSTIPortRestrictedRole(
	portConfiguration *stp.StpPortConfiguration,
	mstiID uint32,
	restricted bool,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	if err := validateMSTIID(mstiID); err != nil {
		return err
	}

	configuration, err := GetOrCreateMSTIPortConfiguration(
		portConfiguration,
		mstiID,
	)
	if err != nil {
		return err
	}

	configuration.RestrictedRole = &restricted

	return nil
}

// ClearMSTIPortRestrictedRole removes the explicit MSTI restrictedRole
// override, allowing the absence of the field to have its defined semantic.
func ClearMSTIPortRestrictedRole(
	portConfiguration *stp.StpPortConfiguration,
	mstiID uint32,
) error {
	if portConfiguration == nil {
		return errors.New("STP port configuration is nil")
	}

	if err := validateMSTIID(mstiID); err != nil {
		return err
	}

	configuration, err := FindMSTIPortConfiguration(
		portConfiguration,
		mstiID,
	)
	if err != nil {
		return err
	}

	configuration.RestrictedRole = nil

	return nil
}

// -----------------------------------------------------------------------------
// FID / MSTI mapping setters
// -----------------------------------------------------------------------------

// SetFIDToMSTI updates or creates an explicit FID-to-MSTI mapping.
//
// CIST (MSTID 0) is permitted because the protobuf explicitly models the
// CIST as a valid mapping target.
func SetFIDToMSTI(
	mstConfig *stp.BridgeMstConfig,
	fid uint32,
	mstiID uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateFID(fid); err != nil {
		return err
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return err
	}

	mapping, err := GetOrCreateFIDToMSTIMapping(mstConfig, fid)
	if err != nil {
		return err
	}

	mapping.MstId = mstiID

	return nil
}

// SetVIDToMSTI updates or creates a VLAN-to-MSTI mapping.
//
// CIST (MSTID 0) is permitted because VLANs not assigned to an MSTI belong
// to the CIST.
func SetVIDToMSTI(
	mstConfig *stp.BridgeMstConfig,
	vlanID uint32,
	mstiID uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateVLANID(vlanID); err != nil {
		return err
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return err
	}

	mapping, err := GetOrCreateVIDToMSTIMapping(mstConfig, vlanID)
	if err != nil {
		return err
	}

	mapping.MstId = mstiID

	return nil
}

// AddFIDToMSTIRange adds an FID to a grouped FID-to-MSTI allocation.
//
// The helper validates the FID and target instance but does not introduce
// defaults or remove duplicate entries already present.
func AddFIDToMSTIRange(
	mstConfig *stp.BridgeMstConfig,
	mstiID uint32,
	fid uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return err
	}

	if err := validateFID(fid); err != nil {
		return err
	}

	allocation, err := GetOrCreateFIDRangeAllocation(mstConfig, mstiID)
	if err != nil {
		return err
	}

	for _, existingFID := range allocation.GetFids() {
		if existingFID == fid {
			return nil
		}
	}

	allocation.Fids = append(allocation.Fids, fid)

	return nil
}

// RemoveFIDFromMSTIRange removes an FID from a grouped FID-to-MSTI
// allocation.
func RemoveFIDFromMSTIRange(
	mstConfig *stp.BridgeMstConfig,
	mstiID uint32,
	fid uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateSpanningTreeInstanceID(mstiID); err != nil {
		return err
	}

	if err := validateFID(fid); err != nil {
		return err
	}

	allocation, err := FindFIDRangeAllocation(mstConfig, mstiID)
	if err != nil {
		return err
	}

	fids := allocation.GetFids()

	for i, existingFID := range fids {
		if existingFID == fid {
			allocation.Fids = append(fids[:i], fids[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf(
		"FID %d not found in MSTI %d allocation",
		fid,
		mstiID,
	)
}

// -----------------------------------------------------------------------------
// MST Configuration Identifier setters
// -----------------------------------------------------------------------------

// SetMSTConfigurationName updates the MST Configuration Name.
func SetMSTConfigurationName(
	mstConfig *stp.BridgeMstConfig,
	name string,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateMSTConfigurationName(name); err != nil {
		return err
	}

	identifier, err := EnsureMSTConfigurationIdentifier(mstConfig)
	if err != nil {
		return err
	}

	identifier.ConfigurationName = name

	return nil
}

// SetMSTRevisionLevel updates the MST Configuration Revision Level.
func SetMSTRevisionLevel(
	mstConfig *stp.BridgeMstConfig,
	revisionLevel uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateMSTRevisionLevel(revisionLevel); err != nil {
		return err
	}

	identifier, err := EnsureMSTConfigurationIdentifier(mstConfig)
	if err != nil {
		return err
	}

	identifier.RevisionLevel = revisionLevel

	return nil
}

// SetMSTFormatSelector updates the MST Configuration Identifier Format
// Selector.
//
// IEEE 802.1Q currently requires this value to be zero.
func SetMSTFormatSelector(
	mstConfig *stp.BridgeMstConfig,
	formatSelector uint32,
) error {
	if mstConfig == nil {
		return errors.New("MST configuration is nil")
	}

	if err := validateMSTFormatSelector(formatSelector); err != nil {
		return err
	}

	identifier, err := EnsureMSTConfigurationIdentifier(mstConfig)
	if err != nil {
		return err
	}

	identifier.FormatSelector = formatSelector

	return nil
}
