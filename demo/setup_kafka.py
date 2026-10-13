#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Connects a Core to the local Kafka broker and starts a read task that writes both
turbines' readings to channels. Run it once against a fresh Core, after the broker and
the producer are up.
"""

import argparse
import uuid

import synnax as sy
from synnax import kafka

FIELDS = ["rpm", "exhaust_temp", "vibration"]
UNITS = ["turbine_a", "turbine_b"]

parser = argparse.ArgumentParser()
parser.add_argument("--port", type=int, default=9090)
parser.add_argument("--broker", default="localhost:9092")
parser.add_argument("--topic", default="turbines")
args = parser.parse_args()

client = sy.Synnax(
    host="localhost", port=args.port, username="synnax", password="seldon"
)
rack = client.tasks.retrieve(type="kafka_scan").rack
dev = client.devices.create(
    key=str(uuid.uuid4()),
    name="Plant",
    make="kafka",
    model="cluster",
    rack=rack,
    location=args.broker,
    properties=kafka.Properties(brokers=[args.broker]).model_dump(
        by_alias=True, exclude_none=True
    ),
)

records = []
for unit in UNITS:
    index = client.channels.create(
        name=f"plant_{unit}_time", data_type="timestamp", is_index=True
    )
    stamp = kafka.ReadField(
        pointer="/time",
        data_type="timestamp",
        time_format="unix_ns",
        channel=index.key,
        name=index.name,
    )
    fields = [stamp]
    for field in FIELDS:
        ch = client.channels.create(
            name=f"plant_{unit}_{field}", data_type="float64", index=index.key
        )
        fields.append(
            kafka.ReadField(pointer=f"/{field}", data_type="float64", channel=ch.key)
        )
    records.append(kafka.ReadRecord(record_key=unit, index=stamp.key, fields=fields))

tsk = client.tasks.create(
    name="Turbines",
    type="kafka_read",
    config=kafka.ReadConfig(device=dev.key, topic=args.topic, records=records),
    rack=rack,
)
status = tsk.execute_command_sync("start", timeout=10)
print(f"{tsk.name}: {status.message}")
