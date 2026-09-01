package netconf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"OpenCNC/common/observability"
	devicemodelregistry "OpenCNC/common/structures/devicemodelregistry"
	"OpenCNC/common/structures/topology"
	topology_config "OpenCNC/common/structures/topology_config"
	vlan "OpenCNC/common/structures/vlan"
	opencncModel "OpenCNC/config_service/opencnc_model"
	managementSessions "OpenCNC/config_service/pkg/managementSessions"
	"OpenCNC/config_service/pkg/plugins"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"
)

var _ plugins.Plugin = (*VlanNetconfPlugin_2021)(nil)

type VlanNetconfPlugin_2021 struct {
	logger observability.Logger
}

type bridgeVlanPayload struct {
	BridgeName    string
	ComponentName string
	Config        *vlan.BridgeVlanConfig
}

func NewVlanNetconfPlugin2021(logger observability.Logger) *VlanNetconfPlugin_2021 {
	return &VlanNetconfPlugin_2021{logger: observability.NormalizeLogger(logger)}
}

// plugin registers itself
func init() {
	plugins.Register(plugins.PluginFactory{
		Protocol: topology.ManagementProtocol_NETCONF,
		New: func(logger observability.Logger) plugins.Plugin {
			return NewVlanNetconfPlugin2021(logger)
		},
	})
}

func (v *VlanNetconfPlugin_2021) Name() string {
	return "vlan-netconf"
}

func (v *VlanNetconfPlugin_2021) FeatureName() string {
	return "Vlan"
}

func (v *VlanNetconfPlugin_2021) SupportedByDevice(model *devicemodelregistry.DeviceModel) bool {
	requiredYangs := []devicemodelregistry.YangModule{{
		Name:     "ieee802-dot1q-bridge.yang",
		Revision: "2018-03-07",
	}}

	for i := range requiredYangs {
		req := &requiredYangs[i]
		found := false
		for _, yf := range model.YangFiles {
			if yf.Name == req.Name && yf.Revision == req.Revision {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (v *VlanNetconfPlugin_2021) SupportedFields(msg proto.Message) []string {
	switch msg.(type) {
	case *topology_config.PortConfig:
		return []string{
			"DefaultVlanId",
			"VlanMemberships",
			"VlanAdvanced",
		}
	case *vlan.BridgeVlanConfig:
		return []string{
			"VlanRegistrationEntries",
			"VidToFidMappings",
			"ProtocolGroupDatabase",
		}
	case *topology_config.BridgeConfig:
		return []string{"VlanConfig"}
	case *topology_config.NodeConfig:
		return []string{"Bridge", "PortConfigs"}
	default:
		return nil
	}
}

func (v *VlanNetconfPlugin_2021) Map(msg proto.Message, node *topology.Node) (any, error) {
	switch typed := msg.(type) {
	case *topology_config.PortConfig:
		if !hasPortVlanData2021(typed) {
			return nil, fmt.Errorf("VlanNetconfPlugin_2021: PortConfig has no VLAN data")
		}
		return mapPortConfigToBridgePort2021(typed), nil
	case *vlan.BridgeVlanConfig:
		if !hasBridgeVlanData2021(typed) {
			return nil, fmt.Errorf("VlanNetconfPlugin_2021: BridgeVlanConfig has no VLAN data")
		}
		return &bridgeVlanPayload{
			BridgeName:    node.GetName(),
			ComponentName: node.GetName(),
			Config:        typed,
		}, nil
	case *topology_config.BridgeConfig:
		if typed.GetVlanConfig() == nil || !hasBridgeVlanData2021(typed.GetVlanConfig()) {
			return nil, fmt.Errorf("VlanNetconfPlugin_2021: BridgeConfig has no VLAN config data")
		}
		return &bridgeVlanPayload{
			BridgeName:    node.GetName(),
			ComponentName: node.GetName(),
			Config:        typed.GetVlanConfig(),
		}, nil
	case *topology_config.NodeConfig:
		if typed.GetBridge() != nil && typed.GetBridge().GetVlanConfig() != nil && hasBridgeVlanData2021(typed.GetBridge().GetVlanConfig()) {
			name := typed.GetNodeId()
			if name == "" {
				name = node.GetName()
			}
			return &bridgeVlanPayload{
				BridgeName:    name,
				ComponentName: name,
				Config:        typed.GetBridge().GetVlanConfig(),
			}, nil
		}

		if len(typed.GetPortConfigs()) == 1 && hasPortVlanData2021(typed.GetPortConfigs()[0]) {
			return mapPortConfigToBridgePort2021(typed.GetPortConfigs()[0]), nil
		}

		return nil, fmt.Errorf("VlanNetconfPlugin_2021: NodeConfig has no mappable VLAN data")
	default:
		return nil, fmt.Errorf("VlanNetconfPlugin_2021: invalid message type %T", msg)
	}
}

func mapPortConfigToBridgePort2021(portCfg *topology_config.PortConfig) *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort {
	bridgePort := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort{}
	if portCfg.DefaultVlanId != nil {
		bridgePort.Pvid = portCfg.DefaultVlanId
	}

	if len(portCfg.GetVlanMemberships()) > 0 {
		bridgePort.VidTranslations = make(map[uint16]*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_VidTranslations)
		for _, membership := range portCfg.GetVlanMemberships() {
			if membership == nil {
				continue
			}
			localVid := uint16(membership.GetVlanId())
			bridgePort.VidTranslations[localVid] = &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_VidTranslations{
				LocalVid: &localVid,
			}
		}
	}

	if adv := portCfg.GetVlanAdvanced(); adv != nil {
		if adv.AcceptableFrame != nil {
			bridgePort.AcceptableFrame = acceptableFrameToYANGModel2021(*adv.AcceptableFrame)
		}
		if adv.IngressFilteringEnabled != nil {
			bridgePort.EnableIngressFiltering = adv.IngressFilteringEnabled
		}
		if adv.RestrictedVlanRegistrationEnabled != nil {
			bridgePort.EnableRestrictedVlanRegistration = adv.RestrictedVlanRegistrationEnabled
		}
		if adv.VidTranslationTableEnabled != nil {
			bridgePort.EnableVidTranslationTable = adv.VidTranslationTableEnabled
		}
		if adv.EgressVidTranslationTableEnabled != nil {
			bridgePort.EnableEgressVidTranslationTable = adv.EgressVidTranslationTableEnabled
		}

		if len(adv.GetVidTranslations()) > 0 && bridgePort.VidTranslations == nil {
			bridgePort.VidTranslations = make(map[uint16]*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_VidTranslations)
		}
		for _, translation := range adv.GetVidTranslations() {
			if translation == nil {
				continue
			}
			localVid := uint16(translation.GetLocalVid())
			entry, exists := bridgePort.VidTranslations[localVid]
			if !exists {
				entry = &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_VidTranslations{}
				bridgePort.VidTranslations[localVid] = entry
			}
			entry.LocalVid = &localVid
			relayVid := uint16(translation.GetRelayVid())
			entry.RelayVid = &relayVid
		}

		if len(adv.GetEgressVidTranslations()) > 0 {
			bridgePort.EgressVidTranslations = make(map[uint16]*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_EgressVidTranslations)
			for _, translation := range adv.GetEgressVidTranslations() {
				if translation == nil {
					continue
				}
				relayVid := uint16(translation.GetRelayVid())
				entry := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_EgressVidTranslations{
					RelayVid: &relayVid,
				}
				localVid := uint16(translation.GetLocalVid())
				entry.LocalVid = &localVid
				bridgePort.EgressVidTranslations[relayVid] = entry
			}
		}

		if len(adv.GetProtocolGroupVidSets()) > 0 {
			bridgePort.ProtocolBasedVlanClassification = ygot.Bool(true)
			bridgePort.ProtocolGroupVidSet = make(map[uint32]*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_ProtocolGroupVidSet)
			for _, group := range adv.GetProtocolGroupVidSets() {
				if group == nil {
					continue
				}
				groupId := group.GetGroupId()
				entry := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_ProtocolGroupVidSet{
					GroupId: &groupId,
				}
				for _, vid := range group.GetVlanIds() {
					entry.Vid = append(entry.Vid, uint16(vid))
				}
				bridgePort.ProtocolGroupVidSet[groupId] = entry
			}
		}
	}

	return bridgePort
}

func (v *VlanNetconfPlugin_2021) Push(mapped any, target managementSessions.DeviceTarget) error {
	featurexml, err := v.BuildFeatureXML(mapped)
	if err != nil {
		return fmt.Errorf("failed to build feature XML: %w", err)
	}

	xml := string(featurexml.XML)
	if featurexml.Container == "bridge-port" {
		if target.Info == nil {
			return fmt.Errorf("device target info is nil")
		}

		xml, err = v.wrapXML(featurexml, target)
		if err != nil {
			return fmt.Errorf("failed to build XML: %w", err)
		}
	}

	if target.Info == nil {
		return fmt.Errorf("device target info is nil")
	}

	session, err := managementSessions.CreateSessionWithPassword(target.Info.IpAddress, target.Credentials.GetUsernamePassword())
	if err != nil {
		return fmt.Errorf("NETCONF session failed: %w", err)
	}
	defer session.Close()

	pushXML := func(xml string, label string) error {
		if v.logger != nil {
			v.logger.Printf("[VLAN] XML generated for %s:\n%s", label, xml)
		}
		if err := managementSessions.EditConfig(session, xml); err != nil {
			return fmt.Errorf("edit-config failed for %s: %w", label, err)
		}
		return nil
	}

	label := target.InterfaceName
	if payload, ok := mapped.(*bridgeVlanPayload); ok {
		label = payload.BridgeName
	}

	return pushXML(xml, label)
}

func (v *VlanNetconfPlugin_2021) BuildFeatureXML(mapped any) (*plugins.FeatureXML, error) {
	switch typed := mapped.(type) {
	case *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort:
		return v.buildBridgePortFeatureXML(typed)
	case *bridgeVlanPayload:
		xml, err := v.buildBridgeVlanXML(typed)
		if err != nil {
			return nil, err
		}
		return &plugins.FeatureXML{Container: "bridges", XML: xml}, nil
	default:
		return nil, fmt.Errorf("VlanNetconfPlugin_2021: invalid mapped type %T", mapped)
	}
}

func (v *VlanNetconfPlugin_2021) buildBridgeVlanXML(payload *bridgeVlanPayload) ([]byte, error) {
	if payload == nil || payload.Config == nil {
		return nil, fmt.Errorf("VlanNetconfPlugin_2021: nil bridge VLAN payload")
	}

	bridgeName := payload.BridgeName
	if bridgeName == "" {
		bridgeName = "br0"
	}
	componentName := payload.ComponentName
	if componentName == "" {
		componentName = bridgeName
	}

	var buf bytes.Buffer
	buf.WriteString(`<bridges xmlns="urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge">`)
	buf.WriteString(`<bridge>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, bridgeName))
	buf.WriteString(`<component>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, componentName))

	vlanIDs := []uint32{}
	if len(payload.Config.GetVlanRegistrationEntries()) > 0 {
		buf.WriteString(`<filtering-database>`)
		for _, reg := range payload.Config.GetVlanRegistrationEntries() {
			if reg == nil {
				continue
			}
			vlanIDs = append(vlanIDs, reg.GetVlanIds()...)

			buf.WriteString(`<vlan-registration-entry>`)
			buf.WriteString(fmt.Sprintf(`<database-id>%d</database-id>`, reg.GetDatabaseId()))
			buf.WriteString(fmt.Sprintf(`<vids>%s</vids>`, joinUint32CSV2021(reg.GetVlanIds())))
			buf.WriteString(fmt.Sprintf(`<entry-type>%s</entry-type>`, vlanRegistrationEntryTypeToXML2021(reg.GetEntryType())))

			for _, pm := range reg.GetPortMaps() {
				if pm == nil {
					continue
				}

				buf.WriteString(`<port-map>`)
				buf.WriteString(fmt.Sprintf(`<port-ref>%d</port-ref>`, portRefFromPortID2021(pm.GetPortId())))
				buf.WriteString(`<static-vlan-registration-entries>`)
				buf.WriteString(fmt.Sprintf(`<registrar-admin-control>%s</registrar-admin-control>`, registrarAdminControlToXML2021(pm.GetRegistrarAdminControl())))
				buf.WriteString(fmt.Sprintf(`<vlan-transmitted>%s</vlan-transmitted>`, vlanTransmittedToXML2021(pm.GetVlanTransmitted())))
				buf.WriteString(`</static-vlan-registration-entries>`)
				buf.WriteString(`</port-map>`)
			}

			buf.WriteString(`</vlan-registration-entry>`)
		}
		buf.WriteString(`</filtering-database>`)
	}

	//Configure port membership in the Filtering Database
	if len(vlanIDs) > 0 || len(payload.Config.GetVidToFidMappings()) > 0 {
		buf.WriteString(`<bridge-vlan>`)

		// Write the <vlan> names to instantiate them
		for vid := range vlanIDs {
			buf.WriteString(`<vlan>`)
			buf.WriteString(fmt.Sprintf(`<vid>%d</vid>`, vid))
			buf.WriteString(fmt.Sprintf(`<name>VLAN_%d</name>`, vid))
			buf.WriteString(`</vlan>`)
		}

		// Write the fid mappings
		for _, m := range payload.Config.GetVidToFidMappings() {
			if m != nil {
				buf.WriteString(`<vid-to-fid>`)
				buf.WriteString(fmt.Sprintf(`<vid>%d</vid>`, m.GetVid()))
				buf.WriteString(fmt.Sprintf(`<fid>%d</fid>`, m.GetFid()))
				buf.WriteString(`</vid-to-fid>`)
			}
		}

		buf.WriteString(`</bridge-vlan>`)
	}

	buf.WriteString(`</component>`)
	buf.WriteString(`</bridge>`)
	buf.WriteString(`</bridges>`)

	return buf.Bytes(), nil
}

func (v *VlanNetconfPlugin_2021) buildBridgePortFeatureXML(root *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort) (*plugins.FeatureXML, error) {
	var buf bytes.Buffer

	buf.WriteString(`<bridge-port xmlns="urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge">`)

	if root.Pvid != nil {
		buf.WriteString(fmt.Sprintf(`<pvid>%d</pvid>`, *root.Pvid))
	}

	if root.AcceptableFrame != opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_UNSET {
		buf.WriteString(fmt.Sprintf(`<acceptable-frame>%s</acceptable-frame>`, acceptableFrameToXML2021(root.AcceptableFrame)))
	}

	if root.EnableIngressFiltering != nil {
		buf.WriteString(fmt.Sprintf(`<enable-ingress-filtering>%t</enable-ingress-filtering>`, *root.EnableIngressFiltering))
	}
	if root.EnableRestrictedVlanRegistration != nil {
		buf.WriteString(fmt.Sprintf(`<enable-restricted-vlan-registration>%t</enable-restricted-vlan-registration>`, *root.EnableRestrictedVlanRegistration))
	}
	if root.EnableVidTranslationTable != nil {
		buf.WriteString(fmt.Sprintf(`<enable-vid-translation-table>%t</enable-vid-translation-table>`, *root.EnableVidTranslationTable))
	}
	if root.EnableEgressVidTranslationTable != nil {
		buf.WriteString(fmt.Sprintf(`<enable-egress-vid-translation-table>%t</enable-egress-vid-translation-table>`, *root.EnableEgressVidTranslationTable))
	}

	if len(root.VidTranslations) > 0 {
		buf.WriteString(`<vid-translations>`)
		for _, vidTrans := range root.VidTranslations {
			if vidTrans == nil {
				continue
			}
			if vidTrans.LocalVid != nil {
				buf.WriteString(fmt.Sprintf(`<local-vid>%d</local-vid>`, *vidTrans.LocalVid))
			}
			if vidTrans.RelayVid != nil {
				buf.WriteString(fmt.Sprintf(`<relay-vid>%d</relay-vid>`, *vidTrans.RelayVid))
			}
		}
		buf.WriteString(`</vid-translations>`)
	}

	if len(root.EgressVidTranslations) > 0 {
		for _, vidTrans := range root.EgressVidTranslations {
			if vidTrans == nil {
				continue
			}
			buf.WriteString(`<egress-vid-translations>`)
			if vidTrans.RelayVid != nil {
				buf.WriteString(fmt.Sprintf(`<relay-vid>%d</relay-vid>`, *vidTrans.RelayVid))
			}
			if vidTrans.LocalVid != nil {
				buf.WriteString(fmt.Sprintf(`<local-vid>%d</local-vid>`, *vidTrans.LocalVid))
			}
			buf.WriteString(`</egress-vid-translations>`)
		}
	}

	if root.ProtocolBasedVlanClassification != nil {
		buf.WriteString(fmt.Sprintf(`<protocol-based-vlan-classification>%t</protocol-based-vlan-classification>`, *root.ProtocolBasedVlanClassification))
	}

	if len(root.ProtocolGroupVidSet) > 0 {
		for _, group := range root.ProtocolGroupVidSet {
			if group == nil || group.GroupId == nil {
				continue
			}
			buf.WriteString(`<protocol-group-vid-set>`)
			buf.WriteString(fmt.Sprintf(`<group-id>%d</group-id>`, *group.GroupId))
			for _, vid := range group.Vid {
				buf.WriteString(fmt.Sprintf(`<vid>%d</vid>`, vid))
			}
			buf.WriteString(`</protocol-group-vid-set>`)
		}
	}

	buf.WriteString(`</bridge-port>`)

	return &plugins.FeatureXML{Container: "bridge-port", XML: buf.Bytes()}, nil
}

func (v *VlanNetconfPlugin_2021) wrapXML(featurexml *plugins.FeatureXML, target managementSessions.DeviceTarget) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">`)
	buf.WriteString(`<interface>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))
	buf.WriteString(string(featurexml.XML))
	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)

	return buf.String(), nil
}

func acceptableFrameToYANGModel2021(vf vlan.AcceptableFrameType) opencncModel.E_IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame {
	switch vf {
	case vlan.AcceptableFrameType_ACCEPTABLE_FRAME_TYPE_ADMIT_ONLY_VLAN_TAGGED:
		return opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_admit_only_VLAN_tagged_frames
	case vlan.AcceptableFrameType_ACCEPTABLE_FRAME_TYPE_ADMIT_ONLY_UNTAGGED_AND_PRIORITY_TAGGED:
		return opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_admit_only_untagged_and_priority_tagged
	case vlan.AcceptableFrameType_ACCEPTABLE_FRAME_TYPE_ADMIT_ALL_FRAMES:
		return opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_admit_all_frames
	default:
		return opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_UNSET
	}
}

func acceptableFrameToXML2021(vf opencncModel.E_IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame) string {
	switch vf {
	case opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_admit_only_VLAN_tagged_frames:
		return "admit_only_VLAN_tagged_frames"
	case opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_admit_only_untagged_and_priority_tagged:
		return "admit_only_untagged_and_priority_tagged"
	case opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_admit_all_frames:
		return "admit_all_frames"
	default:
		return ""
	}
}

func hasPortVlanData2021(portCfg *topology_config.PortConfig) bool {
	if portCfg == nil {
		return false
	}
	if portCfg.DefaultVlanId != nil {
		return true
	}
	if len(portCfg.GetVlanMemberships()) > 0 {
		return true
	}
	if portCfg.GetVlanAdvanced() != nil {
		return true
	}
	return false
}

func hasBridgeVlanData2021(cfg *vlan.BridgeVlanConfig) bool {
	if cfg == nil {
		return false
	}
	if len(cfg.GetVlanRegistrationEntries()) > 0 {
		return true
	}
	if len(cfg.GetVidToFidMappings()) > 0 {
		return true
	}
	if len(cfg.GetProtocolGroupDatabase()) > 0 {
		return true
	}
	return false
}

func joinUint32CSV2021(values []uint32) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.FormatUint(uint64(v), 10))
	}
	return strings.Join(parts, ",")
}

func vlanRegistrationEntryTypeToXML2021(t vlan.VlanRegistrationEntryType) string {
	switch t {
	case vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_STATIC:
		return "static"
	case vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_DYNAMIC:
		return "dynamic"
	default:
		return "static"
	}
}

func registrarAdminControlToXML2021(v vlan.RegistrarAdminControl) string {
	switch v {
	case vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_FIXED_NEW_IGNORED:
		return "fixed_new_ignored"
	case vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_FIXED_NEW_PROPAGATED:
		return "fixed_new_propagated"
	case vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_FORBIDDEN:
		return "forbidden"
	case vlan.RegistrarAdminControl_REGISTRAR_ADMIN_CONTROL_NORMAL:
		return "normal"
	default:
		return "normal"
	}
}

func vlanTransmittedToXML2021(v vlan.VlanTransmitted) string {
	switch v {
	case vlan.VlanTransmitted_VLAN_TRANSMITTED_TAGGED:
		return "tagged"
	case vlan.VlanTransmitted_VLAN_TRANSMITTED_UNTAGGED:
		return "untagged"
	default:
		return "tagged"
	}
}

func portRefFromPortID2021(portID string) uint32 {
	if portID == "" {
		return 0
	}

	last := -1
	for i := len(portID) - 1; i >= 0; i-- {
		if portID[i] < '0' || portID[i] > '9' {
			break
		}
		last = i
	}
	if last == -1 {
		return 0
	}

	value, err := strconv.ParseUint(portID[last:], 10, 32)
	if err != nil {
		return 0
	}

	return uint32(value)
}
