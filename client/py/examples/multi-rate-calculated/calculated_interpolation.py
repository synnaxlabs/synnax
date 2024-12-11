#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

"""
This example demonstrates how to calculate the average of two sensor channels that are
being sampled at different rates, using numpy's interpolation function to correctly
align the timestamps of the two channels. This example is more complex than the
'calculate_simple.py' example, and requires more computational resources to run.

This example must be run in conjunction with the 'simulated_daq.py' file in this
directory. Run the 'simulated_daq.py' file first, and then run this file.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# We create a separate index channel to store the timestamps for the calculated values.
# These will store the same timestamps as the raw time channel, but will be used to
# index the calculated values.
derived_time_ch = client.channels.create(
    name="derived_time", is_index=True, retrieve_if_name_exists=True
)

# We'll store the average of "stream_write_example_data_1" and "stream_write_example_data_2"
# in this channel.
average_example_data_1 = client.channels.create(
    name="average_example_data_1",
    index=derived_time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

current_values = dict()

TO_READ = ["time_ch_1", "time_ch_2", "data_ch_1", "data_ch_2"]

import numpy as np


def interpolate(data_ch_1_time, data_ch_1, data_ch_2_time, data_ch_2):
    # Start off by converting the data to numpy arrays
    data_ch_1_time = np.array(data_ch_1_time)
    data_ch_1 = np.array(data_ch_1)
    data_ch_2_time = np.array(data_ch_2_time)
    data_ch_2 = np.array(data_ch_2)

    # Check whether any of the timestamps overlap. If not, we can't interpolate.
    start_time = max(data_ch_1_time[0], data_ch_2_time[0])
    end_time = min(data_ch_1_time[-1], data_ch_2_time[-1])
    if start_time > end_time:
        return np.array([]), np.array([]), np.array([])

    combined_timestamps = np.unique(np.concatenate((data_ch_1_time, data_ch_2_time)))
    # We only want to interpolate values that are within the range of both channels.
    avg_timestamps = combined_timestamps[
        (combined_timestamps >= start_time) & (combined_timestamps <= end_time)
    ]
    # Interpolate the values for each channel
    sensor1_values_interp = np.interp(avg_timestamps, data_ch_1_time, data_ch_1)
    sensor2_values_interp = np.interp(avg_timestamps, data_ch_2_time, data_ch_2)
    # Return the interpolated values and the timestamps
    return sensor1_values_interp, sensor2_values_interp, avg_timestamps


with client.open_writer(
    start=sy.TimeStamp.now(),
    channels=["derived_time", "average_example_data_1"],
    enable_auto_commit=True,
) as writer:
    with client.open_streamer(TO_READ) as s:
        for fr in s:
            time = fr["time_ch_1"][-1]
            for k, v in fr.items():
                current_values[k] = fr[k]
            # If we still don't have data yet from all four channels, skip and wait for
            # the next frame.
            if len(current_values.items()) < 4:
                continue
            # Interpolate the values for each channel, and get the timestamps for the average
            # channel.
            sensor_1, sensor_2, time = interpolate(
                current_values["time_ch_1"],
                current_values["data_ch_1"],
                current_values["time_ch_2"],
                current_values["data_ch_2"],
            )
            # This means we have no samples to write
            if len(sensor_1) == 0:
                continue
            # Calculate the average of the two sensors
            avg = (sensor_1 + sensor_2) / 2
            writer.write({"derived_time": time, "average_example_data_1": avg})
