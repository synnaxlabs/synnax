#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to start and configure a read task on an OPC UA server.

Before running this example, you will need to connect Synnax to an OPC UA server. See
https://docs.synnaxlabs.com/reference/driver/opc-ua/connect-server.
"""

import synnax as sy
from synnax.hardware import opcua

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the OPC UA server from Synnax.
dev = client.hardware.devices.retrieve(name="OPC UA Server")

# Create an index channel that will be used to store the timestamps for the data.
opcua_time = client.channels.create(
    name="opcua_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create two Synnax channels that will be used to store the node data. Notice how these
# channels aren't specifically bound to the device. You'll do that in a later step when
# you create the Read Task.
node_0 = client.channels.create(
    name="node_0",
    index=opcua_time.key,
    data_type=sy.DataType.INT64,
    retrieve_if_name_exists=True,
)
node_1 = client.channels.create(
    name="node_1",
    index=opcua_time.key,
    data_type=sy.DataType.INT64,
    retrieve_if_name_exists=True,
)

tsk = opcua.ReadTask(
    name="Basic OPC UA Read Task",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 100,
    stream_rate=sy.Rate.HZ * 25,
    data_saving=True,
    channels=[
        # Bind the Synnax channels to the OPC UA node IDs.
        opcua.Channel(channel=node_0.key, node_id="NS=2;I=8"),
        opcua.Channel(channel=node_1.key, node_id="NS=2;I=10"),
    ],
)

# Note that our server is being sampled at 100 Hz, but we're only streaming at 25 Hz.
# This means that every frame of data will contain 100 / 25 = 4 samples per frame.

client.hardware.tasks.configure(tsk)

total_reads = 100

frame = sy.Frame()

# Start the task and read 100 frames from the OPC UA server, which will contain a total
# of 400 samples per channel (100 frames * 4 samples per frame).
with tsk.run(timeout=10):
    with client.open_streamer(["node_0", "node_1"]) as streamer:
        for i in range(total_reads):
            frame.append(streamer.read())

frame.to_df().to_csv("opcua_read_result.csv")
