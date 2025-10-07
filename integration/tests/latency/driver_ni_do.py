#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import platform
from collections import deque

import matplotlib
import matplotlib.pyplot as plt
import numpy as np
import synnax as sy
from synnax.hardware import ni

from framework.utils import (
    get_cpu_cores,
    get_machine_info,
    get_memory_info,
    get_synnax_version,
)
from tests.latency.latency import Latency

matplotlib.use("Agg")  # Use non-interactive backend


class DriverNiDo(Latency):
    """
    Send a command to an NI DO channel and measure the latency between the
    core and loop-back (python) timestamp.
    """

    def setup(self) -> None:
        super().setup()
        if platform.system().lower() != "windows":
            self.auto_pass(msg="Windows DAQmx drivers required")
        super().setup()

    def run(self) -> None:

        client = self.client

        time_index: deque[sy.TimeStamp] = deque()
        latencies_core: deque[float] = deque()
        latencies_loop: deque[float] = deque()

        self.log("Searching for NI DO device: SYMod1")
        devices = client.hardware.devices.retrieve(keys=[], ignore_not_found=True)
        if not devices:
            raise RuntimeError("No devices found")

        device = None
        for device in devices:
            # Sim device must be set up in NI MAX
            if device.location == "SYMod1":
                dev = device
                self.log(f"Found NI DO device: {dev.location}")
                break

        self.log("Creating Channels")

        # Create CMD Channels
        do_1_cmd_time = client.channels.create(
            name="do_1_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        do_1_cmd = client.channels.create(
            name="do_1_cmd",
            data_type=sy.DataType.UINT8,
            index=do_1_cmd_time.key,
            retrieve_if_name_exists=True,
        )

        # Create STATE Channels
        do_state_time = client.channels.create(
            name="do_state_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        do_1_state = client.channels.create(
            name="do_1_state",
            index=do_state_time.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        self.log("Creating Task")
        # Create Task
        tsk = ni.DigitalWriteTask(
            name="Basic Digital Write",
            device=dev.key,
            state_rate=sy.Rate.HZ * 2000,
            data_saving=True,
            channels=[
                ni.DOChan(
                    cmd_channel=do_1_cmd.key,
                    state_channel=do_1_state.key,
                    port=0,
                    line=0,
                ),
            ],
        )
        tsk = client.hardware.tasks.configure(tsk)

        # Run NI DO Task
        self.log("Running NI DO Task")
        with tsk.run():
            with client.open_streamer([do_1_state.key, do_state_time.key]) as stream:
                with client.open_writer(sy.TimeStamp.now(), do_1_cmd.key) as writer:

                    start_time = sy.TimeStamp.now()
                    now_time = sy.TimeStamp.now()
                    cmd_state: bool = False

                    self.log("Begin latency test")
                    while (now_time - start_time) < sy.TimeSpan.SECOND * 3:
                        now_time = sy.TimeStamp.now()

                        # Prepare
                        cmd_state = not cmd_state
                        val_found = False

                        # Write
                        writer.write(do_1_cmd.key, int(cmd_state))
                        write_time = sy.TimeStamp.now()

                        # Read
                        while not val_found:
                            frame = stream.read(timeout=1)
                            if frame is not None:
                                for i, v in enumerate(frame[do_1_state.key]):
                                    if v == int(cmd_state):
                                        state_timestamp_loop = sy.TimeStamp.now()
                                        state_timestamp_core = sy.TimeStamp(
                                            frame[do_state_time.key][i]
                                        )
                                        val_found = True

                        # Calculate latency
                        latency_core = sy.TimeSpan(
                            state_timestamp_core - write_time
                        ).milliseconds
                        latency_loop = sy.TimeSpan(
                            state_timestamp_loop - write_time
                        ).milliseconds

                        # Store
                        time_index.append(write_time)
                        latencies_core.append(latency_core)
                        latencies_loop.append(latency_loop)

                    # Set back to 0
                    writer.write(do_1_cmd.key, int(False))
                    frame = stream.read(timeout=1)

        self.log(f"Total samples: {len(latencies_core)}")

        # Convert to numpy arrays and milliseconds
        latencies_core_ms = np.array(latencies_core)
        latencies_loop_ms = np.array(latencies_loop)

        # Get statistics
        stats_core = self.calculate_stats(latencies_core_ms, "Driver Latency")
        stats_loop = self.calculate_stats(
            latencies_loop_ms, "Loop Latency (Python timestamp)"
        )

        # Create range for the latency benchmark
        self.log("Creating latency_benchmark range")
        latency_range = client.ranges.create(
            name="Latency Benchmark: NI DO Driver Loop-back",
            time_range=sy.TimeRange(time_index[0], time_index[-1]),
        )

        # Add metadata to the range
        self.log("Adding metadata to range")
        machine_name = get_machine_info()
        memory_info = get_memory_info()
        cpu_cores = get_cpu_cores()

        latency_range.meta_data.set(
            {
                "machine": machine_name,
                "memory": memory_info if memory_info else "",
                "cpu_cores": cpu_cores if cpu_cores else "",
                "driver_mean_ms": round(stats_core["mean"], 3),
                "driver_median_ms": round(stats_core["median"], 3),
                "driver_std_ms": round(stats_core["std"], 3),
                "driver_p90_ms": round(stats_core["p90"], 3),
                "driver_p95_ms": round(stats_core["p95"], 3),
                "driver_p99_ms": round(stats_core["p99"], 3),
                "loopback_mean_ms": round(stats_loop["mean"], 3),
                "loopback_median_ms": round(stats_loop["median"], 3),
                "loopback_std_ms": round(stats_loop["std"], 3),
                "loopback_p90_ms": round(stats_loop["p90"], 3),
                "loopback_p95_ms": round(stats_loop["p95"], 3),
                "loopback_p99_ms": round(stats_loop["p99"], 3),
            }
        )

        # Create latency channels and publish to Synnax
        latency_time = client.channels.create(
            name="latency_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        latency_core_ch = client.channels.create(
            name="latency_core",
            data_type=sy.DataType.FLOAT32,
            index=latency_time.key,
            retrieve_if_name_exists=True,
        )
        latency_loop_ch = client.channels.create(
            name="latency_loopback",
            data_type=sy.DataType.FLOAT32,
            index=latency_time.key,
            retrieve_if_name_exists=True,
        )

        # Write latency data to Synnax
        self.log("Writing latency data to Synnax")
        with client.open_writer(
            start=sy.TimeStamp.now(),
            channels=[latency_time.key, latency_core_ch.key, latency_loop_ch.key],
        ) as writer:
            writer.write(
                {
                    latency_time.key: list(time_index),
                    latency_core_ch.key: latencies_core_ms,
                    latency_loop_ch.key: latencies_loop_ms,
                }
            )

        self.plot_latencies(
            latencies_core_ms, latencies_loop_ms, stats_core, stats_loop
        )

        self.check_results(stats_core, stats_loop)

    def calculate_stats(self, latencies_ms: np.ndarray, name: str) -> dict[str, float]:
        """Calculate and print statistics for a latency dataset"""
        mean = np.mean(latencies_ms)
        median = np.median(latencies_ms)
        std = np.std(latencies_ms)
        min_lat = np.min(latencies_ms)
        max_lat = np.max(latencies_ms)
        p50 = np.percentile(latencies_ms, 50)
        p90 = np.percentile(latencies_ms, 90)
        p95 = np.percentile(latencies_ms, 95)
        p99 = np.percentile(latencies_ms, 99)
        peak_to_peak = max_lat - min_lat
        jitter = np.abs(np.diff(latencies_ms))
        avg_jitter = np.mean(jitter)

        self.log(f"=== {name} ===")
        self.log(f"Mean: {mean:.2f} ms")
        self.log(f"Median: {median:.2f} ms")
        self.log(f"Std: {std:.2f} ms")
        self.log(f"Min: {min_lat:.2f} ms")
        self.log(f"Max: {max_lat:.2f} ms")
        self.log(f"P50: {p50:.2f} ms")
        self.log(f"P90: {p90:.2f} ms")
        self.log(f"P95: {p95:.2f} ms")
        self.log(f"P99: {p99:.2f} ms")
        self.log(f"Peak-to-peak jitter: {peak_to_peak:.2f} ms")
        self.log(f"Avg jitter: {avg_jitter:.2f} ms")

        return {
            "mean": mean,
            "median": median,
            "std": std,
            "min": min_lat,
            "max": max_lat,
            "p50": p50,
            "p90": p90,
            "p95": p95,
            "p99": p99,
            "peak_to_peak": peak_to_peak,
            "jitter": jitter,
            "avg_jitter": avg_jitter,
        }

    def check_results(
        self, stats_driver: dict[str, float], stats_loop: dict[str, float]
    ) -> None:

        assert stats_driver["mean"] <= 4, "Driver mean latency is greater than 4 ms"
        assert stats_loop["mean"] <= 4, "Loop mean latency is greater than 4 ms"

        assert stats_driver["median"] <= 4, "Driver median latency is greater than 4 ms"
        assert stats_loop["median"] <= 4, "Loop median latency is greater than 4 ms"

        assert stats_driver["std"] <= 4, "Driver std latency is greater than 4 ms"
        assert stats_loop["std"] <= 4, "Loop std latency is greater than 4 ms"

        assert stats_driver["p90"] <= 6, "Driver p90 latency is greater than 6 ms"
        assert stats_loop["p90"] <= 6, "Loop p90 latency is greater than 6 ms"

        assert stats_driver["p95"] <= 8, "Driver p95 latency is greater than 8 ms"
        assert stats_loop["p95"] <= 8, "Loop p95 latency is greater than 8 ms"

        assert stats_driver["p99"] <= 10, "Driver p99 latency is greater than 10 ms"
        assert stats_loop["p99"] <= 10, "Loop p99 latency is greater than 10 ms"

        assert (
            stats_driver["peak_to_peak"] < 40
        ), "Driver peak-to-peak latency is greater than 10 ms"
        assert (
            stats_loop["peak_to_peak"] < 40
        ), "Loop peak-to-peak latency is greater than 10 ms"

        assert stats_driver["avg_jitter"] < 4, "Driver avg jitter is greater than 4 ms"
        assert stats_loop["avg_jitter"] < 4, "Loop avg jitter is greater than 4 ms"

    def plot_latencies(
        self,
        latencies_core_ms: np.ndarray,
        latencies_loop_ms: np.ndarray,
        stats_core: dict[str, float],
        stats_loop: dict[str, float],
    ) -> None:
        fig = plt.figure(figsize=(10, 8))
        gs = fig.add_gridspec(3, 2, height_ratios=[2, 1, 1])

        # Plot 1: Core vs Loop Latency over time (spans both columns)
        ax1 = fig.add_subplot(gs[0, :])
        ax1.plot(
            latencies_core_ms,
            label="Core Latency (Core timestamp)",
            alpha=0.7,
            linewidth=0.5,
            color="blue",
        )
        ax1.plot(
            latencies_loop_ms,
            label="Loop-back Latency (Python timestamp)",
            alpha=0.7,
            linewidth=0.5,
            color="red",
        )
        ax1.axhline(
            y=stats_core["mean"],
            color="blue",
            linestyle="--",
            alpha=0.5,
            label=f"Core Mean: {stats_core['mean']:.2f}ms",
        )
        ax1.axhline(
            y=stats_loop["mean"],
            color="red",
            linestyle="--",
            alpha=0.5,
            label=f"Loop Mean: {stats_loop['mean']:.2f}ms",
        )
        ax1.set_title("NI Digital Output Latency Comparison Over Time")
        ax1.set_xlabel("Sample Number")
        ax1.set_ylabel("Latency (ms)")
        ax1.grid(True, alpha=0.3)
        ax1.legend()

        # Plot 2: Core Jitter over time
        ax2 = fig.add_subplot(gs[1, 0])
        ax2.plot(
            stats_core["jitter"],
            label="Core Jitter",
            color="blue",
            alpha=0.6,
            linewidth=0.5,
        )
        ax2.axhline(
            y=stats_core["avg_jitter"],
            color="blue",
            linestyle="--",
            label=f"Core Avg: {stats_core['avg_jitter']:.2f}ms",
        )
        ax2.set_title("Core Latency Jitter Over Time")
        ax2.set_xlabel("Sample Number")
        ax2.set_ylabel("Jitter (ms)")
        ax2.grid(True, alpha=0.3)
        ax2.legend()

        # Plot 3: Loop Jitter over time
        ax3 = fig.add_subplot(gs[1, 1])
        ax3.plot(
            stats_loop["jitter"],
            label="Loop Jitter",
            color="red",
            alpha=0.6,
            linewidth=0.5,
        )
        ax3.axhline(
            y=stats_loop["avg_jitter"],
            color="red",
            linestyle="--",
            label=f"Loop Avg: {stats_loop['avg_jitter']:.2f}ms",
        )
        ax3.set_title("Loop-back Latency Jitter Over Time")
        ax3.set_xlabel("Sample Number")
        ax3.set_ylabel("Jitter (ms)")
        ax3.grid(True, alpha=0.3)
        ax3.legend()

        # Plot 4: Core Latency histogram
        ax4 = fig.add_subplot(gs[2, 0])
        ax4.hist(latencies_core_ms, bins=50, alpha=0.7, color="blue", edgecolor="black")
        ax4.axvline(
            x=stats_core["mean"],
            color="orange",
            linestyle=":",
            label=f"Mean: {stats_core['mean']:.2f}ms",
        )
        ax4.axvline(
            x=stats_core["median"],
            color="green",
            linestyle="--",
            label=f"Median: {stats_core['median']:.2f}ms",
        )
        ax4.set_title("Core Latency Distribution")
        ax4.set_xlabel("Latency (ms)")
        ax4.set_ylabel("Frequency")
        ax4.set_xlim(left=0)
        ax4.legend()
        ax4.grid(True, alpha=0.3)

        # Plot 5: Loop Latency histogram
        ax5 = fig.add_subplot(gs[2, 1])
        ax5.hist(latencies_loop_ms, bins=50, alpha=0.7, color="red", edgecolor="black")
        ax5.axvline(
            x=stats_loop["mean"],
            color="orange",
            linestyle=":",
            label=f"Mean: {stats_loop['mean']:.2f}ms",
        )
        ax5.axvline(
            x=stats_loop["median"],
            color="green",
            linestyle="--",
            label=f"Median: {stats_loop['median']:.2f}ms",
        )
        ax5.set_title("Loop-back Latency Distribution")
        ax5.set_xlabel("Latency (ms)")
        ax5.set_ylabel("Frequency")
        ax5.set_xlim(left=0)
        ax5.legend()
        ax5.grid(True, alpha=0.3)

        # Get machine information
        machine_name = get_machine_info()
        memory_info = get_memory_info()
        cpu_cores = get_cpu_cores()

        machine_desc = f"Machine: {machine_name}"
        if cpu_cores:
            machine_desc += f", {cpu_cores}"
        if memory_info:
            machine_desc += f", {memory_info}"

        plt.suptitle(
            "NI Digital Output Latency Analysis - Core vs Loop-back",
            fontsize=14,
            y=0.98,
        )
        plt.figtext(
            0.5,
            0.92,
            f"{machine_desc} | Platform Version: {get_synnax_version()}",
            fontsize=10,
            ha="center",
        )
        plt.tight_layout()
        plt.subplots_adjust(top=0.85)

        # Save the plot
        os.makedirs("tests/results", exist_ok=True)
        output_path = "tests/results/ni_do_latency_analysis.png"
        plt.savefig(output_path, dpi=300, bbox_inches="tight")
        self.log(f"Plot saved to: {os.path.abspath(output_path)}")
        plt.close(fig)
