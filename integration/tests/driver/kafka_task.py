#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Kafka-specific task test cases."""

import sys
from abc import abstractmethod

from examples.kafka_sim import KafkaSim

import synnax as sy
from synnax import kafka
from tests.driver.simulator_case import CORE_RACK_NAME, SimulatorCase
from tests.driver.task import ReadTaskCase


def create_topic(
    client: sy.Synnax,
    *,
    topic: str,
    bindings: list[tuple[sy.icd.JSONField, int]],
    index: sy.icd.JSONField | None = None,
    record_key: str = "",
) -> kafka.ReadTopic:
    """Create an ICD with one JSON message, named after topic, that holds the fields of
    bindings. Return a topic that reads its records of record_key as that message.

    :param bindings: Each field of the message, with the channel it writes to.
    :param index: The field of bindings whose channel indexes the others. None stamps
        each sample with the timestamp of its record.
    :param record_key: The Kafka key of the records. Empty reads every record.
    """
    message = sy.icd.Message(
        name=topic, payload=sy.icd.JSONPayload(fields=[f for f, _ in bindings])
    )
    ic = client.icds.create(name=topic, messages=[message])
    return kafka.ReadTopic(
        name=topic,
        records=[
            kafka.ReadRecord(
                record_key=record_key,
                icd=ic.key,
                messages=[
                    sy.icd.ReadMessage(
                        message=message.key,
                        index=None if index is None else index.key,
                        bindings=[
                            sy.icd.ReadBinding(field=f.key, channel=ch)
                            for f, ch in bindings
                        ],
                    )
                ],
            )
        ],
    )


class KafkaCase(SimulatorCase):
    """Base class for cases that need the Kafka simulator."""

    sim_classes = [KafkaSim]
    rack_name = CORE_RACK_NAME

    def setup(self) -> None:
        if sys.platform == "win32":
            self.skip("The Kafka broker of the simulator runs only on POSIX hosts")
        super().setup()


class KafkaReadTaskCase(KafkaCase, ReadTaskCase[kafka.ReadTask]):
    """Base class for Kafka read task tests."""

    EVENT_DRIVEN = True
    # The rate at which the simulator produces each topic.
    SAMPLE_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 3 * sy.TimeSpan.SECOND

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[kafka.ReadTopic]:
        """Create the channels of the task and return its topics."""
        ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> kafka.ReadTask:
        return kafka.ReadTask(
            name=task_name,
            device=device.key,
            topics=self.create_channels(self.client),
        )

    def _channel_keys(self, task: kafka.ReadTask) -> list[int]:
        return [
            b.channel
            for t in task.config.topics
            for r in t.records
            for m in r.messages
            for b in m.bindings
            if b.channel != 0
        ]
