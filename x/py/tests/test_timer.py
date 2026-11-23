#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

from x import timer
from x.telem import TimeSpan


class TestStart:
    def test_remaining_starts_at_the_span(self) -> None:
        """A new timer has at most its span left."""
        span = TimeSpan.SECOND * 10
        remaining = timer.start(span).remaining()
        assert TimeSpan.ZERO < remaining <= span

    def test_remaining_reaches_zero_after_the_span(self) -> None:
        """A timer has nothing left once its span passes."""
        t = timer.start(TimeSpan.MILLISECOND * 20)
        time.sleep(0.05)
        assert t.remaining() == TimeSpan.ZERO

    def test_remaining_never_grows(self) -> None:
        """Each read of the remaining time is no more than the one before it."""
        t = timer.start(TimeSpan.SECOND)
        first = t.remaining()
        assert t.remaining() <= first

    def test_bare_number_is_seconds(self) -> None:
        """A bare number is a span in seconds."""
        assert (
            TimeSpan.MILLISECOND * 500 < timer.start(1).remaining() <= TimeSpan.SECOND
        )
