#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates streaming data from a large number of channels into Synnax,
while also acknowledging commands sent to a large number of simulated valves. This
example essentially serves as a basic simulated data acquisition system (DAQ).
"""

import time

import numpy as np
import synnax as sy

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

NUM_VALVES = 50
NUM_SENSORS = 250

# Some lists to store our channels.
valve_commands = list()
valve_responses = list()

# Maps the keys of valve command channels to response channels.
command_to_res = {}

# Stores the timestamps for both the sensors and the valve responses.
sensor_idx = client.channels.create(
    name="Sensor Time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create the necessary channels for each valve.
for i in range(NUM_VALVES):
    # The index channel for the command is used to track the time at which the command
    # was sent.
    cmd_idx = client.channels.create(
        name=f"Valve Command Time {i}",
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
        retrieve_if_name_exists=True,
    )
    cmd_res = client.channels.create(
        [
            # The command channel is used to send a command to the valve.
            sy.Channel(
                name=f"Valve Command {i}",
                index=cmd_idx.key,
                data_type=sy.DataType.UINT8,
            ),
            # The response channel is used to acknowledge the command from our simulated
            # DAQ.
            sy.Channel(
                name=f"Valve Response {i}",
                index=sensor_idx.key,
                data_type=sy.DataType.UINT8,
            ),
        ],
        retrieve_if_name_exists=True,
    )
    cmd = cmd_res[0]
    res = cmd_res[1]
    valve_commands.append(cmd)
    valve_responses.append(res)
    command_to_res[cmd.key] = res

sensors = [
    sy.Channel(
        name=f"Sensor {i}",
        index=sensor_idx.key,
        data_type=sy.DataType.FLOAT32,
    )
    for i in range(NUM_SENSORS)
]

sensors = client.channels.create(sensors, retrieve_if_name_exists=True)

# Define the list of channels we'll write to i.e. the sensors and the valve responses as
# well as the sensor index.
write_to = [
    *[s.key for s in sensors],
    *[v.key for v in valve_responses],
    sensor_idx.key,
]

# Define the list of channels we'll read from i.e. the incoming valve commands.
read_from = [v.key for v in valve_commands]

# Define a crude rate at which we'll write data.
rate = (sy.Rate.HZ * 100).period.seconds

# Set up the initial state of the valves (closed).
sensor_states = {v.key: np.uint8(False) for v in valve_responses}

# Open a streamer to listen for incoming valve commands.
with client.open_streamer([a.key for a in valve_commands]) as streamer:
    i = 0
    # Open a writer to write data to Synnax.
    with client.open_writer(
        sy.TimeStamp.now(),
        write_to,
        enable_auto_commit=True,
    ) as writer:
        start = sy.TimeStamp.now()
        while True:
            time.sleep(rate)
            # If we've received a command, update the state of the corresponding valve.
            f = streamer.read(timeout=0)
            if f is not None:
                for k in f.channels:
                    # 1 is open, 0 is closed.
                    sensor_states[command_to_res[k].key] = np.uint8(f[k][-1] > 0.9)
            for j, s in enumerate(sensors):
                sensor_states[s.key] = np.float32(np.sin(i / 1000) + j / 100)
            sensor_states[sensor_idx.key] = sy.TimeStamp.now()
            writer.write(sensor_states)
            i += 1
