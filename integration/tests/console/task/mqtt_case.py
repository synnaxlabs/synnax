#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.mqtt_sim import MQTTSim

import synnax as sy
from console.task.mqtt import MQTTTask
from console.task_page import TaskPage
from synnax import mqtt
from tests.console.task.task_case import ConsoleTaskCase
from tests.driver.mqtt_task import CORE_RACK_NAME
from tests.driver.task import assert_sample_counts_in_range, collect_samples

_Source = (
    mqtt.PlainReadEntry
    | mqtt.SparkplugReadEntry
    | mqtt.PlainWriteTarget
    | mqtt.SparkplugWriteTarget
)


def _sources(task: sy.Task) -> list[_Source]:
    """Return the entries of a read task or the targets of a write task.

    :raises ValueError: If the task is not an MQTT read or write task.
    """
    if task.type == mqtt.ReadTask.TYPE:
        return list(mqtt.ReadTask(task).config.entries)
    if task.type == mqtt.WriteTask.TYPE:
        return list(mqtt.WriteTask(task).config.targets)
    raise ValueError(f"{task.type!r} is not an MQTT read or write task")


class MQTTCase(ConsoleTaskCase):
    """ConsoleTaskCase against the mock MQTT plant. MQTT tasks run on the driver
    inside the Core, which has its own rack."""

    sim_classes = [MQTTSim]
    RACK_NAME = CORE_RACK_NAME
    # The rate at which the simulator publishes each topic.
    SAMPLE_RATE = 10 * sy.Rate.HZ
    STREAM_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 3 * sy.TimeSpan.SECOND

    def select_broker(self, page: MQTTTask) -> None:
        """Select the simulator's broker and wait for the form body."""
        page.select_broker(self.device_name)

    @staticmethod
    def channel_keys(task: sy.Task) -> list[int]:
        """Return every Synnax channel key an MQTT task config points at.

        :raises ValueError: If the task is not an MQTT task.
        """
        keys: list[int] = []
        if task.type == mqtt.EdgeTask.TYPE:
            for tag in mqtt.EdgeTask(task).config.tags:
                keys.extend((tag.channel, tag.command_channel))
            return [k for k in keys if k != 0]
        for source in _sources(task):
            if isinstance(source, mqtt.PlainReadEntry):
                keys.extend(f.channel for f in source.fields)
            elif isinstance(source, mqtt.PlainWriteTarget):
                keys.append(source.channel.channel)
            else:
                keys.append(source.channel)
        return [k for k in keys if k != 0]

    @staticmethod
    def data_channel_keys(task: sy.Task) -> list[int]:
        """Return the channels of the enabled entries and fields of a read task."""
        keys: list[int] = []
        for entry in mqtt.ReadTask(task).config.entries:
            if entry.disabled:
                continue
            if isinstance(entry, mqtt.SparkplugReadEntry):
                keys.append(entry.channel)
                continue
            keys.extend(f.channel for f in entry.fields if not f.disabled)
        return [k for k in keys if k != 0]

    @staticmethod
    def topics(task: sy.Task) -> set[str]:
        """Return the topics of the plain entries or targets of a task."""
        return {
            s.topic
            for s in _sources(task)
            if isinstance(s, mqtt.PlainReadEntry | mqtt.PlainWriteTarget)
        }

    @staticmethod
    def tags(task: sy.Task) -> set[str]:
        """Return the tags of the Sparkplug B entries or targets of a task."""
        return {
            s.tag
            for s in _sources(task)
            if isinstance(s, mqtt.SparkplugReadEntry | mqtt.SparkplugWriteTarget)
        }

    def assert_sample_count(self, page: TaskPage, keys: list[int]) -> None:
        """Hold the running task, stop it from the form, then check each channel
        against the publish rate of the simulator. Topics publish on their own
        indexes, so the channels are counted one at a time."""
        start = collect_samples(
            self.client, keys, duration=self.TASK_DURATION, frame_size=1
        )
        self.stop(page)
        time_range = sy.TimeRange(start, sy.TimeStamp.now())
        expected = int(float(self.SAMPLE_RATE) * self.TASK_DURATION.seconds)
        for key in keys:
            assert_sample_counts_in_range(
                self.client,
                channel_keys=[key],
                time_range=time_range,
                expected_samples=expected,
                task_name=page.page_name,
            )
