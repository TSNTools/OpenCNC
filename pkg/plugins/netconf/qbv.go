package netconf

import (
	"bytes"
	"fmt"
	"log"

	opencncModel "OpenCNC_config_service/opencnc_model"
	managementSessions "OpenCNC_config_service/pkg/managementSessions"
	"OpenCNC_config_service/pkg/plugins"
	devicemodelregistry "OpenCNC_config_service/pkg/structures/devicemodelregistry"
	qbv "OpenCNC_config_service/pkg/structures/qbv"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"
)

// Ensure it implements the Plugin interface
var _ plugins.Plugin = (*QbvNetconfPlugin)(nil)

type QbvNetconfPlugin struct {
	logger *log.Logger
}

func NewQbvNetconfPlugin(logger *log.Logger) *QbvNetconfPlugin {
	return &QbvNetconfPlugin{logger: logger}
}

func (p *QbvNetconfPlugin) Name() string {
	return "qbv-netconf"
}

func (p *QbvNetconfPlugin) FeatureName() string {
	return "qbv"
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

func (p *QbvNetconfPlugin) Supports(msg proto.Message) bool {
	// returns true if the message can be mapped by this features
	_, ok := msg.(*qbv.GateControlList)
	return ok
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
			Numerator:   ygot.Uint32(uint32(gcl.GetCycleTime())), // already ns
			Denominator: ygot.Uint32(1000000000),                 // 1e9 ns per second
		},

		AdminControlList: &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminControlList{
			GateControlEntry: make(map[uint32]*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminControlList_GateControlEntry),
		},
	}

	// Constants for the OperationName enum values (based on ΛEnum in schedule.go)
	const (
		opSetAndHoldMac    = 1
		opSetAndReleaseMac = 2
		opSetGateStates    = 3
	)

	for _, entry := range gcl.GetEntries() {
		idx := entry.GetIndex()

		ygotEntry := &opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable_AdminControlList_GateControlEntry{
			Index:             ygot.Uint32(idx),
			TimeIntervalValue: ygot.Uint32(uint32(entry.GetTimeInterval())),
			OperationName:     opencncModel.E_Ieee802Dot1QTypes_TypeOfOperation(opSetGateStates), // default operation
		}

		// If gate_states is present, use the first byte
		if gateStates := entry.GetGateStates(); len(gateStates) > 0 {
			ygotEntry.GateStatesValue = ygot.Uint8(gateStates[0])
			//ygotGcl.AdminGateStates = ygot.Uint8(gateStates[0])
		}

		ygotGcl.AdminControlList.GateControlEntry[idx] = ygotEntry
	}

	// Log optional vendor-specific field
	if gcl.InterfaceTimeOffsetNs != nil {
		p.logger.Printf("[Qbv] Interface time offset (ns): %d", *gcl.InterfaceTimeOffsetNs)
		// Note: Not mapped in YANG structure
	}

	return ygotGcl, nil
}

func (p *QbvNetconfPlugin) Push(mapped any, target managementSessions.DeviceTarget) error {
	// ---------- check input ----------
	root, ok := mapped.(*opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable)
	if !ok {
		return fmt.Errorf("invalid mapped type for Push: %T", mapped)
	}

	// Automatically populate AdminGateStates from the first entry if not set
	if root.AdminGateStates == nil && root.AdminControlList != nil && len(root.AdminControlList.GateControlEntry) > 0 {
		for _, entry := range root.AdminControlList.GateControlEntry {
			if entry.GateStatesValue != nil {
				root.AdminGateStates = ygot.Uint8(*entry.GateStatesValue)
				break
			}
		}
	}

	// -------- build XML directly ----------
	xmlStr, err := p.BuildXML(root, target)
	if err != nil {
		return fmt.Errorf("failed to build XML: %w", err)
	}

	p.logger.Printf("[QBV] XML generated for interface %s:\n%s", target.InterfaceName, xmlStr)

	// ---------- Push via NETCONF ----------
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

	p.logger.Println("✅ Config pushed successfully")
	return nil
}

func (p *QbvNetconfPlugin) BuildXML(
	root *opencncModel.IETFInterfaces_Interfaces_Interface_BridgePort_GateParameterTable,
	target managementSessions.DeviceTarget,
) (string, error) {

	var buf bytes.Buffer

	nsIf := "urn:ietf:params:xml:ns:yang:ietf-interfaces"
	nsSched := "urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"

	//buf.WriteString(`<config>`)

	// Interfaces
	buf.WriteString(fmt.Sprintf(`<interfaces xmlns="%s">`, nsIf))
	buf.WriteString(`<interface>`)
	buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))

	// Bridge-port
	buf.WriteString(`<bridge-port>`)

	// Gate parameter table
	buf.WriteString(fmt.Sprintf(`<gate-parameter-table xmlns="%s">`, nsSched))

	// Gate enabled
	if root.GateEnabled != nil {
		buf.WriteString(fmt.Sprintf(`<gate-enabled>%t</gate-enabled>`, *root.GateEnabled))
	}

	// Base time
	if root.AdminBaseTime != nil {
		buf.WriteString(`<admin-base-time>`)
		buf.WriteString(fmt.Sprintf(`<seconds>%d</seconds>`, *root.AdminBaseTime.Seconds))
		buf.WriteString(fmt.Sprintf(`<nanoseconds>%d</nanoseconds>`, *root.AdminBaseTime.Nanoseconds))
		buf.WriteString(`</admin-base-time>`)
	}

	// Admin cycle time
	if root.AdminCycleTime != nil {
		buf.WriteString(`<admin-cycle-time>`)
		buf.WriteString(fmt.Sprintf(`<numerator>%d</numerator>`, *root.AdminCycleTime.Numerator))
		buf.WriteString(fmt.Sprintf(`<denominator>%d</denominator>`, *root.AdminCycleTime.Denominator))
		buf.WriteString(`</admin-cycle-time>`)
	}

	// Admin gate states
	if root.AdminGateStates != nil {
		buf.WriteString(fmt.Sprintf(`<admin-gate-states>%d</admin-gate-states>`, *root.AdminGateStates))
	}

	// Control list
	if root.AdminControlList != nil {
		buf.WriteString(`<admin-control-list>`)

		for _, entry := range root.AdminControlList.GateControlEntry {
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

	// Close tags
	buf.WriteString(`</gate-parameter-table>`)
	buf.WriteString(`</bridge-port>`)
	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)
	//buf.WriteString(`</config>`)

	return buf.String(), nil
}
