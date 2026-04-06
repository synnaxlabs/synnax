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
from x.strings import get_random_name, validate_and_sanitize_name
from x.version import get_synnax_version
from x.websocket import (
    WEBSOCKET_ERROR_PATTERNS,
    WebSocketErrorFilter,
    ignore_websocket_errors,
    is_websocket_error,
    suppress_websocket_errors,
)

__all__ = [
    "Platform",
    "WEBSOCKET_ERROR_PATTERNS",
    "WebSocketErrorFilter",
    "get_cpu_cores",
    "get_machine_info",
    "get_memory_info",
    "get_platform",
    "get_random_name",
    "get_synnax_version",
    "ignore_websocket_errors",
    "is_ci",
    "is_websocket_error",
    "suppress_websocket_errors",
    "validate_and_sanitize_name",
]
