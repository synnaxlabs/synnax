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
# - Return test results to test_conductore
# - Return test_conductor results to workflow env
# - Enable async test_case execution if configured
# - Build test case infrastructure
# - Run loaded test cases
# - Integrate with github actions


import json
import uuid
import asyncio
import threading
import time
import signal
import sys
import random
import string
import re
from typing import List, Dict, Any, Optional, Callable
from pathlib import Path
from enum import Enum, auto
from dataclasses import dataclass, field
from datetime import datetime
import importlib.util
import traceback
import synnax as sy

try:
    from .TestCase import TestCase, SynnaxConnection, TestStatus
except ImportError:
    # Handle case when running script directly
    from TestCase import TestCase, SynnaxConnection, TestStatus

class STATE(Enum):
    """Enum representing the status of the test conductor."""
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
    status: TestStatus
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
    test_case: str
    parameters: Dict[str, Any] = field(default_factory=dict)

class Test_Conductor:
    """
    Test conductor that manages the execution of test sequences.
    
    Features:
    - Loads test sequences from configuration files
    - Executes test cases (sequentially or randomly)
    - Monitors execution of test cases (async)
    - Can kill tests if needed (timeout or manual intervention)
    - Provides real-time status updates (async)
    """
    
    def __init__(self, 
                 name: str, 
                 server_address: str = "localhost", 
                 port: int = 9090,
                 username: str = "synnax", 
                 password: str = "seldon", 
                 secure: bool = False,
                 ):
        """
        Initialize the Test Conductor.
        
        Args:
            server_address: Synnax server address
            port: Synnax server port
            username: Authentication username
            password: Authentication password
            secure: Whether to use secure connection
        """

        # Should we use the transport or authentication client address
        # instead of a uuid?
        if name is None:
            # Generate a 6-character random alphanumeric string
            random_id = ''.join(random.choices(string.ascii_lowercase + string.digits, k=6))
            self.name = self.__class__.__name__.lower() + "_" + random_id
        else:
            # Validate and sanitize the provided name
            self.name = self._validate_and_sanitize_name(str(name).lower())
        
        
        # Store connection parameters to
        # pass down to test cases
        self.SynnaxConnection = SynnaxConnection(
            server_address=server_address,
            port=port,
            username=username,
            password=password,
            secure=secure,
        )
        
        # Test Conductor Client
        self.client = sy.Synnax(
            host=server_address,
            port=port,
            username=username,
            password=password,
            secure=secure,
        )
        
        # Initialize Test Conductor
        self.state : STATE = STATE.INITIALIZING

        self.test_definitions: List[TestDefinition] = []
        self.test_results: List[TestResult] = []
        self.current_test: Optional[TestCase] = None
        self.current_test_thread: Optional[threading.Thread] = None
        self.current_test_start_time: Optional[datetime] = None
        self.timeout_monitor_thread: Optional[threading.Thread] = None
        self._timeout_result: Optional[TestResult] = None
        self.client_manager_thread: Optional[threading.Thread] = None

        # Replace these bools with
        self.is_running = False
        self.should_stop = False
        
        # Monitoring
        self.status_callbacks: List[Callable[[TestResult], None]] = []
        self.monitor_task: Optional[asyncio.Task] = None
        
        # Setup signal handlers for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        self._start_client_manager_async()
        time.sleep(2) # Wait for client manager to start
    
    def _validate_and_sanitize_name(self, name: str) -> str:
        """
        Validate and sanitize the name to only contain alphanumeric characters, 
        hyphens, and underscores.
        
        Args:
            name: The name to validate and sanitize
            
        Returns:
            Sanitized name containing only allowed characters
            
        Raises:
            ValueError: If the name is empty after sanitization
        """
        # Remove any characters that aren't alphanumeric, hyphen, or underscore
        sanitized = re.sub(r'[^a-zA-Z0-9_-]', '', name)
        
        # Check if the sanitized name is empty
        if not sanitized:
            raise ValueError("Name must contain at least one alphanumeric character, hyphen, or underscore")
        
        # Ensure name doesn't start or end with hyphen/underscore (optional good practice)
        sanitized = sanitized.strip('_-')
        
        if not sanitized:
            raise ValueError("Name cannot consist only of hyphens and underscores")
            
        return sanitized

    def _start_client_manager_async(self) -> None:
        """Start the client manager in a separate daemon thread."""
        self.client_manager_thread = threading.Thread(
            target=self._client_manager,
            daemon=True,
            name=f"{self.name}_client_manager"
        )
        self.client_manager_thread.start()
        print(f"{self.name} > Client manager started (async)")
    
    def _client_manager(self) -> None:
        
        loop = sy.Loop(1) # 1Hz

        """
        Define Test Conductor Channels
        """
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

        test_case_index = self.client.channels.create(
            name=f"{self.name}_test_case_index",
            data_type=sy.DataType.UINT32,
            index=time.key,
            retrieve_if_name_exists=True,
        )   

        """
        Initialize Test Conductor Telemetry
        """
        start_time = sy.TimeStamp.now()

        self.tlm = {
            f"{self.name}_time": start_time,
            f"{self.name}_uptime": 0,
            f"{self.name}_state": STATE.INITIALIZING.value,
            f"{self.name}_test_case_count": 0,
            f"{self.name}_test_case_index": 0,
        }
        
        """
        Open Test Conductor Writer
        """
        with self.client.open_writer(
            start=start_time,
            channels=[
                time,
                uptime,
                state,
                test_case_count,
                test_case_index,
            ],
            name=self.name,
            enable_auto_commit=True,
        ) as writer:
            # Write initial state
            writer.write(self.tlm)

            while loop.wait() and not self.should_stop:
                """
                Main writer loop
                """
                now = sy.TimeStamp.now()
                uptime_value = (now - start_time)/1E9
                
                self.tlm[f"{self.name}_time"] = now
                self.tlm[f"{self.name}_uptime"] = uptime_value
                self.tlm[f"{self.name}_state"] = self.state.value

                # Write Tlm 
                writer.write(self.tlm)

                # Check for shutdown or completion
                if self.state in [STATE.SHUTDOWN, STATE.COMPLETED]:
                    self.state = STATE.COMPLETED
                    self.tlm[f"{self.name}_state"] = self.state.value
                    writer.write(self.tlm)
                    break

            print(f"{self.name} > Client manager shutting down")
    
    def load_test_sequence(self, sequence: str=None) -> None:
        """
        Load test sequence from a JSON configuration file.
        Args:
            sequence: Path to the test sequence JSON file (can be relative or absolute)
        
        Expected format:
        {
            "tests": [
                {
                    "test_case": "testcases.check_connection_basic",
                    "parameters": {"param1": "value1"}, # Optional
                }
            ]
        }
        """
        self.state = STATE.LOADING
        
        if sequence is None:
            raise ValueError("Path to JSON Sequence file is required (--sequence)")

        # Convert to Path object and resolve relative paths
        sequence_path = Path(sequence)
        print(f"Sequence path: {sequence_path}")
        # If it's a relative path, resolve it relative to the current working directory
        if not sequence_path.is_absolute():
            sequence_path = sequence_path.resolve()
        
        # Check if the resolved path exists
        if not sequence_path.exists():
            # Try some common relative locations if the direct path doesn't work
            possible_paths = [
                Path.cwd() / sequence,  # Current working directory
                Path(__file__).parent / sequence,  # Same directory as this script
                Path(__file__).parent.parent / sequence,  # Parent directory of this script
            ]
            
            # Find the first path that exists
            found_path = None
            for path in possible_paths:
                if path.exists():
                    found_path = path.resolve()
                    break
            
            if found_path:
                sequence_path = found_path
                print(f"Found sequence file at: {sequence_path}")
            else:
                raise FileNotFoundError(
                    f"Test sequence file not found: {sequence}\n"
                    f"Tried paths:\n" + 
                    "\n".join(f"  - {p}" for p in [sequence_path] + possible_paths)
                )
        
        print(f"{self.name} > Loading test campaign from: {sequence_path}")
        time.sleep(2)
        with open(sequence_path, 'r') as f:
            sequence_data = json.load(f)
        
        self.test_definitions = []
        for test in sequence_data.get("tests", []):
            test_def = TestDefinition(
                test_case=test["case"],
                parameters=test.get("parameters", {}),
            )
            self.test_definitions.append(test_def)
        
        # Randomize the test order if specified
        ordering = "Sequential"
        if "sequence_order" in sequence_data:
            if sequence_data["sequence_order"].lower() == "random":
                ordering = "Random"
                random.shuffle(self.test_definitions)

        print(f"{self.name} > Loaded {len(self.test_definitions)} tests ({ordering}) from {sequence_path}")
    
    def run_sequence(self) -> List[TestResult]:
        """
        Run the entire test case set.
        
        Returns:
            List of TestResult objects for all executed tests
        """
        if not self.test_definitions:
            raise ValueError("No test sequence loaded. Call load_test_sequence() first or ensure test sequence JSON is not empty.")
        
        self.state = STATE.RUNNING

        self.is_running = True
        self.should_stop = False
        self.test_results = []
        
        # Start timeout monitoring thread
        self.timeout_monitor_thread = threading.Thread(
            target=self._timeout_monitor_thread,
            args=(1.0,),  # Check every 1 second
            daemon=True
        )
        self.timeout_monitor_thread.start()
        
        print(f"Starting execution of {len(self.test_definitions)} tests...")
        
        for i, test_def in enumerate(self.test_definitions):
            if self.should_stop:
                print("Test execution stopped by user request")
                break
            
            print(f"[{i+1}/{len(self.test_definitions)}] Running test: {test_def.test_case}")
            
            # Run test in a separate thread
            result_container = []
            test_thread = threading.Thread(
                target=self._test_runner_thread,
                args=(test_def, result_container)
            )
            
            self.current_test_thread = test_thread
            test_thread.start()
            
            # Wait for test completion (timeout monitoring is handled by separate thread)
            test_thread.join()
            
            # Get the result from the test execution
            if result_container:
                result = result_container[0]
            else:
                result = TestResult(
                    test_name=test_def.test_case,
                    status=TestStatus.FAILED,
                    error_message="Unknown error - no result returned"
                )
            
            self.test_results.append(result)
            self.current_test_thread = None
        
        self.is_running = False
        self._print_summary()
        return self.test_results

    def wait_for_completion(self) -> None:
        """
        Wait for all async processes to complete before allowing main to exit.
        This ensures proper cleanup and prevents premature termination.
        """
        self.state = STATE.SHUTDOWN
        print(f"{self.name} > Shutting down async processes...")
        
        # Wait for client manager thread to finish
        if self.client_manager_thread and self.client_manager_thread.is_alive():
            print(f"{self.name} > Waiting for client manager to shutdown...")
            self.client_manager_thread.join()
            print(f"{self.name} > Client manager has stopped")
        
        # Wait for any other threads
        if self.timeout_monitor_thread and self.timeout_monitor_thread.is_alive():
            print(f"{self.name} > Waiting for timeout monitor to shutdown...")
            self.timeout_monitor_thread.join()
        
        if self.current_test_thread and self.current_test_thread.is_alive():
            print(f"{self.name} > Waiting for current test to complete...")
            self.current_test_thread.join()
        
        self.state = STATE.COMPLETED
        print(f"{self.name} > All async processes have completed")
    
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
        """Dynamically load a test class from its module path."""
        try:
            spec = importlib.util.spec_from_file_location(
                test_def.module_path, 
                f"{test_def.module_path.replace('.', '/')}.py"
            )
            if spec is None:
                raise ImportError(f"Cannot find module: {test_def.module_path}")
            
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            
            test_class = getattr(module, test_def.class_name)
            if not issubclass(test_class, TestCase):
                raise TypeError(f"{test_def.class_name} is not a subclass of TestCase")
            
            return test_class
        except Exception as e:
            raise ImportError(f"Failed to load test class {test_def.class_name} from {test_def.module_path}: {e}")
    
    def _execute_single_test(self, test_def: TestDefinition) -> TestResult:
        """Execute a single test case."""
        result = TestResult(
            test_name=test_def.test_case,
            status=TestStatus.PENDING,
            start_time=datetime.now()
        )
        
        try:
            # Load and instantiate the test class
            test_class = self._load_test_class(test_def)
            test_instance = test_class(
                SynnaxConnection=self.SynnaxConnection,
                **test_def.parameters
            )
            
            self.current_test = test_instance
            self.current_test_start_time = datetime.now()
            result.status = TestStatus.RUNNING
            result.start_time = self.current_test_start_time
            self._notify_status_change(result)
            
            # Execute the test
            test_instance.execute()
            
            # Check if test was killed/timed out during execution
            if self._timeout_result is not None:
                result = self._timeout_result
                self._timeout_result = None
            else:
                result.status = TestStatus.COMPLETED
            
        except Exception as e:
            # Check if test was killed/timed out during exception
            if self._timeout_result is not None:
                result = self._timeout_result
                self._timeout_result = None
            else:
                result.status = TestStatus.FAILED
                result.error_message = str(e)
                print(f"Test {test_def.test_case} failed: {e}")
                traceback.print_exc()
        
        finally:
            result.end_time = datetime.now()
            self.current_test = None
            self.current_test_start_time = None
            self._notify_status_change(result)
        
        return result
    
    def _test_runner_thread(self, test_def: TestDefinition, result_container: List[TestResult]) -> None:
        """Thread function for running a single test."""
        result = self._execute_single_test(test_def)
        result_container.append(result)
    
    def _timeout_monitor_thread(self, monitor_interval: float = 1.0) -> None:
        """
        Monitor thread that periodically checks if current test has exceeded its Expected_Timeout.
        
        Args:
            monitor_interval: How often to check timeout (in seconds)
        """
        while self.is_running and not self.should_stop:
            if (self.current_test is not None and 
                self.current_test_start_time is not None and 
                hasattr(self.current_test, 'Expected_Timeout') and
                self.current_test.Expected_Timeout > 0):  # Only monitor if timeout is set (not -1)
                
                elapsed_time = (datetime.now() - self.current_test_start_time).total_seconds()
                
                if elapsed_time > self.current_test.Expected_Timeout:
                    print(f"Test exceeded Expected_Timeout ({self.current_test.Expected_Timeout}s). "
                          f"Elapsed time: {elapsed_time:.1f}s. Killing test...")
                    self.kill_current_test()
                    break
            
            time.sleep(monitor_interval)
    
    def kill_current_test(self) -> bool:
        """
        Kill the currently running test.
        
        Returns:
            True if a test was killed, False if no test was running
        """
        if self.current_test is None:
            return False
        
        print(f"Killing current test...")
        
        # Create a timeout result if this is a timeout kill
        if self.current_test_start_time:
            elapsed_time = (datetime.now() - self.current_test_start_time).total_seconds()
            expected_timeout = getattr(self.current_test, 'Expected_Timeout', -1)
            
            # Determine if this is a timeout or manual kill
            if expected_timeout > 0 and elapsed_time > expected_timeout:
                status = TestStatus.TIMEOUT
                error_msg = f"Test exceeded Expected_Timeout ({expected_timeout}s). Elapsed: {elapsed_time:.1f}s"
            else:
                status = TestStatus.KILLED
                error_msg = "Test was manually killed"
            
            # Create timeout/kill result
            current_test_name = "unknown_test"
            # Try to get test name from current test results or test instance
            if self.test_results and self.test_results[-1].status == TestStatus.RUNNING:
                current_test_name = self.test_results[-1].test_name
            
            timeout_result = TestResult(
                test_name=current_test_name,
                status=status,
                start_time=self.current_test_start_time,
                end_time=datetime.now(),
                error_message=error_msg
            )
            
            # This will be picked up by the test runner thread
            self._timeout_result = timeout_result
        
        # Force thread termination if needed
        if self.current_test_thread and self.current_test_thread.is_alive():
            # Note: Python doesn't have a clean way to kill threads
            # The test should implement proper cancellation handling
            pass
        
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
    
    def _print_summary(self) -> None:
        """Print a summary of test execution results."""
        if not self.test_results:
            return
        
        passed = sum(1 for r in self.test_results if r.status == TestStatus.COMPLETED)
        failed = sum(1 for r in self.test_results if r.status == TestStatus.FAILED)
        killed = sum(1 for r in self.test_results if r.status == TestStatus.KILLED)
        timeout = sum(1 for r in self.test_results if r.status == TestStatus.TIMEOUT)
        
        print("\n" + "="*50)
        print("TEST EXECUTION SUMMARY")
        print("="*50)
        print(f"Total tests: {len(self.test_results)}")
        print(f"Passed: {passed}")
        print(f"Failed: {failed}")
        print(f"Killed: {killed}")
        print(f"Timeout: {timeout}")
        print("="*50)
        
        for result in self.test_results:
            status_symbol = {
                TestStatus.COMPLETED: "✓",
                TestStatus.FAILED: "✗",
                TestStatus.KILLED: "⚠",
                TestStatus.TIMEOUT: "⏱"
            }.get(result.status, "?")
            
            duration_str = f"({result.duration:.2f}s)" if result.duration else ""
            print(f"{status_symbol} {result.test_name} {duration_str}")
            if result.error_message:
                print(f"    Error: {result.error_message}")
    
    def _signal_handler(self, signum, frame):
        """Handle system signals for graceful shutdown."""
        print(f"\nReceived signal {signum}. Stopping test execution...")
        self.shutdown()
        sys.exit(0)


# Async monitoring functionality
async def monitor_test_execution(conductor: Test_Conductor, 
                                update_interval: float = 1.0) -> None:
    """
    Asynchronously monitor test execution and provide status updates.
    
    Args:
        conductor: The Test_Conductor instance to monitor
        update_interval: How often to check status (in seconds)
    """
    while conductor.is_running:
        status = conductor.get_current_status()
        print(f"Status: {status['completed_tests']}/{status['total_tests']} tests completed")
        if status['current_test']:
            print(f"Currently running: {status['current_test']}")
        
        await asyncio.sleep(update_interval)


# Example usage functions
def create_sample_test_sequence(output_path: str) -> None:
    """Create a sample test sequence file for demonstration."""
    sample_sequence = {
        "tests": [
            {
                "module_path": "testcases.check_connection_basic",
                "parameters": {},
            }
        ]
    }
    
    with open(output_path, 'w') as f:
        json.dump(sample_sequence, f, indent=2)
    
    print(f"Sample test sequence created at: {output_path}")


if __name__ == "__main__":
    # Example usage
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
    conductor = Test_Conductor( name=args.name, 
                                server_address=args.server, 
                                port=args.port,
                                username=args.username,
                                password=args.password,
                                secure=args.secure,
                                )

    try:
        conductor.load_test_sequence(args.sequence)
        # Run tests
        results = conductor.run_sequence()
        
        # Wait for all async processes to complete before exiting
        conductor.wait_for_completion()
        
    except KeyboardInterrupt:
        print("\nKeyboard interrupt received. Shutting down gracefully...")
        conductor.shutdown()
    except Exception as e:
        print(f"Error occurred: {e}")
        conductor.shutdown()
        raise
    finally:
        # Ensure cleanup even if something goes wrong
        print("Main process ending...")

