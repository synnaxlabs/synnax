#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ctypes
import threading
import time
from dataclasses import dataclass


class Timeout(BaseException):
    """Raised inside a test's thread when the test runs past its timeout. It is not
    an Exception, so a test's own `except Exception` cannot swallow it."""


def interrupt(thread: threading.Thread, exc: type[BaseException]) -> bool:
    """Raises exc in thread at the thread's next Python bytecode. A thread blocked
    inside a C call does not see exc until the call returns.

    :returns: True if exc is now pending in thread.
    """
    if thread.ident is None or not thread.is_alive():
        return False
    ident = ctypes.c_ulong(thread.ident)
    count: int = ctypes.pythonapi.PyThreadState_SetAsyncExc(
        ident, ctypes.py_object(exc)
    )
    if count > 1:
        ctypes.pythonapi.PyThreadState_SetAsyncExc(ident, None)
        return False
    return count == 1


@dataclass
class _Entry:
    deadline: float
    interrupted_at: float | None = None


class Watchdog:
    """Raises Timeout in watched threads that run past their deadlines. Safe for
    concurrent use."""

    def __init__(self, grace: float) -> None:
        """:param grace: Seconds an interrupted thread has to exit before the
        watchdog reports it as abandoned."""
        self._grace = grace
        self._entries: dict[threading.Thread, _Entry] = {}
        self._lock = threading.Lock()

    def watch(self, thread: threading.Thread, timeout: float) -> None:
        """Starts a timeout of the given seconds for thread."""
        with self._lock:
            self._entries[thread] = _Entry(deadline=time.monotonic() + timeout)

    def unwatch(self, thread: threading.Thread) -> None:
        with self._lock:
            self._entries.pop(thread, None)

    def check(self) -> list[threading.Thread]:
        """Raises Timeout in every watched thread past its deadline.

        :returns: The threads this call interrupted.
        """
        now = time.monotonic()
        interrupted: list[threading.Thread] = []
        with self._lock:
            for thread, entry in self._entries.items():
                if entry.interrupted_at is not None or now < entry.deadline:
                    continue
                entry.interrupted_at = now
                if interrupt(thread, Timeout):
                    interrupted.append(thread)
        return interrupted

    def abandoned(self, thread: threading.Thread) -> bool:
        """Returns True if thread was interrupted more than the grace period ago and
        is still alive. Callers stop waiting on an abandoned thread."""
        with self._lock:
            entry = self._entries.get(thread)
            if entry is None or entry.interrupted_at is None:
                return False
            expired = time.monotonic() - entry.interrupted_at > self._grace
        return expired and thread.is_alive()
