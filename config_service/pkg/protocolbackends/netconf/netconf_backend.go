package netconf

import (
	"context"
	"fmt"
	"reflect"

	"OpenCNC/common/managementSessions"
	"OpenCNC/common/observability"
	storewrapper "OpenCNC/common/store-wrapper"
	credentials "OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
	topology_config "OpenCNC/common/structures/topology_config"
	"OpenCNC/config_service/pkg/plugins"
	protocolbackends "OpenCNC/config_service/pkg/protocolbackends"

	"github.com/golang/protobuf/proto"
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

func (b *NetconfBackend) PrepareSnapshot(ctx context.Context, msg *topology_config.NodeConfig) error {
	logger := b.logger

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
		return fmt.Errorf(
			"failed to retrieve device model %q: %w",
			modelName,
			err,
		)
	}

	for _, portConfig := range nodeConfig.PortConfigs {

		// Cancellation check between ports.
		if err := ctx.Err(); err != nil {
			return err
		}

		if portConfig == nil {
			continue
		}

		logger.Printf("======================================================")
		logger.Printf("Processing port %q", portConfig.PortId)

		used := make(map[string]struct{})

		src := reflect.ValueOf(portConfig).Elem()
		srcType := src.Type()

		for _, plugin := range b.plugins {
			// Cancellation check between plugins.
			if err := ctx.Err(); err != nil {
				return err
			}

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
					// Cancellation check inside the field loop too.
					if err := ctx.Err(); err != nil {
						return err
					}

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

			// Check before calling plugin work.
			if err := ctx.Err(); err != nil {
				return err
			}
			mapped, err := plugin.Map(input)
			if err != nil {
				return fmt.Errorf("%s: %w", plugin.Name(), err)
			}
			logger.Printf("  <- Map() returned %T", mapped)
			logger.Printf("  -> building feature XML")
			featureXML, err := plugin.BuildFeatureXML(mapped)
			if err != nil {
				return fmt.Errorf("%s: %w", plugin.Name(), err)
			}

			// Check before modifying the working snapshot.
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := b.snapshots.Working.Update(
				featureXML,
				b.target,
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
		b.target.Name,
	)

	return nil
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
	if err != nil {
		return nil, fmt.Errorf("failed to get running config: %w", err)
	}

	xmlBytes := []byte(rawXML)
	return &NetconfSnapshot{XML: append([]byte(nil), xmlBytes...)}, nil
}
