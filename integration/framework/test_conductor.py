#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import ctypes
import os
import random
import signal
import string
import sys
import threading
import traceback
from collections.abc import Callable
from dataclasses import dataclass
from enum import Enum, auto
from typing import Any

import synnax as sy

from framework.config_client import ConfigClient, Sequence, TestDefinition
from framework.log_client import LogClient, LogMode, SynnaxChannelSink
from framework.report_client import ReportClient
from framework.target_filter import TargetFilter, parse_target
from framework.telemetry_client import TelemetryClient
from framework.test_case import STATUS, SYMBOLS, SynnaxConnection, TestCase
from framework.utils import validate_and_sanitize_name


class STATE(Enum):
    """Test conductor execution states."""

    INITIALIZING = auto()
    LOADING = auto()
    RUNNING = auto()
    CLEANUP = auto()
    ERROR = auto()
    SHUTDOWN = auto()
    COMPLETED = auto()


@dataclass
class Test:
    """Data class to store test execution results."""

    test_name: str
    status: STATUS
    name: str | None = None  # Custom name from test definition
    error_message: str | None = None
    range: sy.Range | None = None

    def __str__(self) -> str:
        """Return display name for test result."""
        if self.name and self.name != self.test_name.split("/")[-1]:
            return f"{self.test_name} ({self.name})"
        return self.test_name


COLORS: list[str] = [
    "#001833",  # Dark Sky Blue (210°)
    "#003333",  # Dark Cyan (180°)
    "#003318",  # Dark Spring Green (150°)
    "#223322",  # Dark Green (120°)
    "#183318",  # Dark Lime (90°)
    "#333300",  # Dark Yellow (60°)
    "#331800",  # Dark Orange (30°)
    "#330000",  # Dark Red (0°)
    "#330018",  # Dark Rose (330°)
    "#330033",  # Dark Magenta (300°)
    "#180033",  # Dark Purple (270°)
    "#000033",  # Dark Blue (240°)
]


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
        self.timeout_monitor_thread: threading.Thread | None = None
        self.is_running = False
        self.should_stop = False
        self.status_callbacks: list[Any] = []
        self.active_tests: list[tuple[TestCase, sy.Range, threading.Thread]] = []
        self.active_tests_lock = threading.Lock()
        self.tests_lock = threading.Lock()

        self.telemetry_client = TelemetryClient(
            client=self.client,
            name=self.name,
            get_state=lambda: self.state,
            get_should_stop=lambda: self.should_stop,
        )

        self.report_client = ReportClient(
            tests=[],
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
        self.telemetry_client.tlm[f"{self.name}_test_case_count"] = len(self.test_definitions)

    def run_sequence(self) -> list[Test]:
        """Execute all tests in the loaded sequence."""
        if not self.test_definitions:
            raise ValueError("No test sequence loaded. Call load() first.")

        self.state = STATE.RUNNING
        self.is_running = True
        self.should_stop = False
        self.tests: list[Test] = []
        self.report_client._tests = self.tests

        # Start timeout monitoring
        self.timeout_monitor_thread = threading.Thread(
            target=self._timeout_monitor_thread,
            args=(1.0,),  # Check every 1 second
            daemon=True,
        )
        self.timeout_monitor_thread.start()

        self.log_message(
            f"Starting execution of {len(self.sequences)} sequences with {len(self.test_definitions)} total tests...\n"
        )

        # Execute sequences linearly (one after another)
        for seq_idx, sequence in enumerate(self.sequences):
            if self.should_stop:
                self.log_message("Test execution stopped by user request")
                break

            self.log_message(
                f"==== SEQUENCE {seq_idx + 1}/{len(self.sequences)}: {sequence.name} ===="
            )
            self.log_message(
                f"Executing {len(sequence.tests)} tests with {sequence.order} order...\n"
            )

            tests_to_execute = sequence.tests.copy()

            if sequence.order == "asynchronous":
                self._execute_sequence_asynchronously(
                    sequence.name, tests_to_execute, sequence.pool_size
                )
            else:
                if sequence.order == "random":
                    random.shuffle(tests_to_execute)
                    self.log_message("Tests randomized for execution")
                self._execute_sequence(tests_to_execute)

            self.log_message(f"Completed sequence '{sequence.name}'\n")

        self.is_running = False
        self._print_summary()
        return self.tests

    def _execute_sequence(self, tests_to_execute: list[TestDefinition]) -> None:
        """Execute tests in a sequence one after another."""
        for test_def in tests_to_execute:
            if self.should_stop:
                self.log_message("Test execution stopped by user request")
                break

            # Calculate global test index
            global_test_idx = len(self.tests) + 1
            self.log_message(
                f"[{global_test_idx}/{len(self.test_definitions)}] {test_def}"
            )

            # Run test in separate thread
            result_container: list[Test] = []
            test_thread = threading.Thread(
                target=self._test_runner_thread, args=(test_def, result_container)
            )

            test_thread.start()
            test_thread.join()

            # Get test result
            if result_container:
                test_result = result_container[0]
            else:
                test_result = Test(
                    test_name=test_def.case,
                    name=test_def.name or test_def.case.split("/")[-1],
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned",
                )

            with self.tests_lock:
                self.tests.append(test_result)
            self.telemetry_client.tlm[f"{self.name}_test_cases_ran"] += 1

    def _execute_sequence_asynchronously(
        self,
        sequence_name: str,
        tests_to_execute: list[TestDefinition],
        pool_size: int = -1,
    ) -> None:
        """Execute tests in a sequence with optional concurrency limit.

        Args:
            sequence_name: Name of the sequence being executed
            tests_to_execute: List of test definitions to execute
            pool_size: Maximum number of concurrent tests. If -1 or greater than
                      the number of tests, all tests run concurrently.
        """
        # If pool_size is -1 or >= number of tests, use unlimited concurrency
        if pool_size <= 0 or pool_size >= len(tests_to_execute):
            self._execute_unlimited_async(sequence_name, tests_to_execute)
        else:
            self._execute_pooled_async(sequence_name, tests_to_execute, pool_size)

    def _execute_unlimited_async(
        self, sequence_name: str, tests_to_execute: list[TestDefinition]
    ) -> None:
        """Execute all tests concurrently without limit."""
        test_threads = []
        test_containers = []

        for i, test_def in enumerate(tests_to_execute):
            if self.should_stop:
                self.log_message("Test execution stopped by user request")
                break

            # Calculate global test index - each test gets a unique index
            global_test_idx = len(self.tests) + i + 1
            self.log_message(
                f"[{global_test_idx}/{len(self.test_definitions)}] {test_def}"
            )

            # Create result container and thread for each test
            result_container: list[Test] = []
            test_thread = threading.Thread(
                target=self._test_runner_thread, args=(test_def, result_container)
            )

            test_threads.append(test_thread)
            test_containers.append(result_container)

            # Start the test thread
            test_thread.start()

        # Wait for all tests in this sequence to complete
        self.log_message(
            f"Waiting for {len(test_threads)} tests in sequence '{sequence_name}' to complete..."
        )
        for i, test_thread in enumerate(test_threads):
            if test_thread.is_alive():
                test_thread.join()

            # Get test result
            if test_containers[i]:
                test_result = test_containers[i][0]
            else:
                test_result = Test(
                    test_name=tests_to_execute[i].case,
                    name=tests_to_execute[i].name,
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned",
                )

            with self.tests_lock:
                self.tests.append(test_result)
            self.telemetry_client.tlm[f"{self.name}_test_cases_ran"] += 1

    def _execute_pooled_async(
        self, sequence_name: str, tests_to_execute: list[TestDefinition], pool_size: int
    ) -> None:
        """Execute tests with a limited concurrency pool.

        Args:
            sequence_name: Name of the sequence being executed
            tests_to_execute: List of test definitions to execute
            pool_size: Maximum number of concurrent tests
        """
        self.log_message(
            f"Running tests with pool size of {pool_size} (max {pool_size} concurrent tests)..."
        )

        semaphore = threading.Semaphore(pool_size)
        test_threads = []
        test_containers = []

        def run_with_semaphore(
            test_def: TestDefinition, result_container: list[Test], test_idx: int
        ) -> None:
            """Wrapper to run test with semaphore control."""
            semaphore.acquire()
            try:
                self.log_message(
                    f"[{test_idx}/{len(self.test_definitions)}] {test_def}"
                )
                self._test_runner_thread(test_def, result_container)
            finally:
                semaphore.release()

        # Create and start all threads (semaphore will control execution)
        for i, test_def in enumerate(tests_to_execute):
            if self.should_stop:
                self.log_message("Test execution stopped by user request")
                break

            global_test_idx = len(self.tests) + i + 1
            result_container: list[Test] = []
            test_thread = threading.Thread(
                target=run_with_semaphore,
                args=(test_def, result_container, global_test_idx),
            )

            test_threads.append(test_thread)
            test_containers.append(result_container)

            test_thread.start()

        self.log_message(
            f"Waiting for {len(test_threads)} tests in sequence '{sequence_name}' to complete..."
        )
        for i, test_thread in enumerate(test_threads):
            if test_thread.is_alive():
                test_thread.join()

            # Get test result
            if test_containers[i]:
                test_result = test_containers[i][0]
            else:
                test_result = Test(
                    test_name=tests_to_execute[i].case,
                    name=tests_to_execute[i].name,
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned",
                )

            with self.tests_lock:
                self.tests.append(test_result)
            self.telemetry_client.tlm[f"{self.name}_test_cases_ran"] += 1

    def wait_for_completion(self) -> None:
        """
        Wait for all async processes to complete before allowing main to exit.
        This ensures proper cleanup and prevents premature termination.
        """
        self.state = STATE.SHUTDOWN
        self.should_stop = True

        if not self.telemetry_client.stop():
            self.log_message("Warning: telemetry thread did not stop within timeout")

        # Wait for timeout monitor to finish
        if self.timeout_monitor_thread and self.timeout_monitor_thread.is_alive():
            self.timeout_monitor_thread.join(timeout=5.0)
            if self.timeout_monitor_thread.is_alive():
                self.log_message(
                    "Warning: timeout_monitor_thread did not stop within timeout"
                )

        self.state = STATE.COMPLETED

    def shutdown(self) -> None:
        """
        Gracefully shutdown the test conductor and all its processes.
        """
        self.log_message("\nShut down initiated...")
        self.state = STATE.SHUTDOWN
        self.should_stop = True

        killed = self.kill_active_tests()
        if killed > 0:
            self.log_message(f"Killed {killed} active test(s)")

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

    def _execute_single_test(self, test_def: TestDefinition) -> Test:
        """Execute a single test case."""
        test = Test(
            test_name=test_def.case,
            name=test_def.name or test_def.case.split("/")[-1],
            status=STATUS.PENDING,
        )

        # Get color for this test based on its index
        test_index = len(self.tests)
        color = COLORS[test_index % len(COLORS)]

        # Create range for this test case (MAX = in progress)
        if self.range is not None:
            test.range = self.range.create_child_range(
                name=test.name or test.test_name,
                time_range=sy.TimeRange(start=sy.TimeStamp.now(), end=sy.TimeStamp.MAX),
                color=color,
            )
        else:
            test.range = None

        test_instance: TestCase | None = None
        try:
            test_class = self.config_client.load_test_class(test_def)
            test_instance = test_class(
                synnax_connection=self.synnax_connection,
                name=test_def.name or test_def.case.split("/")[-1],
                **test_def.parameters,
            )

            # Track test for timeout monitoring
            current_thread = threading.current_thread()
            with self.active_tests_lock:
                self.active_tests.append((test_instance, test.range, current_thread))

            test.status = STATUS.RUNNING
            self._notify_status_change(test)

            # Execute the test
            test_instance.execute()
            test.status = test_instance._status

        except Exception as e:
            test.status = STATUS.FAILED
            test.error_message = str(e)
            self.log_message(f"{test_def.case} FAILED: {e}")

            self.log_message(f"Traceback: {traceback.format_exc()}")

        finally:
            if test.range is not None:
                try:
                    test.range = self._finalize_range(test.range)
                except RuntimeError as e:
                    self.log_message(f"Warning: Could not finalize range: {e}")

            # Dump buffered test logs on failure, discard on success
            if test_instance is not None:
                if test.status in (STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED):
                    self.log_message(f"--- Logs for {test_def} ---")
                    test_instance.log_client.dump()
                    self.log_message(f"--- End logs for {test_def} ---")
                else:
                    test_instance.log_client.discard()

            # Clean up test tracking
            with self.active_tests_lock:
                self.active_tests = [
                    (test, test_range, thread)
                    for test, test_range, thread in self.active_tests
                    if test != test_instance
                ]

            self._notify_status_change(test)

        return test

    def _finalize_range(self, test_range: sy.Range) -> sy.Range:
        """Finalize a test range by updating its end time."""
        try:
            return self.client.ranges.create(
                key=test_range.key,
                name=test_range.name,
                time_range=sy.TimeRange(
                    start=test_range.time_range.start,
                    end=sy.TimeStamp.now(),
                ),
            )
        except Exception as e:
            self.log_client.error(f"Failed to finalize range: {e}")
            raise RuntimeError(
                f"Failed to finalize range '{test_range.name}': {e}"
            ) from e

    def _test_runner_thread(
        self, test_def: TestDefinition, result_container: list[Test]
    ) -> None:
        """Thread function for running a single test."""
        result = self._execute_single_test(test_def)
        result_container.append(result)

    def _timeout_monitor_thread(
        self, monitor_interval: sy.CrudeTimeSpan = 500 * sy.TimeSpan.MILLISECOND
    ) -> None:
        """Monitor test execution for timeout violations."""
        while self.is_running and not self.should_stop:
            self._check_test_timeouts()
            sy.sleep(monitor_interval)

    def _check_test_timeouts(self) -> None:
        """Check all active tests for timeout violations."""
        with self.active_tests_lock:
            if not self.active_tests:
                return

            tests_to_remove = []
            for test_instance, test_range, thread in self.active_tests:
                expected_timeout: sy.CrudeTimeSpan | None = getattr(
                    test_instance, "Expected_Timeout", None
                )
                if expected_timeout is None:
                    continue

                elapsed = sy.TimeStamp.now() - test_range.time_range.start
                timeout_span = sy.TimeSpan.from_seconds(expected_timeout)
                if elapsed <= timeout_span:
                    continue

                self._handle_test_timeout(test_instance, elapsed, timeout_span)
                tests_to_remove.append((test_instance, test_range, thread))

            for test_tuple in tests_to_remove:
                self.active_tests.remove(test_tuple)

    def _handle_test_timeout(
        self,
        test_instance: TestCase,
        elapsed: sy.TimeSpan,
        timeout: sy.TimeSpan,
    ) -> None:
        """Handle a single test timeout."""
        self.log_message(
            f"{test_instance.name} timeout detected ({elapsed} > {timeout})"
        )
        test_instance._status = STATUS.TIMEOUT

    def _determine_kill_status(
        self, test_instance: TestCase, test_range: sy.Range
    ) -> tuple[STATUS, str]:
        """Determine if a test should be marked as TIMEOUT or KILLED."""
        elapsed_time = (
            sy.TimeStamp.now() - test_range.time_range.start
        ) / sy.TimeSpan.SECOND
        expected_timeout = getattr(test_instance, "Expected_Timeout", -1)

        if expected_timeout > 0 and elapsed_time > expected_timeout:
            self.log_message(
                f"Test {test_instance.name} exceeded timeout ({expected_timeout}s)"
            )
            return STATUS.TIMEOUT, f"Test exceeded timeout ({expected_timeout}s)"

        self.log_message(f"Test {test_instance.name} was manually killed")
        return STATUS.KILLED, "Test was manually killed"

    def _terminate_thread(self, thread: threading.Thread) -> None:
        """Force terminate a thread using ctypes."""
        thread.join(timeout=0.1)
        if not thread.is_alive():
            return

        try:
            thread_id = thread.ident
            if thread_id is None:
                return

            res = ctypes.pythonapi.PyThreadState_SetAsyncExc(
                ctypes.c_long(thread_id), ctypes.py_object(SystemExit)
            )
            if res == 0:
                self.log_message(f"Warning: Could not terminate thread {thread.name}")
            elif res > 1:
                # Revert if it affected multiple threads
                ctypes.pythonapi.PyThreadState_SetAsyncExc(thread_id, None)
        except Exception as e:
            self.log_message(f"Warning: Failed to force-terminate thread: {e}")

    def kill_active_tests(self) -> int:
        """Kill all active tests by terminating their threads."""
        with self.active_tests_lock:
            if not self.active_tests:
                return 0

            killed_test_results = []
            threads_to_terminate = []
            killed_instances = []

            for test_instance, test_range, thread in self.active_tests:
                if test_range is None:
                    status = STATUS.KILLED
                    error_msg = "Test was killed (no range available)"
                    finalized_range = None
                else:
                    status, error_msg = self._determine_kill_status(
                        test_instance, test_range
                    )
                    try:
                        finalized_range = self._finalize_range(test_range)
                    except RuntimeError as e:
                        self.log_message(
                            f"Warning: Could not finalize range for killed test: {e}"
                        )
                        finalized_range = test_range  # Keep original range with end=MAX

                test_instance._status = status
                test_result = Test(
                    test_name=test_instance.name,
                    name=getattr(test_instance, "custom_name", None),
                    status=status,
                    error_message=error_msg,
                    range=finalized_range,
                )
                killed_test_results.append(test_result)
                threads_to_terminate.append(thread)
                killed_instances.append(test_instance)

            self.active_tests = []

        with self.tests_lock:
            self.tests.extend(killed_test_results)

        # Terminate threads outside the lock
        for thread in threads_to_terminate:
            self._terminate_thread(thread)

        # Dump buffered logs for killed tests
        for instance in killed_instances:
            self.log_message(f"--- Logs for {instance.name} ---")
            instance.log_client.dump()
            self.log_message(f"--- End logs for {instance.name} ---")

        return len(killed_test_results)

    def stop_sequence(self) -> None:
        """Stop the entire test sequence execution."""
        self.log_message("Stopping test sequence...")
        self.should_stop = True
        killed = self.kill_active_tests()
        if killed > 0:
            self.log_message(f"Killed {killed} active test(s)")
        self.telemetry_client.stop()

    def get_current_status(self) -> dict[str, Any]:
        return self.report_client.get_current_status(self.is_running)

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
