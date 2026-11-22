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
from typing import Protocol

from x.telem import CrudeTimeSpan, TimeSpan
from x.timing import Timer as Stopwatch


class Timer(Protocol):
    """A wait that passes once, after a span."""

    def remaining(self) -> TimeSpan:
        """Returns how long the wait has left. It never grows, and it is zero once the
        wait has passed.
        """
        ...


Start = Callable[[TimeSpan], Timer]
"""Starts a Timer that passes after the span."""


class _System:
    def __init__(self, span: TimeSpan) -> None:
        self._span = span
        self._stopwatch = Stopwatch()

    def remaining(self) -> TimeSpan:
        return max(self._span - self._stopwatch.elapsed(), TimeSpan.ZERO)


def start(span: CrudeTimeSpan) -> Timer:
    """Starts a Timer on the system clock. A bare number is seconds."""
    return _System(TimeSpan.from_seconds(span))
