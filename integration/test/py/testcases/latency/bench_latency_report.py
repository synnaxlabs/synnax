#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from re import S
import sys
import os
import time

import gc
import platform
import subprocess
import sys

import matplotlib
matplotlib.use('Agg')  # Use non-interactive backend
import matplotlib.pyplot as plt
import numpy as np

import synnax as sy


# Set up the path before importing framework modules
sys.path.append(os.path.join(os.path.dirname(__file__), '../..'))
from framework.TestCase import TestCase

def get_machine_info():
    """Get machine information programmatically."""
    system = platform.system()
    
    if system == "Darwin":  # macOS
        try:
            # Try to get Apple Silicon info
            result = subprocess.run(['sysctl', '-n', 'machdep.cpu.brand_string'], 
                                  capture_output=True, text=True, timeout=5)
            if result.returncode == 0:
                cpu_info = result.stdout.strip()
                if 'Apple' in cpu_info:
                    # Extract M1/M2/M3 info
                    if 'M1' in cpu_info:
                        return "Apple Silicon M1"
                    elif 'M2' in cpu_info:
                        return "Apple Silicon M2"
                    elif 'M3' in cpu_info:
                        return "Apple Silicon M3"
                    elif 'M4' in cpu_info:
                        return "Apple Silicon M4"
                    elif 'M5' in cpu_info:
                        return "Apple Silicon M5"
                    else:
                        return "Apple Silicon Mac"
                else:
                    return "Intel Mac"
            else:
                return "macOS"
        except:
            return "macOS"
    
    elif system == "Linux":
        try:
            # Try to get distribution info
            result = subprocess.run(['lsb_release', '-d'], 
                                  capture_output=True, text=True, timeout=5)
            if result.returncode == 0:
                distro = result.stdout.split('\t')[1].strip()
                return distro
            else:
                # Try reading from /etc/os-release
                with open('/etc/os-release', 'r') as f:
                    for line in f:
                        if line.startswith('PRETTY_NAME='):
                            distro = line.split('=')[1].strip().strip('"')
                            return distro
                return "Linux"
        except:
            return "Linux"
    
    elif system == "Windows":
        try:
            # Get Windows version info
            result = subprocess.run(['wmic', 'os', 'get', 'Caption'], 
                                  capture_output=True, text=True, timeout=5)
            if result.returncode == 0:
                lines = result.stdout.strip().split('\n')
                if len(lines) > 1:
                    return lines[1].strip()
                else:
                    return "Windows"
            else:
                return "Windows"
        except:
            return "Windows"
    
    else:
        return system

def get_memory_info():
    """Get memory information."""
    try:
        if platform.system() == "Darwin":  # macOS
            result = subprocess.run(['sysctl', '-n', 'hw.memsize'], 
                                  capture_output=True, text=True, timeout=5)
            if result.returncode == 0:
                mem_bytes = int(result.stdout.strip())
                mem_gb = mem_bytes // (1024**3)
                return f"{mem_gb}GB RAM"
        elif platform.system() == "Linux":
            with open('/proc/meminfo', 'r') as f:
                for line in f:
                    if line.startswith('MemTotal:'):
                        mem_kb = int(line.split()[1])
                        mem_gb = mem_kb // (1024**2)
                        return f"{mem_gb}GB RAM"
        elif platform.system() == "Windows":
            result = subprocess.run(['wmic', 'computersystem', 'get', 'TotalPhysicalMemory'], 
                                  capture_output=True, text=True, timeout=5)
            if result.returncode == 0:
                lines = result.stdout.strip().split('\n')
                if len(lines) > 1:
                    mem_bytes = int(lines[1].strip())
                    mem_gb = mem_bytes // (1024**3)
                    return f"{mem_gb}GB RAM"
    except:
        pass
    
    return ""

def get_synnax_version():
    """Get the current Synnax version from the VERSION file."""
    try:
        # Try to read from the VERSION file in the synnax package
        version_file = "../../../synnax/pkg/version/VERSION"
        with open(version_file, 'r') as f:
            version = f.read().strip()
            return version
    except:
        try:
            # Fallback: try to get version from git tags
            result = subprocess.run(['git', 'describe', '--tags', '--abbrev=0'], 
                                  capture_output=True, text=True, timeout=5)
            if result.returncode == 0:
                version = result.stdout.strip()
                # Remove 'v' prefix if present
                if version.startswith('v'):
                    version = version[1:]
                return version
        except:
            pass
    
    return "unknown"

class Bench_Latency_Report(TestCase):

    def setup(self) -> None:

        self.Expected_Timeout = 15

        self.report_client = sy.Synnax(
            host=self.SynnaxConnection.server_address,
            port=self.SynnaxConnection.port,
            username=self.SynnaxConnection.username,
            password=self.SynnaxConnection.password,
            secure=self.SynnaxConnection.secure,
        )

        self.STATE_CHANNEL = "bench_state"
        self.CMD_CHANNEL = "bench_command"
        self.STATE = True

        self.loop_start = sy.TimeStamp.now()


        # Just make sure to call super() last!
        super().setup()

    def run(self) -> None:
        """
        Run the test case.
        """

        # Wait for the "response" to start
        time.sleep(3)
        cycles = 0
        times = list()
        loop_start = sy.TimeStamp.now()
        STATE_CHANNEL = self.STATE_CHANNEL
        CMD_CHANNEL = self.CMD_CHANNEL
        BENCH_TIME = sy.TimeSpan.SECOND * 3

        # Set channels here to avoid calling "self"
        try:
            with self.report_client.open_streamer(STATE_CHANNEL) as stream:
                with self.report_client.open_writer(sy.TimeStamp.now(), CMD_CHANNEL) as writer:
                    while sy.TimeStamp.since(loop_start) < BENCH_TIME:
                        start = sy.TimeStamp.now()
                        writer.write(CMD_CHANNEL, self.STATE)
                        value = stream.read()
                        times.append(sy.TimeStamp.since(start))
                        cycles += 1
        
        except Exception as e:
            raise Exception(f"EXCEPTION: {e}")
        
        self._log_message(f"Cycles/second: {cycles / BENCH_TIME.seconds}")


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
        ax1.axhline(y=p90, color="r", linestyle="--", label=f"P90: {p90:.2f}ms")
        ax1.axhline(y=p95, color="g", linestyle="--", label=f"P95: {p95:.2f}ms")
        ax1.axhline(y=p99, color="b", linestyle="--", label=f"P99: {p99:.2f}ms")
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
        max_p90 = 0.4
        max_p95 = 0.45
        max_p99 = 0.55
        max_peak_to_peak_jitter = 2
        max_average_jitter = 0.05

        # Print statistics
        if p90 > max_p90:
            self._log_message(f"P90 is greater than {max_p90}ms (FAILED)")
            self.fail()
        else:
            self._log_message(f"P90: {p90:.2f}ms")
            
        if p95 > max_p95:
            self._log_message(f"P95 is greater than {max_p95}ms (FAILED)")
            self.fail()
        else:
            self._log_message(f"P95: {p95:.2f}ms")

        if p99 > max_p99:
            self._log_message(f"P99 is greater than {max_p99}ms (FAILED)")
            self.fail()
        else:
            self._log_message(f"P99: {p99:.2f}ms")

        if peak_to_peak_jitter > max_peak_to_peak_jitter:
            self._log_message(f"Peak-to-peak jitter is greater than {max_peak_to_peak_jitter}ms (FAILED)")  
            self.fail()
        else:
            self._log_message(f"Peak-to-peak jitter: {peak_to_peak_jitter:.2f}ms")
        
        if average_jitter > max_average_jitter:
            self._log_message(f"Average jitter is greater than {max_average_jitter}ms (FAILED)")
            self.fail()
        else:
            self._log_message(f"Average jitter: {average_jitter:.2f}ms")

        plt.savefig("bench_latency_load.jpg", dpi=300, bbox_inches='tight')
        plt.close(fig)  # Close the figure to free memory

        