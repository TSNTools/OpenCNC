package managementSessions

import (
	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
	"fmt"

	"github.com/openshift-telco/go-netconf-client/netconf"
)

type NetconfFactory struct {
	target *topology.Node
	cred   *credentials.ManagementCredentials
}

func NewNetconfFactory(target *topology.Node, cred *credentials.ManagementCredentials) *NetconfFactory {
	return &NetconfFactory{
		target: target,
		cred:   cred,
	}
}

func (f *NetconfFactory) NewSession() (*netconf.Session, error) {
	if f == nil {
		return nil, fmt.Errorf("NETCONF factory is nil")
	}

	if f.target == nil {
		return nil, fmt.Errorf("NETCONF target is nil")
	}

	if f.cred == nil {
		return nil, fmt.Errorf("NETCONF credentials are nil")
	}

	info := f.target.GetManagementInfo()
	if info == nil || info.GetIpAddress() == "" {
		return nil, fmt.Errorf("NETCONF management IP address is required")
	}
	if info.GetTransport() != topology.ManagementTransport_MANAGEMENT_TRANSPORT_UNSPECIFIED && info.GetTransport() != topology.ManagementTransport_SSH {
		return nil, fmt.Errorf("unsupported NETCONF transport %s", info.GetTransport())
	}
	port := info.GetManagementPort()
	if port == 0 {
		port = 830
	}
	if port > 65535 {
		return nil, fmt.Errorf("invalid NETCONF management port %d", port)
	}

	return createSessionWithPasswordOnPort(
		info.GetIpAddress(),
		port,
		f.cred.GetUsernamePassword(),
	)
}

func (f *NetconfFactory) UpdateCredentials(cred *credentials.ManagementCredentials) {
	f.cred = cred
}
