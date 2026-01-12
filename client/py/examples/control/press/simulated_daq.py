#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example sets up a simulated data acquisition system (DAQ) that has a press valve
(press_vlv_cmd) and vent valve (vent_vlv_cmd) connected to a pressure transducer
(press_pt). When the press valve is open, the pressure increases, and when the vent
valve is open, the pressure decreases. The pressure is written to the press_pt channel.

This script can be run in conjuction with the `control_sequence.py` script to
demonstrate how a control sequence can be written in Synnax.
"""

import random

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

daq_time_ch = client.channels.create(
    name="daq_time", is_index=True, retrieve_if_name_exists=True
)

press_pt = client.channels.create(
    name="press_pt",
    index=daq_time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

press_vlv_cmd_time = client.channels.create(
    name="press_vlv_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

press_vlv_cmd = client.channels.create(
    name="press_vlv_cmd",
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
    index=press_vlv_cmd_time.key,
)

press_vlv_state = client.channels.create(
    name="press_vlv_state",
    index=daq_time_ch.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

vent_vlv_cmd_time = client.channels.create(
    name="vent_vlv_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

vent_vlv_cmd = client.channels.create(
    name="vent_vlv_cmd",
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
    index=vent_vlv_cmd_time.key,
)

vent_vlv_state = client.channels.create(
    name="vent_vlv_state",
    index=daq_time_ch.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

start_seq_cmd = client.channels.create(
    name="start_seq_cmd",
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
    virtual=True,
)

loop = sy.Loop(sy.Rate.HZ * 3)

state = {
    "daq_time": sy.TimeStamp.now(),
    "press_vlv_state": 0,
    "vent_vlv_state": 0,
    "press_pt": 0,
}

# The first thing we do is open a streamer to read commands issued to the valves, either
# from the Synnax console or from a control sequence. This ensures that we can
# adequately respond to commands by setting states of valves and increasing/ decreasing
# pressure.
with client.open_streamer(["press_vlv_cmd", "vent_vlv_cmd"]) as streamer:
    # We open a writer to write the state of the system back to Synnax, so it is
    # permanently stored and visualized by the Synnax console and control sequences.
    with client.open_writer(
        # Start is a timestamp at or just before the first sample we write. In this
        # case, we're writing live data, so we start at the current time.
        start=sy.TimeStamp.now(),
        # We write the time, the state of the valves, and the pressure.
        channels=[
            daq_time_ch.key,
            "press_vlv_state",
            "vent_vlv_state",
            "press_pt",
        ],
        # A useful name that identifies the sequence to the rest of the system. We
        # highly recommend keeping these names unique across your sequences.
        name="Simulated DAQ",
    ) as writer:
        # Enter the main loop, which will run at 40Hz as set by the loop rate.
        while loop.wait():
            # Read the latest commands issued to the valves with a timeout of 0, which
            # will return immediately if no commands were sent. The frame data structure
            # returned is essentially a dictionary of channel names and their latest
            # values.
            frame = streamer.read(timeout=0)
            # A non-empty frame indicates that a command was issued.
            if frame is not None:
                # Check if there are 1 or more commands issued to the vent valve. If so,
                # set the vent valve state to the latest command.
                vent_vlv_cmd = frame.get("vent_vlv_cmd")
                if len(vent_vlv_cmd) > 0:
                    state["vent_vlv_state"] = vent_vlv_cmd[-1]

                # Check if there are 1 or more commands issued to the press valve. If
                # so, set the press valve state to the latest command.
                press_vlv_cmd = frame.get("press_vlv_cmd")
                if len(press_vlv_cmd) > 0:
                    state["press_vlv_state"] = press_vlv_cmd[-1]

            # If the press valve is open, increase the pressure.
            if state["press_vlv_state"] == 1:
                state["press_pt"] += 0.1

            # If the vent valve is open, decrease the pressure.
            elif state["vent_vlv_state"] == 1:
                state["press_pt"] -= 0.1

            # If the pressure is less than 0, set it to 0.
            if state["press_pt"] < 0:
                state["press_pt"] = 0

            # inject a bit of random floating point noise into the pressure.
            state["press_pt"] += random.uniform(-0.1, 0.1)

            # Update the timestamp of the state to represent the current time.
            state["daq_time"] = sy.TimeStamp.now()

            # Write the system state to Synnax.
            writer.write(state)
