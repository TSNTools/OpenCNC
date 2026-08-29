PE — Path Entity
it is a guarded domain interface that owns everything about path handling, but does not decide the optimal path.

Responsibilities:
    Determine stream endpoints.
    Validate that endpoints/topology are usable.
    Represent path requirements/constraints.
    Maintain/update StreamConfiguration.path.
    Calculate candidate/feasible paths.
    Detect path conflicts or invalid paths.
    Provide path information to the optimizer.
    Apply the optimizer's selected path to the model.

workflow:
    ForwardingPlaneModel
        │
        ▼
       PE
        │
        │ BuildRoutingInput()
        ▼
   Internal Optimizer
        │
        │ selected paths
        ▼
       PE
        │
        │ ApplyPath()
        ▼
StreamConfiguration.Path



Internal Optimizer
Owns the decision/optimization problem:
    candidate paths
        +
    network/resource constraints
        +
    optimization objective
        ↓
    best path(s)














============================================
================== DRAFT ===================
============================================
Area	        | What it contains
----------------------------------------------------------------------------------------
Topology	    | Bridges, end stations, ports, links, speeds, capabilities, connectivity
Configuration	| VLANs, QoS, PSFP, FRER, schedules, reservations, etc.
Streams	        | Talkers, listeners, traffic requirements, paths, stream properties
Resources	    | Available/allocated bandwidth, queues, VLAN IDs, schedule capacity, etc.
Capabilities	| What each device/port/network element supports
CNC metadata	| Model/version/revision, timestamps, perhaps configuration domain


                 NetworkModel
                      │
        ┌─────────────┼─────────────┐
        │             │             │
     Topology     Configuration    Streams
        │             │             │
        └─────────────┼─────────────┘
                      │
                derived graph
                      │
                  Optimizer



Current NetworkModel
        │
        │ clone
        ▼
Desired NetworkModel
        │
        │ edit
        ▼
   Diff / Validate
        │
        ▼
  MigrationPlan
        │
        ▼
       KV


Lifecycle

    NewForwardingPlaneModel()
    Clone()

Topology navigation

    Node(id string)
    Port(nodeID, portID string)
    Link(id string)

Topology mutation

    AddNode(node)
    RemoveNode(id)
    AddPort(nodeID, port)
    RemovePort(nodeID, portID)
    AddLink(link)
    RemoveLink(id)

Configuration navigation

    NodeConfig(nodeID)
    PortConfig(nodeID, portID)
    LinkConfig(linkID)

Configuration mutation

    SetNodeConfig(config)
    RemoveNodeConfig(nodeID)
    SetPortConfig(nodeID, config)
    RemovePortConfig(nodeID, portID)
    SetLinkConfig(config)
    RemoveLinkConfig(linkID)

Stream navigation

    Stream(id)
    StreamConfiguration(id)

Stream mutation

    AddStream(stream)
    RemoveStream(id)
    SetStreamConfiguration(id, config)
    RemoveStreamConfiguration(id)