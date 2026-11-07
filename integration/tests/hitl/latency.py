#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Limits on the latencies that the HITL performance tests measure."""

import numpy as np

from framework.test_case import TestCase


def check(
    case: TestCase,
    name: str,
    latencies_ms: list[float],
    *,
    max_median_ms: float,
    max_p99_ms: float,
) -> None:
    """Records the median, p99, and maximum of latencies_ms in the metrics of case,
    under names that start with name. Fails case when the median or p99 is over its
    limit."""
    measured = np.array(latencies_ms)
    median = float(np.median(measured))
    p99 = float(np.percentile(measured, 99))
    case.metrics[f"{name}_median_ms"] = median
    case.metrics[f"{name}_p99_ms"] = p99
    case.metrics[f"{name}_max_ms"] = float(measured.max())
    case.log(
        f"{name} in ms: median {median:.3f}, p99 {p99:.3f}, max {measured.max():.3f}"
    )
    if median > max_median_ms or p99 > max_p99_ms:
        case.fail(
            f"{name} took {median:.2f} ms at the median and {p99:.2f} ms at p99. The "
            f"limits are {max_median_ms} ms and {max_p99_ms} ms."
        )
