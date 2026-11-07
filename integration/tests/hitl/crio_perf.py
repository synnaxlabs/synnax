#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Performance of the cRIO's Driver and its real-time Arc runtime. No test needs C
Series modules."""

import time

import numpy as np

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.wait_accuracy_samples import _WaitSamples
from tests.hitl.crio_arc import _ArcOnCRIO

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
                while True:
                    frame = streamer.read(timeout=_REPLY_TIMEOUT_S)
                    if frame is None:
                        self.fail(f"No reply to {value} in {_REPLY_TIMEOUT_S} s")
                    if frame["crio_rt_out"].to_numpy()[-1] == value:
                        break
                round_trips_ms.append((time.perf_counter() - start) * 1000)
        measured = np.array(round_trips_ms[_WARMUP:])
        self.metrics["round_trip_median_ms"] = float(np.median(measured))
        self.metrics["round_trip_p99_ms"] = float(np.percentile(measured, 99))
        self.metrics["round_trip_max_ms"] = float(measured.max())
        self.log(f"round trip in ms: {self.metrics}")


class CRIOWaitSamples(_ArcOnCRIO, _WaitSamples):
    """Measures the waits of a sequence on the cRIO's real-time runtime."""

    profile_ms = [20, 10, 5, 1]
    driver_os = "NI Linux RT"
