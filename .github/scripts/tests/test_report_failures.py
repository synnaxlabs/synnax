#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path
from typing import Any

import pytest
import yaml

WORKFLOWS = Path(__file__).resolve().parents[2] / "workflows"
RELAY = "./.github/workflows/relay.failure.yaml"
SELF_REPORTING = {
    "Agent - Console QA",
    "Agent - Lint",
    "Agent - Nightly",
    "Check - Runners",
    "Test - HITL",
}
"""Scheduled workflows that post their own failures to Slack."""
REQUEUED = {
    "Agent - Bug hunt",
    "Agent - Console QA",
    "Agent - Lint",
    "Agent - Nightly",
    "Agent - Shepherd",
    "Agent - Task",
    "Dismiss - Sedosaur",
    "Dispatch - Agents",
    "Nightly",
    "Review - Sedosaur",
}
"""Workflows whose jobs can stop at the usage limit of an agent login."""


def load(path: Path) -> dict[Any, Any]:
    # YAML 1.1 reads the bare `on` key as True.
    workflow: dict[Any, Any] = yaml.safe_load(path.read_text())
    return workflow


ALL = [load(p) for p in sorted(WORKFLOWS.glob("*.y*ml"))]


def relays(workflow: dict[Any, Any]) -> list[tuple[str, dict[str, Any]]]:
    return [
        (name, job)
        for name, job in workflow["jobs"].items()
        if job.get("uses") == RELAY
    ]


def relayed(flag: str) -> set[str]:
    return {
        w["name"]
        for w in ALL
        for _, job in relays(w)
        if (job.get("with") or {}).get(flag)
    }


def test_every_scheduled_workflow_reports_its_failures() -> None:
    scheduled = {w["name"] for w in ALL if "schedule" in (w.get(True) or {})}
    assert scheduled >= SELF_REPORTING
    assert relayed("report") == scheduled - SELF_REPORTING


def test_only_agent_workflows_requeue_their_failures() -> None:
    assert relayed("requeue") == REQUEUED


@pytest.mark.parametrize(
    "workflow", [w for w in ALL if relays(w)], ids=lambda w: w["name"]
)
def test_the_relay_needs_every_other_job(workflow: dict[Any, Any]) -> None:
    [(name, job)] = relays(workflow)
    needs = job["needs"]
    assert set([needs] if isinstance(needs, str) else needs) == (
        workflow["jobs"].keys() - {name}
    )
    assert job["if"].startswith("always() && ")
    assert job["if"].endswith(" && contains(needs.*.result, 'failure')")
