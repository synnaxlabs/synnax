#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
import sys
import tokenize
from collections.abc import Mapping
from pathlib import Path

import pytest
from conftest import GitRepo

from check_skipped_tests import check, language, markers

GO = 'package a\n\nvar _ = It("a", func() {})\n'
TS = 'it("a", () => {});\n'
PY = "def test_a() -> None:\n    pass\n"
CPP = "TEST(A, B) {}\n"
SCRIPT = Path(__file__).resolve().parents[1] / "check_skipped_tests.py"


def _findings(
    repo: GitRepo,
    base: Mapping[str, str | None],
    head: Mapping[str, str | None],
    moves: tuple[tuple[str, str], ...] = (),
) -> list[tuple[str, int]]:
    """Commits base, then moves and writes head, commits again, and returns the path and
    line of each finding."""
    repo.commit(base)
    for old, new in moves:
        (repo.path / new).parent.mkdir(parents=True, exist_ok=True)
        repo.run("mv", old, new)
    repo.commit(head)
    return [(f.path, f.line) for f in check(root=repo.path, base="HEAD~1")]


class TestLanguage:
    """The file names that the check reads as test files."""

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
            ("src/a.bench.tsx", "ts"),
            ("src/a.ts", None),
            ("py/test_a.py", "py"),
            ("py/a_test.py", "py"),
            ("py/conftest.py", "py"),
            ("integration/tests/arc/arc.py", "integration"),
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


class TestMarkers:
    """The markers that the check finds in one file, with comments and strings out."""

    @pytest.mark.parametrize(
        ("lang", "source", "lines"),
        [
            ("go", 'var _ = PIt("b", func() {})\n', [1]),
            ("go", 'var _ = XDescribe("b", func() {})\n', [1]),
            ("go", 'var _ = PEntry("b", 1)\n', [1]),
            ("go", 'var _ = XDescribeTable("b", f)\n', [1]),
            ("go", 'var _ = PDescribeTableSubtree("b", f)\n', [1]),
            ("go", 'var _ = PContext("b", f)\n', [1]),
            ("go", 'var _ = PWhen("b", f)\n', [1]),
            ("go", 'var _ = PSpecify("b", f)\n', [1]),
            ("go", 'var _ = It("b", Pending, f)\n', [1]),
            ("go", 'func f() { Skip("b") }\n', [1]),
            ("go", 'func f() { ginkgo.Skip("b") }\n', [1]),
            ("go", "func f(t *T) { t.Skip() }\n", [1]),
            ("go", 'func f(t *T) { t.Skipf("b") }\n', [1]),
            ("go", "func f(t *T) { t.SkipNow() }\n", [1]),
            ("go", "func f(b *B) { b.Skip() }\n", [1]),
            ("go", "func f(tb TB) { tb.Skip() }\n", [1]),
            ("go", "func f() { GinkgoT().Skip() }\n", [1]),
            ("go", "func (s *S) f() { s.T().Skip() }\n", [1]),
            ("ts", 'it.skip("b", () => {});\n', [1]),
            ("ts", 'describe.skip("b", () => {});\n', [1]),
            ("ts", 'it.skipIf(x)("b", () => {});\n', [1]),
            ("ts", 'it.runIf(x)("b", () => {});\n', [1]),
            ("ts", 'it.todo("b");\n', [1]),
            ("ts", 'xit("b", () => {});\n', [1]),
            ("ts", 'xdescribe("b", () => {});\n', [1]),
            ("ts", 'xtest("b", () => {});\n', [1]),
            ("ts", 'it.skip.each([1])("a", () => {});\n', [1]),
            ("ts", "it.skip`a`(() => {});\n", [1]),
            ("py", "@pytest.mark.skip\n" + PY, [1]),
            ("py", '@pytest.mark.skipif(x, reason="b")\n' + PY, [1]),
            ("py", "@pytest.mark.xfail\n" + PY, [1]),
            ("py", PY + "    pytest.skip()\n", [3]),
            ("py", PY + "    pytest.xfail('b')\n", [3]),
            ("py", PY + "    pytest.importorskip('b')\n", [3]),
            ("py", "@unittest.skipIf(x, 'b')\n" + PY, [1]),
            ("py", "@unittest.expectedFailure\n" + PY, [1]),
            ("py", PY + "    raise unittest.SkipTest('b')\n", [3]),
            ("py", PY + "    self.skipTest('b')\n", [3]),
            ("integration", PY + "    self.skip('b')\n", [3]),
            ("cpp", "TEST(A, DISABLED_C) {}\n", [1]),
            ("cpp", "TEST(DISABLED_A, C) {}\n", [1]),
            ("cpp", "TEST_F(A, DISABLED_C) {}\n", [1]),
            ("cpp", "TEST(A, C) { GTEST_SKIP(); }\n", [1]),
        ],
    )
    def test_should_find_a_marker(
        self, lang: str, source: str, lines: list[int]
    ) -> None:
        assert list(markers(source, lang)) == lines

    @pytest.mark.parametrize(
        ("lang", "source"),
        [
            ("go", '// var _ = PIt("b", f)\nvar s = "Skip(" + `t.Skip()`\n'),
            ("go", "func f() error { return errors.Skip(err, x) }\n"),
            (
                "go",
                "func f(r R) { r.Skip(1) }\nfunc (r R) Skip(n int) {}\n"
                "func Skip() {}\n",
            ),
            ("go", "var s = state{Pending: true}\n"),
            ("ts", '// it.skip("b")\nconst s = "xit(";\n'),
            ("ts", "const s = `it.skip(${x})`;\n"),
            ("ts", 'const s = "a \\" it.skip(";\n'),
            ("ts", 'const s = `a ${b ? `it.skip(` : ""}`;\n'),
            ("py", '# pytest.skip()\nS = "pytest.mark.skip"\n'),
            ("py", "class Reader:\n    def f(self) -> None:\n        self.skip(3)\n"),
            ("cpp", '// DISABLED_A\nconst char* s = "GTEST_SKIP";\n'),
            ("cpp", 'auto s = R"x(a" GTEST_SKIP() ")x";\n'),
            ("cpp", "constexpr uint8_t DISABLED_GROUP = 2;\n"),
        ],
    )
    def test_should_not_find_text_that_is_no_marker(
        self, lang: str, source: str
    ) -> None:
        assert markers(source, lang) == {}

    @pytest.mark.parametrize(
        ("lang", "source", "lines"),
        [
            ("go", "func f(t *T) { p := `C:\\`; t.Skip() }\n", [1]),
            ("ts", 'const q = /"/;\nit.skip("b", () => {});\n', [2]),
            ("ts", 'const s = p.replace(/\\/*$/, "");\nit.skip("b", () => {});\n', [2]),
            ("ts", 'const s = `${q ? `"` : ""}`;\nit.skip("b", () => {});\n', [2]),
            ("ts", 'const n = a / 2; const m = b / 3;\nit.skip("b", () => {});\n', [2]),
            ("cpp", "int x = 1'000; GTEST_SKIP();\n", [1]),
            ("cpp", 'auto s = u8R"(say "hi)";\nTEST(A, C) { GTEST_SKIP(); }\n', [2]),
            (
                "cpp",
                'auto q = L\'"\';\nTEST(A, C) { GTEST_SKIP(); }\nauto t = "x";\n',
                [2],
            ),
            ("py", "# a\x0cb\n" + PY + "    pytest.skip()\nX = 'a'\n", [4]),
            ("py", "S = 'a\u2028b'\n" + PY + "    pytest.skip()\nX = 'a'\n", [4]),
        ],
    )
    def test_should_find_a_marker_after_a_lexer_edge(
        self, lang: str, source: str, lines: list[int]
    ) -> None:
        assert list(markers(source, lang)) == lines


class TestCheck:
    """The findings of a change: the sum of the markers of its test files rose."""

    def test_should_report_an_added_marker(self, git_repo: GitRepo) -> None:
        head = GO + 'var _ = PIt("b", func() {})\n'
        assert _findings(git_repo, {"a_test.go": GO}, {"a_test.go": head}) == [
            ("a_test.go", 4)
        ]

    def test_should_report_a_new_test_file_with_a_marker(
        self, git_repo: GitRepo
    ) -> None:
        head = GO + 'var _ = PIt("b", func() {})\n'
        assert _findings(git_repo, {"b_test.go": GO}, {"a_test.go": head}) == [
            ("a_test.go", 4)
        ]

    def test_should_not_report_a_test_file_with_no_change(
        self, git_repo: GitRepo
    ) -> None:
        assert _findings(git_repo, {"a_test.go": GO}, {"a.go": "package a\n"}) == []

    def test_should_not_report_a_marker_that_moves_inside_a_file(
        self, git_repo: GitRepo
    ) -> None:
        base = GO + 'var _ = PIt("b", func() {})\nvar _ = It("c", func() {})\n'
        head = GO + 'var _ = It("c", func() {})\nvar _ = PIt("b", func() {})\n'
        assert _findings(git_repo, {"a_test.go": base}, {"a_test.go": head}) == []

    def test_should_not_report_a_marker_that_moves_to_another_file(
        self, git_repo: GitRepo
    ) -> None:
        marked = GO + 'var _ = PIt("b", func() {})\n'
        base = {"a_test.go": marked, "b_test.go": GO}
        assert _findings(git_repo, base, {"a_test.go": GO, "b_test.go": marked}) == []

    def test_should_not_report_a_file_split_in_two(self, git_repo: GitRepo) -> None:
        base = GO + 'var _ = PIt("b", f)\nvar _ = XIt("c", f)\n'
        head = {
            "a_test.go": GO + 'var _ = PIt("b", f)\n',
            "b_test.go": 'var _ = XIt("c", f)\n',
        }
        assert _findings(git_repo, {"a_test.go": base}, head) == []

    def test_should_not_report_a_renamed_file(self, git_repo: GitRepo) -> None:
        marked = GO + 'var _ = PIt("b", func() {})\n'
        moves = (("a_test.go", "pkg/b_test.go"),)
        assert _findings(git_repo, {"a_test.go": marked}, {}, moves) == []

    def test_should_not_report_a_renamed_file_with_edits(
        self, git_repo: GitRepo
    ) -> None:
        base = 'package a\n\nfunc TestA(t *T) { t.Skip("needs buf") }\n'
        head = 'package b\n\nfunc TestB(t *T) { t.Skip("needs buf") }\n'
        moves = (("a/a_test.go", "b/b_test.go"),)
        assert (
            _findings(git_repo, {"a/a_test.go": base}, {"b/b_test.go": head}, moves)
            == []
        )

    def test_should_not_report_a_removed_marker(self, git_repo: GitRepo) -> None:
        base = GO + 'var _ = PIt("b", func() {})\n'
        assert _findings(git_repo, {"a_test.go": base}, {"a_test.go": GO}) == []

    def test_should_not_report_a_removed_file(self, git_repo: GitRepo) -> None:
        base = GO + 'var _ = PIt("b", func() {})\n'
        assert _findings(git_repo, {"a_test.go": base}, {"a_test.go": None}) == []

    def test_should_not_report_a_swap_of_one_marker_for_another(
        self, git_repo: GitRepo
    ) -> None:
        base = {"a.spec.ts": 'it.skip("b", () => {});\n'}
        assert _findings(git_repo, base, {"a.spec.ts": 'xit("b", () => {});\n'}) == []

    def test_should_report_a_rise_that_a_removal_elsewhere_does_not_cover(
        self, git_repo: GitRepo
    ) -> None:
        base = {"a_test.go": GO + 'var _ = PIt("b", f)\n', "b_test.go": GO}
        head = {
            "a_test.go": GO,
            "b_test.go": GO + 'var _ = PIt("b", f)\nvar _ = XIt("c", f)\n',
        }
        assert _findings(git_repo, base, head) == [("b_test.go", 4), ("b_test.go", 5)]

    def test_should_name_each_added_marker_line(self, git_repo: GitRepo) -> None:
        head = GO + 'var _ = PIt("b", f)\nvar _ = It("c", f)\nvar _ = XIt("d", f)\n'
        assert _findings(git_repo, {"a_test.go": GO}, {"a_test.go": head}) == [
            ("a_test.go", 4),
            ("a_test.go", 6),
        ]

    def test_should_name_a_changed_line_that_now_holds_a_marker(
        self, git_repo: GitRepo
    ) -> None:
        head = GO.replace('It("a"', 'PIt("a"')
        assert _findings(git_repo, {"a_test.go": GO}, {"a_test.go": head}) == [
            ("a_test.go", 3)
        ]

    def test_should_report_a_marker_that_an_edit_elsewhere_reveals(
        self, git_repo: GitRepo
    ) -> None:
        base = "package a\n\nvar s = `\nfunc f(t *testing.T) { t.Skip() }\n`\n"
        head = base.replace("var s = `\n", "var s = ``\n")
        assert _findings(git_repo, {"a_test.go": base}, {"a_test.go": head}) == [
            ("a_test.go", 0)
        ]

    def test_should_give_the_counts_in_the_message(self, git_repo: GitRepo) -> None:
        _findings(
            git_repo, {"a_test.go": GO}, {"a_test.go": GO + 'var _ = PIt("b", f)\n'}
        )
        (finding,) = check(root=git_repo.path, base="HEAD~1")
        assert finding.message == (
            "adds a skipped test: `PIt(` (skip markers in the changed test files: 0 "
            "before, 1 now)"
        )

    @pytest.mark.parametrize(("line", "code"), [("// a note\n", 0), ("t.Skip()\n", 1)])
    def test_should_exit_with_one_on_a_finding(
        self, git_repo: GitRepo, line: str, code: int
    ) -> None:
        _findings(git_repo, {"a_test.go": GO}, {"a_test.go": GO + line})
        result = subprocess.run(
            [sys.executable, str(SCRIPT), "--base", "HEAD~1"],
            cwd=git_repo.path,
            capture_output=True,
            check=False,
        )
        assert result.returncode == code

    def test_should_need_a_base(self, git_repo: GitRepo) -> None:
        result = subprocess.run(
            [sys.executable, str(SCRIPT), "--base", ""],
            cwd=git_repo.path,
            capture_output=True,
            check=False,
        )
        assert result.returncode == 2


class TestFailLoud:
    """The check stops on input that it cannot read, so CI fails with the cause."""

    def test_should_raise_on_a_python_file_that_does_not_tokenize(
        self, git_repo: GitRepo
    ) -> None:
        head = PY + "x = (\n# pytest.mark.skip\n"
        with pytest.raises(tokenize.TokenError):
            _findings(git_repo, {"test_a.py": PY}, {"test_a.py": head})

    def test_should_raise_on_an_unknown_language(self) -> None:
        with pytest.raises(KeyError):
            markers("GTEST_SKIP()\n", "rust")
