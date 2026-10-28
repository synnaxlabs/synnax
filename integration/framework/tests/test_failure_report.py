#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import zipfile
from pathlib import Path

from framework import failure_report, models


def _write_trace(path: Path, clicks: list[tuple[str, str]]) -> None:
    """Write a trace.zip with one click per (selector, calling method)."""
    events: list[dict[str, object]] = []
    stacks: list[list[object]] = []
    for i, (selector, caller) in enumerate(clicks):
        call_id = f"call@{i}"
        events.append(
            {
                "type": "before",
                "callId": call_id,
                "startTime": float(i),
                "class": "Frame",
                "method": "click",
                "params": {"selector": selector},
            }
        )
        events.append({"type": "after", "callId": call_id, "endTime": i + 0.5})
        stacks.append([call_id, [[0, 1, 0, caller]]])
    with zipfile.ZipFile(path, "w") as z:
        z.writestr("trace.trace", "\n".join(json.dumps(e) for e in events))
        z.writestr("trace.stacks", json.dumps({"files": ["case.py"], "stacks": stacks}))


class TestFailureReport:
    def test_should_leave_teardown_actions_out_of_the_last_actions(
        self, tmp_path: Path
    ) -> None:
        bundle = tmp_path / "tests" / "case"
        bundle.mkdir(parents=True)
        _write_trace(
            bundle / "trace.zip",
            [("#run", "Case.run"), ("#cleanup", "Case.teardown")],
        )
        test = models.Test(
            test_name="console/case",
            status=models.STATUS.FAILED,
            bundle_dir=str(bundle),
        )
        report = failure_report.safe_render(tmp_path, [test])
        assert report is not None
        text = report.read_text(encoding="utf-8")
        assert "selector=#run" in text
        assert "selector=#cleanup" not in text
