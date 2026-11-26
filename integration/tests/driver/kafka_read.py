#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax import kafka
from tests.driver.kafka_task import KafkaReadTaskCase, create_topic
from tests.driver.task import create_channel, create_index
from x import payload


class KafkaReadJSON(KafkaReadTaskCase):
    """Reads numbers and an enum label from the JSON value of each record, with the
    timestamp of the record."""

    task_name = "Kafka Read JSON"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.ReadTopic]:
        idx = create_index(client, "kafka_sensors_time")
        temperature = create_channel(
            client,
            name="kafka_sensors_temperature",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        count = create_channel(
            client,
            name="kafka_sensors_count",
            data_type=sy.DataType.INT64,
            index=idx.key,
        )
        running = create_channel(
            client,
            name="kafka_sensors_running",
            data_type=sy.DataType.UINT8,
            index=idx.key,
        )
        return [
            create_topic(
                client,
                topic="plant.sensors",
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
