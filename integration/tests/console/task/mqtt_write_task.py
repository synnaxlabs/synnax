#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.mqtt_sim.server import (
    SPARKPLUG_DEVICE,
    SPARKPLUG_EDGE_NODE,
    SPARKPLUG_GROUP,
)

from console.task.mqtt import MQTTWrite
from tests.console.task.mqtt_case import MQTTCase

# A command topic the simulator echoes, and a tag of a device of its edge node.
TOPIC = "plant/valve/set"
TAG = "setpoint"
# The list titles a Sparkplug B target by its full tag path.
TAG_PATH = f"{SPARKPLUG_GROUP}/{SPARKPLUG_EDGE_NODE}/{SPARKPLUG_DEVICE}/{TAG}"


class MQTTWriteTask(MQTTCase):
    """Configure, start, and stop an MQTT write task through the Console form."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_targets(page)
        self.test_deploy(page)
        self.test_send_commands(page)
        self.test_stop(page)
        self.test_reopen_config(page, [TOPIC, TAG])

    def test_create_task(self) -> MQTTWrite:
        """Create the task page and select the broker."""
        self.log("Testing: Create MQTT write task")
        page = self.create_page(MQTTWrite, prefix="MQTT Console Write")
        self.select_broker(page)
        return page

    def test_add_targets(self, page: MQTTWrite) -> None:
        """A plain target typed in and a tag picked in the add targets dialog are
        listed."""
        self.log("Testing: Add a plain target and a Sparkplug B tag")
        page.add_entry()
        page.set_topic(TOPIC)
        page.configure_channel(
            pointer="/position", json_type="Number", data_type="float64"
        )
        page.add_tags(SPARKPLUG_EDGE_NODE, [TAG])
        listed = page.entries()
        expected = [TOPIC, TAG_PATH]
        assert listed == expected, f"Targets should be {expected}, got {listed}"

    def test_send_commands(self, page: MQTTWrite) -> None:
        """Deploy created one command channel per target; commands publish without
        the task reporting an error."""
        self.log("Testing: Send commands")
        task = self.retrieve_task(page.page_name)
        topics, tags = self.topics(task), self.tags(task)
        assert topics == {TOPIC}, f"Task topics should be {{{TOPIC!r}}}, got {topics}"
        assert tags == {TAG}, f"Task tags should be {{{TAG!r}}}, got {tags}"
        keys = self.channel_keys(task)
        assert len(keys) == 2, f"Every target needs a command channel, got {keys}"
        self.send_commands(page, task=task)
