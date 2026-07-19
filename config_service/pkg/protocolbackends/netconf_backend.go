package protocolbackends

import (
	"fmt"
	"reflect"

	"OpenCNC_config_service/common/observability"
	storewrapper "OpenCNC_config_service/common/store-wrapper"
	"OpenCNC_config_service/common/structures/topology"
	topology_config "OpenCNC_config_service/common/structures/topology_config"
	"OpenCNC_config_service/config_service/pkg/managementSessions"
	"OpenCNC_config_service/config_service/pkg/plugins"

	"github.com/beevik/etree"
	"github.com/golang/protobuf/proto"
)

var _ ProtocolBackend = (*NetconfBackend)(nil)

type NetconfSnapshot struct {
	XML []byte // parsed model, cached payload, metadata...

}

func (s *NetconfSnapshot) Clone() Snapshot {
	//todo: check it, this is a placeholder so far
	return &NetconfSnapshot{
		XML: append([]byte(nil), s.XML...),
	}
}

func (s *NetconfSnapshot) Update(feature *plugins.FeatureXML, target managementSessions.DeviceTarget) error {

	if feature == nil {
		return fmt.Errorf("feature XML is nil")
	}

	if len(feature.XML) == 0 {
		return fmt.Errorf("feature XML is empty")
	}

	doc := etree.NewDocument()

	if err := doc.ReadFromBytes(s.XML); err != nil {
		return fmt.Errorf("failed parsing snapshot XML: %w", err)
	}

	interfaces := doc.FindElement("//interfaces")
	if interfaces == nil {
		return fmt.Errorf("snapshot does not contain <interfaces>")
	}

	var interfaceElement *etree.Element

	for _, intf := range interfaces.FindElements("interface") {

		name := intf.FindElement("name")

		if name == nil {
			continue
		}

		if name.Text() == target.InterfaceName {
			interfaceElement = intf
			break
		}
	}

	if interfaceElement == nil {
		return fmt.Errorf(
			"interface %q not found in snapshot",
			target.InterfaceName,
		)
	}

	// Remove existing feature subtree.
	if existing := interfaceElement.FindElement(feature.Container); existing != nil {
		interfaceElement.RemoveChild(existing)
	}

	// Parse new feature subtree.
	featureDoc := etree.NewDocument()

	if err := featureDoc.ReadFromBytes(feature.XML); err != nil {
		return fmt.Errorf(
			"failed parsing feature XML: %w",
			err,
		)
	}

	if featureDoc.Root() == nil {
		return fmt.Errorf(
			"feature XML has no root element",
		)
	}

	// Add the updated feature subtree.
	interfaceElement.AddChild(
		featureDoc.Root().Copy(),
	)

	// Store updated snapshot.
	doc.Indent(2)

	updatedXML, err := doc.WriteToBytes()
	if err != nil {
		return fmt.Errorf(
			"failed serializing updated snapshot: %w",
			err,
		)
	}

	s.XML = updatedXML

	return nil
}

//-----------------------------------
// Definition of the NetconfBackend
//-----------------------------------

type NetconfBackend struct {
	name      string
	protocol  topology.ManagementProtocol
	plugins   []plugins.Plugin
	logger    observability.Logger
	snapshots map[string]*SnapshotSet[*NetconfSnapshot]
}

func NewNetconfBackend(name string, logger observability.Logger, plugins ...plugins.Plugin) *NetconfBackend {
	return &NetconfBackend{
		name:      name,
		protocol:  topology.ManagementProtocol_NETCONF,
		plugins:   plugins,
		logger:    observability.NormalizeLogger(logger),
		snapshots: make(map[string]*SnapshotSet[*NetconfSnapshot]),
	}
}

func (b *NetconfBackend) Name() string {
	return b.name
}

func (b *NetconfBackend) Protocol() topology.ManagementProtocol {
	return b.protocol
}

func (b *NetconfBackend) AddPlugin(plugin plugins.Plugin) {
	b.plugins = append(b.plugins, plugin)
}

func (b *NetconfBackend) Plugins() []plugins.Plugin {
	return b.plugins
}

func (b *NetconfBackend) PrepareSnapshot(msg *topology_config.NodeConfig, node *topology.Node) error {
	logger := b.logger

	if node == nil {
		return fmt.Errorf("PrepareSnapshot: node is nil")
	}

	nodeConfig := msg
	if nodeConfig == nil {
		return fmt.Errorf("PrepareSnapshot: nodeConfig is nil")
	}

	snapshotSet, ok := b.snapshots[node.Name]
	if !ok {
		return fmt.Errorf("no snapshot exists for node %s", node.Name)
	}

	//
	// Snapshot starts:
	// Current -> Working snapshot
	//
	snapshotSet.Working = snapshotSet.Current.Clone().(*NetconfSnapshot)

	if snapshotSet.Working == nil {
		return fmt.Errorf("failed creating working snapshot")
	}

	modelName := node.DeviceInfo.GetDeviceModel()

	nodeDeviceModel, err := storewrapper.GetDeviceModel(modelName)
	if err != nil {
		return fmt.Errorf(
			"failed to retrieve device model %q: %w",
			modelName,
			err,
		)
	}

	for _, portConfig := range nodeConfig.PortConfigs {

		if portConfig == nil {
			continue
		}

		logger.Printf("======================================================")
		logger.Printf("Processing port %q", portConfig.PortId)

		target := managementSessions.DeviceTarget{
			InterfaceName: portConfig.PortId,
			Logger:        logger,
			Secret:        "",
			Info:          node.ManagementInfo,
		}

		used := make(map[string]struct{})

		src := reflect.ValueOf(portConfig).Elem()
		srcType := src.Type()

		for _, plugin := range b.plugins {

			if !plugin.SupportedByDevice(nodeDeviceModel) {
				logger.Printf(
					"Skipping plugin %s: unsupported by device model %s",
					plugin.Name(),
					modelName,
				)
				continue
			}

			fields := plugin.SupportedFields(portConfig)

			if len(fields) == 0 {
				logger.Printf(
					"Plugin %-20s : no supported fields declared",
					plugin.Name(),
				)
				continue
			}

			logger.Printf(
				"Plugin %-20s : supports %v",
				plugin.Name(),
				fields,
			)

			var input proto.Message

			if len(fields) == 1 {

				f := src.FieldByName(fields[0])

				if !f.IsValid() || f.IsNil() {
					logger.Printf(
						"  -> field %q is not valid, skipping",
						fields[0],
					)
					continue
				}

				fieldMsg, ok := f.Interface().(proto.Message)
				if !ok {
					return fmt.Errorf(
						"field %q does not implement proto.Message",
						fields[0],
					)
				}

				logger.Printf(
					"  -> mapping field %q (%T)",
					fields[0],
					fieldMsg,
				)

				input = fieldMsg
				used[fields[0]] = struct{}{}

			} else {

				dst := &topology_config.PortConfig{}
				dstVal := reflect.ValueOf(dst).Elem()

				found := false
				var mappedFields []string

				for _, name := range fields {

					sf := src.FieldByName(name)

					if !sf.IsValid() {
						logger.Printf(
							"  -> field %q does not exist",
							name,
						)
						continue
					}

					switch sf.Kind() {
					case reflect.Pointer, reflect.Slice, reflect.Map:
						if sf.IsNil() {
							continue
						}
					}

					dstVal.FieldByName(name).Set(sf)

					used[name] = struct{}{}
					mappedFields = append(mappedFields, name)
					found = true
				}

				if !found {
					logger.Printf(
						"  -> none of the supported fields are present",
					)
					continue
				}

				logger.Printf(
					"  -> mapping PortConfig with fields %v",
					mappedFields,
				)

				input = dst
			}

			logger.Printf("  -> calling Map()")

			mapped, err := plugin.Map(input)
			if err != nil {
				return fmt.Errorf(
					"%s: %w",
					plugin.Name(),
					err,
				)
			}

			logger.Printf(
				"  <- Map() returned %T",
				mapped,
			)

			logger.Printf("  -> building feature XML")

			featureXML, err := plugin.BuildFeatureXML(mapped)

			if err != nil {
				return fmt.Errorf("%s: %w", plugin.Name(), err)
			}

			if err := snapshotSet.Working.Update(
				featureXML,
				target,
			); err != nil {
				return fmt.Errorf("failed to update snapshot: %w", err)
			}

			logger.Printf("  -> snapshot update successful")
		}

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
			logger.Printf(
				"Unused populated fields: %v",
				unused,
			)
		}

		logger.Printf(
			"Finished processing port %q",
			portConfig.PortId,
		)

		logger.Printf("======================================================")
	}

	logger.Printf(
		"Snapshot prepared successfully for node %s",
		node.Name,
	)

	return nil
}

func (b *NetconfBackend) Commit(target *topology.Node) error {

	if target == nil {
		return fmt.Errorf("Commit: node is nil")
	}

	snapshotSet, ok := b.snapshots[target.Name]
	if !ok {
		return fmt.Errorf(
			"no snapshot exists for node %s",
			target.Name,
		)
	}

	if snapshotSet.Working == nil {
		return fmt.Errorf(
			"no working snapshot for node %s",
			target.Name,
		)
	}

	b.logger.Printf(
		"Committing configuration for node %s",
		target.Name,
	)

	//
	// Push working configuration
	//
	if err := b.pushSnapshot(
		snapshotSet.Working,
		target,
	); err != nil {
		return fmt.Errorf(
			"commit failed: %w",
			err,
		)
	}

	//
	// Snapshot promotion
	//
	snapshotSet.LastStable = snapshotSet.Current
	snapshotSet.Current = snapshotSet.Working
	snapshotSet.Working = nil

	b.logger.Printf(
		"Commit successful for node %s",
		target.Name,
	)

	return nil
}

func (b *NetconfBackend) Rollback(target *topology.Node) error {

	if target == nil {
		return fmt.Errorf("Rollback: node is nil")
	}

	snapshotSet, ok := b.snapshots[target.Name]
	if !ok {
		return fmt.Errorf(
			"no snapshot exists for node %s",
			target.Name,
		)
	}

	if snapshotSet.LastStable == nil {
		return fmt.Errorf(
			"no last stable snapshot for node %s",
			target.Name,
		)
	}

	b.logger.Printf(
		"Rolling back configuration for node %s",
		target.Name,
	)

	//
	// Restore device configuration
	//
	if err := b.pushSnapshot(
		snapshotSet.LastStable,
		target,
	); err != nil {
		return fmt.Errorf(
			"rollback failed: %w",
			err,
		)
	}

	//
	// Restore runtime state
	//
	snapshotSet.Current = snapshotSet.LastStable.Clone().(*NetconfSnapshot)
	snapshotSet.Working = nil

	b.logger.Printf(
		"Rollback successful for node %s",
		target.Name,
	)

	return nil
}

func (b *NetconfBackend) pushSnapshot(snapshot *NetconfSnapshot, node *topology.Node) error {

	if snapshot == nil {
		return fmt.Errorf("snapshot is nil")
	}

	if len(snapshot.XML) == 0 {
		return fmt.Errorf("snapshot XML is empty")
	}

	session, err := managementSessions.CreateSession(
		node.ManagementInfo.IpAddress,
		node.ManagementInfo.UserName,
		"",
	)

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
