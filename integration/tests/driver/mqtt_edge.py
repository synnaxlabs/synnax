#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from examples.mqtt_sim import MQTTSim

import synnax as sy
from synnax import mqtt
from tests.driver.mqtt_task import CORE_RACK_NAME
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import WriteTaskCase, create_channel, create_index

GROUP = "Plant"
EDGE_NODE = "SynnaxLine"
MIRROR = f"plant/sparkplug/{GROUP}/{EDGE_NODE}"


class MQTTEdgeNode(SimulatorCase, WriteTaskCase):
    """Runs a Sparkplug B edge node on the Core. The simulator mirrors the tags that
    the edge node publishes on plain topics, and turns a plain message into a command
    for a tag, so plain read and write tasks can check both directions."""

    sim_classes = [MQTTSim]
    RACK_NAME = CORE_RACK_NAME
    task_name = "MQTT Sparkplug Edge Node"

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> mqtt.EdgeTask:
        idx = create_index(self.client, "mqtt_edge_node_time")
        temperature = create_channel(
            self.client,
            name="mqtt_edge_node_temperature",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        setpoint = create_channel(
            self.client,
            name="mqtt_edge_node_setpoint",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        cmd_idx = create_index(self.client, "mqtt_edge_node_setpoint_cmd_time")
        setpoint_cmd = create_channel(
            self.client,
            name="mqtt_edge_node_setpoint_cmd",
            data_type=sy.DataType.FLOAT64,
            index=cmd_idx.key,
        )
        return mqtt.EdgeTask(
            name=task_name,
            device=device.key,
            group=GROUP,
            edge_node=EDGE_NODE,
            authority=200,
            tags=[
                mqtt.EdgeTag(
                    name="temperature", channel=temperature, sparkplug_type="double"
                ),
                mqtt.EdgeTag(
                    name="setpoint",
                    channel=setpoint,
                    sparkplug_type="double",
                    command_channel=setpoint_cmd,
                ),
            ],
        )

    def _channel_keys(self, task: sy.Task) -> list[int]:
        return [t.channel for t in task.config.tags]

    def run(self) -> None:
        super().run()
        self.test_tag_reaches_host()
        self.test_command_reaches_channel()

    def _mirror_reader(self, tag: str, channel: sy.Channel) -> mqtt.ReadTask:
        device = self.client.devices.retrieve(name=self.device_name)
        reader = mqtt.ReadTask(
            name=f"{self.task_name} {tag} Mirror",
            device=device.key,
            entries=[
                mqtt.PlainReadEntry(
                    topic=f"{MIRROR}/{tag}",
                    fields=[
                        mqtt.ReadField(
                            pointer="/value", channel=channel.key, data_type="float64"
                        )
                    ],
                )
            ],
        )
        self.client.tasks.configure(reader)
        return reader

    def test_tag_reaches_host(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Tag value reaches the host")
        mirror = self.client.channels.create(
            name="mqtt_edge_node_temperature_mirror",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        reader = self._mirror_reader("temperature", mirror)
        temperature = self.client.channels.retrieve("mqtt_edge_node_temperature")
        try:
            with reader.run(), self.tsk.run():
                with self.client.open_streamer([mirror.key]) as streamer:
                    with self.client.open_writer(
                        start=sy.TimeStamp.now(),
                        channels=[temperature.index, temperature.key],
                    ) as writer:
                        value = None
                        timer = sy.Timer()
                        while (
                            value != 21.5 and timer.elapsed() < 20 * sy.TimeSpan.SECOND
                        ):
                            writer.write(
                                {
                                    temperature.index: sy.TimeStamp.now(),
                                    temperature.key: 21.5,
                                }
                            )
                            frame = streamer.read(timeout=1)
                            if frame is not None and mirror.key in frame:
                                value = float(frame[mirror.key][-1])
            if value != 21.5:
                raise AssertionError(f"The host saw {value}, not 21.5")
        finally:
            self.client.tasks.delete(reader.key)
            self.client.channels.delete([mirror.key])

    def test_command_reaches_channel(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Command reaches the command channel")
        device = self.client.devices.retrieve(name=self.device_name)
        set_idx = create_index(self.client, "mqtt_edge_node_setpoint_set_time")
        set_ch = self.client.channels.create(
            name="mqtt_edge_node_setpoint_set",
            data_type=sy.DataType.FLOAT64,
            index=set_idx.key,
            retrieve_if_name_exists=True,
        )
        writer_task = mqtt.WriteTask(
            name=f"{self.task_name} Command Sender",
            device=device.key,
            targets=[
                mqtt.PlainWriteTarget(
                    topic=f"{MIRROR}/setpoint/set",
                    channel=mqtt.ChannelField(
                        pointer="", json_type="number", channel=set_ch.key
                    ),
                )
            ],
        )
        self.client.tasks.configure(writer_task)
        setpoint = self.client.channels.retrieve("mqtt_edge_node_setpoint")
        setpoint_cmd = self.client.channels.retrieve("mqtt_edge_node_setpoint_cmd")
        try:
            with self.tsk.run(), writer_task.run():
                # The mirror learns the tag from the birth. A value on the tag proves
                # that the birth arrived before the command goes out.
                with self.client.open_writer(
                    start=sy.TimeStamp.now(),
                    channels=[setpoint.index, setpoint.key],
                ) as writer:
                    writer.write(
                        {setpoint.index: sy.TimeStamp.now(), setpoint.key: 1.0}
                    )
                with self.client.open_streamer([setpoint_cmd.key]) as streamer:
                    with self.client.open_writer(
                        start=sy.TimeStamp.now(), channels=[set_idx.key, set_ch.key]
                    ) as writer:
                        value = None
                        timer = sy.Timer()
                        while (
                            value != 42.5 and timer.elapsed() < 20 * sy.TimeSpan.SECOND
                        ):
                            writer.write(
                                {set_idx.key: sy.TimeStamp.now(), set_ch.key: 42.5}
                            )
                            frame = streamer.read(timeout=1)
                            if frame is not None and setpoint_cmd.key in frame:
                                value = float(frame[setpoint_cmd.key][-1])
            if value != 42.5:
                raise AssertionError(f"The command channel got {value}, not 42.5")
        finally:
            self.client.tasks.delete(writer_task.key)
