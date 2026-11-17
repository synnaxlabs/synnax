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
from tests.driver.mqtt_task import MQTTReadTaskCase, create_plain_entry
from tests.driver.task import create_channel, create_index
from x import payload


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
            create_plain_entry(
                client,
                topic="plant/sensors",
                bindings=[
                    (
                        sy.icd.FloatJSONField(
                            name="temperature", pointer="/temperature"
                        ),
                        temperature,
                    ),
                    (sy.icd.IntegerJSONField(name="count", pointer="/count"), count),
                    (
                        sy.icd.UnsignedJSONField(
                            name="state",
                            pointer="/state",
                            enum_values=[
                                payload.UnsignedEntry(
                                    label="idle", value=payload.Unsigned(0)
                                ),
                                payload.UnsignedEntry(
                                    label="running", value=payload.Unsigned(1)
                                ),
                            ],
                        ),
                        running,
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
        stamp = sy.icd.TimestampJSONField(
            name="timestamp", pointer="/timestamp", time_format="unix_ns"
        )
        return [
            create_plain_entry(
                client,
                topic="plant/scalar",
                qos="at_least_once",
                bindings=[(sy.icd.FloatJSONField(name="value", pointer=""), scalar)],
            ),
            create_plain_entry(
                client,
                topic="plant/timed",
                index=stamp,
                bindings=[
                    (stamp, timed_idx.key),
                    (sy.icd.FloatJSONField(name="value", pointer="/value"), timed),
                ],
            ),
        ]


def _sparkplug_entry(
    client: sy.Synnax, tag: str, *, data_type: sy.DataType, device: str = ""
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
        data_type=data_type,
    )


class MQTTReadSparkplug(MQTTReadTaskCase):
    """Reads Sparkplug B tags of an edge node and of one of its devices. The task must
    ask the edge node for a rebirth, because the birth came before the task started."""

    task_name = "MQTT Read Sparkplug"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[mqtt.ReadEntry]:
        return [
            _sparkplug_entry(client, "temperature", data_type=sy.DataType.FLOAT64),
            _sparkplug_entry(client, "count", data_type=sy.DataType.INT64),
            _sparkplug_entry(client, "running", data_type=sy.DataType.UINT8),
            _sparkplug_entry(
                client, "speed", data_type=sy.DataType.FLOAT32, device=SPARKPLUG_DEVICE
            ),
        ]
