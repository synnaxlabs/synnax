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

| Phase | Scope                        | Touches                        | Depends on |
| ----- | ---------------------------- | ------------------------------ | ---------- |
"""


def table(*rows: str) -> str:
    return HEADER + "\n".join(rows) + "\n\n## 6 Resolved decisions\n"


class TestParse:
    """Tests for reading the phase table out of an RFC."""

    def test_should_parse_every_column(self) -> None:
        """Should return one phase per row with paths and dependencies split."""
        rfc = table(
            "| 1 | Add the `foo` codec | `core/pkg/foo` | - |",
            "| 2 | Read `foo` in the clients | `client/ts`, `client/py` | 1 |",
            "| 3 | Remove the old reader | `core/pkg/old` | 1, 2 |",
        )
        assert parse(rfc) == [
            Phase(
                number=1,
                scope="Add the `foo` codec",
                touches=["core/pkg/foo"],
                depends_on=[],
            ),
            Phase(
                number=2,
                scope="Read `foo` in the clients",
                touches=["client/ts", "client/py"],
                depends_on=[1],
            ),
            Phase(
                number=3,
                scope="Remove the old reader",
                touches=["core/pkg/old"],
                depends_on=[1, 2],
            ),
        ]

    def test_should_keep_escaped_pipes_in_a_cell(self) -> None:
        """Should read an escaped pipe as text and drop its escape."""
        rfc = table(r"| 1 | Add `a \| b` support | `core` | - |")
        assert parse(rfc)[0].scope == "Add `a | b` support"

    def test_should_stop_at_the_first_line_after_the_table(self) -> None:
        """Should ignore a later table with a different header."""
        rfc = table("| 1 | Add the codec | `core` | - |") + "| a | b |\n"
        assert len(parse(rfc)) == 1

    @pytest.mark.parametrize(
        ("markdown", "error"),
        [
            ("# 1 Title\n", "want one phase table, found 0"),
            (HEADER + HEADER, "want one phase table, found 2"),
            (table(), "the phase table has no rows"),
            (
                HEADER.replace("| ----- |", "text |", 1)
                + "| 1 | Add the codec | `core` | - |\n",
                "no separator row",
            ),
            (HEADER.split("| -----")[0], "no separator row"),
            (table("| 2 | Add the codec | `core` | - |"), "phase is '2'"),
            (table("| 1 | Add the codec | `core` |"), "want 4 cells, got 3"),
            (table("| 1 |  | `core` | - |"), "scope is empty"),
            (table(f"| 1 | {'a' * 61} | `core` | - |"), "over 60 characters"),
            (table("| 1 | Add the codec. | `core` | - |"), "ends in a period"),
            (table("| 1 | Add the codec | core | - |"), "backticked paths"),
            (table("| 1 | Add the codec | `core` | 1 |"), "not an earlier phase"),
            (table("| 1 | Add the codec | `core` |  |"), "not an earlier phase"),
        ],
    )
    def test_should_reject_a_malformed_table(self, markdown: str, error: str) -> None:
        """Should raise an error that names the problem."""
        with pytest.raises(ValueError, match=error):
            parse(markdown)

    def test_should_report_every_error_at_once(self) -> None:
        """Should list the errors of all rows, not only the first."""
        rfc = table(
            "| 1 | Add the codec | `core` | 2 |",
            "| 2 | Add the reader | core | - |",
        )
        with pytest.raises(ValueError, match=r"(?s)phase 1: .*\nphase 2: "):
            parse(rfc)
