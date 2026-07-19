# Config service — Architecture Overview

This project implements a Configuration service for Time-Sensitive Networking (TSN).
It receives a topology-wide Protobuf-based configuration and translates it into device-specific configurations,
pushing them using various southbound protocols such as NETCONF and SNMP.

---

## ✅ Key Concepts

### Config Source: Protobuf
All configuration data is defined using .proto files and stored in a key/value store.
Core types include:
- TopologyConfig
- NodeConfig (contains ManagementInfo and DeviceInfo)
- PortConfig, StreamConfig, etc.
- Feature-specific configs: Qbv, PSFP, Qav, stp, frer, etc.

### Plugins
A **Plugin** is a per-feature, per-protocol implementation that knows how to:
- Map a Protobuf config message to protocol-specific data (e.g., ygot, SNMP varbinds)
- Optionally push the mapped config to a target device

Example:
- `QbvNetconfPlugin`: Handles mapping and applying Qbv config over NETCONF

### Protocol Backends
A **ProtocolBackend** is a group of plugins that share the same southbound protocol.
Examples:
- `NetconfBackend`: All NETCONF plugins (Qbv, PSFP, etc.)
- `SnmpBackend`: All SNMP plugins (Qbv, etc.)

### Mapping Engine
The `MappingEngine` is the top-level orchestrator.
It receives the full configuration and:
- Iterates over the topology
- For each node, selects the appropriate ProtocolBackend based on `ManagementInfo.protocol`
- Delegates mapping and pushing to the right plugin

What it does now
- accepts a full Topology and TopologyConfig
- finds the matching node config by node_id
- finds the matching port config by port_id
- selects the correct backend from ManagementInfo.protocol
- builds a DeviceTarget
- dispatches the config to the backend/plugin for mapping and push
- validate that the topology and config are consistent before applying anything
- resolve global profile context before per-node/port config is applied
- group and apply feature-specific messages in a deterministic order
- report per-node/per-port success and failure clearly
- support dry-run, rollback, and retry behavior
- track applied state so later reconfiguration can be reconciled safely
---

## 📁 Code Structure
/yourproject/
├── common/
│ ├── structures/ # Generated protobuf Go code
│ └── store-wrapper/ # Store access helpers shared by services
│ └── observability/ # kafka and cmd for observability
│
├── config_service/
│ ├── pkg/plugins/ # Per-feature plugins grouped by protocol
│ ├── netconf/
│ │ ├── qbv.go # QbvNetconfPlugin
│ │ └── psfp.go # PsfpNetconfPlugin
│ └── snmp/
│   └── qbv.go # QbvSnmpPlugin
│
├── config_service/pkg/protocolbackends/ # Protocol-level orchestrators 
│ ├── netconf.go # NetconfBackend implementation
│ └── snmp.go # SnmpBackend implementation
│
├── config_service/pkg/engine/ # Top-level config orchestrator
│ └── mappingengine.go # Applies entire TopologyConfig
│
├── config_service/pkg/managementSessions/ # Device runtime metadata and session wrappers
│ └── devicetarget.go # Wrapper for runtime connection info
│
└── config_service/pkg/utils/ # Shared helper utilities
  └── logger.go

## 🔌 Plugin Interface

Each plugin implements:

```go
type Plugin interface {
    Name() string                          // e.g. "qbv-netconf"
    FeatureName() string                   // e.g. "qbv"
    Supports(msg proto.Message) bool      // Accepts a proto message?
    Map(msg proto.Message, model *DeviceModel) (any, error)
    Push(mapped any, target DeviceTarget) error
}

⚙️ ProtocolBackend Interface

type ProtocolBackend interface {
    Name() string
    AddPlugin(p Plugin)
    GetPlugin(feature string) (Plugin, bool)
    MapAndPush(msg proto.Message, model *DeviceModel, target DeviceTarget) error
    SupportedFeatures() []string
}

🧠 DeviceTarget
This struct lives in the managementSessions/ package. It wraps runtime device metadata (session info, IP, credentials, etc.).
It is decoupled from raw Protobuf to allow future extension (e.g., session reuse, secrets injection).

type DeviceTarget struct {
    Info   *ManagementInfo   // From proto
    Secret string            // Runtime-injected credentials
    Logger *log.Logger
    // Optionally: cached session or retry logic
}

✅ Flow Summary
CNC receives TopologyConfig

Iterates over Nodes

For each node:

Reads ManagementProtocol (e.g., NETCONF)

Selects corresponding ProtocolBackend

Dispatches feature configs to corresponding Plugins

Each plugin maps and (optionally) pushes config


🚀 Extending the CNC
To add support for a new protocol: implement a new ProtocolBackend and corresponding Plugins

To support a new feature: implement Plugin(s) for each protocol backend you want

All config flows remain driven by the central Protobuf schema (TopologyConfig)


This structure ensures a clean, modular, and future-proof configuration pipeline.
Each layer is testable in isolation and extensible by design.




















