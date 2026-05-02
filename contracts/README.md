# container-monitoring-contracts

Versioned service contracts.

This folder is intended to become a separate repository used by both
`container-monitoring-agent` and `container-monitoring-core`.

## Layout

- `proto/container/v1/telemetry.proto` - Kafka payload contracts for metrics and
  container events.
- `buf.yaml` - protobuf module configuration.

Generated code should be published from this repository instead of being copied
between services.
