#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
import time

import pytest

from framework import resources
from framework.resources import Resources, Unavailable

WAIT = 5.0
# Long enough for a thread that would wrongly acquire to do so.
SETTLE = 0.1


class Holder:
    """A thread that holds names once it acquires them, until released."""

    def __init__(self, pool: Resources, names: frozenset[str]) -> None:
        self.acquired = threading.Event()
        self.released = threading.Event()
        self.error: Unavailable | None = None
        self._pool = pool
        self._names = names
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()

    def _run(self) -> None:
        try:
            self._pool.acquire(self._names)
        except Unavailable as e:
            self.error = e
            return
        self.acquired.set()
        self.released.wait(WAIT)
        self._pool.free(threading.current_thread())

    def stop(self) -> None:
        self.released.set()
        self.thread.join(WAIT)


class TestResources:
    def test_one_holder_by_default(self) -> None:
        """Should admit one holder of a resource with no limit set."""
        pool = Resources()
        first = Holder(pool, frozenset({"a"}))
        assert first.acquired.wait(WAIT)
        second = Holder(pool, frozenset({"a"}))
        time.sleep(SETTLE)
        assert not second.acquired.is_set()
        first.stop()
        assert second.acquired.wait(WAIT)
        second.stop()

    def test_limit(self) -> None:
        """Should admit as many holders as the limit of a resource."""
        pool = Resources({resources.BROWSER: 2})
        names = frozenset({resources.BROWSER})
        holders = [Holder(pool, names) for _ in range(3)]
        deadline = time.monotonic() + WAIT
        while sum(h.acquired.is_set() for h in holders) < 2:
            assert time.monotonic() < deadline
            time.sleep(0.01)
        time.sleep(SETTLE)
        assert sum(h.acquired.is_set() for h in holders) == 2
        for h in holders:
            h.stop()
        assert all(h.acquired.is_set() for h in holders)

    def test_no_limit(self) -> None:
        """Should admit any number of holders of a resource whose limit is None."""
        pool = Resources({resources.BROWSER: None})
        holders = [Holder(pool, frozenset({resources.BROWSER})) for _ in range(5)]
        assert all(h.acquired.wait(WAIT) for h in holders)
        for h in holders:
            h.stop()

    def test_disjoint(self) -> None:
        """Should let holders of different resources run at the same time."""
        pool = Resources()
        a = Holder(pool, frozenset({"a"}))
        b = Holder(pool, frozenset({"b"}))
        assert a.acquired.wait(WAIT)
        assert b.acquired.wait(WAIT)
        a.stop()
        b.stop()

    def test_all_or_nothing(self) -> None:
        """Should hold none of a set of resources until all of them are free."""
        pool = Resources()
        a = Holder(pool, frozenset({"a"}))
        assert a.acquired.wait(WAIT)
        both = Holder(pool, frozenset({"a", "b"}))
        time.sleep(SETTLE)
        assert not both.acquired.is_set()
        b = Holder(pool, frozenset({"b"}))
        assert b.acquired.wait(WAIT)
        b.stop()
        a.stop()
        assert both.acquired.wait(WAIT)
        both.stop()

    def test_abandoned(self) -> None:
        """Should keep the resources of an abandoned holder, and fail each thread
        that waits for them."""
        pool = Resources()
        stuck = Holder(pool, frozenset({"a"}))
        assert stuck.acquired.wait(WAIT)
        waiting = Holder(pool, frozenset({"a"}))
        time.sleep(SETTLE)
        pool.abandon(stuck.thread, "stuck")
        waiting.thread.join(WAIT)
        assert str(waiting.error) == "Abandoned tests still hold a: stuck"
        with pytest.raises(Unavailable, match=r"^Abandoned tests still hold a: stuck$"):
            pool.acquire(frozenset({"a"}))
        stuck.stop()

    def test_abandoned_under_limit(self) -> None:
        """Should admit a holder while abandoned holders leave room under the
        limit."""
        pool = Resources({resources.BROWSER: 2})
        names = frozenset({resources.BROWSER})
        stuck = Holder(pool, names)
        assert stuck.acquired.wait(WAIT)
        pool.abandon(stuck.thread, "stuck")
        holder = Holder(pool, names)
        assert holder.acquired.wait(WAIT)
        holder.stop()
        stuck.stop()

    def test_abandoned_frees(self) -> None:
        """Should admit holders again once an abandoned holder frees."""
        pool = Resources()
        stuck = Holder(pool, frozenset({"a"}))
        assert stuck.acquired.wait(WAIT)
        pool.abandon(stuck.thread, "stuck")
        stuck.stop()
        holder = Holder(pool, frozenset({"a"}))
        assert holder.acquired.wait(WAIT)
        holder.stop()

    def test_free_without_holding(self) -> None:
        """Should do nothing when a thread that holds nothing frees."""
        pool = Resources()
        pool.free(threading.current_thread())
        holder = Holder(pool, frozenset({"a"}))
        assert holder.acquired.wait(WAIT)
        holder.stop()

    def test_empty(self) -> None:
        """Should not block a thread that needs no resources."""
        pool = Resources()
        pool.acquire(frozenset())
        pool.free(threading.current_thread())


class TestSimulator:
    def test_distinct_names(self) -> None:
        """Should give each simulator its own resource."""
        assert resources.simulator("a") != resources.simulator("b")
        assert resources.simulator("a") not in (resources.BROWSER, resources.NI)
