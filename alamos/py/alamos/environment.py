#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from typing import Literal

Environment = Literal["bench", "debug", "prod"]


ENVIRONMENTS: list[Environment] = ["bench", "debug", "prod"]
"""List of valid environments."""


EnvironmentFilter = Callable[[Environment], bool]
"""
Function that returns True if a given environment is valid for use (i.e., should this
trace be executed?).
"""


def env_threshold_filter(threshold: Environment) -> EnvironmentFilter:
    """
    Returns an environment filter that evaluates True if the provided environment is
    greater than or equal to the threshold.

    :param threshold: The minimum environment level to allow.
    """
    return lambda env: ENVIRONMENTS.index(env) >= ENVIRONMENTS.index(threshold)
