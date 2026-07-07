package netconf

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	opencncModel "OpenCNC_config_service/opencnc_model"
	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	"OpenCNC_config_service/pkg/plugins"
	devicemodelregistry "OpenCNC_config_service/pkg/structures/devicemodelregistry"
	topology_config "OpenCNC_config_service/pkg/structures/topology_config"
	traffic_type "OpenCNC_config_service/pkg/structures/trafficType"

	"github.com/golang/protobuf/proto"
)

var _ plugins.Plugin = (*VlanNetconfPlugin)(nil)

type VlanNetconfPlugin struct {
	logger *log.Logger
}

func NewVlanNetconfPlugin(logger *log.Logger) *VlanNetconfPlugin {
	return &VlanNetconfPlugin{logger: logger}
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
		req := requiredYangs[i]
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

func (v *VlanNetconfPlugin) Supports(msg proto.Message) bool {
	switch msg.(type) {
	case *traffic_type.VlanTrafficMappings, *topology_config.BridgeVlanConfig, *topology_config.BridgeConfig:
		return true
	default:
		return false
	}
}

func (p *VlanNetconfPlugin) SupportedFields() []string {
	return []string{
		"VlanMemberships",
		"DefaultVlanId",
		// TODO: it actually takes a traffic_type.VlanTrafficMappings
		// it is not part of portConfig. fix it!
	}
}

func (v *VlanNetconfPlugin) Map(msg proto.Message) (any, error) {
	switch typed := msg.(type) {
	case *traffic_type.VlanTrafficMappings:
		bridgePort := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort{
			VidTranslations: make(map[uint16]*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_VidTranslations),
		}

		for _, mapping := range typed.GetMappings() {
			if mapping == nil {
				continue
			}

			localVid := uint16(mapping.GetVlanId())
			vidTrans, err := bridgePort.NewVidTranslations(localVid)
			if err != nil {
				return nil, fmt.Errorf("failed to create VID translation for VLAN %d: %w", localVid, err)
			}

			if vidTrans != nil {
				vidTrans.LocalVid = &localVid
			}
		}

		return bridgePort, nil
	case *topology_config.BridgeVlanConfig:
		return typed, nil
	case *topology_config.BridgeConfig:
		if typed.GetVlanConfig() == nil {
			return nil, fmt.Errorf("VlanNetconfPlugin: BridgeConfig has no vlan_config")
		}
		return typed.GetVlanConfig(), nil
	default:
		return nil, fmt.Errorf("VlanNetconfPlugin: invalid message type %T", msg)
	}
}

func (v *VlanNetconfPlugin) Push(mapped any, target managementSessions.DeviceTarget) error {
	var xml string
	var err error

	if bridgePort, ok := mapped.(*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort); ok {
		xml, err = v.BuildXML(bridgePort, target)
	} else if vlanConfig, ok := mapped.(*topology_config.BridgeVlanConfig); ok {
		xml, err = v.BuildBridgeVlanXML(vlanConfig, target)
	} else {
		return fmt.Errorf("VlanNetconfPlugin: invalid mapped type %T", mapped)
	}

	if err != nil {
		return err
	}

	if v.logger != nil {
		v.logger.Printf("[VLAN] XML generated for target %s:\n%s", target.InterfaceName, xml)
	}

	if target.Info == nil {
		return fmt.Errorf("device target info is nil")
	}

	session, err := managementSessions.CreateSession(target.Info.IpAddress, target.Info.UserName, target.Secret)
	if err != nil {
		return fmt.Errorf("NETCONF session failed: %w", err)
	}
	defer session.Close()

	if err := managementSessions.EditConfig(session, xml); err != nil {
		return fmt.Errorf("edit-config failed: %w", err)
	}

	return nil
}

func (v *VlanNetconfPlugin) BuildXML(root *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort, target managementSessions.DeviceTarget) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">`)
	buf.WriteString(`<interface>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))
	buf.WriteString(`<bridge-port xmlns="urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge">`)

	if root.VidTranslations != nil && len(root.VidTranslations) > 0 {
		buf.WriteString(`<vid-translations>`)
		for _, vidTrans := range root.VidTranslations {
			if vidTrans != nil && vidTrans.LocalVid != nil {
				buf.WriteString(fmt.Sprintf(`<local-vid>%d</local-vid>`, *vidTrans.LocalVid))
			}
		}
		buf.WriteString(`</vid-translations>`)
	}

	buf.WriteString(`</bridge-port>`)
	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)

	return buf.String(), nil
}

func (v *VlanNetconfPlugin) BuildBridgeVlanXML(cfg *topology_config.BridgeVlanConfig, target managementSessions.DeviceTarget) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("BuildBridgeVlanXML: vlan config is nil")
	}

	bridgeName := target.InterfaceName
	if bridgeName == "" {
		bridgeName = "br0"
	}

	var buf bytes.Buffer
	buf.WriteString(`<bridges xmlns="urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge">`)
	buf.WriteString(`<bridge>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, bridgeName))
	buf.WriteString(`<component>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, bridgeName))

	if len(cfg.GetVlanRegistrationEntries()) > 0 {
		buf.WriteString(`<filtering-database>`)
		for _, entry := range cfg.GetVlanRegistrationEntries() {
			if entry == nil {
				continue
			}

			buf.WriteString(`<vlan-registration-entry>`)
			buf.WriteString(fmt.Sprintf(`<database-id>%d</database-id>`, entry.GetDatabaseId()))

			vids := entry.GetVlanIds()
			if len(vids) > 0 {
				stringIds := make([]string, 0, len(vids))
				for _, vid := range vids {
					stringIds = append(stringIds, fmt.Sprintf("%d", vid))
				}
				buf.WriteString(fmt.Sprintf(`<vids>%s</vids>`, strings.Join(stringIds, ",")))
			}

			if entry.GetEntryType() != "" {
				buf.WriteString(fmt.Sprintf(`<entry-type>%s</entry-type>`, entry.GetEntryType()))
			}

			for _, portMap := range entry.GetPortMaps() {
				if portMap == nil {
					continue
				}
				buf.WriteString(`<port-map>`)
				buf.WriteString(fmt.Sprintf(`<port-ref>%d</port-ref>`, portMap.GetPortRef()))
				if portMap.GetRegistrarAdminControl() != "" {
					buf.WriteString(fmt.Sprintf(`<registrar-admin-control>%s</registrar-admin-control>`, portMap.GetRegistrarAdminControl()))
				}
				if portMap.GetVlanTransmitted() != "" {
					buf.WriteString(fmt.Sprintf(`<vlan-transmitted>%s</vlan-transmitted>`, portMap.GetVlanTransmitted()))
				}
				buf.WriteString(`</port-map>`)
			}

			buf.WriteString(`</vlan-registration-entry>`)
		}
		buf.WriteString(`</filtering-database>`)
	}

	if len(cfg.GetVidToFidMappings()) > 0 {
		buf.WriteString(`<bridge-vlan>`)
		for _, mapping := range cfg.GetVidToFidMappings() {
			if mapping == nil {
				continue
			}
			buf.WriteString(`<vid-to-fid>`)
			buf.WriteString(fmt.Sprintf(`<vid>%d</vid>`, mapping.GetVid()))
			buf.WriteString(fmt.Sprintf(`<fid>%d</fid>`, mapping.GetFid()))
			buf.WriteString(`</vid-to-fid>`)
		}
		buf.WriteString(`</bridge-vlan>`)
	}

	buf.WriteString(`</component>`)
	buf.WriteString(`</bridge>`)
	buf.WriteString(`</bridges>`)

	return buf.String(), nil
}
