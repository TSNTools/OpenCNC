# ⚠ Capability Validation Limitation (Future Work)

## Current Design

`SupportedByDevice()` determines feature support by checking:

- YANG filename  
- YANG revision  

from the static `DeviceModel` definition.

This assumes:

- Vendors advertise correct module names
- Revisions accurately reflect implementation
- Full module functionality is implemented

---

## Known Risks

### 1. Vendor Misreporting

- Device advertises a standard YANG module + revision
- Implementation deviates from the actual specification

### 2. Partial Implementation

- Module is present
- But some leaves / containers / RPCs are missing
- `if-feature` statements or deviations are not detected
- Feature appears supported but fails during configuration

### 3. Static Model vs Runtime Reality

- `DeviceModel` JSON may not match actual NETCONF `<hello>` capabilities
- Device firmware upgrades may change capabilities without updating registry

---

## Consequence

Feature detection may return **true**,  
but `edit-config` can fail at runtime.

Failure occurs late (during push phase).

---

## Planned Improvement (Future)

Introduce runtime capability validation.

Possible approaches:

- Parse NETCONF `<hello>` capabilities
- Detect deviations and `if-feature` support
- Cache actual supported modules per device session
- Optionally probe feature using minimal test configuration

Goal:

Move from **static declaration-based validation**  
to **runtime capability-based validation**

---

## Current Status

Accepted technical debt.

- Safe in controlled or vendor-aligned environments
- Needs enhancement for full multi-vendor robustness
