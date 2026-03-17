#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Shared Python utilities for the Synnax monorepo.

This package is named `xpy` instead of `x` to avoid conflicts with Python's
single-letter import resolution. Other languages use `x` directly (Go imports
`github.com/synnaxlabs/x`, TS publishes `@synnaxlabs/x`) because they have
namespace scoping. Python does not, and `import x` is too likely to collide.
"""

from xpy.color import rgb_to_hex
from xpy.env import is_ci
from xpy.os import get_cpu_cores, get_machine_info, get_memory_info
from xpy.strings import get_random_name, validate_and_sanitize_name
from xpy.version import get_synnax_version
from xpy.websocket import (
    WEBSOCKET_ERROR_PATTERNS,
    WebSocketErrorFilter,
    ignore_websocket_errors,
    is_websocket_error,
    suppress_websocket_errors,
)

__all__ = [
    "WEBSOCKET_ERROR_PATTERNS",
    "WebSocketErrorFilter",
    "get_cpu_cores",
    "get_machine_info",
    "get_memory_info",
    "get_random_name",
    "get_synnax_version",
    "ignore_websocket_errors",
    "is_ci",
    "is_websocket_error",
    "rgb_to_hex",
    "suppress_websocket_errors",
    "validate_and_sanitize_name",
]
