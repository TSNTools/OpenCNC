This repository is a microservice of the tool OpenCNC. OpenCNC is a unit that automatically manages the elements of a Time Sensitive Network (TSN) according to the specification presented in the IEEE 801.2Q standard and its relevant amendments. For more information about OpenCNC, Links to the rest of its microservices as well the deployment script, please visit the repository [OpenCNC_demo: https://git.cs.kau.se/hamzchah/opencnc_demo.](https://git.cs.kau.se/hamzchah/opencnc_demo)

# Brief introduction of OpenCNC

OpenCNC is an implementation of the Control plane for TSN networks that orchestrates and manages the operations in the forwarding plane. The term management entails more than configuring
the network. It includes:
* Calculating the configuration of the TSN switches and the end stations.
* Loading and retrieving the configuration to/from the TSN switches automatically depending on the current state and target state of the network.
* Discovering the switches and retrieving their capabilities.
* Maintaining a global view of the topology and the available resources.
* Handling network events in order to maintain a desired state.
* Monitoring of performance key values for network analysis.

OpenCNC is designed as a micro-service based system. In total, it has the following micro-services, each micro-service is implemented in a different repository to allow easier use, extension and maintenance of the tool:
* Main service
* TSN service
* policing service (not implemented yet)
* Monitoring interface
* Monitor service (current repository)
* Topology service
* Config service
* gnmi to Netconf adapter
* Onos plugin (not implemented yet)

an overview of the OpenCNC target architecture is presented in the following Figure:
<img src="./images/arch.png">

# Monitor service

The monitor service makes sure that the configuration applied to the forwarding plane is acting as it should. The main functionalities of this service are:
* QoS assurance : collect statistics, compute metrics
* Analyze synchronization precision
* Streaming telemetry for malfunctioning disclose
* gets the operational state of the devices for anomaly detection
* Network events classification (form southbound)
* Triggers the fallback mechanism

## Reference design of the monitor service
The target design of the monitor service is depicted in the following figure: <img src="./images/monitor-service.png">

The core component of the monitor service is the Device monitor, it is the module that is instantiated for every device that is monitored. The module takes in a gNMI request defining, among other, the set of counters to watch as well as the frequency of pulling them, in the form of a structure. The Device monitor module then loops through the requests provided and starts a new go-routine for every interval which can contain multiple counters. These new go-routines send their requests at the given interval they have been provided. The go-routines will also be responsive to a shutdown command. If monitoring is updated, all go-routines assigned to the different intervals are stopped and new ones with the updated monitoring are started.

In the current design we define two modes of monitoring the regular mode which is the default. During regular monitoring the device monitor pulls less counters and with bigger inter-pulling interval. The second mode is the deep mode in which more counters are pulled and relatively more frequent. The deep monitoring is activated when the network is in transitional state such as after applying a new configuration. The deep monitoring is implemented to make sure that the network reaches a stable state without issues and to detect any malfunctioning in the forwarding plane.

The configuration of the monitoring is stored in the k/v store. It is up to the Request builder to get it from the store interface and use it to build the required gNMI requests that are sent by the device monitor to the switches. If the config specifies any other protocol than gNMI it requests an adapter for the defined protocol from the store interface and configure the gNMI requests to use the address of the adapter to reach the switch.

The storage interface provides two public functions: GetConfig and GetAdapter. Both will go to a store and get the monitor config or protocol adapter.

The main manager of the monitoring process is the config manager. It handles all the requests coming from main service through the northbound gNMI interface. Currently, The options provided by the config manager and available through the northbound gNMI interface are Start, Stop, and Update. These commands are used to manage monitoring.

Monitor service also provides the possibility to subscribe to a set of counters. The subscription manager manages the subscriptions. A shared list of subscriptions is kept in monitor-service for all go-routines. The list is not currently providing any consistency as it does not have any mutex locks or similar functionality implemented. The subscription manager provides two functions, one for registering and unregistering subscribers, and one for pushing data to potential subscribers.

One of the subscribers is the diagnostics entity which combine row counters' data to produce metrics and detect network events. The processed data is then stored, optionally, in the k/v store. The diagnostics entity also sends alerts to the config service(i.e. th network provisioning entity - NPE) and to the main service. The alert sent to the config service initiates a fallback mechanism that brings the network to the last stable state in case of malfunctioning detected and the alert sent to the main service triggers a reconfiguration process if needed.

## Implementation state
the following sequence diagram summarizes the current state of implementation. Only the main functions are presented.

<img src="./images/monitor_sequenceDiagram.svg">.

- Northbound: Starts two servers on ports 11161 (insecure) and 10161 (secure), both servers provide gNMI get, set, and subscribe requests. However, only set and subscribe are implemented.
The diagnostics entity is not implemented, only one of its sub-modules (i.e. dataProcessingManager in the sequence diagram) is created and it is still empty.

- Storage: both functions provided by the storage will go to a store in Atomix and get the config or adapter using predefined URNs. The functions will then unmarshal the byte slices into protobuf messages, which are then sent back to the caller of the functions.
