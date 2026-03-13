#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""HTTP-specific task test cases."""

from abc import abstractmethod

import synnax as sy
from examples.http import HTTPSim

from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, WriteTaskCase, assert_sample_counts_in_range


class HTTPReadTaskCase(SimulatorCase, ReadTaskCase):
    """Base class for HTTP read task tests."""

    sim_classes = [HTTPSim]
    SAMPLE_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 3 * sy.TimeSpan.SECOND

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax,
    ) -> tuple[list[sy.http.ReadEndpoint], list[int]]:
        """Create endpoints and return (endpoints, channel_keys)."""
        ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.http.ReadTask:
        """Create an HTTP read task."""
        endpoints, _ = self.create_channels(self.client)
        return sy.http.ReadTask(
            name=task_name,
            device=device.key,
            rate=float(sample_rate),
            data_saving=True,
            endpoints=endpoints,
        )

    def _channel_keys(self, task: sy.Task) -> list[int]:
        """Extract channel keys from HTTP read task config."""
        keys = []
        for ep in task.config.endpoints:
            for field in ep.fields:
                if field.channel != 0:
                    keys.append(field.channel)
        return keys

    def assert_sample_count(
        self,
        *,
        task: sy.Task,
        duration: sy.TimeSpan = 1 * sy.TimeSpan.SECOND,
        strict: bool = True,
        started: bool = False,
    ) -> None:
        """Assert sample count using task.config.rate instead of sample_rate.

        Checks each channel independently to handle multi-endpoint tasks where
        different endpoints may produce slightly different sample counts.
        """
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
        expected_samples = int(task.config.rate * duration.seconds)
        time_range = sy.TimeRange(start_time, end_time)
        for key in channel_keys:
            assert_sample_counts_in_range(
                self.client,
                channel_keys=[key],
                time_range=time_range,
                expected_samples=expected_samples,
                strict=strict,
            )

    def test_reconfigure_rate(self) -> None:
        """Halve the polling rate with auto_start enabled."""
        assert self.tsk is not None
        self.log("Testing: Reconfigure task rate with auto_start")
        new_rate = self.tsk.config.rate / 2
        self.tsk.config.rate = new_rate
        self.tsk.config.auto_start = True
        self.client.tasks.configure(self.tsk)
        self.assert_sample_count(
            task=self.tsk, duration=self.TASK_DURATION, started=True
        )


class HTTPWriteTaskCase(SimulatorCase, WriteTaskCase):
    """Base class for HTTP write task tests."""

    sim_classes = [HTTPSim]

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax,
    ) -> tuple[list[sy.http.WriteEndpoint], list[int]]:
        """Create endpoints and return (endpoints, cmd_channel_keys)."""
        ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.http.WriteTask:
        """Create an HTTP write task."""
        endpoints, _ = self.create_channels(self.client)
        return sy.http.WriteTask(
            name=task_name,
            device=device.key,
            endpoints=endpoints,
        )

    def _channel_keys(self, task: sy.Task) -> list[int]:
        """Extract command channel keys from HTTP write task config."""
        return [ep.channel.channel for ep in task.config.endpoints]
