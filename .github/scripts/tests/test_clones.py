#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import sys
from pathlib import Path
from typing import Any

import pytest

import clones
from clones import Fragment, Group, groups, markdown


def pair(
    fmt: str, *, a: tuple[str, int, int], b: tuple[str, int, int]
) -> dict[str, Any]:
    return {
        "format": fmt,
        "firstFile": {"name": a[0], "start": a[1], "end": a[2]},
        "secondFile": {"name": b[0], "start": b[1], "end": b[2]},
    }


class TestGroup:
    """Counts the lines that removing a clone group's copies saves."""

    def test_should_count_every_copy_but_the_longest(self) -> None:
        group = Group(
            (
                Fragment(path="a.go", start=1, end=10),
                Fragment(path="b.go", start=1, end=10),
            )
        )
        assert group.removable == 10

    def test_should_count_overlapping_lines_once(self) -> None:
        group = Group(
            (
                Fragment(path="a.go", start=1, end=10),
                Fragment(path="a.go", start=6, end=15),
            )
        )
        assert group.removable == 5


class TestGroups:
    """Joins clone pairs into groups and ranks them."""

    def test_should_join_pairs_that_share_a_fragment_and_rank_them(self) -> None:
        duplicates = [
            pair("go", a=("a.go", 1, 10), b=("b.go", 1, 10)),
            pair("go", a=("b.go", 1, 10), b=("c.go", 5, 14)),
            pair("go", a=("d.go", 1, 30), b=("e.go", 1, 30)),
            pair("python", a=("a.py", 1, 100), b=("b.py", 1, 100)),
        ]
        ranked = groups(duplicates, formats={"go"})
        assert [g.removable for g in ranked] == [30, 20]
        assert [f.path for f in ranked[1].fragments] == ["a.go", "b.go", "c.go"]
        assert markdown(ranked[:1]) == (
            "1. 30 lines, 2 copies: `d.go:1-30`, `e.go:1-30`"
        )

    def test_should_leave_out_a_group_that_a_larger_group_covers(self) -> None:
        duplicates = [
            pair("go", a=("a.go", 1, 40), b=("b.go", 1, 40)),
            pair("go", a=("a.go", 5, 20), b=("b.go", 5, 20)),
        ]
        assert [g.removable for g in groups(duplicates, formats={"go"})] == [40]

    def test_should_keep_a_group_that_a_larger_group_only_overlaps(self) -> None:
        duplicates = [
            pair("go", a=("a.go", 1, 40), b=("b.go", 1, 40)),
            pair("go", a=("a.go", 40, 100), b=("b.go", 40, 100)),
        ]
        assert [g.removable for g in groups(duplicates, formats={"go"})] == [61, 40]

    def test_should_give_the_same_order_for_any_input_order(self) -> None:
        duplicates = [
            pair("go", a=("a.go", 1, 10), b=("b.go", 1, 10)),
            pair("go", a=("c.go", 1, 10), b=("d.go", 1, 10)),
        ]
        assert groups(duplicates, formats={"go"}) == groups(
            duplicates[::-1], formats={"go"}
        )


class TestMain:
    """Prints the top clone groups of a jscpd report."""

    def test_should_print_the_top_groups_of_the_language(
        self,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        report = tmp_path / "report.json"
        duplicates = [
            pair("go", a=("a.go", 1, 10), b=("b.go", 1, 10)),
            pair("go", a=("c.go", 1, 30), b=("d.go", 1, 30)),
            pair("python", a=("a.py", 1, 100), b=("b.py", 1, 100)),
        ]
        report.write_text(json.dumps({"duplicates": duplicates}))
        argv = ["clones.py", str(report), "--language", "go", "--top", "1"]
        monkeypatch.setattr(sys, "argv", argv)
        clones.main()
        assert capsys.readouterr().out == (
            "1. 30 lines, 2 copies: `c.go:1-30`, `d.go:1-30`\n"
        )
