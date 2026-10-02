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

import synnax as sy

# The shortest wait in ms each runtime holds inside the error limit, by OS.
MIN_WAIT_MS: dict[str, dict[str, int]] = {
    "C++": {"Linux": 1, "Darwin": 1, "Windows": 5},
    "Go": {"Linux": 1, "Darwin": 1, "Windows": 1},
}


def runtime(rack: sy.Rack) -> str:
    """Returns the name of the Arc runtime that runs the tasks of ``rack``.

    :param rack: The rack that holds the Arc task.
    """
    # The Core has two embedded racks. The one that is not the Driver runs Go.
    if rack.embedded and not rack.name.endswith("Embedded Driver"):
        return "Go"
    return "C++"


def min_wait_ms(rack: sy.Rack) -> int:
    """Returns the shortest wait in ms the runtime of ``rack`` supports. The OS is
    that of the test host, which also runs the Core and its Driver.

    :param rack: The rack that holds the Arc task.
    """
    return MIN_WAIT_MS[runtime(rack)][platform.system()]
