#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from examples.mqtt_sim import MQTTSim

import synnax as sy
from console.task.mqtt import MQTTTask
from console.task_page import TaskPage
from tests.console.task.task_case import ConsoleTaskCase
from tests.driver.mqtt_task import CORE_RACK_NAME
from tests.driver.task import assert_sample_counts_in_range, collect_samples


def _key(item: dict[str, Any], *names: str) -> int:
    """Return the first of ``names`` held by a config item, in either key case."""
    for name in names:
        value = item.get(name)
        if isinstance(value, int) and value != 0:
            return value
    return 0


def _items(task: sy.Task) -> list[dict[str, Any]]:
    """Return the entries, targets, or tags of an MQTT task config."""
    config = task.config
    for group in ("entries", "targets", "tags"):
        if group in config:
            return list(config[group])
    return []


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
        """Select the simulator's broker and wait for its Browser."""
        page.select_broker(self.device_name)

    @staticmethod
    def channel_keys(task: sy.Task) -> list[int]:
        """Return every Synnax channel key an MQTT task config points at."""
        keys: list[int] = []
        for item in _items(task):
            channel = item.get("channel")
            if isinstance(channel, dict):
                channel = channel.get("channel")
            if isinstance(channel, int) and channel != 0:
                keys.append(channel)
            keys.extend(_key(f, "channel") for f in item.get("fields", []))
            command = _key(item, "command_channel", "commandChannel")
            if command != 0:
                keys.append(command)
        return [k for k in keys if k != 0]

    @staticmethod
    def data_channel_keys(task: sy.Task) -> list[int]:
        """Return the channels of the enabled entries and fields of a read task."""
        keys: list[int] = []
        for entry in _items(task):
            if entry.get("disabled"):
                continue
            if entry.get("type") == "sparkplug":
                keys.append(_key(entry, "channel"))
                continue
            keys.extend(
                _key(f, "channel") for f in entry["fields"] if not f.get("disabled")
            )
        return [k for k in keys if k != 0]

    @staticmethod
    def topics(task: sy.Task) -> set[str]:
        """Return the topics of the plain entries or targets of a task."""
        return {item["topic"] for item in _items(task) if item.get("type") == "plain"}

    @staticmethod
    def tags(task: sy.Task) -> set[str]:
        """Return the tags of the Sparkplug B entries or targets of a task."""
        return {item["tag"] for item in _items(task) if item.get("type") == "sparkplug"}

    def assert_sample_count(self, page: TaskPage, keys: list[int]) -> None:
        """Hold the running task, stop it from the form, then check each channel
        against the publish rate of the simulator. Topics publish on their own
        indexes, so the channels are counted one at a time."""
        start = collect_samples(self.client, keys, self.TASK_DURATION, 1)
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
