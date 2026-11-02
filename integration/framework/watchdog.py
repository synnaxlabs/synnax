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
    armed: bool = True
    interrupted: bool = False


class Watchdog:
    """Raises Timeout in watched threads that run past their deadlines. Safe for
    concurrent use."""

    def __init__(self, grace: float) -> None:
        """:param grace: Seconds past its deadline a thread has to exit before the
        watchdog reports it as abandoned."""
        self._grace = grace
        self._entries: dict[threading.Thread, _Entry] = {}
        self._lock = threading.Lock()

    def watch(self, thread: threading.Thread, *, timeout: float) -> None:
        """Starts a timeout of the given seconds for thread."""
        with self._lock:
            self._entries[thread] = _Entry(deadline=time.monotonic() + timeout)

    def disarm(self, thread: threading.Thread) -> None:
        """Stops raising Timeout in thread and drops a Timeout not yet raised. The
        thread can still become abandoned."""
        with self._lock:
            entry = self._entries.get(thread)
            if entry is None:
                return
            entry.armed = False
            if entry.interrupted and thread.ident is not None:
                ctypes.pythonapi.PyThreadState_SetAsyncExc(
                    ctypes.c_ulong(thread.ident), None
                )

    def unwatch(self, thread: threading.Thread) -> None:
        with self._lock:
            self._entries.pop(thread, None)

    def check(self) -> list[threading.Thread]:
        """Raises Timeout in every armed thread past its deadline.

        :returns: The threads this call interrupted.
        """
        now = time.monotonic()
        interrupted: list[threading.Thread] = []
        with self._lock:
            for thread, entry in self._entries.items():
                if not entry.armed or entry.interrupted or now < entry.deadline:
                    continue
                entry.interrupted = True
                if interrupt(thread, Timeout):
                    interrupted.append(thread)
        return interrupted

    def abandoned(self, thread: threading.Thread) -> bool:
        """Returns True if thread is watched and still alive more than the grace
        period past its deadline. Callers stop waiting on an abandoned thread."""
        with self._lock:
            entry = self._entries.get(thread)
            if entry is None:
                return False
            expired = time.monotonic() > entry.deadline + self._grace
        return expired and thread.is_alive()


class Slots:
    """A pool of slots where each holder frees its slot at most once, so a holder
    can be freed early while it still runs. Safe for concurrent use."""

    def __init__(self, size: int) -> None:
        self._semaphore = threading.Semaphore(size)
        self._held: set[threading.Thread] = set()
        self._lock = threading.Lock()

    def acquire(self) -> None:
        """Blocks until a slot is free, then holds it for the calling thread."""
        self._semaphore.acquire()
        with self._lock:
            self._held.add(threading.current_thread())

    def free(self, thread: threading.Thread) -> None:
        """Frees the slot thread holds. Does nothing if thread holds no slot."""
        with self._lock:
            if thread not in self._held:
                return
            self._held.remove(thread)
        self._semaphore.release()
