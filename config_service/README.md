# OpenCNC Monitoring Service Architecture

## 1. Overview

The Monitoring Service is responsible for observing the operational state of the managed TSN network. It continuously collects runtime information from devices, derives metrics from the collected data, detects abnormal conditions, and reports events or corrective actions such as rollback requests.

The service is designed around four principles:

- A standardized set of counters is defined by OpenCNC.
- Protocol-specific logic is isolated inside protocol collectors.
- The monitoring core is independent of communication protocols and device models.
- Metrics and decisions operate only on normalized counter samples.

---

# 2. High-Level Architecture

```
                 +------------------------------+
                 |      Operation Manager       |
                 |------------------------------|
                 | Lifecycle                    |
                 | Monitoring configuration     |
                 | Scheduler                    |
                 | Alert subscriptions          |
                 +--------------+---------------+
                                |
                                v
                 +------------------------------+
                 |       Monitoring Engine      |
                 |------------------------------|
                 | Metric Engine                |
                 | Decision Engine              |
                 +--------------+---------------+
                                |
                                v
                 +------------------------------+
                 |         Collectors           |
                 |------------------------------|
                 | NETCONF Collector            |
                 | gNMI Collector               |
                 | SNMP Collector               |
                 +--------------+---------------+
                                |
                                v
                         Network Devices
```

---

# 3. Monitoring Model

The monitoring service operates on three kinds of information.

## Topology Status

Represents generic network operational state.

Examples

- Device reachability
- Interface status
- Link status
- Packet counters
- Error counters

Defined by:

- topology_status.proto

---

## Feature Status

Represents runtime state of TSN mechanisms.

Examples

- STP
- Qbv
- Qav
- PSFP
- FRER

Defined by

- stp_status.proto
- qbv_status.proto
- qav_status.proto
- psfp_status.proto
- frer_status.proto

---

## System Status

Represents the health of the device itself.

Examples

- CPU utilization
- Memory usage
- Queue occupancy
- Buffer utilization
- Synchronization state
- Resource fragmentation

---

# 4. Standardized Counter Model

The monitoring service does not allow arbitrary device-specific counters.

Instead, OpenCNC defines a standardized catalog of supported counters.

Each counter is described by `monitoring.proto`.

Example

```proto
message Counter {
    string id = 1;
    string name = 2;
    string path = 3;
    uint32 poll_interval_ms = 4;
    string description = 5;
}
```

The `path` field represents the protocol-specific object to retrieve:

- NETCONF XPath
- gNMI path
- SNMP OID
- Vendor-specific path

The catalog is distributed as JSON and generated from the OpenCNC YANG model.

---

# 5. Collector Layer

A collector implements a single southbound protocol.

Examples

- NETCONF Collector
- gNMI Collector
- SNMP Collector

A collector is responsible for

- communicating with devices
- understanding protocol semantics
- translating OpenCNC counters into protocol requests
- discovering which standardized counters are available on a device
- collecting requested counters
- converting protocol responses into normalized samples

The monitoring engine never communicates directly with devices.

---

## Device Capability Discovery

OpenCNC maintains a catalog of supported counters.

Each device exposes its supported YANG modules and revisions.

The collector determines the counters available on a device by computing the intersection between

```
OpenCNC Counter Catalog

∩

Device YANG Modules

=

Supported Counters
```

This ensures that metric definitions remain portable while still adapting to device capabilities.

---

# 6. Counter Collection

The Operation Manager schedules counter collection according to the polling interval defined for each counter.

Different counters may be collected at different rates.

Example

```
CPU utilization      5 s

Interface errors     1 s

Queue occupancy      100 ms

PTP synchronization  100 ms
```

Collectors may internally optimize protocol requests by grouping compatible counters into a single transaction.

For example

- multiple NETCONF XPaths in one `<get>`
- multiple gNMI paths in one Subscribe request
- multiple SNMP OIDs in one PDU

This optimization is transparent to the monitoring engine.

---

# 7. Counter Samples

Collectors convert raw protocol responses into normalized samples.

```go
type CounterSample struct {
    CounterID string
    NodeID    string
    Timestamp time.Time
    Value     float64
}
```

Regardless of whether a value originated from NETCONF, gNMI, SNMP, or another protocol, the monitoring engine always receives the same structure.

---

# 8. Metric Engine

The Metric Engine consumes normalized samples and computes metrics defined in `monitoring.proto`.

Examples include

- Throughput
- Utilization
- Delay
- Jitter
- Queue occupancy
- Error rate
- Congestion
- Clock synchronization error

A metric may depend on one or several counters and may require historical samples.

Example

```
rx_octets(t1)

rx_octets(t2)

↓

Throughput

↓

900 Mbps
```

Output

```go
type MetricResult struct {
    MetricName string
    NodeID     string
    Value      float64
    Timestamp  time.Time
}
```

---

# 9. Decision Engine

The Decision Engine evaluates computed metrics against the monitoring policies defined in `monitoring.proto`.

Each metric may define multiple thresholds.

Example

```
Queue Occupancy

Warning  > 70%

Critical > 90%
```

When a threshold is violated, the Decision Engine generates an alert.

```go
type Alert struct {
    Metric      string
    NodeID      string
    Severity    monitoring.Severity
    Description string
    Action      AlertAction
}
```

Possible actions include

- Notify
- Request rollback
- Trigger reconfiguration

The Decision Engine never performs these actions directly; it only emits requests.

---

# 10. Collector Interface

```go
type Collector interface {

    Name() string

    Protocol() string

    SupportedCounters(target Target) ([]*monitoring.Counter, error)

    Collect(
        target Target,
        counters []*monitoring.Counter,
    ) ([]*monitoring.CounterSample, error)
}
```

Target represents the managed device.

```go
type Target struct {
    ID          string
    Address     string
    Port        uint16
    YangModules []YangModule
}
```

```go
type YangModule struct {
    Name     string
    Revision string
}
```

---

# 11. Runtime Flow

```
Operation Manager
        │
        │ schedule collection
        ▼
Protocol Collector
        │
        │ Collect()
        ▼
CounterSample
        │
        ▼
Metric Engine
        │
        ▼
MetricResult
        │
        ▼
Decision Engine
        │
        ▼
Alert
        │
        ▼
Configuration Service (optional rollback)
```

---

# 12. Summary

The Monitoring Service separates concerns into independent layers.

- The **Operation Manager** determines *what* should be monitored and *when*.
- **Collectors** determine *how* standardized counters are retrieved using a specific protocol.
- The **Metric Engine** determines *what the collected data means*.
- The **Decision Engine** determines *whether the observed behavior requires action*.

By defining a fixed catalog of OpenCNC counters and discovering device support through YANG capabilities, metrics remain portable across vendors while collectors encapsulate all protocol-specific behavior. This architecture makes the monitoring service extensible, protocol-independent, and compatible with heterogeneous TSN deployments.