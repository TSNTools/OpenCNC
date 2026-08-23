package collectors

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"OpenCNC_config_service/common/observability"
	observabilityv1 "OpenCNC_config_service/common/structures/logging"
	counters "OpenCNC_config_service/monitor_service/pkg/catalog"
	"OpenCNC_config_service/monitor_service/pkg/managementSessions"
	"OpenCNC_config_service/monitor_service/structures/monitoring"

	"github.com/openshift-telco/go-netconf-client/netconf"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NetconfCollector struct {
	target         *monitoring.ResourceKey
	session        *netconf.Session
	catalog        *counters.Catalog
	obs            *observability.Client
	PollIntervalMs uint32
	counters       []*monitoring.Counter
}

func NewNetconfCollector(
	target *monitoring.ResourceKey,
	session *netconf.Session,
	catalog *counters.Catalog,
	obs *observability.Client,
	PollIntervalMs uint32,
) *NetconfCollector {
	return &NetconfCollector{
		target:         target,
		session:        session,
		catalog:        catalog,
		obs:            obs,
		PollIntervalMs: PollIntervalMs,
		counters:       []*monitoring.Counter{},
	}
}

func (c *NetconfCollector) Protocol() string {
	return "netconf"
}

func (c *NetconfCollector) SupportedCounters() ([]*monitoring.Counter, error) {

	// TODO:
	// Later filter based on:
	// - NETCONF capabilities
	// - YANG modules
	// - device model
	// c.catalog.Counters

	return c.counters, nil
}

func (c *NetconfCollector) Target() *monitoring.ResourceKey {
	return c.target
}

func (c *NetconfCollector) PollInterval() uint32 {
	return c.PollIntervalMs
}

func (c *NetconfCollector) Counters() []*monitoring.Counter {
	return c.counters
}

func (c *NetconfCollector) AddCounter(counter *monitoring.Counter) error {
	if counter == nil {
		return fmt.Errorf("counter is nil")
	}

	counterData := c.catalog.GetCounter(counter.Name) // Ensure the counter exists in the catalog
	if counterData == nil {
		return fmt.Errorf("counter %s not found in catalog", counter.Name)
	}
	counterData.PollIntervalMs = c.PollIntervalMs // default to 1 second

	c.counters = append(c.counters, counterData)

	return nil
}

func (c *NetconfCollector) HasCounter(counterName string) bool {
	for _, counter := range c.counters {
		if counter.Name == counterName {
			return true
		}
	}
	return false
}

func (c *NetconfCollector) Collect(requestedCounters []*monitoring.Counter, ctx context.Context) ([]*monitoring.DataSample, error) {

	fmt.Println("counter:", requestedCounters)
	filter, err := buildFilter(requestedCounters)
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

	values, err := parseCounters(reply, requestedCounters)

	if err != nil {
		return nil, err
	}

	timestamp := timestamppb.New(time.Now())

	var samples []*monitoring.DataSample

	fmt.Printf("Collected: %f\n", values)

	for id, value := range values {

		samples = append(
			samples,
			&monitoring.DataSample{
				Id:        id,
				Source:    &monitoring.ResourceKey{NodeId: c.target.NodeId, PortId: c.target.PortId},
				Timestamp: timestamp,
				Value:     value,
				Kind:      monitoring.DataType_RAW,
			},
		)

		c.obs.Metric(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			id,
			observabilityv1.MetricType_METRIC_TYPE_GAUGE,
			value,
			"",
			map[string]string{
				"node_id": c.target.NodeId,
				"port_id": *c.target.PortId,
			},
		)
	}

	return samples, nil
}

// Builds a NETCONF subtree filter.
func buildFilter(requested []*monitoring.Counter) (string, error) {

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
	//buf.WriteString(fmt.Sprintf(`<name>%s</name>`, c.target.NodeId))

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
	Value   string `xml:",chardata"`
}

type interfaceStatistics struct {
	Counters []xmlCounter `xml:",any"`
}

type interfaceReply struct {
	XMLName xml.Name `xml:"rpc-reply"`

	Data struct {
		Interfaces []struct {
			Name string `xml:"name"`

			Statistics interfaceStatistics `xml:"statistics"`

			Ethernet struct {
				Statistics interfaceStatistics `xml:"statistics"`
			} `xml:"ethernet"`
		} `xml:"interfaces>interface"`
	} `xml:"data"`
}

func parseCounters(reply string, requested []*monitoring.Counter) (map[string]float64, error) {

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

		leafToID[leaf] = counter.Name
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
