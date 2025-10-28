#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax.hardware import ni

"""
This example demonstrates how to configure and start a Counter Write Task on a National
Instruments USB-6289 device to generate pulse outputs.

To run this example, you'll need to have your Synnax cluster properly configured to
detect National Instruments devices: https://docs.synnaxlabs.com/reference/driver/ni/get-started

You'll also need to have either a Counter Output module or create a simulated device
via the NI-MAX software.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# Retrieve the USB-6289 device from Synnax.
dev = client.hardware.devices.retrieve(model="NI 9474")

# Create a channel that will be used to send pulse commands to the device. We're using
# a virtual channel here that won't store any data to disk. Don't worry, we're
# still going to get data on the state of the counter output.
co_0_cmd = client.channels.create(
    name="co_0_cmd",
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
    virtual=True,
)

# Create a channel that will store timestamps for the state of the counter output.
co_state_time = client.channels.create(
    name="co_state_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create a channel that will store the state of the counter output.
co_0_state = client.channels.create(
    name="co_0_state",
    # Pass in the index key here to associate the channel with the index channel.
    index=co_state_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Instantiate the task. A task is a background process that can be used to acquire data
# from, or, in this case, write pulse commands to a device. Tasks are the primary method
# for interacting with Synnax hardware devices.
tsk = ni.CounterWriteTask(
    # A name to find and monitor the task via the Synnax Console.
    name="Basic Counter Write",
    # The key of the device to execute the task on.
    device=dev.key,
    # The rate at which the task will sample the current state of the counter
    # outputs on the device.
    state_rate=sy.Rate.HZ * 1000,
    # Whether to save the states of the counter outputs to disk. If set to False,
    # the data will be streamed into Synnax for real-time consumption but not saved
    # to disk.
    data_saving=True,
    # The mapping of the counter output channels on the device to the Synnax channels.
    channels=[
        ni.COChan(
            # The cmd channel will be used to send pulse commands to the device.
            cmd_channel=co_0_cmd.key,
            # The state channel will be used to store the state of the counter output
            # after it has been commanded.
            state_channel=co_0_state.key,
            # The counter port on the device (ctr0, ctr1, etc.)
            port=0,
            # Idle state of the counter output (High or Low)
            idle_state="Low",
            # Initial delay before pulse generation starts (in seconds)
            initial_delay=0.0,
            # Duration of the high portion of the pulse (in seconds)
            high_time=0.001,
            # Duration of the low portion of the pulse (in seconds)
            low_time=0.001,
            # Units for timing (Seconds)
            units="Seconds",
        ),
    ],
)

# Create the task in Synnax and wait for the driver to validate that the configuration
# is correct.
client.hardware.tasks.configure(tsk)

# Start the task and use a control sequence to command pulse generation.
with tsk.start():
    with client.control.acquire(
        name="Counter Control Sequence",
        read=["co_0_state"],
        write=["co_0_cmd"],
        write_authorities=50,
    ) as ctrl:
        # Start pulse generation
        ctrl["co_0_cmd"] = 1
        ctrl.wait_until(lambda c: c["co_0_state"] == 1, timeout=1)
        # Let it run for a bit
        ctrl.sleep(2)
        # Stop pulse generation
        ctrl["co_0_cmd"] = 0
        ctrl.wait_until(lambda c: c["co_0_state"] == 0, timeout=1)

# Clean up by deleting the task
client.hardware.tasks.delete(tsk.key)
