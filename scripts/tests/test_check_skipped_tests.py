#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
from collections.abc import Mapping
from pathlib import Path

import pytest

from check_skipped_tests import check, language

GO = 'package a\n\nvar _ = It("a", func() {})\n'
TS = 'it("a", () => {});\n'
PY = "def test_a() -> None:\n    pass\n"
CPP = "TEST(A, B) {}\n"


def _git(root: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-c", "user.name=t", "-c", "user.email=t@t", *args],
        cwd=root,
        check=True,
        capture_output=True,
    )


def _run(
    root: Path,
    base: dict[str, str],
    head: Mapping[str, str | None],
    moves: tuple[tuple[str, str], ...] = (),
) -> list[tuple[str, int]]:
    """Commits base, then moves and writes head (None removes a file), commits again,
    and returns the path and line of each finding."""
    _git(root, "init", "-q", "-b", "main")
    for path, text in base.items():
        (root / path).parent.mkdir(parents=True, exist_ok=True)
        (root / path).write_text(text)
    _git(root, "add", ".")
    _git(root, "commit", "-qm", "base")
    for old, new in moves:
        (root / new).parent.mkdir(parents=True, exist_ok=True)
        _git(root, "mv", old, new)
    for path, change in head.items():
        if change is None:
            (root / path).unlink()
        else:
            (root / path).parent.mkdir(parents=True, exist_ok=True)
            (root / path).write_text(change)
    _git(root, "add", "-A")
    _git(root, "commit", "-qm", "head")
    return [(f.path, f.line) for f in check(root=root, base="HEAD~1")]


class TestLanguage:
    @pytest.mark.parametrize(
        ("path", "want"),
        [
            ("x/a_test.go", "go"),
            ("a.go", None),
            ("src/a.spec.ts", "ts"),
            ("src/a.spec.tsx", "ts"),
            ("src/a.test.ts", "ts"),
            ("src/a.test.tsx", "ts"),
            ("src/a.bench.ts", "ts"),
            ("src/a.ts", None),
            ("py/test_a.py", "py"),
            ("py/a_test.py", "py"),
            ("py/conftest.py", "py"),
            ("integration/tests/arc/arc.py", "py"),
            ("py/a.py", None),
            ("driver/a_test.cpp", "cpp"),
            ("driver/a_test.cc", "cpp"),
            ("driver/a.cpp", None),
        ],
    )
    def test_should_find_the_language_of_a_test_file(
        self, path: str, want: str | None
    ) -> None:
        assert language(path) == want


class TestCheck:
    @pytest.mark.parametrize(
        ("path", "base", "head", "line"),
        [
            ("a_test.go", GO, GO + 'var _ = PIt("b", func() {})\n', 4),
            ("a_test.go", GO, GO + 'var _ = XDescribe("b", func() {})\n', 4),
            ("a_test.go", GO, GO + 'var _ = PEntry("b", 1)\n', 4),
            ("a_test.go", GO, GO + 'var _ = XDescribeTable("b", f)\n', 4),
            ("a_test.go", GO, GO + 'var _ = PContext("b", f)\n', 4),
            ("a_test.go", GO, GO + 'var _ = PWhen("b", f)\n', 4),
            ("a_test.go", GO, GO + 'var _ = PSpecify("b", f)\n', 4),
            ("a_test.go", GO, GO + 'var _ = It("b", Pending, f)\n', 4),
            ("a_test.go", GO, GO + 'func f() { Skip("b") }\n', 4),
            ("a_test.go", GO, GO + 'func f() { ginkgo.Skip("b") }\n', 4),
            ("a_test.go", GO, GO + "func f(t *T) { t.Skip() }\n", 4),
            ("a_test.go", GO, GO + 'func f(t *T) { t.Skipf("b") }\n', 4),
            ("a_test.go", GO, GO + "func f(t *T) { t.SkipNow() }\n", 4),
            ("a_test.go", GO, GO + "func f(b *B) { b.Skip() }\n", 4),
            ("a.spec.ts", TS, TS + 'it.skip("b", () => {});\n', 2),
            ("a.spec.ts", TS, TS + 'describe.skip("b", () => {});\n', 2),
            ("a.spec.ts", TS, TS + 'it.skipIf(x)("b", () => {});\n', 2),
            ("a.spec.ts", TS, TS + 'it.runIf(x)("b", () => {});\n', 2),
            ("a.spec.ts", TS, TS + 'it.todo("b");\n', 2),
            ("a.spec.ts", TS, TS + 'xit("b", () => {});\n', 2),
            ("a.spec.ts", TS, TS + 'xdescribe("b", () => {});\n', 2),
            ("a.spec.ts", TS, TS + 'xtest("b", () => {});\n', 2),
            ("test_a.py", PY, "@pytest.mark.skip\n" + PY, 1),
            ("test_a.py", PY, '@pytest.mark.skipif(x, reason="b")\n' + PY, 1),
            ("test_a.py", PY, "@pytest.mark.xfail\n" + PY, 1),
            ("test_a.py", PY, PY + "    pytest.skip()\n", 3),
            ("test_a.py", PY, PY + "    pytest.importorskip('b')\n", 3),
            ("test_a.py", PY, "@unittest.skipIf(x, 'b')\n" + PY, 1),
            ("test_a.py", PY, PY + "    self.skipTest('b')\n", 3),
            ("test_a.py", PY, PY + "    self.skip('b')\n", 3),
            ("a_test.cpp", CPP, CPP + "TEST(A, DISABLED_C) {}\n", 2),
            ("a_test.cpp", CPP, CPP + "TEST(DISABLED_A, C) {}\n", 2),
            ("a_test.cpp", CPP, CPP + "TEST(A, C) { GTEST_SKIP(); }\n", 2),
        ],
    )
    def test_should_report_an_added_marker(
        self, tmp_path: Path, path: str, base: str, head: str, line: int
    ) -> None:
        assert _run(tmp_path, {path: base}, {path: head}) == [(path, line)]

    def test_should_report_a_new_test_file_with_a_marker(self, tmp_path: Path) -> None:
        head = GO + 'var _ = PIt("b", func() {})\n'
        assert _run(tmp_path, {"b_test.go": GO}, {"a_test.go": head}) == [
            ("a_test.go", 4)
        ]

    @pytest.mark.parametrize(
        ("path", "head"),
        [
            (
                "a_test.go",
                GO + '// var _ = PIt("b", f)\nvar s = "Skip(" + `t.Skip()`\n',
            ),
            ("a_test.go", GO + "func f() error { return errors.Skip(err, x) }\n"),
            (
                "a_test.go",
                GO + "func f(r R) { r.Skip(1) }\nfunc (r R) Skip(n int) {}\n",
            ),
            ("a_test.go", GO + "var s = state{Pending: true}\n"),
            ("a.spec.ts", TS + '// it.skip("b")\nconst s = "xit(";\n'),
            ("a.spec.ts", TS + "const s = `it.skip(${x})`;\n"),
            ("test_a.py", PY + '# pytest.skip()\nS = "pytest.mark.skip"\n'),
            ("a_test.cpp", CPP + '// DISABLED_A\nconst char* s = "GTEST_SKIP";\n'),
            ("a_test.cpp", CPP + "int n = 1'000;\n"),
            ("a.go", 'package a\n\nvar _ = PIt("b", f)\n'),
            ("a.ts", 'it.skip("b", () => {});\n'),
            ("a.py", "@pytest.mark.skip\ndef f() -> None:\n    pass\n"),
        ],
    )
    def test_should_not_report_text_that_is_no_marker(
        self, tmp_path: Path, path: str, head: str
    ) -> None:
        base = {"a_test.go": GO, "a.spec.ts": TS, "test_a.py": PY, "a_test.cpp": CPP}
        base.setdefault(path, "")
        assert _run(tmp_path, base, {path: head}) == []

    def test_should_not_report_a_marker_that_moves_inside_a_file(
        self, tmp_path: Path
    ) -> None:
        base = GO + 'var _ = PIt("b", func() {})\nvar _ = It("c", func() {})\n'
        head = GO + 'var _ = It("c", func() {})\nvar _ = PIt("b", func() {})\n'
        assert _run(tmp_path, {"a_test.go": base}, {"a_test.go": head}) == []

    def test_should_not_report_a_renamed_file(self, tmp_path: Path) -> None:
        text = GO + 'var _ = PIt("b", func() {})\n'
        moves = (("a_test.go", "pkg/b_test.go"),)
        assert _run(tmp_path, {"a_test.go": text}, {}, moves) == []

    def test_should_not_report_a_removed_marker(self, tmp_path: Path) -> None:
        base = GO + 'var _ = PIt("b", func() {})\n'
        assert _run(tmp_path, {"a_test.go": base}, {"a_test.go": GO}) == []

    def test_should_not_report_a_swap_of_one_marker_for_another(
        self, tmp_path: Path
    ) -> None:
        base = 'it.skip("b", () => {});\n'
        head = 'xit("b", () => {});\n'
        assert _run(tmp_path, {"a.spec.ts": base}, {"a.spec.ts": head}) == []

    def test_should_not_report_a_removed_file(self, tmp_path: Path) -> None:
        base = GO + 'var _ = PIt("b", func() {})\n'
        assert _run(tmp_path, {"a_test.go": base}, {"a_test.go": None}) == []

    def test_should_report_a_marker_that_moves_to_another_file(
        self, tmp_path: Path
    ) -> None:
        base = {"a_test.go": GO + 'var _ = PIt("b", func() {})\n', "b_test.go": GO}
        head = {"a_test.go": GO, "b_test.go": GO + 'var _ = PIt("b", func() {})\n'}
        assert _run(tmp_path, base, head) == [("b_test.go", 4)]

    def test_should_name_each_added_marker_line(self, tmp_path: Path) -> None:
        head = GO + 'var _ = PIt("b", f)\nvar _ = It("c", f)\nvar _ = XIt("d", f)\n'
        assert _run(tmp_path, {"a_test.go": GO}, {"a_test.go": head}) == [
            ("a_test.go", 4),
            ("a_test.go", 6),
        ]

    def test_should_give_the_counts_in_the_message(self, tmp_path: Path) -> None:
        _run(tmp_path, {"a_test.go": GO}, {"a_test.go": GO + 'var _ = PIt("b", f)\n'})
        (finding,) = check(root=tmp_path, base="HEAD~1")
        assert finding.message == (
            "adds a skipped test: `PIt(` (skip markers in the file: 0 before, 1 now)"
        )
