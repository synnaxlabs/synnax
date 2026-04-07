#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections.abc import Callable

from x.telem.telem import TimeSpan, TimeStamp


class ClockSkewCalculator:
    """Calculates and tracks clock skew between two systems using a midpoint
    synchronization algorithm.
    """

    def __init__(self, now: Callable[[], TimeStamp] | None = None) -> None:
        self._now = now or TimeStamp.now
        self._local_start_t = TimeStamp(0)
        self._accumulated_skew = 0
        self._n = 0

    def start(self) -> None:
        self._local_start_t = self._now()

    def end(self, remote_midpoint_t: TimeStamp) -> None:
        local_end_t = self._now()
        start = int(self._local_start_t)
        end = int(local_end_t)
        local_mid = start + (end - start) // 2
        skew = local_mid - int(remote_midpoint_t)
        self._accumulated_skew += skew
        self._n += 1

    @property
    def skew(self) -> TimeSpan:
        if self._n == 0:
            return TimeSpan(0)
        return TimeSpan(self._accumulated_skew // self._n)

    def exceeds(self, threshold: TimeSpan) -> bool:
        return abs(self.skew) > abs(threshold)
