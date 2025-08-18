#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# TODO:
# - Use system logging
# - Return test_conductor results to workflow env
# - Enable async test_case execution if configured
# - Build test case infrastructure
# - Integrate with github actions


import json
import threading
import time
import signal
import sys
import random
import string
import re
from typing import List, Dict, Any, Optional, Callable, Tuple
from pathlib import Path
from enum import Enum, auto
from dataclasses import dataclass, field
from datetime import datetime
import importlib.util
import traceback
import synnax as sy

try:
    # Import from the framework module to ensure we get the same class objects
    import os
    sys.path.append(os.path.join(os.path.dirname(__file__), '..'))
    from framework.TestCase import TestCase, SynnaxConnection, STATUS, SYMBOLS
except ImportError:
    # Handle case when running script directly
    from TestCase import TestCase, SynnaxConnection, STATUS, SYMBOLS

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
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    error_message: Optional[str] = None
    duration: Optional[float] = None
    
    def __post_init__(self):
        if self.start_time and self.end_time:
            self.duration = (self.end_time - self.start_time).total_seconds()


@dataclass
class TestDefinition:
    """Data class representing a test case definition from the sequence file."""
    case: str
    params: Dict[str, Any] = field(default_factory=dict)
    expect: str = "PASSED"  # Expected test outcome, defaults to "PASSED"

class Test_Conductor:
    """Manages execution of test sequences with timeout monitoring and result collection."""
    
    def __init__(self, 
                 name: str, 
                 server_address: str = "localhost", 
                 port: int = 9090,
                 username: str = "synnax", 
                 password: str = "seldon", 
                 secure: bool = False):
        """Initialize test conductor with connection parameters."""
        
        # Generate or validate name
        if name is None:
            random_id = ''.join(random.choices(string.ascii_lowercase + string.digits, k=6))
            self.name = self.__class__.__name__.lower() + "_" + random_id
        else:
            self.name = self._validate_and_sanitize_name(str(name).lower())
        
        # Create connection parameters
        self.SynnaxConnection = SynnaxConnection(
            server_address=server_address,
            port=port,
            username=username,
            password=password,
            secure=secure,
        )
        
        # Initialize Synnax client
        self.client = sy.Synnax(
            host=server_address,
            port=port,
            username=username,
            password=password,
            secure=secure,
        )
        
        # Initialize state and collections
        self.state = STATE.INITIALIZING
        self.test_definitions: List[TestDefinition] = []
        self.test_results: List[TestResult] = []
        self.sequences: List[dict] = []
        self.current_test: Optional[TestCase] = None
        self.current_test_thread: Optional[threading.Thread] = None
        self.current_test_start_time: Optional[datetime] = None
        self.timeout_monitor_thread: Optional[threading.Thread] = None
        self._timeout_result: Optional[TestResult] = None
        self.client_manager_thread: Optional[threading.Thread] = None
        self.is_running = False
        self.should_stop = False
        self.status_callbacks: List[Callable[[TestResult], None]] = []
        self.sequence_ordering: str = "Sequential"
        # For asynchronous execution, track multiple tests
        self.active_tests: List[Tuple[TestCase, datetime]] = []
        
        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        # Start client manager
        self._start_client_manager_async()
        time.sleep(1)  # Allow client manager to start
    
    def _validate_and_sanitize_name(self, name: str) -> str:
        """Sanitize name to contain only alphanumeric characters, hyphens, and underscores."""
        sanitized = re.sub(r'[^a-zA-Z0-9_-]', '', name)
        
        if not sanitized:
            raise ValueError("Name must contain at least one alphanumeric character")
        
        sanitized = sanitized.strip('_-')
        if not sanitized:
            raise ValueError("Name cannot consist only of hyphens and underscores")
            
        return sanitized

    def _start_client_manager_async(self) -> None:
        """Start client manager in separate daemon thread."""
        self.client_manager_thread = threading.Thread(
            target=self._client_manager,
            daemon=True,
            name=f"{self.name}_client_manager"
        )
        self.client_manager_thread.start()
        print(f"{self.name} > Client manager started (async)")
    
    def _client_manager(self) -> None:
        
        """Manage telemetry channels and writer for test conductor."""
        loop = sy.Loop(1)  # 1Hz

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
            enable_auto_commit=True,
        ) as writer:
            writer.write(self.tlm)  # Write initial state

            while loop.wait() and not self.should_stop:
                now = sy.TimeStamp.now()
                uptime_value = (now - start_time)/1E9
                
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
    
    def load_test_sequence(self, sequence: str = None) -> None:
        """Load test sequence from JSON configuration file."""
        self.state = STATE.LOADING
        
        if sequence is None:
            raise ValueError("Path to JSON Sequence file is required (--sequence)")

        # Resolve sequence file path
        sequence_path = Path(sequence)
        if not sequence_path.is_absolute():
            sequence_path = sequence_path.resolve()
        
        # Try to find the file in common locations
        if not sequence_path.exists():
            possible_paths = [
                Path.cwd() / sequence,
                Path(__file__).parent / sequence,
                Path(__file__).parent.parent / sequence,
            ]
            
            for path in possible_paths:
                if path.exists():
                    sequence_path = path.resolve()
                    break
            else:
                raise FileNotFoundError(
                    f"{self.name} > Test sequence file not found: {sequence}\n"
                    f"Tried paths:\n" + 
                    "\n".join(f"  - {p}" for p in [sequence_path] + possible_paths)
                )
        
        time.sleep(1)
        with open(sequence_path, 'r') as f:
            sequence_data = json.load(f)
        
        # Load test definitions - support both single sequence and multi-sequence format
        self.test_definitions = []
        self.sequences = []
        
        if isinstance(sequence_data, list):
            # New format: array of sequences
            for seq_idx, sequence in enumerate(sequence_data):
                seq_name = sequence.get("sequence_name", f"Sequence_{seq_idx + 1}")
                seq_order = sequence.get("sequence_order", "Sequential").lower()
                seq_tests = sequence.get("tests", [])
                
                # Create sequence object
                seq_obj = {
                    "name": seq_name,
                    "order": seq_order,
                    "tests": [],
                    "start_idx": len(self.test_definitions)
                }
                
                # Load tests for this sequence
                for test in seq_tests:
                    test_def = TestDefinition(
                        case=test["case"],
                        params=test.get("parameters", {}),
                        expect=test.get("expect", "PASSED"),
                    )
                    self.test_definitions.append(test_def)
                    seq_obj["tests"].append(test_def)
                
                seq_obj["end_idx"] = len(self.test_definitions)
                self.sequences.append(seq_obj)
                
                print(f"{self.name} > Loaded sequence '{seq_name}' with {len(seq_tests)} tests ({seq_order})")
            
            print(f"{self.name} > Total: {len(self.test_definitions)} tests across {len(self.sequences)} sequences from: \n{sequence}")
        else:
            # Old format: single sequence
            for test in sequence_data.get("tests", []):
                test_def = TestDefinition(
                    case=test["case"],
                    params=test.get("parameters", {}),
                    expect=test.get("expect", "PASSED"),
                )
                self.test_definitions.append(test_def)
            
            # Handle test ordering for single sequence
            ordering = "Sequential"
            sequence_order = sequence_data.get("sequence_order", "Sequential").lower()
            if sequence_order == "random":
                random.shuffle(self.test_definitions)
            elif sequence_order == "asynchronous":
                ordering = "Asynchronous"
            
            # Create single sequence object for compatibility
            self.sequences = [{
                "name": "Main Sequence",
                "order": sequence_order,
                "tests": self.test_definitions,
                "start_idx": 0,
                "end_idx": len(self.test_definitions)
            }]
            
            print(f"{self.name} > Sequence loaded with {len(self.test_definitions)} tests ({ordering}) from {sequence}")
        
        # Store the ordering for use in run_sequence (for backward compatibility)
        self.sequence_ordering = self.sequences[0]["order"] if self.sequences else "Sequential"
        
        # Update telemetry
        self.tlm[f"{self.name}_test_case_count"] = len(self.test_definitions)
    
    def run_sequence(self) -> List[TestResult]:
        """Execute all tests in the loaded sequence."""
        if not self.test_definitions:
            raise ValueError("No test sequence loaded. Call load_test_sequence() first.")
        
        self.state = STATE.RUNNING
        self.is_running = True
        self.should_stop = False
        self.test_results = []
        
        # Start timeout monitoring
        self.timeout_monitor_thread = threading.Thread(
            target=self._timeout_monitor_thread,
            args=(1.0,),  # Check every 1 second
            daemon=True
        )
        self.timeout_monitor_thread.start()
        
        print(f"\n{self.name} > Starting execution of {len(self.sequences)} sequences with {len(self.test_definitions)} total tests...\n")

        # Execute sequences linearly (one after another)
        for seq_idx, sequence in enumerate(self.sequences):
            if self.should_stop:
                print(f"{self.name} > Test execution stopped by user request")
                break
            
            print(f"\n{self.name} > ==== SEQUENCE {seq_idx + 1}/{len(self.sequences)}: {sequence['name']} ====")
            print(f"{self.name} > Executing {len(sequence['tests'])} tests with {sequence['order']} order...")
            
            # Execute tests within this sequence according to its order
            if sequence['order'] == "asynchronous":
                self._execute_sequence_asynchronously(sequence)
            elif sequence['order'] == "random":
                self._execute_sequence_randomly(sequence)
            else:  # sequential
                self._execute_sequence_sequentially(sequence)
            
            print(f"{self.name} > Completed sequence '{sequence['name']}'")
        
        self.is_running = False
        self._print_summary()
        return self.test_results
    
    def _execute_tests_sequentially(self) -> None:
        """Execute tests one after another."""
        for i, test_def in enumerate(self.test_definitions):
            if self.should_stop:
                print(f"{self.name} > Test execution stopped by user request")
                break
            
            print(f"\n{self.name} [{i+1}/{len(self.test_definitions)}] > ==== {test_def.case} ====")
            
            # Run test in separate thread
            result_container = []
            test_thread = threading.Thread(
                target=self._test_runner_thread,
                args=(test_def, result_container)
            )
            
            self.current_test_thread = test_thread
            test_thread.start()
            test_thread.join()
            
            # Get test result
            if result_container:
                result = result_container[0]
            else:
                result = TestResult(
                    test_name=test_def.case,
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned"
                )
            
            self.test_results.append(result)
            self.current_test_thread = None
            self.tlm[f"{self.name}_test_cases_ran"] += 1
    
    def _execute_sequence_sequentially(self, sequence: dict) -> None:
        """Execute tests in a sequence one after another."""
        for i, test_def in enumerate(sequence['tests']):
            if self.should_stop:
                print(f"{self.name} > Test execution stopped by user request")
                break
            
            # Calculate global test index
            global_test_idx = len(self.test_results) + 1
            print(f"\n{self.name} [{global_test_idx}/{len(self.test_definitions)}] > ==== {test_def.case} ====")
            
            # Run test in separate thread
            result_container = []
            test_thread = threading.Thread(
                target=self._test_runner_thread,
                args=(test_def, result_container)
            )
            
            self.current_test_thread = test_thread
            test_thread.start()
            test_thread.join()
            
            # Get test result
            if result_container:
                result = result_container[0]
            else:
                result = TestResult(
                    test_name=test_def.case,
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned"
                )
            
            self.test_results.append(result)
            self.current_test_thread = None
            self.tlm[f"{self.name}_test_cases_ran"] += 1
    
    def _execute_sequence_randomly(self, sequence: dict) -> None:
        """Execute tests in a sequence in random order."""
        # Create a copy of tests and shuffle them
        shuffled_tests = sequence['tests'].copy()
        random.shuffle(shuffled_tests)
        
        for i, test_def in enumerate(shuffled_tests):
            if self.should_stop:
                print(f"{self.name} > Test execution stopped by user request")
                break
            
            # Calculate global test index
            global_test_idx = len(self.test_results) + 1
            print(f"\n{self.name} [{global_test_idx}/{len(self.test_definitions)}] > ==== {test_def.case} ====")
            
            # Run test in separate thread
            result_container = []
            test_thread = threading.Thread(
                target=self._test_runner_thread,
                args=(test_def, result_container)
            )
            
            self.current_test_thread = test_thread
            test_thread.start()
            test_thread.join()
            
            # Get test result
            if result_container:
                result = result_container[0]
            else:
                result = TestResult(
                    test_name=test_def.case,
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned"
                )
            
            self.test_results.append(result)
            self.current_test_thread = None
            self.tlm[f"{self.name}_test_cases_ran"] += 1
    
    def _execute_sequence_asynchronously(self, sequence: dict) -> None:
        """Execute tests in a sequence simultaneously."""
        # Launch all tests in this sequence at once
        test_threads = []
        result_containers = []
        
        for i, test_def in enumerate(sequence['tests']):
            if self.should_stop:
                print(f"{self.name} > Test execution stopped by user request")
                break
            
            # Calculate global test index
            global_test_idx = len(self.test_results) + 1
            print(f"\n{self.name} [{global_test_idx}/{len(self.test_definitions)}] > ==== {test_def.case} ====")
            
            # Create result container and thread for each test
            result_container = []
            test_thread = threading.Thread(
                target=self._test_runner_thread,
                args=(test_def, result_container)
            )
            
            test_threads.append(test_thread)
            result_containers.append(result_container)
            
            # Start the test thread
            test_thread.start()
        
        # Wait for all tests in this sequence to complete
        print(f"{self.name} > Waiting for {len(test_threads)} tests in sequence '{sequence['name']}' to complete...")
        for i, test_thread in enumerate(test_threads):
            if test_thread.is_alive():
                test_thread.join()
            
            # Get test result
            if result_containers[i]:
                result = result_containers[i][0]
            else:
                result = TestResult(
                    test_name=sequence['tests'][i].case,
                    status=STATUS.FAILED,
                    error_message="Unknown error - no result returned"
                )
            
            self.test_results.append(result)
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
            print(f"{self.name} > Test Thread has stopped")
        
        self.state = STATE.COMPLETED
    
    def shutdown(self) -> None:
        """
        Gracefully shutdown the test conductor and all its processes.
        """
        print("\n")
        print(f"{self.name} > Shut down initiated...")
        self.state = STATE.SHUTDOWN
        self.should_stop = True
        
        # Wait for all processes to complete
        self.wait_for_completion()
        
        print(f"{self.name} > Shutdown complete\n")
    
    def add_status_callback(self, callback: Callable[[TestResult], None]) -> None:
        """Add a callback function to be called when test status changes."""
        self.status_callbacks.append(callback)
    
    def _notify_status_change(self, result: TestResult) -> None:
        """Notify all registered callbacks about status changes."""
        for callback in self.status_callbacks:
            try:
                callback(result)
            except Exception as e:
                print(f"Error in status callback: {e}")
    
    def _load_test_class(self, test_def: TestDefinition) -> type:
        """Dynamically load a test class from its case identifier."""
        try:
            # Parse the case string to extract directory, module, and class name
            # Flexible format handling:
            # - "testcases.module_name.ClassName" (3 parts)
            # - "testcases.module_name" (2 parts - infer class name)
            case_parts = test_def.case.split('.')
            if len(case_parts) < 2:
                raise ValueError(f"{self.name} > Invalid test case format: {test_def.case}. Expected 'testcases.module_name[.ClassName]'")
            
            directory = case_parts[0]  # "testcases"
            module_name = case_parts[1]  # "check_connection_basic1" or "check_connection_basic"
            
            # If 3 parts, use the third as class name, otherwise infer from module name
            if len(case_parts) >= 3:
                class_name = case_parts[2]
            else:
                # Convert module_name to PascalCase class name
                # "check_connection_basic" -> "CheckConnectionBasic"
                class_name = ''.join(word.capitalize() for word in module_name.split('_'))
            
            # Try different possible file paths (exact match only)
            import os
            possible_paths = [
                f"../{directory}/{module_name}.py",           # ../testcases/check_connection_basic1.py
                f"{directory}/{module_name}.py",              # testcases/check_connection_basic1.py
            ]
            
            # Find the first path that exists
            file_path = None
            for path in possible_paths:
                if os.path.exists(path):
                    file_path = path
                    break
            
            if file_path is None:
                raise FileNotFoundError(f"Could not find test module for {test_def.case}. Tried: {possible_paths}")
            
            spec = importlib.util.spec_from_file_location(module_name, file_path)
            if spec is None:
                raise ImportError(f"Cannot create spec for module: {module_name} at {file_path}")
            
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            
            # Try to get the class by name
            try:
                test_class = getattr(module, class_name)
            except AttributeError:
                # If exact class name not found, try to find any TestCase subclass
                test_classes = [getattr(module, name) for name in dir(module) 
                              if (not name.startswith('_') and 
                                  hasattr(getattr(module, name), '__bases__') and
                                  TestCase in getattr(module, name).__bases__)]
                if test_classes:
                    test_class = test_classes[0]  # Use the first TestCase subclass found
                else:
                    raise AttributeError(f"No TestCase subclass found in {file_path}")
            
            if not issubclass(test_class, TestCase):
                raise TypeError(f"{class_name} is not a subclass of TestCase")
            return test_class
        except Exception as e:
            raise ImportError(f"Failed to load test class from {test_def.case}: {e}\n")
    
    def _execute_single_test(self, test_def: TestDefinition) -> TestResult:
        """Execute a single test case."""
        result = TestResult(
            test_name=test_def.case,
            status=STATUS.PENDING,
            start_time=datetime.now()
        )
        
        try:
            # Load and instantiate the test class
            test_class = self._load_test_class(test_def)
            test_instance = test_class(
                SynnaxConnection=self.SynnaxConnection,
                expect=test_def.expect,
                **test_def.params
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
                print(f"{self.name} > {test_def.case} FAILED: {e}")
                traceback.print_exc()
        
        finally:
            result.end_time = datetime.now()
            
            # Clean up test tracking
            if self.sequence_ordering == "asynchronous":
                # Remove from active tests list
                self.active_tests = [(test, start_time) for test, start_time in self.active_tests if test != test_instance]
            else:
                self.current_test = None
                self.current_test_start_time = None
            
            self._notify_status_change(result)
        
        return result
    
    def _test_runner_thread(self, test_def: TestDefinition, result_container: List[TestResult]) -> None:
        """Thread function for running a single test."""
        result = self._execute_single_test(test_def)
        result_container.append(result)
    
    def _timeout_monitor_thread(self, monitor_interval: float = 1.0) -> None:
        """Monitor test execution for timeout violations."""
        while self.is_running and not self.should_stop:
            # Check current test (for sequential execution)
            if (self.current_test is not None and 
                self.current_test_start_time is not None and 
                hasattr(self.current_test, 'Expected_Timeout') and
                self.current_test.Expected_Timeout > 0):
                
                elapsed_time = (datetime.now() - self.current_test_start_time).total_seconds()
                if elapsed_time > self.current_test.Expected_Timeout:
                    self.kill_current_test()
                    break
            
            # Check active tests (for asynchronous execution)
            if self.sequence_ordering == "asynchronous" and self.active_tests:
                tests_to_remove = []
                for test_instance, start_time in self.active_tests:
                    if (hasattr(test_instance, 'Expected_Timeout') and 
                        test_instance.Expected_Timeout > 0):
                        
                        elapsed_time = (datetime.now() - start_time).total_seconds()
                        if elapsed_time > test_instance.Expected_Timeout:
                            print(f"{self.name} > {test_instance.name} timeout detected ({elapsed_time:.1f}s > {test_instance.Expected_Timeout}s)")
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
            elapsed_time = (datetime.now() - self.current_test_start_time).total_seconds()
            expected_timeout = getattr(self.current_test, 'Expected_Timeout', -1)
            
            # Determine if timeout or manual kill
            if expected_timeout > 0 and elapsed_time > expected_timeout:
                status = STATUS.TIMEOUT
                error_msg = f"Test exceeded Expected_Timeout ({expected_timeout}s). Elapsed: {elapsed_time:.1f}s"
            else:
                status = STATUS.KILLED
                error_msg = "Test was manually killed"
            
            # Get test name
            current_test_name = self.current_test.name if self.current_test else "unknown_test"
            
            # Create timeout result
            self._timeout_result = TestResult(
                test_name=current_test_name,
                status=status,
                start_time=self.current_test_start_time,
                end_time=datetime.now(),
                error_message=error_msg
            )
        
        # Clear current test info
        self.current_test = None
        self.current_test_start_time = None
        return True
    
    def stop_sequence(self) -> None:
        """Stop the entire test sequence execution."""
        print("Stopping test sequence...")
        self.should_stop = True
        self.kill_current_test()
        self._stop_client_manager()
    
    def _stop_client_manager(self) -> None:
        """Stop the client manager thread gracefully."""
        if self.client_manager_thread and self.client_manager_thread.is_alive():
            print("Stopping client manager...")
            # The thread will stop when self.should_stop becomes True
            # or when status reaches SHUTDOWN
            self.client_manager_thread.join(timeout=5.0)
            if self.client_manager_thread.is_alive():
                print("Warning: Client manager thread did not stop gracefully")
            else:
                print("Client manager stopped successfully")
    
    def get_current_status(self) -> Dict[str, Any]:
        """Get the current status of test execution."""
        return {
            "is_running": self.is_running,
            "total_tests": len(self.test_definitions),
            "completed_tests": len(self.test_results),
            "current_test": self.current_test.__class__.__name__ if self.current_test else None,
            "results": [
                {
                    "name": result.test_name,
                    "status": result.status.value,
                    "duration": result.duration,
                    "error": result.error_message
                }
                for result in self.test_results
            ]
        }
    
    def _get_test_statistics(self) -> dict:
        """Calculate and return test execution statistics."""
        if not self.test_results:
            return {
                'total': 0,
                'passed': 0,
                'failed': 0,
                'killed': 0,
                'timeout': 0,
                'total_failed': 0
            }
        
        passed = sum(1 for r in self.test_results if r.status == STATUS.PASSED)
        failed = sum(1 for r in self.test_results if r.status == STATUS.FAILED)
        killed = sum(1 for r in self.test_results if r.status == STATUS.KILLED)
        timeout = sum(1 for r in self.test_results if r.status == STATUS.TIMEOUT)
        
        # KILLED and TIMEOUT tests are also considered failed
        total_failed = failed + killed + timeout
        
        return {
            'total': len(self.test_results),
            'passed': passed,
            'failed': failed,
            'killed': killed,
            'timeout': timeout,
            'total_failed': total_failed
        }

    def _print_summary(self) -> None:
        """Print a summary of test execution results."""
        if not self.test_results:
            return
        
        # Store stats for reuse in the finally block]
        stats = self._get_test_statistics()
        self._last_stats = stats

        print("\n" + "="*50)
        print("TEST EXECUTION SUMMARY")
        print("="*50)
        print(f"Total tests: {stats['total']}")
        print(f"Passed: {stats['passed']}")
        print(f"Failed: {stats['total_failed']} (includes {stats['failed']} failed, {stats['killed']} killed, {stats['timeout']} timeout)")
        print("="*50)
        
        for result in self.test_results:
            
            status_symbol = SYMBOLS.get_symbol(result.status)
            
            duration_str = f"({result.duration:.2f}s)" if result.duration else ""
            print(f"{status_symbol} {result.test_name} {duration_str}")
            if result.error_message:
                print(f"    Error: {result.error_message}")
           
    def _signal_handler(self, signum, frame):
        """Handle system signals for graceful shutdown."""
        print(f"\nReceived signal {signum}. Stopping test execution...")
        self.shutdown()
        sys.exit(0)


def monitor_test_execution(conductor: Test_Conductor) -> None:
    """Monitor test execution and provide status updates."""
    while conductor.is_running:
        status = conductor.get_current_status()
        print(f"Status: {status['completed_tests']}/{status['total_tests']} tests completed")
        if status['current_test']:
            print(f"Currently running: {status['current_test']}")
        time.sleep(1)

if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="Run test sequences")
    parser.add_argument("--name", default=None, help="Test conductor name")
    parser.add_argument("--server", default="localhost", help="Synnax server address")
    parser.add_argument("--port", type=int, default=9090, help="Synnax server port")
    parser.add_argument("--username", default="synnax", help="Synnax username")
    parser.add_argument("--password", default="seldon", help="Synnax password")
    parser.add_argument("--secure", default=False, help="Use secure connection")
    parser.add_argument("--sequence", help="Path to test sequence JSON file (required)")
    
    args = parser.parse_args()
    
    # Create and run test conductor
    conductor = Test_Conductor(
        name=args.name, 
        server_address=args.server, 
        port=args.port,
        username=args.username,
        password=args.password,
        secure=args.secure,
    )

    try:
        conductor.load_test_sequence(args.sequence)
        results = conductor.run_sequence()
        conductor.wait_for_completion()
        
    except KeyboardInterrupt:
        print("\nKeyboard interrupt received. Shutting down gracefully...")
        conductor.shutdown()
    except Exception as e:
        print(f"Error occurred: {e}")
        conductor.shutdown()
        raise
    finally:
        print(f"\n{conductor.name} > Fin.")
        
        if conductor.test_results:
            stats = conductor._get_test_statistics()

            if stats['total_failed'] > 0:
                print(f"\nExiting with failure code due to {stats['total_failed']}/{stats['total']} failed tests")
                sys.exit(1)
            else:
                print(f"\nAll {stats['total']} tests passed successfully")
                sys.exit(0)
        else:
            print("\nNo test results available")
            sys.exit(1)

