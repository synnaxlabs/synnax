#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The timing limits of the Arc runtimes."""

import platform
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
    # SY-5052: a Go wake is 0.2 to 0.4 ms late, and 1 ms late on macOS. A spin to the
    # deadline in the Go loop would remove most of it and lower these shortest waits.
    "Go": {
        "Linux": Limits(min_wait_ms=10, max_error_percent=3.0, max_spread_ms=1.25),
        "Darwin": Limits(min_wait_ms=20, max_error_percent=6.0, max_spread_ms=2.0),
        "Windows": Limits(min_wait_ms=10, max_error_percent=4.5, max_spread_ms=1.25),
    },
}

# The limits of waits of 50 ms or more, measured the same way. On Windows, the C++
# runtime does not spin before these waits, so its timer is up to 0.7 ms late.
LONG_LIMITS: dict[str, dict[str, Limits]] = {
    "C++": {
        "Linux": Limits(min_wait_ms=50, max_error_percent=0.5, max_spread_ms=0.1),
        "Darwin": Limits(min_wait_ms=50, max_error_percent=0.5, max_spread_ms=1.5),
        "Windows": Limits(min_wait_ms=50, max_error_percent=1.0, max_spread_ms=1.0),
    },
    "Go": {
        "Linux": Limits(min_wait_ms=50, max_error_percent=1.5, max_spread_ms=1.5),
        "Darwin": Limits(min_wait_ms=50, max_error_percent=3.0, max_spread_ms=2.0),
        "Windows": Limits(min_wait_ms=50, max_error_percent=1.5, max_spread_ms=1.25),
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


def limits(rack: sy.Rack, table: dict[str, dict[str, Limits]] = LIMITS) -> Limits:
    """Returns the timing limits of the runtime of ``rack``. The OS is that of the test
    host, which also runs the Core and its Driver.

    :param rack: The rack that holds the Arc task.
    :param table: The limits of each runtime and OS.
    :returns: The limits of the runtime of ``rack`` on the OS of the test host.
    """
    return table[runtime(rack)][platform.system()]
