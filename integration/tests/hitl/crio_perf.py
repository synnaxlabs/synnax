#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Performance of the cRIO's Driver, its real-time Arc runtime, and its NI tasks."""

import time
from functools import partial
from operator import eq

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.wait_accuracy_samples import _WaitSamples
from tests.hitl import latency
from tests.hitl.crio_arc import _ArcOnCRIO
from tests.hitl.crio_ni import CRIOAnalogRead
from tests.hitl.read_rate import ReadRateCase

ECHO_SOURCE = """
func echo() {x f64} f64 {
    return x
}
crio_rt_in -> echo() -> crio_rt_out
"""

_ROUND_TRIPS = 500
# The first round trips are slow while the Core opens its relay to the cRIO.
_WARMUP = 20
_REPLY_TIMEOUT_S = 5
# About three times what the station measures. The maximum varies too much for a limit.
_MAX_MEDIAN_MS = 2
_MAX_P99_MS = 15


class CRIORoundTrip(_ArcOnCRIO):
    """Measures the time from a write of this machine's client to the reply of an Arc
    program on the cRIO, through the Core both ways."""

    arc_source = ECHO_SOURCE
    arc_name_prefix = "CRIORoundTrip"
    start_cmd_channel = "crio_rt_start_cmd"
    subscribe_channels = ["crio_rt_out"]

    def setup(self) -> None:
        create_virtual_channel(self.client, "crio_rt_in", data_type=sy.DataType.FLOAT64)
        create_indexed_pair(self.client, "crio_rt_out", data_type=sy.DataType.FLOAT64)
        super().setup()

    def verify_sequence_execution(self) -> None:
        round_trips_ms: list[float] = []
        with (
            self.client.open_streamer("crio_rt_out") as streamer,
            self.client.open_writer(sy.TimeStamp.now(), "crio_rt_in") as writer,
        ):
            for value in range(_WARMUP + _ROUND_TRIPS):
                start = time.perf_counter()
                writer.write("crio_rt_in", float(value))
                if not latency.wait_for(
                    streamer,
                    "crio_rt_out",
                    partial(eq, value),
                    timeout_s=_REPLY_TIMEOUT_S,
                ):
                    self.fail(f"No reply to {value} in {_REPLY_TIMEOUT_S} s")
                round_trips_ms.append((time.perf_counter() - start) * 1000)
        latency.check(
            self,
            "round_trip",
            round_trips_ms[_WARMUP:],
            max_median_ms=_MAX_MEDIAN_MS,
            max_p99_ms=_MAX_P99_MS,
        )


class CRIOWaitSamples(_ArcOnCRIO, _WaitSamples):
    """Measures the waits of a sequence on the cRIO's real-time runtime."""

    profile_ms = [20, 10, 5, 1]
    driver_os = "NI Linux RT"


class CRIOAnalogReadRate(ReadRateCase[sy.ni.AnalogReadTask], CRIOAnalogRead):
    """Acquires four inputs of the NI 9205 at rising rates."""

    task_name = "HITL cRIO Analog Read Rate"
    # Up to the 250 kS/s of the NI 9205 across four inputs.
    RATES_HZ = [10_000, 25_000, 50_000, 62_500]
