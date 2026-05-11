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
    targets = api_json("/api/v1/targets")
    assert isinstance(targets, list)
    for target in targets:
        name = target.get("name", "")
        if DEMO_SERVICE in name and target.get("source") == "docker":
            return target
    return None


def compose(*args: str) -> None:
    result = subprocess.run(
        ["docker", "compose", "-f", COMPOSE_FILE, *args],
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
