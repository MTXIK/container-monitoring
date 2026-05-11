import json
import os
import subprocess
import time
import unittest
from datetime import datetime, timezone
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


API_URL = os.getenv("E2E_API_URL", "http://localhost:8080").rstrip("/")
COMPOSE_FILE = os.getenv("E2E_COMPOSE_FILE", "docker-compose.yml")
DEMO_COMPOSE_FILES = os.getenv(
    "E2E_DEMO_COMPOSE_FILES",
    "docker-compose.yml:docker-compose.demo-targets.yml",
).split(":")
DEMO_SERVICE = os.getenv("E2E_DEMO_SERVICE", "target-nginx")
POLL_INTERVAL_SECONDS = float(os.getenv("E2E_POLL_INTERVAL_SECONDS", "2"))
DEFAULT_TIMEOUT_SECONDS = float(os.getenv("E2E_TIMEOUT_SECONDS", "90"))


class APIError(AssertionError):
    pass


def api_request(
    method: str,
    path: str,
    query: dict[str, Any] | None = None,
    body: dict[str, Any] | None = None,
    expected_status: int | None = None,
) -> tuple[int, Any]:
    url = f"{API_URL}{path}"
    if query:
        url = f"{url}?{urlencode(query)}"
    payload = None
    headers = {"Accept": "application/json"}
    if body is not None:
        payload = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    request = Request(url, data=payload, headers=headers, method=method)
    try:
        with urlopen(request, timeout=5) as response:
            response_body = response.read().decode("utf-8")
            status = response.status
    except HTTPError as exc:
        response_body = exc.read().decode("utf-8")
        status = exc.code
    if expected_status is not None and status != expected_status:
        raise APIError(f"{method} {url} returned {status}, want {expected_status}: {response_body}")
    if expected_status is None and status >= 400:
        raise APIError(f"{method} {url} returned {status}: {response_body}")
    if not response_body:
        return status, None
    try:
        return status, json.loads(response_body)
    except json.JSONDecodeError:
        return status, response_body


def api_json(path: str, query: dict[str, Any] | None = None) -> Any:
    return api_request("GET", path, query=query)[1]


def post_json(path: str, body: dict[str, Any], expected_status: int = 201) -> Any:
    return api_request("POST", path, body=body, expected_status=expected_status)[1]


def patch_json(path: str, body: dict[str, Any]) -> Any:
    return api_request("PATCH", path, body=body, expected_status=200)[1]


def send_status(method: str, path: str, expected_status: int) -> None:
    api_request(method, path, expected_status=expected_status)


def delete_alert_rule(rule_id: str) -> None:
    try:
        send_status("DELETE", f"/api/v1/alert-rules/{rule_id}", 204)
    except APIError:
        pass


def alert_rule_payload(name: str, target_id: str, **overrides: Any) -> dict[str, Any]:
    payload = {
        "name": name,
        "target_id": target_id,
        "metric_name": "cpu_usage_percent",
        "operator": ">=",
        "threshold": 0,
        "duration": "0s",
        "severity": "warning",
        "recovery_policy": "notify_only",
        "enabled": True,
    }
    payload.update(overrides)
    return payload


def incidents_for_rule(rule_id: str) -> list[dict[str, Any]]:
    incidents = api_json("/api/v1/incidents")
    assert isinstance(incidents, list)
    return [incident for incident in incidents if incident.get("rule_id") == rule_id]


def open_incidents_for_rule(rule_id: str) -> list[dict[str, Any]]:
    return [incident for incident in incidents_for_rule(rule_id) if incident.get("status") != "resolved"]


def open_incidents_for_target(target_id: str) -> list[dict[str, Any]]:
    incidents = api_json("/api/v1/incidents")
    assert isinstance(incidents, list)
    return [
        incident
        for incident in incidents
        if incident.get("target_id") == target_id and incident.get("status") != "resolved"
    ]


def resolve_open_incidents(target_id: str, rule_ids: set[str]) -> None:
    for incident in open_incidents_for_target(target_id):
        if incident.get("rule_id") in rule_ids:
            send_status("POST", f"/api/v1/incidents/{incident['id']}/resolve", 204)


def wait_for_metric_batch() -> None:
    time.sleep(max(8, POLL_INTERVAL_SECONDS * 2))


def unique_test_name(suffix: str) -> str:
    return f"e2e {suffix} {int(time.time() * 1000)}"


def assert_recent_timestamp(testcase: unittest.TestCase, value: str) -> None:
    parsed = parse_rfc3339(value)
    testcase.assertLessEqual(parsed, datetime.now(timezone.utc))
    testcase.assertLess(datetime.now(timezone.utc).timestamp() - parsed.timestamp(), DEFAULT_TIMEOUT_SECONDS * 2)


def wait_until(description: str, predicate, timeout: float = DEFAULT_TIMEOUT_SECONDS):
    deadline = time.monotonic() + timeout
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        try:
            result = predicate()
            if result:
                return result
        except Exception as exc:  # noqa: BLE001 - include last transient error in timeout.
            last_error = exc
        time.sleep(POLL_INTERVAL_SECONDS)
    suffix = f": {last_error}" if last_error else ""
    raise AssertionError(f"Timed out waiting for {description}{suffix}")


def parse_rfc3339(value: str) -> datetime:
    normalized = value.replace("Z", "+00:00")
    parsed = datetime.fromisoformat(normalized)
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def find_demo_target() -> dict[str, Any] | None:
    return find_target_by_name(DEMO_SERVICE)


def find_target_by_name(name_part: str) -> dict[str, Any] | None:
    targets = api_json("/api/v1/targets")
    assert isinstance(targets, list)
    for target in targets:
        name = target.get("name", "")
        if name_part in name and target.get("source") == "docker":
            return target
    return None


def compose(*args: str) -> None:
    compose_with_files([COMPOSE_FILE], *args)


def demo_compose(*args: str) -> None:
    compose_with_files([file for file in DEMO_COMPOSE_FILES if file], *args)


def compose_with_files(files: list[str], *args: str) -> None:
    command = ["docker", "compose"]
    for file in files:
        command.extend(["-f", file])
    command.extend(args)
    result = subprocess.run(
        command,
        check=False,
        text=True,
        capture_output=True,
    )
    if result.returncode != 0:
        raise AssertionError(
            f"docker compose {' '.join(args)} failed with {result.returncode}\n"
            f"stdout:\n{result.stdout}\n"
            f"stderr:\n{result.stderr}"
        )


def http_ok(url: str) -> bool:
    request = Request(url, headers={"Accept": "*/*"})
    with urlopen(request, timeout=5) as response:
        response.read()
        return response.status < 400


def setUpModule():
    try:
        wait_until("Core API readiness", lambda: api_json("/ready").get("status") == "ready", timeout=15)
    except (AssertionError, URLError) as exc:
        raise unittest.SkipTest(f"local e2e stack is not available at {API_URL}: {exc}") from exc


class MonitoringStackE2ETest(unittest.TestCase):
    def test_core_api_discovers_docker_target_with_required_metadata(self):
        target = wait_until("demo target registration", find_demo_target)

        self.assertTrue(target["id"])
        self.assertIn(DEMO_SERVICE, target["name"])
        self.assertEqual(target["source"], "docker")
        self.assertEqual(target["external_id"], target["id"])
        self.assertTrue(target["node_id"])
        self.assertIn(target["status"], {"OK", "WARNING", "CRITICAL", "UNKNOWN", "RECOVERING"})
        self.assertTrue(target["last_seen_at"])
        assert_recent_timestamp(self, target["last_seen_at"])

    def test_core_api_exposes_demo_target_and_latest_metrics(self):
        target = wait_until("demo target registration", find_demo_target)

        def latest_metric_for_target():
            metrics = api_json("/api/v1/metrics/latest", {"target_id": target["id"], "limit": 5})
            self.assertIsInstance(metrics, list)
            return next((metric for metric in metrics if metric.get("target_id") == target["id"]), None)

        metric = wait_until("latest metric for demo target", latest_metric_for_target)

        self.assertTrue(metric["container_name"])
        self.assertEqual(metric["node_id"], target["node_id"])
        self.assertIsInstance(metric["cpu_usage_percent"], (int, float))
        self.assertIsInstance(metric["memory_usage_bytes"], int)
        self.assertLessEqual(parse_rfc3339(metric["timestamp"]), datetime.now(timezone.utc))

    def test_metric_history_filters_by_target_and_metric_name(self):
        target = wait_until("demo target registration", find_demo_target)

        def cpu_history():
            points = api_json(
                "/api/v1/metrics/history",
                {"target_id": target["id"], "metric_name": "cpu_usage_percent", "limit": 20},
            )
            self.assertIsInstance(points, list)
            return points or None

        points = wait_until("metric history for demo target", cpu_history)
        previous_timestamp: datetime | None = None
        for point in points:
            self.assertEqual(point["target_id"], target["id"])
            self.assertEqual(point["metric_name"], "cpu_usage_percent")
            self.assertIsInstance(point["value"], (int, float))
            current_timestamp = parse_rfc3339(point["timestamp"])
            if previous_timestamp is not None:
                self.assertLessEqual(current_timestamp, previous_timestamp)
            previous_timestamp = current_timestamp

    def test_alert_rule_lifecycle_validation_and_incident_state_transitions(self):
        target = wait_until("demo target registration", find_demo_target)
        disabled_rule_id = ""
        patched_rule_id = ""
        delete_rule_id = ""

        try:
            disabled_rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(
                    unique_test_name("disabled cpu"),
                    target["id"],
                    enabled=False,
                    severity="critical",
                ),
            )
            disabled_rule_id = disabled_rule["id"]
            self.assertEqual(disabled_rule["target_id"], target["id"])
            self.assertEqual(disabled_rule["operator"], "gte")
            self.assertEqual(disabled_rule["duration"], "0s")
            self.assertEqual(disabled_rule["recovery_policy"], "notify_only")
            self.assertFalse(disabled_rule["enabled"])

            post_json(
                "/api/v1/alert-rules",
                {
                    "name": unique_test_name("bad duration"),
                    "metric_name": "cpu_usage_percent",
                    "operator": ">=",
                    "threshold": 0,
                    "duration": "not-a-duration",
                    "severity": "warning",
                    "recovery_policy": "notify_only",
                    "enabled": True,
                },
                expected_status=400,
            )

            patched_rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(
                    unique_test_name("patch inactive cpu"),
                    target["id"],
                    threshold=1_000_000,
                    severity="warning",
                ),
            )
            patched_rule_id = patched_rule["id"]
            wait_for_metric_batch()
            self.assertEqual(open_incidents_for_rule(disabled_rule_id), [])
            self.assertEqual(open_incidents_for_rule(patched_rule_id), [])

            updated_rule = patch_json(
                f"/api/v1/alert-rules/{patched_rule_id}",
                alert_rule_payload(
                    unique_test_name("patch active cpu"),
                    target["id"],
                    threshold=0,
                    severity="critical",
                ),
            )
            self.assertEqual(updated_rule["id"], patched_rule_id)
            self.assertEqual(updated_rule["severity"], "critical")

            def patched_incident():
                incidents = open_incidents_for_rule(patched_rule_id)
                return incidents[0] if incidents else None

            incident = wait_until("incident after alert rule patch", patched_incident)
            self.assertEqual(incident["target_id"], target["id"])
            self.assertEqual(incident["severity"], "critical")
            self.assertEqual(incident["status"], "open")

            send_status("POST", f"/api/v1/incidents/{incident['id']}/ack", 204)
            acknowledged = api_json(f"/api/v1/incidents/{incident['id']}")
            self.assertEqual(acknowledged["status"], "acknowledged")
            self.assertNotIn("resolved_at", acknowledged)

            send_status("POST", f"/api/v1/incidents/{incident['id']}/resolve", 204)
            resolved = api_json(f"/api/v1/incidents/{incident['id']}")
            self.assertEqual(resolved["status"], "resolved")
            self.assertTrue(resolved["resolved_at"])

            def new_incident_after_resolve():
                incidents = open_incidents_for_rule(patched_rule_id)
                return next((item for item in incidents if item.get("id") != incident["id"]), None)

            repeated_incident = wait_until("new incident after resolving active rule", new_incident_after_resolve)
            self.assertEqual(repeated_incident["target_id"], target["id"])
            self.assertEqual(repeated_incident["severity"], "critical")

            delete_rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(unique_test_name("delete cpu"), target["id"], threshold=1_000_000),
            )
            delete_rule_id = delete_rule["id"]
            send_status("DELETE", f"/api/v1/alert-rules/{delete_rule_id}", 204)
            delete_rule_id = ""
            rules = api_json("/api/v1/alert-rules")
            self.assertNotIn(delete_rule["id"], {rule["id"] for rule in rules})
        finally:
            for rule_id in (disabled_rule_id, patched_rule_id, delete_rule_id):
                if rule_id:
                    delete_alert_rule(rule_id)

    def test_alert_rule_operators_duration_and_deduplication(self):
        target = wait_until("demo target registration", find_demo_target)
        rule_ids: list[str] = []

        try:
            operator_cases = [
                (">", "gt", -1),
                (">=", "gte", 0),
                ("<", "lt", 1_000_000),
                ("<=", "lte", 1_000_000),
            ]
            for raw_operator, normalized_operator, threshold in operator_cases:
                rule = post_json(
                    "/api/v1/alert-rules",
                    alert_rule_payload(
                        unique_test_name(f"operator {normalized_operator}"),
                        target["id"],
                        operator=raw_operator,
                        threshold=threshold,
                    ),
                )
                rule_ids.append(rule["id"])
                self.assertEqual(rule["operator"], normalized_operator)

            eq_rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(
                    unique_test_name("operator eq accepted"),
                    target["id"],
                    operator="==",
                    threshold=-1,
                    enabled=False,
                ),
            )
            rule_ids.append(eq_rule["id"])
            self.assertEqual(eq_rule["operator"], "eq")

            for rule_id in rule_ids[:4]:
                incident = wait_until(
                    f"incident for operator rule {rule_id}",
                    lambda rule_id=rule_id: (open_incidents_for_rule(rule_id) or [None])[0],
                )
                self.assertEqual(incident["target_id"], target["id"])

            duration_rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(
                    unique_test_name("duration cpu"),
                    target["id"],
                    threshold=0,
                    duration="15s",
                ),
            )
            rule_ids.append(duration_rule["id"])
            time.sleep(5)
            self.assertEqual(open_incidents_for_rule(duration_rule["id"]), [])
            wait_until(
                "duration-based incident after threshold window",
                lambda: (open_incidents_for_rule(duration_rule["id"]) or [None])[0],
            )

            dedup_rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(unique_test_name("dedup cpu"), target["id"], threshold=0, severity="critical"),
            )
            rule_ids.append(dedup_rule["id"])
            wait_until("dedup incident", lambda: (open_incidents_for_rule(dedup_rule["id"]) or [None])[0])
            wait_for_metric_batch()
            dedup_incidents = open_incidents_for_rule(dedup_rule["id"])
            self.assertEqual(len(dedup_incidents), 1)
        finally:
            for rule_id in rule_ids:
                delete_alert_rule(rule_id)

    def test_unknown_incident_id_returns_not_found(self):
        api_request("GET", "/api/v1/incidents/999999999", expected_status=404)

    def test_target_scoped_alert_rule_does_not_create_incident_for_demo_target(self):
        demo_target = wait_until("demo target registration", find_demo_target)
        targets = api_json("/api/v1/targets")
        other_target = next((target for target in targets if target.get("id") != demo_target["id"]), None)
        if other_target is None:
            self.skipTest("target-scoped alert e2e requires at least two discovered targets")

        rule_id = ""
        try:
            rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(unique_test_name("other target cpu"), other_target["id"], severity="warning"),
            )
            rule_id = rule["id"]

            def incident_for_other_target():
                incidents = open_incidents_for_rule(rule_id)
                if not incidents:
                    return None
                for incident in incidents:
                    self.assertEqual(incident["target_id"], other_target["id"])
                return incidents[0]

            wait_until("target-scoped incident for other target", incident_for_other_target)
            self.assertFalse(
                any(incident.get("target_id") == demo_target["id"] for incident in incidents_for_rule(rule_id))
            )
        finally:
            if rule_id:
                delete_alert_rule(rule_id)

    def test_demo_recovery_scenario_records_stop_incident_and_recovery_action(self):
        try:
            compose("start", DEMO_SERVICE)
        except AssertionError as exc:
            self.skipTest(f"Docker Compose is required for recovery e2e test: {exc}")

        target = wait_until("demo target registration", find_demo_target)
        resolve_open_incidents(target["id"], {"container_stopped", "container_died"})
        marker = datetime.now(timezone.utc)

        try:
            compose("stop", DEMO_SERVICE)

            def stopped_event():
                events = api_json("/api/v1/events", {"limit": 50})
                self.assertIsInstance(events, list)
                for event in events:
                    if (
                        event.get("target_id") == target["id"]
                        and event.get("event_type") == "container_stopped"
                        and parse_rfc3339(event["timestamp"]) >= marker
                    ):
                        return event
                return None

            event = wait_until("container_stopped event from demo scenario", stopped_event)

            def incident_for_event():
                incidents = api_json("/api/v1/incidents")
                self.assertIsInstance(incidents, list)
                for incident in incidents:
                    if (
                        incident.get("target_id") == target["id"]
                        and incident.get("rule_id") == "container_stopped"
                        and parse_rfc3339(incident["started_at"]) >= marker
                    ):
                        return incident
                return None

            incident = wait_until("incident for stopped demo container", incident_for_event)

            def recovery_for_incident():
                actions = api_json("/api/v1/recovery-actions")
                self.assertIsInstance(actions, list)
                for action in actions:
                    if (
                        action.get("incident_id") == incident["id"]
                        and action.get("target_id") == target["id"]
                        and action.get("action_type") == "restart_container"
                    ):
                        return action
                return None

            recovery = wait_until("restart_container recovery action", recovery_for_incident)

            self.assertEqual(event["severity"], "warning")
            self.assertEqual(incident["status"], "open")
            self.assertEqual(incident["severity"], event["severity"])
            self.assertIn(recovery["status"], {"succeeded", "failed", "skipped"})
        finally:
            compose("start", DEMO_SERVICE)

        def started_event():
            events = api_json("/api/v1/events", {"limit": 50})
            self.assertIsInstance(events, list)
            for event in events:
                if (
                    event.get("target_id") == target["id"]
                    and event.get("event_type") == "container_started"
                    and parse_rfc3339(event["timestamp"]) >= marker
                ):
                    return event
            return None

        start_event = wait_until("container_started event after demo cleanup", started_event)
        self.assertEqual(start_event["severity"], "info")

        def target_returns_to_ok():
            current = api_json(f"/api/v1/targets/{target['id']}")
            if current.get("status") == "OK":
                return current
            return None

        wait_until("demo target status returns to OK", target_returns_to_ok)

    def test_recovery_retry_for_failed_action_when_available(self):
        actions = api_json("/api/v1/recovery-actions")
        self.assertIsInstance(actions, list)
        failed_action = next((action for action in actions if action.get("status") == "failed"), None)
        if failed_action is None:
            self.skipTest("recovery retry e2e requires an existing failed recovery action")

        before = api_json("/api/v1/recovery-actions")
        send_status("POST", f"/api/v1/recovery-actions/{failed_action['id']}/retry", 202)

        def retried_action():
            after = api_json("/api/v1/recovery-actions")
            if len(after) <= len(before):
                return None
            for action in after:
                if (
                    action.get("id") != failed_action["id"]
                    and action.get("incident_id") == failed_action["incident_id"]
                    and action.get("target_id") == failed_action["target_id"]
                    and action.get("action_type") == failed_action["action_type"]
                ):
                    return action
            return None

        retry = wait_until("new recovery action after retry", retried_action)
        self.assertIn(retry["status"], {"running", "succeeded", "failed", "skipped"})

    def test_recovery_lock_does_not_leave_multiple_running_actions_for_target(self):
        try:
            compose("start", DEMO_SERVICE)
        except AssertionError as exc:
            self.skipTest(f"Docker Compose is required for recovery lock e2e test: {exc}")

        target = wait_until("demo target registration", find_demo_target)
        resolve_open_incidents(target["id"], {"container_stopped", "container_died"})
        marker = datetime.now(timezone.utc)
        try:
            compose("stop", DEMO_SERVICE)
            time.sleep(2)
            compose("stop", DEMO_SERVICE)
            wait_until("recovery action after repeated stop", lambda: open_incidents_for_target(target["id"]))
            actions = api_json("/api/v1/recovery-actions")
            recent_actions = [
                action
                for action in actions
                if action.get("target_id") == target["id"] and parse_rfc3339(action["started_at"]) >= marker
            ]
            running_actions = [action for action in recent_actions if action.get("status") == "running"]
            self.assertLessEqual(len(running_actions), 1)
        finally:
            compose("start", DEMO_SERVICE)

    def test_demo_stable_and_labels_targets_are_discovered_and_report_metrics(self):
        cases = [
            ("demo-nginx-stable", "http://localhost:8082"),
            ("demo-labels", "http://localhost:8084"),
        ]
        missing = [name for name, _ in cases if find_target_by_name(name) is None]
        if missing:
            self.skipTest(f"demo-targets override is not running; missing {', '.join(missing)}")

        for name, url in cases:
            self.assertTrue(http_ok(url))
            target = wait_until(f"{name} target registration", lambda name=name: find_target_by_name(name))
            details = api_json(f"/api/v1/targets/{target['id']}")
            self.assertIn(name, details["name"])

            def latest_metric():
                metrics = api_json("/api/v1/metrics/latest", {"target_id": target["id"], "limit": 5})
                self.assertIsInstance(metrics, list)
                return next((metric for metric in metrics if metric.get("target_id") == target["id"]), None)

            metric = wait_until(f"latest metric for {name}", latest_metric)
            self.assertTrue(metric["container_name"])

    def test_demo_cpu_and_memory_stress_alerts(self):
        cpu_target = find_target_by_name("demo-cpu-stress")
        memory_target = find_target_by_name("demo-memory-stress")
        if cpu_target is None or memory_target is None:
            self.skipTest("demo-targets override is not running; stress targets are missing")

        rule_ids: list[str] = []
        try:
            cpu_rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(
                    unique_test_name("demo cpu stress"),
                    cpu_target["id"],
                    threshold=10,
                    duration="10s",
                    severity="critical",
                ),
            )
            rule_ids.append(cpu_rule["id"])

            memory_rule = post_json(
                "/api/v1/alert-rules",
                alert_rule_payload(
                    unique_test_name("demo memory stress"),
                    memory_target["id"],
                    metric_name="memory_usage_bytes",
                    threshold=50_000_000,
                    duration="10s",
                    severity="warning",
                ),
            )
            rule_ids.append(memory_rule["id"])

            def cpu_metric_above_threshold():
                metrics = api_json("/api/v1/metrics/latest", {"target_id": cpu_target["id"], "limit": 5})
                metric = next((item for item in metrics if item.get("target_id") == cpu_target["id"]), None)
                if metric and metric.get("cpu_usage_percent", 0) >= 10:
                    return metric
                return None

            wait_until("CPU stress metric above threshold", cpu_metric_above_threshold)
            cpu_incident = wait_until(
                "CPU stress incident",
                lambda: (open_incidents_for_rule(cpu_rule["id"]) or [None])[0],
            )
            self.assertEqual(cpu_incident["severity"], "critical")

            def memory_metric_above_threshold():
                metrics = api_json("/api/v1/metrics/latest", {"target_id": memory_target["id"], "limit": 5})
                metric = next((item for item in metrics if item.get("target_id") == memory_target["id"]), None)
                if metric and metric.get("memory_usage_bytes", 0) >= 50_000_000:
                    return metric
                return None

            wait_until("memory stress metric above threshold", memory_metric_above_threshold)
            memory_incident = wait_until(
                "memory stress incident",
                lambda: (open_incidents_for_rule(memory_rule["id"]) or [None])[0],
            )
            self.assertEqual(memory_incident["severity"], "warning")
        finally:
            for rule_id in rule_ids:
                delete_alert_rule(rule_id)

    def test_demo_flaky_lifecycle_events_do_not_create_unlimited_open_incidents(self):
        flaky_target = find_target_by_name("demo-flaky")
        if flaky_target is None:
            self.skipTest("demo-targets override is not running; demo-flaky target is missing")

        def flaky_events():
            events = api_json("/api/v1/events", {"target_id": flaky_target["id"], "limit": 100})
            matched = [
                event
                for event in events
                if event.get("event_type") in {"container_died", "container_stopped", "container_started"}
            ]
            return matched or None

        events = wait_until("flaky lifecycle events", flaky_events, timeout=max(DEFAULT_TIMEOUT_SECONDS, 120))
        self.assertGreaterEqual(len(events), 1)
        open_incidents = open_incidents_for_target(flaky_target["id"])
        self.assertGreaterEqual(len(open_incidents), 1)
        counts: dict[str, int] = {}
        for incident in open_incidents:
            counts[incident["rule_id"]] = counts.get(incident["rule_id"], 0) + 1
        self.assertLessEqual(max(counts.values()), 1)

    def test_demo_manual_recovery_target_stop_creates_recovery_action(self):
        target = find_target_by_name("demo-manual-recovery")
        if target is None:
            self.skipTest("demo-targets override is not running; demo-manual-recovery target is missing")

        resolve_open_incidents(target["id"], {"container_stopped", "container_died"})
        marker = datetime.now(timezone.utc)
        try:
            demo_compose("stop", "demo-manual-recovery")

            def stopped_event():
                events = api_json("/api/v1/events", {"target_id": target["id"], "limit": 100})
                for event in events:
                    if (
                        event.get("event_type") == "container_stopped"
                        and parse_rfc3339(event["timestamp"]) >= marker
                    ):
                        return event
                return None

            wait_until("manual recovery stop event", stopped_event)

            def stopped_incident():
                incidents = open_incidents_for_target(target["id"])
                return next((incident for incident in incidents if incident.get("rule_id") == "container_stopped"), None)

            incident = wait_until("manual recovery stop incident", stopped_incident)

            def recovery_action():
                actions = api_json("/api/v1/recovery-actions")
                for action in actions:
                    if (
                        action.get("incident_id") == incident["id"]
                        and action.get("target_id") == target["id"]
                        and action.get("action_type") == "restart_container"
                    ):
                        return action
                return None

            recovery = wait_until("manual recovery action", recovery_action)
            self.assertIn(recovery["status"], {"succeeded", "failed", "skipped"})
        finally:
            demo_compose("start", "demo-manual-recovery")
