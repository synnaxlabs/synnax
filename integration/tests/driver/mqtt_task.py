#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""MQTT-specific task test cases."""

import os
from abc import abstractmethod
from collections.abc import Callable, Iterator
from contextlib import contextmanager

from examples.mqtt_sim import MQTTSim

import synnax as sy
from synnax import mqtt
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, WriteTaskCase
from x.telem import SampleValue, to_float

# MQTT tasks run on the driver inside the Core, which has its own rack.
CORE_RACK_NAME = os.environ.get("SYNNAX_CORE_RACK", "Node 1")
WRITE_TIMEOUT = 20 * sy.TimeSpan.SECOND


@contextmanager
def run_reader(
    client: sy.Synnax,
    *,
    device: str,
    name: str,
    channel: str,
    data_type: sy.DataType,
    entry: Callable[[sy.Channel], mqtt.ReadEntry],
) -> Iterator[sy.Channel]:
    """Run a read task of one entry that writes to a new virtual channel. On exit,
    delete the task and the channel.

    :param device: The key of the broker device to read from.
    :param name: The name of the read task.
    :param channel: The name of the virtual channel.
    :param data_type: The data type of the virtual channel.
    :param entry: Returns the entry of the task for the virtual channel.
    :returns: A context that yields the virtual channel.
    """
    ch = client.channels.create(
        name=channel, data_type=data_type, virtual=True, retrieve_if_name_exists=True
    )
    reader = mqtt.ReadTask(name=name, device=device, entries=[entry(ch)])
    client.tasks.configure(reader)
    try:
        with reader.run():
            yield ch
    finally:
        client.tasks.delete(reader.key)
        client.channels.delete([ch.key])


def write_until(
    client: sy.Synnax,
    *,
    channel: sy.Channel,
    value: float,
    watched: sy.Channel,
    matches: Callable[[SampleValue], bool] | None = None,
) -> SampleValue:
    """Write ``value`` to ``channel`` until a sample of ``watched`` matches, for at most
    20 seconds. A message sent before its subscriber or Sparkplug birth is ready is
    lost, so each pass writes the value again.

    :param matches: Whether a sample of ``watched`` ends the wait. By default, a sample
        that equals ``value``.
    :returns: The sample that matched.
    :raises AssertionError: If no sample matches in time.
    """
    last: SampleValue | None = None
    timer = sy.Timer()
    with client.open_streamer([watched.key]) as streamer:
        with client.open_writer(
            start=sy.TimeStamp.now(), channels=[channel.index, channel.key]
        ) as writer:
            while timer.elapsed() < WRITE_TIMEOUT:
                writer.write({channel.index: sy.TimeStamp.now(), channel.key: value})
                frame = streamer.read(timeout=1)
                if frame is None or watched.key not in frame:
                    continue
                last = frame[watched.key][-1]
                done = to_float(last) == value if matches is None else matches(last)
                if done:
                    return last
    raise AssertionError(
        f"{watched.name} did not match within {WRITE_TIMEOUT}, last sample: {last}"
    )


class MQTTReadTaskCase(SimulatorCase, ReadTaskCase[mqtt.ReadTask]):
    """Base class for MQTT read task tests."""

    sim_classes = [MQTTSim]
    RACK_NAME = CORE_RACK_NAME
    EVENT_DRIVEN = True
    # The rate at which the simulator publishes each topic.
    SAMPLE_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 3 * sy.TimeSpan.SECOND

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[mqtt.ReadEntry]:
        """Create the channels of the task and return its entries."""
        ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> mqtt.ReadTask:
        return mqtt.ReadTask(
            name=task_name,
            device=device.key,
            entries=self.create_channels(self.client),
        )

    def _channel_keys(self, task: mqtt.ReadTask) -> list[int]:
        keys: list[int] = []
        for entry in task.config.entries:
            if isinstance(entry, mqtt.SparkplugReadEntry):
                keys.append(entry.channel)
                continue
            keys.extend(f.channel for f in entry.fields if f.channel != 0)
        return keys


class MQTTWriteTaskCase(SimulatorCase, WriteTaskCase[mqtt.WriteTask]):
    """Base class for MQTT write task tests."""

    sim_classes = [MQTTSim]
    RACK_NAME = CORE_RACK_NAME

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[mqtt.WriteTarget]:
        """Create the command channels of the task and return its targets."""
        ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> mqtt.WriteTask:
        return mqtt.WriteTask(
            name=task_name,
            device=device.key,
            targets=self.create_channels(self.client),
        )

    def _channel_keys(self, task: mqtt.WriteTask) -> list[int]:
        return [
            t.channel if isinstance(t, mqtt.SparkplugWriteTarget) else t.channel.channel
            for t in task.config.targets
        ]
