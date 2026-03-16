#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import logging
import sys
from dataclasses import dataclass
from enum import Enum, auto
from typing import Protocol

import synnax as sy
from xpy import is_ci


class LogMode(Enum):
    """Controls whether log entries are emitted immediately or buffered."""

    REALTIME = auto()
    BUFFERED = auto()


@dataclass
class LogEntry:
    """A single log entry with timestamp, level, source name, and message."""

    timestamp: sy.TimeStamp
    level: int
    name: str
    message: str

    def format(self) -> str:
        ts = self.timestamp.datetime().strftime("%H:%M:%S.%f")[:-4]
        if self.name:
            return f"{ts} | {self.name} > {self.message}"
        return self.message


class LogSink(Protocol):
    """Protocol for log output destinations."""

    def emit(self, entry: LogEntry) -> None: ...

    def flush(self) -> None: ...

    def close(self) -> None: ...


class StdoutSink:
    """Writes log entries to stdout via Python's logging module."""

    def __init__(self, name: str) -> None:
        self._logger = logging.getLogger(f"synnax.{name}")
        self._logger.setLevel(logging.DEBUG)
        self._logger.propagate = False
        for h in self._logger.handlers[:]:
            self._logger.removeHandler(h)
        handler = logging.StreamHandler(sys.stdout)
        handler.setLevel(logging.DEBUG)
        handler.setFormatter(logging.Formatter("%(message)s"))
        self._logger.addHandler(handler)
        self._handler = handler

    def emit(self, entry: LogEntry) -> None:
        self._logger.log(entry.level, entry.format())

    def flush(self) -> None:
        if hasattr(self._handler, "stream"):
            self._handler.stream.flush()

    def close(self) -> None:
        pass


class SynnaxChannelSink:
    """Writes log entries to a virtual STRING channel in Synnax."""

    def __init__(self, client: sy.Synnax, channel_name: str) -> None:
        self._channel = client.channels.create(
            name=channel_name,
            virtual=True,
            data_type=sy.DataType.STRING,
            retrieve_if_name_exists=True,
        )
        self._writer = client.open_writer(
            start=sy.TimeStamp.now(),
            channels=[self._channel.key],
            name=f"{channel_name}_writer",
        )

    def emit(self, entry: LogEntry) -> None:
        self._writer.write(self._channel.key, [entry.message])

    def flush(self) -> None:
        pass

    def close(self) -> None:
        self._writer.close()


_ci_configured = False


def _ensure_ci_stdout() -> None:
    global _ci_configured
    if _ci_configured:
        return
    _ci_configured = True
    if is_ci() and hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(line_buffering=True)


class LogClient:
    """Logging client with real-time or buffered emission modes.

    In REALTIME mode, entries are emitted to all sinks immediately.
    In BUFFERED mode, entries are stored in memory and only emitted
    when dump() is called (intended for test cases that dump on failure).
    """

    def __init__(
        self,
        name: str,
        mode: LogMode = LogMode.REALTIME,
        sinks: list[LogSink] | None = None,
        persistent_sinks: list[LogSink] | None = None,
    ) -> None:
        _ensure_ci_stdout()
        self._name = name
        self._mode = mode
        self._sinks: list[LogSink] = sinks or [StdoutSink(name)]
        self._persistent_sinks: list[LogSink] = persistent_sinks or []
        self._buffer: list[LogEntry] = []

    def info(self, message: str) -> None:
        self._log(logging.INFO, message)

    def error(self, message: str) -> None:
        self._log(logging.ERROR, message)

    def raw(self, message: str) -> None:
        """Emit a message without timestamp or name prefix."""
        entry = LogEntry(
            timestamp=sy.TimeStamp.now(),
            level=logging.INFO,
            name="",
            message=message,
        )
        self._emit_persistent(entry)
        if self._mode == LogMode.REALTIME:
            self._emit(entry)
        else:
            self._buffer.append(entry)

    def dump(self) -> None:
        """Emit all buffered entries to sinks, then clear the buffer."""
        for entry in self._buffer:
            self._emit(entry)
        self._buffer.clear()

    def discard(self) -> None:
        """Clear the buffer without emitting."""
        self._buffer.clear()

    def close(self) -> None:
        """Close all sinks."""
        for sink in self._sinks:
            sink.close()
        for sink in self._persistent_sinks:
            sink.close()

    def _log(self, level: int, message: str) -> None:
        entry = LogEntry(
            timestamp=sy.TimeStamp.now(),
            level=level,
            name=self._name,
            message=message,
        )
        self._emit_persistent(entry)
        if self._mode == LogMode.REALTIME:
            self._emit(entry)
        else:
            self._buffer.append(entry)

    def _emit(self, entry: LogEntry) -> None:
        for sink in self._sinks:
            sink.emit(entry)
            sink.flush()

    def _emit_persistent(self, entry: LogEntry) -> None:
        for sink in self._persistent_sinks:
            sink.emit(entry)
            sink.flush()
