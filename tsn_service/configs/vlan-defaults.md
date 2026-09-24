# Default VLAN configuration

The fallback optimizer reads `default-vlan.yaml` alongside `default-schedule.yaml`
on startup and on each optimization run. Set `DEFAULT_VLAN_PATH` to use another file.
The legacy `PopulateDefaultVlanConfig` helper is not called by this path.

```yaml
default-vlan:                 # optional shared fallback
  vid: 100
  fid: 1
  tagged: true
switches:
  - management-ip: "192.168.4.64"
    default-vlan:
      vid: 200
      fid: 1
      tagged: true
      ports:
        - port-id: P1
        - port-id: P2
```

Switches match topology management IPs, independently of node names. Each override
replaces the shared settings in full. Without a shared fallback, unlisted switches
receive no VLAN defaults. Entries for switches absent from the topology are allowed.
`vid` is required (1–4094); `fid` defaults to 1 and `tagged` defaults to true.
Omit `ports` to select all topology ports; use `ports: []` to select none.
Port IDs must exist on the matched switch. Invalid IDs, duplicate ports, duplicate
management IPs, and unknown YAML fields cause an error before configuration is stored.

Only selected ports receive PVID, membership and bridge VLAN registration entries.
Qbv scheduling remains independent. This does not create interfaces or attach them
to a bridge. Fallback VLAN settings come directly from YAML, just like the default schedule.
AddStream request VLAN IDs do not overwrite these defaults.

VLAN registration keeps topology port IDs unchanged. At XML generation, the
NETCONF serializer converts `P1` to `<port-ref>1</port-ref>`, `P2` to 2, and so on.
Positive numeric IDs are also supported. This is the deployment's numbering
convention, not a universal rule for device port names. Interface names remain
unchanged and are used for interface settings. No custom topology `port_ref`
field is needed. The YAML loader validates selected topology IDs; invalid or zero numeric
references fail XML preparation. Set `bridge_name` to the device bridge name (`SWITCH` for RELY).

## Per-port transmission and registration

The supplied `default-vlan.yaml` now selects VLAN 10 / FID 1 on 192.168.4.64,
with P1 untagged, P2 tagged, and `registrar-admin-control: normal`.
Port IDs stay P1/P2; the serializer converts them to XML references 1/2.

Each port may override `tagged` and `registrar-admin-control`. Omitted port options
inherit the VLAN-level options. Registration control accepts `normal`,
`fixed-new-ignored` (the default), `fixed-new-propagated`, or `forbidden`.
The 2021 config-service serializer creates the VLAN name `VLAN_10` from VID 10;
the 2018 serializer does not emit the VLAN-name block. This change does not alter
either serializer. The generated configuration also includes a VID-to-FID mapping
and selected-port PVID settings, so it is not a byte-for-byte copy of the example.
The YAML VID is retained even when an AddStream request specifies a different VID.
