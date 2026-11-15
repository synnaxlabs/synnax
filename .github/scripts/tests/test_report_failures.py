#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import yaml

WORKFLOWS = Path(__file__).resolve().parents[2] / "workflows"
# These post their own failures to Slack.
SELF_REPORTING = {"Check - Runners", "Test - HITL"}


def load(path: Path) -> dict:
    # YAML 1.1 reads the bare `on` key as True.
    return yaml.safe_load(path.read_text())


def test_every_scheduled_workflow_reports_its_failures() -> None:
    workflows = [load(p) for p in WORKFLOWS.glob("*.y*ml")]
    scheduled = {w["name"] for w in workflows if "schedule" in (w.get(True) or {})}
    report = load(WORKFLOWS / "report.failures.yaml")
    watched = set(report[True]["workflow_run"]["workflows"])
    assert scheduled - SELF_REPORTING == watched
