#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.task_page import TaskPage
from synnax import kafka
from tests.console.task.task_case import ConsoleTaskCase
from tests.driver import kafka_task
from tests.driver.task import assert_sample_counts_in_range, collect_samples


class KafkaCase(kafka_task.KafkaCase, ConsoleTaskCase):
    """ConsoleTaskCase against the Kafka simulator. Kafka tasks run on the Driver inside
    the Core, which has its own rack."""

    # The rate at which the simulator produces each topic.
    SAMPLE_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 3 * sy.TimeSpan.SECOND

    @staticmethod
    def channel_keys(task: sy.Task) -> list[int]:
        """Return the channels of the records of a read task or the command channels
        of the targets of a write task.

        :raises ValueError: If the task is not a Kafka read or write task.
        """
        if task.type == kafka.ReadTask.TYPE:
            keys = [
                b.channel
                for t in kafka.ReadTask(task).config.topics
                for r in t.records
                for m in r.messages
                for b in m.bindings
            ]
        elif task.type == kafka.WriteTask.TYPE:
            keys = [
                b.channel
                for t in kafka.WriteTask(task).config.targets
                for b in t.message.bindings
                if isinstance(b, sy.icd.ChannelWriteBinding)
            ]
        else:
            raise ValueError(f"{task.type!r} is not a Kafka read or write task")
        return [k for k in keys if k != 0]

    def assert_sample_count(self, page: TaskPage, *, keys: list[int]) -> None:
        """Hold the running task, stop it from the form, then check each channel against
        the produce rate of the simulator. The channels of one topic share an index and
        must hold the same count. Topics produce on their own, so their counts may
        differ."""
        start = collect_samples(
            self.client, keys, duration=self.TASK_DURATION, frame_size=1
        )
        self.stop(page)
        time_range = sy.TimeRange(start, sy.TimeStamp.now())
        expected = int(float(self.SAMPLE_RATE) * self.TASK_DURATION.seconds)
        topics: dict[int, list[int]] = {}
        for ch in self.client.channels.retrieve(keys):
            topics.setdefault(ch.index, []).append(ch.key)
        for group in topics.values():
            assert_sample_counts_in_range(
                self.client,
                channel_keys=group,
                time_range=time_range,
                expected_samples=expected,
                task_name=page.page_name,
            )
