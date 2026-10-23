#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Shared Python utilities for the Synnax monorepo."""

from x.env import is_ci
from x.os import (
    Platform,
    get_cpu_cores,
    get_machine_info,
    get_memory_info,
    get_platform,
)
from x.strings import random_name, validate_and_sanitize_name

__all__ = [
    "Platform",
    "get_cpu_cores",
    "get_machine_info",
    "get_memory_info",
    "get_platform",
    "is_ci",
    "random_name",
    "validate_and_sanitize_name",
]
