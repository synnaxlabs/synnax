#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Base classes for driver task lifecycle tests.

TaskCase: Shared base with setup, cleanup, and generic assertions.
ReadTaskCase: Read-specific lifecycle (sample counting, data saving, rate changes).
WriteTaskCase: Write-specific lifecycle (command sending).
"""

import os
import sys
from abc import abstractmethod

import synnax as sy

from framework.test_case import TestCase

# ── Channel creation helpers ─────────────────────────────────────


def create_index(client: sy.Synnax, name: str) -> sy.Channel:
    """Create a timestamp index channel."""
    return client.channels.create(
        name=name,
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )


def create_channel(
    client: sy.Synnax,
    *,
    name: str,
    data_type: sy.DataType,
    index: int,
) -> int:
    """Create a data channel and return its key."""
    return int(
        client.channels.create(
            name=name,
            data_type=data_type,
            index=index,
            retrieve_if_name_exists=True,
        ).key
    )


# ── Helpers (module-level for use by any test case) ──────────────


def send_and_verify_commands(
    client: sy.Synnax,
    *,
    cmd_keys: list[int],
    writer_name: str,
    task_name: str = "",
    task_key: int = 0,
    max_attempts: int = 3,
    timeout_per_round: sy.TimeSpan = 10 * sy.TimeSpan.SECOND,
    command_values: list[list[float]] | None = None,
) -> None:
    """Open a streamer/writer pair, send two rounds of commands, and verify.

    Retries the entire operation up to ``max_attempts`` times to handle relay
    latency and transient errors under heavy server load.

    Automatically detects whether command channels are indexed (non-virtual)
    and includes timestamp writes for their index channels.

    Args:
        command_values: Two lists of values to send in each round, one value per
            command key. If None, defaults to [42+i, ...] and [100+i, ...].
        task_key: If non-zero, also streams ``sy_status_set`` and fails if the
            driver emits any warning or error status for this task.
    """
    if command_values is None:
        command_values = [
            [42.0 + i for i in range(len(cmd_keys))],
            [100.0 + i for i in range(len(cmd_keys))],
        ]

    channels = client.channels.retrieve(cmd_keys)
    index_keys = list({ch.index for ch in channels if ch.index != 0})
    all_writer_keys = cmd_keys + index_keys

    prefix = f"{task_name}: " if task_name else ""
    last_err: Exception | None = None
    for attempt in range(max_attempts):
        verified = False
        try:
            with client.open_streamer(cmd_keys) as streamer:
                writer = client.open_writer(
                    start=sy.TimeStamp.now(),
                    channels=all_writer_keys,
                    name=writer_name,
                    enable_auto_commit=True,
                )
                try:
                    expected = {
                        key: float(v) for key, v in zip(cmd_keys, command_values[0])
                    }
                    writer.write(
                        {**expected, **{k: sy.TimeStamp.now() for k in index_keys}}
                    )
                    assert_streamed_values(
                        client,
                        streamer,
                        expected,
                        timeout=timeout_per_round,
                        task_name=task_name,
                    )

                    expected = {
                        key: float(v) for key, v in zip(cmd_keys, command_values[1])
                    }
                    writer.write(
                        {**expected, **{k: sy.TimeStamp.now() for k in index_keys}}
                    )
                    assert_streamed_values(
                        client,
                        streamer,
                        expected,
                        timeout=timeout_per_round,
                        task_name=task_name,
                    )

                    verified = True
                except Exception as e:
                    last_err = e
                finally:
                    try:
                        writer.close()
                    except Exception:
                        pass
        except Exception as e:
            if not verified:
                last_err = e
        if verified:
            if task_key:
                _assert_no_task_errors(client, task_key, task_name=task_name)
            return
        if attempt < max_attempts - 1:
            print(
                f">>> Retrying command send and verify... ({attempt + 1}/{max_attempts})"
            )
            sy.sleep(2)
    raise AssertionError(
        f"{prefix}Failed to send and verify commands after "
        f"{max_attempts} attempts: {last_err}"
    )


def _assert_no_task_errors(
    client: sy.Synnax,
    task_key: int,
    *,
    task_name: str = "",
    drain_timeout: sy.TimeSpan = 2 * sy.TimeSpan.SECOND,
) -> None:
    """Stream task status briefly and fail if warnings/errors were emitted."""
    from synnax.task.payload import Status

    prefix = f"{task_name}: " if task_name else ""
    with client.open_streamer(["sy_status_set"]) as streamer:
        timer = sy.Timer()
        while timer.elapsed() < drain_timeout:
            frame = streamer.read(timeout=drain_timeout)
            if frame is None:
                break
            if "sy_status_set" not in frame:
                continue
            for raw in frame["sy_status_set"]:
                status = Status.model_validate(raw)
                if status.details is None or status.details.task != task_key:
                    continue
                if status.variant in ("warning", "error"):
                    raise AssertionError(
                        f"{prefix}Driver reported {status.variant}: "
                        f"{status.message}"
                    )


def assert_streamed_values(
    client: sy.Synnax,
    streamer: sy.Streamer,
    expected: dict[int, float],
    timeout: sy.TimeSpan = 30 * sy.TimeSpan.SECOND,
    task_name: str = "",
) -> None:
    """Read from streamer until all expected channel values are received."""
    prefix = f"{task_name}: " if task_name else ""
    received: dict[int, float] = {}
    timer = sy.Timer()

    while len(received) < len(expected):
        if timer.elapsed() > timeout:
            missing = set(expected.keys()) - set(received.keys())
            raise AssertionError(
                f"{prefix}Timeout waiting for command values. "
                f"Missing keys: {missing}"
            )
        frame = streamer.read(timeout=timeout)
        if frame is None:
            continue
        for key in expected:
            if key in frame and len(frame[key]) > 0:
                received[key] = float(frame[key][-1])

    for key, exp_val in expected.items():
        if received[key] != exp_val:
            ch = client.channels.retrieve(key)
            raise AssertionError(
                f"{prefix}Channel '{ch.name}': "
                f"expected {exp_val}, got {received[key]}"
            )


def assert_sample_counts_in_range(
    client: sy.Synnax,
    *,
    channel_keys: list[int],
    time_range: sy.TimeRange,
    expected_samples: int,
    strict: bool = True,
    task_name: str = "",
) -> list[int]:
    """Assert sample counts for channels fall within expected range.

    Returns the list of per-channel sample counts.
    """
    prefix = f"{task_name}: " if task_name else ""
    min_samples = int(expected_samples * 0.50) if strict else 1
    max_samples = int(expected_samples * 1.5) if strict else sys.maxsize

    sample_counts = []
    for key in channel_keys:
        ch = client.channels.retrieve(key)
        num_samples = len(ch.read(time_range))
        sample_counts.append(num_samples)

        if num_samples < min_samples or num_samples > max_samples:
            if strict:
                raise AssertionError(
                    f"{prefix}Channel '{ch.name}' has {num_samples} samples, "
                    f"expected {expected_samples} ±50% "
                    f"({min_samples}-{max_samples})"
                )
            else:
                raise AssertionError(
                    f"{prefix}Channel '{ch.name}' has {num_samples} samples, "
                    f"expected at least {min_samples} sample(s)"
                )

    if len(set(sample_counts)) > 1:
        raise AssertionError(
            f"{prefix}Channels have different sample counts: {sample_counts}"
        )
    return sample_counts


class TaskCase(TestCase):
    """Base class for driver task lifecycle tests.

    Provides setup/cleanup, channel key extraction, and generic assertions.
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
        """Factory method to create protocol-specific task."""
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

    def teardown(self) -> None:
        """Delete the task created during setup, then delegate to parent."""
        if self.tsk is not None:
            try:
                self.client.tasks.delete(self.tsk.key)
                self.log(f"Task '{self.task_name}' deleted")
            except sy.NotFoundError:
                self.log(f"Task '{self.task_name}' already deleted")
        super().teardown()

    def _channel_keys(self, task: sy.Task) -> list[int]:
        """Extract channel keys from task config."""
        return [ch.channel for ch in task.config.channels]

    def test_task_exists(self) -> None:
        """Verify the task exists and has the expected channels."""
        assert self.tsk is not None
        self.log("Testing: Verify task exists")
        self.assert_task_exists(task_key=self.tsk.key)

        channel_keys = self._channel_keys(self.tsk)
        channels = self.client.channels.retrieve(channel_keys)
        expected_names = [ch.name for ch in channels]
        self.assert_channel_names(task=self.tsk, expected_names=expected_names)

    def assert_channel_names(
        self, *, task: sy.Task, expected_names: list[str]
    ) -> list[str]:
        """Assert that the task's channels match the expected channel names."""
        actual_names = []
        for key in self._channel_keys(task):
            ch = self.client.channels.retrieve(key)
            actual_names.append(ch.name)

        expected_sorted = sorted(expected_names)
        actual_sorted = sorted(actual_names)

        if actual_sorted != expected_sorted:
            raise AssertionError(
                f"Channel names mismatch. Expected: {expected_sorted}, "
                f"Actual: {actual_sorted}"
            )
        return actual_names

    def assert_task_exists(self, *, task_key: int) -> sy.Task:
        """Assert that a task exists in Synnax."""
        try:
            task = self.client.tasks.retrieve(task_key)
            if task is None:
                raise AssertionError(f"Task {task_key} does not exist (None)")
        except sy.NotFoundError:
            raise AssertionError(f"Task {task_key} does not exist (NotFoundError)")
        except Exception as e:
            raise AssertionError(f"Task {task_key} does not exist (Exception): {e}")
        return task

    def assert_task_deleted(self, *, task_key: str) -> None:
        """Assert that a task has been deleted from Synnax."""
        try:
            self.client.tasks.retrieve(task_key)
            raise AssertionError(f"Task {task_key} still exists after deletion")
        except sy.NotFoundError:
            return  # Win condition
        except Exception as e:
            raise AssertionError(f"Unexpected error asserting task deletion: {e}")

    def assert_device_exists(self, *, device_key: str) -> sy.Device:
        """Assert that a device exists in Synnax."""
        try:
            device = self.client.devices.retrieve(key=device_key)
            if device is None:
                raise AssertionError(f"Device {device_key} does not exist (None)")
        except sy.NotFoundError:
            raise AssertionError(f"Device {device_key} does not exist (NotFoundError)")
        except Exception as e:
            raise AssertionError(f"Device {device_key} does not exist (Exception): {e}")
        return device

    def assert_device_deleted(self, *, device_key: str) -> None:
        """Assert that a device has been deleted from Synnax."""
        try:
            device = self.client.devices.retrieve(key=device_key)
            raise AssertionError(f"Device '{device.name}' still exists after deletion")
        except sy.NotFoundError:
            return
        except Exception as e:
            raise AssertionError(f"Unexpected error asserting device deletion: {e}")


class ReadTaskCase(TaskCase):
    """Base for read task lifecycle tests.

    Adds sample counting, data saving toggle, rate reconfiguration,
    and streamer-based channel deletion survival testing.
    """

    def run(self) -> None:
        """Execute the standard read task lifecycle test."""
        if self.tsk is None:
            self.fail("Task not configured. Subclass must set self.tsk in setup()")
            return

        self.test_task_exists()
        self.test_start_and_stop()
        sy.sleep(1)
        self.test_disable_data_saving()
        sy.sleep(1)
        self.test_enable_data_saving()
        sy.sleep(1)
        self.test_reconfigure_rate()
        sy.sleep(1)
        self.test_survives_channel_deletion()

    def test_start_and_stop(self) -> None:
        """Start the task, collect samples, and stop it."""
        self.log("Testing: Start and stop")
        self.assert_sample_count(task=self.tsk, duration=self.TASK_DURATION)

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
        """Halve the sample rate with auto_start enabled.

        Enables auto_start before configuring so the task starts automatically
        after reconfiguration, removing the need for an explicit run() call.
        """
        assert self.tsk is not None
        self.log("Testing: Reconfigure task rate with auto_start")
        new_rate = int(self.SAMPLE_RATE / 2)
        self.tsk.config.sample_rate = new_rate
        self.tsk.config.auto_start = True
        self.client.tasks.configure(self.tsk)
        self.assert_sample_count(
            task=self.tsk, duration=self.TASK_DURATION, started=True
        )

    def test_survives_channel_deletion(self) -> None:
        """Attempt to delete a channel while the task is running."""
        assert self.tsk is not None
        self.log("Testing: Delete channel while running")

        channel_keys = self._channel_keys(self.tsk)
        ch = self.client.channels.retrieve(channel_keys[0])

        with self.tsk.run():
            with self.client.open_streamer(channel_keys) as streamer:
                frame = streamer.read(timeout=30)
                if frame is None:
                    raise AssertionError(
                        "Task is not streaming data — cannot test channel deletion"
                    )
            try:
                self.client.channels.delete(ch.key)
                raise AssertionError(
                    f"Channel '{ch.name}' deletion should have been rejected while task is running"
                )
            except AssertionError:
                raise
            except Exception:
                pass  # Win condition

    def assert_sample_count(
        self,
        *,
        task: sy.Task,
        duration: sy.TimeSpan = 1 * sy.TimeSpan.SECOND,
        strict: bool = True,
        started: bool = False,
    ) -> None:
        """Assert that the task collects the expected number of samples.

        Args:
            started: If True, the task is already running (e.g. via auto_start)
                and will be stopped after collection. If False, the task will be
                started and stopped via task.run().
        """
        sample_rate = task.config.sample_rate
        channel_keys = self._channel_keys(task)

        def collect() -> sy.TimeStamp:
            with self.client.open_streamer(channel_keys) as streamer:
                frame = streamer.read(timeout=30)
                if frame is None:
                    raise AssertionError("Task did not start — no data received")
            sy.sleep(1)
            start = sy.TimeStamp.now()
            sy.sleep(duration.seconds * 1.25)
            return start

        if started:
            start_time = collect()
            task.stop()
        else:
            with task.run():
                start_time = collect()

        end_time = sy.TimeStamp.now()
        expected_samples = int(sample_rate * duration.seconds)
        time_range = sy.TimeRange(start_time, end_time)
        assert_sample_counts_in_range(
            self.client,
            channel_keys=channel_keys,
            time_range=time_range,
            expected_samples=expected_samples,
            strict=strict,
        )

    def assert_no_samples_persisted(
        self,
        *,
        task: sy.Task,
        duration: sy.TimeSpan = 1 * sy.TimeSpan.SECOND,
    ) -> None:
        """Assert that no samples are persisted while the task is running."""
        channel_keys = self._channel_keys(task)

        with task.run():
            with self.client.open_streamer(channel_keys) as streamer:
                frame = streamer.read(timeout=30)
                if frame is None:
                    raise AssertionError(
                        "Task is not streaming data with data_saving disabled"
                    )
            start_time = sy.TimeStamp.now()
            sy.sleep(duration.seconds * 1.25)

        end_time = sy.TimeStamp.now()
        time_range = sy.TimeRange(start_time, end_time)

        for key in channel_keys:
            ch = self.client.channels.retrieve(key)
            num_samples = len(ch.read(time_range))
            if num_samples > 0:
                raise AssertionError(
                    f"Channel '{ch.name}' has {num_samples} persisted samples "
                    f"with data_saving disabled, expected 0"
                )


class WriteTaskCase(TaskCase):
    """Base for write task lifecycle tests.

    Adds command sending and task reconfiguration testing.

    Subclasses can override ``command_values`` to provide two rounds of
    values appropriate for the channel type (e.g. [0, 1] for digital).
    """

    command_values: list[list[float]] | None = None

    def run(self) -> None:
        """Execute the standard write task lifecycle test."""
        if self.tsk is None:
            self.fail("Task not configured. Subclass must set self.tsk in setup()")
            return

        self.test_task_exists()
        self.test_send_commands()
        self.test_reconfigure_name()

    def test_send_commands(self) -> None:
        """Start the write task, send commands, and verify they arrive on the stream."""
        assert self.tsk is not None
        self.log("Testing: Send commands")
        with self.tsk.run():
            send_and_verify_commands(
                self.client,
                cmd_keys=self._channel_keys(self.tsk),
                writer_name=f"{self.task_name}_test_writer",
                task_name=self.tsk.name,
                task_key=self.tsk.key,
                command_values=self.command_values,
            )

    def test_reconfigure_name(self) -> None:
        """Rename the task, verify it persists, then send commands again."""
        assert self.tsk is not None
        self.log("Testing: Reconfigure task name")

        original_name = self.tsk.name
        new_name = f"{original_name} (Renamed)"
        self.tsk._internal.name = new_name
        self.client.tasks.configure(self.tsk)

        retrieved = self.client.tasks.retrieve(self.tsk.key)
        if retrieved.name != new_name:
            raise AssertionError(
                f"Task name mismatch after rename. "
                f"Expected: {new_name}, Actual: {retrieved.name}"
            )

        with self.tsk.run():
            send_and_verify_commands(
                self.client,
                cmd_keys=self._channel_keys(self.tsk),
                writer_name=f"{new_name}_test_writer",
                task_name=self.tsk.name,
                task_key=self.tsk.key,
                command_values=self.command_values,
            )
