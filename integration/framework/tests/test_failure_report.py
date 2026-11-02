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


def _render(
    tmp_path: Path,
    clicks: list[tuple[str, str]],
    *,
    stacks: str = "",
    lines: tuple[str, ...] = (),
    network: tuple[str, ...] = (),
) -> str:
    """Render the report for a failed test whose trace has one click per (selector,
    calling method). A non-empty ``stacks`` replaces the generated trace.stacks.
    ``lines`` adds raw lines to trace.trace, and ``network`` is trace.network."""
    events: list[dict[str, object]] = []
    frames: list[list[object]] = []
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
        frames.append([call_id, [[0, 1, 0, caller]]])
    bundle = tmp_path / "tests" / "case"
    bundle.mkdir(parents=True)
    with zipfile.ZipFile(bundle / "trace.zip", "w") as z:
        trace = [json.dumps(e) for e in events]
        z.writestr("trace.trace", "\n".join([*trace, *lines]))
        z.writestr("trace.network", "\n".join(network))
        z.writestr(
            "trace.stacks",
            stacks or json.dumps({"files": ["case.py"], "stacks": frames}),
        )
    test = models.Test(
        test_name="console/case",
        status=models.STATUS.FAILED,
        bundle_dir=str(bundle),
    )
    report = failure_report.safe_render(tmp_path, tests=[test])
    assert report is not None
    return report.read_text(encoding="utf-8")


class TestFailureReport:
    """The report lists the last trace actions before a failure."""

    def test_should_leave_teardown_actions_out_of_the_last_actions(
        self, tmp_path: Path
    ) -> None:
        text = _render(tmp_path, [("#run", "Case.run"), ("#cleanup", "Case.teardown")])
        assert "selector=#run" in text
        assert "selector=#cleanup" not in text

    def test_should_list_every_action_when_the_stacks_are_unreadable(
        self, tmp_path: Path
    ) -> None:
        text = _render(
            tmp_path,
            [("#run", "Case.run"), ("#cleanup", "Case.teardown")],
            stacks="{",
        )
        assert "selector=#run" in text
        assert "selector=#cleanup" in text

    def test_should_list_console_errors_and_failed_requests(
        self, tmp_path: Path
    ) -> None:
        console = {"type": "console", "messageType": "error", "text": "boom"}
        failed = {
            "snapshot": {
                "request": {"method": "GET", "url": "http://core/api"},
                "response": {"status": 500, "statusText": "Server Error"},
            }
        }
        ok = {"snapshot": {"response": {"status": 200}}}
        text = _render(
            tmp_path,
            [("#run", "Case.run")],
            lines=(json.dumps(console),),
            network=(json.dumps(failed), json.dumps(ok)),
        )
        assert "boom" in text
        assert "500 Server Error" in text
        assert "GET http://core/api" in text
        assert text.count("GET ") == 1

    def test_should_skip_trace_lines_it_cannot_read(self, tmp_path: Path) -> None:
        text = _render(
            tmp_path,
            [("#run", "Case.run")],
            lines=("{", "[1, 2]", json.dumps({"type": "before", "startTime": "x"})),
            network=("{", json.dumps({"snapshot": {"response": {"status": "x"}}})),
        )
        assert "selector=#run" in text
