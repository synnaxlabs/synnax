#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
import time
from functools import partial
from operator import eq

import pytest

import synnax as sy
from framework.log_client import LogClient
from framework.models import STATUS
from framework.test_case import TestCase
from framework.tests.fake import Stream
from tests.hitl import latency

CHANNEL = "state"


def frame(*values: float) -> sy.Frame:
    return sy.Frame({CHANNEL: sy.Series(list(values))})


class Case(TestCase):
    """A TestCase that needs no Core."""

    def __init__(self) -> None:
        self.name = "case"
        self._status = STATUS.INITIALIZING
        self._error_message = None
        self.metrics = {}
        self.log_client = LogClient(name=self.name)

    def run(self) -> None:
        pass


class TestWaitFor:
    def test_should_return_true_when_the_last_value_matches(self) -> None:
        stream = Stream()
        stream.push(frame(0.0, 1.0))
        assert latency.wait_for(stream, CHANNEL, partial(eq, 1.0), timeout_s=5)

    def test_should_skip_frames_whose_last_value_differs(self) -> None:
        stream = Stream()
        stream.push(frame(1.0, 0.0))
        stream.push(frame(0.0, 1.0))
        assert latency.wait_for(stream, CHANNEL, partial(eq, 1.0), timeout_s=5)
        assert stream.read(timeout=sy.TimeSpan.from_seconds(0.01)) is None

    def test_should_return_false_at_the_deadline_despite_other_frames(self) -> None:
        stream = Stream()
        stop = threading.Event()

        def push() -> None:
            for _ in range(200):
                if stop.wait(0.005):
                    return
                stream.push(frame(0.0))

        pusher = threading.Thread(target=push)
        pusher.start()
        start = time.perf_counter()
        try:
            assert not latency.wait_for(
                stream, CHANNEL, partial(eq, 1.0), timeout_s=0.05
            )
        finally:
            stop.set()
            pusher.join()
        assert 0.05 <= time.perf_counter() - start < 0.5


class TestCheck:
    def test_should_record_the_median_p99_and_max(self) -> None:
        case = Case()
        latency.check(case, "rt", [1.0] * 99 + [9.0], max_median_ms=2, max_p99_ms=10)
        assert case.metrics == pytest.approx(
            {"rt_median_ms": 1.0, "rt_p99_ms": 1.08, "rt_max_ms": 9.0}
        )

    def test_should_fail_over_the_median_limit(self) -> None:
        with pytest.raises(
            RuntimeError,
            match=r"rt took 3\.00 ms at the median and 3\.00 ms at p99\. The limits "
            r"are 2 ms and 10 ms\.",
        ):
            latency.check(Case(), "rt", [3.0] * 10, max_median_ms=2, max_p99_ms=10)

    def test_should_fail_over_the_p99_limit(self) -> None:
        case = Case()
        with pytest.raises(
            RuntimeError,
            match=r"rt took 1\.00 ms at the median and 20\.00 ms at p99\. The limits "
            r"are 2 ms and 10 ms\.",
        ):
            latency.check(
                case, "rt", [1.0] * 98 + [20.0] * 2, max_median_ms=2, max_p99_ms=10
            )
        assert case.metrics["rt_max_ms"] == 20.0
