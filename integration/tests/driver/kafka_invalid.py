#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import synnax as sy
from synnax import kafka
from tests.driver.kafka_task import KafkaCase, create_target, create_topic
from tests.driver.task import (
    assert_create_refused,
    assert_start_rejected,
    create_channel,
    create_index,
)


def assert_text(label: str, text: str, expected: str) -> None:
    if text != expected:
        raise AssertionError(f"Refusal of {label} is {text!r}, expected {expected!r}")


class KafkaInvalidConfig(KafkaCase):
    """Saves Kafka tasks that the Core refuses, on save or on start, and checks the text
    of each refusal."""

    def setup(self) -> None:
        super().setup()
        self.device = self.client.devices.retrieve(name=self.device_name)

    def run(self) -> None:
        self.test_no_enabled_targets()
        self.test_bad_icd_binding()
        self.test_missing_channel()

    def test_no_enabled_targets(self) -> None:
        self.log("Testing: No enabled targets")
        idx = create_index(self.client, "kafka_inv_targets_time")
        cmd = create_channel(
            self.client,
            name="kafka_inv_targets_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        target = create_target(
            self.client,
            topic="plant.invalid.set",
            bindings=[(sy.icd.FloatJSONField(name="value", pointer="/value"), cmd)],
        )
        target.disabled = True
        task = kafka.WriteTask(
            name="Kafka No Enabled Targets",
            device=self.device.key,
            targets=[target],
        )
        label = "no enabled targets"
        assert_text(
            label,
            assert_start_rejected(self.client, task, label),
            "Failed to configure task: targets: the task has no enabled targets: "
            "validation error",
        )

    def test_bad_icd_binding(self) -> None:
        self.log("Testing: A binding to a field that the message does not hold")
        idx = create_index(self.client, "kafka_inv_binding_time")
        ch = create_channel(
            self.client,
            name="kafka_inv_binding_value",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        topic = create_topic(
            self.client,
            topic="plant.sensors",
            bindings=[
                (sy.icd.FloatJSONField(name="temperature", pointer="/temperature"), ch)
            ],
        )
        missing = uuid4()
        topic.records[0].messages[0].bindings = [
            sy.icd.ReadBinding(field=missing, channel=ch)
        ]
        task = kafka.ReadTask(
            name="Kafka Bad Binding", device=self.device.key, topics=[topic]
        )
        label = "a bad ICD binding"
        assert_text(
            label,
            assert_create_refused(self.client, task, label),
            "topics.0.records.0.messages.0.bindings.0.field: field "
            f"{missing} does not exist in message plant.sensors",
        )

    def test_missing_channel(self) -> None:
        self.log("Testing: A target bound to a channel that does not exist")
        target = create_target(
            self.client,
            topic="plant.invalid.set",
            bindings=[
                (sy.icd.FloatJSONField(name="value", pointer="/value"), 999999999)
            ],
        )
        task = kafka.WriteTask(
            name="Kafka Missing Channel", device=self.device.key, targets=[target]
        )
        label = "a missing channel"
        assert_text(
            label,
            assert_create_refused(self.client, task, label),
            "targets.0.message.bindings.0.channel: channel 999999999 does not exist",
        )
