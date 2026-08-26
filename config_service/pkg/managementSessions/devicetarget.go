package managementSessions

import (
	"OpenCNC/common/observability"
	"OpenCNC/common/structures/credentials"
	topology "OpenCNC/common/structures/topology"

	"github.com/openshift-telco/go-netconf-client/netconf"
)

type DeviceTarget struct {
	Info          *topology.ManagementInfo
	Credentials   *credentials.ManagementCredentials
	Logger        observability.Logger
	Session       *netconf.Session
	InterfaceName string
	// You can extend this with sessions, retry, TLS configs, etc.
}

type NodeCredentials struct {
	ID          string
	Credentials *credentials.ManagementCredentials
}
