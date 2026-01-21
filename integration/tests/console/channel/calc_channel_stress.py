#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import atexit
import os
import signal
import subprocess
import sys
from pathlib import Path
from typing import Any

import synnax as sy

from console.case import ConsoleCase
from console.plot import Plot

CALC_CHANNELS = [
    "calc_avg_sum_div_50_sine",
    "calc_avg_explicit_50_sine",
    "calc_avg_pairwise_50_sine",
]

INTEGRATION_DIR = Path(__file__).resolve().parent.parent.parent.parent
SCRIPT_PATH = (
    INTEGRATION_DIR.parent
    / "client"
    / "py"
    / "examples"
    / "dev"
    / "calc_channel_stress.py"
)


class CalcChannelStress(ConsoleCase):
    """Test calculated channel stress at various rates."""

    process: subprocess.Popen[bytes] | None = None

    def __init__(self, *, rate: int = 10, **params: Any) -> None:
        params.pop("name", None)
        super().__init__(name=f"calc_stress_{rate}hz", **params)
        self.rate = rate

    def setup(self) -> None:
        super().setup()
        self.start_stress_script()

    def start_stress_script(self) -> None:
        """Start the calc_channel_stress.py script as a subprocess."""
        env = os.environ.copy()
        env["PYTHONUNBUFFERED"] = "1"

        if not SCRIPT_PATH.exists():
            raise FileNotFoundError(f"Stress script not found at: {SCRIPT_PATH}")

        self.process = subprocess.Popen(
            [sys.executable, str(SCRIPT_PATH), "--rate", str(self.rate)],
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        atexit.register(self.cleanup_process)
        self.log(f"Started stress script with PID {self.process.pid} at {self.rate} Hz")

        exit_code = self.process.poll()
        if exit_code is not None:
            stdout, stderr = self.process.communicate()
            self.log(f"Script exited early with code {exit_code}")
            if stderr:
                self.log(f"Script stderr: {stderr.decode()[:500]}")
            raise RuntimeError(f"Stress script failed to start: exit code {exit_code}")

    def cleanup_process(self) -> None:
        """Terminate the stress script subprocess."""
        if self.process is None:
            return

        if self.process.poll() is not None:
            self.process = None
            return

        try:
            if sys.platform == "win32":
                self.process.terminate()
            else:
                self.process.send_signal(signal.SIGTERM)
            self.process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait(timeout=2)
        except Exception as e:
            self.log(f"Error terminating stress script: {e}")
        finally:
            self.process = None

    def teardown(self) -> None:
        self.cleanup_process()
        super().teardown()

    def run(self) -> None:
        """Run calculated channel stress test."""
        self.test_plot_calc_channels()

    def test_plot_calc_channels(self) -> None:
        """Test plotting calculated channels from the stress script."""
        self.log(f"Testing plot calculated channels at {self.rate} Hz")

        client = self.client
        console = self.console

        self.wait_for_channels(CALC_CHANNELS)

        plot = Plot(client, console, f"Calc Stress {self.rate}Hz")
        plot.add_channels("Y1", CALC_CHANNELS)

        for ch in CALC_CHANNELS:
            assert ch in plot.data["Y1"], f"Channel {ch} should be on Y1"

        sy.sleep(1)

        csv_content = plot.download_csv()

        for ch in CALC_CHANNELS:
            assert ch in csv_content, f"CSV should contain {ch}"

        lines = csv_content.strip().split("\n")
        assert len(lines) > 1, "CSV should have header and data rows"
        data_rows = len(lines) - 1
        header = lines[0].split(",")

        self.log(f"CSV: {data_rows} rows, {len(header)} columns")

        for ch in CALC_CHANNELS:
            assert ch in header, f"Channel {ch} should be in CSV header"
            idx = header.index(ch)
            non_empty = sum(1 for line in lines[1:] if line.split(",")[idx].strip())
            assert non_empty > 0, f"Channel {ch} should have non-empty values"
            self.log(f"Channel {ch}: {non_empty}/{data_rows} non-empty values")

        plot.close()
        self.log(f"Successfully verified calculated channels at {self.rate} Hz")

    def wait_for_channels(self, channels: list[str], timeout: float = 5) -> None:
        """Wait for channels to be available in the server."""
        self.log(f"Waiting for channels: {channels}")
        start = sy.TimeStamp.now()
        timeout_ns = timeout * sy.TimeSpan.SECOND

        while True:
            elapsed = sy.TimeStamp.now() - start
            if elapsed > timeout_ns:
                raise TimeoutError(
                    f"Channels not available after {timeout}s: {channels}"
                )

            try:
                retrieved = self.client.channels.retrieve(channels)
                if isinstance(retrieved, list) and len(retrieved) == len(channels):
                    self.log("All channels available")
                    return
            except Exception:
                pass
