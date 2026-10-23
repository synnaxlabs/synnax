#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Measures whether a series of samples keeps rising, such as a leaking resource."""

import statistics
from collections.abc import Sequence


def slope(times: Sequence[float], values: Sequence[float]) -> float:
    """Returns the Theil-Sen slope of values over times: the median of the slopes
    between every two samples taken at different times. Spikes and periodic swings
    barely move it.

    :raises ValueError: If fewer than two samples have different times, or if times
        and values differ in length.
    """
    if len(times) != len(values):
        raise ValueError(f"Got {len(times)} times but {len(values)} values")
    slopes = [
        (values[j] - values[i]) / (times[j] - times[i])
        for i in range(len(times))
        for j in range(i + 1, len(times))
        if times[j] != times[i]
    ]
    if not slopes:
        raise ValueError("Need two samples taken at different times")
    return statistics.median(slopes)


def rise(times: Sequence[float], values: Sequence[float]) -> float:
    """Returns how much values rise from the first time to the last, by their slope.

    :raises ValueError: As slope does.
    """
    return slope(times, values) * (max(times) - min(times))
