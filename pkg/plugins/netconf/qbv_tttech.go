package netconf

import (
	"bytes"
	"fmt"
	"log"
	"sort"

	opencncModel "OpenCNC_config_service/opencnc_model"
	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	"OpenCNC_config_service/pkg/plugins"
	devicemodelregistry "OpenCNC_config_service/pkg/structures/devicemodelregistry"
	qbv "OpenCNC_config_service/pkg/structures/qbv"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"
)

// Ensure it implements Plugin interface
var _ plugins.Plugin = (*OldQbvNetconfPlugin)(nil)

type OldQbvNetconfPlugin struct {
	logger *log.Logger
}

func NewQbvNetconfPlugin_tttech(logger *log.Logger) *OldQbvNetconfPlugin {
	return &OldQbvNetconfPlugin{logger: logger}
}

func (p *OldQbvNetconfPlugin) Name() string {
	return "qbv-netconf"
}

func (p *OldQbvNetconfPlugin) FeatureName() string {
	return "qbv"
}

func (p *OldQbvNetconfPlugin) SupportedByDevice(model *devicemodelregistry.DeviceModel) bool {
	requiredYangs := []devicemodelregistry.YangFile{
		{Name: "ieee802-dot1q-sched.yang", Revision: "2018-09-10"},
		{Name: "ieee802-dot1q-sched-bridge.yang", Revision: "2018-09-10"},
	}

	for _, req := range requiredYangs {
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

func (p *OldQbvNetconfPlugin) Supports(msg proto.Message) bool {
	_, ok := msg.(*qbv.GateControlList)
	return ok
}

func (p *OldQbvNetconfPlugin) Map(msg proto.Message) (any, error) {
	gcl, ok := msg.(*qbv.GateControlList)
	if !ok {
		return nil, fmt.Errorf("invalid message type for OldQbvNetconfPlugin: %T", msg)
	}

	p.logger.Printf("[OldQbv] Mapping schedule ID: %s", gcl.GetScheduleId())

	// Calculate AdminGateStates as OR of all gateStates
	var adminGateStates uint8
	for _, e := range gcl.GetEntries() {
		if len(e.GateStates) > 0 {
			adminGateStates |= e.GateStates[0]
		}
	}

	ygotGcl := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable{
		GateEnabled:     ygot.Bool(gcl.GetAdminState() == qbv.AdminState_ENABLED),
		AdminGateStates: ygot.Uint8(adminGateStates),
		AdminBaseTime: &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminBaseTime{
			Seconds:     ygot.Uint64(gcl.GetBaseTime() / 1e9),
			Nanoseconds: ygot.Uint32(uint32(gcl.GetBaseTime() % 1e9)),
		},
		AdminCycleTime: &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminCycleTime{
			Numerator:   ygot.Uint32(uint32(gcl.GetCycleTime())),
			Denominator: ygot.Uint32(1000),
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
		if len(entry.GateStates) > 0 {
			ygotEntry.GateStatesValue = ygot.Uint8(entry.GateStates[0])
		}
		ygotGcl.AdminControlList.GateControlEntry[idx] = ygotEntry
	}

	return ygotGcl, nil
}

func (p *OldQbvNetconfPlugin) BuildXML(root *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable, target managementSessions.DeviceTarget) (string, error) {
	const (
		opSetAndHoldMac    = 1
		opSetAndReleaseMac = 2
		opSetGateStates    = 3
	)

	var buf bytes.Buffer
	nsIf := "urn:ietf:params:xml:ns:yang:ietf-interfaces"
	nsSched := "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"

	//buf.WriteString(`<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">`)
	buf.WriteString(fmt.Sprintf(`<interfaces xmlns="%s">`, nsIf))
	buf.WriteString(`<interface>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))
	buf.WriteString(fmt.Sprintf(`<gate-parameters xmlns="%s">`, nsSched))

	if root.GateEnabled != nil {
		buf.WriteString(fmt.Sprintf(`<gate-enabled>%t</gate-enabled>`, *root.GateEnabled))
	}
	if root.AdminGateStates != nil {
		buf.WriteString(fmt.Sprintf(`<admin-gate-states>%d</admin-gate-states>`, *root.AdminGateStates))
	}
	if root.AdminControlList != nil {
		buf.WriteString(fmt.Sprintf(`<admin-control-list-length>%d</admin-control-list-length>`, len(root.AdminControlList.GateControlEntry)))
	}

	// --- Write entries in order ---
	if root.AdminControlList != nil {
		indices := make([]int, 0, len(root.AdminControlList.GateControlEntry))
		for idx := range root.AdminControlList.GateControlEntry {
			indices = append(indices, int(idx))
		}
		sort.Ints(indices)
		for _, i := range indices {
			entry := root.AdminControlList.GateControlEntry[uint32(i)]
			buf.WriteString(`<admin-control-list>`)
			buf.WriteString(fmt.Sprintf(`<index>%d</index>`, *entry.Index))
			buf.WriteString(fmt.Sprintf(`<operation-name>%s</operation-name>`, entry.OperationName.String()))

			var paramsTag string
			switch entry.OperationName {
			case opSetAndHoldMac:
				paramsTag = "shm-params"
			case opSetAndReleaseMac:
				paramsTag = "srm-params"
			default:
				paramsTag = "sgs-params"
			}
			buf.WriteString(fmt.Sprintf(`<%s>`, paramsTag))
			if entry.GateStatesValue != nil {
				buf.WriteString(fmt.Sprintf(`<gate-states-value>%d</gate-states-value>`, *entry.GateStatesValue))
			}
			buf.WriteString(fmt.Sprintf(`<time-interval-value>%d</time-interval-value>`, *entry.TimeIntervalValue))
			buf.WriteString(fmt.Sprintf(`</%s>`, paramsTag))
			buf.WriteString(`</admin-control-list>`)
		}
	}

	if root.AdminCycleTime != nil {
		buf.WriteString(`<admin-cycle-time>`)
		buf.WriteString(fmt.Sprintf(`<numerator>%d</numerator>`, *root.AdminCycleTime.Numerator))
		buf.WriteString(fmt.Sprintf(`<denominator>%d</denominator>`, *root.AdminCycleTime.Denominator))
		buf.WriteString(`</admin-cycle-time>`)
	}
	buf.WriteString(`<admin-cycle-time-extension>0</admin-cycle-time-extension>`)

	if root.AdminBaseTime != nil {
		buf.WriteString(`<admin-base-time>`)
		buf.WriteString(fmt.Sprintf(`<seconds>%d</seconds>`, *root.AdminBaseTime.Seconds))
		buf.WriteString(fmt.Sprintf(`<fractional-seconds>%d</fractional-seconds>`, *root.AdminBaseTime.Nanoseconds))
		buf.WriteString(`</admin-base-time>`)
	}

	buf.WriteString(`<config-change>true</config-change>`)

	buf.WriteString(`</gate-parameters>`)
	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)
	//buf.WriteString(`</config>`)

	return buf.String(), nil
}

func (p *OldQbvNetconfPlugin) Push(mapped any, target managementSessions.DeviceTarget) error {
	root, ok := mapped.(*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable)
	if !ok {
		return fmt.Errorf("invalid mapped type for Push: %T", mapped)
	}

	xmlStr, err := p.BuildXML(root, target)
	if err != nil {
		return fmt.Errorf("failed to build XML: %w", err)
	}

	p.logger.Printf("[OldQBV] XML generated for interface %s", target.InterfaceName)

	session, err := managementSessions.CreateSession(target.Info.IpAddress, target.Info.UserName, target.Secret)
	if err != nil {
		return fmt.Errorf("NETCONF session failed: %w", err)
	}
	defer session.Close()

	if err := managementSessions.EditConfig(session, xmlStr); err != nil {
		return fmt.Errorf("edit-config failed: %w", err)
	}

	p.logger.Println("✅ Config pushed successfully")
	return nil
}
