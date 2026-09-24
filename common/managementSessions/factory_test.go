package managementSessions

import (
	"OpenCNC/common/structures/credentials"
	"OpenCNC/common/structures/topology"
	"net"
	"testing"
	"time"
)

func TestFactoryUsesTopologyManagementPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
			close(accepted)
		}
	}()
	node := &topology.Node{ManagementInfo: &topology.ManagementInfo{
		IpAddress: "127.0.0.1", ManagementPort: uint32(listener.Addr().(*net.TCPAddr).Port),
		Transport: topology.ManagementTransport_SSH,
	}}
	creds := &credentials.ManagementCredentials{Authentication: &credentials.ManagementCredentials_UsernamePassword{
		UsernamePassword: &credentials.UsernamePassword{Username: "test", Password: "test"},
	}}
	_, err = NewNetconfFactory(node, creds).NewSession()
	if err == nil {
		t.Fatal("expected SSH handshake failure from test listener")
	}
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("did not connect to topology management port")
	}
}
