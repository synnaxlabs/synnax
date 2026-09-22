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

import synnax as sy
from synnax import mqtt
from tests.driver.mqtt_task import MQTTReadTaskCase
from tests.driver.task import create_channel, create_index


class MQTTReadJSON(MQTTReadTaskCase):
    """Reads numbers and an enum label from a JSON payload, with the arrival time."""

    task_name = "MQTT Read JSON"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[mqtt.ReadEntry]:
        idx = create_index(client, "mqtt_sensors_time")
        temperature = create_channel(
            client,
            name="mqtt_sensors_temperature",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        count = create_channel(
            client,
            name="mqtt_sensors_count",
            data_type=sy.DataType.INT64,
            index=idx.key,
        )
        running = create_channel(
            client,
            name="mqtt_sensors_running",
            data_type=sy.DataType.UINT8,
            index=idx.key,
        )
        return [
            mqtt.PlainReadEntry(
                topic="plant/sensors",
                fields=[
                    mqtt.ReadField(
                        pointer="/temperature",
                        channel=temperature,
                        data_type="float64",
                    ),
                    mqtt.ReadField(pointer="/count", channel=count, data_type="int64"),
                    mqtt.ReadField(
                        pointer="/state",
                        channel=running,
                        data_type="uint8",
                        enum_values=[
                            mqtt.EnumEntry(label="idle", value=0),
                            mqtt.EnumEntry(label="running", value=1),
                        ],
                    ),
                ],
            ),
        ]


class MQTTReadMultipleTopics(MQTTReadTaskCase):
    """Reads two topics at once: a bare scalar at quality of service 1, and a payload
    that carries its own timestamp."""

    task_name = "MQTT Read Multiple Topics"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[mqtt.ReadEntry]:
        scalar_idx = create_index(client, "mqtt_scalar_time")
        scalar = create_channel(
            client,
            name="mqtt_scalar_value",
            data_type=sy.DataType.FLOAT32,
            index=scalar_idx.key,
        )
        timed_idx = create_index(client, "mqtt_timed_time")
        timed = create_channel(
            client,
            name="mqtt_timed_value",
            data_type=sy.DataType.FLOAT64,
            index=timed_idx.key,
        )
        stamp = mqtt.ReadField(
            pointer="/timestamp",
            channel=timed_idx.key,
            data_type="timestamp",
            time_format="unix_ns",
        )
        return [
            mqtt.PlainReadEntry(
                topic="plant/scalar",
                qos="at_least_once",
                fields=[
                    mqtt.ReadField(pointer="", channel=scalar, data_type="float32")
                ],
            ),
            mqtt.PlainReadEntry(
                topic="plant/timed",
                index=stamp.key,
                fields=[
                    stamp,
                    mqtt.ReadField(
                        pointer="/value", channel=timed, data_type="float64"
                    ),
                ],
            ),
        ]


def _sparkplug_entry(
    client: sy.Synnax, tag: str, data_type: sy.DataType, device: str = ""
) -> mqtt.SparkplugReadEntry:
    """Creates the channel of a tag of the simulator, on an index of its own."""
    idx = create_index(client, f"mqtt_sparkplug_{tag}_time")
    ch = create_channel(
        client,
        name=f"mqtt_sparkplug_{tag}",
        data_type=data_type,
        index=idx.key,
    )
    return mqtt.SparkplugReadEntry(
        group=SPARKPLUG_GROUP,
        edge_node=SPARKPLUG_EDGE_NODE,
        device=device,
        tag=tag,
        channel=ch,
        index=idx.key,
        data_type=data_type,
    )


class MQTTReadSparkplug(MQTTReadTaskCase):
    """Reads Sparkplug B tags of an edge node and of one of its devices. The task must
    ask the edge node for a rebirth, because the birth came before the task started."""

    task_name = "MQTT Read Sparkplug"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[mqtt.ReadEntry]:
        return [
            _sparkplug_entry(client, "temperature", sy.DataType.FLOAT64),
            _sparkplug_entry(client, "count", sy.DataType.INT64),
            _sparkplug_entry(client, "running", sy.DataType.UINT8),
            _sparkplug_entry(
                client, "speed", sy.DataType.FLOAT32, device=SPARKPLUG_DEVICE
            ),
        ]
