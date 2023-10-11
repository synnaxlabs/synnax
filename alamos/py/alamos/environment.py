#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal, Callable

ENVIRONMENTS = ["bench", "debug", "prod"]
"""List of valid environments"""

Environment = Literal["bench", "debug", "prod"]
"""Environment defines the environment in which instrumentation is running. Traces can
be constrained to run only in certain environments.
"""

EnvironmentFilter = Callable[[Environment], bool]
"""EnvironmentFilter is a function takes a environment and returns true if the environment
is valid for use i.e. 'should this trace be executed?'"""


def env_threshold_filter(threshold: Environment) -> EnvironmentFilter:
    """returns an environment filter that returns true if the environment is greater than
    or equal to the threshold.

    :param threshold: The threshold environment to compare against
    """
    return lambda env: ENVIRONMENTS.index(env) >= ENVIRONMENTS.index(threshold)
