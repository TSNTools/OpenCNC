package collectors

import (
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

func (c *NetconfCollector) CounterIds() []string {
	ids := make([]string, len(c.counters))

	for i, counter := range c.counters {
		ids[i] = counter.Name
	}

	return ids
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

func (c *NetconfCollector) Collect(ctx context.Context) ([]*monitoring.DataSample, error) {

	requestedCounters := c.Counters()

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

	values, err := parseCounters(reply, requestedCounters, c.target)

	if err != nil {
		return nil, err
	}

	timestamp := timestamppb.New(time.Now())

	var samples []*monitoring.DataSample

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

		labels := map[string]string{
			"node_id": c.target.NodeId,
		}
		if c.target != nil && c.target.PortId != nil {
			labels["port_id"] = *c.target.PortId
		}

		c.obs.Metric(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			id,
			observabilityv1.MetricType_METRIC_TYPE_GAUGE,
			value,
			"",
			labels,
		)
	}

	return samples, nil
}

// Builds a broad NETCONF subtree filter. Counter selection is done locally.
func buildFilter(_ []*monitoring.Counter) (string, error) {
	const nsIf = "urn:ietf:params:xml:ns:yang:ietf-interfaces"
	const nsEth = "urn:ieee:std:802.3:yang:ieee802-ethernet-interface"

	filter := fmt.Sprintf(
		`<interfaces xmlns="%s"><interface><statistics/><ethernet xmlns="%s"><statistics/></ethernet></interface></interfaces>`,
		nsIf,
		nsEth,
	)

	return filter, nil
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

type interfaceStatistics struct {
	XMLName  xml.Name
	Value    string                `xml:",chardata"`
	Children []interfaceStatistics `xml:",any"`
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

type InterfaceCounters struct {
	Name     string
	Counters map[string]float64
}

func parseCounters(reply string, requested []*monitoring.Counter, target *monitoring.ResourceKey) (map[string]float64, error) {

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

	pathToID := make(map[string]string)

	for _, counter := range requested {
		pathToID[normalizePath(counter.Path)] = counter.Name
	}

	parsedInterfaces := make([]InterfaceCounters, 0, len(data.Data.Interfaces))

	for _, iface := range data.Data.Interfaces {
		ifaceCounters := InterfaceCounters{
			Name:     strings.TrimSpace(iface.Name),
			Counters: map[string]float64{},
		}

		collectStatisticsByPath(
			iface.Statistics,
			"/interfaces/interface/statistics",
			ifaceCounters.Counters,
		)

		collectStatisticsByPath(
			iface.Ethernet.Statistics,
			"/interfaces/interface/ethernet/statistics",
			ifaceCounters.Counters,
		)

		parsedInterfaces = append(parsedInterfaces, ifaceCounters)
	}

	values := make(map[string]float64)
	targetPort := ""
	if target != nil {
		targetPort = strings.TrimSpace(target.GetPortId())
	}

	for _, iface := range parsedInterfaces {
		if targetPort != "" && iface.Name != targetPort {
			continue
		}

		for counterPath, counterID := range pathToID {
			value, exists := iface.Counters[counterPath]
			if !exists {
				continue
			}

			values[counterID] = value
		}

		if targetPort != "" {
			break
		}
	}

	return values, nil
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	return "/" + strings.Trim(strings.TrimPrefix(trimmed, "/"), "/")
}

func collectStatisticsByPath(node interfaceStatistics, basePath string, out map[string]float64) {
	if len(node.Children) == 0 {
		value, err := strconv.ParseFloat(strings.TrimSpace(node.Value), 64)
		if err != nil {
			return
		}

		out[normalizePath(basePath)] = value
		return
	}

	for _, child := range node.Children {
		childPath := basePath + "/" + child.XMLName.Local
		collectStatisticsByPath(child, childPath, out)
	}
}
