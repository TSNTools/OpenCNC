package netconf

import (
	"bytes"
	"fmt"
	"sort"

	"OpenCNC_config_service/common/observability"
	devicemodelregistry "OpenCNC_config_service/common/structures/devicemodelregistry"
	qbv "OpenCNC_config_service/common/structures/qbv"
	"OpenCNC_config_service/common/structures/topology"
	topology_config "OpenCNC_config_service/common/structures/topology_config"
	opencncModel "OpenCNC_config_service/config_service/opencnc_model"
	managementSessions "OpenCNC_config_service/config_service/pkg/managementSessions"
	"OpenCNC_config_service/config_service/pkg/plugins"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"
)

// Ensure it implements the Plugin interface.
var _ plugins.Plugin = (*QbvNetconfPlugin)(nil)

type QbvNetconfPlugin struct {
	logger observability.Logger
}

func NewQbvNetconfPlugin(logger observability.Logger) *QbvNetconfPlugin {
	return &QbvNetconfPlugin{logger: observability.NormalizeLogger(logger)}
}

// plugin register itself
func init() {
	plugins.Register(plugins.PluginFactory{
		Protocol: topology.ManagementProtocol_NETCONF,
		New: func(logger observability.Logger) plugins.Plugin {
			return NewQbvNetconfPlugin(logger)
		},
	})
}

func (p *QbvNetconfPlugin) Name() string {
	return "qbv-netconf"
}

func (p *QbvNetconfPlugin) FeatureName() string {
	return "qbv"
}

func (p *QbvNetconfPlugin) SupportedFields(msg proto.Message) []string {
	if _, ok := msg.(*topology_config.PortConfig); !ok {
		return nil
	}

	return []string{
		"Gcl",
	}
}

func (p *QbvNetconfPlugin) SupportedByDevice(model *devicemodelregistry.DeviceModel) bool {
	requiredYangs := []devicemodelregistry.YangFile{
		{
			Name:     "ieee802-dot1q-sched.yang",
			Revision: "2021-04-09",
		},
		{
			Name:     "ieee802-dot1q-sched-bridge.yang",
			Revision: "2021-04-09",
		},
	}

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

func (p *QbvNetconfPlugin) Map(msg proto.Message) (any, error) {
	gcl, ok := msg.(*qbv.GateControlList)
	if !ok {
		return nil, fmt.Errorf("invalid message type for QbvNetconfPlugin: %T", msg)
	}

	p.logger.Printf("[Qbv] Mapping schedule ID: %s", gcl.GetScheduleId())

	// Mapping GateControlList to YGOT IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable
	ygotGcl := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable{
		GateEnabled: ygot.Bool(gcl.GetAdminState() == qbv.AdminState_ENABLED),
		AdminBaseTime: &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminBaseTime{
			Seconds:     ygot.Uint64(gcl.GetBaseTime() / 1e9),
			Nanoseconds: ygot.Uint32(uint32(gcl.GetBaseTime() % 1e9)),
		},
		AdminCycleTime: &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminCycleTime{
			Numerator:   ygot.Uint32(uint32(gcl.GetCycleTime())),
			Denominator: ygot.Uint32(1000000000),
		},
		AdminControlList: &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminControlList{
			GateControlEntry: make(map[uint32]*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminControlList_GateControlEntry),
		},
	}

	const opSetGateStates = 3

	for _, entry := range gcl.GetEntries() {
		idx := entry.GetIndex()

		ygotEntry := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminControlList_GateControlEntry{
			Index:             ygot.Uint32(idx),
			TimeIntervalValue: ygot.Uint32(uint32(entry.GetTimeInterval())),
			OperationName:     opencncModel.E_Ieee802Dot1QTypes_TypeOfOperation(opSetGateStates),
		}

		if gateStates := entry.GetGateStates(); len(gateStates) > 0 {
			ygotEntry.GateStatesValue = ygot.Uint8(gateStates[0])
		}

		ygotGcl.AdminControlList.GateControlEntry[idx] = ygotEntry
	}

	if gcl.InterfaceTimeOffsetNs != nil {
		p.logger.Printf("[Qbv] Interface time offset (ns): %d", *gcl.InterfaceTimeOffsetNs)
	}

	return ygotGcl, nil
}

func (p *QbvNetconfPlugin) BuildFeatureXML(mapped any) (*plugins.FeatureXML, error) {
	root, ok := mapped.(*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable)
	if !ok {
		return nil, fmt.Errorf("invalid mapped type for QbvNetconfPlugin: %T", mapped)
	}

	if root.AdminGateStates == nil && root.AdminControlList != nil && len(root.AdminControlList.GateControlEntry) > 0 {
		for _, entry := range root.AdminControlList.GateControlEntry {
			if entry.GateStatesValue != nil {
				root.AdminGateStates = ygot.Uint8(*entry.GateStatesValue)
				break
			}
		}
	}

	var buf bytes.Buffer
	nsSched := "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"

	buf.WriteString(fmt.Sprintf(`<gate-parameter-table xmlns="%s">`, nsSched))

	if root.GateEnabled != nil {
		buf.WriteString(fmt.Sprintf(`<gate-enabled>%t</gate-enabled>`, *root.GateEnabled))
	}

	if root.AdminBaseTime != nil {
		buf.WriteString(`<admin-base-time>`)
		buf.WriteString(fmt.Sprintf(`<seconds>%d</seconds>`, *root.AdminBaseTime.Seconds))
		buf.WriteString(fmt.Sprintf(`<nanoseconds>%d</nanoseconds>`, *root.AdminBaseTime.Nanoseconds))
		buf.WriteString(`</admin-base-time>`)
	}

	if root.AdminCycleTime != nil {
		buf.WriteString(`<admin-cycle-time>`)
		buf.WriteString(fmt.Sprintf(`<numerator>%d</numerator>`, *root.AdminCycleTime.Numerator))
		buf.WriteString(fmt.Sprintf(`<denominator>%d</denominator>`, *root.AdminCycleTime.Denominator))
		buf.WriteString(`</admin-cycle-time>`)
	}

	if root.AdminGateStates != nil {
		buf.WriteString(fmt.Sprintf(`<admin-gate-states>%d</admin-gate-states>`, *root.AdminGateStates))
	}

	if root.AdminControlList != nil {
		indices := make([]int, 0, len(root.AdminControlList.GateControlEntry))
		for idx := range root.AdminControlList.GateControlEntry {
			indices = append(indices, int(idx))
		}
		sort.Ints(indices)

		buf.WriteString(`<admin-control-list>`)
		for _, i := range indices {
			entry := root.AdminControlList.GateControlEntry[uint32(i)]
			buf.WriteString(`<gate-control-entry>`)
			buf.WriteString(fmt.Sprintf(`<index>%d</index>`, *entry.Index))
			buf.WriteString(fmt.Sprintf(`<operation-name>%s</operation-name>`, entry.OperationName.String()))
			buf.WriteString(fmt.Sprintf(`<time-interval-value>%d</time-interval-value>`, *entry.TimeIntervalValue))
			if entry.GateStatesValue != nil {
				buf.WriteString(fmt.Sprintf(`<gate-states-value>%d</gate-states-value>`, *entry.GateStatesValue))
			}
			buf.WriteString(`</gate-control-entry>`)
		}
		buf.WriteString(`</admin-control-list>`)
	}

	buf.WriteString(`</gate-parameter-table>`)

	return &plugins.FeatureXML{Container: "gate-parameter-table", XML: buf.Bytes()}, nil
}

func (p *QbvNetconfPlugin) Push(mapped any, target managementSessions.DeviceTarget) error {
	featurexml, err := p.BuildFeatureXML(mapped)
	if err != nil {
		return fmt.Errorf("failed to build feature XML: %w", err)
	}

	xmlStr, err := p.wrapXML(featurexml, target)
	if err != nil {
		return fmt.Errorf("failed to build XML: %w", err)
	}

	session, err := managementSessions.CreateSession(
		target.Info.IpAddress,
		target.Info.UserName,
		target.Secret,
	)
	if err != nil {
		return fmt.Errorf("NETCONF session failed: %w", err)
	}
	defer session.Close()

	if err := managementSessions.EditConfig(session, xmlStr); err != nil {
		return fmt.Errorf("edit-config failed: %w", err)
	}

	return nil
}

func (p *QbvNetconfPlugin) wrapXML(featurexml *plugins.FeatureXML, target managementSessions.DeviceTarget) (string, error) {
	var buf bytes.Buffer

	nsIf := "urn:ietf:params:xml:ns:yang:ietf-interfaces"
	nsBridge := "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge"

	buf.WriteString(fmt.Sprintf(`<interfaces xmlns="%s">`, nsIf))
	buf.WriteString(`<interface>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))
	buf.WriteString(fmt.Sprintf(`<bridge-port xmlns="%s">`, nsBridge))
	buf.WriteString(string(featurexml.XML))
	buf.WriteString(`</bridge-port>`)
	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)

	return buf.String(), nil
}
