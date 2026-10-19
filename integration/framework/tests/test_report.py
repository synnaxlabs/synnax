#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from pathlib import Path

from framework import models
from framework.failure_report import render_failure_report
from framework.models import STATUS
from framework.report_client import ReportClient


def flaky() -> models.Test:
    return models.Test(
        test_name="console/flaky",
        status=STATUS.FLAKY,
        error_message="first attempt failed",
    )


class TestFlaky:
    def test_counts_as_failed(self) -> None:
        """Should count a test that passed only on retry as failed."""
        tests = [flaky(), models.Test(test_name="console/ok", status=STATUS.PASSED)]
        report = ReportClient(
            tests=tests,
            tests_lock=threading.Lock(),
            test_definitions=[],
            active_tests=[],
            active_tests_lock=threading.Lock(),
            log=lambda _message, _flush: None,
        )
        stats = report.get_statistics()
        assert stats["flaky"] == 1
        assert stats["total_failed"] == 1

    def test_in_failure_report(self, tmp_path: Path) -> None:
        """Should list a flaky test with the failed attempt's error."""
        out = render_failure_report(tmp_path, [flaky()])
        assert out is not None
        text = out.read_text(encoding="utf-8")
        assert "console/flaky" in text
        assert "Passed on retry" in text
        assert "first attempt failed" in text
