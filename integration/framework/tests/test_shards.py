#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import json
from pathlib import Path

import pytest

from framework import config_client, resources, shards


def sequence(
    name: str, *cases: str, isolated: bool = False, independent: bool = False
) -> config_client.Sequence:
    tests = [config_client.TestDefinition(case=c) for c in cases]
    return config_client.Sequence(
        name=name,
        order="asynchronous",
        pool_size=3,
        tests=tests,
        isolated=isolated,
        independent=independent,
    )


def browser_cases(test: config_client.TestDefinition) -> frozenset[str]:
    if test.case.startswith("console/"):
        return frozenset({resources.BROWSER})
    return frozenset()


def select(
    sequences: list[config_client.Sequence],
    shard: str,
    durations: dict[str, float],
    *,
    cores: int = 1,
    browsers: int | None = None,
) -> list[str]:
    picked = shards.select(
        sequences,
        shards.parse(shard),
        durations=durations,
        resources_of=browser_cases,
        cores=cores,
        browsers=browsers,
    )
    return [s.name for s in picked]


class TestSelect:
    def test_every_sequence_once(self) -> None:
        """Should give each sequence to exactly one shard."""
        sequences = [sequence(n, f"t/{n}") for n in "abcdefg"]
        durations = {f"t/{n}": float(i + 1) for i, n in enumerate("abcdefg")}
        picked = [select(sequences, f"{i}/3", durations) for i in (1, 2, 3)]
        assert sorted(n for p in picked for n in p) == list("abcdefg")

    def test_balance(self) -> None:
        """Should give the longest sequence first to the shard with the least time."""
        sequences = [sequence(n, f"t/{n}") for n in "abcd"]
        durations = {"t/a": 10.0, "t/b": 9.0, "t/c": 8.0, "t/d": 1.0}
        assert select(sequences, "1/2", durations) == ["a", "d"]
        assert select(sequences, "2/2", durations) == ["b", "c"]

    def test_file_order(self) -> None:
        """Should keep the sequences of a shard in their file order."""
        sequences = [sequence(n, f"t/{n}") for n in "abc"]
        durations = {"t/a": 1.0, "t/b": 5.0, "t/c": 3.0}
        assert select(sequences, "1/1", durations) == ["a", "b", "c"]

    def test_sum_of_tests(self) -> None:
        """Should count a sequence as the sum of its tests."""
        sequences = [
            sequence("a", "t/a1", "t/a2"),
            sequence("b", "t/b"),
            sequence("c", "t/c"),
        ]
        durations = {"t/a1": 3.0, "t/a2": 3.0, "t/b": 5.0, "t/c": 1.0}
        assert select(sequences, "1/2", durations) == ["a"]

    def test_median_for_unknown(self) -> None:
        """Should count a test without a recorded time as the median time."""
        sequences = [sequence("a", "t/new"), sequence("b", "t/b"), sequence("c", "t/c")]
        durations = {"t/b": 4.0, "t/c": 2.0, "t/x": 30.0}
        assert select(sequences, "1/2", durations) == ["a", "c"]

    def test_browsers_and_cores(self) -> None:
        """Should divide the time of a Console test by the browsers and of any other
        test by the Cores."""
        sequences = [
            sequence("ui", "console/ui"),
            sequence("a", "t/a"),
            sequence("b", "t/b"),
        ]
        durations = {"console/ui": 12.0, "t/a": 10.0, "t/b": 10.0}
        assert select(sequences, "1/2", durations, cores=1, browsers=1) == ["ui"]
        assert select(sequences, "1/2", durations, cores=1, browsers=3) == ["ui", "a"]

    def test_isolated(self) -> None:
        """Should count an isolated sequence at its full time, as it runs alone."""
        sequences = [
            sequence("iso", "t/iso", isolated=True),
            sequence("a", "t/a"),
            sequence("b", "t/b"),
        ]
        durations = {"t/iso": 6.0, "t/a": 10.0, "t/b": 10.0}
        assert select(sequences, "1/2", durations, cores=4) == ["iso"]

    def test_split_independent(self) -> None:
        """Should split an independent sequence into single tests, and keep its tests
        and its order in file order."""
        sequences = [
            sequence("a", "t/a1", "t/a2", "t/a3", "t/a4", independent=True),
            sequence("b", "t/b"),
        ]
        durations = {"t/a1": 4.0, "t/a2": 3.0, "t/a3": 2.0, "t/a4": 1.0, "t/b": 5.0}
        picked = [
            shards.select(
                sequences,
                shards.parse(f"{i}/2"),
                durations=durations,
                resources_of=browser_cases,
                cores=1,
                browsers=None,
            )
            for i in (1, 2)
        ]
        assert [[(s.name, [t.case for t in s.tests]) for s in p] for p in picked] == [
            [("a", ["t/a3", "t/a4"]), ("b", ["t/b"])],
            [("a", ["t/a1", "t/a2"])],
        ]
        assert {(s.order, s.pool_size) for p in picked for s in p} == {
            ("asynchronous", 3)
        }

    def test_empty_shard(self) -> None:
        """Should raise when a shard gets no sequences."""
        with pytest.raises(ValueError, match="Shard 3/3 has no sequences"):
            select([sequence("a", "t/a"), sequence("b", "t/b")], "3/3", {"t/a": 1.0})


class TestParse:
    def test_parse(self) -> None:
        """Should parse index/count."""
        assert shards.parse("2/3") == shards.Shard(index=2, count=3)

    @pytest.mark.parametrize("value", ["0/2", "3/2", "2", "a/b", "1/"])
    def test_invalid(self, value: str) -> None:
        """Should reject a value that is not index/count with index from 1 to
        count."""
        with pytest.raises(argparse.ArgumentTypeError):
            shards.parse(value)


def summary(path: Path, *tests: tuple[str, str, str, float]) -> None:
    path.mkdir(parents=True)
    rows = [
        {"case": c, "name": n, "status": s, "duration_s": d} for c, n, s, d in tests
    ]
    (path / "summary.json").write_text(json.dumps({"tests": rows}))


class TestDurations:
    def test_record(self, tmp_path: Path) -> None:
        """Should record the median seconds of each test that passed or skipped in a
        run bundle, keyed by the test's string form."""
        summary(
            tmp_path / "a" / "run-1",
            ("t/a", "a", "PASSED", 1.0),
            ("t/m", "Many", "SKIPPED", 0.5),
            ("t/f", "f", "FAILED", 99.0),
        )
        summary(tmp_path / "b" / "run-2", ("t/a", "a", "PASSED", 3.0))
        summary(tmp_path / "b" / "latest", ("t/a", "a", "PASSED", 50.0))
        assert shards.record(tmp_path) == {"t/a": 2.0, "t/m (Many)": 0.5}

    def test_record_nothing(self, tmp_path: Path) -> None:
        """Should raise when no test passed or skipped."""
        summary(tmp_path / "run-1", ("t/f", "f", "FAILED", 1.0))
        with pytest.raises(ValueError, match="No passed or skipped tests"):
            shards.record(tmp_path)

    def test_load(self, tmp_path: Path) -> None:
        """Should load the seconds of each test."""
        path = tmp_path / "durations.json"
        path.write_text('{"t/a": 1.5}')
        assert shards.load_durations(path) == {"t/a": 1.5}

    def test_recorded(self) -> None:
        """Should hold a recorded time for each test of the checked-in suites."""
        assert len(shards.load_durations()) > 0
