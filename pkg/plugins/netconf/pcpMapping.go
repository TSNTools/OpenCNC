package netconf

import (
	"bytes"
	"fmt"
	"log"

	opencncModel "OpenCNC_config_service/opencnc_model"
	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	"OpenCNC_config_service/pkg/plugins"
	devicemodelregistry "OpenCNC_config_service/pkg/structures/devicemodelregistry"
	topology_config "OpenCNC_config_service/pkg/structures/topology_config"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"
)

var _ plugins.Plugin = (*PcpMappingNetconfPlugin)(nil)

type PcpMappingNetconfPlugin struct {
	logger *log.Logger
}

func NewPcpMappingNetconfPlugin(logger *log.Logger) *PcpMappingNetconfPlugin {
	return &PcpMappingNetconfPlugin{logger: logger}
}

func (p *PcpMappingNetconfPlugin) Name() string {
	return "pcp-mapping-netconf"
}

func (p *PcpMappingNetconfPlugin) FeatureName() string {
	return "PcpMapping"
}

func (p *PcpMappingNetconfPlugin) Supports(msg proto.Message) bool {
	_, ok := msg.(*topology_config.PortConfig)
	return ok
}

func (p *PcpMappingNetconfPlugin) SupportedFields() []string {
	return []string{
		"DefaultPriority",
		"PcpMappingEnabled",
		"TrafficClassTable",
	}
}

func (p *PcpMappingNetconfPlugin) SupportedByDevice(model *devicemodelregistry.DeviceModel) bool {
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

func (p *PcpMappingNetconfPlugin) Map(msg proto.Message) (any, error) {
	portCfg, ok := msg.(*topology_config.PortConfig)
	if !ok {
		return nil, fmt.Errorf("PcpMappingNetconfPlugin: invalid message type %T", msg)
	}

	pcpDecodingTable := opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_PcpDecodingTable{}
	pcpEncodingTable := opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_PcpEncodingTable{}
	priorityRegeneration := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_PriorityRegeneration{}

	for _, entry := range portCfg.GetTrafficClassTable() {
		if entry == nil {
			continue
		}

		queue := uint8(entry.GetEgressQueueId())
		switch uint8(entry.GetPcp()) {
		case 0:
			priorityRegeneration.Priority0 = ygot.Uint8(queue)
		case 1:
			priorityRegeneration.Priority1 = ygot.Uint8(queue)
		case 2:
			priorityRegeneration.Priority2 = ygot.Uint8(queue)
		case 3:
			priorityRegeneration.Priority3 = ygot.Uint8(queue)
		case 4:
			priorityRegeneration.Priority4 = ygot.Uint8(queue)
		case 5:
			priorityRegeneration.Priority5 = ygot.Uint8(queue)
		case 6:
			priorityRegeneration.Priority6 = ygot.Uint8(queue)
		case 7:
			priorityRegeneration.Priority7 = ygot.Uint8(queue)
		}
	}

	bridgePort := opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort{
		DefaultPriority:      nil,
		PcpDecodingTable:     &pcpDecodingTable,
		PcpEncodingTable:     &pcpEncodingTable,
		PriorityRegeneration: priorityRegeneration,
	}

	if portCfg.GetDefaultPriority() <= 7 {
		bridgePort.DefaultPriority = ygot.Uint8(uint8(portCfg.GetDefaultPriority()))
	}

	return &bridgePort, nil
}

func (p *PcpMappingNetconfPlugin) Push(mapped any, target managementSessions.DeviceTarget) error {
	bridgePort, ok := mapped.(*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort)
	if !ok {
		return fmt.Errorf("PcpMappingNetconfPlugin: invalid mapped type %T", mapped)
	}

	xml, err := p.BuildXML(bridgePort, target)
	if err != nil {
		return err
	}

	if p.logger != nil {
		p.logger.Printf("[PcpMapping] XML generated for interface %s:\n%s", target.InterfaceName, xml)
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

func (p *PcpMappingNetconfPlugin) BuildXML(root *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort, target managementSessions.DeviceTarget) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">`)
	buf.WriteString(`<interface>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))
	buf.WriteString(`<bridge-port xmlns="urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge">`)

	if root.DefaultPriority != nil {
		buf.WriteString(fmt.Sprintf(`<default-priority>%d</default-priority>`, *root.DefaultPriority))
	}

	// PCP Decoding Table - maps incoming PCP to internal priority
	// Note: PortConfig does not provide PCP decoding mappings; this would need separate configuration
	if root.PcpDecodingTable != nil && root.PcpDecodingTable.PcpDecodingMap != nil && len(root.PcpDecodingTable.PcpDecodingMap) > 0 {
		buf.WriteString(`<pcp-decoding-table>`)
		for _, decodingMap := range root.PcpDecodingTable.PcpDecodingMap {
			if decodingMap != nil && decodingMap.PriorityMap != nil {
				buf.WriteString(`<pcp-decoding-map>`)
				buf.WriteString(fmt.Sprintf(`<pcp>%v</pcp>`, decodingMap.Pcp))
				for _, priorityMap := range decodingMap.PriorityMap {
					if priorityMap != nil {
						buf.WriteString(`<priority-map>`)
						if priorityMap.PriorityCodePoint != nil {
							buf.WriteString(fmt.Sprintf(`<priority-code-point>%d</priority-code-point>`, *priorityMap.PriorityCodePoint))
						}
						if priorityMap.Priority != nil {
							buf.WriteString(fmt.Sprintf(`<priority>%d</priority>`, *priorityMap.Priority))
						}
						if priorityMap.DropEligible != nil {
							buf.WriteString(fmt.Sprintf(`<drop-eligible>%t</drop-eligible>`, *priorityMap.DropEligible))
						}
						buf.WriteString(`</priority-map>`)
					}
				}
				buf.WriteString(`</pcp-decoding-map>`)
			}
		}
		buf.WriteString(`</pcp-decoding-table>`)
	}

	// PCP Encoding Table - maps internal priority + DEI to outgoing PCP
	// Note: PortConfig does not provide PCP encoding mappings; this would need separate configuration
	if root.PcpEncodingTable != nil && root.PcpEncodingTable.PcpEncodingMap != nil && len(root.PcpEncodingTable.PcpEncodingMap) > 0 {
		buf.WriteString(`<pcp-encoding-table>`)
		for _, encodingMap := range root.PcpEncodingTable.PcpEncodingMap {
			if encodingMap != nil && encodingMap.PriorityMap != nil {
				buf.WriteString(`<pcp-encoding-map>`)
				buf.WriteString(fmt.Sprintf(`<pcp>%v</pcp>`, encodingMap.Pcp))
				for _, priorityMap := range encodingMap.PriorityMap {
					if priorityMap != nil {
						buf.WriteString(`<priority-map>`)
						if priorityMap.Priority != nil {
							buf.WriteString(fmt.Sprintf(`<priority>%d</priority>`, *priorityMap.Priority))
						}
						if priorityMap.Dei != nil {
							buf.WriteString(fmt.Sprintf(`<dei>%t</dei>`, *priorityMap.Dei))
						}
						if priorityMap.PriorityCodePoint != nil {
							buf.WriteString(fmt.Sprintf(`<priority-code-point>%d</priority-code-point>`, *priorityMap.PriorityCodePoint))
						}
						buf.WriteString(`</priority-map>`)
					}
				}
				buf.WriteString(`</pcp-encoding-map>`)
			}
		}
		buf.WriteString(`</pcp-encoding-table>`)
	}

	if root.PriorityRegeneration != nil {
		buf.WriteString(`<priority-regeneration>`)
		for _, field := range []struct {
			name  string
			value *uint8
		}{
			{name: "priority0", value: root.PriorityRegeneration.Priority0},
			{name: "priority1", value: root.PriorityRegeneration.Priority1},
			{name: "priority2", value: root.PriorityRegeneration.Priority2},
			{name: "priority3", value: root.PriorityRegeneration.Priority3},
			{name: "priority4", value: root.PriorityRegeneration.Priority4},
			{name: "priority5", value: root.PriorityRegeneration.Priority5},
			{name: "priority6", value: root.PriorityRegeneration.Priority6},
			{name: "priority7", value: root.PriorityRegeneration.Priority7},
		} {
			if field.value != nil {
				buf.WriteString(fmt.Sprintf(`<%s>%d</%s>`, field.name, *field.value, field.name))
			}
		}
		buf.WriteString(`</priority-regeneration>`)
	}

	buf.WriteString(`</bridge-port>`)
	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)

	return buf.String(), nil
}
