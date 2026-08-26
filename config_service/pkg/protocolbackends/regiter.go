package protocolbackends

import (
	"fmt"

	"OpenCNC/common/observability"
	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
)

type BackendFactory struct {
	Protocol topology.ManagementProtocol

	New func(
		target *topology.Node,
		cred *credentials.ManagementCredentials,
		logger observability.Logger,
	) (ProtocolBackend, error)
}

var registry = []BackendFactory{}

func Register(f BackendFactory) {
	registry = append(registry, f)
}

func ForProtocol(
	protocol topology.ManagementProtocol,
	target *topology.Node,
	cred *credentials.ManagementCredentials,
	logger observability.Logger,
) (ProtocolBackend, error) {

	logger = observability.NormalizeLogger(logger)

	for _, factory := range registry {
		if factory.Protocol == protocol {
			return factory.New(target, cred, logger)
		}
	}

	return nil, fmt.Errorf("no backend registered for protocol %v", protocol)
}
