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
import subprocess
import sys
from abc import abstractmethod
from typing import Callable

import synnax as sy
from synnax.hardware.device import Device as SynnaxDevice

from framework.test_case import TestCase
from tests.driver.devices import Simulator


class Driver(TestCase):
    """
    Abstract base class for driver integration tests.

    Usage patterns:

    1. Simulator tests (Modbus/OPC UA):
       - Set simulator: Simulator.MODBUS or Simulator.OPCUA
       - In setup(), simulator starts automatically
       - Device created from simulator's device_factory

       Example:
           from tests.driver.driver import Driver
           from tests.driver.devices import Simulator

           class ModbusBasic(Driver):
               simulator = Simulator.MODBUS

    2. Hardware tests without simulator:
       - Leave simulator as None
       - Call self.connect_device(KnownDevices.my_device) in run()

       Example:
           from tests.driver.driver import Driver
           from tests.driver.devices import KnownDevices

           class NITest(Driver):
               def run(self):
                   device = self.connect_device(KnownDevices.ni_daq_6001)

    3. Existing hardware (already in cluster):
       - Leave simulator as None
       - Manually retrieve device using self.client.hardware.devices.retrieve()

    Subclasses must implement:
    - run(): Test execution logic
    """
    
    # Sim server process variables
    simulator: Simulator | None = None
    server_process: subprocess.Popen[bytes] | None = None

    def setup(self) -> None:
        """Start simulator and connect to device (if simulator configured)."""
        if self.simulator is not None:
            self._start_simulator()
            device = self.connect_device(self.simulator.device_factory)
            self.log(f"Device: {device.name} (key={device.key})")


    @abstractmethod
    def run(self) -> None:
        """Execute the test-specific logic."""
        pass

    def connect_device(
        self, device_factory: Callable[[int], SynnaxDevice]
    ) -> sy.Device:
        """
        Get or create a hardware device using a factory from KnownDevices.

        This is a public method that can be called for any device, whether or not
        a simulator is running. This enables tests to connect to multiple devices
        or to hardware devices without simulators.

        Args:
            device_factory: A factory function from KnownDevices (e.g., KnownDevices.modbus_sim)

        Returns:
            The created or retrieved Synnax device

        Example:
            device = self.connect_device(KnownDevices.modbus_sim)
        """
        client = self.client
        rack = client.hardware.racks.retrieve(name="Node 1 Embedded Driver")

        # Create device instance to get its name
        device_instance = device_factory(rack.key)
        device_name = device_instance.name

        try:
            device = client.hardware.devices.retrieve(name=device_name)
            self.log(f"Found existing device: {device.name}")
        except:
            self.log(f"Creating new device: {device_name}")
            device = client.hardware.devices.create(device_instance)
            self.log(f"Created device: {device.name} (key={device.key})")

        return device

    def assert_sample_count(
        self, task: sy.Task, sample_rate: sy.Rate, time_range: sy.TimeRange
    ) -> None:
        """Assert that the task has the expected number of samples.

        Args:
            task: The task to assert the sample count of
            sample_rate: The sample rate of the task
            time_range: The time range to read samples from
        """

        # Calculate duration from time range
        duration_seconds = time_range.end.span(time_range.start).seconds

        # Allow 10% tolerance for CI environments with timing variance
        expected_samples = int(sample_rate * duration_seconds)
        min_samples = int(expected_samples * 0.90)
        max_samples = int(expected_samples * 1.1)

        for channel_config in task.config.channels:
            ch = self.client.channels.retrieve(channel_config.channel)
            num_samples = len(ch.read(time_range))

            if num_samples < min_samples or num_samples > max_samples:
                self.fail(
                    f"Channel '{ch.name}' has {num_samples} samples, "
                    f"expected {expected_samples} ±10% ({min_samples}-{max_samples})"
                )
            else:
                self.log(
                    f"✓ Channel '{ch.name}': {num_samples} samples "
                    f"(expected {expected_samples} ±10%)"
                )

    def assert_task_exists(self, task_key: str) -> None:
        """Assert that a task exists in Synnax.

        Args:
            task_key: The key of the task to check

        Raises:
            Fails the test if the task does not exist
        """
        try:
            task = self.client.hardware.tasks.retrieve(task_key)
            self.log(f"✓ Task {task_key} exists (name: {task.name})")
        except Exception as e:
            self.fail(f"Task {task_key} does not exist: {e}")

    def assert_task_deleted(self, task_key: str) -> None:
        """Assert that a task has been deleted from Synnax.

        Args:
            task_key: The key of the task that should be deleted

        Raises:
            Fails the test if the task still exists
        """
        try:
            self.client.hardware.tasks.retrieve(task_key)
            self.fail(f"Task {task_key} still exists after deletion")
        except:
            # Expected: task should not be found
            self.log(f"✓ Task {task_key} successfully deleted")

    def assert_device_deleted(self, device: sy.Device) -> None:
        """Assert that a device has been deleted from Synnax.

        Args:
            device_name: The name of the device that should be deleted

        Raises:
            Fails the test if the device still exists
        """
        try:
            self.client.hardware.devices.retrieve(name=device.name)
            self.fail(f"Device '{device.name}' still exists after deletion")
        except:
            # Expected: device should not be found
            self.log(f"✓ Device '{device.name}' successfully deleted")

    def assert_channel_names(
        self, task: sy.Task, expected_names: list[str]
    ) -> None:
        """Assert that the task's channels match the expected channel names.

        Args:
            task: The task to check channel names for
            expected_names: List of expected channel names in any order

        Raises:
            Fails the test if channel names don't match
        """
        # Retrieve all channel names from the task
        actual_names = []
        for channel_config in task.config.channels:
            ch = self.client.channels.retrieve(channel_config.channel)
            actual_names.append(ch.name)

        # Sort both lists for comparison (order doesn't matter)
        expected_sorted = sorted(expected_names)
        actual_sorted = sorted(actual_names)

        if actual_sorted != expected_sorted:
            self.fail(
                f"Channel names mismatch. Expected: {expected_sorted}, "
                f"Actual: {actual_sorted}"
            )
        else:
            self.log(f"✓ Channel names match: {actual_sorted}")

    def _start_simulator(self) -> None:
        """Start the simulator server."""
        server_script = self.repo_root / self.simulator.server_script

        if not server_script.exists():
            raise FileNotFoundError(f"Server script not found: {server_script}")

        # Launch the simulator server
        self.server_process = subprocess.Popen(
            [sys.executable, str(server_script)],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.log(f"Server started with PID: {self.server_process.pid}")

        # Register cleanup handler for process exit scenarios
        atexit.register(self._cleanup_simulator)

        # Wait for server to start
        sy.sleep(self.simulator.startup_delay_seconds)

        if self.server_process.poll() is not None:
            # Server failed - capture output for debugging
            stdout, stderr = self.server_process.communicate()
            error_msg = stderr.decode() if stderr else "No error output"
            raise RuntimeError(
                f"Server failed to start (exit code: {self.server_process.returncode})\n{error_msg}"
            )

    def _cleanup_simulator(self, log: bool = False) -> None:
        """Terminate simulator server process (internal use only)."""

        # Check if server process exists and hasn't been cleaned up already
        if self.server_process is None:
            return

        if self.server_process.poll() is not None:
            self.log("Server already terminated")
            self.server_process = None
            return

        self.log("Terminating server...")
        try:
            self.server_process.terminate()
            try:
                self.server_process.wait(timeout=5 if log else 3)
                self.log("Server terminated successfully")

            except subprocess.TimeoutExpired:
                self.log("Server did not terminate gracefully, killing...")
                self.server_process.kill()
                self.server_process.wait(timeout=2 if log else 1)
                self.log("Server killed")

        except Exception as e:
            raise RuntimeError(f"Error during server cleanup: {e}")

        finally:
            self.server_process = None

    def teardown(self) -> None:
        """Terminate the simulator server if one was started."""
        if self.server_process is not None:
            self._cleanup_simulator()
