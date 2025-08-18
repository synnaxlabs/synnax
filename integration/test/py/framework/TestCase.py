#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import logging
import os
import sys
from dataclasses import dataclass, field
from enum import Enum, auto
import re
import threading
import time
from abc import ABC, abstractmethod
from typing import Any


try:
    # Import from the framework module to ensure we get the same class objects``
    sys.path.append(os.path.join(os.path.dirname(__file__), '..'))
    from framework.utils import validate_and_sanitize_name
except ImportError:
    # Handle case when running script directly
    from utils import validate_and_sanitize_name



@dataclass
class SynnaxConnection:
    """Data class representing the Synnax connection parameters."""
    server_address: str
    port: int
    username: str
    password: str
    secure: bool

class STATUS(Enum):
    """Enum representing the status of a test."""
    INITIALIZING = auto()
    RUNNING = auto()
    PENDING = auto()
    PASSED = auto()
    FAILED = auto()
    TIMEOUT = auto()
    KILLED = auto()

class SYMBOLS(Enum):
    PASSED = "✅"      # Green checkmark
    FAILED = "❌"      # Red X
    KILLED = "💀"      # Skull
    TIMEOUT = "⏰"      # Alarm clock
    
    @classmethod
    def get_symbol(cls, status):
        """Get symbol for a given status, with fallback to '?' if not found."""
        try:
            return cls[status.name].value
        except (KeyError, AttributeError):
            return "❓"  # Question mark emoji

class TestCase(ABC):
    """
    Parent class for all test cases in the integration test framework.
    
    This class handles the connection to the Synnax server and provides
    three key lifecycle methods that can be overridden by subclasses:
    - setup(): Called before the test runs
    - run(): The main test logic (must be implemented by subclasses)
    - teardown(): Called after the test runs
    """
    
    def __init__(self, SynnaxConnection: SynnaxConnection, name:str=None, expect: str = "PASSED", **params):

        # Store for test cases to use
        self.SynnaxConnection = SynnaxConnection

        if expect in ["FAILED", "TIMEOUT", "KILLED"]:
            # Use this wisely!
            if expect == "FAILED":
                self.expected_outcome = STATUS.FAILED
            elif expect == "TIMEOUT":
                self.expected_outcome = STATUS.TIMEOUT
            elif expect == "KILLED":
                self.expected_outcome = STATUS.KILLED
        else:
            self.expected_outcome = STATUS.PASSED
        
        """Initialize test case with Synnax server connection."""
        self._status = STATUS.INITIALIZING
        self.params = params
        self.Expected_Timeout: int = -1  # -1 = no timeout
        
        if name is None:
            # Convert PascalCase class name to lowercase with underscores
            self.name = re.sub(r'([a-z0-9])([A-Z])', r'\1_\2', self.__class__.__name__).lower()
        else:
            self.name = validate_and_sanitize_name(name)

        self._setup_logging()

        # Connect to Synnax server
        self.client = sy.Synnax(
            host=SynnaxConnection.server_address,
            port=SynnaxConnection.port,
            username=SynnaxConnection.username,
            password=SynnaxConnection.password,
            secure=SynnaxConnection.secure,
        )
        
        # Default 1Hz loop 
        self.loop = sy.Loop(1)
        self.client_thread = None
        self.should_stop = False
        self.is_running = False
        
        # Create telemetry channels
        self.time_index = self.client.channels.create(
            name=f"{self.name}_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )  
        
        self.tlm = {
            f"{self.name}_time": sy.TimeStamp.now(),
        }
        
        self.add_channel(name="uptime", data_type=sy.DataType.UINT32, initial_value=0)
        self.add_channel(name="state", data_type=sy.DataType.UINT8, initial_value=self._status.value)
    
    def _setup_logging(self) -> None:
        """Setup logging for real-time output (same approach as Test_Conductor)."""
        # Check if running in CI environment
        is_ci = any(env_var in os.environ for env_var in ['CI', 'GITHUB_ACTIONS', 'GITLAB_CI', 'JENKINS_URL'])
        
        # Force unbuffered output in CI environments
        if is_ci:
            sys.stdout.reconfigure(line_buffering=True)
        
        # Create logger for this test case (don't configure root logger)
        self.logger = logging.getLogger(self.name)
        self.logger.setLevel(logging.INFO)
        
        # Remove any existing handlers to avoid duplicates
        for handler in self.logger.handlers[:]:
            self.logger.removeHandler(handler)
        
        # Add single handler
        handler = logging.StreamHandler(sys.stdout)
        handler.setLevel(logging.INFO)
        formatter = logging.Formatter('%(message)s')
        handler.setFormatter(formatter)
        self.logger.addHandler(handler)
        
        # Prevent propagation to root logger to avoid duplicate output
        self.logger.propagate = False
        
        # Force immediate flush for real-time output in CI
        for handler in self.logger.handlers:
            if hasattr(handler.stream, 'flush'):
                handler.flush = lambda h=handler: h.stream.flush()
    
    @property
    def STATUS(self) -> STATUS:
        """Get the current test status."""
        return self._status
    
    @STATUS.setter
    def STATUS(self, value: STATUS) -> None:
        """Set the test status and update telemetry if client is running."""
        self._status = value
        # Update telemetry if client thread is running
        if hasattr(self, 'tlm') and self.is_client_running():
            try:
                self.tlm[f"{self.name}_state"] = value.value
            except Exception:
                pass  # Ignore errors if tlm is not fully initialized
    
    def _log_message(self, message: str) -> None:
        """Log a message to the console with real-time output."""
        self.logger.info(f"{self.name} > {message}")
        
        # Force flush to ensure immediate output in CI
        for handler in self.logger.handlers:
            if hasattr(handler, 'flush'):
                handler.flush()
        
    def add_channel(self, name: str, data_type: sy.DataType, initial_value: Any = None):
        """Create a telemetry channel with name {self.name}_{name}."""
        self.client.channels.create(
            name=f"{self.name}_{name}",
            data_type=data_type,
            index=self.time_index.key,
            retrieve_if_name_exists=True,
        )
        self.tlm[f"{self.name}_{name}"] = initial_value

    def subscribe_to_channels(self, channels: list[str]) -> None:
        """
        Subscribe to the given channels.
        """
        pass

    def setup(self) -> None:
        """Start telemetry client thread."""
        self.is_running = True
        self.should_stop = False
        
        # Start client thread
        self.client_thread = threading.Thread(target=self._client_loop, daemon=True)
        self.client_thread.start()
        time.sleep(1)  # Allow client thread to start
        self._log_message("client thread started")
    
    def _client_loop(self) -> None:
        """Main telemetry client loop running in separate thread."""
        start_time = sy.TimeStamp.now()
        
        try:
            with self.client.open_writer(
                start=start_time,
                channels=list(self.tlm.keys()),
                name=self.name,
                enable_auto_commit=True,
            ) as client:
                while self.loop.wait() and not self.should_stop:
                    now = sy.TimeStamp.now()
                    uptime_value = (now - start_time)/1E9                    

                    # Update telemetry
                    self.tlm[f"{self.name}_time"] = now
                    self.tlm[f"{self.name}_uptime"] = uptime_value
                    self.tlm[f"{self.name}_state"] = self._status.value
                    client.write(self.tlm)

                    # Check for timeout
                    if self.Expected_Timeout > 0 and uptime_value > self.Expected_Timeout:
                        self._status = STATUS.TIMEOUT

                    # Check for completion
                    if self._status in [STATUS.FAILED, STATUS.KILLED, STATUS.TIMEOUT]:
                        self.tlm[f"{self.name}_state"] = self._status.value
                        client.write(self.tlm)
                        break
                
                self._check_expectation()

                # Final write for redundancy
                client.write(self.tlm)
                self._log_message("client thread shutting down")
                
        except Exception as e:
            self._log_message(f"client thread error: {e}")
            self._status = STATUS.FAILED
        finally:
            self.is_running = False 

    @abstractmethod
    def run(self) -> None:
        """
        Main test logic method.
        This method must be implemented by all subclasses.
        """
        pass
    
    def teardown(self) -> None:
        """Cleanup after test execution. Override for custom cleanup logic."""
        
        # Unload configs 
        # or open vents
        # or whatever else
        pass

    def stop_client(self) -> None:
        """Stop client thread and wait for completion."""
        if self.client_thread and self.is_running:
            self.should_stop = True
            self.is_running = False
            
            if self.client_thread.is_alive():
                self.client_thread.join(timeout=5.0)
                if self.client_thread.is_alive():
                    self._log_message("Warning: client thread did not stop within timeout")

        # All done? All done.                
        if self._status == STATUS.PENDING:
            self._status = STATUS.PASSED
    
    def wait_for_client_completion(self, timeout: float = None) -> None:
        """Wait for client thread to complete."""
        if self.client_thread and self.client_thread.is_alive():
            self.client_thread.join(timeout=timeout)
    
    def is_client_running(self) -> bool:
        """Check if client thread is running."""
        return self.client_thread is not None and self.client_thread.is_alive()
    
    def get_client_status(self) -> str:
        """Get client thread status."""
        if self.client_thread is None:
            return "Not started"
        elif self.client_thread.is_alive():
            return "Running"
        else:
            return "Stopped"
    
    def shutdown(self) -> None:
        """Gracefully shutdown test case and stop all threads."""
        self._log_message("Shutting down test case...")
        self._status = STATUS.KILLED
        self.stop_client()
        self._log_message("Test case shutdown complete")
    
    def __enter__(self):
        """Context manager entry point."""
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit point - ensures cleanup."""
        self.shutdown()
    
    def _check_expectation(self) -> None:
        """Check if test met expected outcome and handle failures gracefully."""
        # Convert PENDING to PASSED if no final status set
        if self._status == STATUS.PENDING:
            self._status = STATUS.PASSED

        status_symbol = SYMBOLS.get_symbol(self._status)
        expected_symbol = SYMBOLS.get_symbol(self.expected_outcome)

        # Handle expected outcome logic
        if self._status == STATUS.PASSED:
            if self.expected_outcome == STATUS.PASSED:
                self._log_message(f"PASSED ({status_symbol})")
            else:
                self._status = STATUS.FAILED
                self._log_message(f"FAILED (❌): Expected {expected_symbol}, got {status_symbol}")

        elif self._status == self.expected_outcome:
            self._log_message(f"PASSED (✅): Expected outcome achieved ({status_symbol})")
            self._status = STATUS.PASSED
        elif self._status == STATUS.FAILED:
            self._log_message(f"FAILED ({status_symbol})")
        elif self._status == STATUS.TIMEOUT:
            self._log_message(f"TIMEOUT ({status_symbol}): {self.Expected_Timeout} seconds")
        elif self._status == STATUS.KILLED:
            self._log_message(f"KILLED ({status_symbol})")

    def fail(self) -> None:
        self._status = STATUS.FAILED

    def execute(self) -> None:
        """Execute complete test lifecycle: setup -> run -> teardown."""
        try:
            self._status = STATUS.INITIALIZING
            self.setup()

            self._status = STATUS.RUNNING
            self.run()

            # Set to PENDING only if not in final state
            if self._status not in [STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED]:
                self._status = STATUS.PENDING
            
            self.teardown()
            
            # PASS condition set at final 
            # spot of activity: _client_loop()

        except Exception as e:
            self._status = STATUS.FAILED
            self._log_message(f"EXCEPTION: {e}")
        finally:
            self.stop_client()
            self.wait_for_client_completion()
