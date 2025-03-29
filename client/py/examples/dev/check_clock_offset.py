#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to stream live data from a channel in Synnax.
Live-streaming is useful for real-time data processing and analysis, and is an integral
part of Synnax's control sequence and data streaming capabilities.

This example requires the `stream_write.py` file to be running in a separate terminal.
"""

import synnax as sy
import numpy as np
import matplotlib.pyplot as plt
# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# We can just specify the names of the channels we'd like to stream from. These channels
# were created by running the `stream_write.py`` script.
channels = ["T7_time"]

# Number of samples to collect before plotting
N = 1000
offsets = []

# We will open the streamer with a context manager. The context manager will
# automatically close the streamer after we're done reading.
with client.open_streamer(channels) as streamer:
    print("HERE")
    count = 0
    while count < N:
        data = streamer.read()[channels[0]]
        offset = sy.TimeSpan(sy.TimeStamp.now() - sy.TimeStamp(data[-1]))
        offsets.append(float(offset.microseconds))
        print(f"Offset: {offset.microseconds}")
        count += 1
        if count % 100 == 0:
            print(f"Collected {count}/{N} samples...")
    print("Done collecting samples...")


# Convert to microseconds and calculate statistics
offsets = np.array(offsets)
mean = np.mean(offsets)
std = np.std(offsets)

# Create the plot
plt.figure(figsize=(10, 6))
plt.hist(offsets, bins=50, density=True, alpha=0.7, color='b')

# Plot the Gaussian fit
x = np.linspace(min(offsets), max(offsets), 100)
gaussian = (1/(std * np.sqrt(2*np.pi))) * np.exp(-(x-mean)**2 / (2*std**2))
plt.plot(x, gaussian, 'r-', lw=2, label='Gaussian fit')

plt.title('Distribution of Clock Offsets')
plt.xlabel('Offset (microseconds)')
plt.ylabel('Density')
plt.legend()
plt.grid(True)

print(f"Mean offset: {mean:.2f} microseconds")
print(f"Standard deviation: {std:.2f} microseconds")

plt.show()
