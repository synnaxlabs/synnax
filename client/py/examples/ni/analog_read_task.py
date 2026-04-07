#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax import ni

"""
This example demonstrates how to configure and start a multi-device Analog Read Task
on National Instruments hardware. The task reads voltage from one device, current from
another, and temperature (thermocouple) from a third — all in a single task.

To run this example, you'll need to have your Synnax cluster properly configured to
detect National Instruments devices:
https://docs.synnaxlabs.com/reference/driver/ni/get-started

You'll also need physical NI devices or simulated devices via NI-MAX.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/client/quick-start for more information.
client = sy.Synnax()

# Retrieve devices — each module is a separate device in Synnax.
v_dev = client.devices.retrieve(name="Mod1_Voltage")
c_dev = client.devices.retrieve(name="Mod2_Current")
tc_dev = client.devices.retrieve(name="Mod1_TC")

# Create an index channel for timestamps.
ai_time = client.channels.create(
    name="ai_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create data channels — one per physical input.
voltage_chan = client.channels.create(
    name="voltage_chan",
    index=ai_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
current_chan = client.channels.create(
    name="current_chan",
    index=ai_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
temp_chan = client.channels.create(
    name="temp_chan",
    index=ai_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Create and configure the task. Each channel specifies its own device, allowing
# a single task to read from multiple NI modules simultaneously.
task = ni.AnalogReadTask(
    name="Analog Read Task",
    sample_rate=sy.Rate.HZ * 100,
    stream_rate=sy.Rate.HZ * 25,
    data_saving=True,
    channels=[
        ni.AIVoltageChan(
            channel=voltage_chan.key,
            device=v_dev.key,
            port=0,
            min_val=-10.0,
            max_val=10.0,
            terminal_config="Diff",
        ),
        ni.AICurrentChan(
            channel=current_chan.key,
            device=c_dev.key,
            port=0,
            min_val=0.004,
            max_val=0.02,
            shunt_resistor_loc="Internal",
            ext_shunt_resistor_val=249.0,
        ),
        ni.AIThermocoupleChan(
            channel=temp_chan.key,
            device=tc_dev.key,
            port=0,
            units="DegC",
            thermocouple_type="J",
            cjc_source="BuiltIn",
            cjc_val=None,
            cjc_port=None,
        ),
    ],
)

# This will create the task in Synnax and wait for the driver to validate that the
# configuration is correct.
client.tasks.configure(task)

# Stream 100 reads, which will accumulate a total of 400 samples
# for each channel over a period of 4 seconds.
total_reads = 100

frame = sy.Frame()

with task.run():
    with client.open_streamer(
        ["voltage_chan", "current_chan", "temp_chan"]
    ) as streamer:
        for i in range(total_reads):
            frame.append(streamer.read())

frame.to_df().to_csv("analog_read_result.csv")
