#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import subprocess

import pytest

from check_glibc_floor import main, problems

SYMBOLS = """
0000000000000000      DF *UND*  0000000000000000 (GLIBC_2.2.5) memcpy
0000000000000000      DF *UND*  0000000000000000 (GLIBC_2.34) pthread_create
0000000000000000      DF *UND*  0000000000000000 (GLIBC_2.35) __memcpy_chk
"""

HEADERS = """
Dynamic Section:
  NEEDED               libm.so.6
  NEEDED               libc.so.6
"""


class TestProblems:
    """Tests for finding why a binary cannot load on the glibc floor."""

    def test_should_accept_symbols_at_or_below_the_floor(self) -> None:
        """Should pass a binary that needs glibc 2.35 at most and no libstdc++."""
        assert problems(SYMBOLS, HEADERS) == []

    def test_should_reject_a_newer_glibc_symbol(self) -> None:
        """Should name each symbol version above the floor, numerically sorted."""
        symbols = SYMBOLS + "(GLIBC_2.39) pidfd_spawnp\n(GLIBC_2.38) __isoc23_strtol\n"
        assert problems(symbols, HEADERS) == [
            "needs GLIBC_2.38, GLIBC_2.39, newer than the glibc 2.35 floor"
        ]

    def test_should_compare_versions_as_numbers(self) -> None:
        """Should treat 2.4 as older than 2.35, not as a newer string."""
        assert problems("(GLIBC_2.4) fn\n(GLIBC_2.17) fn\n", HEADERS) == []

    def test_should_reject_shared_libstdcpp_and_libgcc(self) -> None:
        """Should fail a binary that loads libstdc++ or libgcc_s at runtime."""
        headers = HEADERS + "  NEEDED libstdc++.so.6\n  NEEDED libgcc_s.so.1\n"
        assert problems(SYMBOLS, headers) == [
            "links libstdc++.so.6 dynamically",
            "links libgcc_s.so.1 dynamically",
        ]


class TestMain:
    """Tests for checking a binary through objdump."""

    def fake_objdump(self, monkeypatch: pytest.MonkeyPatch, symbols: str) -> None:
        """Answers objdump -T with symbols and objdump -p with HEADERS."""

        def check_output(args: list[str], text: bool) -> str:
            assert args[0] == "objdump"
            assert args[2] == "driver"
            assert text
            return symbols if args[1] == "-T" else HEADERS

        monkeypatch.setattr(subprocess, "check_output", check_output)

    def test_should_pass_a_binary_at_the_floor(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should exit 0 when the binary loads on the floor."""
        self.fake_objdump(monkeypatch, SYMBOLS)
        assert main("driver") == 0

    def test_should_fail_a_binary_above_the_floor(
        self, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
    ) -> None:
        """Should exit 1 and name the binary and the problem on stderr."""
        self.fake_objdump(monkeypatch, SYMBOLS + "(GLIBC_2.39) pidfd_spawnp\n")
        assert main("driver") == 1
        assert capsys.readouterr().err == (
            "driver needs GLIBC_2.39, newer than the glibc 2.35 floor\n"
        )
