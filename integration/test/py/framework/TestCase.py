#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from dataclasses import dataclass, field
from enum import Enum, auto
import random
import threading
import time
from abc import ABC, abstractmethod
from typing import Any, Optional


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


class TestCase(ABC):
    """
    Parent class for all test cases in the integration test framework.
    
    This class handles the connection to the Synnax server and provides
    three key lifecycle methods that can be overridden by subclasses:
    - setup(): Called before the test runs
    - run(): The main test logic (must be implemented by subclasses)
    - teardown(): Called after the test runs
    """
    
    def __init__(self, SynnaxConnection: SynnaxConnection):
        """
        Initialize the test case with connection to Synnax server.
        
        Args:
            SynnaxConnection: The connection parameters for the Synnax server
        """
        
        self._status = STATUS.INITIALIZING
        
        # Expected timeout in seconds (-1 means no timeout specified)
        self.Expected_Timeout: int = -1

        # Generate name
        self.name = self.__class__.__name__.lower()

        # Connect to Synnax server
        self.client = sy.Synnax(
            host=SynnaxConnection.server_address,
            port=SynnaxConnection.port,
            username=SynnaxConnection.username,
            password=SynnaxConnection.password,
            secure=SynnaxConnection.secure,
        )
        
        # Default rate
        self.loop = sy.Loop(1)
        
        # Thread management
        self.writer_thread = None
        self.should_stop = False
        self.is_running = False
        
        """
        Define Test case channels
        """
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
    
    @property
    def STATUS(self) -> STATUS:
        """Get the current test status."""
        return self._status
    
    @STATUS.setter
    def STATUS(self, value: STATUS) -> None:
        """Set the test status and update telemetry if writer is running."""
        self._status = value
        # Update telemetry if writer thread is running
        if hasattr(self, 'tlm') and self.is_writer_running():
            try:
                self.tlm[f"{self.name}_state"] = value.value
            except Exception:
                pass  # Ignore errors if tlm is not fully initialized



    def add_channel(self, name: str, data_type: sy.DataType, initial_value: Any = None):

        """
        This function Exists for your convenience.
        It will create a channel with the name {self.name}_{name}
        and the data type {data_type}
        """

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
        """
        Setup logic. 
        Creates writer with initialized tlm channels and runs it in a separate thread.
        """
        self.is_running = True
        self.should_stop = False
        
        # Start writer thread
        self.writer_thread = threading.Thread(target=self._writer_loop, daemon=True)
        self.writer_thread.start()

        # Give the writer thread a chance to start
        time.sleep(2)
        print(f"{self.name} > Writer thread started")
    
    def _writer_loop(self) -> None:
        """
        Main writer loop that runs in a separate thread.
        """
        start_time = sy.TimeStamp.now()
        
        try:
            with self.client.open_writer(
                start=start_time,
                channels=list(self.tlm.keys()),
                name=self.name,
                enable_auto_commit=True,
            ) as writer:
                while self.loop.wait() and not self.should_stop:
                    """
                    Main writer loop
                    """
                    now = sy.TimeStamp.now()
                    uptime_value = (now - start_time)/1E9
                    
                    # Update State telemetry 
                    self.tlm[f"{self.name}_time"] = now
                    self.tlm[f"{self.name}_uptime"] = uptime_value
                    self.tlm[f"{self.name}_state"] = self._status.value

                    # Write Tlm 
                    writer.write(self.tlm)

                    # Check for timeout
                    if self.Expected_Timeout > 0 and uptime_value > self.Expected_Timeout:
                        self._status = STATUS.TIMEOUT
                        self.tlm[f"{self.name}_state"] = self._status.value
                        writer.write(self.tlm)
                        break

                    # Check for shutdown or completion
                    if self._status in [STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED]:
                        # Update state telemetry        
                        self.tlm[f"{self.name}_state"] = self._status.value
                        writer.write(self.tlm)
                        break
                
                # Final write for redundancy
                writer.write(self.tlm)
                print(f"{self.name} > Writer thread shutting down")
                
        except Exception as e:
            print(f"{self.name} > Writer thread error: {e}")
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

        """
        Teardown method called after the test runs.
        Override this method in subclasses to implement test-specific cleanup logic.
        """
        # Wait for writer thread to complete before finishing

        self.stop_writer()

        # Happy path:
        if self._status == STATUS.PENDING:
            self._status = STATUS.PASSED

        # Timeout path:
        elif self._status == STATUS.TIMEOUT:
            print(f"{self.name} > TIMEOUT ({self.Expected_Timeout} seconds)")   

        # Failed path:
        elif self._status == STATUS.FAILED:
            print(f"{self.name} > FAILED")

        
    
    def stop_writer(self) -> None:
        """
        Stop the writer thread and wait for it to complete.
        """
        if self.writer_thread and self.is_running:
            print(f"{self.name} > Stopping writer thread...")
            self.should_stop = True
            self.is_running = False
            
            # Wait for writer thread to complete (with timeout)
            if self.writer_thread.is_alive():
                self.writer_thread.join(timeout=5.0)
                if self.writer_thread.is_alive():
                    print(f"{self.name} > Warning: Writer thread did not stop within timeout")
                else:
                    print(f"{self.name} > Writer thread stopped successfully")
    
    def wait_for_writer_completion(self, timeout: float = None) -> None:
        """
        Wait for the writer thread to complete.
        
        Args:
            timeout: Maximum time to wait in seconds. If None, wait indefinitely.
        """
        if self.writer_thread and self.writer_thread.is_alive():
            self.writer_thread.join(timeout=timeout)
    
    def is_writer_running(self) -> bool:
        """
        Check if the writer thread is currently running.
        
        Returns:
            True if the writer thread is running, False otherwise.
        """
        return self.writer_thread is not None and self.writer_thread.is_alive()
    
    def get_writer_status(self) -> str:
        """
        Get the current status of the writer thread.
        
        Returns:
            String describing the writer thread status.
        """
        if self.writer_thread is None:
            return "Not started"
        elif self.writer_thread.is_alive():
            return "Running"
        else:
            return "Stopped"
    
    def shutdown(self) -> None:
        """
        Gracefully shutdown the test case and stop all running threads.
        """
        print(f"{self.name} > Shutting down test case...")
        self._status = STATUS.KILLED
        self.stop_writer()
        print(f"{self.name} > Test case shutdown complete")
    
    def __enter__(self):
        """Context manager entry point."""
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit point - ensures cleanup."""
        self.shutdown()
    
    def execute(self) -> None:
        """
        Execute the complete test lifecycle: setup -> run -> teardown.
        This method should typically be called to run the test.
        """
        try:
            # Update the status here so you can forget
            # to call .super() in an override without concern.

            self._status = STATUS.INITIALIZING
            self.setup()

            self._status = STATUS.RUNNING
            self.run()

            # Only set to PENDING if not already in a final state
            if self._status not in [STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED]:
                self._status = STATUS.PENDING
            self.teardown()
            
            # PASS condition set within the last spot 
            # of activity: _writer_loop() (but don't override TIMEOUT/FAILED/KILLED)

        except Exception as e:
            self._status = STATUS.FAILED
            print(f"{self.name} > Test execution failed: {e}")
        finally:
            self.wait_for_writer_completion()
