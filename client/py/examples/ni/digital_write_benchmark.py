#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy


# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# Retrieve the USB-6289 device from Synnax.
dev = client.devices.retrieve(model="USB-6289")

# Create a channel that will be used to send commands to the device. We're using
# a virtual channel here that won't store any data to disk. Don't worry, we're
# still going to get data on the state of the digital output.
do_1_cmd = client.channels.create(
    name="do_1_cmd",
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
    virtual=True,
)

# Create a channel that will store timestamps for the state of the digital output.
do_state_time = client.channels.create(
    name="do_state_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create a channel that will store the state of the digital output.
do_1_state = client.channels.create(
    name="do_1_state",
    # Pass in the index key here to associate the channel with the index channel.
    index=do_state_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Instantiate the task. A task is a background process that can be used to acquire data
# from, or, in this case, write commands to a device. Tasks are the primary method for
# interacting with Synnax hardware devices.
tsk = sy.ni.DigitalWriteTask(
    # A name to find and monitor the task via the Synnax Console.
    name="Basic Digital Write",
    # The key of the device to execute the task on.
    device=dev.key,
    # The rate at which the task will sample the current state of the digital
    # outputs on the device.
    state_rate=sy.Rate.HZ * 2000,
    # Whether to save the states of the digital outputs to disk. If set to False,
    # the data will be streamed into Synnax for real-time consumption but not saved
    # to disk.
    data_saving=True,
    # The mapping of the digital output channels on the device to the Synnax channels.
    channels=[
        sy.ni.DOChan(
            # The cmd channel will be used to send commands to the device.
            cmd_channel=do_1_cmd.key,
            # The state channel will be used to store the state of the digital output
            # after it has been commanded.
            state_channel=do_1_state.key,
            # The port and line on the device that the digital output is connected to.
            port=0,
            line=0,
        ),
    ],
)

# Create the task in Synnax and wait for the driver to validate that the configuration
# is correct.
client.tasks.configure(tsk)

durations = list()

# Start the task.
with tsk.run():
    with client.control.acquire(
        name="Control Sequence",
        read=["do_1_state"],
        write=["do_1_cmd"],
        write_authorities=50,
    ) as ctrl:
        for i in range(2500):
            ctrl["do_1_cmd"] = 1
            start = sy.TimeStamp.now()
            ctrl.wait_until(lambda c: c["do_1_state"] == 1)
            durations.append(sy.TimeSpan.since(start))
            ctrl["do_1_cmd"] = 0
            ctrl.wait_until(lambda c: c["do_1_state"] == 0)

client.tasks.delete(tsk.key)

import matplotlib.pyplot as plt
import numpy as np

amortized = np.array([dur.milliseconds for dur in durations]) / 2
# remove the outliers
amortized = amortized[amortized < 2]

# Get the P99 latency
p99 = np.percentile(amortized, 99)
print(f"P99 latency: {p99} ms")
p95 = np.percentile(amortized, 95)
print(f"P95 latency: {p95} ms")
p50 = np.percentile(amortized, 50)
print(f"P50 latency: {p50} ms")

# make the line skinnier and the plot wider
plt.plot(amortized, linewidth=0.5)
# plot the average for every 50 requests in red
plt.plot(np.convolve(amortized, np.ones(50) / 50, mode="valid"), color="red")


plt.xlabel("Request #")
plt.ylabel("Latency (ms)")
plt.savefig("latency.png")

jitter = np.std(amortized)

print(f"Jitter: {jitter} ms")
