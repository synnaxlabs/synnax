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

from examples.mqtt_sim import MQTTSim

import synnax as sy
from synnax import mqtt
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, WriteTaskCase, assert_sample_counts_in_range

# MQTT tasks run on the driver inside the Core, which has its own rack.
CORE_RACK_NAME = os.environ.get("SYNNAX_CORE_RACK", "Node 1")


class MQTTReadTaskCase(SimulatorCase, ReadTaskCase):
    """Base class for MQTT read task tests."""

    sim_classes = [MQTTSim]
    RACK_NAME = CORE_RACK_NAME
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

    def _channel_keys(self, task: sy.Task) -> list[int]:
        keys: list[int] = []
        for entry in task.config.entries:
            if isinstance(entry, mqtt.SparkplugReadEntry):
                keys.append(entry.channel)
                continue
            keys.extend(f.channel for f in entry.fields if f.channel != 0)
        return keys

    def run(self) -> None:
        """An MQTT read task has no rate, so the rate step of the lifecycle is left
        out."""
        if self.tsk is None:
            self.fail("Task not configured. Subclass must set self.tsk in setup()")
            return
        self.test_task_exists()
        self.test_start_and_stop()
        self.test_disable_data_saving()
        self.test_enable_data_saving()
        self.test_survives_channel_deletion()

    def assert_sample_count(
        self,
        *,
        task: sy.Task,
        duration: sy.TimeSpan = 1 * sy.TimeSpan.SECOND,
        strict: bool = True,
    ) -> None:
        """Assert the sample count against the publish rate of the simulator."""
        channel_keys = self._channel_keys(task)

        def collect() -> sy.TimeStamp:
            with self.client.open_streamer(channel_keys) as streamer:
                if streamer.read(timeout=30) is None:
                    raise AssertionError("Task did not start: no data received")
            sy.sleep(1)
            start = sy.TimeStamp.now()
            sy.sleep(duration.seconds * 1.25)
            return start

        with task.run():
            start_time = collect()

        time_range = sy.TimeRange(start_time, sy.TimeStamp.now())
        for key in channel_keys:
            assert_sample_counts_in_range(
                self.client,
                channel_keys=[key],
                time_range=time_range,
                expected_samples=int(float(self.SAMPLE_RATE) * duration.seconds),
                strict=strict,
            )


class MQTTWriteTaskCase(SimulatorCase, WriteTaskCase):
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

    def _channel_keys(self, task: sy.Task) -> list[int]:
        return [
            t.channel if isinstance(t, mqtt.SparkplugWriteTarget) else t.channel.channel
            for t in task.config.targets
        ]
