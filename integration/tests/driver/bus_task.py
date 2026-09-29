#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Base class and assertions for the CAN, serial, TCP, and UDP task cases."""

from abc import abstractmethod
from collections.abc import Callable, Iterator, Sequence
from contextlib import contextmanager
from dataclasses import dataclass
from typing import Any

import numpy as np
from examples.simulators.bus_sim import BusSim
from pydantic import ValidationError

import synnax as sy
from synnax import bus, can, library, serial, tcp, udp
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import delete_channels

STATUS_CHANNEL = "sy_status_set"
PROBLEMS = ("warning", "error")

type Task = (
    can.ReadTask
    | can.WriteTask
    | serial.ReadTask
    | serial.WriteTask
    | tcp.ReadTask
    | tcp.WriteTask
    | udp.ReadTask
    | udp.WriteTask
)


def typed_message(
    name: str,
    *,
    type_value: int,
    type_start_bit: int,
    length: int,
    fields: list[library.BinaryField],
) -> library.MessageEntry:
    """Returns a binary message identified by the value of a one-byte type field.

    :param type_value: The raw type value that selects this message.
    :param type_start_bit: The first bit of the type byte.
    :param length: The message length in bytes.
    :param fields: The fields after the type field.
    """
    kind = library.BinaryField(name="type", start_bit=type_start_bit, bit_length=8)
    return library.MessageEntry(
        name=name,
        payload=library.BinaryPayload(
            identifier=library.FieldIdentifier(field=kind.key, value=type_value),
            length=length,
            fields=[kind, *fields],
        ),
    )


@dataclass
class Bound:
    """The channels that one read message writes to."""

    index: int
    fields: dict[str, int]

    @property
    def keys(self) -> list[int]:
        return [self.index, *self.fields.values()]


class Collector:
    """Accumulates the samples a streamer receives on the channels of read messages."""

    def __init__(self, streamer: sy.Streamer, bounds: Sequence[Bound]) -> None:
        self._streamer = streamer
        self._bounds = bounds
        self._chunks: dict[int, list[np.ndarray]] = {
            k: [] for b in bounds for k in b.keys
        }

    def count(self, bound: Bound) -> int:
        """Returns the number of messages received for bound so far."""
        return sum(len(c) for c in self._chunks[bound.index])

    def wait(self, count: int, timeout: sy.TimeSpan = 15 * sy.TimeSpan.SECOND) -> None:
        """Reads until every message has at least count samples in total.

        :raises AssertionError: If a message falls short when the timeout expires.
        """
        timer = sy.Timer()
        while any(self.count(b) < count for b in self._bounds):
            if timer.elapsed() > timeout:
                counts = [self.count(b) for b in self._bounds]
                raise AssertionError(
                    f"Expected {count} samples per message within {timeout}, "
                    f"got {counts}"
                )
            self._read(timeout=1)

    def drain(self) -> None:
        """Takes in every sample the streamer has received so far."""
        while self._read(timeout=0):
            pass

    def _read(self, timeout: float) -> bool:
        frame = self._streamer.read(timeout=timeout)
        if frame is None:
            return False
        for key, chunks in self._chunks.items():
            if key in frame:
                chunks.append(frame[key].to_numpy())
        return True

    def __getitem__(self, key: int) -> np.ndarray:
        chunks = self._chunks[key]
        return np.concatenate(chunks) if chunks else np.array([])


def assert_equal(name: str, actual: np.ndarray, expected: Sequence[float]) -> None:
    """Asserts that actual holds exactly the expected values."""
    want = np.asarray(expected, dtype=np.float64)
    got = actual.astype(np.float64)
    if got.shape != want.shape:
        raise AssertionError(f"{name}: expected {len(want)} samples, got {len(got)}")
    mismatched = np.flatnonzero(got != want)
    if len(mismatched) > 0:
        i = int(mismatched[0])
        raise AssertionError(
            f"{name}: {len(mismatched)} of {len(want)} samples differ, first at "
            f"sample {i}: expected {want[i]!r}, got {got[i]!r}"
        )


def assert_increasing(name: str, values: np.ndarray, step: int | None = None) -> None:
    """Asserts that values strictly increase, by exactly step when given."""
    if len(values) < 2:
        raise AssertionError(f"{name}: expected at least 2 samples, got {len(values)}")
    diffs = np.diff(values.astype(np.int64))
    bad = np.flatnonzero(diffs <= 0 if step is None else diffs != step)
    if len(bad) > 0:
        i = int(bad[0])
        want = "increase" if step is None else f"increase by {step}"
        raise AssertionError(
            f"{name}: expected samples to {want}, got {values[i]} then "
            f"{values[i + 1]} at sample {i}"
        )


def assert_indexed(name: str, bound: Bound, samples: Collector) -> None:
    """Asserts that every field of a message has one sample per index timestamp and
    that the timestamps strictly increase."""
    times = samples[bound.index]
    assert_increasing(f"{name} index", times)
    for field, key in bound.fields.items():
        if len(samples[key]) != len(times):
            raise AssertionError(
                f"{name}.{field}: expected {len(times)} samples, one per index "
                f"timestamp, got {len(samples[key])}"
            )


class BusCase(SimulatorCase):
    """Runs one bus task against the simulator in sim_classes.

    Setup creates a library from create_entries, then the task from create_task, and
    configures the task. Teardown deletes the task, the channels made through
    create_index and create_channel, and the library.
    """

    prefix: str
    tsk: Task | None = None
    library_key: library.Key | None = None

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self._created_channels: list[int] = []
        self._messages: dict[str, library.MessageEntry] = {}

    @abstractmethod
    def create_entries(self) -> list[library.Entry]:
        """Returns the library entries the task uses."""

    @abstractmethod
    def create_task(self, device: sy.Device, library_key: library.Key) -> Task:
        """Returns the task to configure.

        :param library_key: The key of the library made from create_entries.
        """

    def unsupported(self) -> str | None:
        """Returns why this host cannot run the case, or None when it can."""
        return None

    def setup(self) -> None:
        reason = self.unsupported()
        if reason is not None:
            self.auto_pass(msg=reason)
            return
        super().setup()
        created = self.client.libraries.create(
            name=f"{type(self).__name__} Library", entries=self.create_entries()
        )
        self.library_key = created.key
        self._messages = {
            e.name: e for e in created.entries if isinstance(e, library.MessageEntry)
        }
        device = self.client.devices.retrieve(name=self.device_name)
        task = self.create_task(device, created.key)
        self.client.tasks.configure(task)
        self.tsk = task

    def teardown(self) -> None:
        try:
            if self.tsk is not None:
                with self._try_to("delete task"):
                    self.client.tasks.delete(self.tsk.key)
            with self._try_to("delete channels"):
                delete_channels(self.client, self._created_channels)
            if self.library_key is not None:
                with self._try_to("delete library"):
                    self.client.libraries.delete(self.library_key)
        finally:
            super().teardown()

    @property
    def task(self) -> Task:
        if self.tsk is None:
            raise RuntimeError("Task is not configured")
        return self.tsk

    def simulator[S: BusSim[Any]](self, cls: type[S]) -> S:
        """Returns the running simulator as cls."""
        if not isinstance(self.sim, cls):
            raise TypeError(f"Expected a running {cls.__name__}, got {self.sim!r}")
        return self.sim

    def create_index(self, name: str) -> int:
        """Creates an index channel that teardown deletes."""
        key = int(
            self.client.channels.create(
                name=name,
                data_type=sy.DataType.TIMESTAMP,
                is_index=True,
                retrieve_if_name_exists=True,
            ).key
        )
        self._created_channels.append(key)
        return key

    def create_channel(self, name: str, data_type: sy.DataType, index: int) -> int:
        """Creates a data channel that teardown deletes."""
        key = int(
            self.client.channels.create(
                name=name,
                data_type=data_type,
                index=index,
                retrieve_if_name_exists=True,
            ).key
        )
        self._created_channels.append(key)
        return key

    def bind_read(
        self,
        message: str,
        fields: list[str],
        data_types: dict[str, sy.DataType] | None = None,
    ) -> tuple[bus.ReadMessage, Bound]:
        """Creates an index for a message and a channel for each named field.

        :param data_types: The data type of each field's channel. FLOAT64 when absent.
        """
        entry = self._messages[message]
        types = data_types or {}
        index = self.create_index(f"{self.prefix}_{entry.name}_time")
        keys = {f.name: f.key for f in entry.payload.fields}
        channels = {
            name: self.create_channel(
                f"{self.prefix}_{entry.name}_{name}",
                types.get(name, sy.DataType.FLOAT64),
                index,
            )
            for name in fields
        }
        read = bus.ReadMessage(
            message=entry.key,
            index=index,
            fields=[
                bus.ReadField(field=keys[name], channel=key)
                for name, key in channels.items()
            ],
        )
        return read, Bound(index=index, fields=channels)

    def bind_write(
        self, message: str, fields: list[str]
    ) -> tuple[bus.WriteMessage, dict[str, int]]:
        """Creates a FLOAT64 command channel for each named field of a message."""
        entry = self._messages[message]
        index = self.create_index(f"{self.prefix}_{entry.name}_cmd_time")
        keys = {f.name: f.key for f in entry.payload.fields}
        channels = {
            name: self.create_channel(
                f"{self.prefix}_{entry.name}_{name}_cmd", sy.DataType.FLOAT64, index
            )
            for name in fields
        }
        write = bus.WriteMessage(
            message=entry.key,
            fields=[
                bus.WriteField(field=keys[name], channel=key)
                for name, key in channels.items()
            ],
        )
        return write, channels

    @contextmanager
    def collect(self, bounds: Sequence[Bound]) -> Iterator[Collector]:
        """Streams the channels of bounds for the duration of the block."""
        keys = [k for b in bounds for k in b.keys]
        with self.client.open_streamer(keys) as streamer:
            yield Collector(streamer, bounds)

    @contextmanager
    def statuses(self) -> Iterator[sy.Streamer]:
        """Streams task statuses for the duration of the block."""
        with self.client.open_streamer([STATUS_CHANNEL]) as streamer:
            yield streamer

    def command(self, values: dict[int, float]) -> None:
        """Writes one sample to each command channel in values."""
        channels = self.client.channels.retrieve(list(values))
        indexes = list({ch.index for ch in channels if ch.index != 0})
        with self.client.open_writer(
            start=sy.TimeStamp.now(),
            channels=[*values, *indexes],
            name=f"{type(self).__name__} commands",
        ) as writer:
            now = sy.TimeStamp.now()
            writer.write({**values, **{k: now for k in indexes}})

    def command_until(
        self,
        values: dict[int, float],
        delivered: Callable[[], None],
        attempts: int = 3,
    ) -> None:
        """Writes values to the command channels until the device receives them. A
        write task acknowledges its start before its streamer opens, so a command
        sent right after the start can be lost.

        :param delivered: Raises AssertionError when the device did not receive the
            command.
        :raises AssertionError: If every attempt fails.
        """
        for attempt in range(attempts):
            self.command(values)
            try:
                delivered()
                return
            except AssertionError:
                if attempt == attempts - 1:
                    raise
                print(f">>> Retrying command ({attempt + 1}/{attempts})")

    def _task_statuses(self, frame: sy.Frame) -> Iterator[sy.task.Status]:
        if STATUS_CHANNEL not in frame:
            return
        for raw in frame[STATUS_CHANNEL]:
            try:
                status = sy.task.Status.model_validate(raw)
            except ValidationError:
                continue
            if status.details is not None and status.details.task == self.task.key:
                yield status

    def assert_no_problems(self, streamer: sy.Streamer) -> None:
        """Asserts that the task has reported no warning or error so far."""
        problems: list[str] = []
        while (frame := streamer.read(timeout=0)) is not None:
            problems.extend(
                f"{s.variant}: {s.message}"
                for s in self._task_statuses(frame)
                if s.variant in PROBLEMS
            )
        if problems:
            raise AssertionError(f"{self.task.name} reported {problems}")

    def wait_for_status(
        self,
        streamer: sy.Streamer,
        variant: str,
        timeout: sy.TimeSpan = 10 * sy.TimeSpan.SECOND,
    ) -> str:
        """Blocks until the task reports a status of the given variant and returns its
        message.

        :raises AssertionError: If no such status arrives before the timeout.
        """
        timer = sy.Timer()
        while timer.elapsed() < timeout:
            frame = streamer.read(timeout=0.5)
            if frame is None:
                continue
            for status in self._task_statuses(frame):
                if status.variant == variant:
                    return str(status.message)
        raise AssertionError(
            f"{self.task.name} reported no {variant} status within {timeout}"
        )
