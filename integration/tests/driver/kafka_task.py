#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Kafka-specific task test cases."""

import json
import sys
from abc import abstractmethod
from collections.abc import Callable, Iterator
from contextlib import contextmanager
from uuid import uuid4

from confluent_kafka import Consumer, KafkaError, KafkaException, TopicPartition
from examples.kafka_sim import KafkaSim

import synnax as sy
from synnax import kafka
from tests.driver.simulator_case import CORE_RACK_NAME, SimulatorCase
from tests.driver.task import ReadTaskCase, WriteTaskCase

BROKER = f"{KafkaSim.host}:{KafkaSim.port}"
WRITE_TIMEOUT = 20 * sy.TimeSpan.SECOND
# The records that tail reads from the end of a topic.
TAIL = 64

type Record = tuple[str | None, object]


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

    :param bindings: Each field of the message, with the channel it writes to. A field
        can write to several channels.
    :param index: The field of bindings whose channel indexes the others. None stamps
        each sample with the timestamp of its record.
    :param record_key: The Kafka key of the records. Empty reads every record.
    """
    fields = list({f.key: f for f, _ in bindings}.values())
    message = sy.icd.Message(name=topic, payload=sy.icd.JSONPayload(fields=fields))
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


def create_target(
    client: sy.Synnax,
    *,
    topic: str,
    bindings: list[tuple[sy.icd.JSONField, int]],
    fields: list[sy.icd.JSONField] | None = None,
    record_key: str = "",
) -> kafka.WriteTarget:
    """Create an ICD with one JSON message, named after topic and record_key, that
    holds the fields of bindings and fields. Return a target that sends that message to
    topic with record_key.

    :param bindings: Each field that a command channel fills, with the channel.
    :param fields: The other fields of the message. Each sends its default.
    :param record_key: The Kafka key of the records. Empty sends records with no key.
    """
    name = f"{topic} {record_key}".strip()
    message = sy.icd.Message(
        name=name,
        payload=sy.icd.JSONPayload(fields=[f for f, _ in bindings] + (fields or [])),
    )
    ic = client.icds.create(name=name, messages=[message])
    return kafka.WriteTarget(
        topic=topic,
        record_key=record_key,
        icd=ic.key,
        message=sy.icd.WriteMessage(
            message=message.key,
            bindings=[
                sy.icd.ChannelWriteBinding(field=f.key, channel=ch)
                for f, ch in bindings
            ],
        ),
    )


@contextmanager
def open_consumer() -> Iterator[Consumer]:
    """Open a consumer of the simulator broker that commits no offsets."""
    consumer = Consumer(
        {
            "bootstrap.servers": BROKER,
            "group.id": f"synnax-test-{uuid4()}",
            "enable.auto.commit": False,
        }
    )
    try:
        yield consumer
    finally:
        consumer.close()


def watermarks(consumer: Consumer, topic: str) -> tuple[int, int]:
    """Return the low and high watermarks of topic, or 0 and 0 when the topic does not
    exist yet.

    :raises KafkaException: For any other error, such as a timeout.
    """
    try:
        return consumer.get_watermark_offsets(TopicPartition(topic, 0), timeout=5)
    except KafkaException as e:
        if e.args[0].code() != KafkaError._UNKNOWN_PARTITION:
            raise
        return 0, 0


def tail(consumer: Consumer, topic: str, start: int) -> list[Record]:
    """Return the key and JSON value of each record of topic from offset start, oldest
    first, and at most the last 64. Empty when the topic does not exist yet."""
    low, high = watermarks(consumer, topic)
    start = max(low, start, high - TAIL)
    if start >= high:
        return []
    consumer.assign([TopicPartition(topic, 0, start)])
    records: list[Record] = []
    while start + len(records) < high:
        msg = consumer.poll(5)
        if msg is None:
            offset = start + len(records)
            raise AssertionError(f"{topic} returned no record at offset {offset}")
        if (err := msg.error()) is not None:
            raise AssertionError(f"Failed to read {topic}: {err}")
        key, value = msg.key(), msg.value()
        assert value is not None
        records.append((None if key is None else key.decode(), json.loads(value)))
    return records


def wait_for_record(
    consumer: Consumer,
    *,
    topic: str,
    start: int,
    key: str | None = None,
    expected: object,
) -> list[Record]:
    """Wait at most 20 seconds until the newest record of key on topic, from offset
    start, holds the JSON value expected.

    :param start: The high watermark of topic before the commands went out, so that an
        older record cannot match.
    :returns: The records of topic from start, which hold the matching record.
    :raises AssertionError: If no record matches in time.
    """
    return wait_for_match(
        consumer, topic=topic, start=start, key=key, matches=lambda v: v == expected
    )


def wait_for_match(
    consumer: Consumer,
    *,
    topic: str,
    start: int,
    key: str | None = None,
    matches: Callable[[object], bool],
) -> list[Record]:
    """Wait at most 20 seconds until matches returns true for the JSON value of the
    newest record of key on topic, from offset start.

    :param start: The high watermark of topic before the commands went out, so that an
        older record cannot match.
    :returns: The records of topic from start, which hold the matching record.
    :raises AssertionError: If no record matches in time.
    """
    last: object = None
    timer = sy.Timer()
    while timer.elapsed() < WRITE_TIMEOUT:
        records = tail(consumer, topic, start)
        values = [v for k, v in records if k == key]
        if values and matches(last := values[-1]):
            return records
        # Blocks until the topic has a new record, for at most a second.
        consumer.poll(1)
    raise AssertionError(
        f"The newest record of key {key} on {topic} from offset {start} did not match "
        f"within {WRITE_TIMEOUT}, last value: {last}"
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


class KafkaWriteTaskCase(KafkaCase, WriteTaskCase[kafka.WriteTask]):
    """Base class for Kafka write task tests. A subclass checks the records of its
    targets in assert_written, from the offsets in marks."""

    # The high watermark of each target topic before the last round of commands.
    marks: dict[str, int]

    def mark(self) -> None:
        """Record the high watermark of each target topic before a round of commands."""
        assert self.tsk is not None
        with open_consumer() as consumer:
            self.marks = {
                t.topic: watermarks(consumer, t.topic)[1]
                for t in self.tsk.config.targets
            }

    def test_send_commands(self) -> None:
        self.mark()
        super().test_send_commands()

    def test_reconfigure_name(self) -> None:
        self.mark()
        super().test_reconfigure_name()

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[kafka.WriteTarget]:
        """Create the command channels of the task and return its targets."""
        ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> kafka.WriteTask:
        return kafka.WriteTask(
            name=task_name,
            device=device.key,
            targets=self.create_channels(self.client),
        )

    def _channel_keys(self, task: kafka.WriteTask) -> list[int]:
        return [
            b.channel
            for t in task.config.targets
            for b in t.message.bindings
            if isinstance(b, sy.icd.ChannelWriteBinding) and b.channel != 0
        ]
