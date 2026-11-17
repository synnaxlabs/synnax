#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
from pathlib import Path

from check_suppressions_py import added_lines, check, check_source

SOURCE = """from typing import cast
import typing

a = f()  # type: ignore[arg-type]
b = g()  # noqa: E501
c = h()  # pyright: ignore
d = cast(int, a)
e = typing.cast(int, b)
f = wrap_cast(c)
g = arr.cast(d)
h = "x  # type: ignore"


def cast(x: int) -> int:
    return x
"""

ALL = set(range(1, 16))


def _git(root: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-c", "user.name=t", "-c", "user.email=t@t", *args],
        cwd=root,
        check=True,
        capture_output=True,
    )


class TestAddedLines:
    def test_should_map_each_file_to_its_added_lines(self) -> None:
        diff = (
            "+++ b/a.py\n@@ -1,0 +2,3 @@\n+x\n+y\n+z\n"
            "+++ b/b.py\n@@ -4 +4 @@\n-old\n+new\n@@ -9,2 +8,0 @@\n-gone\n-gone\n"
        )
        assert added_lines(diff) == {"a.py": {2, 3, 4}, "b.py": {4}}


class TestCheckSource:
    def test_should_report_each_comment_and_cast(self) -> None:
        lines = [f.line for f in check_source(SOURCE, path="a.py", lines=ALL)]
        assert lines == [4, 5, 6, 7, 8]

    def test_should_skip_lines_the_change_does_not_add(self) -> None:
        assert check_source(SOURCE, path="a.py", lines={1, 2, 9, 10, 11, 14}) == []


class TestCheck:
    def test_should_report_only_what_the_change_adds(self, tmp_path: Path) -> None:
        _git(tmp_path, "init", "-q", "-b", "main")
        (tmp_path / "a.py").write_text("x = f()  # noqa\n")
        _git(tmp_path, "add", ".")
        _git(tmp_path, "commit", "-qm", "base")
        (tmp_path / "a.py").write_text("x = f()  # noqa\ny = g()  # type: ignore\n")
        (tmp_path / "b.py").write_text("z = 1\n")
        _git(tmp_path, "add", ".")
        findings = check("HEAD", root=tmp_path)
        assert [(f.path, f.line) for f in findings] == [("a.py", 2)]
