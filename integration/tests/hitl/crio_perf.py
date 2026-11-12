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

import numpy as np
from pydantic import ValidationError

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.wait_accuracy_samples import _WaitSamples
from tests.driver.task import status_text
from tests.hitl import latency
from tests.hitl.crio_arc import _ArcOnCRIO
from tests.hitl.crio_ni import CRIOAnalogRead

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
# Up to the 250 kS/s of the NI 9205 across four inputs.
_RATES_HZ = [10_000, 25_000, 50_000, 62_500]
_ACQUISITION_S = 10
_STOP_STATUS_TIMEOUT_S = 5


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
                    streamer, "crio_rt_out", value, timeout_s=_REPLY_TIMEOUT_S
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


class CRIOAnalogReadRate(CRIOAnalogRead):
    """Acquires four inputs of the NI 9205 at rising rates. At each rate, the Driver
    must send no warning or error, and the samples must fill the whole run with no
    gap."""

    task_name = "HITL cRIO Analog Read Rate"

    def run(self) -> None:
        if self.tsk is None:
            self.fail("Task not configured")
        config = self.tsk.config
        assert isinstance(config, sy.task.ReadConfig)
        for rate in _RATES_HZ:
            config.sample_rate = sy.Rate(rate)
            self.client.tasks.configure(self.tsk)
            problems = self._acquire(rate)
            if problems:
                self.fail(f"At {rate} Hz: {'; '.join(problems)}")
            self.metrics["max_gap_free_rate_hz"] = rate

    def _acquire(self, rate: int) -> list[str]:
        before = sy.TimeStamp.now()
        with self.client.open_streamer(["sy_status_set"]) as statuses:
            assert self.tsk is not None
            with self.tsk.run():
                started = sy.TimeStamp.now()
                sy.sleep(_ACQUISITION_S)  # noqa: TID251 fixed sample window
                stopping = sy.TimeStamp.now()
            problems = self._read_until_stopped(statuses)
        stamps = (
            self.client.read(
                sy.TimeRange(before, sy.TimeStamp.MAX), "hitl_crio_ai_time"
            )
            .to_numpy()
            .astype(np.int64)
        )
        # The hardware runs from before started to after stopping. The Driver reads in
        # chunks of one stream period, and a stop can land partway through one.
        span = stopping - started - self.STREAM_RATE.period
        if len(stamps) < (expected := int(rate * span.seconds)):
            return [*problems, f"{len(stamps)} samples, not the {expected} of the run"]
        gap = sy.TimeSpan(int(np.diff(stamps).max()))
        if gap > 1.5 * sy.Rate(rate).period:
            problems.append(f"a gap of {gap} between samples")
        self.log(f"{rate} Hz: {len(stamps)} of {expected} samples, max gap {gap}")
        return problems

    def _read_until_stopped(self, statuses: sy.Streamer) -> list[str]:
        """Reads statuses until the task reports that it stopped. Returns the warnings
        and errors of the task."""
        assert self.tsk is not None
        problems: list[str] = []
        stopped = False
        deadline = time.perf_counter() + _STOP_STATUS_TIMEOUT_S
        while not stopped and (remaining := deadline - time.perf_counter()) > 0:
            if (frame := statuses.read(timeout=remaining)) is None:
                continue
            for raw in frame["sy_status_set"]:
                try:
                    status = sy.task.Status.model_validate(raw)
                except ValidationError:
                    continue
                if status.owner != sy.task.resource_id(self.tsk.key):
                    continue
                if status.variant in ("warning", "error"):
                    problems.append(f"{status.variant}: {status_text(status)}")
                if status.details is not None and not status.details.running:
                    stopped = True
        if not stopped:
            self.fail(f"No stop status in {_STOP_STATUS_TIMEOUT_S} s")
        return problems
