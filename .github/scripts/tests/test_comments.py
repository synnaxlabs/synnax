#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from comments import go, python, strip

GO = 'package a\n\nfunc A() string { return "a" }\n'


class TestGo:
    """Tests stripping Go comments."""

    @pytest.mark.parametrize(
        "text",
        [
            'package a\n\n// A is a.\nfunc A() string { return "a" }\n',
            'package a\n\nfunc A() string { return "a" } // A is a.\n',
            'package a\n\nfunc A() string { /* a */ return "a" }\n',
            'package a\n\n/*\nA is a.\n*/\nfunc A() string { return "a" }\n',
            'package a\n\nfunc A() string {    return "a" }\n',
        ],
    )
    def test_should_match_the_source_without_comments(self, text: str) -> None:
        assert go(text) == go(GO)

    @pytest.mark.parametrize(
        "text",
        [
            'package a\n\nfunc A() string { return "a // b" }\n',
            'package a\n\nfunc A() string { return "a /* b */" }\n',
            "package a\n\nfunc A() string { return `a // b` }\n",
            "package a\n\nfunc A() rune { return '/' }\n",
            '//go:build linux\n\npackage a\n\nfunc A() string { return "a" }\n',
            'package a\n\n//go:generate stringer\nfunc A() string { return "a" }\n',
            'package a\n\nfunc A() string { return "b" }\n',
        ],
    )
    def test_should_keep_code_that_looks_like_a_comment(self, text: str) -> None:
        assert go(text) != go(GO)

    def test_should_keep_a_newline_a_block_comment_spans(self) -> None:
        assert go("a /*\n*/ b") == "a\nb"
        assert go("a /* */ b") == "a b"

    @pytest.mark.parametrize(
        "text", ['package a\n\nimport "C"\n', 'a := "b\n', "a /* b", "a := `b"]
    )
    def test_should_reject_cgo_and_source_that_does_not_lex(self, text: str) -> None:
        assert go(text) is None


class TestPython:
    """Tests stripping Python comments."""

    def test_should_match_the_source_without_comments(self) -> None:
        assert python(b"# A.\na = 1  # One.\n") == python(b"a = 1\n")

    def test_should_keep_a_hash_in_a_string(self) -> None:
        assert python(b'a = "# b"\n') != python(b'a = ""\n')

    def test_should_reject_source_that_does_not_tokenize(self) -> None:
        assert python(b'a = """b\n') is None


class TestStrip:
    """Tests choosing the language by suffix."""

    def test_should_strip_go_and_python(self) -> None:
        assert strip("a.go", GO.encode()) == go(GO)
        assert strip("a.py", b"a = 1\n") == python(b"a = 1\n")

    @pytest.mark.parametrize("path", ["a.ts", "a.cpp", "a.md"])
    def test_should_skip_other_languages(self, path: str) -> None:
        assert strip(path, b"// a\n") is None

    def test_should_reject_invalid_utf8(self) -> None:
        assert strip("a.go", b"\xff") is None
