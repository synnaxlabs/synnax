#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import synnax as sy
from synnax.hardware import sequence

client = sy.Synnax()

# Create a signal channel that can be used to start and stop the control sequence.
signal_channel = client.channels.create(
    name="Start Control Sequence",
    data_type="uint8",
    virtual=True,
    retrieve_if_name_exists=True,
)
ch_key = signal_channel.key


# Retrieve the control sequence from Synnax.
tsk = client.hardware.tasks.retrieve(name="Control Sequence")
sequence = sequence.Sequence(tsk)

# Open a stream on the signal channel. Start and stop the control sequence based on what
# gets written to the signal channel.
with client.open_streamer(ch_key) as streamer:
    while True:
        frame = streamer.read()
        if frame[ch_key] == 1:
            sequence.start()
        elif frame[ch_key] == 0:
            sequence.stop()
