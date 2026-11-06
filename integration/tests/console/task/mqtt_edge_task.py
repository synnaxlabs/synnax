#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.task.mqtt import MQTTEdge
from synnax import mqtt
from tests.console.task.mqtt_case import MQTTCase
from tests.driver.mqtt_task import run_reader, write_until
from tests.driver.task import create_channel, create_index, delete_channels

# An edge node of its own, so it does not collide with the one the simulator runs.
GROUP = "ConsolePlant"
EDGE_NODE = "Line9"
TAG = "temperature"
CHANNEL = "mqtt_console_edge_temperature"
# The context menu names the command channel after the tag; configure creates it.
COMMAND_CHANNEL = f"{TAG}_cmd"
# The simulator mirrors every tag it sees on a plain topic.
MIRROR_TOPIC = f"plant/sparkplug/{GROUP}/{EDGE_NODE}/{TAG}"
VALUE = 21.5


class MQTTEdgeTask(MQTTCase):
    """Configure, start, and stop a Sparkplug B edge node through the Console form,
    and check that a tag it publishes reaches the broker."""

    def setup(self) -> None:
        super().setup()
        self.index = create_index(self.client, f"{CHANNEL}_time").key
        self.channel = create_channel(
            self.client, name=CHANNEL, data_type=sy.DataType.FLOAT64, index=self.index
        )

    def teardown(self) -> None:
        try:
            with self._try_to("delete the tag channel"):
                delete_channels(self.client, [self.channel, self.index])
        finally:
            super().teardown()

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_tag(page)
        self.test_add_command_channel(page)
        self.test_deploy(page)
        self.test_command_channel_created(page)
        self.test_tag_reaches_broker(page)
        self.test_stop(page)
        self.test_reopen_config(page, [COMMAND_CHANNEL])
        tags = page.tags()
        assert tags == [TAG], f"Tags should be {[TAG]}, got {tags}"

    def test_create_task(self) -> MQTTEdge:
        """A new edge node is not deployed; select the broker and name the node."""
        self.log("Testing: Create Sparkplug edge node")
        page = self.create_page(MQTTEdge, prefix="MQTT Console Edge")
        self.assert_not_deployed(page, label="new task")
        page.select_broker(self.device_name)
        page.set_node(GROUP, EDGE_NODE)
        return page

    def test_add_tag(self, page: MQTTEdge) -> None:
        """A tag takes the name of its channel, and keeps a name typed over it."""
        self.log("Testing: Add a tag")
        page.add_tag(CHANNEL, name=TAG)
        tags = page.tags()
        assert tags == [TAG], f"Tags should be {[TAG]}, got {tags}"

    def test_add_command_channel(self, page: MQTTEdge) -> None:
        """The context menu names a command channel after the tag."""
        self.log("Testing: Add a command channel")
        page.add_command_channel(TAG)
        name = page.command_channel(TAG)
        assert name == COMMAND_CHANNEL, (
            f"The command channel should be '{COMMAND_CHANNEL}', got '{name}'"
        )

    def test_command_channel_created(self, page: MQTTEdge) -> None:
        """Configure created the named command channel, of the type of the tag."""
        self.log("Testing: Configure created the command channel")
        [tag] = mqtt.EdgeTask(self.retrieve_task(page.page_name)).config.tags
        assert tag.channel == self.channel, (
            f"The tag should publish {self.channel}, got {tag.channel}"
        )
        assert tag.command_channel != 0, "Configure should create the command channel"
        command = self.client.channels.retrieve(tag.command_channel)
        assert command.name == COMMAND_CHANNEL, (
            f"The command channel should be '{COMMAND_CHANNEL}', got '{command.name}'"
        )
        assert command.data_type == sy.DataType.FLOAT64, (
            f"A double tag needs a float64 command channel, got {command.data_type}"
        )

    def test_tag_reaches_broker(self, page: MQTTEdge) -> None:
        """A sample written to the channel of the tag shows up on the mirror topic of
        the simulator, read back through a plain read task."""
        self.log("Testing: Tag value reaches the broker")
        device = self.client.devices.retrieve(name=self.device_name)
        channel = self.client.channels.retrieve(self.channel)
        with run_reader(
            self.client,
            device=device.key,
            name=f"{page.page_name} Mirror",
            channel=f"{CHANNEL}_mirror",
            data_type=sy.DataType.FLOAT64,
            entry=lambda mirror: mqtt.PlainReadEntry(
                topic=MIRROR_TOPIC,
                fields=[
                    mqtt.ReadField(
                        pointer="/value",
                        channel=mirror.key,
                        data_type=sy.DataType.FLOAT64,
                    )
                ],
            ),
        ) as mirror:
            write_until(self.client, channel=channel, value=VALUE, watched=mirror)
        status = page.status()
        assert status["level"] == "success", (
            f"The edge node should keep running, but the status is "
            f"'{status['level']}': '{status['msg']}'"
        )
