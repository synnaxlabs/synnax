#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import contextlib
import sys
from collections.abc import Generator
from typing import Any

WEBSOCKET_ERROR_PATTERNS = [
    "1011",
    "keepalive ping timeout",
    "keepalive ping failed",
    "keepalive ping",
    "timed out while closing connection",
    "ConnectionClosedError",
    "WebSocketException",
]


class WebSocketErrorFilter:
    """Wraps stderr to filter out WebSocket error messages."""

    def __init__(self) -> None:
        self.original_stderr = sys.stderr

    def write(self, text: str) -> None:
        if any(phrase in text for phrase in WEBSOCKET_ERROR_PATTERNS):
            return
        self.original_stderr.write(text)

    def flush(self) -> None:
        self.original_stderr.flush()


def ignore_websocket_errors(
    type: type[BaseException], value: BaseException, traceback: Any
) -> None:
    """Exception hook to silently ignore WebSocket errors."""
    error_str = str(value)
    if any(phrase in error_str for phrase in WEBSOCKET_ERROR_PATTERNS):
        return
    sys.__excepthook__(type, value, traceback)


def is_websocket_error(error: Exception) -> bool:
    """Check if an exception is a WebSocket-related error that should be ignored."""
    error_str = str(error)
    return any(phrase in error_str for phrase in WEBSOCKET_ERROR_PATTERNS)


@contextlib.contextmanager
def suppress_websocket_errors() -> Generator[None, None, None]:
    """Suppress WebSocket errors, re-raising everything else."""
    try:
        yield
    except Exception as e:
        if not is_websocket_error(e):
            raise
