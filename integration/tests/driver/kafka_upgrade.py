#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import tempfile
from typing import Any
from uuid import UUID

import synnax as sy
from framework.utils import get_core_fixture_path
from synnax import kafka
from tests.driver.kafka_task import KafkaCase, open_consumer, wait_for_match, watermarks
from tests.driver.task import (
    assert_sample_counts_in_range,
    collect_samples,
    create_channel,
    create_index,
)

DURATION = 3 * sy.TimeSpan.SECOND
COMMAND = 42.0


def load(fixture: str) -> dict[str, Any]:
    """Return the task file that a 0.62 Core exported, from the Core testdata."""
    path = get_core_fixture_path(f"task/testdata/legacy/{fixture}.json")
    with open(path, encoding="utf-8") as f:
        file: dict[str, Any] = json.load(f)
    return file


def rekeyed(value: object, channels: dict[int, int], device: str) -> object:
    """Return value with each channel key in channels swapped for its value, and each
    device key swapped for device."""
    if isinstance(value, list):
        return [rekeyed(v, channels, device) for v in value]
    if not isinstance(value, dict):
        return value
    out: dict[str, object] = {}
    for k, v in value.items():
        if k == "device":
            out[k] = device
        elif k == "channel" and v in channels:
            out[k] = channels[v]
        else:
            out[k] = rekeyed(v, channels, device)
    return out


class KafkaUpgrade(KafkaCase):
    """Imports a Kafka read task and a Kafka write task that a 0.62 Core exported,
    configures each one, then runs it against the broker."""

    # The rate at which the simulator produces each topic.
    SAMPLE_RATE = 10 * sy.Rate.HZ

    def setup(self) -> None:
        self.imported: list[UUID] = []
        super().setup()
        self.device = self.client.devices.retrieve(name=self.device_name)

    def teardown(self) -> None:
        if self.imported:
            with self._try_to("delete the imported tasks"):
                self.client.tasks.delete(self.imported)
        super().teardown()

    def run(self) -> None:
        self.test_read()
        self.test_write()

    def import_file(
        self, file: dict[str, Any], channels: dict[int, int], name: str
    ) -> sy.Task:
        """Import file as a draft task named name, bound to the created channels and to
        the device of the simulator.

        :param channels: The created channel for each channel key of file.
        """
        body = rekeyed(file, channels, self.device.key)
        assert isinstance(body, dict)
        body["name"] = name
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, f"{name}.json")
            with open(path, "w", encoding="utf-8") as f:
                json.dump(body, f)
            rid = self.client.imex.import_(path)
        self.imported.append(UUID(rid.key))
        return self.client.tasks.retrieve(UUID(rid.key))

    def test_read(self) -> None:
        self.log("Testing: A 0.62 read task reads the timed topic")
        file = load("kafka_read")
        idx = create_index(self.client, "kafka_v062_time")
        value = create_channel(
            self.client,
            name="kafka_v062_value",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        created = {"kafka_timed_time": idx.key, "kafka_timed_value": value}
        fields = file["records"][0]["fields"]
        draft = self.import_file(
            file,
            {f["channel"]: created[f["name"]] for f in fields},
            "Kafka Upgrade Read",
        )
        tsk = self.client.tasks.configure(kafka.ReadTask(internal=draft))
        keys = [idx.key, value]
        with tsk.run():
            start = collect_samples(self.client, keys, duration=DURATION, frame_size=1)
        assert_sample_counts_in_range(
            self.client,
            channel_keys=keys,
            time_range=sy.TimeRange(start, sy.TimeStamp.now()),
            expected_samples=int(float(self.SAMPLE_RATE) * DURATION.seconds),
            task_name=tsk.name,
        )

    def test_write(self) -> None:
        self.log("Testing: A 0.62 write task sends a command")
        file = load("kafka_write")
        idx = create_index(self.client, "kafka_v062_cmd_time")
        name = "kafka_v062_cmd"
        cmd = create_channel(
            self.client, name=name, data_type=sy.DataType.FLOAT64, index=idx.key
        )
        draft = self.import_file(
            file, {file["channels"][0]["channel"]: cmd}, "Kafka Upgrade Write"
        )
        tsk = self.client.tasks.configure(kafka.WriteTask(internal=draft))
        topic = file["topic"]
        with open_consumer() as consumer, tsk.run():
            start = watermarks(consumer, topic)[1]
            sent = sy.TimeStamp.now()
            with self.client.open_writer(start=sent, channels=[idx.key, cmd]) as w:
                w.write({idx.key: sent, cmd: COMMAND})
            # The 0.62 record sends the value, the channel name, and the write time.
            wait_for_match(
                consumer,
                topic=topic,
                start=start,
                key=name,
                matches=lambda v: (
                    isinstance(v, dict)
                    and v.get("value") == COMMAND
                    and v.get("channel") == name
                    and isinstance(v.get("time"), int)
                    and v["time"] >= int(sent)
                ),
            )
