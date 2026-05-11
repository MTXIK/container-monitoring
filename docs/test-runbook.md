# Test Runbook

This document describes how to verify the MVP after code changes. It separates
fast local checks from the full Docker-based integration run.

## Prerequisites

- Docker Desktop or Docker Engine with Compose v2.
- Go toolchain matching the `go.mod` files.
- Node.js 22 for the frontend.
- Python 3 for the e2e unittest runner.
- `jq` for the manual API checks.

All commands below are run from the repository root unless stated otherwise.

## Fast Local Checks

Run these before pushing code or opening a pull request:

```bash
make test
make build
cd frontend
npm run lint
npm run build
```

Expected result:

- Go tests pass for `agent/` and `core/`.
- Go packages compile.
- ESLint exits with code 0.
- Vite production build exits with code 0.

Validate Compose syntax without starting services:

```bash
docker compose config
```

## Full Stack Run

Start the complete local stack:

```bash
docker compose up --build
```

Wait until these services are healthy or available:

- Core API: `http://localhost:8080/ready`
- Frontend: `http://localhost:5173`
- Grafana: `http://localhost:3000`
- Demo target: `http://localhost:8081`

Basic readiness checks:

```bash
curl -sS http://localhost:8080/health | jq
curl -sS http://localhost:8080/ready | jq
curl -sS http://localhost:8081 >/dev/null
```

## Demo Targets Stack

The default stack contains one demo target, `target-nginx`. For a richer
demonstration of monitoring features, run the stack with the demo-targets
override:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo-targets.yml up --build
```

This starts the core platform plus additional containers designed for manual
testing and defense demonstrations.

Demo services:

| Service | Purpose | Useful checks |
| --- | --- | --- |
| `demo-nginx-stable` | Stable HTTP target | Discovery, latest metrics, Grafana panels |
| `demo-cpu-stress` | Continuous CPU load | CPU threshold alerts, duration rules, deduplication |
| `demo-memory-stress` | Controlled memory pressure | Memory metrics and memory alert rules |
| `demo-flaky` | Exits and restarts repeatedly | Docker `die`, `start`, and restart event handling |
| `demo-manual-recovery` | Manually stopped nginx target | `container_stopped` incident and restart recovery |
| `demo-labels` | Target with rich labels | Metadata/labels display and target details |

Demo target URLs:

- `demo-nginx-stable`: `http://localhost:8082`
- `demo-manual-recovery`: `http://localhost:8083`
- `demo-labels`: `http://localhost:8084`

Readiness checks for HTTP demo targets:

```bash
curl -sS http://localhost:8082 >/dev/null
curl -sS http://localhost:8083 >/dev/null
curl -sS http://localhost:8084 >/dev/null
```

Stop only the demo targets while keeping the platform running:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo-targets.yml stop \
  demo-nginx-stable demo-cpu-stress demo-memory-stress demo-flaky \
  demo-manual-recovery demo-labels
```

Remove only stopped demo target containers:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo-targets.yml rm -f \
  demo-nginx-stable demo-cpu-stress demo-memory-stress demo-flaky \
  demo-manual-recovery demo-labels
```

## Automated E2E

With the stack running, execute:

```bash
make e2e-test
```

Expected result:

- The e2e suite discovers `target-nginx`.
- Latest metrics exist for the demo target.
- Stopping `target-nginx` creates a Docker event, incident, and recovery action.

If the stack is not running, the e2e suite is expected to skip instead of fail.

## Manual Business Scenarios

The scenarios below verify the main business logic of the monitoring platform:
target discovery, metric ingestion, alert rules, incident lifecycle, recovery
actions, API contracts, and frontend behavior.

Before running the business scenarios, capture a baseline:

```bash
curl -sS http://localhost:8080/api/v1/targets | jq
curl -sS http://localhost:8080/api/v1/alert-rules | jq
curl -sS http://localhost:8080/api/v1/incidents | jq
curl -sS http://localhost:8080/api/v1/recovery-actions | jq
curl -sS 'http://localhost:8080/api/v1/events?limit=10' | jq
```

If the database contains old demo data, reset the stack with
`docker compose down -v` and start again.

### Shared Test Data

Most scenarios use the demo target. Store its id in a shell variable:

```bash
TARGET_ID="$(curl -sS http://localhost:8080/api/v1/targets \
  | jq -r '.[] | select(.name | contains("target-nginx")) | .id' \
  | head -n 1)"
echo "$TARGET_ID"
```

Pass criteria:

- `TARGET_ID` is not empty.
- The matching target has `source = docker`.
- The matching target has a non-empty `node_id`.

If `TARGET_ID` is empty, wait one agent collection interval and repeat:

```bash
sleep 8
curl -sS http://localhost:8080/api/v1/targets | jq
```

When running with `docker-compose.demo-targets.yml`, capture specific demo
target ids:

```bash
STABLE_TARGET_ID="$(curl -sS http://localhost:8080/api/v1/targets \
  | jq -r '.[] | select(.name | contains("demo-nginx-stable")) | .id' \
  | head -n 1)"
CPU_TARGET_ID="$(curl -sS http://localhost:8080/api/v1/targets \
  | jq -r '.[] | select(.name | contains("demo-cpu-stress")) | .id' \
  | head -n 1)"
MEMORY_TARGET_ID="$(curl -sS http://localhost:8080/api/v1/targets \
  | jq -r '.[] | select(.name | contains("demo-memory-stress")) | .id' \
  | head -n 1)"
FLAKY_TARGET_ID="$(curl -sS http://localhost:8080/api/v1/targets \
  | jq -r '.[] | select(.name | contains("demo-flaky")) | .id' \
  | head -n 1)"
RECOVERY_TARGET_ID="$(curl -sS http://localhost:8080/api/v1/targets \
  | jq -r '.[] | select(.name | contains("demo-manual-recovery")) | .id' \
  | head -n 1)"
LABELS_TARGET_ID="$(curl -sS http://localhost:8080/api/v1/targets \
  | jq -r '.[] | select(.name | contains("demo-labels")) | .id' \
  | head -n 1)"

printf 'stable=%s\ncpu=%s\nmemory=%s\nflaky=%s\nrecovery=%s\nlabels=%s\n' \
  "$STABLE_TARGET_ID" "$CPU_TARGET_ID" "$MEMORY_TARGET_ID" \
  "$FLAKY_TARGET_ID" "$RECOVERY_TARGET_ID" "$LABELS_TARGET_ID"
```

Pass criteria:

- Every variable is non-empty when the demo-targets override is running.
- The target names contain the matching service names.

### TC-BL-001: Target Discovery

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that the agent discovers Docker containers and core persists
them as monitoring targets.

Steps:

```bash
curl -sS http://localhost:8080/api/v1/targets | jq \
  '.[] | {id, name, source, external_id, node_id, status, last_seen_at}'
```

Expected result:

- `target-nginx` is present.
- `source` is `docker`.
- `external_id` equals the Docker container id used as target id.
- `node_id` is `local-node` in the default Compose stack.
- `last_seen_at` is populated.
- `status` is one of `OK`, `WARNING`, `CRITICAL`, `UNKNOWN`, or `RECOVERING`.

Fail conditions:

- No target appears after two collection intervals.
- `source` is empty or not `docker`.
- `last_seen_at` is empty.

### TC-BL-002: Latest Metrics Are Ingested

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify metric flow from Docker stats to Kafka, core, ClickHouse, and
HTTP API.

Steps:

```bash
curl -sS "http://localhost:8080/api/v1/metrics/latest?target_id=$TARGET_ID&limit=5" | jq
```

Expected result:

- At least one metric row is returned.
- `target_id` equals `$TARGET_ID`.
- `container_name` is not empty.
- `cpu_usage_percent` is a number.
- `memory_usage_bytes` is a number.
- `network_rx_bytes` and `network_tx_bytes` are numbers.
- `timestamp` is recent relative to the running stack.

Fail conditions:

- Empty array after two collection intervals.
- Numeric metric fields are missing or encoded as strings.
- Metrics are returned for the wrong target.

### TC-BL-003: Metric History Filtering

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that metric history returns points for a target and metric name.

Steps:

```bash
curl -sS "http://localhost:8080/api/v1/metrics/history?target_id=$TARGET_ID&metric_name=cpu_usage_percent&limit=20" | jq
```

Expected result:

- The response is an array.
- Every returned point has `target_id = $TARGET_ID`.
- Every returned point has `metric_name = cpu_usage_percent`.
- Points are ordered from newest to oldest.

Fail conditions:

- Points for another target are mixed in.
- A different metric name is returned.
- The endpoint returns a server error for a valid target and metric.

### TC-BL-004: Alert Rule Create

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that the admin API creates a threshold rule with all business
fields needed by the analyzer.

Steps:

```bash
RULE_ID="$(curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC CPU target scoped\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 0,
    \"duration\": \"0s\",
    \"severity\": \"critical\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq -r '.id')"
echo "$RULE_ID"
curl -sS http://localhost:8080/api/v1/alert-rules | jq ".[] | select(.id == \"$RULE_ID\")"
```

Expected result:

- `RULE_ID` is not empty.
- The stored rule has `target_id = $TARGET_ID`.
- `operator` is normalized to `gte`.
- `duration` is `0s`.
- `recovery_policy` is `notify_only`.
- `enabled` is `true`.

Fail conditions:

- `target_id` is lost.
- `operator` is not normalized.
- `enabled` becomes false.

### TC-BL-005: Target-Scoped Alert Does Not Affect Other Targets

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that a target-scoped rule is evaluated only for its target.

Precondition:

- At least two Docker containers are visible in `/api/v1/targets`. In the
  default stack this is usually true because core, agent, frontend, and
  `target-nginx` are all Docker containers.

Steps:

```bash
OTHER_TARGET_ID="$(curl -sS http://localhost:8080/api/v1/targets \
  | jq -r --arg target "$TARGET_ID" '.[] | select(.id != $target) | .id' \
  | head -n 1)"
echo "$OTHER_TARGET_ID"

curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC other target scoped\",
    \"target_id\": \"$OTHER_TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 0,
    \"duration\": \"0s\",
    \"severity\": \"warning\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq

sleep 8
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg target "$TARGET_ID" '.[] | select(.target_id == $target and (.rule_id | startswith("TC")))'
```

Expected result:

- The rule for `$OTHER_TARGET_ID` does not create an incident for `$TARGET_ID`.
- Any incident opened by that rule has `target_id = $OTHER_TARGET_ID`.

Fail conditions:

- A target-scoped rule creates incidents for unrelated targets.

### TC-BL-006: Operators `gt`, `gte`, `lt`, `lte`, and `eq`

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that all supported rule operators are accepted and evaluated.

Create rules with thresholds that should match current metrics:

```bash
curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC operator gte\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 0,
    \"duration\": \"0s\",
    \"severity\": \"warning\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq

curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC operator lt\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \"<\",
    \"threshold\": 1000000,
    \"duration\": \"0s\",
    \"severity\": \"info\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq
```

Wait and inspect incidents:

```bash
sleep 8
curl -sS http://localhost:8080/api/v1/incidents | jq \
  '.[] | select(.description | contains("cpu_usage_percent"))'
```

Expected result:

- Rules using `>=` and `<` can create incidents.
- Stored operators are normalized to backend values such as `gte` and `lt`.

Additional checks:

- Use `<=` with a very high threshold to verify `lte`.
- Use `>` with a low threshold to verify `gt`.
- Use `==` only with a stable value that is known exactly; otherwise it may be
  hard to trigger with live Docker metrics.

Fail conditions:

- A supported operator is accepted by the API but never creates an incident when
  the condition is true.

### TC-BL-007: Duration-Based Alert

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that `duration` means the threshold must stay true for the
configured time window.

Steps:

```bash
DURATION_RULE_ID="$(curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC sustained CPU\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 0,
    \"duration\": \"15s\",
    \"severity\": \"warning\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq -r '.id')"
echo "$DURATION_RULE_ID"

sleep 5
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg rule "$DURATION_RULE_ID" '.[] | select(.rule_id == $rule)'

sleep 15
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg rule "$DURATION_RULE_ID" '.[] | select(.rule_id == $rule)'
```

Expected result:

- No incident exists for `$DURATION_RULE_ID` after the first `sleep 5`.
- One incident exists after the second wait.

Fail conditions:

- The incident is created immediately before the duration elapses.
- More than one open incident is created for the same rule and target.

### TC-BL-008: Alert Deduplication

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that repeated matching metric batches do not create duplicate
open incidents for the same `rule_id + target_id`.

Steps:

```bash
DEDUP_RULE_ID="$(curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC dedup CPU\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 0,
    \"duration\": \"0s\",
    \"severity\": \"critical\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq -r '.id')"

sleep 20
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg rule "$DEDUP_RULE_ID" '[.[] | select(.rule_id == $rule and .status != "resolved")] | length'
```

Expected result:

- The final count is `1`.

Fail conditions:

- Count is greater than `1` while the first incident remains open or
  acknowledged.

### TC-BL-009: Incident Acknowledge

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that an operator can acknowledge an open incident without
marking it resolved.

Steps:

```bash
INCIDENT_ID="$(curl -sS http://localhost:8080/api/v1/incidents \
  | jq -r '.[] | select(.status == "open") | .id' \
  | head -n 1)"
echo "$INCIDENT_ID"

curl -sS -X POST "http://localhost:8080/api/v1/incidents/$INCIDENT_ID/ack" -i
curl -sS "http://localhost:8080/api/v1/incidents/$INCIDENT_ID" | jq
```

Expected result:

- The acknowledge endpoint returns HTTP 204.
- The incident status becomes `acknowledged`.
- `resolved_at` remains empty.

Fail conditions:

- Acknowledgement deletes the incident.
- `resolved_at` is set by acknowledgement.

### TC-BL-010: Incident Resolve and New Incident After Resolve

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify incident closure and dedup reset after resolution.

Steps:

```bash
curl -sS -X POST "http://localhost:8080/api/v1/incidents/$INCIDENT_ID/resolve" -i
curl -sS "http://localhost:8080/api/v1/incidents/$INCIDENT_ID" | jq
sleep 8
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg target "$TARGET_ID" '.[] | select(.target_id == $target)'
```

Expected result:

- The resolve endpoint returns HTTP 204.
- The incident status becomes `resolved`.
- `resolved_at` is populated.
- If the rule condition is still true, a later metric batch may create a new
  incident for the same rule and target because the old one is resolved.

Fail conditions:

- Resolved incident still blocks all future incidents forever.
- `resolved_at` is not populated.

### TC-BL-011: Disabled Rule Does Not Fire

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that disabled alert rules are ignored by the analyzer.

Steps:

```bash
DISABLED_RULE_ID="$(curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC disabled CPU\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 0,
    \"duration\": \"0s\",
    \"severity\": \"critical\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": false
  }" | jq -r '.id')"

sleep 8
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg rule "$DISABLED_RULE_ID" '.[] | select(.rule_id == $rule)'
```

Expected result:

- No incident is created for `$DISABLED_RULE_ID`.

Fail conditions:

- A disabled rule creates an incident.

### TC-BL-012: Alert Rule Update Changes Evaluation

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that PATCH changes rule behavior.

Steps:

```bash
PATCH_RULE_ID="$(curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC patch CPU\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 1000000,
    \"duration\": \"0s\",
    \"severity\": \"warning\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq -r '.id')"

sleep 8
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg rule "$PATCH_RULE_ID" '.[] | select(.rule_id == $rule)'

curl -sS -X PATCH "http://localhost:8080/api/v1/alert-rules/$PATCH_RULE_ID" \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC patch CPU active\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 0,
    \"duration\": \"0s\",
    \"severity\": \"critical\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq

sleep 8
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg rule "$PATCH_RULE_ID" '.[] | select(.rule_id == $rule)'
```

Expected result:

- Before PATCH, no incident exists for `$PATCH_RULE_ID`.
- After PATCH, one incident is created.
- The created incident has severity `critical`.

Fail conditions:

- PATCH returns success but analyzer still uses old threshold/severity.

### TC-BL-013: Alert Rule Delete Stops Future Evaluation

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that deleting a rule removes it from active monitoring.

Steps:

```bash
DELETE_RULE_ID="$(curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC delete CPU\",
    \"target_id\": \"$TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 1000000,
    \"duration\": \"0s\",
    \"severity\": \"warning\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq -r '.id')"

curl -sS -X DELETE "http://localhost:8080/api/v1/alert-rules/$DELETE_RULE_ID" -i
curl -sS http://localhost:8080/api/v1/alert-rules | jq \
  --arg rule "$DELETE_RULE_ID" '.[] | select(.id == $rule)'
```

Expected result:

- DELETE returns HTTP 204.
- The deleted rule no longer appears in `/api/v1/alert-rules`.
- No future incident uses that rule id.

Fail conditions:

- Deleted rule still appears in the rule list.

### TC-BL-014: Docker Stop Event Creates Incident and Recovery

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify event-driven incident and self-healing behavior.

Steps:

```bash
MARKER="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
docker compose stop target-nginx
sleep 8

curl -sS 'http://localhost:8080/api/v1/events?limit=50' | jq \
  --arg target "$TARGET_ID" '.[] | select(.target_id == $target)'
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg target "$TARGET_ID" '.[] | select(.target_id == $target and .rule_id == "container_stopped")'
curl -sS http://localhost:8080/api/v1/recovery-actions | jq \
  --arg target "$TARGET_ID" '.[] | select(.target_id == $target and .action_type == "restart_container")'
```

Expected result:

- An event with `event_type = container_stopped` appears.
- The target status becomes `CRITICAL`.
- An incident with `rule_id = container_stopped` appears.
- A recovery action with `action_type = restart_container` appears.
- Recovery status is one of `succeeded`, `failed`, or `skipped`.

Fail conditions:

- Stop event is stored but no incident is created.
- Incident exists but no recovery action is recorded.
- Target remains `OK` after a stop event.

Cleanup:

```bash
docker compose start target-nginx
```

### TC-BL-015: Docker Start Event Restores Target Status

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that lifecycle events update target status back to healthy.

Steps:

```bash
docker compose start target-nginx
sleep 8
curl -sS "http://localhost:8080/api/v1/targets/$TARGET_ID" | jq
curl -sS 'http://localhost:8080/api/v1/events?limit=20' | jq \
  --arg target "$TARGET_ID" '.[] | select(.target_id == $target)'
```

Expected result:

- A `container_started` event appears.
- Target status becomes `OK`.
- Metrics resume for `$TARGET_ID`.

Fail conditions:

- Status remains `CRITICAL` after a start event and new metrics.
- No start event is stored.

### TC-BL-016: Recovery Retry

Automated E2E: covered by `e2e/test_monitoring_stack.py` when a failed recovery action exists.

Purpose: verify that retry is a real business action, not a mock response.

Precondition:

- At least one recovery action has `status = failed`. If none exists, this
  scenario is not applicable for the current run.

Steps:

```bash
ACTION_ID="$(curl -sS http://localhost:8080/api/v1/recovery-actions \
  | jq -r '.[] | select(.status == "failed") | .id' \
  | head -n 1)"
echo "$ACTION_ID"

curl -sS -X POST "http://localhost:8080/api/v1/recovery-actions/$ACTION_ID/retry" -i
curl -sS http://localhost:8080/api/v1/recovery-actions | jq
```

Expected result:

- Retry endpoint returns HTTP 202.
- A new recovery attempt is recorded.
- The new attempt references the same incident and target.
- The new attempt status eventually becomes `succeeded`, `failed`, or
  `skipped`.

Fail conditions:

- Endpoint returns success but no new action is recorded.
- Retry action is recorded against the wrong incident.

### TC-BL-017: Recovery Lock Prevents Concurrent Restarts

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify that duplicate recovery requests for the same target do not run
concurrently.

Steps:

```bash
docker compose stop target-nginx
sleep 2
docker compose stop target-nginx
sleep 8
curl -sS http://localhost:8080/api/v1/recovery-actions | jq \
  --arg target "$TARGET_ID" '.[] | select(.target_id == $target)'
```

Expected result:

- At most one recovery action for the active failure runs at a time.
- If a second action is created while the lock is held, its status is
  `skipped`.

Fail conditions:

- Multiple simultaneous `running` recovery actions exist for the same target.

Cleanup:

```bash
docker compose start target-nginx
```

### TC-BL-018: Bad Alert Rule Payload Is Rejected

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify API validation for malformed business input.

Steps:

```bash
curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "TC bad duration",
    "metric_name": "cpu_usage_percent",
    "operator": ">=",
    "threshold": 0,
    "duration": "not-a-duration",
    "severity": "warning",
    "recovery_policy": "notify_only",
    "enabled": true
  }' -i
```

Expected result:

- API returns HTTP 400.
- The rule is not persisted.

Fail conditions:

- API returns HTTP 201 for invalid duration.

### TC-BL-019: Unknown Incident Id

Automated E2E: covered by `e2e/test_monitoring_stack.py`.

Purpose: verify not-found behavior for incident details.

Steps:

```bash
curl -sS http://localhost:8080/api/v1/incidents/999999999 -i
```

Expected result:

- API returns HTTP 404.

Fail conditions:

- API returns an empty incident with HTTP 200.

### TC-BL-020: Frontend Uses Real API by Default

Purpose: verify that the frontend does not silently show mock data when the API
is unavailable.

Steps:

1. Ensure `VITE_ENABLE_MOCK_FALLBACK` is not set to `true`.
2. Stop only the core service:

   ```bash
   docker compose stop core
   ```

3. Open `http://localhost:5173`.
4. Navigate to Dashboard, Targets, Incidents, and Recovery Actions.

Expected result:

- API-dependent pages show error states instead of mock data.
- Browser network tab shows failed requests to `http://localhost:8080`.

Fail conditions:

- The frontend shows realistic demo data while core is stopped.

Cleanup:

```bash
docker compose start core
```

### TC-BL-021: Frontend Mock Mode Is Explicit

Purpose: verify that mock data is available only when explicitly requested.

Steps:

```bash
cd frontend
VITE_ENABLE_MOCK_FALLBACK=true npm run dev
```

Stop core and open the frontend dev server.

Expected result:

- Mock data appears only in this explicit mock mode.
- This behavior is suitable for UI-only demos, not for integration validation.

Fail conditions:

- Mock data appears without `VITE_ENABLE_MOCK_FALLBACK=true`.

### TC-BL-022: Grafana Dashboard Data

Purpose: verify that Grafana can read ClickHouse-backed metrics and events.

Steps:

1. Open `http://localhost:3000`.
2. Log in with `admin` / `admin`.
3. Open the `Container Monitoring MVP` dashboard.
4. Inspect panels:
   - CPU usage by container.
   - Memory usage by container.
   - Container events over time.
   - Critical event count.
   - Latest container events table.

Expected result:

- Panels load without datasource errors.
- CPU and memory panels show data after at least one collection interval.
- Event panels update after stopping or starting `target-nginx`.

Fail conditions:

- Grafana shows ClickHouse datasource errors.
- Panels remain empty while API metrics/events are populated.

### TC-BL-023: Demo Stable Target

Automated E2E: covered by `e2e/test_monitoring_stack.py` when the demo-targets override is running.

Purpose: verify the additional stable demo HTTP target.

Precondition:

- Stack is running with `docker-compose.demo-targets.yml`.
- `STABLE_TARGET_ID` is set.

Steps:

```bash
curl -sS http://localhost:8082 >/dev/null
curl -sS "http://localhost:8080/api/v1/targets/$STABLE_TARGET_ID" | jq
curl -sS "http://localhost:8080/api/v1/metrics/latest?target_id=$STABLE_TARGET_ID&limit=5" | jq
```

Expected result:

- HTTP endpoint on port `8082` responds.
- Target details show a Docker target whose name contains `demo-nginx-stable`.
- Latest metrics exist for `STABLE_TARGET_ID`.

Fail conditions:

- HTTP service responds but target is not discovered.
- Target exists but latest metrics stay empty after two collection intervals.

### TC-BL-024: Demo CPU Stress Alert

Automated E2E: covered by `e2e/test_monitoring_stack.py` when the demo-targets override is running.

Purpose: verify high-CPU monitoring with a dedicated CPU load container.

Precondition:

- Stack is running with `docker-compose.demo-targets.yml`.
- `CPU_TARGET_ID` is set.

Steps:

```bash
CPU_RULE_ID="$(curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC demo CPU stress\",
    \"target_id\": \"$CPU_TARGET_ID\",
    \"metric_name\": \"cpu_usage_percent\",
    \"operator\": \">=\",
    \"threshold\": 10,
    \"duration\": \"10s\",
    \"severity\": \"critical\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq -r '.id')"

sleep 20
curl -sS "http://localhost:8080/api/v1/metrics/latest?target_id=$CPU_TARGET_ID&limit=5" | jq
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg rule "$CPU_RULE_ID" '.[] | select(.rule_id == $rule)'
```

Expected result:

- CPU metric for `demo-cpu-stress` is above the threshold during the test.
- Exactly one open incident appears for `CPU_RULE_ID`.
- Incident severity is `critical`.

Fail conditions:

- CPU target produces no metrics.
- Multiple duplicate open incidents appear for the same rule and target.

### TC-BL-025: Demo Memory Stress Alert

Automated E2E: covered by `e2e/test_monitoring_stack.py` when the demo-targets override is running.

Purpose: verify memory metric collection and memory alerting.

Precondition:

- Stack is running with `docker-compose.demo-targets.yml`.
- `MEMORY_TARGET_ID` is set.

Steps:

```bash
MEMORY_RULE_ID="$(curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d "{
    \"name\": \"TC demo memory stress\",
    \"target_id\": \"$MEMORY_TARGET_ID\",
    \"metric_name\": \"memory_usage_bytes\",
    \"operator\": \">=\",
    \"threshold\": 50000000,
    \"duration\": \"10s\",
    \"severity\": \"warning\",
    \"recovery_policy\": \"notify_only\",
    \"enabled\": true
  }" | jq -r '.id')"

sleep 20
curl -sS "http://localhost:8080/api/v1/metrics/latest?target_id=$MEMORY_TARGET_ID&limit=5" | jq
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg rule "$MEMORY_RULE_ID" '.[] | select(.rule_id == $rule)'
```

Expected result:

- `memory_usage_bytes` for `demo-memory-stress` is above `50000000`.
- One incident appears for `MEMORY_RULE_ID`.
- Incident severity is `warning`.

Fail conditions:

- Memory metric remains near zero after the container has been running.
- Alert rule never creates an incident while the metric is above threshold.

### TC-BL-026: Demo Flaky Lifecycle Events

Automated E2E: covered by `e2e/test_monitoring_stack.py` when the demo-targets override is running.

Purpose: verify lifecycle event ingestion from a repeatedly crashing container.

Precondition:

- Stack is running with `docker-compose.demo-targets.yml`.
- `FLAKY_TARGET_ID` is set.

Steps:

```bash
sleep 45
curl -sS 'http://localhost:8080/api/v1/events?limit=100' | jq \
  --arg target "$FLAKY_TARGET_ID" '.[] | select(.target_id == $target)'
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg target "$FLAKY_TARGET_ID" '.[] | select(.target_id == $target)'
```

Expected result:

- Events include at least one `container_died` or restart-related lifecycle
  event for `demo-flaky`.
- An event-driven incident appears for the flaky target.
- Repeated crashes do not create unlimited duplicate open incidents for the same
  event rule and target.

Fail conditions:

- The container restarts in Docker but no events reach the API.
- Each restart creates another duplicate open incident for the same target and
  event type.

### TC-BL-027: Demo Manual Recovery Target

Automated E2E: covered by `e2e/test_monitoring_stack.py` when the demo-targets override is running.

Purpose: verify the controlled manual stop and recovery flow on a dedicated
container instead of the default `target-nginx`.

Precondition:

- Stack is running with `docker-compose.demo-targets.yml`.
- `RECOVERY_TARGET_ID` is set.

Steps:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo-targets.yml stop demo-manual-recovery
sleep 10

curl -sS 'http://localhost:8080/api/v1/events?limit=100' | jq \
  --arg target "$RECOVERY_TARGET_ID" '.[] | select(.target_id == $target)'
curl -sS http://localhost:8080/api/v1/incidents | jq \
  --arg target "$RECOVERY_TARGET_ID" '.[] | select(.target_id == $target and .rule_id == "container_stopped")'
curl -sS http://localhost:8080/api/v1/recovery-actions | jq \
  --arg target "$RECOVERY_TARGET_ID" '.[] | select(.target_id == $target)'
```

Expected result:

- `container_stopped` event appears for `demo-manual-recovery`.
- `container_stopped` incident appears for the same target.
- `restart_container` recovery action is recorded.
- Recovery status is `succeeded`, `failed`, or `skipped`.

Cleanup:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo-targets.yml start demo-manual-recovery
```

Fail conditions:

- Recovery action references a different target.
- Stop event is stored but recovery is not attempted.

### TC-BL-028: Demo Labels Target

Automated E2E: covered by `e2e/test_monitoring_stack.py` when the demo-targets override is running.

Purpose: verify a target with rich Docker labels is visible and usable as a
normal monitoring target.

Precondition:

- Stack is running with `docker-compose.demo-targets.yml`.
- `LABELS_TARGET_ID` is set.

Steps:

```bash
curl -sS http://localhost:8084 >/dev/null
curl -sS "http://localhost:8080/api/v1/targets/$LABELS_TARGET_ID" | jq
curl -sS "http://localhost:8080/api/v1/metrics/latest?target_id=$LABELS_TARGET_ID&limit=5" | jq
```

Expected result:

- HTTP endpoint on port `8084` responds.
- Target details show a target whose name contains `demo-labels`.
- Latest metrics exist for `LABELS_TARGET_ID`.
- Frontend target details page can open this target without errors.

Note:

- Docker labels are included in the compose file to make the container easy to
  identify in Docker and future metadata views. Current metric ingestion may not
  persist every Docker label into the API response.

Fail conditions:

- Target is not discovered.
- Frontend target details page fails for the labels target.

## Telegram Scenario

Telegram is optional. To verify notifications:

```bash
export TELEGRAM_BOT_TOKEN="<bot-token>"
export TELEGRAM_CHAT_ID="<chat-id>"
docker compose up --build
```

Trigger either a metric alert or Docker stop event. Expected result:

- A Telegram message is sent with severity, target, reason, incident id, and
  recovery policy.

## Cleanup

Stop the stack while keeping volumes:

```bash
docker compose down
```

Reset all local data:

```bash
docker compose down -v
```

Use `down -v` when schema or migration behavior needs a clean database.
