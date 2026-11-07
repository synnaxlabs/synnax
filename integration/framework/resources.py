#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from collections import Counter
from collections.abc import Mapping

BROWSER = "browser"
"""A browser that a Console test drives."""

NI = "ni"
"""The NI devices of the machine. Only the first Core's Driver reaches them."""

CRIO = "crio"
"""The cRIO of the HITL station. Only one Driver at a time runs on it."""


def simulator(device_name: str) -> str:
    """Returns the resource of a device simulator. A simulator listens on a fixed
    address, so only one test can run it at a time."""
    return f"simulator {device_name}"


class Unavailable(Exception):
    """Abandoned tests hold a resource, so it cannot come free while they run."""


class Resources:
    """Machine resources that tests running at the same time share. A resource
    admits one holder unless its limit admits more. Safe for concurrent use."""

    def __init__(self, limits: Mapping[str, int | None] | None = None) -> None:
        """:param limits: Holders each named resource admits at once. None admits
        any number."""
        self._limits = dict(limits or {})
        self._used: Counter[str] = Counter()
        self._held: dict[threading.Thread, frozenset[str]] = {}
        self._abandoned: dict[threading.Thread, str] = {}
        self._cond = threading.Condition()

    def acquire(self, names: frozenset[str]) -> None:
        """Blocks until every resource in names is free, then holds all of them for
        the calling thread. Taking them together cannot deadlock.

        :raises Unavailable: if a resource in names is not free and abandoned holders
        fill its limit.
        """
        thread = threading.current_thread()
        with self._cond:
            self._cond.wait_for(
                lambda: all(self._free(n) for n in names) or self._lost(names)
            )
            if lost := self._lost(names):
                raise Unavailable(lost)
            self._used.update(names)
            self._held[thread] = names

    def abandon(self, thread: threading.Thread, test: str) -> None:
        """Marks thread as an abandoned holder that runs the named test. It keeps its
        resources until it frees them."""
        with self._cond:
            if thread in self._held:
                self._abandoned[thread] = test
                self._cond.notify_all()

    def free(self, thread: threading.Thread) -> None:
        """Frees the resources thread holds. Does nothing if it holds none."""
        with self._cond:
            self._abandoned.pop(thread, None)
            names = self._held.pop(thread, None)
            if names is None:
                return
            self._used.subtract(names)
            self._cond.notify_all()

    def _free(self, name: str) -> bool:
        limit = self._limits.get(name, 1)
        return limit is None or self._used[name] < limit

    def _lost(self, names: frozenset[str]) -> str | None:
        for name in sorted(names):
            limit = self._limits.get(name, 1)
            if limit is None or self._used[name] < limit:
                continue
            tests = sorted(
                test
                for thread, test in self._abandoned.items()
                if name in self._held[thread]
            )
            if len(tests) >= limit:
                return f"Abandoned tests still hold {name}: {', '.join(tests)}"
        return None
