#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example sets up a simulated data acquisition system (DAQ) that has a press valve
(ox_mpv_cmd) and vent valve (ox_pre_vlv_cmd) connected to a pressure transducer
(press_pt). When the press valve is open, the pressure increases, and when the vent
valve is open, the pressure decreases. The pressure is written to the press_pt channel.

This script can be run in conjuction with the `control_sequence.py` script to
demonstrate how a control sequence can be written in Synnax.
"""

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax(
    host="localhost",
    port=9093,
    username="synnax",
    password="seldon",
    secure=False
)

daq_time_ch = client.channels.create(
    name="daq_time", is_index=True, retrieve_if_name_exists=True
)

ox_pre_vlv_cmd = client.channels.create(
    name="ox_pre_vlv_cmd",
    data_type=sy.DataType.UINT8,
    virtual=True,
    retrieve_if_name_exists=True,
)

ox_mpv_cmd = client.channels.create(
    name="ox_mpv_cmd",
    data_type=sy.DataType.UINT8,
    virtual=True,
    retrieve_if_name_exists=True,
)

ox_mpv_state = client.channels.create(
    name="ox_mpv_state",
    index=daq_time_ch.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

ox_pre_vlv_state = client.channels.create(
    name="ox_pre_vlv_state",
    index=daq_time_ch.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

loop = sy.Loop(sy.Rate.HZ * 500)

state = {
    "daq_time": sy.TimeStamp.now(),
    "ox_pre_vlv_state": 0,
    "ox_mpv_state": 0,
}

# The first thing we do is open a streamer to read commands issued to the valves, either
# from the Synnax console or from a control sequence. This ensures that we can
# adequately respond to commands by setting states of valves and increasing/ decreasing
# pressure.
with client.open_streamer(["ox_mpv_cmd", "ox_pre_vlv_cmd"]) as streamer:
    # We open a writer to write the state of the system back to Synnax, so it is
    # permanently stored and visualized by the Synnax console and control sequences.
    with client.open_writer(
        # Start is a timestamp at or just before the first sample we write. In this
        # case, we're writing live data, so we start at the current time.
        start=sy.TimeStamp.now(),
        # We write the time, the state of the valves, and the pressure.
        channels=[
            daq_time_ch.key,
            ox_mpv_state.key,
            ox_pre_vlv_state.key,
        ],
        # A useful name that identifies the sequence to the rest of the system. We
        # highly recommend keeping these names unique across your sequences.
        name="Simulated DAQ",
        # We enable auto-commit, which means that the writer will automatically persist
        # all data written to Synnax as soon as it arrives.
        enable_auto_commit=True,
    ) as writer:
        # Enter the main loop, which will run at 40Hz as set by the loop rate.
        while loop.wait():
            frame = streamer.read(timeout=0)
            if frame is not None:
                if "ox_mpv_cmd" in frame:
                    state["ox_mpv_state"] = frame["ox_mpv_cmd"]
                if "ox_pre_vlv_cmd" in frame:
                    state["ox_pre_vlv_state"] = frame["ox_pre_vlv_cmd"]

            state["daq_time"] = sy.TimeStamp.now()
            writer.write(state)
