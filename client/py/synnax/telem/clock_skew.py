#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Callable

from synnax.telem.telem import TimeSpan, TimeStamp


class ClockSkewCalculator:
    """Calculates and tracks clock skew between two systems using a midpoint
    synchronization algorithm.
    """

    now: Callable[[], TimeStamp]
    local_start_t: TimeStamp
    accumulated_skew: int
    n: int

    def __init__(self, now: Callable[[], TimeStamp] = TimeStamp.now) -> None:
        self.now = now
        self.local_start_t = TimeStamp(0)
        self.accumulated_skew = 0
        self.n = 0

    def start(self) -> None:
        """Starts a new clock skew measurement."""
        self.local_start_t = self.now()

    def end(self, remote_midpoint: TimeStamp) -> None:
        """Completes a clock skew measurement.

        Uses the midpoint method: local_midpoint = start + (end - start) / 2.
        The skew is then: local_midpoint - remote_midpoint.
        """
        local_end = self.now()
        local_mid = (
            int(self.local_start_t) + (int(local_end) - int(self.local_start_t)) // 2
        )
        skew = local_mid - int(remote_midpoint)
        self.accumulated_skew += skew
        self.n += 1

    def skew(self) -> TimeSpan:
        """Returns the average clock skew across all measurements."""
        if self.n == 0:
            return TimeSpan.ZERO
        return TimeSpan(self.accumulated_skew // self.n)

    def exceeds(self, threshold: TimeSpan) -> bool:
        """Checks if the absolute average clock skew exceeds a threshold."""
        return abs(self.skew()) > threshold
