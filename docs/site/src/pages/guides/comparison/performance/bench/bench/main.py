import json

from bench.bench_synnax import bench_synnax
from bench.bench_influx import bench_influx
from bench.bench_timescale import bench_timescale
from bench.config import TestConfig, SAMPLES_PER_CHANNEL
import matplotlib.pyplot as plt
import synnax as sy

def calc_throughput(samples_per_channel: list[int],
                    channel_counts: list[int],
                    time: list[sy.TimeSpan]):
    return [
        (samples_per_channel[i] * channel_counts[i] / time[i].seconds) for i in range(len(samples_per_channel))]

def plot_throughput(
    synnax_times,
    influx_times,
    timescale_times,
    samples_per_channel: list[int],
    channel_counts: list[int],
    vary: str,
    file_name
):
    synnax_throughput = calc_throughput(samples_per_channel, channel_counts, synnax_times)
    influx_throughput = calc_throughput(samples_per_channel, channel_counts, influx_times)
    timescale_throughput = calc_throughput(samples_per_channel, channel_counts, timescale_times)
    if vary == "samples_per_channel":
        plt.plot(samples_per_channel, synnax_throughput, label="Synnax")
        plt.plot(samples_per_channel, influx_throughput, label="InfluxDB")
        plt.plot(samples_per_channel, timescale_throughput, label="TimescaleDB")
        plt.xlabel("Samples per channel")
    elif vary == "channel_count":
        plt.plot(channel_counts, synnax_throughput, label="Synnax")
        plt.plot(channel_counts, influx_throughput, label="InfluxDB")
        plt.plot(channel_counts, timescale_throughput, label="TimescaleDB")
        plt.xlabel("Channel count")

    plt.ylabel("Throughput (samples/ms)")
    plt.legend()
    plt.savefig(file_name)

def plot_stats(
    synnax_stats,
    influx_stats,
    timescale_stats,
) -> None:
    synnax_mem = [stat.memory for stat in synnax_stats]
    influx_mem = [stat.memory for stat in influx_stats]
    timescale_mem = [stat.memory for stat in timescale_stats]
    synnax_cpu = [stat.cpu for stat in synnax_stats]
    influx_cpu = [stat.cpu for stat in influx_stats]
    timescale_cpu = [stat.cpu for stat in timescale_stats]

    plt.clf()
    plt.plot(synnax_mem, label="Synnax")
    plt.plot(influx_mem, label="InfluxDB")
    plt.plot(timescale_mem, label="TimescaleDB")
    plt.xlabel("Configuration")
    plt.ylabel("Memory usage (bytes)")
    plt.legend()
    plt.savefig("memory_usage.png")

    plt.clf()
    plt.plot(synnax_cpu, label="Synnax")
    plt.plot(influx_cpu, label="InfluxDB")
    plt.plot(timescale_cpu, label="TimescaleDB")

    plt.xlabel("Configuration")
    plt.ylabel("CPU usage (%)")
    plt.legend()
    plt.savefig("cpu_usage.png")


def varying_samples_per_iter():
    channel_count = 20
    samples_per_channel_per_iteration = [
        10, #1
        50, #2
        100, #3
        500, #4
        1_000, #5
        5_000, #6
        10_000, #7
        50_000, #8
        100_000, #9
        1_000_000, #10
    ]
    samples_per_channel = [
        10 * 5_000, #1
        50 * 2_000, #2
        100 * 1_000, #3
        500 * 500, #4
        1_000 * 250, #5
        5_000 * 50, #6,
        10_000 * 20, #7,
        50_000 * 20, #8
        100_000 * 20,
        1_000_000 * 10, #10
    ]
    cfgs = [TestConfig(
        channel_count=channel_count,
        samples_per_channel_per_iteration=spi,
        samples_per_channel=samples_per_channel[i]
    ) for i, spi in enumerate(samples_per_channel_per_iteration)]
    bench_throughput(cfgs, "samples_per_channel.png", "samples_per_channel")

def varying_channels():
    samples_per_channel_per_iteration = [
        100_000, #1
        50_000, #2
        25_000, #3
        10_000, #4
        5_000, #5
        2_500, #6
        1_000, #7
        500, #8
        300, #9
    ]
    samples_per_channel = [
        1_000_000, #1
        500_000, #2
        250_000, #3
        100_000, #4
        50_000, #5
        25_000, #6
        10_000, #7
        5_000, #8
        3_000, #9
    ]
    channel_counts = [
        2, #1
        5, #2
        10, #3
        20, #4
        50, #5
        100, #6
        200, #7
        500, #8
        1000, #9
    ]
    cfgs = [TestConfig(
        channel_count=cc,
        samples_per_channel_per_iteration=samples_per_channel_per_iteration[i],
        samples_per_channel=samples_per_channel[i]
    ) for i, cc in enumerate(channel_counts)]
    bench_throughput(cfgs, "channel_count.png", "channel_count")

def base_throughput():
    cfgs = [TestConfig(
        channel_count=50,
        samples_per_channel=2_000_000,
        samples_per_channel_per_iteration=50_000,
    )]
    bench_throughput(cfgs, "base_throughput.png", "samples_per_channel")

def bench_throughput(cfgs: list[TestConfig], file_name: str, vary: str):
    synnax_times = []
    influx_times = []
    timescale_times = []
    synnax_stats = []
    influx_stats = []
    timescale_stats = []
    for i, cfg in enumerate(cfgs):
        print(
            f"Running benchmark {i + 1}/{len(cfgs)} with "
            f"{cfg.samples_per_channel_per_iteration} samples per iteration and {cfg.channel_count} channels")
        synnax_stat, synnax_time = bench_synnax(cfg)
        influx_stat, influx_time = bench_influx(cfg)
        timescale_stat, timescale_time = bench_timescale(cfg)
        synnax_times.append(synnax_time)
        influx_times.append(influx_time)
        timescale_times.append(timescale_time)
        synnax_stats.append(synnax_stat)
        influx_stats.append(influx_stat)
        timescale_stats.append(timescale_stat)

    plot_throughput(
        synnax_times,
        influx_times,
        timescale_times,
        [cfg.samples_per_channel_per_iteration for cfg in cfgs],
        [cfg.channel_count for cfg in cfgs],
        vary,
        file_name
    )
    plot_stats(synnax_stats, influx_stats, timescale_stats)
    with open("stats-b.json", "w") as f:
        json.dump({
            "synnax_memory": [stat.memory for stat in synnax_stats],
            "influx_memory": [stat.memory for stat in influx_stats],
            "timescale_memory": [stat.memory for stat in timescale_stats],
            "synnax_cpu": [stat.cpu for stat in synnax_stats],
            "influx_cpu": [stat.cpu for stat in influx_stats],
            "timescale_cpu": [stat.cpu for stat in timescale_stats],
            "synnax_time": [time.seconds for time in synnax_times],
            "influx_time": [time.seconds for time in influx_times],
            "timescale_time": [time.seconds for time in timescale_times],
        }, f, indent=4)


if __name__ == "__main__":
    # varying_samples_per_iter()
    varying_channels()
    # base_throughput()
