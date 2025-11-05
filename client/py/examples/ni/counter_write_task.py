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
Instruments device to generate pulse outputs.

IMPORTANT: Counter output channels are configuration-only. The pulse parameters
(idle_state, initial_delay, high_time, low_time) are fixed when the task is created
and cannot be changed at runtime. To change these parameters, you must stop the task,
reconfigure it, and start it again.

To run this example, you'll need to have your Synnax cluster properly configured to
detect National Instruments devices: https://docs.synnaxlabs.com/reference/driver/ni/get-started

You'll also need to have either a Counter Output module or create a simulated device
via the NI-MAX software.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# Retrieve the NI device from Synnax.
dev = client.hardware.devices.retrieve(model="NI 9474")

# Instantiate the task. A task is a background process that can be used to acquire data
# from, or, in this case, generate pulse outputs on a device. Tasks are the primary
# method for interacting with Synnax hardware devices.
tsk = ni.CounterWriteTask(
    # A name to find and monitor the task via the Synnax Console.
    name="Basic Counter Write",
    # The key of the device to execute the task on.
    device=dev.key,
    # The rate at which the task will update internal state (not used for counter write)
    state_rate=sy.Rate.HZ * 1,
    # Whether to save task state to disk. Counter write tasks typically don't need this.
    data_saving=False,
    # The counter output channel configuration
    channels=[
        ni.COChan(
            # The counter port on the device (ctr0, ctr1, etc.)
            port=0,
            # Idle state of the counter output when not generating pulses
            idle_state="Low",
            # Initial delay before pulse generation starts (in seconds)
            initial_delay=0.0,
            # Duration of the high portion of the pulse (in seconds)
            # This creates a 1kHz pulse (1ms high + 1ms low = 2ms period)
            high_time=0.001,
            # Duration of the low portion of the pulse (in seconds)
            low_time=0.001,
            # Units for timing parameters
            units="Seconds",
        ),
    ],
)

# Create the task in Synnax and wait for the driver to validate that the configuration
# is correct.
client.hardware.tasks.configure(tsk)

# Start the task to begin generating pulses. The pulses will continue with the
# configured parameters until the task is stopped.
print("Starting pulse generation...")
tsk.start()

# Let the pulses run for 5 seconds
sy.sleep(5)

# Stop pulse generation
print("Stopping pulse generation...")
tsk.stop()

# To change pulse parameters, you would need to:
# 1. Stop the task (done above)
# 2. Delete and recreate with new parameters, or reconfigure if supported
# 3. Start the task again

# Clean up by deleting the task
client.hardware.tasks.delete(tsk.key)
print("Task deleted.")
