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
from typing import Callable, TypedDict

import synnax as sy
from synnax.hardware.device import Device as SynnaxDevice

from framework.test_case import TestCase
from tests.driver.devices import SimulatorConfig


class ChannelConfig(TypedDict, total=False):
    """Channel configuration with protocol-specific fields."""

    # Common fields (required)
    name: str
    data_type: sy.DataType

    # Modbus-specific fields
    address: int
    modbus_data_type: str

    # OPC UA-specific fields
    node_id: str
    opcua_data_type: str


def create_channels(
    client: sy.Synnax,
    device_name: str,
    task_key: str,
    channel_configs: list[ChannelConfig],
) -> tuple[sy.Device, list[sy.Channel], list[str]]:
    """
    Create Synnax channels for a task.

    Args:
        client: Synnax client instance
        device_name: Name of the hardware device
        task_key: Task identifier (used for index channel naming)
        channel_configs: List of channel configurations

    Returns:
        Tuple of (device, created channels, channel names)
    """
    # Retrieve the device
    device = client.hardware.devices.retrieve(name=device_name)

    # Auto-generate index channel name from task key
    index_channel_name = f"{task_key}_index"
    index_ch = client.channels.create(
        name=index_channel_name,
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
        retrieve_if_name_exists=True,
    )

    # Create data channels from config
    channels = []
    channel_names = []
    for ch_config in channel_configs:
        ch = client.channels.create(
            name=ch_config["name"],
            index=index_ch.key,
            data_type=ch_config["data_type"],
            retrieve_if_name_exists=True,
        )
        channels.append(ch)
        channel_names.append(ch_config["name"])

    return device, channels, channel_names


class Task(TestCase):
    """
    Base class for driver task lifecycle tests.

    Tests standard lifecycle: create, start/stop, reconfigure, delete task and device.

    Subclasses must:
    - Set TASK_NAME, TASK_KEY, and CHANNELS as class attributes
    - Implement create_task() factory method
    - Optionally override setup() for custom configuration (e.g., matrix parameters)
    - Optionally override run() for custom test logic
    """

    # Task configuration (must be set by subclasses)
    RACK_NAME: str = "Node 1 Embedded Driver"
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
            self.connect_device(self.simulator.device_factory)

        # Create channels
        assert self.simulator is not None
        device, channels, self.channel_names = create_channels(
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
        self.assert_task_exists(tsk.key)
        self.assert_channel_names(tsk, self.channel_names)

        self.log("Test 1 - Start and Stop")
        self.assert_sample_count(
            tsk, duration=self.TEST_DURATION
        )

        # SY-3310: OPC Read Array - rapid restart race condition
        sy.sleep(0.2)

        self.log("Test 2 - Reconfigure Task")
        new_rate = int(self.SAMPLE_RATE * 2)
        tsk.config.sample_rate = new_rate
        client.hardware.tasks.configure(tsk)
        self.assert_sample_count(
            tsk, duration=self.TEST_DURATION
        )

        self.log("Test 3 - Delete Task")
        client.hardware.tasks.delete(tsk.key)
        self.assert_task_deleted(tsk.key)

        self.log("Test 4 - Delete Device")
        # Get device from task config (already embedded in task object)
        device = client.hardware.devices.retrieve(key=tsk.config.device)
        client.hardware.devices.delete([device.key])
        self.assert_device_deleted(device)

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
        rack = client.hardware.racks.retrieve(name=self.RACK_NAME)

        # Create device instance to get its name
        device_instance = device_factory(rack.key)
        device_name = device_instance.name

        try:
            device = client.hardware.devices.retrieve(name=device_name)
            self.log(f"Found existing device: {device.name}")
        except sy.NotFoundError:
            device = client.hardware.devices.create(device_instance)
            self.log(f"Created device: {device.name}")
        except Exception as e:
            self.fail(f"Unexpected error creating device: {e}")

        return device

    def assert_sample_count(
        self, 
        task: sy.Task, 
        duration: sy.TimeSpan = 1,
        strict: bool = True
    ) -> None:
        """Assert that the task has the expected number of samples.

        Args:
            task: The task to assert the sample count of
            sample_rate: The sample rate of the task
            time_range: The time range to read samples from
            strict: Sample count within 20% tolerance if True, else no check
        """
        start_time = sy.TimeStamp.now()
        with task.run():
            sy.sleep(duration)
        end_time = sy.TimeStamp.now()

        sample_rate = task.config.sample_rate
        time_range = sy.TimeRange(start_time, end_time)
        duration_seconds = time_range.end.span(time_range.start).seconds

        # Allow 20% tolerance for CI environments with timing variance
        expected_samples = int(sample_rate * duration_seconds)
        min_samples = int(expected_samples * 0.8) if strict else 1
        max_samples = int(expected_samples * 1.2) if strict else sys.maxsize

        sample_counts = []
        for channel_config in task.config.channels:
            ch = self.client.channels.retrieve(channel_config.channel)
            num_samples = len(ch.read(time_range))
            sample_counts.append(num_samples)

            if num_samples < min_samples or num_samples > max_samples:
                self.fail(
                    f"Channel '{ch.name}' has {num_samples} samples, "
                    f"expected {expected_samples} ±10% ({min_samples}-{max_samples})"
                )

        if len(set(sample_counts)) > 1:
            self.fail(f"Channels have different sample counts: {sample_counts}")

    def assert_task_exists(self, task_key: str) -> None:
        """Assert that a task exists in Synnax.

        Args:
            task_key: The key of the task to check

        Raises:
            Fails the test if the task does not exist
        """
        try:
            task = self.client.hardware.tasks.retrieve(task_key)
            if task is None:
                self.fail(f"Task does not exist (None)")
        except sy.NotFoundError:
            self.fail(f"Task does not exist (NotFoundError)")
        except Exception as e:
            self.fail(f"Task does not exist (Exception): {e}")

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
        except sy.NotFoundError:
            # This is the expected behavior
            pass
        except Exception as e:
            self.fail(f"Unexpected error asserting task deletion: {e}")

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
        except sy.NotFoundError:
            # Win condition
            pass
        except Exception as e:
            self.fail(
                f"Unexpected error asserting device deletion '{device.name}': {e}"
            )

    def assert_channel_names(self, task: sy.Task, expected_names: list[str]) -> None:
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
        self.log(f"Server started with PID: {self.simulator_process.pid}")

        # Register cleanup handler for process exit scenarios
        atexit.register(self._cleanup_simulator)

        # Wait for server to start
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
            raise RuntimeError(f"Error during server cleanup: {e}")

        finally:
            self.simulator_process = None

    def teardown(self) -> None:
        """Terminate the simulator server if one was started."""
        if self.simulator_process is not None:
            self._cleanup_simulator()
