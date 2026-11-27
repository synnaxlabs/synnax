#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.task.kafka import KafkaWrite
from tests.console.task.kafka_case import KafkaCase
from tests.driver.kafka_task import open_consumer, wait_for_record, watermarks

# A topic of bare numbers. The simulator produces it with no key, so the target's key
# tells its records apart.
TOPIC = "plant.scalar"
RECORD_KEY = "console"
# The second round of commands.
LAST_COMMAND = 100.0


class KafkaWriteTask(KafkaCase):
    """Configure, start, and stop a Kafka write task through the Console form, with a
    target added from a topic that the cluster lists."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_target(page)
        self.test_deploy(page)
        self.test_send_commands(page)
        self.test_stop(page)
        self.test_reopen_config(page, [TOPIC, RECORD_KEY])

    def test_create_task(self) -> KafkaWrite:
        """Create the task page and select the cluster."""
        self.log("Testing: Create Kafka write task")
        page = self.create_page(KafkaWrite, prefix="Kafka Console Write")
        page.select_cluster(self.device_name)
        return page

    def test_add_target(self, page: KafkaWrite) -> None:
        """A topic picked in the add targets dialog is listed with one target, which
        takes the record key set in its details."""
        self.log("Testing: Add a target from a listed topic")
        page.add_topics([TOPIC])
        listed = page.topics()
        assert listed == [TOPIC], f"Topics should be {[TOPIC]}, got {listed}"
        page.set_record_key(TOPIC, RECORD_KEY)
        keys = page.record_keys(TOPIC)
        assert keys == [RECORD_KEY], f"Targets should be {[RECORD_KEY]}, got {keys}"

    def test_send_commands(self, page: KafkaWrite) -> None:
        """Deploy created a command channel for the target; each command produces a
        record of the bare value under the target's key."""
        self.log("Testing: Send commands")
        task = self.retrieve_task(page.page_name)
        keys = self.channel_keys(task)
        assert len(keys) == 1, f"The target needs a command channel, got {keys}"
        with open_consumer() as consumer:
            start = watermarks(consumer, TOPIC)[1]
            self.send_commands(page, task=task)
            wait_for_record(
                consumer,
                topic=TOPIC,
                start=start,
                key=RECORD_KEY,
                expected=LAST_COMMAND,
            )
