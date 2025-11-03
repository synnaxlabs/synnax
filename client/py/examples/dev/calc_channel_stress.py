#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse

import numpy as np

import synnax as sy

parser = argparse.ArgumentParser(
    description="Write data to Synnax channels at a specified rate."
)
parser.add_argument(
    "--rate", type=float, default=10.0, help="Write rate in Hz (default: 10.0 Hz)"
)
args = parser.parse_args()

client = sy.Synnax()

timestamp_channel = client.channels.create(
    name="timestamp_channel",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)


def create_multiple_channels(base_name, count, index_key):
    return [
        client.channels.create(
            name=f"{base_name}_{i + 1}",
            index=index_key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )
        for i in range(count)
    ]


# Create base channels
constant_channels = create_multiple_channels(
    "constant_value_channel", 2, timestamp_channel.key
)
sine_channels = create_multiple_channels("sine_wave_channel", 2, timestamp_channel.key)
cosine_channel = client.channels.create(
    name="cosine_wave_channel",
    index=timestamp_channel.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
linear_channels = create_multiple_channels(
    "linear_function_channel", 3, timestamp_channel.key
)

# Create channels for complex calculations
sine_50_channels = create_multiple_channels(
    "sine_50_channel", 50, timestamp_channel.key
)
linear_500_channels = create_multiple_channels(
    "linear_500_channel", 500, timestamp_channel.key
)

# Different average calculations
calc_avg_sum_div_50_sine = client.channels.create(
    name="calc_avg_sum_div_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression="return ("
    + " + ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + ") / 50.0",
    retrieve_if_name_exists=True,
)

# Average using running sum divided by count
running_sum = " + ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
count = "50.0"
calc_avg_explicit_50_sine = client.channels.create(
    name="calc_avg_explicit_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return ({running_sum}) / {count}",
    retrieve_if_name_exists=True,
)

# Average using pairwise additions and division
pairs = [
    f"(sine_50_channel_{i * 2 + 1} + sine_50_channel_{i * 2 + 2})/2.0"
    for i in range(25)
]
pairwise_avg = "(" + " + ".join(pairs) + ") / 25.0"
calc_avg_pairwise_50_sine = client.channels.create(
    name="calc_avg_pairwise_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return {pairwise_avg}",
    retrieve_if_name_exists=True,
)

# Set up constants
WRAP_THRESHOLD = 100.0
WRAP_VALUE = 0.0

# Set up the loop with configurable rate
loop = sy.Loop(sy.Rate.HZ * args.rate)

# Collect all channels
all_channels = (
    [timestamp_channel]
    + constant_channels
    + sine_channels
    + [cosine_channel]
    + linear_channels
    + sine_50_channels
    + linear_500_channels
    + [
        calc_avg_sum_div_50_sine,
        calc_avg_explicit_50_sine,
        calc_avg_pairwise_50_sine,
    ]
)

# Open writer with all channels
with client.open_writer(
    sy.TimeStamp.now(),
    channels=[ch.key for ch in all_channels],
) as writer:
    i = 0
    while loop.wait():
        current_time = sy.TimeStamp.now()

        # Prepare data dictionary
        data_to_write = {timestamp_channel.key: current_time}

        # Constants
        for j, ch in enumerate(constant_channels):
            data_to_write[ch.key] = 42.0 + j * 58.0  # Different constants

        # Sine waves
        for j, ch in enumerate(sine_channels):
            data_to_write[ch.key] = (j + 5) * np.sin(i / 10.0)

        # Cosine wave
        data_to_write[cosine_channel.key] = np.cos(i / 10.0)

        # Linear functions
        for j, ch in enumerate(linear_channels):
            data_to_write[ch.key] = ((j + 0.5) * i + j * 5) % WRAP_THRESHOLD

        # 50 sine channels with different phases
        for j, ch in enumerate(sine_50_channels):
            data_to_write[ch.key] = 5 * np.sin(i / 10.0 + j * (2 * np.pi / 50))

        # 500 linear channels with different slopes
        for j, ch in enumerate(linear_500_channels):
            data_to_write[ch.key] = ((j + 1) * 0.1 * i + j) % WRAP_THRESHOLD

        # Write the data
        writer.write(data_to_write)
        i += 1
