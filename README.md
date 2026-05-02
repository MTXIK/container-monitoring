# container-monitoring

Container monitoring platform scaffold.

This repository is organized as a workspace that contains separate folders for
future service repositories:

- `agent/` - node-side Go agent that collects container telemetry and publishes
  it to Kafka.
- `core/` - central Go service that consumes telemetry, stores data, evaluates
  alert rules, sends notifications, and exposes the HTTP API.
- `contracts/` - versioned protobuf contracts shared by services through a
  separate package/repository boundary.

The storage and visualization infrastructure belongs to the receiving side and
is therefore placed under `core/deploy/`, not under `agent/`.

## Local Commands

```bash
make test
make build
make core-up
make core-down
```
