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
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

# Average using running sum divided by count
running_sum = " + ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
count = "50.0"
calc_avg_explicit_50_sine = client.channels.create(
    name="calc_avg_explicit_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return ({running_sum}) / {count}",
    requires=[ch.key for ch in sine_50_channels],
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
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

# Statistical Operations
calc_npmean_50_sine = client.channels.create(
    name="calc_npmean_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return np.array([np.mean(["
    + ", ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + "])])",
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

calc_npmedian = client.channels.create(
    name="calc_npmedian_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return np.array([np.median(["
    + ", ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + "])])",
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

calc_npstd = client.channels.create(
    name="calc_npstd_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return np.array([np.std(["
    + ", ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + "])])",
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

calc_npvar = client.channels.create(
    name="calc_npvar_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return np.array([np.var(["
    + ", ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + "])])",
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

# MinMax Operations
calc_npmin = client.channels.create(
    name="calc_npmin_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return np.array([np.min(["
    + ", ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + "])])",
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

calc_npmax = client.channels.create(
    name="calc_npmax_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return np.array([np.max(["
    + ", ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + "])])",
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

calc_npptp = client.channels.create(
    name="calc_npptp_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return np.array([np.ptp(["
    + ", ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + "])])",
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

# Element-wise Operations
calc_npsquare = client.channels.create(
    name="calc_npsquare_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.square(sine_wave_channel_1)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_npsqrt = client.channels.create(
    name="calc_npsqrt_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.sqrt(np.abs(sine_wave_channel_1))",
    # abs to handle negative values
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_npabs = client.channels.create(
    name="calc_npabs_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.abs(sine_wave_channel_1)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_npexp = client.channels.create(
    name="calc_npexp_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.exp(sine_wave_channel_1)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_nplog = client.channels.create(
    name="calc_nplog_abs_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.log(np.abs(sine_wave_channel_1) + 1)",
    # +1 to avoid log(0), abs for negative values
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

# Rounding Operations
calc_npround = client.channels.create(
    name="calc_npround_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.round(sine_wave_channel_1)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_npfloor = client.channels.create(
    name="calc_npfloor_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.floor(sine_wave_channel_1)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_npceil = client.channels.create(
    name="calc_npceil_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.ceil(sine_wave_channel_1)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

# Logical Operations
calc_npgreater = client.channels.create(
    name="calc_npgreater_sine1_vs_0",
    data_type=sy.DataType.FLOAT32,
    expression="return np.greater(sine_wave_channel_1, 0).astype(np.float32)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_npless = client.channels.create(
    name="calc_npless_sine1_vs_0",
    data_type=sy.DataType.FLOAT32,
    expression="return np.less(sine_wave_channel_1, 0).astype(np.float32)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

# Percentile Operations
calc_nppercentile_75 = client.channels.create(
    name="calc_nppercentile75_50_sine",
    data_type=sy.DataType.FLOAT32,
    expression=f"return np.array([np.percentile(["
    + ", ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
    + "], 75)])",
    requires=[ch.key for ch in sine_50_channels],
    retrieve_if_name_exists=True,
)

# Cumulative Operations
calc_npcumsum = client.channels.create(
    name="calc_npcumsum_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.cumsum(sine_wave_channel_1)",
    requires=[sine_channels[0].key],
    retrieve_if_name_exists=True,
)

calc_npcumprod = client.channels.create(
    name="calc_npcumprod_sine1",
    data_type=sy.DataType.FLOAT32,
    expression="return np.cumprod(sine_wave_channel_1)",
    requires=[sine_channels[0].key],
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
        calc_npmean_50_sine,
        calc_avg_sum_div_50_sine,
        calc_avg_explicit_50_sine,
        calc_avg_pairwise_50_sine,
        calc_npmedian,
        calc_npstd,
        calc_npvar,
        calc_npmin,
        calc_npmax,
        calc_npptp,
        calc_npsquare,
        calc_npsqrt,
        calc_npabs,
        calc_npexp,
        calc_nplog,
        calc_npround,
        calc_npfloor,
        calc_npceil,
        calc_npgreater,
        calc_npless,
        calc_nppercentile_75,
        calc_npcumsum,
        calc_npcumprod,
    ]
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
