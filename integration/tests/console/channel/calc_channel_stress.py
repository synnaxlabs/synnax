#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from typing import Any

import numpy as np
import synnax as sy

from console.case import ConsoleCase
from console.plot import Plot

CrudeFrame = dict[int, sy.TimeStamp | float | np.floating]

CALC_CHANNELS = [
    "calc_avg_sum_div_50_sine",
    "calc_avg_explicit_50_sine",
    "calc_avg_pairwise_50_sine",
]

WRAP_THRESHOLD = 100.0


class CalcChannelStress(ConsoleCase):
    """Test calculated channel stress at various rates."""

    _stress_thread: threading.Thread | None = None
    _stop_event: threading.Event

    def __init__(self, *, rate: int = 10, **params: Any) -> None:
        params.pop("name", None)
        super().__init__(name=f"calc_stress_{rate}hz", **params)
        self.rate = rate
        self._stop_event = threading.Event()

    def setup(self) -> None:
        super().setup()
        self._start_stress_writer()

    def _create_multiple_channels(
        self, base_name: str, count: int, index_key: int
    ) -> list[sy.Channel]:
        return [
            self.client.channels.create(
                name=f"{base_name}_{i + 1}",
                index=index_key,
                data_type=sy.DataType.FLOAT32,
                retrieve_if_name_exists=True,
            )
            for i in range(count)
        ]

    def _start_stress_writer(self) -> None:
        """Start the stress writer in a background thread."""
        client = self.client

        timestamp_channel = client.channels.create(
            name="timestamp_channel",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        constant_channels = self._create_multiple_channels(
            "constant_value_channel", 2, timestamp_channel.key
        )
        sine_channels = self._create_multiple_channels(
            "sine_wave_channel", 2, timestamp_channel.key
        )
        cosine_channel = client.channels.create(
            name="cosine_wave_channel",
            index=timestamp_channel.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )
        linear_channels = self._create_multiple_channels(
            "linear_function_channel", 3, timestamp_channel.key
        )
        sine_50_channels = self._create_multiple_channels(
            "sine_50_channel", 50, timestamp_channel.key
        )
        linear_50_channels = self._create_multiple_channels(
            "linear_50_channel", 50, timestamp_channel.key
        )

        # Different average calculations
        calc_avg_sum_div_50_sine = client.channels.create(
            name="calc_avg_sum_div_50_sine",
            data_type=sy.DataType.FLOAT32,
            expression="return ("
            + " + ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
            + ") / 50.0",
            retrieve_if_name_exists=True,
        )

        running_sum = " + ".join([f"sine_50_channel_{i + 1}" for i in range(50)])
        calc_avg_explicit_50_sine = client.channels.create(
            name="calc_avg_explicit_50_sine",
            data_type=sy.DataType.FLOAT32,
            expression=f"return ({running_sum}) / 50.0",
            retrieve_if_name_exists=True,
        )

        pairs = [
            f"(sine_50_channel_{i * 2 + 1} + sine_50_channel_{i * 2 + 2})/2.0"
            for i in range(25)
        ]
        pairwise_avg = "(" + " + ".join(pairs) + ") / 25.0"
        calc_avg_pairwise_50_sine = client.channels.create(
            name="calc_avg_pairwise_50_sine",
            data_type=sy.DataType.FLOAT32,
            expression=f"return {pairwise_avg}",
            retrieve_if_name_exists=True,
        )

        all_channels = (
            [timestamp_channel]
            + constant_channels
            + sine_channels
            + [cosine_channel]
            + linear_channels
            + sine_50_channels
            + linear_50_channels
            + [
                calc_avg_sum_div_50_sine,
                calc_avg_explicit_50_sine,
                calc_avg_pairwise_50_sine,
            ]
        )

        def write_loop() -> None:
            loop = sy.Loop(sy.Rate.HZ * self.rate)
            with client.open_writer(
                sy.TimeStamp.now(),
                channels=[ch.key for ch in all_channels],
            ) as writer:
                i = 0
                while loop.wait() and not self._stop_event.is_set():
                    current_time = sy.TimeStamp.now()
                    data_to_write: CrudeFrame = {timestamp_channel.key: current_time}

                    for j, ch in enumerate(constant_channels):
                        data_to_write[ch.key] = 42.0 + j * 58.0

                    for j, ch in enumerate(sine_channels):
                        data_to_write[ch.key] = (j + 5) * np.sin(i / 10.0)

                    data_to_write[cosine_channel.key] = np.cos(i / 10.0)

                    for j, ch in enumerate(linear_channels):
                        data_to_write[ch.key] = ((j + 0.5) * i + j * 5) % WRAP_THRESHOLD

                    for j, ch in enumerate(sine_50_channels):
                        data_to_write[ch.key] = 5 * np.sin(
                            i / 10.0 + j * (2 * np.pi / 50)
                        )

                    for j, ch in enumerate(linear_50_channels):
                        data_to_write[ch.key] = ((j + 1) * 0.1 * i + j) % WRAP_THRESHOLD

                    writer.write(data_to_write)
                    i += 1

        self._stress_thread = threading.Thread(target=write_loop, daemon=True)
        self._stress_thread.start()
        self.log(f"Started stress writer thread at {self.rate} Hz")

    def _stop_stress_writer(self) -> None:
        """Stop the stress writer thread."""
        if self._stress_thread is None:
            return

        self._stop_event.set()
        self._stress_thread.join(timeout=5)
        if self._stress_thread.is_alive():
            self.log("Warning: stress writer thread did not stop cleanly")
        self._stress_thread = None

    def teardown(self) -> None:
        self._stop_stress_writer()
        super().teardown()

    def run(self) -> None:
        """Run calculated channel stress test."""

        self.log(f"Testing plot calculated channels at {self.rate} Hz")

        client = self.client
        console = self.console

        self.log("Waiting for calculated channels to appear in console")
        channels_exist = console.channels.wait_for_channels(CALC_CHANNELS, timeout=15.0)
        if not channels_exist:
            available = console.channels.list_all()
            raise RuntimeError(
                f"Calculated channels did not appear in console within timeout. "
                f"Available channels: {available}"
            )
        self.log("All calculated channels are now visible in console")

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
