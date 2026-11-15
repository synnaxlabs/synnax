#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The runner pools and their channels."""

from typing import NamedTuple

import synnax as sy

from common import BUSY, IDLE
from publisher import Publisher, on_change

# A pool is the label a job asks for and its runners carry.
Pool = str


class Spec(NamedTuple):
    title: str
    channel: str
    # Whether the scaler starts and stops its runners.
    scaled: bool = True


POOLS: dict[Pool, Spec] = {
    "windows-build-bot": Spec(title="Windows Build", channel="win_build"),
    "ubuntu-build-bot": Spec(title="Ubuntu Build", channel="ubuntu_build"),
    "windows-test-bot": Spec(title="Windows Test", channel="win_test"),
    "ubuntu-test-bot": Spec(title="Ubuntu Test", channel="ubuntu_test"),
    "macos-build-bot": Spec(title="macOS Build", channel="macos_build", scaled=False),
}


def pool_of(labels: set[str]) -> Pool | None:
    """The pool a runner belongs to or a job asks for, from its labels."""
    return next((label for label in sorted(labels) if label in POOLS), None)


POOL_INDEX = "runner_pool_time"
# The waits change every cycle while a job waits, so they keep their own index.
WAIT_INDEX = "runner_pool_wait_time"
TOTAL = "total"
# The channel name of each pool. The total sums every pool.
NAMES = (*(spec.channel for spec in POOLS.values()), TOTAL)
QUEUED = "queued"
QUEUE_WAIT = "queue_wait"
BUSY_RUNNERS = "busy"
IDLE_RUNNERS = "idle"
STARTING = "starting"
OFFLINE_RUNNERS = "offline"
COUNTS = (QUEUED, BUSY_RUNNERS, IDLE_RUNNERS, STARTING, OFFLINE_RUNNERS)


def pool_channel(name: str, *, metric: str) -> str:
    return f"runner_{name}_{metric}"


def count_channels() -> dict[str, sy.DataType]:
    return {
        pool_channel(name, metric=metric): sy.DataType.INT32
        for name in NAMES
        for metric in COUNTS
    }


def wait_channels() -> dict[str, sy.DataType]:
    return {pool_channel(name, metric=QUEUE_WAIT): sy.DataType.INT32 for name in NAMES}


def pool_counts(
    *,
    status: dict[str, int],
    labels: dict[str, set[str]],
    queued: dict[Pool, int],
    booting: set[str],
) -> dict[str, int]:
    """The queued jobs and the runner counts of every pool. A pool's four runner
    counts add up to its size."""
    values = dict.fromkeys(count_channels(), 0)
    for pool, spec in POOLS.items():
        values[pool_channel(spec.channel, metric=QUEUED)] = queued.get(pool, 0)
    for runner, state in status.items():
        own = pool_of(labels.get(runner, set()))
        if own is None:
            continue
        if runner in booting:
            metric = STARTING
        elif state == BUSY:
            metric = BUSY_RUNNERS
        elif state == IDLE:
            metric = IDLE_RUNNERS
        else:
            metric = OFFLINE_RUNNERS
        values[pool_channel(POOLS[own].channel, metric=metric)] += 1
    for metric in COUNTS:
        values[pool_channel(TOTAL, metric=metric)] = sum(
            values[pool_channel(spec.channel, metric=metric)] for spec in POOLS.values()
        )
    return values


def pool_waits(waits: dict[Pool, int]) -> dict[str, int]:
    """Seconds the oldest queued job of every pool has waited, 0 without one."""
    values = {
        pool_channel(spec.channel, metric=QUEUE_WAIT): waits.get(pool, 0)
        for pool, spec in POOLS.items()
    }
    return values | {pool_channel(TOTAL, metric=QUEUE_WAIT): max(values.values())}


class PoolReport:
    """Writes the counts and the waits of every pool, on change. Values that a failed
    write did not land go out again on the next step."""

    def __init__(self, *, publisher: Publisher) -> None:
        self.publisher = publisher
        self.counts: dict[str, int] = {}
        self.waits: dict[str, int] = {}

    def step(
        self,
        *,
        ready: bool,
        counts: dict[str, int],
        waits: dict[str, int],
        now: sy.TimeStamp,
    ) -> None:
        """Writes what changed since the last write. Does nothing until ready, because
        a queue that was never read looks empty."""
        if not ready:
            return
        change = on_change(POOL_INDEX, last=self.counts, current=counts, now=now) or {}
        change |= on_change(WAIT_INDEX, last=self.waits, current=waits, now=now) or {}
        if change and self.publisher.write(change):
            self.counts, self.waits = counts, waits
