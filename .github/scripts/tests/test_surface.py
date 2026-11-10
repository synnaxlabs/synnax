#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "surface.sh"


class Compare:
    """Runs surface.sh over two checkouts, with uv faked to print report.txt."""

    def __init__(self, tmp_path: Path) -> None:
        self.base = tmp_path / "base"
        self.head = tmp_path / "head"
        self.out = tmp_path / "out"
        self.base.mkdir()
        self.head.mkdir()
        bin_dir = tmp_path / "bin"
        bin_dir.mkdir()
        uv = bin_dir / "uv"
        uv.write_text("#!/bin/sh\ncat report.txt\n")
        uv.chmod(0o755)
        self.env = {**os.environ, "PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}"}

    def run(self) -> int:
        return subprocess.run(
            [str(SCRIPT), "driver", str(self.base), str(self.head), str(self.out)],
            env=self.env,
            capture_output=True,
        ).returncode

    @property
    def status(self) -> str:
        return (self.out / "status").read_text().strip()


@pytest.fixture
def compare(tmp_path: Path) -> Compare:
    return Compare(tmp_path)


class TestSurface:
    def test_exits_non_zero_when_both_reports_fail(self, compare: Compare) -> None:
        assert compare.run() != 0
        assert compare.status == "error"

    def test_reports_same_when_the_reports_match(self, compare: Compare) -> None:
        (compare.base / "report.txt").write_text("a\n")
        (compare.head / "report.txt").write_text("a\n")
        assert compare.run() == 0
        assert compare.status == "same"

    def test_reports_changed_with_a_diff(self, compare: Compare) -> None:
        (compare.base / "report.txt").write_text("a\n")
        (compare.head / "report.txt").write_text("b\n")
        assert compare.run() == 0
        assert compare.status == "changed"
        assert "+b" in (compare.out / "diff").read_text()

    def test_records_an_error_and_exits_zero_when_the_base_report_fails(
        self, compare: Compare
    ) -> None:
        (compare.head / "report.txt").write_text("a\n")
        assert compare.run() == 0
        assert compare.status == "error"

    def test_exits_non_zero_when_the_head_report_fails(self, compare: Compare) -> None:
        (compare.base / "report.txt").write_text("a\n")
        assert compare.run() != 0
        assert compare.status == "error"
