#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math

import synnax as sy

client = sy.Synnax()

NUM_SENSOR_CHANNELS = range(0, 5)

daq_time_ch = client.channels.create(
    name="daq_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

sensor_channels = [
    sy.Channel(
        name=f"sensor_{i}",
        index=daq_time_ch.key,
        data_type=sy.DataType.FLOAT32,
    )
    for i in NUM_SENSOR_CHANNELS
]
sensor_channels = client.channels.create(sensor_channels, retrieve_if_name_exists=True)

NUM_VALVE_CHANNELS = range(0, 3)

valve_response_channels = client.channels.create(
    [
        sy.Channel(
            name=f"valve_response_{i}",
            index=daq_time_ch.key,
            data_type=sy.DataType.FLOAT32,
        )
        for i in NUM_VALVE_CHANNELS
    ],
    retrieve_if_name_exists=True,
)

valve_command_time_channels = [
    sy.Channel(
        name=f"valve_command_{i}_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
    )
    for i in NUM_VALVE_CHANNELS
]

valve_command_time_channels = client.channels.create(
    valve_command_time_channels, retrieve_if_name_exists=True
)

valve_command_channels = [
    sy.Channel(
        name=f"valve_command_{i}",
        index=valve_command_time_channels[i].key,
        data_type=sy.DataType.FLOAT32,
    )
    for i in NUM_VALVE_CHANNELS
]

valve_command_channels = client.channels.create(
    valve_command_channels,
    retrieve_if_name_exists=True,
)

loop = sy.Loop(sy.Rate.HZ * 40)

state = {
    **{ch.key: 0 for ch in valve_response_channels},
    **{ch.key: i for i, ch in enumerate(sensor_channels)},
}

valve_command_to_response_channels = {
    cmd.key: valve_response_channels[i].key
    for i, cmd in enumerate(valve_command_channels)
}

i = 0
with client.open_streamer([c.key for c in valve_command_channels]) as streamer:
    with client.open_writer(
        sy.TimeStamp.now(),
        channels=[
            daq_time_ch.key,
            *[c.key for c in sensor_channels],
            *[c.key for c in valve_response_channels],
        ],
        name="Simulated DAQ",
        enable_auto_commit=True,
    ) as writer:
        press = 0
        while loop.wait():
            while True:
                f = streamer.read(0)
                if f is None:
                    break
                for k in f.channels:
                    state[valve_command_to_response_channels[k]] = f[k][-1]

            now = sy.TimeStamp.now()
            state[daq_time_ch.key] = now

            for j, ch in enumerate(valve_response_channels):
                if state[ch.key] == 1:
                    if (j % 2) == 0:
                        state[sensor_channels[math.floor(j / 2)].key] += 1
                    else:
                        state[sensor_channels[math.floor(j / 2)].key] -= 1

            writer.write(state)

            i += 1
