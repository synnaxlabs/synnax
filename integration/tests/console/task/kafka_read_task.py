#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.task.kafka import KafkaRead
from tests.console.task.kafka_case import KafkaCase

# A JSON record and a bare number, both produced at the simulator's rate.
SENSORS = "plant.sensors"
SCALAR = "plant.scalar"
# The add data dialog adds topics in order of name.
TOPICS = [SCALAR, SENSORS]
# A pick in the add data dialog takes every field of plant.sensors.
SENSOR_FIELDS = ["temperature", "pressure", "count", "state"]
# The fields of plant.sensors and the whole value of plant.scalar.
DATA_CHANNELS = 5


class KafkaReadTask(KafkaCase):
    """Configure, start, and stop a Kafka read task through the Console form, with data
    added from topics that the cluster lists."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_data(page)
        self.test_deploy(page)
        self.test_data_flow(page)
        self.test_reopen_config(page, TOPICS)

    def test_create_task(self) -> KafkaRead:
        """A new task is not deployed; select the cluster."""
        self.log("Testing: Create Kafka read task")
        page = self.create_page(KafkaRead, prefix="Kafka Console Read")
        self.assert_not_deployed(page, label="new task")
        page.select_cluster(self.device_name)
        return page

    def test_add_data(self, page: KafkaRead) -> None:
        """Topics picked in the add data dialog are listed with one record for any key,
        and the task reads the fields of their records."""
        self.log("Testing: Add data from listed topics")
        page.add_topics(TOPICS)
        listed = page.topics()
        assert listed == TOPICS, f"Topics should be {TOPICS}, got {listed}"
        records = page.record_keys(SENSORS)
        assert records == ["Any key"], f"Records should be ['Any key'], got {records}"
        fields = page.fields(SENSORS)
        assert fields == SENSOR_FIELDS, (
            f"Fields should be {SENSOR_FIELDS}, got {fields}"
        )

    def test_data_flow(self, page: KafkaRead) -> None:
        """Deploy created one channel per field; each collects samples at the produce
        rate of the simulator."""
        self.log("Testing: Data flows on the created channels")
        task = self.retrieve_task(page.page_name)
        keys = self.channel_keys(task)
        assert len(keys) == DATA_CHANNELS, (
            f"Every read field needs a channel, got {keys}"
        )
        self.assert_sample_count(page, keys=keys)
