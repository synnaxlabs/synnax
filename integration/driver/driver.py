#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Driver test utilities.

Provides stateless helper functions for driver integration tests including
channel creation, device management, and assertion utilities.
"""

import sys
from typing import Callable, TypedDict

import synnax as sy
from synnax.hardware.device import Device as SynnaxDevice


class ChannelConfig(TypedDict, total=False):
    """Channel configuration with protocol-specific fields."""

    # Common fields (required)
    name: str
    data_type: sy.DataType

    # Modbus-specific fields
    address: int
    modbus_data_type: str
    modbus_channel_type: str

    # OPC UA-specific fields
    node_id: str
    opcua_data_type: str


class Driver:
    """
    Driver test helper class.

    Provides stateless utilities for driver integration tests including
    channel creation and device management.
    """

    @staticmethod
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

    @staticmethod
    def assert_channel_names(
        client: sy.Synnax, task: sy.Task, expected_names: list[str]
    ) -> list[str]:
        """Assert that the task's channels match the expected channel names.

        Args:
            client: Synnax client instance
            task: The task to check channel names for
            expected_names: List of expected channel names in any order

        Raises:
            AssertionError: If channel names don't match

        Returns:
            List of channel names in task
        """
        # Retrieve all channel names from the task
        actual_names = []
        for channel_config in task.config.channels:
            ch = client.channels.retrieve(channel_config.channel)
            actual_names.append(ch.name)

        # Sort both lists for comparison (order doesn't matter)
        expected_sorted = sorted(expected_names)
        actual_sorted = sorted(actual_names)

        if actual_sorted != expected_sorted:
            raise AssertionError(
                f"Channel names mismatch. Expected: {expected_sorted}, "
                f"Actual: {actual_sorted}"
            )
        return actual_names

    @staticmethod
    def assert_device_deleted(client: sy.Synnax, device_key: str) -> None:
        """Assert that a device has been deleted from Synnax.

        Args:
            client: Synnax client instance
            device: The device that should be deleted

        Raises:
            AssertionError: If the device still exists
        """
        try:
            device = client.hardware.devices.retrieve(key=device_key)
            raise AssertionError(f"Device '{device.name}' still exists after deletion")
        except sy.NotFoundError:
            return
        except Exception as e:
            raise AssertionError(
                f"Unexpected error asserting device deletion '{device.name}': {e}"
            )

    @staticmethod
    def assert_device_exists(client: sy.Synnax, device_key: str) -> sy.Device:
        """
        Assert that a device exists in Synnax.

        Args:
            client: Synnax client instance
            device_key: The key of the device to check

        Raises:
            AssertionError: If the device does not exist

        Returns:
            The retrieved device if it exists
        """
        try:
            device = client.hardware.devices.retrieve(key=device_key)
            if device is None:
                raise AssertionError(f"Device {device_key} does not exist (None)")
        except sy.NotFoundError:
            raise AssertionError(f"Device {device_key} does not exist (NotFoundError)")
        except Exception as e:
            raise AssertionError(f"Device {device_key} does not exist (Exception): {e}")
        return device

    @staticmethod
    def assert_sample_count(
        client: sy.Synnax, task: sy.Task, duration: sy.TimeSpan = 1, strict: bool = True
    ) -> None:
        """Assert that the task has the expected number of samples.

        Args:
            client: Synnax client instance
            task: The task to assert the sample count of
            duration: Duration (s) to run the task for
            strict: Sample count within 20% tolerance if True, else no check

        Raises:
            AssertionError: If sample counts are incorrect or inconsistent
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
            ch = client.channels.retrieve(channel_config.channel)
            num_samples = len(ch.read(time_range))
            sample_counts.append(num_samples)

            if num_samples < min_samples or num_samples > max_samples:
                if strict:
                    raise AssertionError(
                        f"Channel '{ch.name}' has {num_samples} samples, "
                        f"expected {expected_samples} ±20% ({min_samples}-{max_samples})"
                    )
                else:
                    raise AssertionError(
                        f"Channel '{ch.name}' has {num_samples} samples, "
                        f"expected at least {min_samples} sample(s)"
                    )

        if len(set(sample_counts)) > 1:
            raise AssertionError(
                f"Channels have different sample counts: {sample_counts}"
            )

        return

    @staticmethod
    def assert_task_deleted(client: sy.Synnax, task_key: str) -> None:
        """Assert that a task has been deleted from Synnax.

        Args:
            client: Synnax client instance
            task_key: The key of the task that should be deleted

        Raises:
            AssertionError: If the task still exists
        """
        try:
            client.hardware.tasks.retrieve(task_key)
            raise AssertionError(f"Task {task_key} still exists after deletion")
        except sy.NotFoundError:
            return  ## Win condition
        except Exception as e:
            raise AssertionError(f"Unexpected error asserting task deletion: {e}")

    @staticmethod
    def assert_task_exists(client: sy.Synnax, task_key: int) -> sy.Task:
        """
        Assert that a task exists in Synnax.

        Args:
            client: Synnax client instance
            task_key: The key of the task to check

        Raises:
            AssertionError: If the task does not exist

        Returns:
            The retrieved task if it exists
        """
        try:
            task = client.hardware.tasks.retrieve(task_key)
            if task is None:
                raise AssertionError(f"Task {task_key} does not exist (None)")
        except sy.NotFoundError:
            raise AssertionError(f"Task {task_key} does not exist (NotFoundError)")
        except Exception as e:
            raise AssertionError(f"Task {task_key} does not exist (Exception): {e}")
        return task

    @staticmethod
    def connect_device(
        client: sy.Synnax,
        rack_name: str,
        device_factory: Callable[[int], SynnaxDevice],
    ) -> sy.Device:
        """
        Get or create a hardware device using a factory from KnownDevices.

        This is a public method that can be called for any device, whether or not
        a simulator is running. This enables tests to connect to multiple devices
        or to hardware devices without simulators.

        Args:
            client: Synnax client instance
            rack_name: Name of the rack to connect the device to
            device_factory: A factory function from KnownDevices (e.g., KnownDevices.modbus_sim)

        Returns:
            The created or retrieved Synnax device

        Example:
            device = Driver.connect_device(client, "Node 1 Embedded Driver", KnownDevices.modbus_sim)
        """
        rack = client.hardware.racks.retrieve(name=rack_name)

        # Create device instance to get its name
        device_instance = device_factory(rack.key)
        device_name = device_instance.name

        try:
            device = client.hardware.devices.retrieve(name=device_name)
        except sy.NotFoundError:
            device = client.hardware.devices.create(device_instance)
        except Exception as e:
            raise AssertionError(f"Unexpected error creating device: {e}")

        return device
