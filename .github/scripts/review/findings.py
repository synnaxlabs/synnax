#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass
from typing import Any

SEVERITIES = ("P0", "P1", "P2")

_TEXT = {"type": "string"}
_LINE = {"type": "integer", "minimum": 1}
SCHEMA: dict[str, Any] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["score", "verdict", "summary", "findings"],
    "properties": {
        "score": {"type": "integer", "minimum": 1, "maximum": 5},
        "verdict": _TEXT,
        "summary": {"type": "array", "items": _TEXT},
        "findings": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "required": ["severity", "title", "body", "path", "line"],
                "properties": {
                    "severity": {"enum": list(SEVERITIES)},
                    "title": _TEXT,
                    "body": _TEXT,
                    "path": _TEXT,
                    "line": _LINE,
                    "start_line": _LINE,
                    "suggestion": _TEXT,
                    "rule": _TEXT,
                    "deferrable": {"type": "boolean"},
                },
            },
        },
    },
}
"""The JSON Schema of the review the reviewer writes, which parse reads."""


@dataclass(frozen=True, kw_only=True)
class Finding:
    """One problem the reviewer found, anchored to lines of a file at the head."""

    severity: str
    title: str
    body: str
    path: str
    line: int
    start_line: int | None = None
    suggestion: str | None = None
    """Replacement text for lines start_line through line."""
    rule: str | None = None
    """Repository path of the document that states the rule the finding enforces."""
    deferrable: bool = False
    """The finding can wait for a follow-up issue."""

    @property
    def blocks(self) -> bool:
        """Whether the finding blocks merge. A P0 always does."""
        return self.severity == "P0" or not self.deferrable

    @property
    def lines(self) -> range:
        start = self.line if self.start_line is None else self.start_line
        return range(start, self.line + 1)


@dataclass(frozen=True, kw_only=True)
class Dismissal:
    """A finding that a member told the bot to stop reporting on a pull request."""

    path: str
    title: str
    reason: str
    """What the member wrote beside the command. It can be empty."""


@dataclass(frozen=True, kw_only=True)
class Review:
    score: int
    """Confidence that the pull request is safe to merge, from 1 to 5."""
    verdict: str
    summary: list[str]
    findings: list[Finding]


def parse(data: dict[str, Any]) -> Review:
    """Builds a review from the JSON the reviewer wrote.

    :raises TypeError: If a finding has a missing or unknown field.
    :raises ValueError: If the score, a severity, or a line range is out of range.
    """
    review = Review(
        score=data["score"],
        verdict=data["verdict"],
        summary=list(data["summary"]),
        findings=[Finding(**f) for f in data["findings"]],
    )
    if not 1 <= review.score <= 5:
        raise ValueError(f"score {review.score} is not between 1 and 5")
    for f in review.findings:
        if f.severity not in SEVERITIES:
            raise ValueError(f"severity {f.severity!r} is not one of {SEVERITIES}")
        if f.line < 1 or not 1 <= f.lines.start <= f.line:
            raise ValueError(f"lines {f.start_line}-{f.line} of {f.path} are invalid")
    return review
