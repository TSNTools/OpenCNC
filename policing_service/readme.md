# OpenCNC Policing Service

## 1. Purpose

The Policing service defines which network-level functionalities should be activated and which constraints must be satisfied for a given network profile or application type.

It operates at the **policy level**, separating desired network behaviour from the detailed configuration required to implement it.

The service allows OpenCNC to support different application requirements and network profiles by activating and combining policies without duplicating the underlying TSN configuration logic.

## 2. Separation from TSN Configuration

The Policing service specifies **what the network should achieve**, while the TSN service determines **how to configure the network to achieve it**.

The Policing service defines policy selection, parameters, and requirements. Configuration, resource allocation, scheduling, and device-level enforcement remain with the appropriate services.

Policy requirements must be evaluated against available network capabilities and resources. If requirements cannot be satisfied, the system should report the conflict or infeasibility.

## 3. Policy Categories

### 3.1 Resource Policy

Defines how network resources are allocated, isolated, and protected.

**Isolation**
- Define virtual network domains and their separation requirements.
- Assign application types or streams to virtual domains.
- Specify resource budgets for each domain, such as bandwidth and transmission capacity.

**Traffic Protection**
- Set maximum link or network utilization.
- Define reserved resources and amount of available resource headroom.
- Specify admission constraints to prevent new streams from violating existing resource commitments.

### 3.2 Reliability Policy

Defines the protection, monitoring, and recovery behaviour required for different application types.

**Monitoring**
- Enable or disable monitoring for selected network elements or application types.
- Specify which metrics to monitor, such as link status.
- Configure monitoring intervals, thresholds, and notification requirements.

**Redundancy**
- Specify which application types require redundant forwarding.
- Define the required number of paths.
- Specify path-independence requirements, such as link-disjoint or node-disjoint paths.

**Failure Management**
- Define maximum allowed recovery times.
- Specify failure conditions that trigger alerts or recovery actions.
- Define severity levels, such as green (normal), yellow (degraded or warning), and red (critical).
- Associate severity levels with appropriate notifications and response actions.

### 3.3 timing Policy (Sync Tree)

Defines the network's synchronization requirements and the devices that must participate in synchronization.

- Specify which devices or application domains require synchronization.
- Define constraints on master-clock selection and synchronization relationships.
- Specify synchronization requirements where needed.

The Sync Tree functionality determines the required synchronization structure, while the relevant configuration mechanisms configure PTP roles and device settings.

### 3.4 Security Policy

Defines which devices, applications, and virtual domains are allowed to communicate and which access restrictions must be enforced.

- Define communication permissions between application types and virtual domains.
- Specify whether devices must be authorized before joining the network.
- Define how unauthorized streams or communication attempts should be handled.
- Specify whether additional traffic isolation or access-control mechanisms must be activated.

The Policing service defines the security requirements, while the appropriate network configuration or security mechanisms enforce them.

## 4. Policy Application

Policies can be selected and parameterized according to a network profile or application type. Multiple policies may apply simultaneously. For example, an industrial-control profile may require isolated resources, redundant forwarding, fast failure recovery, and PTP synchronization.

The mechanisms in this service ensure that simultaneously active policies are compatible and can be satisfied:

- Detect conflicts between active policies.
- Check whether policy requirements can be delivered by the network.
- Identify resource constraints that prevent policies from being fulfilled.
- Define which policies take precedence when requirements conflict.
- Report unsatisfied requirements and propose alternatives where possible.

The mechanisms in this service define how policies are created, activated, updated, deactivated, and removed:

- Support creating, modifying, and deleting policies.
- Enable or disable policies for specific network profiles or application types.
- Apply policy changes while maintaining consistency with active network configurations.
- Track policy status, including pending, active, failed, and deactivated.
- Handle failed policy activation and configuration updates.
- Ensure that deactivating a policy does not unintentionally violate requirements imposed by other active policies.

The Policing service provides a policy-level mechanism for adapting OpenCNC to different network profiles while reusing the existing TSN configuration infrastructure.



Next steps:
* define the service components and responsibilities
* define the rpc calls and workflows of the service
* define the data model and generate the proto files that governs the behaviour of the service
* implementation and testing