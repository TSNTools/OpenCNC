Config Flags:
export OBS_ENABLED=true
export OBS_KAFKA_ENABLED=true
export OBS_FAIL_OPEN=true
export OBS_BROKERS=localhost:9092
export OBS_CMD_MIRROR=false



## Local Kafka Test Environment

`start_kafka_local_testing.sh` starts a local Kafka development environment for OpenCNC testing.

The script:

- Starts the Kafka container if it is not already running
- Creates the required `opencnc.*` Kafka topics
- Starts a live consumer that follows all OpenCNC topics
- Displays consumed messages in the format:

```text
<topic:message>
```

### Created Topics

- `opencnc.logs`
- `opencnc.events`
- `opencnc.metrics`
- `opencnc.audit`
- `opencnc.health`

### Usage

```bash
./start_kafka_local_testing.sh
```

This script is intended for local development and testing only.