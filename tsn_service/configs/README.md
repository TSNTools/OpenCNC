# Default switch schedules

`default-schedule.yaml` supports overrides selected by a switch's topology
`managementInfo.ip_address`. The name is not used to select an override.
The mapping is **management IP → port ID → gating cycle and traffic classes**.
Each port has its own gate list; the same port ID on another switch can have
entirely different settings. No protobuf changes are needed: the generator
fills the existing `NodeConfig.PortConfigs[].Gcl` fields.

The supplied settings preserve the Relyum schedule at `192.168.4.64`.
At `192.168.4.65`, P1–P4 have example cycles of 2, 3, 4, and 5 ms respectively,
with separate traffic allocations. Adjust these examples to your requirements.

```yaml
switches:
  - management-ip: "192.168.4.64"
    ports:
      - port-id: P1
        gating-cycle: 1 # milliseconds
        traffic-classes:
          - name: Timecritical
            assigned-portion: 70
          - name: best-effort
            assigned-portion: 30
      - port-id: P2
        gating-cycle: 2
        traffic-classes:
          - name: cyclic-sync
            assigned-portion: 50
          - name: best-effort
            assigned-portion: 50
  - management-ip: "192.168.4.65"
    ports:
      - port-id: P1
        gating-cycle: 3
        traffic-classes:
          - name: network-control
            assigned-portion: 20
          - name: best-effort
            assigned-portion: 80
      - port-id: P2
        gating-cycle: 4
        traffic-classes:
          - name: alarms-events
            assigned-portion: 40
          - name: best-effort
            assigned-portion: 60
```

A switch entry can define either a `ports` list or a single schedule for all
of that switch's topology ports. An override replaces the shared schedule
entirely; unlisted ports do not inherit shared settings. Switches without an
IP override use the existing top-level `ports` or `traffic-classes` settings.
If no fallback is defined, unmatched devices receive no schedule.
Port lists take precedence over global traffic classes, as before.

Percentages must total 100 for every schedule. Cycles must be positive integer
milliseconds. IPv4 and IPv6 literals are supported; duplicate IP overrides
are rejected. Overrides for devices not currently in the topology are allowed.

Topology node `name` values must still be unique: storage, credentials, links,
and configuration deployment use them as IDs. Repeated hardware/display names
do not change this requirement. Schedule generation rejects duplicate node IDs
rather than risk deploying the wrong configuration.

The default schedule is rebuilt for each optimization request. Set
`DEFAULT_SCHEDULE_PATH` in TSN-service to load an alternative YAML file. An
explicit path must exist and be readable. For an image-bundled file, rebuild
and redeploy TSN-service after editing it.
