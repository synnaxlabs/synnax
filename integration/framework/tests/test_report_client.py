#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading

from framework import models
from framework.models import STATUS
from framework.report_client import ReportClient


def report(statuses: list[STATUS]) -> tuple[ReportClient, list[str]]:
    lines: list[str] = []

    def log(message: str, /, *, use_name: bool = True) -> None:
        lines.append(message)

    tests = [
        models.Test(
            test_name=f"driver/case_{i}",
            status=status,
            skip_reason="No hardware" if status == STATUS.SKIPPED else None,
        )
        for i, status in enumerate(statuses)
    ]
    client = ReportClient(
        tests=tests,
        tests_lock=threading.Lock(),
        test_definitions=[],
        active_tests=[],
        active_tests_lock=threading.Lock(),
        log=log,
    )
    return client, lines


class TestReportClient:
    def test_skipped_neither_passes_nor_fails(self) -> None:
        """Should count skipped tests apart from passed and failed tests."""
        client, _ = report([STATUS.PASSED, STATUS.SKIPPED, STATUS.SKIPPED])
        stats = client.get_statistics()
        assert stats["total"] == 3
        assert stats["passed"] == 1
        assert stats["skipped"] == 2
        assert stats["total_failed"] == 0

    def test_summary_shows_skip(self) -> None:
        """Should print the skip count and the reason of each skipped test."""
        client, lines = report([STATUS.PASSED, STATUS.SKIPPED])
        client.print_summary(None)
        assert "SKIPPED: No hardware" in lines
        assert "Skipped: 1" in lines

    def test_summary_hides_zero_skips(self) -> None:
        """Should not print a skip count when no test skipped."""
        client, lines = report([STATUS.PASSED])
        client.print_summary(None)
        assert not any(line.startswith("Skipped") for line in lines)
        assert client.get_statistics()["skipped"] == 0
