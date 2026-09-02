package netconf

import (
	"context"
	"fmt"
	"strings"

	"OpenCNC/common/managementSessions"
	"OpenCNC/common/observability"
	storewrapper "OpenCNC/common/store-wrapper"
	credentials "OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/devicemodelregistry"
	"OpenCNC/common/structures/topology"
	topology_config "OpenCNC/common/structures/topology_config"
	"OpenCNC/config_service/pkg/plugins"
	protocolbackends "OpenCNC/config_service/pkg/protocolbackends"

	"github.com/openshift-telco/go-netconf-client/netconf"
)

var _ protocolbackends.ProtocolBackend = (*NetconfBackend)(nil)

//-----------------------------------
// Definition of the NetconfBackend
//-----------------------------------

type NetconfBackend struct {
	target         *topology.Node
	protocol       topology.ManagementProtocol
	plugins        []plugins.Plugin
	logger         observability.Logger
	snapshots      *protocolbackends.SnapshotSet[*NetconfSnapshot]
	sessionFactory *managementSessions.NetconfFactory
}

func NewNetconfBackend(target *topology.Node, cred *credentials.ManagementCredentials, logger observability.Logger) (*NetconfBackend, error) {

	if target == nil {
		return nil, fmt.Errorf("target node is nil")
	}
	if cred == nil {
		return nil, fmt.Errorf("management credentials are nil")
	}

	backend := &NetconfBackend{
		target:         target,
		protocol:       topology.ManagementProtocol_NETCONF,
		plugins:        plugins.ForProtocol(topology.ManagementProtocol_NETCONF, logger),
		logger:         observability.NormalizeLogger(logger),
		snapshots:      &protocolbackends.SnapshotSet[*NetconfSnapshot]{},
		sessionFactory: managementSessions.NewNetconfFactory(target, cred),
	}

	// get curent configuration
	if err := backend.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize NETCONF backend: %w", err)
	}
	return backend, nil
}

func init() {
	protocolbackends.Register(protocolbackends.BackendFactory{
		Protocol: topology.ManagementProtocol_NETCONF,
		New: func(
			target *topology.Node,
			cred *credentials.ManagementCredentials,
			logger observability.Logger,
		) (protocolbackends.ProtocolBackend, error) {
			return NewNetconfBackend(target, cred, logger)
		},
	})
}

func (b *NetconfBackend) Initialize() error {

	if b.sessionFactory == nil {
		return fmt.Errorf("session factory is nil")
	}

	session, err := b.sessionFactory.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create NETCONF session: %w", err)
	}
	defer session.Close()

	snapshot, err := GetRunningSnapshot(session)
	if err != nil {
		return fmt.Errorf("failed to retrieve running configuration: %w", err)
	}

	b.snapshots.Current = snapshot

	return nil
}

func (b *NetconfBackend) Name() string {
	return b.target.Name + "-" + b.protocol.String()
}

func (b *NetconfBackend) Protocol() topology.ManagementProtocol {
	return b.protocol
}

func (b *NetconfBackend) UpdateCredentials(cred *credentials.ManagementCredentials) {
	b.sessionFactory.UpdateCredentials(cred)
}

func (b *NetconfBackend) AddPlugin(plugin plugins.Plugin) {
	b.plugins = append(b.plugins, plugin)
}

func (b *NetconfBackend) Plugins() []plugins.Plugin {
	return b.plugins
}

func processUsedFields(nodeconfig *topology_config.NodeConfig, nodeUsed []string, portsUsed map[string][]string) {
	// Implement the logic to process used fields for the node and its ports.
	// This is a placeholder function and should be filled with the actual processing logic.
	// treat used/unused fields

	/*
		used := make(map[string]struct{})
		src := reflect.ValueOf(nodeconfig).Elem()
		srcType := src.Type()
		var unused []string

		for i := 0; i < src.NumField(); i++ {

			field := srcType.Field(i)

			if _, ok := used[field.Name]; ok {
				continue
			}

			v := src.Field(i)

			switch v.Kind() {
			case reflect.Pointer, reflect.Slice, reflect.Map:
				if v.IsNil() {
					continue
				}
			}

			unused = append(unused, field.Name)
		}

		if len(unused) == 0 {
			logger.Printf("All populated fields were handled.")
		} else {
			logger.Printf("Unused populated fields: %v", unused)
		}
	*/
}

func (b *NetconfBackend) PrepareSnapshot(ctx context.Context, msg *topology_config.NodeConfig) error {
	// Check cancellation before doing any work.
	if err := ctx.Err(); err != nil {
		return err
	}

	if b.target == nil {
		return fmt.Errorf("PrepareSnapshot: node is nil")
	}

	nodeConfig := msg
	if nodeConfig == nil {
		return fmt.Errorf("PrepareSnapshot: nodeConfig is nil")
	}

	if b.snapshots == nil || b.snapshots.Current == nil {
		return fmt.Errorf("no snapshot exists for node %s", b.target.Name)
	}

	//
	// Snapshot starts:
	// Current -> Working snapshot
	//
	b.snapshots.Working = b.snapshots.Current.Clone().(*NetconfSnapshot)
	if b.snapshots.Working == nil {
		return fmt.Errorf("failed creating working snapshot")
	}

	modelName := b.target.DeviceInfo.GetDeviceModel()
	nodeDeviceModel, err := storewrapper.GetDeviceModel(modelName)
	if err != nil {
		return fmt.Errorf("failed to retrieve device model %q: %w", modelName, err)
	}

	// apply node-level configuration
	nodeUsed, err := b.applyNodePlugin(ctx, nodeConfig, nodeDeviceModel, modelName)
	if err != nil {
		return err
	}

	portsUsed := map[string][]string{}

	for _, portConfig := range nodeConfig.PortConfigs {
		if portConfig == nil {
			continue
		}
		// Cancellation check between ports.
		if err := ctx.Err(); err != nil {
			return err
		}
		portUsed, err := b.applyPortPlugin(ctx, portConfig, nodeDeviceModel, modelName)
		if err != nil {
			return err
		}
		portsUsed[portConfig.PortId] = portUsed
	}
	processUsedFields(msg, nodeUsed, portsUsed)

	return nil
}

func (b *NetconfBackend) applyNodePlugin(ctx context.Context, config *topology_config.NodeConfig, nodeDeviceModel *devicemodelregistry.DeviceModel, modelName string) ([]string, error) {
	logger := b.logger

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	used := []string{}

	for _, plugin := range b.plugins {
		// Skip this plugin if it is not supported by the device model.
		if !plugin.SupportedByDevice(nodeDeviceModel) {
			logger.Printf(
				"Skipping node plugin %s: unsupported by device model %s",
				plugin.Name(),
				modelName,
			)
			continue
		}

		// get supported fields names for this plugin
		fields := plugin.SupportedFields(config)

		if len(fields) == 0 { //no fields supported, skip plugin
			logger.Printf(
				"Node plugin %-20s : no supported fields declared",
				plugin.Name(),
			)
			continue
		}

		if len(fields) == 1 && fields[0] == "PortConfigs" {
			// portConfig is treated separately
			continue
		}
		/////////////////////////
		// plugin application  //
		/////////////////////////
		// check for context cancellation before applying plugin
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// calling Map()
		mapped, err := plugin.Map(config, b.target)
		if err != nil { // map failed
			continue
		}

		//building feature XML
		featureXML, err := plugin.BuildFeatureXML(mapped)
		if err != nil { // build XML failed
			continue
		}

		////////////////////////
		//  snapshot update   //
		////////////////////////
		//check for context cancellation before updating snapshot
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// update snapshot
		if err := b.snapshots.Working.Update(featureXML, b.target, ""); err != nil { // update snapshot failed
			continue
		}

		////////////////////////////////////
		// register plugin fields as used //
		////////////////////////////////////
		for _, field := range fields {
			if field == "PortConfigs" {
				//"PortConfigs" are treated separately and not registered as used
				continue
			}
			used = append(used, field)
		}
	}

	return used, nil
}

func (b *NetconfBackend) applyPortPlugin(ctx context.Context, portConfig *topology_config.PortConfig, nodeDeviceModel *devicemodelregistry.DeviceModel, modelName string) ([]string, error) {
	logger := b.logger

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	used := []string{}
	for _, plugin := range b.plugins {
		// Skip this plugin if it is not supported by the device model.
		if !plugin.SupportedByDevice(nodeDeviceModel) {
			logger.Printf(
				"Skipping plugin %s: unsupported by device model %s",
				plugin.Name(),
				modelName,
			)
			continue
		}
		// get supported fields names for this plugin
		fields := plugin.SupportedFields(portConfig)
		if len(fields) == 0 {
			logger.Printf(
				"Plugin %-20s : no supported fields declared",
				plugin.Name(),
			)
			continue
		}

		// Check cancellation before calling plugin work.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		//calling Map
		mapped, err := plugin.Map(portConfig, b.target)
		if err != nil {
			continue
		}
		//building feature XML
		featureXML, err := plugin.BuildFeatureXML(mapped)
		if err != nil {
			continue
		}

		// Check cancellation before modifying the working snapshot.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if err := b.snapshots.Working.Update(featureXML, b.target, portConfig.PortId); err != nil {
			continue
		}
		logger.Printf("  -> snapshot update successful")

		// register plugin fields as used
		used = append(used, fields...)
	}
	return used, nil
}

func (b *NetconfBackend) Commit(ctx context.Context) error {

	if b.snapshots == nil {
		return fmt.Errorf(
			"no snapshot exists for node %s",
			b.target.Name,
		)
	}

	if b.snapshots.Working == nil {
		return fmt.Errorf(
			"no working snapshot for node %s",
			b.target.Name,
		)
	}

	b.logger.Printf(
		"Committing configuration for node %s",
		b.target.Name,
	)

	//
	// Push working configuration
	//
	if err := b.pushSnapshot(b.snapshots.Working); err != nil {
		return fmt.Errorf(
			"commit failed: %w",
			err,
		)
	}

	//
	// Snapshot promotion
	//
	b.snapshots.LastStable = b.snapshots.Current.Clone().(*NetconfSnapshot)
	b.snapshots.Current = b.snapshots.Working
	b.snapshots.Working = nil

	b.logger.Printf(
		"Commit successful for node %s",
		b.target.Name,
	)

	return nil
}

func (b *NetconfBackend) Rollback(ctx context.Context) error {

	if b.snapshots == nil {
		return fmt.Errorf(
			"no snapshot exists for node %s",
			b.target.Name,
		)
	}

	if b.snapshots.LastStable == nil {
		return fmt.Errorf(
			"no last stable snapshot for node %s, need to commit before rollback",
			b.target.Name,
		)
	}

	b.logger.Printf(
		"Rolling back configuration for node %s",
		b.target.Name,
	)

	//
	// Restore device configuration
	//
	if err := b.pushSnapshot(b.snapshots.LastStable); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	//
	// Restore runtime state
	//
	b.snapshots.Current = b.snapshots.LastStable.Clone().(*NetconfSnapshot)
	b.snapshots.LastStable = nil
	b.snapshots.Working = nil

	b.logger.Printf("Rollback successful for node %s", b.target.Name)

	return nil
}

func (b *NetconfBackend) pushSnapshot(snapshot *NetconfSnapshot) error {

	if snapshot == nil {
		return fmt.Errorf("snapshot is nil")
	}

	if len(snapshot.XML) == 0 {
		return fmt.Errorf("snapshot XML is empty")
	}

	session, err := b.sessionFactory.NewSession()
	if err != nil {
		return fmt.Errorf("NETCONF session failed: %w", err)
	}

	defer session.Close()

	if err := managementSessions.EditConfig(
		session,
		string(snapshot.XML),
	); err != nil {
		return fmt.Errorf("failed pushing snapshot: %w", err)
	}

	return nil
}

func GetRunningSnapshot(session *netconf.Session) (*NetconfSnapshot, error) {
	if session == nil {
		return nil, fmt.Errorf("GetRunningSnapshot: session is nil")
	}

	rawXML, err := managementSessions.GetRunningConfig(session)
	rawXML = stripRPCReply(rawXML)

	//DumpXML(rawXML)

	if err != nil {
		return nil, fmt.Errorf("failed to get running config: %w", err)
	}

	xmlBytes := []byte(rawXML)
	return &NetconfSnapshot{XML: append([]byte(nil), xmlBytes...)}, nil
}

func stripRPCReply(xml string) string {
	// Strip <rpc-reply ...>
	start := strings.Index(xml, ">")
	end := strings.LastIndex(xml, "</rpc-reply>")

	if start == -1 || end == -1 {
		return xml
	}

	xml = xml[start+1 : end]

	// Strip <data ...>
	start = strings.Index(xml, ">")
	end = strings.LastIndex(xml, "</data>")

	if start == -1 || end == -1 {
		return xml
	}

	// Keep everything inside <data> and wrap it in <config>
	return "<config>" + xml[start+1:end] + "</config>"
}

// //////////////////
/*
func DumpXML(s string) {
	f, err := os.Create("/home/opencnc/OpenCNC/debug.xml")
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(s)
}
*/
