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

	"github.com/golang/protobuf/proto"
)

var _ ProtocolBackend = (*NetconfBackend)(nil)

type NetconfBackend struct {
	name     string
	protocol topology.ManagementProtocol
	plugins  []plugins.Plugin
	logger   observability.Logger
}

func NewNetconfBackend(name string, logger observability.Logger, plugins ...plugins.Plugin) *NetconfBackend {
	return &NetconfBackend{
		name:     name,
		protocol: topology.ManagementProtocol_NETCONF,
		plugins:  plugins,
		logger:   observability.NormalizeLogger(logger),
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

/*
func (b *PluginBackend) GetPlugin(feature string) (plugins.Plugin, bool) {
	for _, plugin := range b.plugins {
	// TODO: Very important, it needs to filter according to device model
			return plugin, true
		}
	}
	return nil, false
}
*/

func (b *NetconfBackend) Plugins() []plugins.Plugin {
	return b.plugins
}

func (b *NetconfBackend) MapAndPush(msg proto.Message, node *topology.Node) error {
	logger := b.logger
	if node == nil {
		return fmt.Errorf("MapAndPush: node is nil")
	}

	// TODO: Very important, it needs to filter according to device model
	// need to pass the node instead of mgnInfo and get deviceModel from deviceInfo
	// then use supportedbydevice() to filterout the plugins that are not supported by the device model
	nodeConfig, ok := msg.(*topology_config.NodeConfig)
	if !ok {
		return fmt.Errorf("MapAndPush: invalid message type %T", msg)
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

		modelName := node.DeviceInfo.GetDeviceModel()
		nodeDeviceModel, err := storewrapper.GetDeviceModel(modelName)
		if err != nil {
			return fmt.Errorf(
				"failed to retrieve device model %q: %w",
				modelName,
				err,
			)
		}
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
				logger.Printf("Plugin %-20s : no supported fields declared", plugin.Name())
				continue
			}

			logger.Printf("Plugin %-20s : supports %v", plugin.Name(), fields)

			var input proto.Message

			if len(fields) == 1 {
				// Single field -> pass the field itself.
				f := src.FieldByName(fields[0])
				if !f.IsValid() {
					logger.Printf("  -> field %q does not exist", fields[0])
					continue
				}

				if f.IsNil() {
					logger.Printf("  -> field %q is nil, skipping", fields[0])
					continue
				}

				msg, ok := f.Interface().(proto.Message)
				if !ok {
					return fmt.Errorf("field %q does not implement proto.Message", fields[0])
				}

				logger.Printf("  -> mapping field %q (%T)", fields[0], msg)

				input = msg
				used[fields[0]] = struct{}{}

			} else {
				// Multiple fields -> build a partial PortConfig.
				dst := &topology_config.PortConfig{}
				dstVal := reflect.ValueOf(dst).Elem()

				found := false
				var mappedFields []string

				for _, name := range fields {
					sf := src.FieldByName(name)
					if !sf.IsValid() {
						logger.Printf("  -> field %q does not exist", name)
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
					logger.Printf("  -> none of the supported fields are present")
					continue
				}

				logger.Printf("  -> mapping PortConfig with fields %v", mappedFields)

				input = dst
			}

			logger.Printf("  -> calling Map()")

			mapped, err := plugin.Map(input)
			if err != nil {
				return fmt.Errorf("%s: %w", plugin.Name(), err)
			}

			logger.Printf("  <- Map() returned %T", mapped)

			logger.Printf("  -> calling Push()")

			if err := plugin.Push(mapped, target); err != nil {
				return fmt.Errorf("%s: %w", plugin.Name(), err)
			}

			logger.Printf("  <- Push() successful")
		}

		// Print remaining populated fields.
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

		logger.Printf("Finished processing port %q", portConfig.PortId)
		logger.Printf("======================================================")
	}

	return nil
}
