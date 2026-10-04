#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.task.mqtt import MQTTRead
from tests.console.task.mqtt_case import MQTTCase

# A JSON payload and a bare number, both published at the simulator's rate.
TOPICS = ["plant/sensors", "plant/scalar"]
# The fields the Browser reads off the payload sample of plant/sensors. The string
# field comes in disabled, because it cannot share the index of the numbers.
SENSOR_FIELDS = ["/temperature", "/pressure", "/count", "/state"]
DATA_CHANNELS = 4


class MQTTReadTask(MQTTCase):
    """Configure, start, and stop an MQTT read task through the Console form."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_entries(page)
        self.test_deploy(page)
        self.test_data_flow(page)
        self.test_reopen_config(page, TOPICS)

    def test_create_task(self) -> MQTTRead:
        """A new task is not deployed; select the broker and open its Browser."""
        self.log("Testing: Create MQTT read task")
        page = self.create_page(MQTTRead, "MQTT Console Read")
        self.assert_not_deployed(page, "new task")
        self.select_broker(page)
        return page

    def test_add_entries(self, page: MQTTRead) -> None:
        """Topics dragged from the Browser are listed with the fields of their
        payloads."""
        self.log("Testing: Add entries from the Browser")
        page.browse_topics("plant/#")
        page.add_topics(TOPICS)
        listed = page.entries()
        assert listed == TOPICS, f"Entries should be {TOPICS}, got {listed}"
        page.select_entry(TOPICS[0])
        fields = page.fields()
        assert fields == SENSOR_FIELDS, (
            f"Fields should be {SENSOR_FIELDS}, got {fields}"
        )

    def test_data_flow(self, page: MQTTRead) -> None:
        """Deploy created one channel per field; each collects samples at the
        publish rate of the simulator."""
        self.log("Testing: Data flows on the created channels")
        task = self.retrieve_task(page.page_name)
        topics = self.topics(task)
        assert topics == set(TOPICS), (
            f"Task topics should be {set(TOPICS)}, got {topics}"
        )
        keys = self.data_channel_keys(task)
        assert len(keys) == DATA_CHANNELS, (
            f"Every enabled field needs a channel, got {keys}"
        )
        self.assert_sample_count(page, keys)
