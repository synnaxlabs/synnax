#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import signal
import sys

import synnax as sy
from synnax import sequence

"""
This example demonstrates how to configure and start a sequence on a Synnax cluster.

This script will create two sequences. The first sequence will generate a float32 sine
wave and a uint8 sawtooth wave. The second sequence will read both waves, and change the
inputs to a modulated waveform and a uint7 sawtooth wave.

A sequence is a task that is used to execute a script on a Synnax Driver. The script is
written in Lua and is used to control the behavior of the sequence. For more information
about sequences in general, visit
https://docs.synnaxlabs.com/reference/control/embedded.
"""

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Create the channels for the first sequence. Note that both channels will need
# different index channels, or an error will be thrown when trying to write to both of
# them.

primary_sequence_time_channels = client.channels.create(
    [
        sy.Channel(
            name="primary_sequence_float_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        ),
        sy.Channel(
            name="primary_sequence_int_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        ),
    ],
    retrieve_if_name_exists=True,
)

primary_sequence_data_channels = client.channels.create(
    [
        sy.Channel(
            name="primary_sequence_float_data",
            index=primary_sequence_time_channels[0].key,
            data_type=sy.DataType.FLOAT32,
        ),
        sy.Channel(
            name="primary_sequence_int_data",
            index=primary_sequence_time_channels[1].key,
            data_type=sy.DataType.UINT8,
        ),
    ],
    retrieve_if_name_exists=True,
)

# Create the channels for the second sequence. Note that both channels will need
# different index channels, or an error will be thrown when trying to write to both of
# them.

secondary_sequence_time_channels = client.channels.create(
    [
        sy.Channel(
            name="secondary_sequence_int_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        ),
        sy.Channel(
            name="secondary_sequence_float_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        ),
    ],
    retrieve_if_name_exists=True,
)

secondary_sequence_data_channels = client.channels.create(
    [
        sy.Channel(
            name="secondary_sequence_float_data",
            index=secondary_sequence_time_channels[0].key,
            data_type=sy.DataType.FLOAT32,
        ),
        sy.Channel(
            name="secondary_sequence_int_data",
            index=secondary_sequence_time_channels[1].key,
            data_type=sy.DataType.UINT8,
        ),
    ],
    retrieve_if_name_exists=True,
)

# Define the script for the first sequence. This script will set the float32 data
# channel to a sine wave, and the uint8 data channel to a sawtooth wave.

# Depending on your text editor, you may see the below string colored in the Lua
# language because of the language=Lua pragma above it.

# language=Lua
primary_sequence_script = """
set("primary_sequence_float_data", math.sin(iteration * 2 * math.pi / 512 - math.pi / 2) )
set("primary_sequence_int_data", iteration)
"""

# Create the first sequence. This includes specifying the name of the sequence, the rate
# it gets executed, what channels it will read from and write to, and the script that
# controls its behavior.

primary_sequence = sequence.Sequence(
    name="Primary Sequence",
    rate=sy.Rate.HZ * 100,
    write=[
        primary_sequence_data_channels[0].key,
        primary_sequence_data_channels[1].key,
    ],
    script=primary_sequence_script,
)

# This command will configure and create the sequence on the Synnax Cluster.

client.tasks.configure(primary_sequence)

# This command will start the sequence.

primary_sequence.start()

# Define the script for the second sequence. This script will read the float32 and uint8
# data channels from the first sequence, and write a modulated waveform and a uint7
# sawtooth wave to the second sequence's data channels.

# language=Lua
secondary_sequence_script = """
local input_float_data = primary_sequence_float_data or 0
local input_int_data = primary_sequence_int_data or 0

local new_float_data = input_float_data * 10 + math.cos(iteration * 2 * math.pi / 100)
local new_int_data = input_int_data - (iteration & 0x7F)

set("secondary_sequence_float_data", new_float_data)
set("secondary_sequence_int_data", new_int_data)
"""

secondary_sequence = sequence.Sequence(
    name="Secondary Sequence",
    rate=sy.Rate.HZ * 10,
    read=[primary_sequence_data_channels[0].key, primary_sequence_data_channels[1].key],
    write=[
        secondary_sequence_data_channels[0].key,
        secondary_sequence_data_channels[1].key,
    ],
    script=secondary_sequence_script,
)

# This command will configure and create the sequence on the Synnax Cluster.

client.tasks.configure(secondary_sequence)

# This command will start the sequence.

secondary_sequence.start()

print("Sequences are running. Press Ctrl+C to stop.")


# Define signal handler for both SIGINT (Ctrl+C) and SIGTERM to stop the sequences
def signal_handler(sig, frame):
    print(f"\nReceived signal {sig}. Stopping sequences...")
    secondary_sequence.stop()
    primary_sequence.stop()
    print("Sequences stopped.")
    sys.exit(0)


# Register signal handlers
signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)

signal.pause()
