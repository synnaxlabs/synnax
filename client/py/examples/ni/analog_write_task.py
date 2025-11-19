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
This example demonstrates how to configure and start an Analog Write Task on a National
Instruments USB-6289 device.

To run this example, you'll need to have your Synnax cluster properly configured to
detect National Instruments devices: https://docs.synnaxlabs.com/reference/driver/ni/get-started

You'll also need to have either a physical USB-6289 device or create a simulated device
via the NI-MAX software.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

dev = client.hardware.devices.retrieve(model="USB-6289")

# Create virtual command channels that will be used to send commands to the device.
# These are virtual channels that won't store data to disk.
ao_0_cmd = client.channels.create(
    name="ao_0_cmd",
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
    virtual=True,
)
ao_1_cmd = client.channels.create(
    name="ao_1_cmd",
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
    virtual=True,
)

# Create an index channel that will be used to store the timestamps
# for the analog output state data.
ao_state_time = client.channels.create(
    name="ao_state_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create channels that will store the state of the analog outputs
# after they have been commanded.
ao_0_state = client.channels.create(
    name="ao_0_state",
    # Pass in the index key here to associate the channel with the index channel.
    index=ao_state_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
ao_1_state = client.channels.create(
    name="ao_1_state",
    # Pass in the index key here to associate the channel with the index channel.
    index=ao_state_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Instantiate the task. A task is a background process that can be used to acquire data
# from, or, in this case, write commands to a device. Tasks are the primary method for
# interacting with Synnax hardware devices.
tsk = ni.AnalogWriteTask(
    # A name to find and monitor the task via the Synnax Console.
    name="Basic Analog Write",
    # The key of the device to execute the task on.
    device=dev.key,
    # The rate at which the task will sample the current state of the analog
    # outputs on the device.
    state_rate=sy.Rate.HZ * 10,
    # Whether to save the states of the analog outputs to disk. If set to False,
    # the data will be streamed into Synnax for real-time consumption but not saved
    # to disk.
    data_saving=True,
    # The mapping of the analog output channels on the device to the Synnax channels.
    channels=[
        ni.AOVoltageChan(
            # The cmd channel will be used to send commands to the device.
            cmd_channel=ao_0_cmd.key,
            # The state channel will be used to store the state of the analog output
            # after it has been commanded.
            state_channel=ao_0_state.key,
            # The port on the device that the analog output is connected to.
            port=0,
            # The minimum and maximum voltage values for the output.
            min_val=-10.0,
            max_val=10.0,
        ),
        ni.AOVoltageChan(
            cmd_channel=ao_1_cmd.key,
            state_channel=ao_1_state.key,
            port=1,
            min_val=-10.0,
            max_val=10.0,
        ),
    ],
)

# Create the task in Synnax and wait for the driver to validate that the configuration
# is correct.
client.hardware.tasks.configure(tsk)

# Start the task and write some analog values.
with tsk.run():
    with client.control.acquire(
        name="Analog Control Sequence",
        read=["ao_0_state", "ao_1_state"],
        write=["ao_0_cmd", "ao_1_cmd"],
        write_authorities=50,
    ) as ctrl:
        ctrl["ao_0_cmd"] = 5.0
        ctrl["ao_1_cmd"] = 3.0
        ctrl.wait_until(
            lambda c: abs(c["ao_0_state"] - 5.0) < 0.01
            and abs(c["ao_1_state"] - 3.0) < 0.01,
            timeout=1,
        )
        ctrl.sleep(0.5)

        ctrl["ao_0_cmd"] = -2.5
        ctrl["ao_1_cmd"] = 7.5
        ctrl.wait_until(
            lambda c: abs(c["ao_0_state"] - (-2.5)) < 0.01
            and abs(c["ao_1_state"] - 7.5) < 0.01,
            timeout=1,
        )
        ctrl.sleep(0.5)

        ctrl["ao_0_cmd"] = 0.0
        ctrl["ao_1_cmd"] = 0.0
        ctrl.wait_until(
            lambda c: abs(c["ao_0_state"]) < 0.01 and abs(c["ao_1_state"]) < 0.01,
            timeout=1,
        )

client.hardware.tasks.delete(tsk.key)
