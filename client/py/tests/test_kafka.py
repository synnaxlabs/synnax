#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest

import synnax as sy
from tests.json_value import obj
from tests.task_targets import TaskTargets
from x.strings import random_name


@pytest.mark.kafka
class TestKafkaReadRecord:
    """Tests for Kafka read record configuration."""

    def test_take_icd_read_messages(self) -> None:
        """Should take the ICD read messages that an MQTT entry takes."""
        message = sy.icd.ReadMessage(
            message=uuid4(),
            bindings=[sy.icd.ReadBinding(field=uuid4(), channel=12)],
        )
        record = sy.kafka.ReadRecord(icd=uuid4(), messages=[message])
        assert record.messages == [message]


@pytest.mark.kafka
class TestKafkaReadTask:
    """Tests for Kafka read task configuration."""

    def test_payload_type(self) -> None:
        """Should create a task of the Kafka read type."""
        task = sy.kafka.ReadTask(device="cluster-1", name="Oven read")
        pld = task.to_payload()
        assert pld.type == "kafka_read"
        assert pld.name == "Oven read"

    def test_create_and_retrieve_read_task(
        self, client: sy.Synnax, task_targets: TaskTargets
    ) -> None:
        """Should store a read config that the task class parses again."""
        value = sy.icd.FloatJSONField(name="value", pointer="/v")
        message = sy.icd.Message(
            name="Oven", payload=sy.icd.JSONPayload(fields=[value])
        )
        ic = client.icds.create(name=random_name(), messages=[message])
        task = sy.kafka.ReadTask(
            device="some-cluster",
            data_saving_disabled=True,
            start_offset="earliest",
            topics=[
                sy.kafka.ReadTopic(
                    name="sensor-readings",
                    records=[
                        sy.kafka.ReadRecord(
                            record_key="oven",
                            icd=ic.key,
                            messages=[
                                sy.icd.ReadMessage(
                                    message=message.key,
                                    bindings=[
                                        sy.icd.ReadBinding(
                                            field=value.key, channel=1234
                                        )
                                    ],
                                )
                            ],
                        )
                    ],
                )
            ],
        )
        task_targets.attach(task.config)
        created = client.tasks.create(
            name="test-kafka-read-task",
            type="kafka_read",
            config=task.config.model_dump(exclude_none=True),
        )
        parsed = sy.kafka.ReadTask(created)
        assert parsed.config.data_saving_disabled is True
        assert parsed.config.start_offset == "earliest"
        assert parsed.config.topics == task.config.topics


@pytest.mark.kafka
class TestKafkaWriteTask:
    """Tests for Kafka write task configuration."""

    def test_payload_type(self) -> None:
        """Should create a task of the Kafka write type."""
        task = sy.kafka.WriteTask(name="Write", device="cluster-1")
        assert task.to_payload().type == "kafka_write"

    def test_create_and_retrieve_write_task(
        self, client: sy.Synnax, task_targets: TaskTargets
    ) -> None:
        """Should store a write config that the task class parses again."""
        speed = sy.icd.FloatJSONField(name="speed", pointer="/speed")
        unit = sy.icd.StringJSONField(name="unit", pointer="/unit")
        message = sy.icd.Message(
            name="plant.pumps", payload=sy.icd.JSONPayload(fields=[speed, unit])
        )
        ic = client.icds.create(name=random_name(), messages=[message])
        task = sy.kafka.WriteTask(
            device="some-cluster",
            targets=[
                sy.kafka.WriteTarget(
                    topic="plant.pumps",
                    record_key="pump-1",
                    icd=ic.key,
                    message=sy.icd.WriteMessage(
                        message=message.key,
                        bindings=[
                            sy.icd.ChannelWriteBinding(field=speed.key, channel=4321),
                            sy.icd.ValueWriteBinding(
                                field=unit.key,
                                value=sy.icd.StringWriteValue(value="rpm"),
                            ),
                        ],
                    ),
                )
            ],
        )
        task_targets.attach(task.config)
        created = client.tasks.create(
            name="test-kafka-write-task",
            type="kafka_write",
            config=task.config.model_dump(exclude_none=True),
        )
        assert sy.kafka.WriteTask(created).config.targets == task.config.targets

    def test_write_task_updates_device_properties(self, client: sy.Synnax) -> None:
        """Should record the command channel of each field of each enabled target in
        device properties."""
        rack = client.racks.retrieve_embedded_rack()
        dev = client.devices.create(
            sy.Device(
                key=random_name(),
                name="Test Kafka Write Cluster",
                rack=rack.key,
                location="localhost:9092",
                make="kafka",
                model="cluster",
                properties={"brokers": ["localhost:9092"]},
            )
        )
        suffix = random_name()
        cmd_time = client.channels.create(
            name=f"kafka_cmd_time_{suffix}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        cmd_ch = client.channels.create(
            name=f"kafka_cmd_{suffix}",
            data_type=sy.DataType.FLOAT64,
            index=cmd_time.key,
        )
        speed = sy.icd.FloatJSONField(name="speed", pointer="/speed")
        unit = sy.icd.StringJSONField(name="unit", pointer="/unit")
        message = sy.icd.Message(
            name="plant.pumps", payload=sy.icd.JSONPayload(fields=[speed, unit])
        )
        ic = client.icds.create(name=random_name(), messages=[message])

        def target(record_key: str, *, disabled: bool = False) -> sy.kafka.WriteTarget:
            return sy.kafka.WriteTarget(
                topic="plant.pumps",
                record_key=record_key,
                disabled=disabled,
                icd=ic.key,
                message=sy.icd.WriteMessage(
                    message=message.key,
                    bindings=[
                        sy.icd.ChannelWriteBinding(field=speed.key, channel=cmd_ch.key),
                        sy.icd.ValueWriteBinding(
                            field=unit.key, value=sy.icd.StringWriteValue(value="rpm")
                        ),
                    ],
                ),
            )

        task = sy.kafka.WriteTask(
            name="Test Kafka Write",
            device=dev.key,
            targets=[target("pump-1"), target("pump-2", disabled=True)],
        )
        task.update_device_properties(client.devices)

        props = client.devices.retrieve(key=dev.key).properties
        assert obj(props, "write") == {f"plant.pumps#pump-1#{speed.key}": cmd_ch.key}
