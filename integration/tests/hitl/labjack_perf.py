#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Performance of the Driver with the LabJack T4 of the HITL station."""

import time
from collections.abc import Callable

import synnax as sy
from tests.hitl import latency
from tests.hitl.labjack import LabJackAnalogLoopback, LabJackAnalogRead
from tests.hitl.read_rate import ReadRateCase

_COMMANDS = 500
# The first commands are slow while the Core opens its relays to the Driver.
_WARMUP = 20
_INPUT_TIMEOUT_S = 5
# DAC0 steps between the two levels, and AIN0 reads a step once it crosses the middle.
_LOW_V = 0.5
_HIGH_V = 3.5
_MIDDLE_V = (_LOW_V + _HIGH_V) / 2
# About three times what the station measures: 4 ms at the median and 7 ms at p99.
_MAX_MEDIAN_MS = 12
_MAX_P99_MS = 20


def _above_middle(volts: float) -> bool:
    return volts > _MIDDLE_V


def _below_middle(volts: float) -> bool:
    return volts < _MIDDLE_V


class LabJackAnalogReadRate(ReadRateCase[sy.labjack.ReadTask], LabJackAnalogRead):
    """Streams the four high-voltage inputs of the T4 at rising rates."""

    task_name = "HITL LabJack Analog Read Rate"
    # Up to the 50 kS/s that the T4 streams across four inputs.
    RATES_HZ = [1_000, 5_000, 10_000, 12_500]


class LabJackLoopbackLatency(LabJackAnalogLoopback):
    """Measures the time from a DAC0 command of this machine's client to the first
    frame in which AIN0 reads the new level. The time covers the Core both ways, the
    write to the T4, and up to one stream period."""

    task_name = "HITL LabJack Loopback Latency"
    SAMPLE_RATE = 5000 * sy.Rate.HZ
    STREAM_RATE = 500 * sy.Rate.HZ

    def drive(
        self,
        streamer: sy.Streamer,
        inputs: list[int],
        command: Callable[[float], None],
    ) -> None:
        latencies_ms: list[float] = []
        for i in range(_WARMUP + _COMMANDS):
            level, reached = (
                (_HIGH_V, _above_middle) if i % 2 == 0 else (_LOW_V, _below_middle)
            )
            start = time.perf_counter()
            command(level)
            if not latency.wait_for(
                streamer, inputs[0], reached, timeout_s=_INPUT_TIMEOUT_S
            ):
                self.fail(f"AIN0 did not read {level} V in {_INPUT_TIMEOUT_S} s")
            latencies_ms.append((time.perf_counter() - start) * 1000)
        latency.check(
            self,
            "command_to_input",
            latencies_ms[_WARMUP:],
            max_median_ms=_MAX_MEDIAN_MS,
            max_p99_ms=_MAX_P99_MS,
        )
