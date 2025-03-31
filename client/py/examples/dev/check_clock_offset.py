#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import matplotlib.pyplot as plt
import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# We can just specify the names of the channels we'd like to stream from. These channels
# were created by running the `stream_write.py`` script.
# channels = ["T7_time"]
channels = ["abc_ai_time"]

# Number of samples to collect before plotting
N = 10
offsets = []
diffs = []
times = []

# We will open the streamer with a context manager. The context manager will
# automatically close the streamer after we're done reading.
with client.open_streamer(channels) as streamer:
    count = 0
    while count < N:
        data = streamer.read()[channels[0]]
        offset = sy.TimeSpan(sy.TimeStamp.now() - sy.TimeStamp(data[-1]))
        offsets.append(float(offset.microseconds))
        diff = sy.TimeSpan(data[-1] - data[-2])
        diffs.append(float(diff.microseconds))
        times.extend(data)
        count += 1
        if count % 100 == 0:
            print(f"Collected {count}/{N} samples...")
    print("Done collecting samples...")

# Create a plot of times vs indices
plt.figure(figsize=(10, 6))
indices = np.arange(len(times))
plt.plot(indices, times, 'b-', marker='o')
plt.title('Time Values vs Index')
plt.xlabel('Index')
plt.ylabel('Time')
plt.grid(True)
plt.savefig('times_vs_index.png')

# Convert to microseconds and calculate statistics for both offsets and diffs
offsets = np.array(offsets)
diffs = np.array(diffs)

# Calculate and plot the differences between consecutive times
time_differences = np.diff(times) # Convert to milliseconds
plt.figure(figsize=(10, 6))
indices = np.arange(len(time_differences))
plt.plot(indices, time_differences, 'r-', marker='o', markersize=3)

# Find min and max values
min_diff = np.min(time_differences)
max_diff = np.max(time_differences)

# Add min/max annotations
plt.annotate(f'Min: {min_diff:.2f}ms', 
            xy=(np.argmin(time_differences), min_diff),
            xytext=(10, 10), textcoords='offset points')
plt.annotate(f'Max: {max_diff:.2f}ms', 
            xy=(np.argmax(time_differences), max_diff),
            xytext=(10, -10), textcoords='offset points')

plt.title('Time Differences vs Index')
plt.xlabel('Index')
plt.ylabel('Time Difference (milliseconds)')
plt.grid(True)

# Disable scientific notation on y-axis
plt.gca().yaxis.set_major_formatter(plt.FormatStrFormatter('%.2f'))

plt.savefig('time_consecutive_differences.png')

offset_mean = np.mean(offsets)
offset_std = np.std(offsets)

diff_mean = np.mean(diffs)
diff_std = np.std(diffs)

# Create the offset plot
plt.figure(figsize=(10, 6))
# Create custom bins with more resolution in 0-500 range
bins_offset = np.concatenate(
    [np.linspace(min(offsets), max(offsets), 1000)]  # 10 bins for the rest
)
hist_offset, bins_offset, _ = plt.hist(offsets, bins=bins_offset, alpha=0.7, color="b")

# Plot the Gaussian fit for offsets
x_offset = np.linspace(min(offsets), max(offsets), 100)
gaussian_offset = hist_offset.max() * np.exp(
    -((x_offset - offset_mean) ** 2) / (2 * offset_std**2)
)
plt.plot(x_offset, gaussian_offset, "r-", lw=2, label="Gaussian fit")

plt.title("Distribution of Clock Offsets")
plt.xlabel("Offset (microseconds)")
plt.ylabel("Count")
plt.legend()
plt.grid(True)
plt.savefig("clock_offsets.png")

# Create the diffs plot
plt.figure(figsize=(10, 6))
# Create custom bins with more resolution in 0-500 range
bins_diff = np.concatenate(
    [np.linspace(min(diffs), max(diffs), 500)]  # 10 bins for the rest
)
hist_diff, bins_diff, _ = plt.hist(diffs, bins=bins_diff, alpha=0.7, color="g")

# Plot the Gaussian fit for diffs
x_diff = np.linspace(min(diffs), max(diffs), 100)
gaussian_diff = hist_diff.max() * np.exp(
    -((x_diff - diff_mean) ** 2) / (2 * diff_std**2)
)
plt.plot(x_diff, gaussian_diff, "r-", lw=2, label="Gaussian fit")

plt.title("Distribution of Time Differences")
plt.xlabel("Time Difference (microseconds)")
plt.ylabel("Count")
plt.legend()
plt.grid(True)
plt.savefig("time_differences.png")

print(f"Mean offset: {offset_mean:.2f} microseconds")
print(f"Standard deviation: {offset_std:.2f} microseconds")
print(f"Mean time difference: {diff_mean:.2f} microseconds")
print(f"Standard deviation of time differences: {diff_std:.2f} microseconds")
