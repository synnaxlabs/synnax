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
from tests.driver.mqtt_task import CORE_RACK_NAME, run_reader, write_until
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import WriteTaskCase, create_channel, create_index

GROUP = "Plant"
EDGE_NODE = "SynnaxLine"
MIRROR = f"plant/sparkplug/{GROUP}/{EDGE_NODE}"


class MQTTEdgeNode(SimulatorCase, WriteTaskCase[mqtt.EdgeTask]):
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
                    name="temperature", channel=temperature, tag_type="double"
                ),
                mqtt.EdgeTag(
                    name="setpoint",
                    channel=setpoint,
                    tag_type="double",
                    command_channel=setpoint_cmd,
                ),
            ],
        )

    def _channel_keys(self, task: mqtt.EdgeTask) -> list[int]:
        return [
            k
            for t in task.config.tags
            for k in (t.channel, t.command_channel)
            if k != 0
        ]

    def run(self) -> None:
        super().run()
        self.test_tag_reaches_host()
        self.test_command_reaches_channel()

    def test_tag_reaches_host(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Tag value reaches the host")
        device = self.client.devices.retrieve(name=self.device_name)
        temperature = self.client.channels.retrieve("mqtt_edge_node_temperature")
        with (
            run_reader(
                self.client,
                device=device.key,
                name=f"{self.task_name} temperature Mirror",
                channel="mqtt_edge_node_temperature_mirror",
                data_type=sy.DataType.FLOAT64,
                entry=lambda mirror: mqtt.PlainReadEntry(
                    topic=f"{MIRROR}/temperature",
                    fields=[
                        mqtt.ReadField(
                            pointer="/value",
                            channel=mirror.key,
                            data_type=sy.DataType.FLOAT64,
                        )
                    ],
                ),
            ) as mirror,
            self.tsk.run(),
        ):
            write_until(self.client, channel=temperature, value=21.5, watched=mirror)

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
        setpoint_cmd = self.client.channels.retrieve("mqtt_edge_node_setpoint_cmd")
        try:
            with self.tsk.run(), writer_task.run():
                # The mirror drops a command until the birth declares the tag.
                write_until(
                    self.client, channel=set_ch, value=42.5, watched=setpoint_cmd
                )
        finally:
            self.client.tasks.delete(writer_task.key)
            self.client.channels.delete([set_ch.key, set_idx.key])
