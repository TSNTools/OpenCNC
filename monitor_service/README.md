````markdown
# OpenCNC Monitoring Service

## 1. Purpose

The OpenCNC Monitoring Service observes the runtime state of managed network resources and detects conditions that deviate from configured expectations.

The service is responsible for:

1. Collecting runtime counter values from devices and systems.
2. Feeding normalized samples into metric meters.
3. Calculating metrics from the latest available counter and metric data.
4. Respecting independently configured polling and evaluation intervals.
5. Evaluating metric thresholds.
6. Generating monitoring events.
7. Delivering generated events to the component that owns the operational response.

The monitoring service is designed to keep protocol-specific device communication separate from metric calculation and monitoring policy.

---

## 2. Architecture

The monitoring service is organized around four main responsibilities:

```text
                  Device / System
                        |
                        v
                +---------------+
                |   Collector   |
                |---------------|
                | Protocol I/O  |
                +-------+-------+
                        |
                        | DataSample
                        v
              +----------------------+
              |   ResourceMonitor    |
              |----------------------|
              | Scheduler             |
              | Collection            |
              | Measurement            |
              | Dependency ordering   |
              | Threshold evaluation  |
              +----------+-----------+
                         |
                         | MonitoringEvent
                         v
                  +-------------+
                  |    Engine   |
                  |-------------|
                  | Event handler|
                  | Operational  |
                  | response     |
                  +-------------+
````

The responsibilities are intentionally separated:

| Component       | Responsibility                                          |
| --------------- | ------------------------------------------------------- |
| Collector       | Communicates with devices and collects raw counters     |
| Meter           | Calculates one configured metric                        |
| ResourceMonitor | Schedules collection/measurement and coordinates meters |
| Engine          | Owns monitors and handles monitoring events             |

---

## 3. Core design principles

### Collector

> "What did the device report?"

A collector knows how to communicate with a particular protocol or device.

It converts protocol-specific responses into normalized `DataSample` objects.

### Meter

> "What does this data mean?"

A meter calculates one metric from its input samples.

A meter owns:

* its metric definition
* input history
* calculation logic
* readiness of its evaluation window
* metric dependencies

A meter does **not** own a goroutine or a scheduler.

### ResourceMonitor

> "When should data be collected and when should metrics be evaluated?"

A `ResourceMonitor` represents the monitoring of one resource.

It owns:

* collectors
* meters
* scheduling
* collection
* metric evaluation
* dependency ordering
* threshold evaluation
* event generation

### Engine

> "What should happen when a monitoring event occurs?"

The engine creates and manages resource monitors and provides the event handler used by the monitors.

The monitor does not perform reconfiguration, rollback, or other operational actions itself.

---

# 4. Runtime architecture

The runtime pipeline is:

```text
Collector
    |
    | raw DataSample
    v
ResourceMonitor
    |
    | feed samples
    v
Meter A
    |
    | metric DataSample
    v
Meter B
    |
    | metric DataSample
    v
Meter C
    |
    v
Threshold evaluation
    |
    v
MonitoringEvent
    |
    v
EventHandler
    |
    v
Engine / operational response
```

For example:

```text
counter-a ───────┐
                 |
counter-b ───────┼──> metric-A
                 |       |
counter-c ───────┘       |
                         v
                    metric-B
                         |
                         v
                    metric-C
                         |
                         v
                    threshold
                         |
                         v
                    MonitoringEvent
```

A metric can therefore consume either:

* raw counter samples, or
* results produced by other meters.

---

# 5. ResourceMonitor

`ResourceMonitor` is the central runtime component.

A monitor represents one monitored resource identified by a `ResourceKey`.

Conceptually:

```text
ResourceMonitor
|
+-- ResourceKey
|
+-- Collectors
|    +-- NETCONF
|    +-- gNMI
|    +-- SNMP
|    +-- ...
|
+-- Meters
|    +-- throughput
|    +-- utilization
|    +-- loss-rate
|    +-- queue-occupancy
|    +-- ...
|
+-- Scheduler
|
+-- EventHandler
```

A monitor exposes the following main operations:

```go
type Monitor interface {
    Resource() *monitoring.ResourceKey
    Start() error
    Stop()
    AddCollector(collector collectors.Collector) error
    AddMeter(meter meters.Meter) error
}
```

The monitor also performs monitoring cycles internally according to its scheduler.

---

# 6. Scheduling model

Collection and metric evaluation have independent frequencies.

A counter has a configured polling interval:

```text
Counter.PollIntervalMs
```

A metric has a configured evaluation interval:

```text
Metric.EvaluationIntervalMs
```

These intervals are intentionally independent.

For example:

```text
Counter A     1 second
Counter B     1 second
Counter C     5 seconds

Metric A      1 second
Metric B      5 seconds
Metric C      10 seconds
```

The monitor groups equal intervals together.

The scheduler therefore creates:

```text
1s ticker
  ├── Counter A
  ├── Counter B
  └── Metric A

5s ticker
  ├── Counter C
  └── Metric B

10s ticker
  └── Metric C
```

There is only **one scheduler goroutine** managing all schedule groups.

The implementation uses dynamic channel selection so the scheduler can wait on all ticker channels and the monitor's stop channel.

---

# 7. Collection and measurement are independent

Collection and metric evaluation are deliberately not tied to the same ticker.

For example:

```text
Counter polling:
    1s -> collect
    1s -> collect
    1s -> collect
    1s -> collect

Metric evaluation:
    5s -> evaluate
```

The metric uses the latest samples available in its meter state.

This means that a metric does not need a new counter collection immediately before every evaluation.

The architecture is therefore:

```text
              +----------------+
              | Counter poll   |
              +-------+--------+
                      |
                      v
                 meter state
                      |
                      |
              +-------+--------+
              | Metric eval    |
              +----------------+
```

This also allows different counters to have different polling frequencies.

---

# 8. Scheduler execution ordering

When a ticker fires for an interval containing both counters and metrics, the monitor always executes:

```text
1. Collection
2. Measurement
```

For example:

```text
5-second tick
     |
     v
collect counters
     |
     v
feed counter samples
     |
     v
evaluate metrics
     |
     v
feed metric results
     |
     v
evaluate thresholds
     |
     v
generate events
```

This ordering is important because a metric evaluation should see the newest samples collected during the same scheduler tick.

---

# 9. Collection

Collection is handled by:

```text
collectAndFeed()
```

The function:

1. Determines the counters due for the current interval.
2. Builds counter requests.
3. Invokes the configured collectors.
4. Receives normalized `DataSample` values.
5. Feeds those samples into the registered meters.

Collection does not evaluate metrics.

The collector remains responsible only for protocol communication.

---

# 10. Collector interface

The collector abstraction is:

```go
type Collector interface {

    // Protocol implemented by this collector.
    Protocol() string

    // Discover which OpenCNC counters are available on the target.
    SupportedCounters() ([]*monitoring.Counter, error)

    // Read a subset (or all) of the supported counters.
    Collect(
        counters []*monitoring.Counter,
    ) ([]*monitoring.DataSample, error)
}
```

A collector is responsible for:

* protocol communication
* device requests
* response parsing
* converting protocol-specific values into normalized samples

A collector does not:

* calculate metrics
* evaluate thresholds
* generate monitoring events
* perform reconfiguration

---

# 11. NETCONF collector

The current implementation includes a NETCONF-oriented collector.

The NETCONF collector is responsible for:

1. Establishing a NETCONF session.
2. Discovering the supported counters.
3. Building protocol-specific requests.
4. Reading device responses.
5. Parsing returned values.
6. Converting them into normalized `DataSample` objects.

The rest of the monitoring system does not need to know that NETCONF was used.

The dependency direction is:

```text
NETCONF
   |
   v
NETCONF Collector
   |
   v
DataSample
   |
   v
ResourceMonitor
```

---

# 12. Meter model

Meters are responsible for calculating metrics.

The current meter interface is:

```go
type Meter interface {

    // Name returns the configured metric name.
    Name() string

    // Metric returns the configured metric.
    Metric() *monitoring.Metric

    // Type returns the metric type implemented by this meter.
    Type() monitoring.MetricType

    // UsesInput reports whether the meter uses the specified input.
    UsesInput(inputID string) bool

    // AddSample provides a normalized sample to the meter.
    AddSample(sample *monitoring.DataSample) error

    // Ready reports whether enough samples are available.
    Ready() bool

    // Evaluate calculates the metric.
    Evaluate() (*monitoring.DataSample, error)

    // Reset clears the meter's internal state.
    Reset()
}
```

The monitor intentionally does not know the mathematical implementation of a meter.

It only knows how to:

```text
feed sample
    |
    v
meter
    |
    v
evaluate
    |
    v
metric result
```

---

# 13. Meter state

Meters may maintain internal state such as:

* previous counter values
* timestamps
* rolling windows
* accumulated samples
* previous metric values
* calculation-specific state

For example, a rate meter may need:

```text
previous counter value
previous timestamp
        |
        v
current counter value
current timestamp
        |
        v
calculated rate
```

The monitor does not manage this state.

The meter owns it.

---

# 14. Metric dependencies

Metrics may depend on other metrics.

For example:

```text
counter-a ─────┐
               ├──> packet-rate
counter-b ─────┘
                    |
                    v
              congestion
                    |
                    v
                 alert
```

Here:

```text
packet-rate
    ↓
congestion
```

is a metric dependency.

The monitor evaluates meters in dependency order.

A meter is evaluated only when its required metric dependencies have been evaluated for the current measurement cycle.

---

# 15. Metric result propagation

When a meter produces a result, the monitor converts it into a metric `DataSample`.

Conceptually:

```text
Meter.Evaluate()
      |
      v
DataSample
    Id = meter name
    Kind = METRIC
      |
      v
dependent meters
```

The result is immediately fed to all meters that consume that metric.

For example:

```text
Meter A
   |
   | result: metric-A
   v
Meter B
   |
   | result: metric-B
   v
Meter C
```

This allows complex metrics to be built from simpler metrics.

---

# 16. Measurement scheduling

Only metrics whose configured evaluation interval is due are scheduled for evaluation.

For example:

```text
Metric A -> 1s
Metric B -> 5s
Metric C -> 10s
```

At a 5-second tick:

```text
Metric A -> evaluate
Metric B -> evaluate
Metric C -> not evaluated
```

At a 10-second tick:

```text
Metric A -> evaluate
Metric B -> evaluate
Metric C -> evaluate
```

The latest available input values remain available inside the meters between evaluations.

---

# 17. Metric readiness

A meter controls whether it has enough data to calculate its metric.

This is represented by:

```go
Ready() bool
```

For example, a metric requiring a five-sample window may initially report:

```text
Ready() == false
```

until enough samples have been collected.

The monitor currently treats an evaluation cycle with no progress as an error condition:

```text
unresolved dependency,
dependency cycle,
or meter not ready
```

This behavior is intentionally simple for the first implementation.

It can be refined later to distinguish:

* dependency cycles
* unresolved dependencies
* meters that simply need more samples

---

# 18. Threshold evaluation

Threshold evaluation is performed immediately after a metric produces a result.

There is no separate threshold ticker.

The flow is:

```text
Metric evaluation
      |
      v
Metric result
      |
      v
Threshold evaluation
      |
      v
Triggered?
   /       \
 no         yes
 |           |
done       event
```

This means threshold evaluation is naturally associated with metric evaluation.

If a metric is not evaluated, its thresholds are not evaluated.

---

# 19. Threshold model

A metric may contain one or more thresholds.

Supported comparison operators include:

```text
GREATER_THAN
GREATER_OR_EQUAL
LESS_THAN
LESS_OR_EQUAL
EQUAL
NOT_EQUAL
```

A threshold contains the condition and the event definition associated with that condition.

The monitor determines whether:

```text
metric value
      |
      v
threshold condition
      |
      v
triggered
```

The monitor does not decide what operational action should be taken.

---

# 20. Monitoring events

When a threshold is triggered, the monitor creates a `MonitoringEvent`.

The event is cloned from the configured threshold event so that the metric configuration is not modified.

The monitor assigns the monitored resource as the event source.

Conceptually:

```text
Metric
  |
  v
Threshold
  |
  | triggered
  v
MonitoringEvent
  |
  | Source = monitored resource
  v
EventHandler
```

---

# 21. Event handler

The `ResourceMonitor` receives an event handler when it is created.

The handler is defined separately from the monitor implementation.

The monitor's responsibility ends at event generation and dispatch.

It does not directly perform:

* rollback
* reconfiguration
* device changes
* orchestration
* operator notification

Instead:

```text
ResourceMonitor
      |
      v
EventHandler
      |
      v
Engine
      |
      +----> notification
      |
      +----> reconfiguration
      |
      +----> rollback
      |
      +----> other operational response
```

This keeps the monitoring layer independent from the operational control plane.

---

# 22. Engine responsibility

The engine is the higher-level component that manages resource monitors.

The engine is responsible for:

* creating monitors
* configuring monitors
* registering collectors
* registering meters
* starting and stopping monitors
* receiving monitoring events
* deciding how events should be handled

The engine does **not** need to know how a metric is calculated or how a device is queried.

Its relationship with a monitor is:

```text
Engine
  |
  +-- ResourceMonitor A
  |
  +-- ResourceMonitor B
  |
  +-- ResourceMonitor C
```

Each monitor independently manages the monitoring schedule of its resource.

---

# 23. Complete runtime example

Consider the following configuration:

```text
counter:
    interface-bytes
    polling interval = 1 second

metric:
    interface-throughput
    evaluation interval = 5 seconds

threshold:
    throughput < configured minimum
```

The runtime behaves as follows:

```text
Every 1 second:

    Collector
       |
       v
    interface-bytes
       |
       v
    Meter state


Every 5 seconds:

    Meter
       |
       v
    interface-throughput
       |
       v
    Threshold
       |
       +---- not triggered ---> done
       |
       +---- triggered -------> MonitoringEvent
                                  |
                                  v
                              EventHandler
```

The counter is therefore collected five times between normal metric evaluations.

The metric uses the latest meter state when it evaluates.

---

# 24. Complex metric example

Consider:

```text
counter-1 ─────┐
               |
counter-2 ─────┼──> packet-rate
               |
               v
          packet-rate
               |
               +──────────┐
                          |
counter-3 ────────────────┼──> congestion
                          |
                          v
                       threshold
                          |
                          v
                    MonitoringEvent
```

The monitor handles this as:

```text
1. Collect counter-1
2. Collect counter-2
3. Collect counter-3

4. Feed the samples into meters

5. Evaluate packet-rate

6. Feed packet-rate into congestion meter

7. Evaluate congestion

8. Evaluate congestion thresholds

9. Dispatch event if triggered
```

The monitor therefore preserves dependency ordering while still allowing collection and metric evaluation to have independent schedules.

---

# 25. Current package structure

The current repository is organized around the following major components:

```text
monitor_service/
├── main.go
├── cmd/
├── opencnc_counters_catalog/
├── pkg/
│   ├── collectors/
│   │   ├── collector.go
│   │   └── netconf_collector.go
│   │
│   ├── engine/
│   │
│   ├── managementSessions/
│   │
│   ├── meters/
│   │   └── meter.go
│   │
│   └── monitor/
│       ├── monitor.go
│       └── ...
│
├── structures/
│   └── monitoring/
│       ├── monitoring.proto
│       ├── monitoring_data.proto
│       └── monitoring_events.proto
│
└── README.md
```

The exact number of implementation files may grow as additional collectors, meters, and engine functionality are introduced.

---

# 26. Monitoring data model

The monitoring protobuf definitions under:

```text
structures/monitoring/
```

provide the common data model used between the monitoring components.

The important concepts are:

```text
ResourceKey
    |
    v
Counter
    |
    v
DataSample
    |
    v
Metric
    |
    v
Threshold
    |
    v
MonitoringEvent
```

---

# 27. Counter

A `Counter` describes a value that can be collected.

Important properties include:

* identifier
* name
* collection path or access information
* polling interval
* description

The polling interval determines how frequently the monitor schedules collection of that counter.

Conceptually:

```text
Counter
    |
    +-- Id
    +-- PollIntervalMs
    +-- ...
```

---

# 28. DataSample

`DataSample` is the normalized runtime value passed through the monitoring pipeline.

It can represent either:

```text
raw counter data
```

or:

```text
derived metric data
```

The monitor distinguishes metric results by assigning:

```text
Kind = METRIC
```

to the output of a meter.

This allows the same normalized sample mechanism to be used for both raw and derived data.

---

# 29. Metric

A `Metric` describes how a configured meter should operate.

Important properties include:

```text
Name
Type
InputIds
EvaluationIntervalMs
WindowSize
Thresholds
Description
```

`InputIds` may refer to:

* raw counter IDs
* names of other metrics

This allows metric dependency graphs to be represented directly in the configuration.

Example:

```text
Metric A
InputIds:
    counter-1
    counter-2

Metric B
InputIds:
    Metric A
    counter-3
```

---

# 30. Metric dependency graph

The configured metrics form a directed graph.

For example:

```text
counter-a ──┐
            v
         metric-a
            |
            v
         metric-b
            |
            v
         metric-c
```

The monitor evaluates the graph in dependency order.

A dependency cycle is invalid:

```text
metric-a
   |
   v
metric-b
   |
   v
metric-a
```

The monitor detects situations where evaluation cannot make progress and reports an error.

---

# 31. Monitoring events

A `MonitoringEvent` represents a condition detected by monitoring.

An event can contain information such as:

* event identity
* source resource
* event type
* severity
* action information
* description

The event configuration is associated with a metric threshold.

The monitor creates the event when the threshold is triggered and passes it to the configured event handler.

---

# 32. Event and action boundary

The monitoring service detects conditions.

It does not directly execute operational changes.

The boundary is:

```text
Monitoring Service
       |
       | MonitoringEvent
       v
     Engine
       |
       | operational decision
       v
Other OpenCNC services
```

This allows the monitoring service to remain focused on observation and detection while other components own device configuration and operational control.

---

# 33. Lifecycle

A resource monitor has a simple lifecycle:

```text
Created
   |
   v
Configured
   |
   v
Started
   |
   v
Running
   |
   v
Stopped
```

Collectors and meters must be registered before the monitor starts.

Once running, configuration changes are rejected.

This prevents the scheduler from operating against a configuration that is changing concurrently.

---

# 34. Start

Starting a monitor performs the following:

1. Verify that the monitor is not already running.
2. Snapshot the configured collectors and meters.
3. Build the schedule.
4. Create the required ticker groups.
5. Start the scheduler goroutine.

If schedule construction fails, the monitor does not start.

---

# 35. Stop

Stopping a monitor:

1. Marks it as no longer running.
2. Closes the scheduler stop channel.
3. Causes the scheduler goroutine to exit.
4. Stops all schedule tickers.

The monitor can subsequently be started again.

---

# 36. Scheduler example

Suppose the configuration contains:

```text
Counters:

counter-A = 1s
counter-B = 1s
counter-C = 5s

Metrics:

metric-A = 1s
metric-B = 5s
metric-C = 10s
```

The monitor creates:

```text
Schedule Group 1
----------------
interval = 1s

counter-A
counter-B
metric-A


Schedule Group 2
----------------
interval = 5s

counter-C
metric-B


Schedule Group 3
----------------
interval = 10s

metric-C
```

All three groups are consumed by the same scheduler goroutine.

---

# 37. Scheduler execution example

At a 1-second tick:

```text
collect counter-A
collect counter-B

then:

evaluate metric-A
```

At a 5-second tick:

```text
collect counter-C

then:

evaluate metric-B
```

At a 10-second tick:

```text
collect counter-C

then:

evaluate metric-B

then:

evaluate metric-C
```

When metric dependencies exist, the monitor preserves dependency order within the measurement operation.

---

# 38. Why the scheduler belongs to ResourceMonitor

Scheduling is a property of the monitoring configuration of a resource.

Different resources may have completely different:

* counters
* polling intervals
* metrics
* evaluation intervals
* dependency graphs

Therefore each `ResourceMonitor` owns its own scheduler.

The engine manages monitors rather than managing individual metric tickers.

```text
Engine
 |
 +-- Monitor(resource-A)
 |      |
 |      +-- scheduler
 |
 +-- Monitor(resource-B)
        |
        +-- scheduler
```

---

# 39. Error handling

Errors are propagated from the lower layers toward the monitor.

Examples include:

```text
Collector error
    |
    v
collectAndFeed()
    |
    v
scheduler reports error
```

or:

```text
Meter error
    |
    v
measureAndFeed()
    |
    v
scheduler reports error
```

or:

```text
Threshold configuration error
    |
    v
event evaluation
    |
    v
scheduler reports error
```

The scheduler currently provides the location where errors can later be connected to logging or monitoring infrastructure.

---

# 40. Important architectural boundaries

The service follows these dependency rules:

```text
Collector
   |
   v
DataSample
   |
   v
Meter
   |
   v
ResourceMonitor
   |
   v
MonitoringEvent
   |
   v
Engine
```

Lower layers should not depend on higher-level operational logic.

In particular:

### Collectors must not:

* evaluate metrics
* evaluate thresholds
* generate events
* perform actions

### Meters must not:

* communicate with devices
* own monitoring goroutines
* evaluate operational policies
* perform actions

### ResourceMonitor must not:

* perform rollback
* directly reconfigure devices
* own engine-level operational policy

### Engine must not:

* implement protocol-specific collection
* implement metric formulas

---

# 41. Extension model

New functionality should be added by extending the appropriate layer.

### New protocol

Implement a new collector:

```text
pkg/collectors/
    |
    +-- collector.go
    +-- netconf_collector.go
    +-- new_protocol_collector.go
```

### New metric

Implement a new meter:

```text
pkg/meters/
    |
    +-- meter.go
    +-- throughput.go
    +-- utilization.go
    +-- new_metric.go
```

### New operational response

Extend the engine/event handling layer rather than the meter or collector.

```text
MonitoringEvent
      |
      v
EventHandler
      |
      +-- notify
      +-- reconfigure
      +-- rollback
      +-- ...
```

---

# 42. Example end-to-end flow

A complete monitoring flow can be represented as:

```text
                    RESOURCE
                       |
                       v
                ResourceMonitor
                       |
             +---------+---------+
             |                   |
             v                   v
        Scheduler            EventHandler
             |
             v
       Counter ticker
             |
             v
        Collector
             |
             v
         DataSample
             |
             v
          Meter A
             |
             v
       Metric result
             |
             v
          Meter B
             |
             v
       Metric result
             |
             v
        Threshold
             |
       +-----+-----+
       |           |
   not triggered  triggered
       |           |
      done         v
              MonitoringEvent
                    |
                    v
               EventHandler
                    |
                    v
                  Engine
```

---

# 43. Current implementation status

The current monitoring implementation provides the first complete version of the core monitoring runtime.

Implemented architectural building blocks include:

* `Collector` abstraction
* protocol-specific collector support
* `Meter` abstraction
* resource-oriented `ResourceMonitor`
* collector registration
* meter registration
* independent counter polling intervals
* independent metric evaluation intervals
* interval grouping
* one scheduler goroutine per resource monitor
* collection before measurement
* metric dependency ordering
* propagation of metric results between meters
* meter readiness handling
* threshold evaluation
* monitoring event generation
* event handler integration
* monitor lifecycle management

The implementation is intentionally focused on the monitoring runtime. Operational actions remain outside the monitor and are handled through the event handler.

---

# 44. Future improvements

The current architecture leaves room for incremental improvements without changing the core design.

Potential future improvements include:

1. More precise handling of meters that are not yet ready.
2. Explicit detection and reporting of dependency cycles.
3. Scheduler error reporting and logging.
4. Collector discovery and capability validation.
5. Additional protocol collectors.
6. Additional meter implementations.
7. Event state tracking and hysteresis.
8. Event deduplication and suppression.
9. Persistent metric history where required.
10. More sophisticated operational event handling in the engine.

These are extensions of the current architecture rather than changes to its fundamental responsibilities.

---

# 45. Summary

The OpenCNC Monitoring Service separates monitoring into clear layers:

```text
Collector
    ↓
DataSample
    ↓
Meter
    ↓
Metric
    ↓
Threshold
    ↓
MonitoringEvent
    ↓
EventHandler
    ↓
Engine / operational response
```

A `ResourceMonitor` coordinates these components for one monitored resource.

Its scheduler independently manages:

* counter polling intervals
* metric evaluation intervals

Equal intervals share a ticker, while a single scheduler goroutine manages all ticker groups.

Within each scheduler tick, the monitor always preserves the execution order:

```text
Collection
    ↓
Measurement
    ↓
Threshold evaluation
    ↓
Event dispatch
```

Metric dependencies are evaluated in the correct order, and newly calculated metric results are propagated to dependent meters.

The resulting architecture keeps:

* protocol communication inside collectors,
* metric calculation inside meters,
* scheduling and monitoring coordination inside `ResourceMonitor`,
* operational event handling inside the engine.

This provides a simple foundation for adding new protocols, metrics, monitoring policies, and operational responses without coupling the different responsibilities together.

```
```
