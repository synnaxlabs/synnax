#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import io
import threading

from framework import models
from framework.models import STATUS, SYMBOLS
from framework.report_client import ReportClient

SERIAL_REASON = "Serial ports need a POSIX pseudo-terminal"


def create_tests() -> list[models.Test]:
    return [
        models.Test(test_name="driver/can_read", status=STATUS.PASSED),
        models.Test(
            test_name="driver/serial_read",
            status=STATUS.SKIPPED,
            skip_reason=SERIAL_REASON,
        ),
        models.Test(
            test_name="driver/tcp_read", status=STATUS.FAILED, error_message="boom"
        ),
    ]


def create_client(tests: list[models.Test], log: list[str]) -> ReportClient:
    return ReportClient(
        tests=tests,
        tests_lock=threading.Lock(),
        test_definitions=[],
        active_tests=[],
        active_tests_lock=threading.Lock(),
        log=lambda message, _: log.append(message),
    )


class TestStatistics:
    def test_skipped_apart_from_passed(self) -> None:
        """Should count a skipped test as neither passed nor failed."""
        stats = create_client(create_tests(), []).get_statistics()
        assert stats["total"] == 3
        assert stats["passed"] == 1
        assert stats["skipped"] == 1
        assert stats["total_failed"] == 1

    def test_no_tests(self) -> None:
        """Should report zero skipped tests when no test ran."""
        assert create_client([], []).get_statistics()["skipped"] == 0


class TestPrintSummary:
    def test_skip_reason(self) -> None:
        """Should mark a skipped test with its own symbol and print its reason."""
        log: list[str] = []
        create_client(create_tests(), log).print_summary(None)
        assert f"{SYMBOLS.SKIPPED.value} serial_read" in log
        assert f"SKIPPED: {SERIAL_REASON}" in log
        assert "Passed: 1" in log
        assert "Skipped: 1 (unsupported on this host)" in log

    def test_no_skipped_line(self) -> None:
        """Should leave out the skipped count when no test was skipped."""
        log: list[str] = []
        tests = [models.Test(test_name="driver/can_read", status=STATUS.PASSED)]
        create_client(tests, log).print_summary(None)
        assert not any(line.startswith("Skipped") for line in log)


class TestReportToGitHub:
    def test_notice_and_summary(self) -> None:
        """Should write a notice and a job summary table for each skipped test."""
        out, summary = io.StringIO(), io.StringIO()
        create_client(create_tests(), []).report_to_github(out, summary)
        assert out.getvalue() == (
            f"::notice title=Skipped test::driver/serial_read: {SERIAL_REASON}\n"
        )
        assert summary.getvalue() == (
            "### Skipped integration tests\n\n"
            "| Test | Reason |\n"
            "| --- | --- |\n"
            f"| driver/serial_read | {SERIAL_REASON} |\n\n"
        )

    def test_escape(self) -> None:
        """Should escape workflow command data and Markdown table cells."""
        out, summary = io.StringIO(), io.StringIO()
        tests = [
            models.Test(
                test_name="driver/a",
                status=STATUS.SKIPPED,
                skip_reason="100% | down\nnow",
            )
        ]
        create_client(tests, []).report_to_github(out, summary)
        assert out.getvalue() == (
            "::notice title=Skipped test::driver/a: 100%25 | down%0Anow\n"
        )
        assert "| driver/a | 100% \\| down now |" in summary.getvalue()

    def test_no_skips(self) -> None:
        """Should write nothing when no test was skipped."""
        out, summary = io.StringIO(), io.StringIO()
        tests = [models.Test(test_name="driver/can_read", status=STATUS.PASSED)]
        create_client(tests, []).report_to_github(out, summary)
        assert out.getvalue() == ""
        assert summary.getvalue() == ""

    def test_no_step_summary(self) -> None:
        """Should still write notices when the run has no job summary file."""
        out = io.StringIO()
        create_client(create_tests(), []).report_to_github(out, None)
        assert out.getvalue().startswith("::notice")
