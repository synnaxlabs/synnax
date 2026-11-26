#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

import synnax as sy
from synnax import kafka
from tests.driver.kafka_task import KafkaReadTaskCase, create_topic
from tests.driver.task import (
    StatusWatch,
    collect_samples,
    create_channel,
    create_index,
    is_running,
    status_text,
)
from x import payload
from x.telem import to_int


class KafkaReadJSON(KafkaReadTaskCase):
    """Reads numbers and an enum label from the JSON value of each record, with the
    timestamp of the record."""

    task_name = "Kafka Read JSON"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.ReadTopic]:
        idx = create_index(client, "kafka_sensors_time")
        temperature = create_channel(
            client,
            name="kafka_sensors_temperature",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        count = create_channel(
            client,
            name="kafka_sensors_count",
            data_type=sy.DataType.INT64,
            index=idx.key,
        )
        running = create_channel(
            client,
            name="kafka_sensors_running",
            data_type=sy.DataType.UINT8,
            index=idx.key,
        )
        return [
            create_topic(
                client,
                topic="plant.sensors",
                bindings=[
                    (
                        sy.icd.FloatJSONField(
                            name="temperature", pointer="/temperature"
                        ),
                        temperature,
                    ),
                    (sy.icd.IntegerJSONField(name="count", pointer="/count"), count),
                    (
                        sy.icd.UnsignedJSONField(
                            name="state",
                            pointer="/state",
                            enum_values=[
                                payload.UnsignedEntry(
                                    label="idle", value=payload.Unsigned(0)
                                ),
                                payload.UnsignedEntry(
                                    label="running", value=payload.Unsigned(1)
                                ),
                            ],
                        ),
                        running,
                    ),
                ],
            ),
        ]


class KafkaReadMultipleTopics(KafkaReadTaskCase):
    """Reads two topics at once: a bare number with the timestamp of its record, and a
    value that carries its own timestamp."""

    task_name = "Kafka Read Multiple Topics"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.ReadTopic]:
        scalar_idx = create_index(client, "kafka_scalar_time")
        scalar = create_channel(
            client,
            name="kafka_scalar_value",
            data_type=sy.DataType.FLOAT32,
            index=scalar_idx.key,
        )
        timed_idx = create_index(client, "kafka_timed_time")
        timed = create_channel(
            client,
            name="kafka_timed_value",
            data_type=sy.DataType.FLOAT64,
            index=timed_idx.key,
        )
        stamp = sy.icd.TimestampJSONField(
            name="timestamp", pointer="/timestamp", time_format="unix_ns"
        )
        return [
            create_topic(
                client,
                topic="plant.scalar",
                bindings=[(sy.icd.FloatJSONField(name="value", pointer=""), scalar)],
            ),
            create_topic(
                client,
                topic="plant.timed",
                index=stamp,
                bindings=[
                    (stamp, timed_idx.key),
                    (sy.icd.FloatJSONField(name="value", pointer="/value"), timed),
                ],
            ),
        ]


class KafkaReadRecordKinds(KafkaReadTaskCase):
    """Reads two kinds of record with one key, each matched by the field it holds. The
    task does not read the records of another key on the same topic."""

    task_name = "Kafka Read Record Kinds"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.ReadTopic]:
        speed_idx = create_index(client, "kafka_pump_speed_time")
        speed = create_channel(
            client,
            name="kafka_pump_speed",
            data_type=sy.DataType.FLOAT64,
            index=speed_idx.key,
        )
        fault_idx = create_index(client, "kafka_pump_fault_time")
        fault = create_channel(
            client,
            name="kafka_pump_fault",
            data_type=sy.DataType.INT64,
            index=fault_idx.key,
        )
        speed_field = sy.icd.FloatJSONField(name="speed", pointer="/speed")
        fault_field = sy.icd.IntegerJSONField(name="fault", pointer="/fault")
        speed_msg = sy.icd.Message(
            name="Speed",
            payload=sy.icd.JSONPayload(
                match=sy.icd.PresentJSONMatch(pointer="/speed"), fields=[speed_field]
            ),
        )
        fault_msg = sy.icd.Message(
            name="Fault",
            payload=sy.icd.JSONPayload(
                match=sy.icd.PresentJSONMatch(pointer="/fault"), fields=[fault_field]
            ),
        )
        ic = client.icds.create(name="plant.events", messages=[speed_msg, fault_msg])
        return [
            kafka.ReadTopic(
                name="plant.events",
                records=[
                    kafka.ReadRecord(
                        record_key="pump",
                        icd=ic.key,
                        messages=[
                            sy.icd.ReadMessage(
                                message=speed_msg.key,
                                bindings=[
                                    sy.icd.ReadBinding(
                                        field=speed_field.key, channel=speed
                                    )
                                ],
                            ),
                            sy.icd.ReadMessage(
                                message=fault_msg.key,
                                bindings=[
                                    sy.icd.ReadBinding(
                                        field=fault_field.key, channel=fault
                                    )
                                ],
                            ),
                        ],
                    )
                ],
            )
        ]


class KafkaReadDecodeFailure(KafkaReadTaskCase):
    """Reads the count of each record into two index groups, and the value into the
    second. A value that does not decode drops its record from the second group only,
    and the task warns."""

    task_name = "Kafka Read Decode Failure"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.ReadTopic]:
        count_idx = create_index(client, "kafka_mixed_count_time")
        count = create_channel(
            client,
            name="kafka_mixed_count",
            data_type=sy.DataType.INT64,
            index=count_idx.key,
        )
        value_idx = create_index(client, "kafka_mixed_value_time")
        value_count = create_channel(
            client,
            name="kafka_mixed_value_count",
            data_type=sy.DataType.INT64,
            index=value_idx.key,
        )
        value = create_channel(
            client,
            name="kafka_mixed_value",
            data_type=sy.DataType.FLOAT64,
            index=value_idx.key,
        )
        count_field = sy.icd.IntegerJSONField(name="count", pointer="/count")
        return [
            create_topic(
                client,
                topic="plant.mixed",
                bindings=[
                    (count_field, count),
                    (count_field, value_count),
                    (sy.icd.FloatJSONField(name="value", pointer="/value"), value),
                ],
            )
        ]

    def run(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()
        count, value_count, _ = self._channel_keys(self.tsk)
        with StatusWatch(self.client, [self.tsk.key]) as watch:
            with self.tsk.run(timeout=self.COMMAND_TIMEOUT):
                start = collect_samples(
                    self.client,
                    [count],
                    duration=self.TASK_DURATION,
                    frame_size=1,
                )
                warning = watch.wait(
                    lambda s: s.variant == "warning",
                    timeout=10 * sy.TimeSpan.SECOND,
                    expected="the warning of a skipped record",
                )
        message = re.fullmatch(r"Skipped \d+ of \d+ records?", warning.message)
        description = re.fullmatch(
            r'Record value: field value: invalid number "ERR": validation error '
            r"\(topic plant\.mixed, partition 0, offset \d+\)\.",
            warning.description,
        )
        if message is None or description is None or not is_running(warning):
            raise AssertionError(f"Unexpected warning: {status_text(warning)}")
        span = sy.TimeRange(start, sy.TimeStamp.now())
        counts = [to_int(c) for c in self.client.channels.retrieve(count).read(span)]
        if counts != list(range(counts[0], counts[0] + len(counts))):
            raise AssertionError(f"The count group dropped records: {counts}")
        decoded = [
            to_int(c) for c in self.client.channels.retrieve(value_count).read(span)
        ]
        expected = [c for c in counts if c % 10 != 0]
        if decoded != expected or len(decoded) == len(counts):
            raise AssertionError(f"The value group read {decoded}, expected {expected}")
