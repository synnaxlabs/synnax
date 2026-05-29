#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import json
import os
import random
import signal
import string
import sys
import threading
import time
from collections.abc import Callable
from datetime import datetime, timezone
from enum import Enum, auto
from typing import Any

import synnax as sy
from framework import failure_report, run_dir, server_log
from framework.config_client import ConfigClient, Sequence, TestDefinition
from framework.execution_client import ExecutionClient
from framework.log_client import LogClient, LogMode, SynnaxChannelSink
from framework.models import SynnaxConnection, Test
from framework.report_client import ReportClient
from framework.target_filter import TargetFilter, parse_target, split_csv
from framework.telemetry import TelemetryClient
from framework.test_case import TestCase
from x import validate_and_sanitize_name


class STATE(Enum):
    """Test conductor execution states."""

    INITIALIZING = auto()
    LOADING = auto()
    RUNNING = auto()
    CLEANUP = auto()
    ERROR = auto()
    SHUTDOWN = auto()
    COMPLETED = auto()


class TestConductor:
    """Manages execution of test sequences with timeout monitoring and result collection."""

    def __init__(
        self,
        name: str | None = None,
        synnax_connection: SynnaxConnection | None = None,
    ) -> None:
        """
        Initialize test conductor with connection parameters.
        """

        # Generate or validate name
        if name is None:
            random_id = "".join(
                random.choices(string.ascii_lowercase + string.digits, k=6)
            )
            self.name = self.__class__.__name__.lower() + "_" + random_id
        else:
            self.name = validate_and_sanitize_name(str(name).lower())

        # Establish the run directory before anything else writes artifacts.
        # Capture the server log's current size so that the run-scoped slice
        # at exit doesn't include lines from prior server sessions.
        self.run_dir = run_dir.init_run_dir(self.name)
        self.started_at = time.time()
        self._server_log_start_offset = server_log.current_offset()

        # Use provided connection or create default
        if synnax_connection is None:
            self.synnax_connection = SynnaxConnection()
        else:
            self.synnax_connection = synnax_connection

        # Initialize client
        try:
            self.client = self.synnax_connection.create_client()
        except Exception as e:
            raise RuntimeError(f"Failed to initialize client: {e}")

        self.log_client = LogClient(
            name=self.name,
            mode=LogMode.REALTIME,
            persistent_sinks=[
                SynnaxChannelSink(self.client, f"{self.name}_log"),
            ],
        )

        # Initialize range
        self.range: sy.Range | None = None
        try:
            self.range = self.client.ranges.create(
                name=self.name,
                time_range=sy.TimeRange(start=sy.TimeStamp.now(), end=sy.TimeStamp.MAX),
            )
        except Exception as e:
            raise RuntimeError(f"Failed to create range: {e}")

        self.config_client = ConfigClient(log=self.log)

        self.state = STATE.INITIALIZING
        self.test_definitions: list[TestDefinition] = []
        self.sequences: list[Sequence] = []
        self.should_stop = False
        self.status_callbacks: list[Any] = []
        self.tests: list[Test] = []
        self.tests_lock = threading.Lock()
        self.active_tests: list[
            tuple[TestDefinition, TestCase, sy.Range, threading.Thread]
        ] = []
        self.active_tests_lock = threading.Lock()

        self.telemetry_client = TelemetryClient(
            client=self.client,
            name=self.name,
            get_state=lambda: self.state,
            get_should_stop=lambda: self.should_stop,
        )

        self.execution_client = ExecutionClient(
            config_client=self.config_client,
            synnax_connection=self.synnax_connection,
            client=self.client,
            conductor_range=self.range,
            tests=self.tests,
            tests_lock=self.tests_lock,
            active_tests=self.active_tests,
            active_tests_lock=self.active_tests_lock,
            log=self.log,
            on_status_change=self._notify_status_change,
            on_test_ran=self.telemetry_client.increment_tests_ran,
        )

        self.report_client = ReportClient(
            tests=self.tests,
            tests_lock=self.tests_lock,
            test_definitions=self.test_definitions,
            active_tests=self.active_tests,
            active_tests_lock=self.active_tests_lock,
            log=self.log,
        )

        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        self.telemetry_client.start()
        sy.sleep(1)

    def log(self, message: str, use_name: bool = True) -> None:
        """Log message with real-time output."""
        if use_name:
            self.log_client.info(message)
        else:
            self.log_client.raw(message)

    def load(self, target_filter: TargetFilter) -> None:
        """Load test sequences using the config client."""
        self.state = STATE.LOADING
        sequences, new_defs = self.config_client.load(target_filter)
        self.sequences = sequences
        self.test_definitions.clear()
        self.test_definitions.extend(new_defs)
        self.telemetry_client.set_test_case_count(len(self.test_definitions))

    def run_sequence(self) -> list[Test]:
        """Execute all tests in the loaded sequence."""
        if not self.test_definitions:
            raise ValueError("No test sequence loaded. Call load() first.")

        self.state = STATE.RUNNING
        self.tests.clear()
        self.execution_client.run(self.sequences)
        self._print_summary()
        return self.tests

    def wait_for_completion(self) -> None:
        """Wait for all async processes to complete."""
        self.state = STATE.COMPLETED
        self.execution_client.should_stop = True
        self.should_stop = True

        if not self.telemetry_client.stop():
            self.log("Warning: telemetry thread did not stop within timeout")

    def shutdown(self) -> None:
        """Gracefully shutdown the test conductor and all its processes."""
        self.log("\nShut down initiated...")
        self.state = STATE.SHUTDOWN
        self.should_stop = True

        self.execution_client.stop()
        self.wait_for_completion()

        self.log("Shutdown complete\n")
        self.log_client.close()

    def add_status_callback(self, callback: Callable[[Test], None]) -> None:
        """Add a callback function to be called when test status changes."""
        self.status_callbacks.append(callback)

    def _notify_status_change(self, result: Test) -> None:
        """Notify all registered callbacks about status changes."""
        for callback in self.status_callbacks:
            try:
                callback(result)
            except Exception as e:
                self.log(f"Error in status callback: {e}")

    def stop_sequence(self) -> None:
        """Stop the entire test sequence execution."""
        self.execution_client.stop()
        self.should_stop = True
        self.telemetry_client.stop()

    @property
    def is_running(self) -> bool:
        return self.execution_client.is_running

    def get_current_status(self) -> dict[str, Any]:
        return self.report_client.get_current_status(self.execution_client.is_running)

    def _get_test_statistics(self) -> dict[str, int]:
        return self.report_client.get_statistics()

    def _print_summary(self) -> None:
        self.report_client.print_summary(self.range)

    def _signal_handler(self, signal_num: int, frame: Any = None) -> None:
        """Handle system signals for graceful shutdown."""
        self.log(f"Received signal {signal_num}. Stopping test execution...")
        self.shutdown()
        sys.exit(1)

    def finalize_bundle(self) -> None:
        """Write the run-scoped server log slice, summary.json, and README.

        Called at exit regardless of pass/fail; the bundle is the single
        artifact local devs and CI both consume to debug failures.
        """
        try:
            server_log.copy_range(
                self.run_dir / "server.log",
                self._server_log_start_offset,
            )
        except Exception as e:
            self.log(f"Warning: failed to capture server log: {e}")
        try:
            self._write_summary()
        except Exception as e:
            self.log(f"Warning: failed to write summary.json: {e}")
        try:
            (self.run_dir / "README.md").write_text(_BUNDLE_README)
        except Exception as e:
            self.log(f"Warning: failed to write README.md: {e}")
        report_path = failure_report.safe_render(self.run_dir, self.tests)
        if report_path is not None:
            self.log(f"→ failure report: {report_path}")

    def _write_summary(self) -> None:
        ended_at = time.time()
        stats = self.report_client.get_statistics() if self.tests else {}
        tests_json: list[dict[str, Any]] = []
        for t in self.tests:
            tests_json.append(
                {
                    "case": t.test_name,
                    "name": t.name,
                    "status": t.status.name,
                    "started_at": _iso(t.started_at),
                    "ended_at": _iso(t.ended_at),
                    "duration_s": _duration(t.started_at, t.ended_at),
                    "error_message": t.error_message,
                    "bundle_dir": (
                        os.path.relpath(t.bundle_dir, str(self.run_dir))
                        if t.bundle_dir
                        else None
                    ),
                }
            )
        summary = {
            "run": {
                "name": self.name,
                "started_at": _iso(self.started_at),
                "ended_at": _iso(ended_at),
                "duration_s": _duration(self.started_at, ended_at),
                **{
                    k: v
                    for k, v in stats.items()
                    if k
                    in (
                        "total",
                        "total_passed",
                        "total_failed",
                        "passed",
                        "failed",
                        "flaky",
                        "timeout",
                        "killed",
                    )
                },
            },
            "tests": tests_json,
        }
        (self.run_dir / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")


def _iso(ts: float | None) -> str | None:
    if ts is None:
        return None
    return datetime.fromtimestamp(ts, tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def _duration(start: float | None, end: float | None) -> float | None:
    if start is None or end is None:
        return None
    return round(end - start, 3)


_BUNDLE_README = """\
# Synnax Integration Test Debug Bundle

This directory contains everything needed to debug a failed `uv run tc` run.
The layout and inspection workflow are identical between local dev and CI.

## Triage order

1. **Read `all-failures.md`** (only present when there were failures). One
   markdown file that consolidates every failure: error, Python traceback,
   source snippet around the failing line, last actions from the trace,
   browser console errors, network failures, and tail of the server log slice.
   This is the LLM-friendly entry point. Read it first.
2. **Cross-reference `summary.json`** for run-level stats and per-test paths.
   ```bash
   jq '.tests[] | select(.status != "PASSED" and .status != "FLAKY")' summary.json
   ```
3. **Drill into `tests/<name>/`** for the raw artifacts behind a section:
   - `trace.zip` -> full Playwright trace (only on FAILED/TIMEOUT/KILLED)
   - `server.log` -> server log sliced to this test's wall-clock window
   - `*.png`, `*.json`, `*.csv` -> screenshots and exports the test produced
4. **Open the trace shell-first** when `all-failures.md` doesn't pinpoint it:
   ```bash
   npx -y -p @playwright/test playwright trace open tests/<name>/trace.zip
   npx -y -p @playwright/test playwright trace actions --errors-only
   npx -y -p @playwright/test playwright trace action <id>
   npx -y -p @playwright/test playwright trace requests --failed
   npx -y -p @playwright/test playwright trace close
   ```

## Detailed workflow

See the `synnax-integration-debug` skill (`.claude/skills/synnax-integration-debug/SKILL.md`
in the repo) for the full command cheatsheet, common gotchas, and triage heuristics.

## Bundle was generated by

`uv run tc` writing this directory at conductor exit. The breadcrumb line
`→ bundle: <path>` in conductor stdout points here.
"""


def monitor_test_execution(conductor: TestConductor) -> None:
    """Monitor test execution and provide status updates."""
    while conductor.is_running:
        status = conductor.get_current_status()
        conductor.log(
            f"Status: {status['completed_tests']}/{status['total_tests']} tests completed"
        )
        if status["active_tests"]:
            for test in status["active_tests"]:
                conductor.log(
                    f"Running: {test['name']} (elapsed: {test['elapsed_time']:.1f}s)"
                )
        sy.sleep(1)


def main() -> None:
    """Main entry point for the test conductor."""

    parser = argparse.ArgumentParser(
        description="Run test sequences",
        epilog="""
1-part path — run all tests in a file:
  uv run tc console                 all tests from console_tests.json
  uv run tc driver                  all tests from driver_tests.json

2-part path — file + case substring filter:
  uv run tc driver/modbus           driver_tests.json, cases matching "modbus"
  uv run tc console/label           console_tests.json, cases matching "label"
  uv run tc arc/lifecycle           arc_tests.json, cases matching "lifecycle"

3-part path — file + sequence filter + case filter:
  uv run tc console/channel/calc    sequence matching "channel", cases matching "calc"
  uv run tc arc/simple/...          sequence matching "simple", all cases

-f flag — case filter (global or combined with a target):
  uv run tc -f modbus               all *_tests.json, cases matching "modbus"
  uv run tc -f channel              all *_tests.json, cases matching "channel"
  uv run tc migration -f setup      migration_tests.json, cases matching "setup"
  uv run tc driver -f modbus        driver_tests.json, cases matching "modbus"

  -f matches against both the case path and the class name, so
  you can filter into individual test classes within a file:
  uv run tc driver/ni -f Scaled     only NIAnalogReadScaled from ni_read.py
  uv run tc migration -f verify     only Verify classes across migration tests

... wildcard — means "no filter" at that position:
  uv run tc console/...             same as just "console"
  uv run tc console/.../calc        sequence=all, case="calc"

All matching is case-insensitive substring.
        """,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "target",
        nargs="?",
        default=None,
        help="Target path (e.g., console/general/...)",
    )
    parser.add_argument("--name", default="tc", help="Test conductor name")
    parser.add_argument("--server", default="localhost", help="Synnax server address")
    parser.add_argument("--port", type=int, default=9090, help="Synnax server port")
    parser.add_argument("--username", default="synnax", help="Synnax username")
    parser.add_argument("--password", default="seldon", help="Synnax password")
    parser.add_argument("--secure", action="store_true", help="Use secure connection")
    parser.add_argument(
        "--filter",
        "-f",
        help="Filter tests by case path substring. Without a target, searches all *_tests.json. With a target, narrows results within that target.",
    )
    parser.add_argument(
        "--exclude",
        "-x",
        help="Exclude tests whose name matches this substring.",
    )
    parser.add_argument(
        "--headed",
        action="store_true",
        help="Run Playwright Console tests in headed mode (sets PLAYWRIGHT_CONSOLE_HEADED environment variable)",
    )
    parser.add_argument(
        "--driver",
        "-d",
        help="Driver rack name to use for driver tests (sets SYNNAX_DRIVER_RACK environment variable)",
    )

    args = parser.parse_args()

    os.environ["PLAYWRIGHT_CONSOLE_HEADED"] = "1" if args.headed else "0"
    if args.driver:
        os.environ["SYNNAX_DRIVER_RACK"] = args.driver

    # Create connection object
    connection = SynnaxConnection(
        server_address=args.server,
        port=args.port,
        username=args.username,
        password=args.password,
        secure=args.secure,
    )

    # Create and run test conductor
    conductor = TestConductor(
        name=args.name,
        synnax_connection=connection,
    )

    try:
        if args.target:
            target_filter = parse_target(args.target)
            if args.filter:
                target_filter.case_filter = split_csv(args.filter)
        elif args.filter:
            target_filter = TargetFilter(case_filter=split_csv(args.filter))
        else:
            target_filter = TargetFilter()

        if args.exclude:
            target_filter.exclude = split_csv(args.exclude)

        conductor.load(target_filter)
        conductor.run_sequence()
        conductor.wait_for_completion()

    except KeyboardInterrupt:
        conductor.log("Keyboard interrupt received. Shutting down gracefully...")
        conductor.shutdown()
    except Exception as e:
        conductor.log(f"Error occurred: {e}")
        conductor.shutdown()
        raise
    finally:
        # Update conductor range end time
        if conductor.range is not None:
            try:
                conductor.client.ranges.create(
                    key=conductor.range.key,
                    name=conductor.range.name,
                    time_range=sy.TimeRange(
                        start=conductor.range.time_range.start,
                        end=sy.TimeStamp.now(),
                    ),
                )
            except Exception as e:
                conductor.log(f"Warning: Failed to finalize conductor range: {e}")

        conductor.finalize_bundle()

        conductor.log("Fin.")
        if hasattr(conductor, "tests") and conductor.tests:
            stats = conductor._get_test_statistics()

            if stats["total_failed"] > 0:
                conductor.log(
                    f"\nExiting with failure code due to {stats['total_failed']}/{stats['total']} failed tests"
                )
                conductor.log(f"→ bundle: {conductor.run_dir}")
                os._exit(1)
            else:
                conductor.log(
                    f"\nAll {stats['total']} tests passed successfully", False
                )
                conductor.log(f"→ bundle: {conductor.run_dir}")
                os._exit(0)
        else:
            conductor.log("\nNo test results available")
            conductor.log(f"→ bundle: {conductor.run_dir}")
            os._exit(1)


if __name__ == "__main__":
    main()
