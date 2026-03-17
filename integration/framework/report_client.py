#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from collections.abc import Callable
from typing import Any

import synnax as sy

from framework.config_client import TestDefinition
from framework.models import STATUS, SYMBOLS, Test
from framework.test_case import TestCase


class ReportClient:
    """Collects test statistics and prints execution summaries."""

    def __init__(
        self,
        tests: list[Test],
        tests_lock: threading.Lock,
        test_definitions: list[TestDefinition],
        active_tests: list[tuple[TestDefinition, TestCase, sy.Range, threading.Thread]],
        active_tests_lock: threading.Lock,
        log: Callable[[str, bool], None],
    ) -> None:
        self._tests = tests
        self._tests_lock = tests_lock
        self._test_definitions = test_definitions
        self._active_tests = active_tests
        self._active_tests_lock = active_tests_lock
        self._log = log

    def get_statistics(self) -> dict[str, int]:
        """Calculate and return test execution statistics."""
        with self._tests_lock:
            if not self._tests:
                return {
                    "total": 0,
                    "passed": 0,
                    "flaky": 0,
                    "failed": 0,
                    "killed": 0,
                    "timeout": 0,
                    "total_failed": 0,
                }

            passed = sum(1 for r in self._tests if r.status == STATUS.PASSED)
            flaky = sum(1 for r in self._tests if r.status == STATUS.FLAKY)
            failed = sum(1 for r in self._tests if r.status == STATUS.FAILED)
            killed = sum(1 for r in self._tests if r.status == STATUS.KILLED)
            timeout = sum(1 for r in self._tests if r.status == STATUS.TIMEOUT)
            total_failed = failed + killed + timeout

            return {
                "total": len(self._tests),
                "passed": passed,
                "flaky": flaky,
                "failed": failed,
                "killed": killed,
                "timeout": timeout,
                "total_failed": total_failed,
            }

    def get_current_status(self, is_running: bool) -> dict[str, Any]:
        """Get the current status of test execution."""
        with self._active_tests_lock:
            active_snapshot = [
                {
                    "name": test_instance.__class__.__name__,
                    "elapsed_time": (
                        (sy.TimeStamp.now() - test_range.time_range.start)
                        / sy.TimeSpan.SECOND
                        if test_range is not None
                        else 0
                    ),
                }
                for _, test_instance, test_range, _ in self._active_tests
            ]

        with self._tests_lock:
            completed = len(self._tests)
            results = [
                {
                    "name": result.test_name,
                    "status": result.status.value,
                    "duration": (
                        (result.range.time_range.end - result.range.time_range.start)
                        / sy.TimeSpan.SECOND
                        if result.range is not None
                        and result.range.time_range.end != sy.TimeStamp.MAX
                        else None
                    ),
                    "error": result.error_message,
                }
                for result in self._tests
            ]

        return {
            "is_running": is_running,
            "total_tests": len(self._test_definitions),
            "completed_tests": completed,
            "active_tests": active_snapshot,
            "results": results,
        }

    def print_summary(self, conductor_range: sy.Range | None) -> None:
        """Print a summary of test execution results."""
        with self._tests_lock:
            if not self._tests:
                return
            tests_snapshot = list(self._tests)

        stats = self.get_statistics()

        self._log("\n" + "=" * 60, False)
        for test in tests_snapshot:
            if test.range is not None and test.range.time_range.end != sy.TimeStamp.MAX:
                duration = (
                    test.range.time_range.end - test.range.time_range.start
                ) / sy.TimeSpan.SECOND
                duration_str = f" ({duration:.1f}s)"
            else:
                duration_str = ""

            status_symbol = SYMBOLS.get_symbol(test.status)
            case_parts = str(test).split("/")
            display_name = (
                "/".join(case_parts[1:]) if len(case_parts) > 1 else str(test)
            )
            self._log(f"{status_symbol} {display_name}{duration_str}", False)
            if test.error_message:
                self._log(f"ERROR: {test.error_message}", True)

        self._log("=" * 60, False)
        self._log("TEST EXECUTION SUMMARY", False)

        self._log("=" * 60, False)
        self._log(f"Total tests: {stats['total']}", False)
        if conductor_range is not None:
            test_time_secs = (
                sy.TimeStamp.now() - conductor_range.time_range.start
            ) / sy.TimeSpan.SECOND
            minutes = int(test_time_secs // 60)
            seconds = int(test_time_secs % 60)
            self._log(f"Total time: {minutes}m {seconds}s", False)
        self._log(f"Passed: {stats['passed']}", False)
        if stats["flaky"] > 0:
            self._log(f"Flaky: {stats['flaky']} (passed on retry)", False)
        self._log(
            f"Failed: {stats['total_failed']} "
            f"(includes {stats['failed']} failed, "
            f"{stats['killed']} killed, {stats['timeout']} timeout)",
            False,
        )
        self._log("=" * 60, False)
        self._log("\n", False)
