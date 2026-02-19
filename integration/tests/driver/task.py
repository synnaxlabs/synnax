#  Copyright 2026 Synnax Labs, Inc.
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
import sys
from abc import abstractmethod

import synnax as sy

from framework.test_case import TestCase


class TaskCase(TestCase):
    """Base class for driver task lifecycle tests.

    This base class does NOT include simulator logic - that's in SimulatorCase.
    Subclasses must set task_name and device_name as class attributes.
    """

    task_name: str
    device_name: str
    tsk: sy.Task | None = None
    SAMPLE_RATE: sy.Rate = 50 * sy.Rate.HZ
    STREAM_RATE: sy.Rate = 10 * sy.Rate.HZ
    TASK_DURATION: sy.TimeSpan = 1 * sy.TimeSpan.SECOND
    RACK_NAME: str = os.environ.get("SYNNAX_DRIVER_RACK", "Node 1 Embedded Driver")

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
        device = self.client.devices.retrieve(name=self.device_name)

        # Create task using child implementation
        self.tsk = self.create(
            device=device,
            task_name=self.task_name,
            sample_rate=self.SAMPLE_RATE,
            stream_rate=self.STREAM_RATE,
        )

        # Configure task in Synnax
        try:
            self.client.tasks.configure(self.tsk)
            self.log(f"Task '{self.task_name}' configured")
        except Exception as e:
            self.fail(f"Task configuration failed: {e}")
            return

    def run(self) -> None:
        """Execute the standard task lifecycle test."""
        if self.tsk is None:
            self.fail("Task not configured. Subclass must set self.tsk in setup()")
            return

        self.test_task_exists()
        self.test_start_and_stop()
        self.test_disable_data_saving()
        self.test_enable_data_saving()
        self.test_reconfigure_rate()
        self.test_survives_channel_deletion()

    def test_task_exists(self) -> None:
        """Verify the task exists and has the expected channels."""
        assert self.tsk is not None
        self.log("Testing: Verify task exists")
        self.assert_task_exists(task_key=self.tsk.key)

        channel_keys = [ch.channel for ch in self.tsk.config.channels]
        channels = self.client.channels.retrieve(channel_keys)
        expected_names = [ch.name for ch in channels]
        self.assert_channel_names(task=self.tsk, expected_names=expected_names)

    def test_start_and_stop(self) -> None:
        """Start the task, collect samples, and stop it."""
        self.log("Testing: Start and stop")
        self.assert_sample_count(task=self.tsk, duration=self.TASK_DURATION)
        sy.sleep(0.5)

    def test_disable_data_saving(self) -> None:
        """Disable data saving and verify no samples are persisted."""
        assert self.tsk is not None
        self.log("Testing: Disable data saving")
        self.tsk.config.data_saving = False
        self.client.tasks.configure(self.tsk)
        self.assert_no_samples_persisted(task=self.tsk, duration=self.TASK_DURATION)

    def test_enable_data_saving(self) -> None:
        """Re-enable data saving and verify samples are persisted again."""
        assert self.tsk is not None
        self.log("Testing: Enable data saving")
        self.tsk.config.data_saving = True
        self.client.tasks.configure(self.tsk)
        self.assert_sample_count(task=self.tsk, duration=self.TASK_DURATION)

    def test_reconfigure_rate(self) -> None:
        """Halve the sample rate and verify samples are still collected."""
        assert self.tsk is not None
        self.log("Testing: Reconfigure task rate")
        new_rate = int(self.SAMPLE_RATE / 2)
        self.tsk.config.sample_rate = new_rate
        self.client.tasks.configure(self.tsk)
        self.assert_sample_count(task=self.tsk, duration=self.TASK_DURATION)

    def test_survives_channel_deletion(self) -> None:
        """Attempt to delete a channel while the task is running."""
        self.log("Testing: Delete channel while running")
        self.assert_survives_channel_deletion(task=self.tsk)

    def assert_channel_names(
        self, *, task: sy.Task, expected_names: list[str]
    ) -> list[str]:
        """Assert that the task's channels match the expected channel names.

        Args:
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
            ch = self.client.channels.retrieve(channel_config.channel)
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

    def assert_device_deleted(self, *, device_key: str) -> None:
        """Assert that a device has been deleted from Synnax.

        Args:
            device_key: The key of the device that should be deleted

        Raises:
            AssertionError: If the device still exists
        """
        try:
            device = self.client.devices.retrieve(key=device_key)
            raise AssertionError(f"Device '{device.name}' still exists after deletion")
        except sy.NotFoundError:
            return
        except Exception as e:
            raise AssertionError(f"Unexpected error asserting device deletion: {e}")

    def assert_device_exists(self, *, device_key: str) -> sy.Device:
        """Assert that a device exists in Synnax.

        Args:
            device_key: The key of the device to check

        Raises:
            AssertionError: If the device does not exist

        Returns:
            The retrieved device if it exists
        """
        try:
            device = self.client.devices.retrieve(key=device_key)
            if device is None:
                raise AssertionError(f"Device {device_key} does not exist (None)")
        except sy.NotFoundError:
            raise AssertionError(f"Device {device_key} does not exist (NotFoundError)")
        except Exception as e:
            raise AssertionError(f"Device {device_key} does not exist (Exception): {e}")
        return device

    def assert_sample_count(
        self,
        *,
        task: sy.Task,
        duration: sy.TimeSpan = 1 * sy.TimeSpan.SECOND,
        strict: bool = True,
    ) -> None:
        """Assert that the task has the expected number of samples.

        Args:
            task: The task to assert the sample count of
            duration: Duration to run the task for (sy.TimeSpan)
            strict: Sample count within 20% tolerance if True, else no check

        Raises:
            AssertionError: If sample counts are incorrect or inconsistent
        """

        sample_rate = task.config.sample_rate
        channel_keys = [ch.channel for ch in task.config.channels]

        with task.run():
            # Block until first frame arrives
            with self.client.open_streamer(channel_keys) as streamer:
                streamer.read(timeout=1)
            sy.sleep(1)
            start_time = sy.TimeStamp.now()
            sy.sleep(duration.seconds * 1.25)  # Bufffer for CI

        end_time = sy.TimeStamp.now()

        # Allow 35% tolerance for CI environments with timing variance
        expected_samples = int(sample_rate * duration.seconds)
        min_samples = int(expected_samples * 0.60) if strict else 1
        max_samples = int(expected_samples * 1.4) if strict else sys.maxsize

        # Read from start_time to now (captures any buffered/flushed samples)
        time_range = sy.TimeRange(start_time, end_time)

        sample_counts = []
        for channel_config in task.config.channels:
            ch = self.client.channels.retrieve(channel_config.channel)
            num_samples = len(ch.read(time_range))
            sample_counts.append(num_samples)

            if num_samples < min_samples or num_samples > max_samples:
                if strict:
                    raise AssertionError(
                        f"Channel '{ch.name}' has {num_samples} samples, "
                        f"expected {expected_samples} ±40% ({min_samples}-{max_samples})"
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

    def assert_no_samples_persisted(
        self,
        *,
        task: sy.Task,
        duration: sy.TimeSpan = 1 * sy.TimeSpan.SECOND,
    ) -> None:
        """Assert that no samples are persisted while the task is running.

        Runs the task, confirms data is still streaming, then verifies that
        no samples were written to disk.

        Args:
            task: The task to run
            duration: Duration to run the task for
        """
        channel_keys = [ch.channel for ch in task.config.channels]

        with task.run():
            with self.client.open_streamer(channel_keys) as streamer:
                frame = streamer.read(timeout=5)
                if frame is None:
                    raise AssertionError(
                        "Task is not streaming data with data_saving disabled"
                    )
            sy.sleep(1)
            start_time = sy.TimeStamp.now()
            sy.sleep(duration.seconds * 1.25)

        end_time = sy.TimeStamp.now()
        time_range = sy.TimeRange(start_time, end_time)

        for channel_config in task.config.channels:
            ch = self.client.channels.retrieve(channel_config.channel)
            num_samples = len(ch.read(time_range))
            if num_samples > 0:
                raise AssertionError(
                    f"Channel '{ch.name}' has {num_samples} persisted samples "
                    f"with data_saving disabled, expected 0"
                )

    def assert_survives_channel_deletion(self, *, task: sy.Task) -> None:
        """Assert that the server rejects channel deletion while a task is running.

        Starts the task, attempts to delete one of its data channels, and
        verifies the server rejects the deletion due to unclosed writers.

        Args:
            task: The task to test
        """
        ch = self.client.channels.retrieve(task.config.channels[0].channel)

        with task.run():
            with self.client.open_streamer(
                [c.channel for c in task.config.channels]
            ) as streamer:
                streamer.read(timeout=5)
            try:
                self.client.channels.delete(ch.key)
                raise AssertionError(
                    f"Channel '{ch.name}' deletion should have been "
                    f"rejected while task is running"
                )
            except Exception as e:
                if "unclosed writers" not in str(e):
                    raise

    def assert_task_deleted(self, *, task_key: str) -> None:
        """Assert that a task has been deleted from Synnax.

        Args:
            task_key: The key of the task that should be deleted

        Raises:
            AssertionError: If the task still exists
        """
        try:
            self.client.tasks.retrieve(task_key)
            raise AssertionError(f"Task {task_key} still exists after deletion")
        except sy.NotFoundError:
            return  # Win condition
        except Exception as e:
            raise AssertionError(f"Unexpected error asserting task deletion: {e}")

    def assert_task_exists(self, *, task_key: int) -> sy.Task:
        """Assert that a task exists in Synnax.

        Args:
            task_key: The key of the task to check

        Raises:
            AssertionError: If the task does not exist

        Returns:
            The retrieved task if it exists
        """
        try:
            task = self.client.tasks.retrieve(task_key)
            if task is None:
                raise AssertionError(f"Task {task_key} does not exist (None)")
        except sy.NotFoundError:
            raise AssertionError(f"Task {task_key} does not exist (NotFoundError)")
        except Exception as e:
            raise AssertionError(f"Task {task_key} does not exist (Exception): {e}")
        return task

    def cleanup(self) -> None:
        """Cleanup task after test."""
        if self.tsk is not None:
            try:
                self.client.tasks.delete(self.tsk.key)
                self.log(f"Task '{self.task_name}' deleted")
            except sy.NotFoundError:
                self.log(f"Task '{self.task_name}' already deleted")
            except Exception as e:
                self.log(f"Failed to delete task '{self.task_name}': {e}")
