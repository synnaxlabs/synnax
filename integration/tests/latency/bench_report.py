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

matplotlib.use("Agg")  # Use non-interactive backend
import matplotlib.pyplot as plt
import numpy as np
import synnax as sy

from framework.utils import get_machine_info, get_memory_info, get_synnax_version
from tests.latency.latency import Latency


class BenchReport(Latency):

    def setup(self) -> None:
        super().setup()
        self.set_manual_timeout(10)

        self.report_client = sy.Synnax(
            host=self.synnax_connection.server_address,
            port=self.synnax_connection.port,
            username=self.synnax_connection.username,
            password=self.synnax_connection.password,
            secure=self.synnax_connection.secure,
        )

        self.subscribe(["bench_state", "bench_command"])

        self.loop_start = sy.TimeStamp.now()

    def run(self) -> None:
        """
        Run the test case.
        """

        cycles: int = 0
        times: deque[sy.TimeStamp] = deque()
        loop_start: sy.TimeStamp = sy.TimeStamp.now()
        state_channel: str = "bench_state"
        cmd_channel: str = "bench_command"
        bench_time: sy.TimeSpan = sy.TimeSpan.SECOND * 3

        try:
            with self.report_client.open_streamer(state_channel) as stream:
                with self.report_client.open_writer(
                    sy.TimeStamp.now(), cmd_channel
                ) as writer:
                    while sy.TimeStamp.since(loop_start) < bench_time:
                        start = sy.TimeStamp.now()
                        writer.write(cmd_channel, True)
                        value = stream.read()
                        times.append(sy.TimeStamp.since(start))
                        cycles += 1

        except Exception as e:
            raise Exception(f"EXCEPTION: {e}")

        self.log(f"Cycles/second: {cycles / bench_time.seconds:.2f}")

        # Convert times to milliseconds for better readability
        times_ms = [float(t.microseconds) / 1000 for t in times]

        # Calculate jitter metrics
        peak_to_peak_jitter = max(times_ms) - min(times_ms)

        # Calculate average jitter (mean deviation between consecutive samples)
        consecutive_differences = np.abs(np.diff(times_ms))
        average_jitter = np.mean(consecutive_differences)

        # Calculate percentiles
        p90 = np.percentile(times_ms, 90)
        p95 = np.percentile(times_ms, 95)
        p99 = np.percentile(times_ms, 99)

        # Get machine information dynamically
        machine_name = get_machine_info()
        memory_info = get_memory_info()
        machine_desc = f"Machine: {machine_name}"
        if memory_info:
            machine_desc += f", {memory_info}"

        # Create the plot (updated for 2x2 layout)
        fig = plt.figure(figsize=(12, 10))
        gs = fig.add_gridspec(2, 2, height_ratios=[2, 1])
        ax1 = fig.add_subplot(gs[0, :])  # Top row, full width
        ax2 = fig.add_subplot(gs[1, 0])  # Bottom left
        ax3 = fig.add_subplot(gs[1, 1])  # Bottom right

        # Add title and description at the top
        plt.suptitle("Echo Benchmark Results", fontsize=14, y=0.98)
        plt.figtext(
            0.1,
            0.92,
            f"{machine_desc} | Platform Version: {get_synnax_version()} | Config: "
            "LL-PP-C500-R1-50-R2-10",
            fontsize=10,
            ha="left",
        )

        # Top plot: Latency over time with percentiles
        ax1.plot(times_ms, label="Latency", alpha=0.6)
        ax1.axhline(y=float(p90), color="r", linestyle="--", label=f"P90: {p90:.2f}ms")
        ax1.axhline(y=float(p95), color="g", linestyle="--", label=f"P95: {p95:.2f}ms")
        ax1.axhline(y=float(p99), color="b", linestyle="--", label=f"P99: {p99:.2f}ms")
        ax1.set_title("Latency Over Time")
        ax1.set_xlabel("Sample Number")
        ax1.set_ylabel("Latency (ms)")
        ax1.grid(True, alpha=0.3)
        ax1.legend()

        # Bottom left plot: Jitter over time
        ax2.plot(consecutive_differences, label="Jitter", color="purple", alpha=0.6)
        ax2.axhline(
            y=average_jitter,
            color="r",
            linestyle="--",
            label=f"Avg Jitter: {average_jitter:.2f}ms",
        )
        ax2.set_title("Jitter Over Time")
        ax2.set_xlabel("Sample Number")
        ax2.set_ylabel("Jitter (ms)")
        ax2.grid(True, alpha=0.3)
        ax2.legend()

        # Bottom right plot: Histograms
        # Latency histogram
        ax3.hist(times_ms, bins=30, alpha=0.5, color="blue", label="Latency")
        ax3_twin = ax3.twinx()  # Create a twin axis for the jitter histogram
        ax3_twin.hist(
            consecutive_differences, bins=30, alpha=0.5, color="purple", label="Jitter"
        )

        # Set logarithmic scale for both y-axes
        ax3.set_yscale("log")
        ax3_twin.set_yscale("log")

        # Customize the histogram plot
        ax3.set_title("Distribution of Latency and Jitter (Log Scale)")
        ax3.set_xlabel("Time (ms)")
        ax3.set_ylabel("Frequency (Latency)", color="blue")
        ax3_twin.set_ylabel("Frequency (Jitter)", color="purple")

        # Add legends for both histograms
        lines1, labels1 = ax3.get_legend_handles_labels()
        lines2, labels2 = ax3_twin.get_legend_handles_labels()
        ax3.legend(lines1 + lines2, labels1 + labels2, loc="upper right")

        # Adjust layout to make room for the title and description
        plt.tight_layout()
        plt.subplots_adjust(top=0.85)  # Make room for the title and description

        # Selected arbitrarily. However, these values should
        # provide a good maximumm threshold
        max_p90 = 5
        max_p95 = 6
        max_p99 = 10
        max_average_jitter = 5

        if platform.system().lower() == "windows":
            max_peak_to_peak_jitter = 40
        else:
            max_peak_to_peak_jitter = 20

        # Print statistics
        p90_msg = f"P90: {p90:.2f}ms"
        if p90 > max_p90:
            p90_msg += f" is greater than {max_p90}ms (FAILED)"
            self.fail()
        self.log(p90_msg)

        p95_msg = f"P95: {p95:.2f}ms"
        if p95 > max_p95:
            p95_msg += f" is greater than {max_p95}ms (FAILED)"
            self.fail()
        self.log(p95_msg)

        p99_msg = f"P99: {p99:.2f}ms"
        if p99 > max_p99:
            p99_msg += f" is greater than {max_p99}ms (FAILED)"
            self.fail()
        self.log(p99_msg)

        peak_to_peak_jitter_msg = f"Peak-to-peak jitter: {peak_to_peak_jitter:.2f}ms"
        if peak_to_peak_jitter > max_peak_to_peak_jitter:
            peak_to_peak_jitter_msg += (
                f" is greater than {max_peak_to_peak_jitter}ms (FAILED)"
            )
            self.fail()
        self.log(peak_to_peak_jitter_msg)

        average_jitter_msg = f"Average jitter: {average_jitter:.2f}ms"
        if average_jitter > max_average_jitter:
            average_jitter_msg += f" is greater than {max_average_jitter}ms (FAILED)"
            self.fail()
        self.log(average_jitter_msg)

        os.makedirs("tests/results", exist_ok=True)
        plt.savefig("tests/results/bench_load.png", dpi=300, bbox_inches="tight")
        self.log(
            f"Saved benchmark plot to: {os.path.abspath('tests/results/bench_load.png')}"
        )
        plt.close(fig)
