#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

from examples.mqtt_sim.server import (
    SPARKPLUG_DEVICE,
    SPARKPLUG_EDGE_NODE,
    SPARKPLUG_GROUP,
)

import synnax as sy
from synnax import mqtt
from tests.driver.mqtt_task import MQTTWriteTaskCase
from tests.driver.task import create_channel, create_index


class MQTTWriteValve(MQTTWriteTaskCase):
    """Publishes commands, then reads the echo of the simulator to prove that each
    command reached the broker with the right payload."""

    task_name = "MQTT Write Valve"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[mqtt.WriteTarget]:
        idx = create_index(client, "mqtt_valve_cmd_time")
        cmd = create_channel(
            client,
            name="mqtt_valve_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        return [
            mqtt.PlainWriteTarget(
                topic="plant/valve/set",
                channel=mqtt.ChannelField(
                    pointer="/position",
                    json_type="number",
                    channel=cmd,
                    data_type="float64",
                ),
                fields=[
                    mqtt.StaticWriteField(
                        pointer="/source", json_type="string", value="synnax"
                    ),
                    mqtt.GeneratedWriteField(pointer="/id", generator="uuid"),
                ],
            ),
        ]

    def run(self) -> None:
        super().run()
        self.test_command_reaches_broker()

    def test_command_reaches_broker(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Command payload reaches the broker")
        device = self.client.devices.retrieve(name=self.device_name)
        echo = self.client.channels.create(
            name="mqtt_valve_state_payload",
            data_type=sy.DataType.STRING,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        reader = mqtt.ReadTask(
            name=f"{self.task_name} Echo Reader",
            device=device.key,
            entries=[
                mqtt.PlainReadEntry(
                    topic="plant/valve/state",
                    retained_ignored=True,
                    fields=[
                        mqtt.ReadField(pointer="", channel=echo.key, data_type="string")
                    ],
                )
            ],
        )
        self.client.tasks.configure(reader)
        cmd_key = self._channel_keys(self.tsk)[0]
        cmd = self.client.channels.retrieve(cmd_key)
        try:
            with reader.run(), self.tsk.run():
                with self.client.open_streamer([echo.key]) as streamer:
                    with self.client.open_writer(
                        start=sy.TimeStamp.now(), channels=[cmd.index, cmd_key]
                    ) as writer:
                        payload = None
                        timer = sy.Timer()
                        while (
                            payload is None
                            and timer.elapsed() < 20 * sy.TimeSpan.SECOND
                        ):
                            writer.write({cmd.index: sy.TimeStamp.now(), cmd_key: 42.5})
                            frame = streamer.read(timeout=1)
                            if frame is not None and echo.key in frame:
                                payload = json.loads(frame[echo.key][-1])
            if payload is None:
                raise AssertionError("The simulator did not echo the command")
            if payload["position"] != 42.5 or payload["source"] != "synnax":
                raise AssertionError(f"Unexpected command payload: {payload}")
            if len(payload["id"]) != 36:
                raise AssertionError(f"Payload has no generated UUID: {payload}")
        finally:
            self.client.tasks.delete(reader.key)
            self.client.channels.delete([echo.key])


class MQTTWriteSparkplug(MQTTWriteTaskCase):
    """Sends Sparkplug B commands for a tag of a device, then reads the tag back to
    prove that the edge node took each command."""

    task_name = "MQTT Write Sparkplug"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[mqtt.WriteTarget]:
        idx = create_index(client, "mqtt_sparkplug_setpoint_cmd_time")
        cmd = create_channel(
            client,
            name="mqtt_sparkplug_setpoint_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        return [
            mqtt.SparkplugWriteTarget(
                group=SPARKPLUG_GROUP,
                edge_node=SPARKPLUG_EDGE_NODE,
                device=SPARKPLUG_DEVICE,
                tag="setpoint",
                channel=cmd,
                sparkplug_type="double",
            ),
        ]

    def run(self) -> None:
        super().run()
        self.test_command_reaches_edge_node()

    def test_command_reaches_edge_node(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Command reaches the edge node")
        device = self.client.devices.retrieve(name=self.device_name)
        state = self.client.channels.create(
            name="mqtt_sparkplug_setpoint_state",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        reader = mqtt.ReadTask(
            name=f"{self.task_name} State Reader",
            device=device.key,
            entries=[
                mqtt.SparkplugReadEntry(
                    group=SPARKPLUG_GROUP,
                    edge_node=SPARKPLUG_EDGE_NODE,
                    device=SPARKPLUG_DEVICE,
                    tag="setpoint",
                    channel=state.key,
                    data_type=sy.DataType.FLOAT64,
                )
            ],
        )
        self.client.tasks.configure(reader)
        cmd_key = self._channel_keys(self.tsk)[0]
        cmd = self.client.channels.retrieve(cmd_key)
        try:
            with reader.run(), self.tsk.run():
                with self.client.open_streamer([state.key]) as streamer:
                    with self.client.open_writer(
                        start=sy.TimeStamp.now(), channels=[cmd.index, cmd_key]
                    ) as writer:
                        value = None
                        timer = sy.Timer()
                        while (
                            value != 42.5 and timer.elapsed() < 20 * sy.TimeSpan.SECOND
                        ):
                            writer.write({cmd.index: sy.TimeStamp.now(), cmd_key: 42.5})
                            frame = streamer.read(timeout=1)
                            if frame is not None and state.key in frame:
                                value = float(frame[state.key][-1])
            if value != 42.5:
                raise AssertionError(f"The setpoint tag is {value}, not 42.5")
        finally:
            self.client.tasks.delete(reader.key)
            self.client.channels.delete([state.key])
