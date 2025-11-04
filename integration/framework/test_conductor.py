#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
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
import time
from collections.abc import Callable
from dataclasses import dataclass, field
from datetime import datetime
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
class TestResult:
    """Data class to store test execution results."""

    test_name: str
    status: STATUS
    name: str | None = None  # Custom name from test definition
    start_time: datetime | None = None
    end_time: datetime | None = None
    error_message: str | None = None
    duration: float | None = None

    def __post_init__(self) -> None:
        if self.start_time and self.end_time:
            self.duration = (self.end_time - self.start_time).total_seconds()

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

        # Initialize Synnax client
        self.client = sy.Synnax(
            host=self.synnax_connection.server_address,
            port=self.synnax_connection.port,
            username=self.synnax_connection.username,
            password=self.synnax_connection.password,
            secure=self.synnax_connection.secure,
        )

        # Initialize state and collections
        self.start_time = datetime.now()
        self.state = STATE.INITIALIZING
        self.test_definitions: list[TestDefinition] = []
        self.test_results: list[TestResult] = []
        self.sequences: list[dict[str, Any]] = []
        self.current_test: TestCase | None = None
        self.current_test_start_time: datetime | None = None
        self.timeout_monitor_thread: threading.Thread | None = None
        self.client_manager_thread: threading.Thread | None = None
        self.current_test_thread: threading.Thread | None = None
        self._timeout_result: TestResult | None = None
        self.is_running = False
        self.should_stop = False
        self.status_callbacks: list[Callable[[TestResult], None]] = []
        self.sequence_ordering: str = "Sequential"
        # For asynchronous execution, track multiple tests
        self.active_tests: list[tuple[TestCase, datetime]] = []

        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        # Start client manager
        self._start_client_manager_async()
        time.sleep(1)  # Allow client manager to start

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
                else:
                    raise FileNotFoundError(f"Test file not found: {test_file}")

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

            # Create sequence object
            seq_obj = {
                "name": seq_name,
                "order": seq_order,
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

        # Store the ordering for use in run_sequence
        self.sequence_ordering = (
            self.sequences[0]["order"] if self.sequences else "Sequential"
        )

        # Update telemetry
        self.tlm[f"{self.name}_test_case_count"] = len(self.test_definitions)

    def run_sequence(self) -> list[TestResult]:
        """Execute all tests in the loaded sequence."""
        if not self.test_definitions:
            raise ValueError(
                "No test sequence loaded. Call load_test_sequence() first."
            )

        self.state = STATE.RUNNING
        self.is_running = True
        self.should_stop = False
        self.test_results = []

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
                    sequence["name"], tests_to_execute
                )
            else:  # sequential or random (both use the same execution method)
                if sequence["order"] == "random":
                    random.shuffle(tests_to_execute)
                    self.log_message(f"Tests randomized for execution")
                self._execute_sequence(tests_to_execute)

            self.log_message(f"Completed sequence '{sequence['name']}'\n")

        self.is_running = False
        self._print_summary()
        return self.test_results

    def _execute_sequence(self, tests_to_execute: list[TestDefinition]) -> None:
        """Execute tests in a sequence one after another."""
        for test_def in tests_to_execute:
            if self.should_stop:
                self.log_message("Test execution stopped by user request")
                break

            # Calculate global test index
            global_test_idx = len(self.test_results) + 1
            self.log_message(
                f"[{global_test_idx}/{len(self.test_definitions)}] ==== {test_def} ===="
            )

            # Run test in separate thread
            result_container: list[TestResult] = []
            test_thread = threading.Thread(
                target=self._test_runner_thread, args=(test_def, result_container)
            )

            self.current_test_thread = test_thread
            test_thread.start()
            test_thread.join()

            # Get test result
            if result_container:
                test_result = result_container[0]
            else:
                test_result = TestResult(
                    test_name=test_def.case,
                    name=test_def.name or test_def.case.split("/")[-1],
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned",
                )

            self.test_results.append(test_result)
            self.current_test_thread = None
            self.tlm[f"{self.name}_test_cases_ran"] += 1

    def _execute_sequence_asynchronously(
        self, sequence_name: str, tests_to_execute: list[TestDefinition]
    ) -> None:
        """Execute tests in a sequence simultaneously."""
        test_threads = []
        result_containers = []

        for i, test_def in enumerate(tests_to_execute):
            if self.should_stop:
                self.log_message("Test execution stopped by user request")
                break

            # Calculate global test index - each test gets a unique index
            global_test_idx = len(self.test_results) + i + 1
            self.log_message(
                f"[{global_test_idx}/{len(self.test_definitions)}] ==== {test_def} ===="
            )

            # Create result container and thread for each test
            result_container: list[TestResult] = []
            test_thread = threading.Thread(
                target=self._test_runner_thread, args=(test_def, result_container)
            )

            test_threads.append(test_thread)
            result_containers.append(result_container)

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
            if result_containers[i]:
                test_result = result_containers[i][0]
            else:
                test_result = TestResult(
                    test_name=tests_to_execute[i].case,
                    name=tests_to_execute[i].name,
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned",
                )

            self.test_results.append(test_result)
            self.tlm[f"{self.name}_test_cases_ran"] += 1

    def wait_for_completion(self) -> None:
        """
        Wait for all async processes to complete before allowing main to exit.
        This ensures proper cleanup and prevents premature termination.
        """
        self.state = STATE.SHUTDOWN

        # Wait for client manager thread to finish
        if self.client_manager_thread and self.client_manager_thread.is_alive():
            self.client_manager_thread.join()

        # Wait for timeout monitor to finish
        if self.timeout_monitor_thread and self.timeout_monitor_thread.is_alive():
            self.timeout_monitor_thread.join()

        # Wait for current test to complete
        if self.current_test_thread and self.current_test_thread.is_alive():
            self.current_test_thread.join()
            self.log_message("Test Thread has stopped")

        self.state = STATE.COMPLETED

    def shutdown(self) -> None:
        """
        Gracefully shutdown the test conductor and all its processes.
        """
        self.log_message("\nShut down initiated...")
        self.state = STATE.SHUTDOWN
        self.should_stop = True

        # Wait for all processes to complete
        self.wait_for_completion()

        self.log_message("Shutdown complete\n")

    def add_status_callback(self, callback: Callable[[TestResult], None]) -> None:
        """Add a callback function to be called when test status changes."""
        self.status_callbacks.append(callback)

    def _notify_status_change(self, result: TestResult) -> None:
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

    def _execute_single_test(self, test_def: TestDefinition) -> TestResult:
        """Execute a single test case."""
        result = TestResult(
            test_name=test_def.case,
            name=test_def.name or test_def.case.split("/")[-1],
            status=STATUS.PENDING,
            start_time=datetime.now(),
        )

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
            if self.sequence_ordering == "asynchronous":
                self.active_tests.append((test_instance, datetime.now()))
            else:
                self.current_test = test_instance
                self.current_test_start_time = datetime.now()

            result.status = STATUS.RUNNING
            result.start_time = datetime.now()
            self._notify_status_change(result)

            # Execute the test
            test_instance.execute()
            result.status = test_instance._status

        except Exception as e:
            # Check if test was killed/timed out during exception
            if self._timeout_result is not None:
                result = self._timeout_result
                self._timeout_result = None
            else:
                result.status = STATUS.FAILED
                result.error_message = str(e)
                self.log_message(f"{test_def.case} FAILED: {e}")
                # Log the full traceback for debugging
                import traceback

                self.log_message(f"Traceback: {traceback.format_exc()}")

        finally:
            result.end_time = datetime.now()

            # Clean up test tracking
            if self.sequence_ordering == "asynchronous":
                # Remove from active tests list
                self.active_tests = [
                    (test, start_time)
                    for test, start_time in self.active_tests
                    if test != test_instance
                ]
            else:
                self.current_test = None
                self.current_test_start_time = None

            self._notify_status_change(result)

        return result

    def create_ranges(self) -> None:
        """Create a range in Synnax with the given name and time span."""
        try:
            conductor_range = self.client.ranges.create(
                name=self.name,
                time_range=sy.TimeRange(
                    start=self.start_time,
                    end=datetime.now(),
                ),
            )
            for i, test in enumerate(self.test_results):
                color = COLORS[i % len(COLORS)]
                conductor_range.create_child_range(
                    name=test.name,
                    time_range=sy.TimeRange(
                        start=test.start_time,
                        end=test.end_time,
                    ),
                    color=color,
                )
        except Exception as e:
            raise RuntimeError(f"Failed to create range for {self.name}: {e}")

    def _test_runner_thread(
        self, test_def: TestDefinition, result_container: list[TestResult]
    ) -> None:
        """Thread function for running a single test."""
        result = self._execute_single_test(test_def)
        result_container.append(result)

    def _timeout_monitor_thread(self, monitor_interval: float = 0.5) -> None:
        """Monitor test execution for timeout violations."""
        while self.is_running and not self.should_stop:
            # Check current test (for sequential execution)
            if (
                self.current_test is not None
                and self.current_test_start_time is not None
                and hasattr(self.current_test, "Expected_Timeout")
                and self.current_test.Expected_Timeout > 0
            ):
                elapsed_time = (
                    datetime.now() - self.current_test_start_time
                ).total_seconds()
                if elapsed_time > self.current_test.Expected_Timeout:
                    self.kill_current_test()
                    break

            # Check active tests (for asynchronous execution)
            if self.sequence_ordering == "asynchronous" and self.active_tests:
                tests_to_remove = []
                for test_instance, start_time in self.active_tests:
                    if (
                        hasattr(test_instance, "Expected_Timeout")
                        and test_instance.Expected_Timeout > 0
                    ):
                        elapsed_time = (datetime.now() - start_time).total_seconds()
                        if elapsed_time > test_instance.Expected_Timeout:
                            self.log_message(
                                f"{test_instance.name} timeout detected ({elapsed_time:.1f}s > {test_instance.Expected_Timeout}s)"
                            )
                            # Mark test as timed out - it will handle itself
                            test_instance._status = STATUS.TIMEOUT
                            tests_to_remove.append((test_instance, start_time))

                # Remove completed tests from active list
                for test_tuple in tests_to_remove:
                    self.active_tests.remove(test_tuple)

            time.sleep(monitor_interval)

    def kill_current_test(self) -> bool:
        """Kill currently running test and create timeout result."""
        if self.current_test is None:
            return False

        # Create timeout result if this is a timeout kill
        if self.current_test_start_time:
            elapsed_time = (
                datetime.now() - self.current_test_start_time
            ).total_seconds()
            expected_timeout = getattr(self.current_test, "Expected_Timeout", -1)

            # Determine if timeout or manual kill
            if expected_timeout > 0 and elapsed_time > expected_timeout:
                status = STATUS.TIMEOUT
                error_msg = f"Test exceeded Expected_Timeout ({expected_timeout}s). Elapsed: {elapsed_time:.1f}s"
            else:
                status = STATUS.KILLED
                error_msg = "Test was manually killed"

            # Get test name
            current_test_name = (
                self.current_test.name if self.current_test else "unknown_test"
            )

            # Create timeout result
            self._timeout_result = TestResult(
                test_name=current_test_name,
                name=(
                    getattr(self.current_test, "custom_name", None)
                    if self.current_test
                    else None
                ),
                status=status,
                start_time=self.current_test_start_time,
                end_time=datetime.now(),
                error_message=error_msg,
            )

        # Clear current test info
        self.current_test = None
        self.current_test_start_time = None
        return True

    def stop_sequence(self) -> None:
        """Stop the entire test sequence execution."""
        self.log_message("Stopping test sequence...")
        self.should_stop = True
        self.kill_current_test()
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
        return {
            "is_running": self.is_running,
            "total_tests": len(self.test_definitions),
            "completed_tests": len(self.test_results),
            "current_test": (
                self.current_test.__class__.__name__ if self.current_test else None
            ),
            "results": [
                {
                    "name": result.test_name,
                    "status": result.status.value,
                    "duration": result.duration,
                    "error": result.error_message,
                }
                for result in self.test_results
            ],
        }

    def _get_test_statistics(self) -> dict[str, int]:
        """Calculate and return test execution statistics."""
        if not self.test_results:
            return {
                "total": 0,
                "passed": 0,
                "failed": 0,
                "killed": 0,
                "timeout": 0,
                "total_failed": 0,
            }

        passed = sum(1 for r in self.test_results if r.status == STATUS.PASSED)
        failed = sum(1 for r in self.test_results if r.status == STATUS.FAILED)
        killed = sum(1 for r in self.test_results if r.status == STATUS.KILLED)
        timeout = sum(1 for r in self.test_results if r.status == STATUS.TIMEOUT)

        # KILLED and TIMEOUT tests are also considered failed
        total_failed = failed + killed + timeout

        return {
            "total": len(self.test_results),
            "passed": passed,
            "failed": failed,
            "killed": killed,
            "timeout": timeout,
            "total_failed": total_failed,
        }

    def _print_summary(self) -> None:
        """Print a summary of test execution results."""
        if not self.test_results:
            return

        stats = self._get_test_statistics()
        self._last_stats = stats

        # Individual Summary
        self.log_message("\n" + "=" * 60, False)
        for result in self.test_results:
            if result.start_time and result.end_time:
                duration = (result.end_time - result.start_time).total_seconds()
                duration_str = f" ({duration:.1f}s)"
            else:
                duration_str = ""

            status_symbol = SYMBOLS.get_symbol(result.status)
            case_parts = str(result).split("/")
            display_name = (
                "/".join(case_parts[1:]) if len(case_parts) > 1 else str(result)
            )
            self.log_message(f"{status_symbol} {display_name}{duration_str}", False)
            if result.error_message:
                self.log_message(f"ERROR: {result.error_message}")

        # Header
        self.log_message("=" * 60, False)
        self.log_message("TEST EXECUTION SUMMARY", False)

        # Summary Counts
        self.log_message("=" * 60, False)
        self.log_message(f"Total tests: {stats['total']}", False)
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
        sys.exit(0)


def monitor_test_execution(conductor: TestConductor) -> None:
    """Monitor test execution and provide status updates."""
    while conductor.is_running:
        status = conductor.get_current_status()
        conductor.log_message(
            f"Status: {status['completed_tests']}/{status['total_tests']} tests completed"
        )
        if status["current_test"]:
            conductor.log_message(f"Currently running: {status['current_test']}")
        time.sleep(1)


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
        help="Path to test sequence JSON file or comma-separated list of files (optional - will auto-discover *_tests.json if not provided)",
    )

    args = parser.parse_args()

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
        # Ranges must be created last for the test_conductor to be available as a parent.
        conductor.create_ranges()
        conductor.log_message(f"Fin.")
        if conductor.test_results:
            stats = conductor._get_test_statistics()

            if stats["total_failed"] > 0:
                conductor.log_message(
                    f"\nExiting with failure code due to {stats['total_failed']}/{stats['total']} failed tests"
                )
                sys.exit(1)
            else:
                conductor.log_message(
                    f"\nAll {stats['total']} tests passed successfully", False
                )
                sys.exit(0)
        else:
            conductor.log_message("\nNo test results available")
            sys.exit(1)


if __name__ == "__main__":
    main()
