package netconf

import (
	"bytes"
	"fmt"
	"log"

	opencncModel "OpenCNC_config_service/opencnc_model"
	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	"OpenCNC_config_service/pkg/plugins"
	devicemodelregistry "OpenCNC_config_service/pkg/structures/devicemodelregistry"
	traffic_type "OpenCNC_config_service/pkg/structures/trafficType"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"
)

var _ plugins.Plugin = (*TrafficClassNetconfPlugin)(nil)

type TrafficClassNetconfPlugin struct {
	logger *log.Logger
}

func NewTrafficClassNetconfPlugin(logger *log.Logger) *TrafficClassNetconfPlugin {
	return &TrafficClassNetconfPlugin{logger: logger}
}

func (t *TrafficClassNetconfPlugin) Name() string {
	return "traffic-class-netconf"
}

func (t *TrafficClassNetconfPlugin) FeatureName() string {
	return "TrafficClass"
}

func (t *TrafficClassNetconfPlugin) SupportedByDevice(model *devicemodelregistry.DeviceModel) bool {
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

func (t *TrafficClassNetconfPlugin) Supports(msg proto.Message) bool {
	_, ok := msg.(*traffic_type.PortTrafficMappings)
	return ok
}

func (p *TrafficClassNetconfPlugin) SupportedFields() []string {
	return []string{
		"QueueConfigs", // TODO: it actually takes a traffic_type.PortTrafficMapping
		// it is not part of portConfig. fix it!
	}
}

func (t *TrafficClassNetconfPlugin) Map(msg proto.Message) (any, error) {
	portTrafficMappings, ok := msg.(*traffic_type.PortTrafficMappings)
	if !ok {
		return nil, fmt.Errorf("TrafficClassNetconfPlugin: invalid message type %T", msg)
	}

	trafficClass := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_TrafficClass{}
	table := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_TrafficClass_TrafficClassTable{}

	mappingCount := len(portTrafficMappings.GetMappings())
	if mappingCount > 8 {
		mappingCount = 8
	}
	if mappingCount > 0 {
		// this has to be conssitent with the profile definition of the traffic classes
		// as of now, the profile definition of the classes is used to populate the configuration,
		// but the NumberOfTrafficClasses is not provided to the plugin. rather it is infered again here:
		table.NumberOfTrafficClasses = ygot.Uint8(uint8(mappingCount))
		// should make sure it is consistent with the profile definition
	}

	for i := 0; i < mappingCount; i++ {
		m := portTrafficMappings.Mappings[i]
		queue := uint8(m.GetQueueId())

		switch i {
		case 0:
			table.Priority0 = ygot.Uint8(queue)
		case 1:
			table.Priority1 = ygot.Uint8(queue)
		case 2:
			table.Priority2 = ygot.Uint8(queue)
		case 3:
			table.Priority3 = ygot.Uint8(queue)
		case 4:
			table.Priority4 = ygot.Uint8(queue)
		case 5:
			table.Priority5 = ygot.Uint8(queue)
		case 6:
			table.Priority6 = ygot.Uint8(queue)
		case 7:
			table.Priority7 = ygot.Uint8(queue)
		}
	}

	trafficClass.TrafficClassTable = table
	return trafficClass, nil
}

func (t *TrafficClassNetconfPlugin) Push(mapped any, target managementSessions.DeviceTarget) error {
	tc, ok := mapped.(*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_TrafficClass)
	if !ok {
		return fmt.Errorf("TrafficClassNetconfPlugin: invalid mapped type %T", mapped)
	}

	xml, err := t.BuildXML(tc, target)
	if err != nil {
		return err
	}

	if t.logger != nil {
		t.logger.Printf("[TrafficClass] XML generated for interface %s:\n%s", target.InterfaceName, xml)
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

func (t *TrafficClassNetconfPlugin) BuildXML(root *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_TrafficClass, target managementSessions.DeviceTarget) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">`)
	buf.WriteString(`<interface>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))
	buf.WriteString(`<bridge-port xmlns="urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge">`)
	buf.WriteString(`<traffic-class>`)
	//buf.WriteString(`<traffic-class-table>`)

	if root != nil && root.TrafficClassTable != nil {
		// NOTE: TTTech EVB doesn't support number-of-traffic-classes, it has to be activated for other switches
		//if root.TrafficClassTable.NumberOfTrafficClasses != nil {
		//	buf.WriteString(fmt.Sprintf(`<number-of-traffic-classes>%d</number-of-traffic-classes>`, *root.TrafficClassTable.NumberOfTrafficClasses))
		//}

		for _, field := range []struct {
			name  string
			value *uint8
		}{
			{name: "priority0", value: root.TrafficClassTable.Priority0},
			{name: "priority1", value: root.TrafficClassTable.Priority1},
			{name: "priority2", value: root.TrafficClassTable.Priority2},
			{name: "priority3", value: root.TrafficClassTable.Priority3},
			{name: "priority4", value: root.TrafficClassTable.Priority4},
			{name: "priority5", value: root.TrafficClassTable.Priority5},
			{name: "priority6", value: root.TrafficClassTable.Priority6},
			{name: "priority7", value: root.TrafficClassTable.Priority7},
		} {
			if field.value != nil {
				buf.WriteString(fmt.Sprintf(`<%s>%d</%s>`, field.name, *field.value, field.name))
			}
		}
	}

	//buf.WriteString(`</traffic-class-table>`)
	buf.WriteString(`</traffic-class>`)
	buf.WriteString(`</bridge-port>`)
	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)

	return buf.String(), nil
}
