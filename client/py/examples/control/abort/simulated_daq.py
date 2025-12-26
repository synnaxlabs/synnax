#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

"""
This is a simple simulated data acquisition computer (DAQ) that has two valves and a
single pressure sensor. When the press valve is open (press_vlv), the pressure
increases. When the vent valve is open (vent_vlv), the pressure decreases. The pressure
(sensor_0) is sampled at a rate of 50 Hz.

Valves (or any commanded actuator), typically have three associated channels:

1. The command channel - this is where commands are sent down to actuate the valve.
2. The command channel time - stores the timestamps for the command channel,
and 'indexes' the command channel.
3. The state channel - this is where the state of the valve is stored. The DAQ updates
this value when a command is executed. The state channel is indexed by the regular DAQ
timestamp channel.

If you want to add another valve to your simulation, follow the same pattern explained
above using the code below as reference.
"""

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# This will store the timestamps for the samples recorded by the simulated DAQ.
daq_time_ch = client.channels.create(
    name="daq_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# A pressure channel to store simulated pressure values.
pressure = client.channels.create(
    name="pressure",
    # This says that timestamps are stored in the channel 'daq_time'.
    index=daq_time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Stores the state of the press valve.
press_vlv_state = client.channels.create(
    name="press_vlv_state",
    # Again, notice that we're storing the timestamps in the 'daq_time' channel. This is
    # because the DAQ samples from the pressure, vent valve state, and press valve state
    # channels at the same time.
    index=daq_time_ch.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Stores the state of the vent valve.
vent_vlv_state = client.channels.create(
    name="vent_vlv_state",
    # Once again, we're storing the timestamps in the 'daq_time' channel.
    index=daq_time_ch.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# An independent time channel for the press valve command, because it doesn't
# necessarily get emitted in sync with any other channels.
press_vlv_cmd_time_ch = client.channels.create(
    name="press_vlv_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# An independent time channel for the vent valve command, because it doesn't necessarily
# get emitted in sync with any other channels.
vent_vlv_cmd_time_ch = client.channels.create(
    name="vent_vlv_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Channel for the vent valve command.
vent_vlv_cmd_ch = client.channels.create(
    name="vent_vlv_cmd",
    # This is the index channel for the vent valve command, completely independent of
    # the other channels.
    index=vent_vlv_cmd_time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Channel for the press valve command.
press_vlv_cmd_ch = client.channels.create(
    name="press_vlv_cmd",
    # This is the index channel for the press valve command, completely independent of
    # the other channels
    index=press_vlv_cmd_time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)


state = {
    "press_vlv_state": 0,
    "vent_vlv_state": 0,
    "pressure": 0,
    "daq_time": sy.TimeStamp.now(),
}

loop = sy.Loop(sy.Rate.HZ * 50)

with client.open_streamer(["press_vlv_cmd", "vent_vlv_cmd"]) as streamer:
    with client.open_writer(
        sy.TimeStamp.now(),
        channels=["daq_time", "pressure", "press_vlv_state", "vent_vlv_state"],
        name="Simulated DAQ",
    ) as writer:
        while loop.wait():
            while True:
                # Read incoming commands with a non-blocking timeout.
                frame = streamer.read(0)
                # Means we don't have any new data.
                if frame is None:
                    break
                # If the press valve has been commanded, update its state.
                if "press_vlv_cmd" in frame:
                    state["press_vlv_state"] = frame["press_vlv_cmd"][-1]
                # If the vent valve has been commanded, update its state.
                if "vent_vlv_cmd" in frame:
                    state["vent_vlv_state"] = frame["vent_vlv_cmd"][-1]

            state["daq_time"] = sy.TimeStamp.now()

            # If the press valve is open, increase the pressure.
            if state["press_vlv_state"] == 1:
                state["pressure"] += 0.1
            # If the vent valve is open, decrease the pressure.
            if state["vent_vlv_state"] == 1:
                state["pressure"] -= 0.1

            # Clamp the pressure to a minimum of 0, anything less would be physically
            # impossible.
            if state["pressure"] < 0:
                state["pressure"] = 0

            writer.write(state)
