#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import ctypes
import glob
import importlib.util
import itertools
import json
import logging
import os
import random
import signal
import string
import sys
import threading
import traceback
from collections.abc import Callable
from dataclasses import dataclass, field
from enum import Enum, auto
from pathlib import Path
from typing import Any, cast

import synnax as sy

from framework.test_case import STATUS, SYMBOLS, SynnaxConnection, TestCase
from framework.utils import is_ci, validate_and_sanitize_name


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


@dataclass
class TestDefinition:
    """Data class representing a test case definition from the sequence file."""

    case: str
    name: str | None = None  # Optional custom name for the test case
    parameters: dict[str, Any | list[Any]] = field(default_factory=dict)
    expect: str = "PASSED"  # Expected test outcome, defaults to "PASSED"
    matrix: dict[str, list[Any]] | None = None  # Matrix of params to expand

    def __str__(self) -> str:
        """Return display name for test definition."""
        if self.name and self.name != self.case.split("/")[-1]:
            return f"{self.case} ({self.name})"
        return self.case


COLORS: list[str] = [
    "#FF0000",  # Red (0°)
    "#FF8000",  # Orange (30°)
    "#FFFF00",  # Yellow (60°)
    "#80FF80",  # Lime (90°)
    "#AAFFAA",  # Green (120°)
    "#00FF80",  # Spring Green (150°)
    "#00FFFF",  # Cyan (180°)
    "#0080FF",  # Sky Blue (210°)
    "#0000FF",  # Blue (240°)
    "#8000FF",  # Purple (270°)
    "#FF00FF",  # Magenta (300°)
    "#FF0080",  # Rose (330°)
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

        # Configure logging for real-time output in CI
        self._setup_logging()

        # Use provided connection or create default
        if synnax_connection is None:
            self.synnax_connection = SynnaxConnection()
        else:
            self.synnax_connection = synnax_connection

        # Initialize client
        try:
            self.client = sy.Synnax(
                host=self.synnax_connection.server_address,
                port=self.synnax_connection.port,
                username=self.synnax_connection.username,
                password=self.synnax_connection.password,
                secure=self.synnax_connection.secure,
            )
        except Exception as e:
            raise RuntimeError(f"Failed to initialize client: {e}")

        # Initialize range
        self.range: sy.Range | None = None
        try:
            self.range = self.client.ranges.create(
                name=self.name,
                time_range=sy.TimeRange(start=sy.TimeStamp.now(), end=sy.TimeStamp.MAX),
            )
        except Exception as e:
            raise RuntimeError(f"Failed to create range: {e}")

        self.state = STATE.INITIALIZING
        self.test_definitions: list[TestDefinition] = []
        self.sequences: list[dict[str, Any]] = []
        self.timeout_monitor_thread: threading.Thread | None = None
        self.client_manager_thread: threading.Thread | None = None
        self.is_running = False
        self.should_stop = False
        self.status_callbacks: list[Callable[[Test], None]] = []
        # Track active tests
        self.active_tests: list[tuple[TestCase, sy.Range, threading.Thread]] = []
        self.active_tests_lock = threading.Lock()
        self.tests_lock = threading.Lock()
        self.import_lock = threading.Lock()

        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        # Start client manager
        self._start_client_manager_async()
        sy.sleep(1)  # Allow client manager to start

    def _start_client_manager_async(self) -> None:
        """Start client manager in separate daemon thread."""
        self.client_manager_thread = threading.Thread(
            target=self._client_manager, daemon=True, name=f"{self.name}_client_manager"
        )
        self.client_manager_thread.start()
        self.log_message("Client manager started (async)")

    def _client_manager(self) -> None:
        """Manage telemetry channels and writer for test conductor."""
        loop = sy.Loop(sy.Rate.HZ * 5)

        # Create telemetry channels
        time = self.client.channels.create(
            name=f"{self.name}_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        uptime = self.client.channels.create(
            name=f"{self.name}_uptime",
            data_type=sy.DataType.UINT32,
            index=time.key,
            retrieve_if_name_exists=True,
        )

        state = self.client.channels.create(
            name=f"{self.name}_state",
            data_type=sy.DataType.UINT8,
            index=time.key,
            retrieve_if_name_exists=True,
        )

        test_case_count = self.client.channels.create(
            name=f"{self.name}_test_case_count",
            data_type=sy.DataType.UINT32,
            index=time.key,
            retrieve_if_name_exists=True,
        )

        test_cases_ran = self.client.channels.create(
            name=f"{self.name}_test_cases_ran",
            data_type=sy.DataType.UINT32,
            index=time.key,
            retrieve_if_name_exists=True,
        )

        # Initialize telemetry
        start_time = sy.TimeStamp.now()
        self.tlm = {
            f"{self.name}_time": start_time,
            f"{self.name}_uptime": 0,
            f"{self.name}_state": STATE.INITIALIZING.value,
            f"{self.name}_test_case_count": 0,
            f"{self.name}_test_cases_ran": 0,
        }

        # Open telemetry writer
        with self.client.open_writer(
            start=start_time,
            channels=[time, uptime, state, test_case_count, test_cases_ran],
            name=self.name,
        ) as writer:
            writer.write(self.tlm)  # Write initial state

            while loop.wait() and not self.should_stop:
                now = sy.TimeStamp.now()
                uptime_value = (now - start_time) / 1e9

                # Update telemetry
                self.tlm[f"{self.name}_time"] = now
                self.tlm[f"{self.name}_uptime"] = uptime_value
                self.tlm[f"{self.name}_state"] = self.state.value
                writer.write(self.tlm)

                # Check for shutdown
                if self.state in [STATE.SHUTDOWN, STATE.COMPLETED]:
                    self.state = STATE.COMPLETED
                    self.tlm[f"{self.name}_state"] = self.state.value
                    writer.write(self.tlm)
                    break

    def _setup_logging(self) -> None:
        """Configure logging for real-time output in CI environments."""
        # Check if running in CI environment
        ci_environment = is_ci()

        # Force unbuffered output in CI environments
        if ci_environment:
            if hasattr(sys.stdout, "reconfigure"):
                sys.stdout.reconfigure(line_buffering=True)

        # Create logger for this test conductor (don't configure root logger)
        self.logger = logging.getLogger(self.name)
        self.logger.setLevel(logging.INFO)

        # Remove any existing handlers to avoid duplicates
        for handler in self.logger.handlers[:]:
            self.logger.removeHandler(handler)

        # Add single handler
        handler = logging.StreamHandler(sys.stdout)
        handler.setLevel(logging.INFO)
        formatter = logging.Formatter("%(message)s")
        handler.setFormatter(formatter)
        self.logger.addHandler(handler)

        # Prevent propagation to root logger to avoid duplicate output
        self.logger.propagate = False

        # Force immediate flush for real-time output in CI
        for handler in self.logger.handlers:
            if hasattr(handler, "stream") and hasattr(handler.stream, "flush"):

                def make_flush(h: Any) -> Callable[[], None]:
                    return lambda: h.stream.flush()

                setattr(handler, "flush", make_flush(handler))

        if ci_environment:
            self.logger.info("CI environment detected - enabling real-time logging")

    def log_message(self, message: str, use_name: bool = True) -> None:
        """Log message with real-time output using logging module."""
        now = sy.TimeStamp.now()
        timestamp = now.datetime().strftime("%H:%M:%S.%f")[:-4]
        if use_name:
            self.logger.info(f"{timestamp} | {self.name} > {message}")
        else:
            self.logger.info(message)

        # Force flush to ensure immediate output in CI
        for handler in self.logger.handlers:
            if hasattr(handler, "flush"):
                handler.flush()

    def load_test_sequence(self, sequence: str | list[str] | None = None) -> None:
        """Load test sequence from JSON configuration file(s) or auto-discover test.json files."""
        self.state = STATE.LOADING

        # Determine which files to load
        if sequence is None:
            test_files = glob.glob("tests/*_tests.json")
            if not test_files:
                raise FileNotFoundError(
                    "No *_tests.json files found for auto-discovery"
                )
        elif isinstance(sequence, list):
            test_files = sequence
        else:
            test_files = [sequence]

        # Load all sequences from all files
        all_sequences = []
        failed_files = []
        for test_file in test_files:
            self.log_message(f"Loading tests from: {test_file}")

            # Simple path resolution - try current dir first, then tests/ dir
            file_path = Path(test_file)
            if not file_path.exists():
                file_path = Path("tests") / test_file

            try:
                with open(file_path, "r") as f:
                    file_data = json.load(f)
                if "sequences" in file_data:
                    all_sequences.extend(file_data["sequences"])
            except Exception as e:
                if isinstance(sequence, list):
                    self.log_message(f"Warning: Failed to load {test_file}: {e}")
                    failed_files.append((test_file, str(e)))
                else:
                    raise FileNotFoundError(f"Test file not found: {test_file}")

        if failed_files:
            failed_list = "\n".join(
                [f"  - {file}: {error}" for file, error in failed_files]
            )
            raise FileNotFoundError(
                f"Failed to load {len(failed_files)} file(s):\n{failed_list}"
            )

        if not all_sequences:
            raise FileNotFoundError("No valid sequences found")

        self._process_sequences(all_sequences)

    def _expand_parameters(self, test_def: TestDefinition) -> list[TestDefinition]:
        """
        Expand a test definition with parameters into multiple test definitions.

        Parameters can be either single values or lists:
        - Single value: {"timeout": 2} → 1 test
        - List value: {"timeout": [2, 4]} → 2 tests
        - Mixed: {"mode": ["a", "b"], "rate": [100, 200]} → 4 tests

        Example:
            parameters = {"mode": ["a", "b"], "rate": [100, 200], "fixed": 5}
            Expands to 4 tests with params:
            - {"mode": "a", "rate": 100, "fixed": 5}
            - {"mode": "a", "rate": 200, "fixed": 5}
            - {"mode": "b", "rate": 100, "fixed": 5}
            - {"mode": "b", "rate": 200, "fixed": 5}
        """
        if not test_def.parameters:
            # No parameters, return single test
            return [test_def]

        # Separate parameters into single-value and multi-value
        single_params = {}
        multi_params = {}

        for key, value in test_def.parameters.items():
            if isinstance(value, list):
                multi_params[key] = value
            else:
                single_params[key] = value

        # If no multi-value parameters, return single test
        if not multi_params:
            return [test_def]

        # Generate cartesian product of multi-value parameters
        param_keys = list(multi_params.keys())
        param_values = [multi_params[key] for key in param_keys]
        combinations = list(itertools.product(*param_values))

        expanded_tests = []
        for combo in combinations:
            # Create parameter dict from combination
            combo_params = dict(zip(param_keys, combo))

            # Merge single-value params with this combination
            merged_params = {**single_params, **combo_params}

            # Generate name from multi-value parameters only
            matrix_suffix = "_".join(str(v) for v in combo)
            base_name = test_def.name or test_def.case.split("/")[-1]
            generated_name = f"{base_name}_{matrix_suffix}"

            # Create new test definition
            expanded_test = TestDefinition(
                case=test_def.case,
                name=generated_name,
                parameters=merged_params,  # Store as single values
                expect=test_def.expect,
            )
            expanded_tests.append(expanded_test)

        return expanded_tests

    def _process_sequences(self, sequences_array: list[Any]) -> None:
        """Process a list of sequences and populate test_definitions and sequences."""
        self.test_definitions = []
        self.sequences = []

        for seq_idx, sequence in enumerate(sequences_array):
            seq_dict = sequence if isinstance(sequence, dict) else {}
            seq_name = seq_dict.get("sequence_name", f"Sequence_{seq_idx + 1}")
            seq_order = seq_dict.get("sequence_order", "sequential").lower()
            seq_tests = seq_dict.get("tests", [])
            pool_size = seq_dict.get("pool_size", -1)

            # Create sequence object
            seq_obj = {
                "name": seq_name,
                "order": seq_order,
                "pool_size": pool_size,
                "tests": [],
                "start_idx": len(self.test_definitions),
            }

            # Load tests for this sequence
            for test in seq_tests:
                test_def = TestDefinition(
                    case=test["case"],
                    name=test.get("name", None),
                    parameters=test.get("parameters", {}),
                    expect=test.get("expect", "PASSED"),
                    matrix=test.get("matrix", None),
                )
                expanded_tests = self._expand_parameters(test_def)
                for expanded_test in expanded_tests:
                    self.test_definitions.append(expanded_test)
                    seq_obj["tests"].append(expanded_test)

            seq_obj["end_idx"] = len(self.test_definitions)
            self.sequences.append(seq_obj)

            # Improved logging to show expansion
            num_expanded = len(seq_obj["tests"])
            if num_expanded > len(seq_tests):
                self.log_message(
                    f"Loaded sequence '{seq_name}' with {len(seq_tests)} test definitions, "
                    f"expanded to {num_expanded} tests ({seq_order})"
                )
            else:
                self.log_message(
                    f"Loaded sequence '{seq_name}' with {len(seq_tests)} tests ({seq_order})"
                )

        self.log_message(
            f"Total: {len(self.test_definitions)} tests across {len(self.sequences)} sequences"
        )

        # Update telemetry
        self.tlm[f"{self.name}_test_case_count"] = len(self.test_definitions)

    def run_sequence(self) -> list[Test]:
        """Execute all tests in the loaded sequence."""
        if not self.test_definitions:
            raise ValueError(
                "No test sequence loaded. Call load_test_sequence() first."
            )

        self.state = STATE.RUNNING
        self.is_running = True
        self.should_stop = False
        self.tests: list[Test] = []

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
                f"==== SEQUENCE {seq_idx + 1}/{len(self.sequences)}: {sequence['name']} ===="
            )
            self.log_message(
                f"Executing {len(sequence['tests'])} tests with {sequence['order']} order...\n"
            )

            # Prepare tests for execution (randomize if needed)
            tests_to_execute = sequence["tests"].copy()

            # Execute tests within this sequence using the prepared test list
            # This consolidates sequential and random execution into a single path
            if sequence["order"] == "asynchronous":
                self._execute_sequence_asynchronously(
                    sequence["name"], tests_to_execute, sequence["pool_size"]
                )
            else:  # sequential or random (both use the same execution method)
                if sequence["order"] == "random":
                    random.shuffle(tests_to_execute)
                    self.log_message(f"Tests randomized for execution")
                self._execute_sequence(tests_to_execute)

            self.log_message(f"Completed sequence '{sequence['name']}'\n")

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
                f"[{global_test_idx}/{len(self.test_definitions)}] ==== {test_def} ===="
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
            self.tlm[f"{self.name}_test_cases_ran"] += 1

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
                f"[{global_test_idx}/{len(self.test_definitions)}] ==== {test_def} ===="
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
            self.tlm[f"{self.name}_test_cases_ran"] += 1

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
                    f"[{test_idx}/{len(self.test_definitions)}] ==== {test_def} ===="
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
            self.tlm[f"{self.name}_test_cases_ran"] += 1

    def wait_for_completion(self) -> None:
        """
        Wait for all async processes to complete before allowing main to exit.
        This ensures proper cleanup and prevents premature termination.
        """
        self.state = STATE.SHUTDOWN
        self.should_stop = True

        # Wait for client manager thread to finish
        if self.client_manager_thread and self.client_manager_thread.is_alive():
            self.client_manager_thread.join(timeout=5.0)
            if self.client_manager_thread.is_alive():
                self.log_message(
                    "Warning: client_manager_thread did not stop within timeout"
                )

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

    def _load_test_class(self, test_def: TestDefinition) -> type[TestCase]:
        """Dynamically load a test class from its case identifier."""
        try:
            # Parse the case string as a file path (e.g., "console/pages_open_close")
            case_path = f"tests/{test_def.case}"

            # Extract the module name from the path (last part before .py)
            module_name = case_path.split("/")[-1]

            # Convert module_name to PascalCase class name
            # "pages_open_close" -> "PagesOpenClose"
            class_name = "".join(word.capitalize() for word in module_name.split("_"))

            # Try different possible file paths
            current_dir = os.getcwd()
            script_dir = os.path.dirname(os.path.abspath(__file__))

            # Construct possible paths
            possible_paths = [
                os.path.join(script_dir, "..", f"{case_path}.py"),
                os.path.join(current_dir, f"{case_path}.py"),
            ]

            # Find the first path that exists
            file_path = None
            for path in possible_paths:
                if os.path.exists(path):
                    file_path = path
                    break

            if file_path is None:
                # Add debug information to help troubleshoot path issues
                debug_info = f"""
                Current working directory: {os.getcwd()}
                Script directory: {os.path.dirname(os.path.abspath(__file__))}
                Test case: {test_def.case}
                Module name: {module_name}
                Class name: {class_name}
                Tried paths: {possible_paths}
                """
                raise FileNotFoundError(
                    f"Could not find test module for {test_def.case}.\n{debug_info}"
                )

            integration_dir = os.path.dirname(
                os.path.dirname(os.path.dirname(file_path))
            )
            if integration_dir not in sys.path:
                sys.path.insert(0, integration_dir)

            # Prevent deadlock when multiple threads load modules that share dependencies
            with self.import_lock:
                spec = importlib.util.spec_from_file_location(module_name, file_path)
                if spec is None:
                    raise ImportError(
                        f"Cannot create spec for module: {module_name} at {file_path}"
                    )

                module = importlib.util.module_from_spec(spec)
                if spec.loader is not None:
                    spec.loader.exec_module(module)

            # Try to get the class by name
            try:
                test_class = getattr(module, class_name)
            except AttributeError:
                # If exact class name not found, try to find any TestCase subclass
                # that is defined in this module (not imported from elsewhere)
                test_classes = [
                    getattr(module, name)
                    for name in dir(module)
                    if (
                        not name.startswith("_")
                        and isinstance(getattr(module, name), type)
                        and issubclass(getattr(module, name), TestCase)
                        and getattr(module, name) is not TestCase
                        and getattr(module, name).__module__ == module.__name__
                    )
                ]
                if test_classes:
                    test_class = test_classes[
                        0
                    ]  # Use the first TestCase subclass found
                else:
                    raise AttributeError(f"No TestCase subclass found in {file_path}")

            if not issubclass(test_class, TestCase):
                raise TypeError(f"{class_name} is not a subclass of TestCase")
            return cast(type[TestCase], test_class)
        except Exception as e:
            raise ImportError(f"Failed to load test class from {test_def.case}: {e}\n")

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

        try:
            # Load and instantiate the test class
            test_class = self._load_test_class(test_def)
            test_instance = test_class(
                synnax_connection=self.synnax_connection,
                name=test_def.name or test_def.case.split("/")[-1],
                expect=test_def.expect,
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
            self.logger.error(f"Error: Failed to finalize range: {e}")
            raise RuntimeError(
                f"Failed to finalize range '{test_range.name}': {e}"
            ) from e

    def _test_runner_thread(
        self, test_def: TestDefinition, result_container: list[Test]
    ) -> None:
        """Thread function for running a single test."""
        result = self._execute_single_test(test_def)
        result_container.append(result)

    def _timeout_monitor_thread(self, monitor_interval: float = 0.5) -> None:
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
                expected_timeout = getattr(test_instance, "Expected_Timeout", -1)
                if expected_timeout <= 0:
                    continue

                elapsed_time = (
                    sy.TimeStamp.now() - test_range.time_range.start
                ) / sy.TimeSpan.SECOND
                if elapsed_time <= expected_timeout:
                    continue

                self._handle_test_timeout(test_instance, test_range, elapsed_time)
                tests_to_remove.append((test_instance, test_range, thread))

            for test_tuple in tests_to_remove:
                self.active_tests.remove(test_tuple)

    def _handle_test_timeout(
        self, test_instance: TestCase, test_range: sy.Range, elapsed_time: float
    ) -> None:
        """Handle a single test timeout."""
        expected_timeout = getattr(test_instance, "Expected_Timeout", -1)
        self.log_message(
            f"{test_instance.name} timeout detected ({elapsed_time:.1f}s > {expected_timeout}s)"
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

            self.active_tests = []

        with self.tests_lock:
            self.tests.extend(killed_test_results)

        # Terminate threads outside the lock
        for thread in threads_to_terminate:
            self._terminate_thread(thread)

        return len(killed_test_results)

    def stop_sequence(self) -> None:
        """Stop the entire test sequence execution."""
        self.log_message("Stopping test sequence...")
        self.should_stop = True
        killed = self.kill_active_tests()
        if killed > 0:
            self.log_message(f"Killed {killed} active test(s)")
        self._stop_client_manager()

    def _stop_client_manager(self) -> None:
        """Stop the client manager thread gracefully."""
        if self.client_manager_thread and self.client_manager_thread.is_alive():
            self.log_message("Stopping client manager...")
            # The thread will stop when self.should_stop becomes True
            # or when status reaches SHUTDOWN
            self.client_manager_thread.join(timeout=5.0)
            if self.client_manager_thread.is_alive():
                self.log_message(
                    "Warning: Client manager thread did not stop gracefully"
                )
            else:
                self.log_message("Client manager stopped successfully")

    def get_current_status(self) -> dict[str, Any]:
        """Get the current status of test execution."""
        with self.active_tests_lock:
            active_tests_snapshot = [
                {
                    "name": test_instance.__class__.__name__,
                    "elapsed_time": (
                        (sy.TimeStamp.now() - test_range.time_range.start)
                        / sy.TimeSpan.SECOND
                        if test_range is not None
                        else 0
                    ),
                }
                for test_instance, test_range, _ in self.active_tests
            ]

        with self.tests_lock:
            completed_tests = len(self.tests)
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
                for result in self.tests
            ]

        return {
            "is_running": self.is_running,
            "total_tests": len(self.test_definitions),
            "completed_tests": completed_tests,
            "active_tests": active_tests_snapshot,
            "results": results,
        }

    def _get_test_statistics(self) -> dict[str, int]:
        """Calculate and return test execution statistics."""
        with self.tests_lock:
            if not self.tests:
                return {
                    "total": 0,
                    "passed": 0,
                    "failed": 0,
                    "killed": 0,
                    "timeout": 0,
                    "total_failed": 0,
                }

            passed = sum(1 for r in self.tests if r.status == STATUS.PASSED)
            failed = sum(1 for r in self.tests if r.status == STATUS.FAILED)
            killed = sum(1 for r in self.tests if r.status == STATUS.KILLED)
            timeout = sum(1 for r in self.tests if r.status == STATUS.TIMEOUT)

            # KILLED and TIMEOUT tests are also considered failed
            total_failed = failed + killed + timeout

            return {
                "total": len(self.tests),
                "passed": passed,
                "failed": failed,
                "killed": killed,
                "timeout": timeout,
                "total_failed": total_failed,
            }

    def _print_summary(self) -> None:
        """Print a summary of test execution results."""
        with self.tests_lock:
            if not self.tests:
                return
            tests_snapshot = list(self.tests)

        stats = self._get_test_statistics()
        self._last_stats = stats

        # Individual Summary
        self.log_message("\n" + "=" * 60, False)
        for test in tests_snapshot:
            # Calculate duration if range is finalized
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
            self.log_message(f"{status_symbol} {display_name}{duration_str}", False)
            if test.error_message:
                self.log_message(f"ERROR: {test.error_message}")

        # Header
        self.log_message("=" * 60, False)
        self.log_message("TEST EXECUTION SUMMARY", False)

        # Summary Counts
        self.log_message("=" * 60, False)
        self.log_message(f"Total tests: {stats['total']}", False)
        if self.range is not None:
            test_time = (
                sy.TimeStamp.now() - self.range.time_range.start
            ) / sy.TimeSpan.SECOND
            self.log_message(f"Total time: {test_time:.1f} s", False)
        self.log_message(f"Passed: {stats['passed']}", False)
        self.log_message(
            f"Failed: {stats['total_failed']} (includes {stats['failed']} failed, {stats['killed']} killed, {stats['timeout']} timeout)",
            False,
        )
        self.log_message("=" * 60, False)
        self.log_message("\n", False)

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

    parser = argparse.ArgumentParser(description="Run test sequences")
    parser.add_argument("--name", default="tc", help="Test conductor name")
    parser.add_argument("--server", default="localhost", help="Synnax server address")
    parser.add_argument("--port", type=int, default=9090, help="Synnax server port")
    parser.add_argument("--username", default="synnax", help="Synnax username")
    parser.add_argument("--password", default="seldon", help="Synnax password")
    parser.add_argument("--secure", default=False, help="Use secure connection")
    parser.add_argument(
        "--sequence",
        "-s",
        help="Path to test sequence JSON file or comma-separated list of files (optional - will auto-discover *_tests.json if not provided)",
    )
    parser.add_argument(
        "--headed",
        type=bool,
        default=False,
        help="Run Playwright Console tests in headed mode (sets PLAYWRIGHT_CONSOLE_HEADED environment variable)",
    )

    args = parser.parse_args()

    os.environ["PLAYWRIGHT_CONSOLE_HEADED"] = "1" if args.headed else "0"

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
        # Handle sequence parameter - support both single file and list
        sequence_input: str | list[str] | None = None
        if args.sequence:
            # Check if it's a comma-separated list
            if "," in args.sequence:
                raw_list = [s.strip() for s in args.sequence.split(",")]
                sequence_input = [
                    (
                        s
                        if s.endswith(".json")
                        else (
                            f"{s}.json" if s.endswith("_tests") else f"{s}_tests.json"
                        )
                    )
                    for s in raw_list
                ]
            else:
                if args.sequence.endswith(".json"):
                    sequence_input = args.sequence
                elif args.sequence.endswith("_tests"):
                    sequence_input = f"{args.sequence}.json"
                else:
                    sequence_input = f"{args.sequence}_tests.json"

        conductor.load_test_sequence(sequence_input)
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
        if conductor.tests:
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
