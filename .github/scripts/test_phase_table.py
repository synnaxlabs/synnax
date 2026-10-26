#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from phase_table import Phase, parse

HEADER = """## 5 Implementation phases

| Phase | Scope                        | Touches                        | Depends on | Tier  |
| ----- | ---------------------------- | ------------------------------ | ---------- | ----- |
"""


def table(*rows: str) -> str:
    return HEADER + "\n".join(rows) + "\n\n## 6 Resolved decisions\n"


class TestParse:
    """Tests for reading the phase table out of an RFC."""

    def test_should_parse_every_column(self) -> None:
        """Should return one phase per row with paths and dependencies split."""
        rfc = table(
            "| 1 | Add the `foo` codec | `core/pkg/foo` | - | light |",
            "| 2 | Read `foo` in the clients | `client/ts`, `client/py` | 1 | bot |",
            "| 3 | Remove the old reader | `core/pkg/old` | 1, 2 | thorough |",
        )
        assert parse(rfc) == [
            Phase(1, "Add the `foo` codec", ["core/pkg/foo"], [], "light"),
            Phase(
                2, "Read `foo` in the clients", ["client/ts", "client/py"], [1], "bot"
            ),
            Phase(3, "Remove the old reader", ["core/pkg/old"], [1, 2], "thorough"),
        ]

    def test_should_stop_at_the_first_line_after_the_table(self) -> None:
        """Should ignore a later table with a different header."""
        rfc = table("| 1 | Add the codec | `core` | - | light |") + "| a | b |\n"
        assert len(parse(rfc)) == 1

    @pytest.mark.parametrize(
        ("markdown", "error"),
        [
            ("# 1 Title\n", "want one phase table, found 0"),
            (HEADER + HEADER, "want one phase table, found 2"),
            (table(), "the phase table has no rows"),
            (table("| 2 | Add the codec | `core` | - | light |"), "phase is '2'"),
            (table("| 1 | Add the codec | `core` | - |"), "want 5 cells, got 4"),
            (table("| 1 |  | `core` | - | light |"), "scope is empty"),
            (table(f"| 1 | {'a' * 61} | `core` | - | light |"), "over 60 characters"),
            (table("| 1 | Add the codec. | `core` | - | light |"), "ends in a period"),
            (table("| 1 | Add the codec | core | - | light |"), "backticked paths"),
            (
                table("| 1 | Add the codec | `core` | 1 | light |"),
                "not an earlier phase",
            ),
            (
                table("| 1 | Add the codec | `core` |  | light |"),
                "not an earlier phase",
            ),
            (table("| 1 | Add the codec | `core` | - | heavy |"), "tier 'heavy'"),
        ],
    )
    def test_should_reject_a_malformed_table(self, markdown: str, error: str) -> None:
        """Should raise an error that names the problem."""
        with pytest.raises(ValueError, match=error):
            parse(markdown)

    def test_should_report_every_error_at_once(self) -> None:
        """Should list the errors of all rows, not only the first."""
        rfc = table(
            "| 1 | Add the codec | `core` | 2 | light |",
            "| 2 | Add the reader | `core` | - | heavy |",
        )
        with pytest.raises(ValueError, match=r"(?s)phase 1: .*\nphase 2: "):
            parse(rfc)
