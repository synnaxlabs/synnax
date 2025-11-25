#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Base class for driver task lifecycle tests.

This module provides the abstract TaskCase class that defines the standard
pattern for testing driver tasks.
"""

import os
from abc import abstractmethod
from typing import Any

import synnax as sy

from driver.driver import Driver
from framework.test_case import SynnaxConnection, TestCase


class TaskCase(TestCase):
    """
    Base class for driver task lifecycle tests.

    This base class does NOT include simulator logic - that's in SimulatorTaskCase.
    Use this class directly for hardware tests that don't need simulators (e.g., NI, LabJack).

    Subclasses should:
    - Implement create_channels() method to define task-specific channels
    - Implement create() method to return a configured task using those channels
    - Optionally override run() for custom test logic
    - Optionally pass task_name, task_key, and device_name to __init__ to override defaults

    Environment Variables:
    - SYNNAX_DRIVER_RACK: Override the driver rack name (default: "Node 1 Embedded Driver")
      Can be set via command line: --driver "My Custom Rack Name" or -d "My Custom Rack Name"
    """

    # SY-3254: Handle multi-device tasks
    device_name: str

    # Task instance
    tsk: sy.Task | None = None

    def __init__(
        self,
        *,
        task_name: str,
        sample_rate: sy.Rate = 50 * sy.Rate.HZ,
        stream_rate: sy.Rate = 10 * sy.Rate.HZ,
        task_duration: sy.TimeSpan = 1 * sy.TimeSpan.SECOND,
        rack_name: str = os.environ.get("SYNNAX_DRIVER_RACK", "Node 1 Embedded Driver"),
        **params: Any,
    ) -> None:
        """
        Initialize TaskCase.

        Args:
            task_name: Human-readable task name (required)
            **params: Additional test parameters (name, expect, synnax_connection, etc.)
        """

        self.task_name = task_name
        self.SAMPLE_RATE = sample_rate
        self.STREAM_RATE = stream_rate
        self.TASK_DURATION = task_duration
        self.RACK_NAME = rack_name

        super().__init__(**params)

    @abstractmethod
    def create_channels(self, *, device: sy.Device) -> list[object]:
        """
        Create protocol-specific task channels.

        Args:
            device: Synnax device to create channels for

        Returns:
            List of protocol-specific channel objects (e.g., modbus.InputRegisterChan,
            opcua.ReadChannel, ni.AIChannel, etc.)
        """
        pass

    @abstractmethod
    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.Task:
        """Factory method to create protocol-specific task.

        Args:
            device: Synnax device to configure task for
            task_name: Name for the task
            sample_rate: Sampling rate for the task
            stream_rate: Streaming rate for the task

        Returns:
            Configured protocol-specific task (e.g., modbus.ReadTask, opcua.ReadTask)
            with all channels and configuration set.
        """
        pass

    def setup(self) -> None:
        """Create and configure task."""
        # Get device
        device = self.client.hardware.devices.retrieve(name=self.device_name)

        # Create task using child implementation
        self.tsk = self.create(
            device=device,
            task_name=self.task_name,
            sample_rate=self.SAMPLE_RATE,
            stream_rate=self.STREAM_RATE,
        )

        # Configure task in Synnax
        try:
            self.client.hardware.tasks.configure(self.tsk)
            self.log(f"Task '{self.task_name}' configured")
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
        Driver.assert_task_exists(client, task_key=tsk.key)

        # Get channel names from task
        channel_keys = [ch.channel for ch in tsk.config.channels]
        channels = client.channels.retrieve(channel_keys)
        expected_names = [ch.name for ch in channels]
        Driver.assert_channel_names(client, task=tsk, expected_names=expected_names)

        self.log("Test 1 - Start and Stop")
        Driver.assert_sample_count(client, task=tsk, duration=self.TASK_DURATION)

        # SY-3310: OPC Read Array - rapid restart race condition
        sy.sleep(0.2)

        self.log("Test 2 - Reconfigure Task")
        new_rate = int(self.SAMPLE_RATE * 2)
        tsk.config.sample_rate = new_rate
        client.hardware.tasks.configure(tsk)
        Driver.assert_sample_count(client, task=tsk, duration=self.TASK_DURATION)

    def cleanup(self) -> None:
        """Cleanup task after test."""
        if self.tsk is not None:
            try:
                self.client.hardware.tasks.delete(self.tsk.key)
                self.log(f"Task '{self.task_name}' deleted")
            except sy.NotFoundError:
                self.log(f"Task '{self.task_name}' already deleted")
            except Exception as e:
                self.log(f"Failed to delete task '{self.task_name}': {e}")
