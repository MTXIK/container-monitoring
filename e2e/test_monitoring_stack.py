import json
import os
import subprocess
import time
import unittest
from datetime import datetime, timezone
from typing import Any
from urllib.error import URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


API_URL = os.getenv("E2E_API_URL", "http://localhost:8080").rstrip("/")
COMPOSE_FILE = os.getenv("E2E_COMPOSE_FILE", "docker-compose.yml")
DEMO_SERVICE = os.getenv("E2E_DEMO_SERVICE", "target-nginx")
POLL_INTERVAL_SECONDS = float(os.getenv("E2E_POLL_INTERVAL_SECONDS", "2"))
DEFAULT_TIMEOUT_SECONDS = float(os.getenv("E2E_TIMEOUT_SECONDS", "90"))


class APIError(AssertionError):
    pass


def api_json(path: str, query: dict[str, Any] | None = None) -> Any:
    url = f"{API_URL}{path}"
    if query:
        url = f"{url}?{urlencode(query)}"
    request = Request(url, headers={"Accept": "application/json"})
    with urlopen(request, timeout=5) as response:
        body = response.read().decode("utf-8")
        if response.status >= 400:
            raise APIError(f"GET {url} returned {response.status}: {body}")
        return json.loads(body)


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
