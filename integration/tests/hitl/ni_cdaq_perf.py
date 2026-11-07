#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Performance of the Driver with the real cDAQ-9185 (SY2) of the HITL station."""

import time

import numpy as np
from pydantic import ValidationError

import synnax as sy
from tests.hitl import latency
from tests.hitl.ni_cdaq import CDAQAnalogRead, CDAQDigitalWrite

_COMMANDS = 500
# The first commands are slow while the Core opens its relay to the Driver.
_WARMUP = 20
_STATE_TIMEOUT_S = 5
# About three times what the station measures.
_MAX_MEDIAN_MS = 6
_MAX_P99_MS = 10
_CMD = "hitl_cdaq_do_cmd_0"
_STATE = "hitl_cdaq_do_state_0"
# Up to the 250 kS/s of the NI 9205 across four inputs.
_RATES_HZ = [10_000, 25_000, 50_000, 62_500]
_ACQUISITION_S = 10


class CDAQDigitalWriteLatency(CDAQDigitalWrite):
    """Measures the time from a command of this machine's client to the state that the
    Driver sends once it writes the output of the NI 9375."""

    task_name = "HITL cDAQ Digital Write Latency"

    def run(self) -> None:
        if self.tsk is None:
            self.fail("Task not configured")
        latencies_ms: list[float] = []
        with (
            self.tsk.run(),
            self.client.open_streamer(_STATE) as streamer,
            self.client.open_writer(
                sy.TimeStamp.now(),
                [_CMD, "hitl_cdaq_do_cmd_time"],
                enable_auto_commit=True,
            ) as writer,
        ):
            for i in range(_WARMUP + _COMMANDS):
                value = 1 - i % 2
                start = time.perf_counter()
                writer.write({_CMD: value, "hitl_cdaq_do_cmd_time": sy.TimeStamp.now()})
                if not latency.wait_for(
                    streamer, _STATE, value, timeout_s=_STATE_TIMEOUT_S
                ):
                    self.fail(f"No state {value} in {_STATE_TIMEOUT_S} s")
                latencies_ms.append((time.perf_counter() - start) * 1000)
        latency.check(
            self,
            "command_to_state",
            latencies_ms[_WARMUP:],
            max_median_ms=_MAX_MEDIAN_MS,
            max_p99_ms=_MAX_P99_MS,
        )


class CDAQAnalogReadRate(CDAQAnalogRead):
    """Acquires four inputs of the NI 9205 at rising rates. At each rate, the Driver
    must send no warning or error, and the stamps of the samples must have no gap."""

    task_name = "HITL cDAQ Analog Read Rate"

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
            self.log(f"{rate} Hz: no gap")

    def _acquire(self, rate: int) -> list[str]:
        assert self.tsk is not None
        problems: list[str] = []
        start = sy.TimeStamp.now()
        with self.client.open_streamer(["sy_status_set"]) as statuses:
            with self.tsk.run():
                sy.sleep(_ACQUISITION_S)
            while (frame := statuses.read(timeout=0)) is not None:
                for raw in frame["sy_status_set"]:
                    try:
                        status = sy.task.Status.model_validate(raw)
                    except ValidationError:
                        continue
                    if (
                        status.details is not None
                        and status.details.task == self.tsk.key
                        and status.variant in ("warning", "error")
                    ):
                        problems.append(f"{status.variant}: {status.message}")
        stamps = self.client.read(
            sy.TimeRange(start, sy.TimeStamp.now()), "hitl_cdaq_ai_time"
        ).to_numpy()
        if len(stamps) < rate:
            problems.append(f"{len(stamps)} samples in {_ACQUISITION_S} s")
        else:
            gap = sy.TimeSpan(int(np.diff(stamps.astype(np.int64)).max()))
            if gap > 1.5 * sy.Rate(rate).period:
                problems.append(f"a gap of {gap} between samples")
        return problems
