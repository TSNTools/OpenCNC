package discovery

import (
	"OpenCNC/common/managementSessions"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/openshift-telco/go-netconf-client/netconf"
	"github.com/openshift-telco/go-netconf-client/netconf/message"
)

type LLDPNeighbor struct {
	LocalPort        string
	RemoteChassisID  string
	RemotePortID     string
	RemoteSystemName string
	RemoteMgmtAddr   string
}

func GetWithFilter(session *netconf.Session, filter string) (string, error) {
	rpc := message.NewGet("subtree", filter)

	fmt.Printf("RPC: %+v\n", rpc)
	fmt.Println("Calling SyncRPC...")

	reply, err := session.SyncRPC(rpc, 30)
	if err != nil {
		return "", fmt.Errorf("get RPC failed: %w", err)
	}

	fmt.Println("SyncRPC returned")

	if reply == nil {
		return "", fmt.Errorf("nil reply")
	}

	return reply.RawReply, nil
}

func GetLLDPNeighbors(session *netconf.Session) ([]LLDPNeighbor, error) {
	fmt.Println("Testing NETCONF <get> with empty filter...")

	raw, err := managementSessions.GetWithFilter(
		session,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("NETCONF get test failed: %w", err)
	}

	fmt.Println("========== NETCONF RESPONSE ==========")
	fmt.Println(raw)
	fmt.Println("======================================")

	return nil, nil
}

func parseLLDP(data string) ([]LLDPNeighbor, error) {
	var root lldpXML

	if err := xml.Unmarshal([]byte(data), &root); err != nil {
		return nil, fmt.Errorf("parse LLDP XML: %w", err)
	}

	var result []LLDPNeighbor

	for _, intf := range root.Interfaces {
		for _, n := range intf.Neighbors {
			result = append(result, LLDPNeighbor{
				LocalPort:        strings.TrimSpace(intf.Name),
				RemoteChassisID:  strings.TrimSpace(n.ChassisID),
				RemotePortID:     strings.TrimSpace(n.PortID),
				RemoteSystemName: strings.TrimSpace(n.SystemName),
				RemoteMgmtAddr:   strings.TrimSpace(n.ManagementAddress),
			})
		}
	}

	return result, nil
}

type lldpXML struct {
	Interfaces []lldpInterface `xml:"interface"`
}

type lldpInterface struct {
	Name      string         `xml:"name"`
	Neighbors []lldpNeighbor `xml:"neighbor"`
}

type lldpNeighbor struct {
	ChassisID         string `xml:"chassis-id"`
	PortID            string `xml:"port-id"`
	SystemName        string `xml:"system-name"`
	ManagementAddress string `xml:"management-address"`
}
