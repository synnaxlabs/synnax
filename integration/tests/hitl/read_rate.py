#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The highest rate at which a read task of the HITL station acquires with no gap."""

import time

import numpy as np
from pydantic import ValidationError

import synnax as sy
from tests.driver.task import ReadTaskCase, ReadTaskT, status_text

_ACQUISITION_S = 10
_STOP_STATUS_TIMEOUT_S = 5


class ReadRateCase(ReadTaskCase[ReadTaskT]):
    """Acquires at each rate of RATES_HZ in turn. At each rate, the Driver must send no
    warning or error, and the samples must fill the whole run with no gap. Records the
    highest rate in the metric max_gap_free_rate_hz."""

    RATES_HZ: list[int]

    def run(self) -> None:
        if self.tsk is None:
            self.fail("Task not configured")
        config = self.tsk.config
        assert isinstance(config, sy.task.ReadConfig)
        for rate in self.RATES_HZ:
            config.sample_rate = sy.Rate(rate)
            self.client.tasks.configure(self.tsk)
            problems = self._acquire(rate)
            if problems:
                self.fail(f"At {rate} Hz: {'; '.join(problems)}")
            self.metrics["max_gap_free_rate_hz"] = rate

    def _acquire(self, rate: int) -> list[str]:
        assert self.tsk is not None
        index = self.client.channels.retrieve(self._channel_keys(self.tsk)[0]).index
        before = sy.TimeStamp.now()
        with self.client.open_streamer(["sy_status_set"]) as statuses:
            with self.tsk.run():
                started = sy.TimeStamp.now()
                sy.sleep(_ACQUISITION_S)  # noqa: TID251 fixed sample window
                stopping = sy.TimeStamp.now()
            problems = self._read_until_stopped(statuses)
        stamps = (
            self.client.read(sy.TimeRange(before, sy.TimeStamp.MAX), index)
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
