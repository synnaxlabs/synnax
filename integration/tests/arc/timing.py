#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The timing limits of the Arc runtimes."""

from dataclasses import dataclass

import synnax as sy


@dataclass(frozen=True)
class Limits:
    """The timing limits of one Arc runtime on one OS."""

    #: The shortest wait in ms the limits cover.
    min_wait_ms: int
    #: The limit on the median error of a wait, in percent of the wait.
    max_error_percent: float
    #: The limit in ms on the span from the shortest to the longest held time of a wait.
    max_spread_ms: float


# Each limit is the worst value measured at the shortest wait plus a margin.
LIMITS: dict[str, dict[str, Limits]] = {
    "C++": {
        "Linux": Limits(min_wait_ms=1, max_error_percent=3.0, max_spread_ms=0.2),
        # EVENT_DRIVEN does not spin on macOS. Its timer is rarely up to 1.2 ms late.
        "Darwin": Limits(min_wait_ms=1, max_error_percent=2.5, max_spread_ms=1.5),
        "Windows": Limits(min_wait_ms=5, max_error_percent=0.5, max_spread_ms=0.1),
    },
    "Go": {
        "Linux": Limits(min_wait_ms=1, max_error_percent=8.0, max_spread_ms=0.3),
        # A 1 ms Go wait on macOS is up to 7% late.
        "Darwin": Limits(min_wait_ms=5, max_error_percent=2.0, max_spread_ms=1.25),
        "Windows": Limits(min_wait_ms=1, max_error_percent=3.0, max_spread_ms=0.5),
    },
}

# The limits of waits of 50 ms or more, measured the same way.
LONG_LIMITS: dict[str, dict[str, Limits]] = {
    "C++": {
        "Linux": Limits(min_wait_ms=50, max_error_percent=0.5, max_spread_ms=0.1),
        "Darwin": Limits(min_wait_ms=50, max_error_percent=0.5, max_spread_ms=1.5),
        "Windows": Limits(min_wait_ms=50, max_error_percent=1.0, max_spread_ms=1.0),
    },
    "Go": {
        "Linux": Limits(min_wait_ms=50, max_error_percent=0.5, max_spread_ms=0.3),
        "Darwin": Limits(min_wait_ms=50, max_error_percent=0.5, max_spread_ms=2.0),
        "Windows": Limits(min_wait_ms=50, max_error_percent=0.5, max_spread_ms=0.5),
    },
}


def runtime(rack: sy.Rack) -> str:
    """Returns the name of the Arc runtime that runs the tasks of ``rack``.

    :param rack: The rack that holds the Arc task.
    :returns: "Go" or "C++".
    """
    # The Core has two embedded racks. The one that is not the Driver runs Go.
    if rack.embedded and not rack.name.endswith("Embedded Driver"):
        return "Go"
    return "C++"
