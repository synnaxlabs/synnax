#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from pathlib import Path
from typing import Any

import yaml

WORKFLOWS = Path(__file__).resolve().parents[2] / "workflows"


def load(path: Path) -> dict[Any, Any]:
    # YAML 1.1 reads the bare `on` key as True.
    workflow: dict[Any, Any] = yaml.safe_load(path.read_text())
    return workflow


def test_every_scheduled_workflow_reports_its_failures() -> None:
    workflows = [load(p) for p in WORKFLOWS.glob("*.y*ml")]
    scheduled = {w["name"] for w in workflows if "schedule" in (w.get(True) or {})}
    report = load(WORKFLOWS / "report.failures.yaml")
    assert set(report[True]["workflow_run"]["workflows"]) == scheduled


def test_self_reporting_workflows_are_watched() -> None:
    report = load(WORKFLOWS / "report.failures.yaml")
    env = report["jobs"]["report"]["steps"][0]["env"]
    self_reporting = set(json.loads(env["SELF_REPORTING"]))
    assert self_reporting <= set(report[True]["workflow_run"]["workflows"])
