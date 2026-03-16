#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import os
import random
import signal
import string
import sys
import threading
from collections.abc import Callable
from enum import Enum, auto
from typing import Any

import synnax as sy
from xpy import validate_and_sanitize_name

from framework.config_client import ConfigClient, Sequence, TestDefinition
from framework.execution_client import ExecutionClient
from framework.log_client import LogClient, LogMode, SynnaxChannelSink
from framework.models import Test
from framework.report_client import ReportClient
from framework.target_filter import TargetFilter, parse_target
from framework.telemetry_client import TelemetryClient
from framework.test_case import SynnaxConnection, TestCase


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

        self.config_client = ConfigClient(log=self.log_message)

        self.state = STATE.INITIALIZING
        self.test_definitions: list[TestDefinition] = []
        self.sequences: list[Sequence] = []
        self.should_stop = False
        self.status_callbacks: list[Any] = []
        self.tests: list[Test] = []
        self.tests_lock = threading.Lock()
        self.active_tests: list[tuple[TestCase, sy.Range, threading.Thread]] = []
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
            log=self.log_message,
            on_status_change=self._notify_status_change,
            on_test_ran=lambda: self.telemetry_client.tlm.__setitem__(
                self.telemetry_client._ch_test_cases_ran,
                self.telemetry_client.tlm[self.telemetry_client._ch_test_cases_ran] + 1,
            ),
        )

        self.report_client = ReportClient(
            tests=self.tests,
            tests_lock=self.tests_lock,
            test_definitions=self.test_definitions,
            active_tests=self.active_tests,
            active_tests_lock=self.active_tests_lock,
            log=self.log_message,
        )

        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        self.telemetry_client.start()
        sy.sleep(1)

    def log_message(self, message: str, use_name: bool = True) -> None:
        """Log message with real-time output."""
        if use_name:
            self.log_client.info(message)
        else:
            self.log_client.raw(message)

    def load(self, target_filter: TargetFilter) -> None:
        """Load test sequences using the config client."""
        self.state = STATE.LOADING
        self.sequences, self.test_definitions = self.config_client.load(target_filter)
        self.telemetry_client.tlm[self.telemetry_client._ch_test_case_count] = len(
            self.test_definitions
        )

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
        self.state = STATE.SHUTDOWN
        self.execution_client.should_stop = True
        self.should_stop = True

        if not self.telemetry_client.stop():
            self.log_message("Warning: telemetry thread did not stop within timeout")

        self.state = STATE.COMPLETED

    def shutdown(self) -> None:
        """Gracefully shutdown the test conductor and all its processes."""
        self.log_message("\nShut down initiated...")
        self.state = STATE.SHUTDOWN
        self.should_stop = True

        self.execution_client.stop()
        self.wait_for_completion()

        self.log_message("Shutdown complete\n")
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
                self.log_message(f"Error in status callback: {e}")

    def stop_sequence(self) -> None:
        """Stop the entire test sequence execution."""
        self.execution_client.stop()
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
        self.log_message(f"Received signal {signal_num}. Stopping test execution...")
        self.shutdown()
        sys.exit(1)


def monitor_test_execution(conductor: TestConductor) -> None:
    """Monitor test execution and provide status updates."""
    while conductor.is_running:
        status = conductor.get_current_status()
        conductor.log_message(
            f"Status: {status['completed_tests']}/{status['total_tests']} tests completed"
        )
        if status["active_tests"]:
            for test in status["active_tests"]:
                conductor.log_message(
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

-f flag — global case filter across all JSON files:
  uv run tc -f modbus               all *_tests.json, cases matching "modbus"
  uv run tc -f channel              all *_tests.json, cases matching "channel"

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
        help="Filter tests by case path substring across all test files (auto-discovers all *_tests.json)",
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
        elif args.filter:
            target_filter = TargetFilter(case_filter=args.filter)
        else:
            target_filter = TargetFilter()

        conductor.load(target_filter)
        results = conductor.run_sequence()
        conductor.wait_for_completion()

    except KeyboardInterrupt:
        conductor.log_message(
            "Keyboard interrupt received. Shutting down gracefully..."
        )
        conductor.shutdown()
    except Exception as e:
        conductor.log_message(f"Error occurred: {e}")
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
                conductor.log_message(
                    f"Warning: Failed to finalize conductor range: {e}"
                )

        conductor.log_message(f"Fin.")
        if hasattr(conductor, "tests") and conductor.tests:
            stats = conductor._get_test_statistics()

            if stats["total_failed"] > 0:
                conductor.log_message(
                    f"\nExiting with failure code due to {stats['total_failed']}/{stats['total']} failed tests"
                )
                os._exit(1)
            else:
                conductor.log_message(
                    f"\nAll {stats['total']} tests passed successfully", False
                )
                os._exit(0)
        else:
            conductor.log_message("\nNo test results available")
            os._exit(1)


if __name__ == "__main__":
    main()
