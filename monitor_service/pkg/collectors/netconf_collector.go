package collectors

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"OpenCNC_config_service/common/structures/topology"
	"OpenCNC_config_service/monitor_service/pkg/counters"
	"OpenCNC_config_service/monitor_service/pkg/managementSessions"
	"OpenCNC_config_service/monitor_service/structures/monitoring"

	"github.com/openshift-telco/go-netconf-client/netconf"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NetconfCollector struct {
	session *netconf.Session
	catalog *counters.Catalog
}

func NewNetconfCollector(
	session *netconf.Session,
	catalog *counters.Catalog,
) *NetconfCollector {

	return &NetconfCollector{
		session: session,
		catalog: catalog,
	}
}

func (c *NetconfCollector) Protocol() string {
	return "netconf"
}

func (c *NetconfCollector) SupportedCounters(
	target topology.Node,
) ([]*monitoring.Counter, error) {

	// TODO:
	// Later filter based on:
	// - NETCONF capabilities
	// - YANG modules
	// - device model

	return c.catalog.Counters, nil
}

func (c *NetconfCollector) Collect(
	target *topology.Node,
	requestedCounters []*monitoring.Counter,

) ([]*monitoring.CounterSample, error) {

	filter, err := buildFilter(target, requestedCounters)
	if err != nil {
		return nil, fmt.Errorf("failed to build filter: %w", err)
	}

	reply, err := managementSessions.GetWithFilter(
		c.session,
		filter,
	)

	if err != nil {
		return nil,
			fmt.Errorf(
				"NETCONF get failed: %w",
				err,
			)
	}

	values, err := parseCounters(
		reply,
		requestedCounters,
	)

	if err != nil {
		return nil, err
	}

	timestamp := timestamppb.New(time.Now())

	var samples []*monitoring.CounterSample

	for id, value := range values {

		samples = append(
			samples,
			&monitoring.CounterSample{
				CounterId: id,
				NodeId:    target.Name,
				Timestamp: timestamp,
				Value:     value,
			},
		)
	}

	return samples, nil
}

// Builds a NETCONF subtree filter.
func buildFilter(target *topology.Node, requested []*monitoring.Counter) (string, error) {

	// -----------------------------------------------------------------
	// Interface statistics
	// -----------------------------------------------------------------

	var interfaceStatistics []string
	var ethernetStatistics []string

	for _, counter := range requested {

		path := strings.Trim(counter.Path, "/")

		switch {

		case strings.HasPrefix(path, "interfaces/interface/statistics/"):
			interfaceStatistics = append(
				interfaceStatistics,
				path[strings.LastIndex(path, "/")+1:],
			)

		case strings.HasPrefix(path, "interfaces/interface/ethernet/statistics/"):
			ethernetStatistics = append(
				ethernetStatistics,
				path[strings.LastIndex(path, "/")+1:],
			)
		}
	}

	// -----------------------------------------------------------------
	// build filter
	// -----------------------------------------------------------------

	var buf bytes.Buffer

	const nsIf = "urn:ietf:params:xml:ns:yang:ietf-interfaces"
	const nsEth = "urn:ieee:std:802.3:yang:ieee802-ethernet-interface"

	buf.WriteString(fmt.Sprintf(`<interfaces xmlns="%s">`, nsIf))
	buf.WriteString(`<interface>`)
	//buf.WriteString(fmt.Sprintf(`<name>%s</name>`, target.InterfaceName))

	if len(interfaceStatistics) > 0 {

		buf.WriteString(`<statistics>`)

		for _, leaf := range interfaceStatistics {
			buf.WriteString(fmt.Sprintf(`<%s/>`, leaf))
		}

		buf.WriteString(`</statistics>`)
	}

	// -----------------------------------------------------------------
	// Ethernet statistics
	// -----------------------------------------------------------------

	if len(ethernetStatistics) > 0 {

		buf.WriteString(fmt.Sprintf(`<ethernet xmlns="%s">`, nsEth))
		buf.WriteString(`<statistics>`)

		for _, leaf := range ethernetStatistics {
			buf.WriteString(fmt.Sprintf(`<%s/>`, leaf))
		}

		buf.WriteString(`</statistics>`)
		buf.WriteString(`</ethernet>`)
	}

	buf.WriteString(`</interface>`)
	buf.WriteString(`</interfaces>`)

	return buf.String(), nil
}

func formatLeaves(leaves []string) string {

	var result string

	for _, leaf := range leaves {

		result += fmt.Sprintf(
			"     <%s/>\n",
			leaf,
		)
	}

	return result
}

// -----------------------------------------------------
// NETCONF XML parsing
// -----------------------------------------------------

type xmlCounter struct {
	XMLName xml.Name

	Value string `xml:",chardata"`
}

type interfaceStatistics struct {
	Counters []xmlCounter `xml:",any"`
}

type interfaceReply struct {
	XMLName xml.Name `xml:"rpc-reply"`

	Data struct {
		Interfaces []struct {
			Name string `xml:"name"`

			Ethernet struct {
				Statistics interfaceStatistics `xml:"statistics"`
			} `xml:"ethernet"`
		} `xml:"interfaces>interface"`
	} `xml:"data"`
}

func parseCounters(
	reply string,
	requested []*monitoring.Counter,

) (map[string]float64, error) {

	var data interfaceReply

	err := xml.Unmarshal(
		[]byte(reply),
		&data,
	)

	if err != nil {

		return nil,
			fmt.Errorf(
				"failed parsing NETCONF reply: %w",
				err,
			)
	}

	// map XML leaf -> OpenCNC counter ID

	leafToID := make(map[string]string)

	for _, counter := range requested {

		parts := strings.Split(
			counter.Path,
			"/",
		)

		leaf := parts[len(parts)-1]

		leafToID[leaf] = counter.Id
	}

	values := make(map[string]float64)

	for _, iface := range data.Data.Interfaces {

		for _, counter := range iface.Ethernet.Statistics.Counters {

			counterID, exists :=
				leafToID[counter.XMLName.Local]

			if !exists {
				continue
			}

			value, err := strconv.ParseFloat(
				strings.TrimSpace(counter.Value),
				64,
			)

			if err != nil {
				continue
			}

			values[counterID] = value
		}
	}

	return values, nil
}
