#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import click
import matplotlib.pyplot as plt
import numpy as np

import synnax as sy
from synnax.cli import default
from synnax.cli.channel import select_channel
from synnax.cli.connect import connect_client
from synnax.cli.warning import warning


@click.command()
@click.pass_context
def check_timing(ctx: click.Context) -> None:
    """Check the timing characteristics of a time channel."""
    warning(ctx)
    client = connect_client(default.context())
    if client is None:
        return

    # Get all channels and filter for time channels
    channels = client.channels.retrieve(["*"])
    time_channels = [ch for ch in channels if ch.data_type == sy.DataType.TIMESTAMP]

    if not time_channels:
        ctx.console.error("No time channels found in the database")
        return

    # Let user select the time channel
    time_channel = select_channel(default.context(), time_channels, key="name")
    if time_channel is None:
        return

    # Let user specify duration
    duration = default.context().console.ask(
        "How long would you like to collect samples for (in seconds)?",
        type_=int,
        default=10,
    )
    span = sy.TimeSpan.SECOND * duration

    # Collect samples
    offsets, diffs, times, local_start, local_end = collect_samples(
        client, time_channel.key, span
    )

    # Replace the individual report calls with the combined report
    create_timing_report(times, offsets, diffs, local_start, local_end)


def collect_samples(
    client: sy.Synnax,
    time_channel: sy.channel.ChannelKey,
    span: sy.TimeSpan,
):
    # Tracks the offset between the local clock and the time channel
    offsets: list[sy.TimeSpan] = list()
    # Tracks the spacing between the samples inside individual reads
    diffs: list[sy.TimeSpan] = list()
    # Tracks all the times received
    times: list[sy.TimeStamp] = list()
    # Track local start time
    local_start = sy.TimeStamp.now()

    with client.open_streamer(time_channel) as streamer:
        now = local_start
        end = now + span
        while now < end:
            now = sy.TimeStamp.now()
            data = streamer.read()[time_channel]
            offset = sy.TimeSpan(sy.TimeStamp.now() - sy.TimeStamp(data[-1]))
            offsets.append(offset)
            diff = sy.TimeSpan(data[-1] - data[-2])
            diffs.append(diff)
            times.extend(data)

    local_end = sy.TimeStamp.now()
    return offsets, diffs, times, local_start, local_end


def create_timing_report(
    times: list[sy.TimeStamp],
    offsets: list[sy.TimeSpan],
    diffs: list[sy.TimeSpan],
    local_start: sy.TimeStamp,
    local_end: sy.TimeStamp,
) -> None:
    # Set dark theme
    plt.style.use("dark_background")

    # Create figure with subplots
    fig = plt.figure(figsize=(15, 10))
    fig.suptitle("Timing Analysis Report", fontsize=16)

    # Time Values Derivative vs Index plot
    ax1 = plt.subplot(2, 2, 1)
    indices = np.arange(len(times) - 1)
    # Convert timestamps to nanoseconds and then calculate differences
    derivatives = np.diff(times)

    # Check for negative derivatives
    negative_indices = np.where(derivatives < 0)[0]
    if len(negative_indices) > 0:
        print("\nWARNING: Negative derivatives detected!")
        print("Negative derivatives found at indices:", negative_indices)
        print("Derivative values at these indices:", derivatives[negative_indices])
        print("Original timestamps at these locations:")
        for idx in negative_indices:
            print(f"Index {idx}: {times[idx]} -> {times[idx+1]}")

    ax1.plot(indices, derivatives, "cyan", marker="o", markersize=2)
    ax1.set_title("Time Value Derivatives vs Index")
    ax1.set_xlabel("Index")
    ax1.set_ylabel("Time Derivative (ns)")
    ax1.grid(True, alpha=0.2)

    # Clock Offsets plot
    ax2 = plt.subplot(2, 2, 2)
    offsets_array = np.array([float(o.microseconds) for o in offsets])
    offset_mean = np.mean(offsets_array)
    offset_std = np.std(offsets_array)

    # Find offset outliers
    offset_outliers = offsets_array[
        np.abs(offsets_array - offset_mean) > 5 * offset_std
    ]

    bins_offset = np.concatenate(
        [np.linspace(min(offsets_array), max(offsets_array), 1000)]
    )
    hist_offset, bins_offset, _ = ax2.hist(
        offsets_array, bins=bins_offset, alpha=0.7, color="cyan"
    )

    x_offset = np.linspace(min(offsets_array), max(offsets_array), 100)
    gaussian_offset = hist_offset.max() * np.exp(
        -((x_offset - offset_mean) ** 2) / (2 * offset_std**2)
    )
    ax2.plot(x_offset, gaussian_offset, "magenta", lw=2, label="Gaussian fit")

    ax2.set_title(
        "Distribution of Clock Offsets\nDifference between time channel and local machine reception time"
    )
    ax2.set_xlabel("Offset (microseconds)")
    ax2.set_ylabel("Count")
    ax2.legend()
    ax2.grid(True, alpha=0.2)

    # Time Differences plot
    ax3 = plt.subplot(2, 2, 3)
    diffs_array = np.array([float(d.microseconds) for d in diffs])
    diff_mean = np.mean(diffs_array)
    diff_std = np.std(diffs_array)

    # Find diff outliers
    diff_outliers = diffs_array[np.abs(diffs_array - diff_mean) > 5 * diff_std]

    # Convert time differences to rates
    rates = [sy.Rate(d) for d in diffs]
    avg_rate = np.mean([float(r) for r in rates])

    bins_diff = np.concatenate([np.linspace(min(diffs_array), max(diffs_array), 500)])
    hist_diff, bins_diff, _ = ax3.hist(
        diffs_array, bins=bins_diff, alpha=0.7, color="cyan"
    )

    x_diff = np.linspace(min(diffs_array), max(diffs_array), 100)
    gaussian_diff = hist_diff.max() * np.exp(
        -((x_diff - diff_mean) ** 2) / (2 * diff_std**2)
    )
    ax3.plot(x_diff, gaussian_diff, "magenta", lw=2, label="Gaussian fit")

    ax3.set_title("Distribution of Time Differences")
    ax3.set_xlabel("Time Difference (microseconds)")
    ax3.set_ylabel("Count")
    ax3.legend()
    ax3.grid(True, alpha=0.2)

    # Calculate stream rates and times
    stream_start = times[0]
    stream_end = times[-1]
    stream_duration = sy.TimeSpan(stream_end - stream_start)
    stream_rate = sy.Rate(stream_duration / len(times))
    local_duration = sy.TimeSpan(local_end - local_start)

    # Statistics text box
    ax4 = plt.subplot(2, 2, 4)
    ax4.axis("off")
    stats_text = (
        f"Statistics:\n\n"
        f"Clock Offsets:\n"
        f"Mean: {offset_mean:.2f} µs\n"
        f"Std Dev: {offset_std:.2f} µs\n"
        f"Outliers (>5σ): {len(offset_outliers)} values\n\n"
        f"Time Differences:\n"
        f"Mean: {diff_mean:.2f} µs\n"
        f"Std Dev: {diff_std:.2f} µs\n"
        f"Outliers (>5σ): {len(diff_outliers)} values\n"
        f"Calculated Rate: {avg_rate:.2f} Hz\n\n"
        f"Stream Statistics:\n"
        f"Local Start: {local_start}\n"
        f"Local End: {local_end}\n"
        f"Local Duration: {local_duration}\n"
        f"Received Start: {sy.TimeStamp(stream_start)}\n"
        f"Received End: {sy.TimeStamp(stream_end)}\n"
        f"Received Duration: {stream_duration}\n"
        f"Average Rate: {stream_rate}"
    )
    ax4.text(
        0.1, 0.5, stats_text, fontsize=12, verticalalignment="center", color="white"
    )

    # Adjust layout and save
    plt.tight_layout()
    plt.savefig("timing_report.png")
    plt.show()
    plt.close()

    # Print statistics to console as well
    print("\nTiming Analysis Results:")
    print(f"Mean offset: {offset_mean:.2f} microseconds")
    print(f"Standard deviation of offsets: {offset_std:.2f} microseconds")
    print(f"Mean time difference: {diff_mean:.2f} microseconds")
    print(f"Standard deviation of time differences: {diff_std:.2f} microseconds")
