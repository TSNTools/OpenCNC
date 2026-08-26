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

	return CreateSessionWithPassword(
		f.target.ManagementInfo.IpAddress,
		f.cred.GetUsernamePassword(),
	)
}
