#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Splits a suite into shards that separate conductor runs execute."""

import argparse
import json
import statistics
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from pathlib import Path

from pydantic import TypeAdapter

from framework import resources
from framework.config_client import Sequence, TestDefinition

DURATIONS = Path(__file__).resolve().parent.parent / "tests" / "durations.json"
"""Recorded seconds of each test, keyed by the test's string form."""


@dataclass(frozen=True)
class Shard:
    """Shard index of count, from 1."""

    index: int
    count: int

    def __str__(self) -> str:
        return f"{self.index}/{self.count}"


def parse(value: str) -> Shard:
    """Parses "index/count", as the --shard flag takes it.

    :raises argparse.ArgumentTypeError: If value is not "index/count" with index from
        1 to count.
    """
    index, _, count = value.partition("/")
    try:
        shard = Shard(int(index), int(count))
    except ValueError:
        raise argparse.ArgumentTypeError(f"{value!r} is not index/count") from None
    if not 1 <= shard.index <= shard.count:
        raise argparse.ArgumentTypeError(f"{value!r} needs an index from 1 to count")
    return shard


def load_durations(path: Path = DURATIONS) -> dict[str, float]:
    return TypeAdapter(dict[str, float]).validate_json(path.read_bytes())


def record(bundles: Path) -> dict[str, float]:
    """Returns the median seconds of each test that passed or skipped in the run
    bundles under bundles.

    :raises ValueError: If no test passed or skipped.
    """
    seconds: dict[str, list[float]] = {}
    for path in bundles.rglob("run-*/summary.json"):
        for test in json.loads(path.read_text(encoding="utf-8"))["tests"]:
            if test["status"] in ("PASSED", "SKIPPED"):
                key = str(TestDefinition(case=test["case"], name=test["name"]))
                seconds.setdefault(key, []).append(test["duration_s"])
    if not seconds:
        raise ValueError(f"No passed or skipped tests in {bundles}")
    return {k: round(statistics.median(v), 1) for k, v in sorted(seconds.items())}


def select(
    sequences: list[Sequence],
    shard: Shard,
    *,
    durations: Mapping[str, float],
    resources_of: Callable[[TestDefinition], frozenset[str]],
    cores: int,
    browsers: int | None,
) -> list[Sequence]:
    """Returns the sequences of the shard, and their tests, in their file order.

    Shards get whole sequences, or single tests of independent sequences, of about
    equal run time. A test without a recorded duration counts the median. Shards
    agree only when every run gets the same arguments other than shard.index.

    :raises ValueError: If the shard gets no sequences.
    """
    default = statistics.median(durations.values())

    def cost(sequence: Sequence, test: TestDefinition) -> float:
        seconds = durations.get(str(test), default)
        if sequence.isolated:
            return seconds
        if browsers is not None and resources.BROWSER in resources_of(test):
            return seconds / browsers
        return seconds / cores

    # (cost, sequence index, test indexes) of each piece that one shard runs.
    pieces: list[tuple[float, int, list[int]]] = []
    for i, sequence in enumerate(sequences):
        costs = [cost(sequence, test) for test in sequence.tests]
        if sequence.independent:
            pieces.extend((c, i, [j]) for j, c in enumerate(costs))
        else:
            pieces.append((sum(costs), i, list(range(len(costs)))))
    loads = [0.0] * shard.count
    mine: dict[int, set[int]] = {}
    for seconds, i, tests in sorted(pieces, key=lambda p: -p[0]):
        least = loads.index(min(loads))
        loads[least] += seconds
        if least == shard.index - 1:
            mine.setdefault(i, set()).update(tests)
    if not mine:
        raise ValueError(f"Shard {shard} has no sequences")
    return [
        s.model_copy(
            update={"tests": [t for j, t in enumerate(s.tests) if j in mine[i]]}
        )
        for i, s in enumerate(sequences)
        if i in mine
    ]
