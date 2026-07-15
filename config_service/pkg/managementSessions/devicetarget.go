package managementSessions

import (
	topology "OpenCNC_config_service/common/structures/topology"
	"log"

	"github.com/openshift-telco/go-netconf-client/netconf"
)

type DeviceTarget struct {
	Info          *topology.ManagementInfo
	Secret        string
	Logger        *log.Logger
	Session       *netconf.Session
	InterfaceName string
	// You can extend this with sessions, retry, TLS configs, etc.
}
