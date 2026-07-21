# OpenCNC Monitoring Service Architecture

## 1. Overview

The Monitoring Service is responsible for observing the runtime behavior of the managed network and detecting deviations from the expected operational state.

The service has three main responsibilities:

1. Collect operational data from network devices and systems.
2. Transform raw observations into meaningful metrics.
3. Detect abnormal conditions and trigger actions such as alarms or configuration rollback.

The monitoring service is designed around the following principle:

- Protocol-specific logic is isolated in plugins.
- The monitoring core is independent of device models and communication protocols.
- Metrics and decisions operate on normalized data.

The architecture is divided into:

+------------------------------------------------+
|                Monitoring Service              |
|                                                |
|  +------------------------------------------+  |
|  | Operation Manager                        |  |
|  | - lifecycle management                   |  |
|  | - monitoring policies                    |  |
|  | - subscriptions                          |  |
|  +------------------------------------------+  |
|                     |                          |
|  +------------------------------------------+  |
|  | Monitoring Engine                        |  |
|  | - metric evaluation                      |  |
|  | - anomaly detection                      |  |
|  | - alert generation                       |  |
|  +------------------------------------------+  |
|                     |                          |
|  +------------------------------------------+  |
|  | Collector                                |  |
|  | - schedules counter collection           |  |
|  | - manages samples                        |  |
|  +------------------------------------------+  |
|                     |                          |
|  +------------------------------------------+  |
|  | Protocol Plugins                         |  |
|  | - gNMI                                   |  |
|  | - NETCONF                                |  |
|  | - SNMP                                   |  |
|  | - vendor specific APIs                   |  |
|  +------------------------------------------+  |
+------------------------------------------------+

---

# 2. Design Principles

## 2.1 Protocol independence

The monitoring engine must not know:

- whether data was collected through gNMI, NETCONF, SNMP, or another protocol.
- which vendor model is used.
- how a device exposes a counter.

The engine only consumes normalized samples:


CounterSample
|
v
Metric calculation
|
v
MetricResult
|
v
Decision engine

---

## 2.2 Plugin responsibility

Plugins are responsible for device interaction.

A plugin handles:

- protocol communication.
- device model translation.
- path construction.
- subscription handling if supported.
- polling if required.

Examples:

### gNMI plugin

Capabilities:

- native telemetry subscription.
- streaming counters.
- event notification.

Example:

Subscribe
|
|
Device
|
|
CounterSample stream



---

### NETCONF plugin

NETCONF generally does not provide streaming telemetry.

The plugin implements:

Timer
|
|
NETCONF get
|
|
Parse reply
|
|
CounterSample


---

### SNMP plugin

Example:

Timer
|
|
SNMP GET
|
|
OID translation
|
|
CounterSample


---

# 3. Monitoring Data Model

Monitoring is divided into three categories.

## 3.1 Topology Status

Represents generic network state.

Examples:

- device reachable/unreachable.
- interface state.
- link state.
- bandwidth.
- errors.

Source:

topology_status.proto

Examples:

Device lost
Link down
Link degradation
Port errors


---

## 3.2 Feature Status

Represents TSN feature runtime state.

Examples:

- STP state.
- Qbv schedule execution.
- Qav credit state.
- PSFP filtering.
- FRER recovery.

Sources:

stp_status.proto
qbv_status.proto
qav_status.proto
psfp_status.proto
frer_status.proto

Examples:

Gate synchronization failure

STP topology change

PSFP excessive drops

FRER recovery failure

---

## 3.3 System Status

Represents device resource state.

Examples:

- CPU utilization.
- memory usage.
- buffer occupancy.
- queue congestion.
- resource fragmentation.
- synchronization health.

---

# 4. Monitoring Pipeline

## 4.1 Counter Collection

Counters are defined in:

monitoring.proto

Example:

Counter
{
id:
name:
path:
polling interval:
}

The collector schedules collection according to each counter interval.

Different counters may have different polling periods.

Example:

CPU counter        5 seconds

Interface errors   1 second

Queue occupancy    100 ms

---

## 4.2 Counter Samples

Raw collected data is converted into:

```go
type CounterSample struct {

    CounterID string

    NodeID string

    Timestamp time.Time

    Value float64
}
````

The monitoring core does not know the origin.

A sample could come from:

gNMI
NETCONF
SNMP
internal agent

---

# 5. Metric Engine

The metric engine consumes counter samples and produces metrics.

Example:

Input:
rx_bytes(t1)
rx_bytes(t2)

Calculation:
throughput =
(bytes(t2)-bytes(t1))/(t2-t1)

Output:

```go
type MetricResult struct {

    Name string

    NodeID string

    Value float64

    Timestamp time.Time
}
```

Example:
{
 name: "port1 throughput",
 value: 900Mbps
}


---

# 6. Decision Engine

The decision engine compares metrics against policies.

Example:

Metric:
queue occupancy

Policy:
WARNING > 70%
CRITICAL > 90%


Result:
Alert
 |
 |
 CRITICAL congestion detected
 |
 |
 rollback requested

The decision engine does not execute rollback directly.

It generates an action request.

---

# 7. Rollback Integration

The monitoring service communicates with the configuration service.

Example flow:

Monitoring Engine
      |
      |
Critical failure
      |
      v
Rollback request
      |
      v
Configuration Service
      |
      v
Mapping Engine
      |
      v
Transaction.Rollback()

---

# 8. Go Package Structure

Suggested layout:

monitoring_service/

├── cmd/
│   └── monitor/
│       └── main.go
│
├── pkg/
│
│   ├── service/
│   │   ├── server.go
│   │   └── lifecycle.go
│   │
│   ├── collector/
│   │   ├── collector.go
│   │   └── scheduler.go
│   │
│   ├── engine/
│   │   ├── engine.go
│   │   ├── metrics.go
│   │   └── decision.go
│   │
│   ├── plugins/
│   │
│   │   ├── plugin.go
│   │   │
│   │   ├── gnmi/
│   │   │   └── gnmi.go
│   │   │
│   │   ├── netconf/
│   │   │   └── netconf.go
│   │   │
│   │   └── snmp/
    │       └── snmp.go
    │
    ├── model/
    │   └── monitoring.go
    │
    └── structures/
        └── monitoring/
            ├── monitoring.proto
            └── monitoring_data.proto
            └── monitoring_events.proto

---

# 9. Core Go Interfaces

## Protocol Plugin Interface

```go
type MonitoringPlugin interface {

    Name() string

    Protocol() string

    // Check if plugin supports this counter
    Supports(counter monitoring.Counter) bool

    // Start collection
    Start(counter monitoring.Counter,
          callback func(CounterSample)) error

    // Stop collection
    Stop(counterID string) error
}
```

---

## Collector Interface

```go
type Collector interface {

    RegisterPlugin(plugin MonitoringPlugin)

    AddCounter(counter monitoring.Counter) error

    RemoveCounter(counterID string) error

    Samples() <-chan CounterSample
}
```

---

## Counter Sample

```go
type CounterSample struct {

    CounterID string

    NodeID string

    Timestamp time.Time

    Value float64
}
```

---

## Metric Engine Interface

```go
type MetricEngine interface {

    AddMetric(metric monitoring.Metric) error

    Process(sample CounterSample)

    Results() <-chan MetricResult
}
```

---

## Metric Result

```go
type MetricResult struct {

    MetricName string

    NodeID string

    Value float64

    Timestamp time.Time
}
```

---

## Decision Engine Interface

```go
type DecisionEngine interface {

    Evaluate(result MetricResult)

    Alerts() <-chan Alert
}
```

---

## Alert Model

```go
type Alert struct {

    Metric string

    NodeID string

    Severity monitoring.Severity

    Description string

    Action AlertAction
}


type AlertAction int


const (

    ActionNone AlertAction = iota

    ActionNotify

    ActionRollback

)
```

---

# 10. Runtime Flow

Normal operation:

Device
 |
 |
Protocol Plugin
 |
 |
CounterSample
 |
 |
Collector
 |
 |
Metric Engine
 |
 |
MetricResult
 |
 |
Decision Engine
 |
 |
Alert

Failure example:

Queue occupancy > 90%
        |
        v
Congestion metric
        |
        v
CRITICAL alert
        |
        v
Rollback request
        |
        v
Configuration Service
        |
        v
Transaction rollback

---

# 11. Summary

The monitoring service follows the same architectural philosophy as the configuration service:

* Plugins contain protocol/device knowledge.
* Core engines operate on abstract models.
* Metrics and decisions are independent of data acquisition.
* Runtime status models describe the network state.
* Monitoring detects failures and requests corrective actions but does not directly modify configuration.

This separation allows adding new protocols, vendors, counters, or TSN features without modifying the monitoring core.

---

# 12. Engineering Contracts and Runtime Behavior

The previous sections describe the architectural separation. This section defines the execution contracts between components.

The dependency direction is strictly:

Protocol Plugins  
→ Collector  
→ Counter Samples  
→ Metric Engine  
→ Metric Results  
→ Decision Engine  
→ Actions

Upper layers must never depend on lower-layer implementation details.

---

# 12.1 Counter Contract

A counter represents one observable value collected from a device or system.

The counter definition contains the logical identity of the value and the information required by the collector.

The protocol plugin interprets the device-specific path.

Example:

```go
type Counter struct {

    ID string

    Name string

    Type CounterType

    NodeID string

    Path string

    PollInterval time.Duration
}
````

The same counter definition can be collected through different protocols.

Example:

queue_drop_packets

gNMI:
 /interfaces/interface/queues/dropped

NETCONF:
 /interfaces/interface/state/queue/drop

SNMP:
 OID 1.3.x.x.x

The collector and metric engine do not know which representation was used.

---

# 12.2 Counter Sample Contract

A counter sample represents an observation at a specific point in time.

```go
type CounterSample struct {

    CounterID string

    NodeID string

    Timestamp time.Time

    Value float64

    Quality SampleQuality
}
```

The quality field is required because monitoring must distinguish:

* a valid zero value.
* a missing value.
* a communication failure.
* an invalid measurement.

Example:

Interface errors = 0
(valid)

Device unreachable
(unavailable)


Possible values:

```go
type SampleQuality int

const (

    SampleValid SampleQuality = iota

    SampleUnavailable

    SampleTimeout

    SampleInvalid
)
```

---

# 12.3 Plugin Runtime Model

A plugin is responsible only for device communication.

The plugin does not:

* calculate metrics.
* evaluate thresholds.
* generate alarms.
* trigger rollback.

Its responsibility ends when it produces normalized samples.

Example:

Device
   |
   |
Protocol Plugin
   |
   |
CounterSample

---

# 12.4 Collector Lifecycle

The collector owns the runtime scheduling.

For every active counter, the collector maintains a collection worker.

Example:

Counter A
interval: 5 seconds

    |
    |
 goroutine A


Counter B
interval: 100 ms

    |
    |
 goroutine B


The collector manages:

* starting collection.
* stopping collection.
* updating monitored counters.
* forwarding samples.

Plugins should not create uncontrolled background workers.

---

# 12.5 Metric Engine Behavior

The metric engine is stateful.

Many metrics require historical samples.

Examples:

## Throughput

Requires two samples:

rx_bytes(t1)

rx_bytes(t2)


Calculation:

(t2-t1 bytes difference)/(time difference)


---

## Moving average

Requires a window:

samples:

100
110
120
130

window:

4 samples

Result:

average = 115

---

## Rate based metrics

Examples:

* packet loss rate.
* error rate.
* discard rate.
* throughput.
* jitter.

The metric engine maintains the required history internally.

---

# 12.6 Metric Result Contract

A metric result represents a calculated observation.

It is not an alarm.

```go
type MetricResult struct {

    MetricID string

    NodeID string

    Value float64

    Timestamp time.Time
}
```

Example:


Metric:

queue_utilization


Result:

value = 92%

timestamp = 10:30:01


---

# 12.7 Decision Engine Behavior

The decision engine converts metric results into operational events.

The decision engine owns:

* threshold comparison.
* hysteresis handling.
* alarm generation.
* action selection.

Example:

Metric:


queue utilization


Policy:


WARNING  > 70%

CRITICAL > 90%


Result:


MetricResult

      |

Decision Engine

      |

Critical Alert


---

# 12.8 Alert Contract

An alert represents a detected abnormal condition.

```go
type Alert struct {

    MetricID string

    NodeID string

    Severity monitoring.Severity

    Description string

    Action AlertAction
}
```

Possible actions:

```go
type AlertAction int

const (

    ActionNone AlertAction = iota

    ActionNotify

    ActionRollback

    ActionReconfigure

)
```

The monitoring service does not directly modify network configuration.

Actions are forwarded to the responsible service.

---

# 13. Mapping Runtime Status Models to Monitoring

The monitoring service consumes three categories of runtime information.

## 13.1 Topology Runtime Status

Source:

topology_status.proto

Used for:

* device availability.
* link state.
* port state.
* bandwidth.
* interface errors.

Examples:

Device lost

Link down

Port fault

---

## 13.2 TSN Feature Runtime Status

Sources:

stp_status.proto

qbv_status.proto

qav_status.proto

psfp_status.proto

frer_status.proto

Used for:

* STP convergence.
* schedule execution.
* gate synchronization.
* shaping behavior.
* filtering violations.
* frame recovery.

Examples:

Qbv synchronization failure

PSFP excessive drops

FRER recovery degradation

STP topology change

---

## 13.3 System Runtime Status

Additional runtime information should be modeled separately.

Examples:

CPU utilization

Memory pressure

Queue occupancy

Buffer overflow

Clock synchronization

Resource allocation state

These values are treated the same way as other counters and can participate in metric calculations.

---

# 14. End-to-End Example

## Detecting TSN congestion and triggering rollback

### 1. Counter definition

Counter:

ID:
queue_drop_packets


Path:

device queue drop counter


Interval:

100ms

---

### 2. Collection

The selected plugin retrieves the value.

Example:

NETCONF plugin

      |

NETCONF get

      |

CounterSample

Sample:

```go
{
 CounterID: "queue_drop_packets",

 NodeID: "switch1",

 Value: 1200
}
```

---

### 3. Metric calculation

Metric:

```
queue_drop_rate
```

Input:

```
drop(t1)

drop(t2)
```

Output:

```
MetricResult

value:
500 packets/sec
```

---

### 4. Decision

Policy:

```
queue_drop_rate > 400 packets/sec

severity:
CRITICAL

action:
ROLLBACK
```

---

### 5. Action execution

The monitoring service sends:

```
Rollback request
```

to the configuration service.

The configuration service executes:

```
MappingEngine.Rollback()

        |

ConfigurationTransaction.Rollback()

        |

Backend.Rollback()

```

---

# 15. Implementation Order

Recommended implementation sequence:

## Phase 1 - Data model

Implement(done):

* monitoring.proto
* monitoring_data.proto
* monitoring_events.proto

## Phase 2 - Plugin framework

Implement:

* plugin interface.
* collector.
* scheduling engine.

## Phase 3 - First protocol plugins

Start with:

1. NETCONF

## Phase 4 - Metric engine

Implement common metrics:

* throughput.
* utilization.
* error rate.
* loss rate.
* delay.

## Phase 5 - Decision engine

Implement:

* thresholds.
* severity.
* hysteresis.
* alert generation.

## Phase 6 - Recovery integration

Implement:

* rollback requests.
* reconfiguration requests.
* notification events.

```
```
