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
from tests.driver.task import create_channel, create_index, delete_channels

# An edge node of its own, so it does not collide with the one the simulator runs.
GROUP = "ConsolePlant"
EDGE_NODE = "Line9"
TAG = "temperature"
CHANNEL = "mqtt_console_edge_temperature"
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
        self.test_deploy(page)
        self.test_tag_reaches_broker(page)
        self.test_stop(page)
        self.test_reopen_config(page, [])
        tags = page.tags()
        assert tags == [TAG], f"Tags should be {[TAG]}, got {tags}"

    def test_create_task(self) -> MQTTEdge:
        """A new edge node is not deployed; select the broker and name the node."""
        self.log("Testing: Create MQTT Sparkplug B edge node")
        page = self.create_page(MQTTEdge, "MQTT Console Edge")
        self.assert_not_deployed(page, "new task")
        page.select_broker(self.device_name)
        page.set_node(GROUP, EDGE_NODE)
        return page

    def test_add_tag(self, page: MQTTEdge) -> None:
        """A tag takes the name of its channel, and keeps a name typed over it."""
        self.log("Testing: Add a tag")
        page.add_tag(CHANNEL, TAG)
        tags = page.tags()
        assert tags == [TAG], f"Tags should be {[TAG]}, got {tags}"

    def test_tag_reaches_broker(self, page: MQTTEdge) -> None:
        """A sample written to the channel of the tag shows up on the mirror topic of
        the simulator, read back through a plain read task."""
        self.log("Testing: Tag value reaches the broker")
        task = self.retrieve_task(page.page_name)
        keys = self.channel_keys(task)
        assert keys == [self.channel], (
            f"The tag should publish {self.channel}, got {keys}"
        )
        device = self.client.devices.retrieve(name=self.device_name)
        mirror = self.client.channels.create(
            name=f"{CHANNEL}_mirror",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        reader = mqtt.ReadTask(
            name=f"{page.page_name} Mirror",
            device=device.key,
            entries=[
                mqtt.PlainReadEntry(
                    topic=MIRROR_TOPIC,
                    fields=[
                        mqtt.ReadField(
                            pointer="/value", channel=mirror.key, data_type="float64"
                        )
                    ],
                )
            ],
        )
        self.client.tasks.configure(reader)
        index = self.index
        try:
            with reader.run():
                with self.client.open_streamer([mirror.key]) as streamer:
                    with self.client.open_writer(
                        start=sy.TimeStamp.now(), channels=[index, self.channel]
                    ) as writer:
                        value = None
                        timer = sy.Timer()
                        while (
                            value != VALUE and timer.elapsed() < 20 * sy.TimeSpan.SECOND
                        ):
                            writer.write(
                                {index: sy.TimeStamp.now(), self.channel: VALUE}
                            )
                            frame = streamer.read(timeout=1)
                            if frame is not None and mirror.key in frame:
                                value = float(frame[mirror.key][-1])
            if value != VALUE:
                raise AssertionError(f"The broker saw {value}, not {VALUE}")
        finally:
            self.client.tasks.delete(reader.key)
            self.client.channels.delete([mirror.key])
        status = page.status()
        assert status["level"] == "success", (
            f"The edge node should keep running, but the status is "
            f"'{status['level']}': '{status['msg']}'"
        )
