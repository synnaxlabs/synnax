#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import synnax as sy
import argparse

# Set up command-line argument parsing
parser = argparse.ArgumentParser(
    description='Write data to Synnax channels at a specified rate.')
parser.add_argument('--rate', type=float, default=10.0,
                    help='Write rate in Hz (default: 10.0 Hz)')
args = parser.parse_args()

client = sy.Synnax()

# Create timestamp channel
timestamp_channel = client.channels.create(
    name="timestamp_channel",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)


# Function to create multiple similar channels
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
constant_channels = create_multiple_channels("constant_value_channel", 2,
                                             timestamp_channel.key)
sine_channels = create_multiple_channels("sine_wave_channel", 2, timestamp_channel.key)
cosine_channel = client.channels.create(
    name="cosine_wave_channel",
    index=timestamp_channel.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
linear_channels = create_multiple_channels("linear_function_channel", 3,
                                           timestamp_channel.key)

# Create channels for complex calculations
sine_50_channels = create_multiple_channels("sine_50_channel", 50,
                                            timestamp_channel.key)
linear_500_channels = create_multiple_channels("linear_500_channel", 500,
                                               timestamp_channel.key)

# Basic calculated channels using numpy functions
calc_np_add_channel = client.channels.create(
    name="calc_np_add_2ch",
    data_type=sy.DataType.FLOAT32,
    expression="result=np.add(constant_value_channel_1, sine_wave_channel_1)",
    requires=[constant_channels[0].key, sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_np_subtract_channel = client.channels.create(
    name="calc_np_subtract_2ch",
    data_type=sy.DataType.FLOAT32,
    expression="result=np.subtract(constant_value_channel_2, linear_function_channel_1)",
    requires=[constant_channels[1].key, linear_channels[0].key],
    retrieve_if_name_exists=True,
)

# Average of 50 sine channels using explicit sum
sine_50_expr = "result=(" + " + ".join(
    [f"sine_50_channel_{i + 1}" for i in range(50)]) + ") / 50"
calc_explicit_avg_50_sine = client.channels.create(
    name="calc_explicit_avg_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=sine_50_expr,
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

# Average of 50 sine channels using np.mean and np.array
sine_50_np_mean_expr = "result=np.array([np.mean([" + ", ".join(
    [f"sine_50_channel_{i + 1}" for i in range(50)]) + "])])"
calc_np_mean_50_sine = client.channels.create(
    name="calc_sum_div_50_sine",  # renamed to better reflect the operation
    data_type=sy.DataType.FLOAT32,
    expression=sine_50_np_mean_expr,
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

# Log calculations using np.log
calc_np_log_channel = client.channels.create(
    name="calc_np_log_2ch",
    data_type=sy.DataType.FLOAT32,
    expression="result=np.log(np.abs(sine_wave_channel_1) + 1)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

# Using np.log10
calc_np_log10_channel = client.channels.create(
    name="calc_np_log10_2ch",
    data_type=sy.DataType.FLOAT32,
    expression="result=np.log10(np.abs(sine_wave_channel_2) + 1)",
    requires=[sine_channels[1].key],
    retrieve_if_name_exists=True,
)

# Using np.multiply for element-wise multiplication
calc_np_multiply_channel = client.channels.create(
    name="calc_np_multiply_3ch",
    data_type=sy.DataType.FLOAT32,
    expression="result=np.multiply(np.multiply(sine_wave_channel_1, cosine_wave_channel), constant_value_channel_1)",
    requires=[sine_channels[0].key, cosine_channel.key, constant_channels[0].key],
    retrieve_if_name_exists=True,
)

# Using np.divide with safe division
calc_np_divide_channel = client.channels.create(
    name="calc_np_divide_2ch",
    data_type=sy.DataType.FLOAT32,
    expression="result=np.divide(linear_function_channel_2, np.add(linear_function_channel_1, 1))",
    requires=[linear_channels[1].key, linear_channels[0].key],
    retrieve_if_name_exists=True,
)

# Using np.power
calc_np_power_channel = client.channels.create(
    name="calc_np_power_2ch",
    data_type=sy.DataType.FLOAT32,
    expression="result=np.power(np.abs(sine_wave_channel_1), 2)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

# Complex multi-stage calculation using various np functions
calc_np_complex_chain = client.channels.create(
    name="calc_np_complex_chain_4ch",
    data_type=sy.DataType.FLOAT32,
    expression="result=np.divide(np.multiply(calc_np_mean_50_sine, calc_explicit_avg_50_sine), np.add(calc_np_add_2ch, calc_np_subtract_2ch))",
    requires=[calc_np_mean_50_sine.key, calc_explicit_avg_50_sine.key,
              calc_np_add_channel.key, calc_np_subtract_channel.key],
    retrieve_if_name_exists=True,
)

# RMS calculation using np.sqrt and np.mean wrapped in np.array
rms_50_expr = "result=np.array([np.sqrt(np.mean([" + ", ".join(
    [f"np.power(sine_50_channel_{i + 1}, 2)" for i in range(50)]) + "]))])"
calc_rms_50_sine = client.channels.create(
    name="calc_explicit_rms_50_sine",  # renamed to better reflect the operation
    data_type=sy.DataType.FLOAT32,
    expression=rms_50_expr,
    requires=[ch.key for ch in sine_50_channels[:50]],
    retrieve_if_name_exists=True,
)

# Set up constants
WRAP_THRESHOLD = 100.0
WRAP_VALUE = 0.0

# Set up the loop with configurable rate
loop = sy.Loop(sy.Rate.HZ * args.rate)

# Collect all channels for writing
all_channels = (
    [timestamp_channel]
    + constant_channels
    + sine_channels
    + [cosine_channel]
    + linear_channels
    + sine_50_channels
    + linear_500_channels
    + [calc_np_add_channel, calc_np_subtract_channel,
       calc_explicit_avg_50_sine, calc_np_mean_50_sine,
       calc_np_log_channel, calc_np_log10_channel,
       calc_np_multiply_channel, calc_np_divide_channel,
       calc_np_power_channel, calc_np_complex_chain,
       calc_rms_50_sine]
)

# Open writer with all channels
with client.open_writer(
    sy.TimeStamp.now(),
    channels=[ch.key for ch in all_channels],
    enable_auto_commit=True,
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
