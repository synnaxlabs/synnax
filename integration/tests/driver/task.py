#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Base class for driver integration tests.

Provides common functionality for:
- Launching and managing simulator servers
- Creating and retrieving hardware devices
- Server cleanup and teardown
"""

import atexit
import os
import subprocess
import sys
from abc import abstractmethod

import synnax as sy

from driver.devices import SimulatorConfig
from driver.driver import ChannelConfig, Driver
from framework.test_case import TestCase


class TaskCase(TestCase):
    """
    Base class for driver task lifecycle tests.

    Tests standard lifecycle: create, start/stop, reconfigure, delete task and device.

    Subclasses must:
    - Set TASK_NAME, TASK_KEY, and CHANNELS as class attributes
    - Implement create_task() factory method
    - Optionally override setup() for custom configuration (e.g., matrix parameters)
    - Optionally override run() for custom test logic

    Environment Variables:
    - SYNNAX_DRIVER_RACK: Override the driver rack name (default: "Node 1 Embedded Driver")
      Can be set via command line: --driver "My Custom Rack Name" or -d "My Custom Rack Name"
    """

    # Task configuration (must be set by subclasses)
    RACK_NAME: str = os.environ.get("SYNNAX_DRIVER_RACK", "Node 1 Embedded Driver")
    TASK_NAME: str = ""
    TASK_KEY: str = ""
    CHANNELS: list[ChannelConfig] = []

    # Default test parameters
    SAMPLE_RATE: sy.Rate = 50 * sy.Rate.HZ
    STREAM_RATE: sy.Rate = 10 * sy.Rate.HZ
    TEST_DURATION: sy.TimeSpan = 1 * sy.TimeSpan.SECOND

    # Sim server process variables
    simulator: SimulatorConfig | None = None
    simulator_process: subprocess.Popen[bytes] | None = None

    # Task lifecycle test configuration
    tsk: sy.Task | None = None
    channel_names: list[str] = []

    @abstractmethod
    def create_task(
        self,
        device: sy.Device,
        channels: list[sy.Channel],
        channel_metadata: list[ChannelConfig],
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.Task:
        """Factory method to create protocol-specific task.

        Args:
            device: Synnax device to configure task for
            channels: Created Synnax channels
            channel_metadata: List of dicts with protocol-specific config
            task_name: Name for the task
            sample_rate: Sampling rate for the task
            stream_rate: Streaming rate for the task

        Returns:
            Configured protocol-specific task (e.g., modbus.ReadTask)
        """
        pass

    def setup(self) -> None:
        """Start simulator, connect to device, and configure task."""
        if self.simulator is not None:
            self._start_simulator()
            Driver.connect_device(
                client=self.client,
                rack_name=self.RACK_NAME,
                device_factory=self.simulator.device_factory,
            )

        # Create channels
        assert self.simulator is not None
        device, channels, self.channel_names = Driver.create_channels(
            client=self.client,
            device_name=self.simulator.device_name,
            task_key=self.TASK_KEY,
            channel_configs=self.CHANNELS,
        )

        # Create task using child implementation
        self.tsk = self.create_task(
            device=device,
            channels=channels,
            channel_metadata=self.CHANNELS,
            task_name=self.TASK_NAME,
            sample_rate=self.SAMPLE_RATE,
            stream_rate=self.STREAM_RATE,
        )

        # Configure task in Synnax
        try:
            self.client.hardware.tasks.configure(self.tsk)
            self.log(f"Task '{self.TASK_NAME}' configured")
        except Exception as e:
            self.fail(f"Task configuration failed: {e}")
            return

    def run(self) -> None:
        """Execute the standard task lifecycle test."""
        if self.tsk is None:
            self.fail("Task not configured. Subclass must set self.tsk in setup()")
            return

        client = self.client
        tsk = self.tsk

        self.log("Test 0 - Verify Task Exists")
        Driver.assert_task_exists(client, tsk.key)
        Driver.assert_channel_names(client, tsk, self.channel_names)

        self.log("Test 1 - Start and Stop")
        Driver.assert_sample_count(client, tsk, duration=self.TEST_DURATION)

        # SY-3310: OPC Read Array - rapid restart race condition
        sy.sleep(0.2)

        self.log("Test 2 - Reconfigure Task")
        new_rate = int(self.SAMPLE_RATE * 2)
        tsk.config.sample_rate = new_rate
        client.hardware.tasks.configure(tsk)
        Driver.assert_sample_count(client, tsk, duration=self.TEST_DURATION)

        self.log("Test 3 - Delete Task")
        client.hardware.tasks.delete(tsk.key)
        Driver.assert_task_deleted(client, tsk.key)

        self.log("Test 4 - Delete Device")
        device = client.hardware.devices.retrieve(key=tsk.config.device)
        client.hardware.devices.delete([device.key])
        Driver.assert_device_deleted(client, device.key)

    def _start_simulator(self) -> None:
        """Start the simulator server."""
        assert self.simulator is not None
        server_script = self.repo_root / self.simulator.server_script

        if not server_script.exists():
            raise FileNotFoundError(f"Server script not found: {server_script}")

        # Launch the simulator server
        self.simulator_process = subprocess.Popen(
            [sys.executable, str(server_script)],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        atexit.register(self._cleanup_simulator)

        self.log(f"Server started with PID: {self.simulator_process.pid}")
        sy.sleep(self.simulator.startup_delay_seconds)

        if self.simulator_process.poll() is not None:
            _, stderr = self.simulator_process.communicate()
            error_msg = stderr.decode() if stderr else "No error output"
            raise RuntimeError(
                f"Server failed to start (exit code: {self.simulator_process.returncode})\n{error_msg}"
            )

    def _cleanup_simulator(self, log: bool = False) -> None:
        """Terminate simulator server process (internal use only)."""

        # Check if server process exists and hasn't been cleaned up already
        if self.simulator_process is None:
            return

        if self.simulator_process.poll() is not None:
            self.simulator_process = None
            return

        try:
            self.simulator_process.terminate()

            try:
                self.simulator_process.wait(timeout=5 if log else 3)

            except subprocess.TimeoutExpired:
                self.simulator_process.kill()
                self.simulator_process.wait(timeout=2 if log else 1)
                self.log("Server killed")

        except Exception as e:
            self.fail(f"Error during server cleanup: {e}")
            raise RuntimeError(f"Error during server cleanup: {e}")

        finally:
            self.simulator_process = None

    def teardown(self) -> None:
        """Terminate the simulator server if one was started."""
        if self.simulator_process is not None:
            self._cleanup_simulator()
