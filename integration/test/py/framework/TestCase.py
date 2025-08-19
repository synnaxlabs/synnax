#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# Suppress WebSocket keepalive ping warnings
import warnings
warnings.filterwarnings("ignore", message=".*keepalive ping.*")
warnings.filterwarnings("ignore", message=".*timed out while closing connection.*")

from selectors import SelectorKey
import synnax as sy
import traceback
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

import sys

try:
    # Import from the framework module to ensure we get the same class objects``
    sys.path.append(os.path.join(os.path.dirname(__file__), '..'))
    from framework.utils import validate_and_sanitize_name, WebSocketErrorFilter, ignore_websocket_errors
except ImportError:
    # Handle case when running script directly
    from utils import validate_and_sanitize_name, WebSocketErrorFilter, ignore_websocket_errors


# Error filter
sys.excepthook = ignore_websocket_errors
sys.stderr = WebSocketErrorFilter()


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
    
    This class handles the connection to Synnax server and provides
    three key lifecycle methods that can be overridden by subclasses:
    - setup(): Called before the test runs
    - run(): The main test logic (must be implemented by subclasses)
    - teardown(): Called after the test runs
    """
    
    # Configuration constants
    DEFAULT_READ_TIMEOUT = 1
    DEFAULT_LOOP_RATE = 1
    WEBSOCKET_RETRY_DELAY = 1
    MAX_CLEANUP_RETRIES = 3
    CLIENT_THREAD_START_DELAY = 1
    DEFAULT_TIMEOUT_LIMIT = -1
    DEFAULT_MANUAL_TIMEOUT = -1
    
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
        self._Timeout_Limit: int = self.DEFAULT_TIMEOUT_LIMIT  # -1 = no timeout
        self._Manual_Timeout: int = self.DEFAULT_MANUAL_TIMEOUT
        self.read_frame = None
        self.read_timeout = self.DEFAULT_READ_TIMEOUT
        
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
        
        # Default loop rate
        self.loop = sy.Loop(self.DEFAULT_LOOP_RATE)
        self.client_thread = None
        self._should_stop = False
        self.is_running = False
        
        self.subscribed_channels = []
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
        
    def add_channel(self, name: str, data_type: sy.DataType, initial_value: Any = None, append_name: bool = True):
        
        """Create a telemetry channel with name {self.name}_{name}."""
        if append_name:
            tlm_name = f"{self.name}_{name}"
        else:
            tlm_name = name
            
        self.client.channels.create(
            name=tlm_name,
            data_type=data_type,
            index=self.time_index.key,
            retrieve_if_name_exists=True,
        )
    
        self.tlm[tlm_name] = initial_value

    def subscribe(self, channels) -> None:
        """ Subscribe to channels. Can take either a single channel name or a list of channels."""
        if isinstance(channels, str):
            # Single channel name
            self.subscribed_channels.append(channels)
        elif isinstance(channels, list):
            # List of channels - extend the list
            self.subscribed_channels.extend(channels)
        else:
            # Convert to string if it's another type
            self.subscribed_channels.append(str(channels))
        return None

    def setup(self) -> None:
        """Start telemetry client thread."""
        self.is_running = True
        self._should_stop = False
        
        # Start client thread
        self.client_thread = threading.Thread(target=self._client_loop, daemon=True)
        self.client_thread.start()
        time.sleep(self.CLIENT_THREAD_START_DELAY)  # Allow client thread to start
        self._log_message("client thread started")

    def _client_loop(self) -> None:
        """Main telemetry client loop running in separate thread."""
        
        # For simplicity, read/write will both happen here.
        
        start_time = sy.TimeStamp.now()
        
        try:  
            self.read_frame = {}
            for channel in self.subscribed_channels:
                self.read_frame[channel] = None
                
            with self.client.open_streamer(self.subscribed_channels) as streamer:
                with self.client.open_writer(
                    start=start_time,
                    channels=list(self.tlm.keys()),
                    name=self.name,
                    enable_auto_commit=True,
                ) as client:
                    while self.loop.wait() and not self._should_stop:

                        now = sy.TimeStamp.now()
                        uptime_value = (now - start_time)/1E9    
                        # Update telemetry
                        self.tlm[f"{self.name}_time"] = now
                        self.tlm[f"{self.name}_uptime"] = uptime_value
                        self.tlm[f"{self.name}_state"] = self._status.value  

                        # Check for timeout
                        if self._Timeout_Limit > 0 and uptime_value > self._Timeout_Limit:
                            self._status = STATUS.TIMEOUT

                        # Check for completion
                        if self._status in [STATUS.FAILED, STATUS.KILLED, STATUS.TIMEOUT]:
                            self.tlm[f"{self.name}_state"] = self._status.value                                

                        try:
                            client.write(self.tlm)   

                            self.frame_in = streamer.read(self.read_timeout)
                            if self.frame_in != None:
                                #self.read_frame = self.frame_raw
                                for key, value in self.frame_in.items():
                                    self.read_frame[key] = value[-1]

                        except Exception as e:
                            if self._is_websocket_error(e):
                                time.sleep(self.WEBSOCKET_RETRY_DELAY)
                            else:
                                self.STATUS = STATUS.FAILED
                                raise e

                    # Final write for redundancy
                    #client.write(self.tlm)
                    self._log_message("client thread shutting down")
                
        except Exception as e:
            if self._is_websocket_error(e):
                pass
            else:
                self._log_message(f"client thread error: {e}\n {traceback.format_exc()}")
                self._status = STATUS.FAILED
                raise e
        finally:
            self._should_stop = True
            
            # Graceful cleanup - ignore WebSocket close errors
            try:
                if 'client' in locals() and client:
                    client.close()
            except Exception as cleanup_error:
                if self._is_websocket_error(cleanup_error):
                    pass
                else:
                    self._log_message(f"Cleanup error: {cleanup_error}")
                    
            try:
                if 'streamer' in locals() and streamer:
                    streamer.close()
            except Exception as cleanup_error:
                if self._is_websocket_error(cleanup_error):
                    pass
                else:
                    self._log_message(f"Cleanup error: {cleanup_error}")
            
            #self.is_running = False 
        self._should_stop = True

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
    

    def write_tlm(self, channel:str, value: Any = None) -> None:
        """Write values to telemetry dictionary. Can take single key-value or dict of multiple channels."""
        #if isinstance(channel, self.tlm.keys()):
        self.tlm[channel] = value
        #else:
        #    raise KeyError(f"Key {channel} not found in telemetry dictionary ({self.tlm.keys()})")

    def read_tlm(self, key: str, default: Any = None) -> Any:

        try:
            return self.read_frame.get(key, default)
        except:
            return default

    def get_state(self, key: str, default: Any = None) -> Any:
        """
        Easily get state of this object.

        - self.name + "state"
        - self.name + "time"
        - self.name + "uptime"    
            """

        name_ch = self.name + "_" + key
        return self.tlm.get(name_ch, default)

    @property
    def uptime(self) -> float:
        """Get the uptime of the test case."""
        return self.tlm.get(f"{self.name}_uptime", -1)

    @property
    def time(self) -> float:
        """Get the uptime of the test case."""
        return self.tlm.get(f"{self.name}_time", -1)
    
    @property
    def state(self) -> float:
        """Get the state of the test case."""
        return self.tlm.get(f"{self.name}_state", -1)

    @property
    def manual_timeout(self) -> int:
        """Get the manual timeout of the test case."""
        return self._Manual_Timeout
    
    @property
    def should_stop(self) -> bool:
        condition_1 = (self._Manual_Timeout >= 0 and self.uptime > self._Manual_Timeout)
        condition_2 = self._should_stop
        
        return condition_1 or condition_2

    @property
    def should_continue(self) -> bool:
        return not self.should_stop

    def set_manual_timeout(self, value: int) -> None:
        """Set the manual timeout of the test case."""
        self._Manual_Timeout = value
        
    def configure(self, **kwargs) -> None:
        """Configure test case parameters.
        
        Args:
            read_timeout: Timeout for read operations (default: 1)
            loop_rate: Loop frequency in Hz (default: 1)
            websocket_retry_delay: Delay before retrying WebSocket operations (default: 1)
            timeout_limit: Maximum execution time in seconds (default: -1, no limit)
            manual_timeout: Manual timeout value (default: -1, no limit)
        """
        if 'read_timeout' in kwargs:
            self.read_timeout = kwargs['read_timeout']
        if 'loop_rate' in kwargs:
            self.loop = sy.Loop(kwargs['loop_rate'])
        if 'timeout_limit' in kwargs:
            self._Timeout_Limit = kwargs['timeout_limit']
        if 'manual_timeout' in kwargs:
            self._Manual_Timeout = kwargs['manual_timeout']
        
    def _is_websocket_error(self, error: Exception) -> bool:
        """Check if an exception is a WebSocket-related error that should be ignored."""
        error_str = str(error)
        return any(phrase in error_str for phrase in [
            "1011", 
            "keepalive ping timeout", 
            "keepalive ping failed",
            "timed out while closing connection",
            "ConnectionClosedError",
            "WebSocketException"
        ])


    def stop_client(self) -> None:
        """Stop client thread and wait for completion."""
        if self.client_thread and self.is_running:
            self._should_stop = True
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
            self._log_message(f"TIMEOUT ({status_symbol}): {self._Timeout_Limit} seconds")
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
            
            # PASSED set in _check_expectation()

        except Exception as e:
            if self._is_websocket_error(e):
                pass
            else:
                self._status = STATUS.FAILED
                self._log_message(f"EXCEPTION: {e}")
        finally:
            self.stop_client()
            self.wait_for_client_completion()
            self._check_expectation()
