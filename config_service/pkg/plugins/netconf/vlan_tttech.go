package netconf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"OpenCNC_config_service/common/observability"
	devicemodelregistry "OpenCNC_config_service/common/structures/devicemodelregistry"
	"OpenCNC_config_service/common/structures/topology"
	topology_config "OpenCNC_config_service/common/structures/topology_config"
	vlan "OpenCNC_config_service/common/structures/vlan"
	opencncModel "OpenCNC_config_service/config_service/opencnc_model"
	managementSessions "OpenCNC_config_service/config_service/pkg/managementSessions"
	"OpenCNC_config_service/config_service/pkg/plugins"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"
)

var _ plugins.Plugin = (*VlanNetconfPlugin)(nil)

type VlanNetconfPlugin struct {
	logger observability.Logger
}

type bridgeVlanPayload struct {
	BridgeName    string
	ComponentName string
	Config        *vlan.BridgeVlanConfig
}

func NewVlanNetconfPlugin(logger observability.Logger) *VlanNetconfPlugin {
	return &VlanNetconfPlugin{logger: observability.NormalizeLogger(logger)}
}

// plugin registers itself
func init() {
	plugins.Register(plugins.PluginFactory{
		Protocol: topology.ManagementProtocol_NETCONF,
		New: func(logger observability.Logger) plugins.Plugin {
			return NewVlanNetconfPlugin(logger)
		},
	})
}

func (v *VlanNetconfPlugin) Name() string {
	return "vlan-netconf"
}

func (v *VlanNetconfPlugin) FeatureName() string {
	return "Vlan"
}

func (v *VlanNetconfPlugin) SupportedByDevice(model *devicemodelregistry.DeviceModel) bool {
	requiredYangs := []devicemodelregistry.YangFile{{
		Name:     "ieee802-dot1q-bridge.yang",
		Revision: "2021-04-09",
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

func (v *VlanNetconfPlugin) SupportedFields(msg proto.Message) []string {
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

func (v *VlanNetconfPlugin) Map(msg proto.Message) (any, error) {
	switch typed := msg.(type) {
	case *topology_config.PortConfig:
		if !hasPortVlanData(typed) {
			return nil, fmt.Errorf("VlanNetconfPlugin: PortConfig has no VLAN data")
		}
		return mapPortConfigToBridgePort(typed), nil
	case *vlan.BridgeVlanConfig:
		if !hasBridgeVlanData(typed) {
			return nil, fmt.Errorf("VlanNetconfPlugin: BridgeVlanConfig has no VLAN data")
		}
		return &bridgeVlanPayload{
			BridgeName:    "br0",
			ComponentName: "br0",
			Config:        typed,
		}, nil
	case *topology_config.BridgeConfig:
		if typed.GetVlanConfig() == nil || !hasBridgeVlanData(typed.GetVlanConfig()) {
			return nil, fmt.Errorf("VlanNetconfPlugin: BridgeConfig has no VLAN config data")
		}
		return &bridgeVlanPayload{
			BridgeName:    "br0",
			ComponentName: "br0",
			Config:        typed.GetVlanConfig(),
		}, nil
	case *topology_config.NodeConfig:
		if typed.GetBridge() != nil && typed.GetBridge().GetVlanConfig() != nil && hasBridgeVlanData(typed.GetBridge().GetVlanConfig()) {
			name := typed.GetNodeId()
			if name == "" {
				name = "br0"
			}
			return &bridgeVlanPayload{
				BridgeName:    name,
				ComponentName: name,
				Config:        typed.GetBridge().GetVlanConfig(),
			}, nil
		}

		if len(typed.GetPortConfigs()) == 1 && hasPortVlanData(typed.GetPortConfigs()[0]) {
			return mapPortConfigToBridgePort(typed.GetPortConfigs()[0]), nil
		}

		return nil, fmt.Errorf("VlanNetconfPlugin: NodeConfig has no mappable VLAN data")
	default:
		return nil, fmt.Errorf("VlanNetconfPlugin: invalid message type %T", msg)
	}
}

func mapPortConfigToBridgePort(portCfg *topology_config.PortConfig) *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort {
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
			bridgePort.AcceptableFrame = acceptableFrameToYANGModel(*adv.AcceptableFrame)
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

func (v *VlanNetconfPlugin) Push(mapped any, target managementSessions.DeviceTarget) error {
	if target.Info == nil {
		return fmt.Errorf("device target info is nil")
	}

	session, err := managementSessions.CreateSession(target.Info.IpAddress, target.Info.UserName, target.Secret)
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

	switch typed := mapped.(type) {
	case *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort:
		xml, err := v.BuildXML(typed, target)
		if err != nil {
			return err
		}
		return pushXML(xml, target.InterfaceName)
	case *bridgeVlanPayload:
		xml, err := v.BuildBridgeVlanXML(typed)
		if err != nil {
			return err
		}
		return pushXML(xml, typed.BridgeName)
	default:
		return fmt.Errorf("VlanNetconfPlugin: invalid mapped type %T", mapped)
	}
}

func (v *VlanNetconfPlugin) BuildBridgeVlanXML(payload *bridgeVlanPayload) (string, error) {
	if payload == nil || payload.Config == nil {
		return "", fmt.Errorf("VlanNetconfPlugin: nil bridge VLAN payload")
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

	if len(payload.Config.GetVlanRegistrationEntries()) > 0 {
		buf.WriteString(`<filtering-database>`)
		for _, reg := range payload.Config.GetVlanRegistrationEntries() {
			if reg == nil {
				continue
			}

			buf.WriteString(`<vlan-registration-entry>`)
			buf.WriteString(fmt.Sprintf(`<database-id>%d</database-id>`, reg.GetDatabaseId()))
			buf.WriteString(fmt.Sprintf(`<vids>%s</vids>`, joinUint32CSV(reg.GetVlanIds())))
			buf.WriteString(fmt.Sprintf(`<entry-type>%s</entry-type>`, vlanRegistrationEntryTypeToXML(reg.GetEntryType())))

			for _, pm := range reg.GetPortMaps() {
				if pm == nil {
					continue
				}

				buf.WriteString(`<port-map>`)
				buf.WriteString(fmt.Sprintf(`<port-ref>%d</port-ref>`, portRefFromPortID(pm.GetPortId())))
				buf.WriteString(`<static-vlan-registration-entries>`)
				buf.WriteString(fmt.Sprintf(`<registrar-admin-control>%s</registrar-admin-control>`, registrarAdminControlToXML(pm.GetRegistrarAdminControl())))
				buf.WriteString(fmt.Sprintf(`<vlan-transmitted>%s</vlan-transmitted>`, vlanTransmittedToXML(pm.GetVlanTransmitted())))
				buf.WriteString(`</static-vlan-registration-entries>`)
				buf.WriteString(`</port-map>`)
			}

			buf.WriteString(`</vlan-registration-entry>`)
		}
		buf.WriteString(`</filtering-database>`)
	}

	if len(payload.Config.GetVidToFidMappings()) > 0 {
		buf.WriteString(`<bridge-vlan>`)
		for _, m := range payload.Config.GetVidToFidMappings() {
			if m == nil {
				continue
			}
			buf.WriteString(`<vid-to-fid>`)
			buf.WriteString(fmt.Sprintf(`<vid>%d</vid>`, m.GetVid()))
			buf.WriteString(fmt.Sprintf(`<fid>%d</fid>`, m.GetFid()))
			buf.WriteString(`</vid-to-fid>`)
		}
		buf.WriteString(`</bridge-vlan>`)
	}

	buf.WriteString(`</component>`)
	buf.WriteString(`</bridge>`)
	buf.WriteString(`</bridges>`)

	return buf.String(), nil
}

func (v *VlanNetconfPlugin) BuildXML(root *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort, target managementSessions.DeviceTarget) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">`)
	buf.WriteString(`<interface>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))
	buf.WriteString(`<bridge-port xmlns="urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge">`)

	if root.Pvid != nil {
		buf.WriteString(fmt.Sprintf(`<pvid>%d</pvid>`, *root.Pvid))
	}

	if root.AcceptableFrame != opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame_UNSET {
		buf.WriteString(fmt.Sprintf(`<acceptable-frame>%s</acceptable-frame>`, acceptableFrameToXML(root.AcceptableFrame)))
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
	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)

	return buf.String(), nil
}

func acceptableFrameToYANGModel(vf vlan.AcceptableFrameType) opencncModel.E_IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame {
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

func acceptableFrameToXML(vf opencncModel.E_IETFInterfaces_Interfaces_Interface_BridgePort_AcceptableFrame) string {
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

func hasPortVlanData(portCfg *topology_config.PortConfig) bool {
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

func hasBridgeVlanData(cfg *vlan.BridgeVlanConfig) bool {
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

func joinUint32CSV(values []uint32) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.FormatUint(uint64(v), 10))
	}
	return strings.Join(parts, ",")
}

func vlanRegistrationEntryTypeToXML(t vlan.VlanRegistrationEntryType) string {
	switch t {
	case vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_STATIC:
		return "static"
	case vlan.VlanRegistrationEntryType_VLAN_REG_ENTRY_TYPE_DYNAMIC:
		return "dynamic"
	default:
		return "static"
	}
}

func registrarAdminControlToXML(v vlan.RegistrarAdminControl) string {
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

func vlanTransmittedToXML(v vlan.VlanTransmitted) string {
	switch v {
	case vlan.VlanTransmitted_VLAN_TRANSMITTED_TAGGED:
		return "tagged"
	case vlan.VlanTransmitted_VLAN_TRANSMITTED_UNTAGGED:
		return "untagged"
	default:
		return "tagged"
	}
}

func portRefFromPortID(portID string) uint32 {
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
