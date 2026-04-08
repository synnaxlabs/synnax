#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from x.exceptions import ExceptionPayload, register_exception

_FREIGHTER_EXCEPTION_TYPE = "freighter."


class Unreachable(Exception):
    """Raised when a target is unreachable."""

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "unreachable"

    target: str
    message: str

    def __init__(self, target: str = "", message: str = "Unreachable"):
        self.target = target
        self.message = message
        super().__init__(message)

    def __str__(self) -> str:
        return self.message


class StreamClosed(Exception):
    """Raised when a stream is closed."""

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "stream_closed"

    def __str__(self) -> str:
        return "StreamClosed"


class EOF(Exception):
    """Raised when a stream reaches end-of-file."""

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "eof"

    def __str__(self) -> str:
        return "EOF"


def _freighter_encode(exc: Exception) -> ExceptionPayload | None:
    if isinstance(exc, Unreachable):
        return ExceptionPayload(type=Unreachable.TYPE, data=exc.message)
    if isinstance(exc, StreamClosed):
        return ExceptionPayload(type=StreamClosed.TYPE, data=str(exc))
    if isinstance(exc, EOF):
        return ExceptionPayload(type=EOF.TYPE, data=str(exc))
    return None


def _freighter_decode(exc: ExceptionPayload) -> Exception | None:
    if exc.type is None or not exc.type.startswith(_FREIGHTER_EXCEPTION_TYPE):
        return None
    if exc.type == Unreachable.TYPE:
        return Unreachable(message=exc.data) if exc.data is not None else Unreachable()
    if exc.type == StreamClosed.TYPE:
        return StreamClosed()
    if exc.type == EOF.TYPE:
        return EOF()
    raise ValueError(f"Unknown error type: {exc.type}")


register_exception(_freighter_encode, _freighter_decode)
